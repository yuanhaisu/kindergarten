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

2. **同步更新下方「版本清单」**：任何新增/删除 `server/migrations/*.sql` 的 PR
   都必须更新该表格。CI 运行 `node scripts/check-migrations-readme.mjs` 比对
   README 表格与目录内文件，漏列/多列时失败并打印缺失或多出的版本号（不依赖数据库）。
3. `make migrate` 执行。
4. 用 `go run ./cmd/dbcheck <table>` 核对索引与行数。
5. 不要修改 `00001_baseline.sql`。它是从开发库导出的历史基线，只用于新建库。

## 版本清单

当前线性序列共 15 个版本。`00005` 曾与商业化迁移同号，PR #28 已顺延；
新增迁移前先 `ls server/migrations`，确认版本号线性递增。

| 版本 | 用途 |
| --- | --- |
| 00001 | 初始基线：由开发库导出，只用于新建库，不得手改 |
| 00002 | SaaS 商业化第一期：平台账号、套餐、订单收款、功能授权与分销佣金 |
| 00003 | 为存量园所回填 base 永久授权与 premium 模块 30 天试用 |
| 00004 | 园长账号强制改密标记 + 平台方操作留痕 |
| 00005 | 隐私政策同意记录（用户 × 政策版本，追加不软删） |
| 00006 | 佣金按净额结算：累计冲回金额 + 退款收款关联原收款 |
| 00007 | 收款幂等：同一订单下幂等键唯一 |
| 00008 | 平台控制台审计：目标园所、请求链路标识与检索索引 |
| 00009 | 订单付款到期日 `pay_due_at`：回填老订单后置为 NOT NULL |
| 00010 | 平台账号角色列（当前统一 `platform`，为平台内分权预留） |
| 00011 | 订单信用开通：零收款确认需理由并留痕 |
| 00012 | 小程序客户端注册表（#163）：每个客户端一个微信 AppID |
| 00013 | 一人一账号：`users.phone_active` 唯一 + `user_identities` 拆分（#164） |
| 00014 | 园所入口字段：租户通道、公开入口码、品牌主题色（#177） |
| 00015 | 退役 `users.open_id` / `users.union_id`（#173） |

### 00012–00015：共用 v1 身份与会话底座（#160）

这四个迁移属于 #160「共用 v1 身份与会话底座」：00012 建立客户端注册表，
00013 把「一人一账号 + 多客户端身份」落到唯一键与 `user_identities`，
00014 补齐园所入口字段，00015 删除已被 `user_identities` 取代的旧身份列。
它们依赖 00001–00011 的顺序，不能单独重排。

### 历史：开发库版本补记（仅对 00006/00007，勿照搬）

2026-11-15 曾出现开发库对象已被手工创建、而 `goose_db_version` 只记到 5 的情况，
当时用手写 `INSERT` 补记 6/7。该步骤只适用于那一批开发库，**对当前迁移链
（00001–00015）不再适用**：新环境与正常升级都让 `goose up` 按序执行即可。
若 `migrate status` 与实际 schema 不一致，先确认对象是否存在，再让 `goose up`
执行对应迁移；不要手写 `goose_db_version`。

## 干净库验证

```bash
cd server
cp configs/app.ini.example configs/app.ini
MYSQL_DSN='root:root@tcp(127.0.0.1:3306)/kindergarten_migrate_test?charset=utf8mb4&parseTime=True&loc=Local' \
  go run ./cmd/migrate up
go run ./cmd/migrate status
go run ./cmd/dbcheck user_consents
```

## 防复发

- 新增迁移前先 `ls server/migrations`，确认版本号线性递增。
- 新增/删除迁移的 PR 必须同步「版本清单」；CI 的迁移 README 漂移检查会拦截漏改。
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
