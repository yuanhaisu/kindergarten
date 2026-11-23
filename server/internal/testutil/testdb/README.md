# testdb — DB 测试的测试库入口与护栏

需要真实 MySQL 的 service / repository 测试统一走 `testdb.Open(t)`，不要各自
`gorm.Open(configs/app.ini)`。本包负责三件事：

1. **显式测试 DSN**：`KDG_TEST_MYSQL_DSN` 指向一台测试 MySQL。
2. **包级派生测试库**：库名 = `<基础测试库>_test_<分支+测试包哈希>`，`go test ./...`
   的并行包各写各的库，不会互相清表。
3. **安全护栏**：拒绝可疑目标，报错只打印脱敏 `host:port/db`。

## 用法

```go
import "kindergarten-server/internal/testutil/testdb"

func openTestDB(t *testing.T) *gorm.DB {
    t.Helper()
    return testdb.Open(t)
}
```

需要库名/脱敏目标做断言时用 `testdb.OpenInfo(t)`：

```go
info := testdb.OpenInfo(t)
t.Log(info.Database)    // 实际连接的库名
t.Log(info.Target)      // 127.0.0.1:3316/kindergarten_test_ab12cd34（已脱敏）
```

## 环境变量

| 变量 | 作用 |
| --- | --- |
| `KDG_TEST_MYSQL_DSN` | 显式测试 MySQL DSN，例如 `user:pass@tcp(127.0.0.1:3306)/kindergarten_test?charset=utf8mb4&parseTime=True` |
| `CONFIG_PATH` | 配置文件路径；不设时按仓库根目录找 `configs/app.ini`、`configs/app.local.ini`。开发库 DSN 只用于护栏比对与无 DSN 时的本地回退 |

**不要**把真实 DSN 提交到仓库、写进文档或贴进工单；只在本地 shell / CI secret 里设置。

## 行为矩阵

| `KDG_TEST_MYSQL_DSN` | 配置文件 | 行为 |
| --- | --- | --- |
| 已设置 | 有 | 校验 DSN → 派生包级测试库 → 执行 goose 迁移；**不连接开发库** |
| 已设置 | 无 | 校验 DSN（只能靠库名含 `test` 判定）→ 派生测试库 → 迁移 |
| 未设置 | 有 | 保持现有本地行为：直接连接配置里的开发库，不建库、不迁移 |
| 未设置 | 无 | `t.Skip`（与现有“本地无 DB 配置就跳过”一致） |

第一行是 CI 的目标形态；第四行是本地无 MySQL 时的现状。**未设置时连接开发库
是待移除的过渡行为（#208 收口），新测试不要依赖它写真实数据。**

## 护栏规则（失败关闭）

`testdb.Guard` 在以下任一情况拒绝运行，调用方（`Open`）直接 `t.Fatalf`：

- DSN 无法解析（只支持 tcp 形式）；
- DSN 没有数据库名；
- 库名**不含 `test`**（大小写不敏感）——无法确认是测试库就不跑；
- 配置文件可读时，库名与开发库同名；
- 显式 `CONFIG_PATH` 指向的配置文件读不了（无法比对，按失败处理）。

报错与日志只使用 `testdb.Target(dsn)` 的 `host:port/db`，账号与口令一律不出现。
注意：因此 **CI 的 MySQL 库名必须包含 `test`**（如 `kindergarten_test`），
且账号需要对基础库有 `CREATE DATABASE` 权限（helper 会建派生库）。

## 派生与隔离

- 派生库名形如 `kindergarten_test_ab12cd34`，后缀来自「当前 git 分支 + 测试二进制名」；
  同一分支同一包稳定复用，不同包/不同分支不同库。
- 基础库名里的 `_test` 后缀不会重复（不会出现 `_test_test_`），总长度不超过 64。
- 迁移与 authharness 同源：goose + `migrations.FS`，用 `GET_LOCK` 做跨进程互斥。
- **本包不会 DROP 任何数据库。** 测试必须用唯一键（不要依赖自增 ID 猜行），
  并在 `t.Cleanup` 里 `Unscoped()` 硬删自己写入的行——软删行仍占唯一索引。

## 与 authharness 的关系

`internal/testutil/authharness` 是认证 HTTP 单缝，自带一套测试库装配（它还需要
配置里的 JWT/Casbin 参数）。本包面向 service/repository 测试，护栏更严格
（库名必须含 `test`）。两者当前各自独立；把 authharness 也切到本包属于后续
收口，不在 #207 范围。
