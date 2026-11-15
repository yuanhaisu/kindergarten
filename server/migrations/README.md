# 数据库迁移

schema 由 [goose](https://github.com/pressly/goose) 版本化管理；应用启动不再执行
`AutoMigrate`，也不再播种。迁移与种子都是显式命令。

## 新环境首次初始化

```bash
cd server
make migrate    # goose up
make seedbase   # 字典 / 膳食 / 园长角色（幂等）
```

## 日常变更

1. 在 `server/migrations/` 新增 `0000N_<描述>.sql`：

   ```sql
   -- +goose Up
   ALTER TABLE ...;

   -- +goose Down
   ALTER TABLE ...;
   ```

   MySQL 的 DDL 会隐式提交，复杂迁移请在文件顶部加 `-- +goose NO TRANSACTION`。

2. `make migrate` 执行。
3. 用 `go run ./cmd/dbcheck <table>` 核对索引与行数。
4. 不要修改 `00001_baseline.sql`。它是从开发库导出的历史基线，只用于新建库。

## 版本顺序与开发库修复（2026-11-15）

### 线性顺序

`00005` 曾经与商业化迁移同号；PR #28 已把商业化迁移顺延。当前顺序：

```
00001 baseline
00002 SaaS 商业化
00003 回填商业化授权
00004 平台运维与强制改密标记
00005 隐私同意
00006 佣金净额结算
00007 收款幂等
```

### 开发库修复（一次性）

开发库的 schema 可能已经手工执行过 00006/00007，但 `goose_db_version` 只记到 5。
先确认对象存在：

```sql
-- consent（version 5 已记录，确认表存在即可）
SHOW TABLES LIKE 'user_consents';

-- commission（version 6）
SHOW COLUMNS FROM commission_ledger LIKE 'reversed_amount_cents';
SHOW COLUMNS FROM payments LIKE 'refund_of_payment_id';

-- payment（version 7）
SHOW COLUMNS FROM payments LIKE 'idempotency_key';
SHOW INDEX FROM payments WHERE Key_name = 'uk_payments_order_idem';
```

确认都存在后，补版本记录（可重复执行）：

```sql
INSERT INTO goose_db_version (version_id, is_applied, tstamp)
SELECT 6, 1, NOW() FROM DUAL
WHERE NOT EXISTS (SELECT 1 FROM goose_db_version WHERE version_id = 6);

INSERT INTO goose_db_version (version_id, is_applied, tstamp)
SELECT 7, 1, NOW() FROM DUAL
WHERE NOT EXISTS (SELECT 1 FROM goose_db_version WHERE version_id = 7);
```

如果某个对象不存在，**不要**补对应版本；让 `goose up` 正常执行该迁移。
补完后 `go run ./cmd/migrate status` 应显示 7 已应用，`go run ./cmd/migrate up` 应 no-op。

### 干净库验证

```bash
cd server
cp configs/app.ini.example configs/app.ini
MYSQL_DSN='root:root@tcp(127.0.0.1:3306)/kindergarten_migrate_test?charset=utf8mb4&parseTime=True&loc=Local' \
  go run ./cmd/migrate up
go run ./cmd/migrate status
go run ./cmd/dbcheck user_consents
```

### 防复发

- 新增迁移前先 `ls server/migrations`，确认版本号线性递增。
- `server/migrations/**` 是共享文件；并行开发按 `docs/agents/parallel-development.md` 串行修改。
- CI 的「数据库迁移（干净库）」job 会在干净 MySQL 上跑 `goose up`，同号会在 PR 阶段失败。

## 基线是怎么来的

`00001_baseline.sql` 由下面的命令从**已执行完全部 `docs/migrations/` 手工 SQL 的开发库**
导出，不是按 GORM 模型生成的——模型无法完整表达唯一索引、列类型与回填：

```bash
cd server
go run ./cmd/schemadump > migrations/00001_baseline.sql
```

导出只读，不含数据；`AUTO_INCREMENT` 当前值会被剔除。

## 权限

- 迁移账号（DDL）：只给发布流水线的迁移 job 使用。
- 应用账号（DML）：`SELECT / INSERT / UPDATE / DELETE`，应用进程使用。
