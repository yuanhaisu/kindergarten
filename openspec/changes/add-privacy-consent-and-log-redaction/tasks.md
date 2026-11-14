# Tasks

## 1. 数据层（migration / domain / repository）

- [x] 1.1 新增 `server/migrations/00005_user_consents.sql`（Up 建表 + 唯一键 `(user_id, policy_version)`，Down 删表）；执行 `make migrate` 后用 `go run ./cmd/dbcheck user_consents` 核对唯一索引与列
- [x] 1.2 新增 `server/internal/domain/consent.go`：`UserConsent`（无 `deleted_at`，字段见 design D1）；`go build ./...` 通过
- [x] 1.3 新增 consent repository：按 `(user_id, policy_version)` 查询、幂等写入（`ON DUPLICATE KEY UPDATE id=id` 后回读首次时间）、按用户列历史；补 repository 测试覆盖首次写入与重复写入
- [x] 1.4 验证：`go build ./...` 与 `go test ./internal/repository -run Consent -count=1` 通过

## 2. 同意服务与门禁（service / middleware / handler / router / wire / Casbin）

- [x] 2.1 新增 consent service：`CurrentPrivacyPolicyVersion = "v1"`、状态查询、记录同意（校验请求版本必须等于当前版本）、`source` 从令牌角色推导；补 service 测试覆盖首次/重复/版本不匹配
- [x] 2.2 在 `server/internal/config` 与 `configs/app.ini.example` / `deploy/config/app.ini.tmpl` 增加 `[consent] enforce`（默认 false），并接入 provider；`go build ./...` 通过
- [x] 2.3 新增 consent 中间件：已登录但未同意当前版本时对业务接口返回 `403` + 可识别错误码；放行 `/common/consent`、`/auth/logout`、`/health`、`/ready`；仅在 `enforce=true` 时拦截；补中间件测试覆盖拦截与放行
- [x] 2.4 新增 `GET/POST /api/v1/common/consent` handler：状态查询、记录同意、错误码映射；补 handler 测试
- [x] 2.5 在 `router.go` 注册 common consent 路由，并为 parent / teacher / admin 业务分组挂 consent 中间件；同步 `configs/rbac_policy.csv` 的 `/common/consent` 规则
- [x] 2.6 三端登录 handler（家长/老师微信登录、管理端密码登录）在响应中增加 `consent_required` 与 `current_policy_version`；补 handler 测试覆盖已同意/未同意
- [x] 2.7 新增 repo/service/handler 的 Wire provider 与接口 `wire.Bind`，执行 `make wire` 并确认 `wire_gen.go` 更新
- [x] 2.8 验证：`go build ./...`、`go test ./internal/service ./internal/middleware ./internal/handler/... -run Consent -count=1` 通过；对照 `privacy-consent` 的 TC-NEW-CNS-0001..0009 逐条走查
- [x] 2.9 同意来源端改为请求头标识：`terminal_from`（parent/teacher/admin）；服务端校验该端在令牌角色中，缺省回退角色推导；admin 客户端补头；补 handler 测试（头优先、头不在角色内 40031、缺省回退）；同步 `backend/api/api-common.md`（2026-11-10，`go test ./internal/handler/http -run Consent` 通过）

## 3. 契约与文档（backend/api）

- [x] 3.1 更新 `backend/api/api-common.md`：新增同意端点、`CONSENT_REQUIRED` 错误码、当前政策版本 `v1`
- [x] 3.2 更新 `backend/api/api-parent`、`api-teacher`、`api-admin` 的登录响应说明，增加 `consent_required` 与 `current_policy_version`（标注为增量字段）
- [x] 3.3 验证：逐字段对照 handler 实际响应与文档，确认无多余字段、无遗漏；`git diff backend/api/` 只包含本次声明字段

## 4. 家长端（parent/）

- [x] 4.1 `src/utils/login.js` 在静默登录后读取 `consent_required`；`src/utils/http.js` 拦截 `403 CONSENT_REQUIRED` 并 `uni.reLaunch` 到同意页；补登录工具的单测或可复现的手工验证步骤（手工步骤见 `docs/handoffs/2026-11-09-consent-e2e-pending.md`）
- [x] 4.2 新增 `src/pages/auth/consent.vue`：展示政策版本、摘要、完整政策入口、勾选、「同意并继续」「暂不同意」；调用 `POST /common/consent`；同意后 `uni.switchTab('/pages/index/index')`（家长端有 tabBar）；`pages.json` 注册页面
- [x] 4.3 完整政策入口用 `<web-view>` 打开官网 `/privacy`（家长端非个人号支持）；URL 走配置，备案前可用测试地址
- [x] 4.4 验证：`npm run build:mp-weixin` 通过；完成「未同意 → 阻断页 → 勾选 → 同意 → 回首页 tab」（2026-11-10：`wechat_automator tap` 持续 120s 超时，改用 `_verify/tap.py` 的 CDP touchStart/touchEnd 在视图层真实点击；勾选后 `.agree-box.is-checked=true`，同意后写入 `user_consents`（user 61 / v1）、页面栈回到 `pages/index/index`，业务接口恢复 200）
- [x] 4.5 验证：对照 `privacy-consent` 的 TC-NEW-CNS-0010、0013、0014 逐条走查（0010 阻断页只有 consent、无 tabBar 业务页；0013 同意后回首页并落库；0014 暂不同意不写记录且停留在阻断页。原生 `uni.showModal` 的「退出登录」确认按钮不在 webview DOM，CDP 点不中，退出分支为代码走查）

## 5. 老师端（teacher/）

- [x] 5.1 `src/utils/login.js` 与 `src/utils/http.js` 同家长端接入同意状态与 403 拦截
- [x] 5.2 新增 `src/pages/auth/consent.vue` 并在 `pages.json` 注册；同意后 `uni.reLaunch('/pages/index/index')`（老师端无 tabBar，**不要用 switchTab**）
- [x] 5.3 完整政策入口：老师端个人号不支持 `<web-view>`，展示端内摘要 + 「复制完整政策链接」；文案明确“完整文本见官网”
- [x] 5.4 验证：`npm run build:mp-weixin` 通过；完成「未同意 → 阻断页 → 勾选 → 同意 → 回工作台」（2026-11-10：`wechat_automator tap` 超时，改用 `_verify/tap.py` 的 CDP 触摸；勾选后 `.agree-box.is-checked=true`，同意后写入 `user_consents`（user 11 / v1）、页面栈回到 `pages/index/index`；业务接口恢复 200）
- [x] 5.5 验证：对照 `privacy-consent` 的 TC-NEW-CNS-0011、0013、0014 逐条走查（0011 阻断页只有 consent、无工作台；0013 同意后回工作台并落库；0014 同家长端：不写记录且停留阻断页，原生弹窗确认按钮为代码走查）

## 6. 管理端（admin/）

- [x] 6.1 auth store 增加 `ensureConsent()`：启动时调用 `GET /common/consent`；`middleware/auth.global.ts` 在未同意且目标不是 `/consent` 时重定向到 `/consent`
- [x] 6.2 新增 `pages/consent.vue`：展示版本、摘要、官网 `/privacy` 与 `/terms` 链接、勾选、同意/暂不同意；同意后写库并回 `/`
- [x] 6.3 登录页处理 `consent_required`：登录响应要求同意时直接进入 `/consent`，不进入业务页
- [x] 6.4 验证：`npm run generate` 通过；Playwright 真实登录未同意账号 → 落到 `/consent` → 勾选同意 → 回到 dashboard；另验证「暂不同意」→ `/login` 且不写记录（2026-11-10）
- [x] 6.5 验证：对照 `privacy-consent` 的 TC-NEW-CNS-0012、0013 逐条走查（0012 登录后进阻断页、0013 同意后进后台首页；0014 在 admin 端完整验证）

## 7. 操作日志脱敏（Go pkg / service / cmd）

- [x] 7.1 新增 `server/pkg/redact`：`Phone` / `IDCard` / `StudentNo` / `Name` / `Text`，`Text` 只匹配手机号与身份证号且幂等；表驱动单测覆盖边界（11 位手机、18 位身份证、已掩码文本、非 PII 数字）
- [x] 7.2 操作日志服务在唯一写入函数中调用 `redact.Text`；`parent_service.go`、`teacher_service.go`、`admin_service.go`、`student_service.go` 的已知写入点改用 `redact.Phone/Name/StudentNo`
- [x] 7.3 新增 `server/cmd/redactlogs`：`-dry-run`（默认 true）、`-batch`、`-limit`；按 `detail REGEXP` 预筛，执行后打印修改行数；可重复执行
- [x] 7.4 补测试：service 写入日志后 `detail` 不含完整手机号/姓名/学号；`redactlogs` dry-run 不写库、执行后历史手机号被掩码、二次执行修改行数为 0
- [x] 7.5 验证：`go test ./pkg/redact ./internal/service -run 'Redact|Consent' -count=1` 通过；在开发库运行 `go run ./cmd/redactlogs -dry-run` 并核对行数；对照 TC-NEW-LGR-0001..0011 逐条走查

## 8. 测试目录、集成与上线

- [x] 8.1 在 `docs/tests/TEST_CATALOG.md` 登记 `TC-NEW-CNS-0001..0014` 与 `TC-NEW-LGR-0001..0011`，每条含前置、步骤、预期
- [ ] 8.2 集成验证（staging，`enforce=true`）：家长/老师/管理端各跑一次「未同意 → 阻断 → 同意 → 业务可用」；并验证旧版客户端在 `enforce=false` 下仍可用
- [x] 8.3 上线顺序写入部署说明：迁移 → 服务端（`enforce=false`）→ 三端发版 → 观察 → `enforce=true` → `cmd/redactlogs` 正式执行（见 `docs/runbook/deployment.md` §2.1）
- [x] 8.4 收工清理：杀掉残留的 `dev:mp-weixin` watcher；确认 `make wire`、`go build ./...`、两端 `build:mp-weixin`、`admin npm run generate` 均为最新（2026-11-09 全部通过）
- [x] 8.5 最终验证：逐条对照 `specs/privacy-consent/spec.md` 与 `specs/operation-log-redaction/spec.md`（2026-11-10）—— CNS-0001..0015 与 LGR-0001..0011 通过；CNS-0008/0009 版本升级由单测覆盖；CNS-0014 家长/老师端原生弹窗确认按钮未自动化，但「不写记录 + 无法进入业务页」已实测，admin 端完整验证；LGR-0011 用 `/admin/logs` 实测 488 条、0 个完整手机号/身份证号。**来源端已修复并复验（2.9）**：user 61 带 `terminal_from=parent` → `source=parent`；user 11 带 `terminal_from=teacher` → `source=teacher`；非法 header 返回 400；缺省 header 回退为角色推导（teacher）。**2026-11-11 再次复验（真实 HTTP + 开发库落库）**：user 61 带 `terminal_from=parent` → `source=parent`，user 11 带 `terminal_from=teacher` → `source=teacher`，user 4 带 `terminal_from=admin` → `source=admin`；user 61 带 `terminal_from=admin`（不在令牌角色内）→ 400/40031 且记录不变；不带 header → 按令牌角色顺序回退。验证后已清除测试同意记录。**staging 集成（8.2）未完成，不得 archive。**
