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
