# AQR-S1b · 对抗性实现审查

- verdict: **PASS**
- score: 95/100
- 源码覆盖: 6 个 `repository/diet_*_repo.go` + 7 个 `diet_*_test.go` + `service/diet_config_service.go` + `handler/http/diet.go` + `app/router_diet_test.go`；修改 `repository/wire.go`、`service/wire.go`、`app/wire_gen.go`、`app/router.go`、`handler/http/handler.go`、`middleware/admin_perm.go`、`middleware/admin_perm_test.go`、`app/router_perm_test.go`、`service/role_catalog.go` + 2 处测试补参
- 审查范围: git diff/untracked + manifest + 契约 23-diet.md + OpenSpec apply guidance + Casbin policy + Wire 生成物 + 仓储/权限/路由调用链
- 基线: `ebe8f21b`（HEAD 未前移，无 commit）
- 环境: 后端切片，无 UI/DevTools；按编排器指示未使用微信工具

## DevTools 证据

- 构建: `cd server && make wire` → exit 0（重跑幂等，sha1 不变）；`go build ./...` → exit 0
- 测试: `go test ./internal/repository -run 'Diet|Food|Recipe|MealPlan' -count=1 -v` → exit 0，12 个顶层测试 PASS / 0 SKIP（真实 MySQL，15.4s）
- 权限: `go test ./internal/middleware -run AdminPerm -count=1 -v` → 5 PASS；`go test ./internal/app -run RouterPerm` → 1 PASS；补充 `-run RouterDiet` → 2 PASS
- Vet/Gofmt: `go vet ./...` → exit 0；`gofmt -l` 27 个 manifest Go 文件 → 无输出
- Secret scan: `node scripts/check-secrets.mjs --files <27 files>` → 0 findings
- 非 gate: `golangci-lint run`（5 个触及包）→ 0 issues
- Token / 正常/空/错误态 UI / Console / 截图: N/A（后端切片，无页面）

## 实现核对

- **六个仓储均为真实实现**：Config（Get + 事务 Save）、Food（列表/筛选/分页/详情/增改删/批量删/按园计数）、Recipe（列表/联想/详情/创建/整单更新/删除/按名查找）、Plan（列表/详情/创建/更新/删除/ReplaceItems）、Template（列表/详情）、Dri（按年龄）。
- **软删除 + 唯一键坑已处理**：Config meals、Recipe/Plan items 均 `Unscoped()` 硬删后重插；测试用物理行数断言锁回归（连续保存两次后 config meals 仍 6 行）。
- **NULL 营养语义**：Food 27 项指针；`Update` 用 `Select("*")` 让 nil 落 NULL；测试覆盖 NULL 读回与整体替换回 NULL，实测 PASS。
- **租户隔离**：Food 查询强制 `kindergarten_id`；kg=0 预置行不进租户列表、跨园详情/删除按不存在处理。
- **Service 骨架边界**：只有构造函数、`ErrDietNotImplemented`、`GetConfig`/`SaveConfig` 直接返回错误；未实现默认值/时间递增/占比校验/快照。
- **Handler/路由**：GET/PUT `/admin/diet/config` → 501 + `{code:50100}` 统一包络（`// TODO(AQR-S2)`）；`/admin/diet` 组只注册这 2 条，其余 25 条不建 stub。
- **权限**：`diet:view`/`diet:manage` 登记到 fail-closed 表；role catalog 新增 diet 模块（view/manage/export）；测试证明两码互不替代。
- **Wire**：repository.Set 六 provider、service.Set 注册 DietConfigService；`app/wire.go` 已含两个 Set，无需改动；`wire_gen.go` 仅由 make wire 生成。
- **Casbin**：admin 已有 `/api/v1/*` 通配策略，无需改 policy（configs/ 亦在禁止清单）。

## Scope 备注

- manifest 27 文件（16 新增 + 11 修改）与 `git status` 完全一致；无 staged、无 commit。
- 两处清单外测试补参（`app/router_test.go`、`service/admin_album_test.go`）判 **necessary-minimal**：NewHandler 签名变化后不补则 `go vet ./...`（gate 6）失败，每处仅 1 个 `nil`。
- `handler/http/handler.go` 的 123 行 gofmt 重排判 **necessary-minimal**：HEAD 上该文件本就不满足 gofmt，gate 7 要求 slice 文件干净；`git diff -w` 仅 3 处语义新增。
- 禁止路径（domain/、backend/api/、configs/、三端前端）无本 slice 改动；工作区其余 stat-dirty 文件 mtime 早于 brief 创建时间，未被触碰。

## P1 · 阻塞问题 (0)

无。

## P2 · 建议 (0)

无。两点不构成 finding 的观察：

1. `app/router_diet_test.go` 依赖骨架方法的 nil-receiver 安全性；S2 实现真实读写后必须改为注入真实 service（测试注释已声明）。
2. `DietConfigService.cfg` 在 S2 前只写不读，仅为 Wire 依赖图保留；lint 0 issues。

## 结论

9 条 acceptance 独立复跑全绿；manifest 一致；两处 out-of-brief 改动均为编译/门禁必需且语义最小；未发现 P1。判 **PASS**，可进入 commit。
