# 后端 DB-backed 商业测试门禁

合并前必须让「订单确认 / 收款 / 授权 / 平台开通」这类 DB-backed 测试真正跑起来；只跑不依赖数据库的包等于没有门禁。

## CI 里跑什么

`.github/workflows/ci.yml` 的 `migrate` job（「数据库迁移（干净库）」）在原有干净库检查之后追加：

1. `goose up` 从空库迁移到最新版本，`seedbase` 幂等种子（原有步骤保留）。
2. 执行 `go test ./internal/repository ./internal/service -race -p 1 -count=1 -timeout 15m -v -run '<商业域前缀>'`
   - `-race`：数据竞争也要暴露。
   - `-p 1`：两个包共享同一个 MySQL service，串行执行避免互相清表。
   - `-timeout 15m` + step `timeout-minutes: 30`：有界，跑飞不会拖死流水线。
   - `-v`：失败输出能定位到具体测试。
3. 跳过守卫：日志中出现**缺库/缺配置**的具体跳过原因（`跳过：找不到 …configs/app.ini`、`跳过：MySQL DSN 为空`、`跳过：本地无可用 DB 配置`、`跳过：MySQL 未配置`、`authharness:`）即失败；`recognition` 等用例的数据依赖跳过仍允许。随后再断言一组关键商业用例确实出现 `--- PASS`（订单确认、收款上限、零收款理由、派生状态、平台开通复用账号），防止 `-run` 选空或测试改名后门禁静默失效。

MySQL 来自 job 内 `services.mysql`（`mysql:8.0`，库名 `kindergarten_migrate_test`），DSN 通过 job 级环境变量 `MYSQL_DSN` 注入，不引用 staging / 生产凭据。另起一个无鉴权的 `services.redis`（`redis:7-alpine`，6379，健康检查 `redis-cli ping`）；`Prepare config` 会把模板 `[redis]` 段的 `password = CHANGE_ME` 置空，否则会因 `AUTH CHANGE_ME` 被拒（`REDIS_PASSWORD` 的空值不会覆盖文件）。选取内的 Redis 用例实际都走 `fakeredis`；也已在「`REDIS_ADDR` 指向关闭端口」的 MySQL-only 环境实测本选取可全绿，Redis service 只是与真实运行环境对齐的兜底。

### 为什么按测试名选取，而不是整包

仓库里有一批历史 DB 用例假设开发库里存在 `kindergarten 4` / `class 6` 等固定数据，在空库上会因外键失败。本门禁只覆盖商业域（后续商业票 4–7）：测试名按前缀选取

```
^(TestBilling|TestPayment|TestOrder|TestEntitlement|TestCommission|TestPlan_|TestPlanTier|TestPlatform|TestSubscription|TestCommercial|TestActiveStudentCount|TestDeriveOrderStatus|TestCancelGuard)
```

新增商业 DB 用例请沿用这些前缀，否则不会进入门禁。原有的干净库迁移检查（`goose up` / `seedbase` / schema 校验）不受影响。

## 本地复跑

DB-backed 测试固定读取 `server/configs/app.ini`（相对路径 `../../configs/app.ini`），`MYSQL_DSN` 环境变量优先于文件里的 DSN。

```bash
cd server
cp configs/app.ini.example configs/app.ini   # 已有则跳过

# 指向一次性的开发库 / 测试库；不要把生产 DSN 填进来
export MYSQL_DSN="user:pass@tcp(127.0.0.1:3306)/kindergarten?charset=utf8mb4&parseTime=True&loc=Local"

# 复现 CI 的选取与参数
go test ./internal/repository ./internal/service -race -p 1 -count=1 -timeout 15m -v \
  -run '^(TestBilling|TestPayment|TestOrder|TestEntitlement|TestCommission|TestPlan_|TestPlanTier|TestPlatform|TestSubscription|TestCommercial|TestActiveStudentCount|TestDeriveOrderStatus|TestCancelGuard)'
```

只想跑单个文件 / 单个测试时：

```bash
go test ./internal/service -run 'TestBillingFlow|TestPaymentIntegrity|TestDeriveOrderStatus' -race -count=1 -v
```

### 注意事项

- **缺少配置会 skip**：找不到 `configs/app.ini` 或 DSN 为空时，这些测试会 `t.Skip`。CI 用跳过守卫把 skip 变成失败；本地跑出 `--- SKIP` 说明库没接上，不是测试通过。
- **authharness 需要建库权限**：`internal/repository` 的身份/认证测试会为每个 Go 测试进程建一个独立测试库（库名派生自 DSN 的库名）。本地账号没有 `CREATE DATABASE` 时，设置 `KDG_TEST_MYSQL_DSN` 指向一个**已存在**的独立测试库；不要指回开发库。
- **测试会写库**：测试用各自的数据前缀并在结束后硬删自己造的数据（软删行会占唯一索引）。仍应指向可丢弃的库，不要指向生产库。
- **Windows 上跑 `-race`**：需要 64 位 gcc，设置 `CC` 指向它（例如 WinLibs mingw64 的 `gcc.exe`）；32 位 MinGW 会链接失败。
- **验证门禁是否真的会红**：临时加一个匹配前缀的 DB-backed 测试并 `t.Fatal`，重跑上面的命令，应当以非零退出；验证后删除探针。
