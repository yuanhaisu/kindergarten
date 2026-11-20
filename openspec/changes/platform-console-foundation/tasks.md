# Tasks

## 1. 契约（backend/api，先改 SoT）

- [ ] 1.1 在 `backend/api/api-platform/01-auth.md` 增加 `POST /platform/auth/refresh`，登录响应补 `refresh_token`，把 `expires_in` 改为 900 并注明「平台访问凭证从 24 小时缩短为 15 分钟」；写明 401/429 语义；核对文档与 `PlatformLogin` 一致
- [ ] 1.2 在 `02-kindergartens.md` 补 `admin_password` 为必填项，删除「初始密码由系统生成」的陈旧说明；把 `trial_expires_at` 统一为 RFC 3339；用 `rg " admin_password|trial_expires_at" backend/api/api-platform/02-kindergartens.md` 核对
- [ ] 1.3 在 `03-plans.md` 对齐真实分页（`page/page_size/total`），并在 README 修正端点总数；用 `rg "page_size|total" backend/api/api-platform/03-plans.md` 核对
- [ ] 1.4 在 `04-orders.md` 增加 `pay_due_at`（创建/列表/详情）、确认请求 `reason`、`credit_granted`（列表/详情）、`credit_reason`（仅详情）、列表筛选参数；用 `rg "pay_due_at|credit_granted|credit_reason|outstanding" backend/api/api-platform/04-orders.md` 核对
- [ ] 1.5 新建 `backend/api/api-platform/06-audit.md`，定义 `GET /platform/logs` 的筛选参数与返回字段（含 `kindergarten_id`、`request_id`，**不设 result 字段**），并说明审计只覆盖成功操作
- [ ] 1.6 新建 `backend/api/api-platform/07-accounts.md`，定义账号列表、新建、停用/启用、重置密码、修改自己密码 5 个端点；`status` 取 `enabled`/`disabled`，响应不含 `password_hash` 与 `role`；写明口令策略与守卫错误码
- [ ] 1.7 更新 `backend/api/api-platform/README.md` 的端点总数与模块索引（当前写 23，实际需按新增后重算）；用 `rg -c "^### " backend/api/api-platform/*.md` 与 README 对照
- [ ] 1.8 验证：`rg "platformG\.(GET|POST|PATCH|PUT|DELETE)" server/internal/app/router.go` 列出的每条路由都能在 `backend/api/api-platform/` 找到对应文档，且无反向遗漏；proposal 仍保留「本次修改 `backend/api/`，属于契约变更」的声明

## 2. 数据模型与迁移

- [ ] 2.1 在 `server/internal/domain/billing.go` 给 `Order` 增加 `PayDueAt time.Time`、`CreditGranted bool`、`CreditReason string`，并加 `BeforeCreate` 钩子（零值时取 `StartsAt`）；给 `PlatformAdmin` 增加 `Role string`（注释注明「保留，不参与鉴权」）；给 `PlatformOperationLog` 增加 `KindergartenID uint64`、`RequestID string`；`go build ./...` 通过
- [ ] 2.2 新增 `server/migrations/00008_platform_console_audit.sql`（审计 `kindergarten_id`/`request_id` 两列与两个索引）、`00009_orders_pay_due_at.sql`（`pay_due_at` 回填后置 NOT NULL + 索引）、`00010_platform_admin_role.sql`（`role` 列）、`00011_orders_credit_grant.sql`（`credit_granted`/`credit_reason`），按 `design.md` 的 Migration Plan 写 Up/Down；`go run ./cmd/migrate status` 显示四个迁移 pending
- [ ] 2.3 执行 `make migrate`，再运行 `go run ./cmd/dbcheck orders` 与 `go run ./cmd/dbcheck platform_operation_logs`，确认新列与索引存在；用一条 `SELECT COUNT(*) FROM orders WHERE pay_due_at IS NULL` 确认回填后为零
- [ ] 2.4 验证：`go build ./...`；`make migrate` 无报错；Down 语句在一次性测试库回滚一次后 `dbcheck` 确认列已删除（禁止在生产库执行 Down）

## 3. 后端：订单、付款与审计事务

- [ ] 3.1 扩展 `repository.OrderFilter`（`DateFrom/DateTo/OrderNo/OutstandingOnly`），把 `OrderRepo.List` 改为单次 `LEFT JOIN` 聚合已收金额（排除软删收款），去掉 `OrderService.List` 的逐行 `SumByOrder`；用 `go test ./internal/repository -run Billing` 验证列表总数与金额仍正确
- [ ] 3.2 `deriveOrderStatus` 改用 `PayDueAt`，并对零值兜底到 `StartsAt`；`OrderService.List/Detail` 统一返回派生状态；在 `OrderService` 内加注释禁止 handler 直接返回存储状态；用测试覆盖 `unpaid → expired`、`partial → expired` 与「零值到期日不被判过期」
- [ ] 3.3 `OrderService.Confirm` 在净收款为零时要求 `reason`，写入 `credit_granted/credit_reason`；`Create` 接受可选 `pay_due_at`，缺省等于 `starts_at`；用测试覆盖「缺理由 422」「有理由成功且标记信用开通」「未指定到期日取服务起始日」
- [ ] 3.4 `PlatformLogRepo` 增加 `CreateTx`；把 `Confirm/Cancel/AddPayment` 的日志写入移入既有事务，失败即回滚，并写入 `kindergarten_id` 与 handler 传入的 `request_id`；用测试验证审计写失败时业务不落库（通过可控失败的审计写入接口注入）
- [ ] 3.5 更新 `handler/http/platform_billing.go`：解析 `reason`、`pay_due_at` 与筛选参数；把 `request_id` 从 gin context 传给 service；`go build ./...` 通过
- [ ] 3.6 在 `docs/tests/TEST_CATALOG.md` 登记 `TC-NEW-ORD-0014`（付款到期判逾期）、`0015`（零收款缺理由被拒）、`0016`（服务期变更不改到期日）、`0017`（信用开通计入未收余额并留痕）、`0018`（未指定到期日取服务起始日）
- [ ] 3.7 验证：`go test ./internal/service -run "Order|Billing" -timeout 600s` 与 `go test ./internal/repository -run Billing -timeout 600s` 全绿（race 需要时按 `go-race-64bit-mingw` 设置 CC）

## 4. 后端：平台账号、会话、看板、审计查询

- [ ] 4.1 `PlatformAdminRepo` 增加分页列表、状态更新、改密、可用账号计数；**可用账号口径必须是 `status=1 AND deleted_at IS NULL`**，不得复用统计全部账号的 `Count`；用仓储测试验证禁用账号不计入
- [ ] 4.2 `PlatformService` 增加账号列表/新建/停用启用/重置密码/改自己密码；实现「不能停用自己」「不能停用最后一个可用账号」守卫（最后一个账号守卫在事务内复核，且在写 epoch 之前）；口令复用既有 `validatePassword`：至少 8 位，且同时包含大写字母、小写字母与数字；重置他人密码同时写 epoch 使目标账号访问与续期失效；账号的新建/停用/重置写审计并与账号写入同事务；用服务测试覆盖守卫、口令策略、审计与凭证失效
- [ ] 4.3 `auth` 增加按作用域保存/消费 refresh（平台键 `refresh:platform:{jti}`，值存 `adminID:issuedAtUnix`；园所保持 `refresh:{jti}`）；`PlatformService.Login` 签发 refresh，新增 `Refresh` 服务方法与 handler；**续期必须校验 `status=1` 且 `issuedAt >= epoch`**；平台登出接收 refresh 并删除/吊销平台键；新增按作用域读取 TTL 的方法（现有 `RefreshRemainingTTL` 写死 `refresh:`）；用测试验证平台 refresh 不能在园所 `/auth/refresh` 兑换、停用后旧 refresh 被拒、登出后 refresh 被拒
- [ ] 4.4 `middleware.PlatformAuth` 增加 epoch 校验：`claims.IssuedAt < epoch` 返回 401，Redis 错误 fail closed 503；账号停用/改密/重置时 epoch 写入失败必须让操作失败，不得返回成功；用中间件与服务测试覆盖 epoch 前后与写失败
- [ ] 4.5 新增 `PlatformService.Dashboard` 与 handler，按 `design.md` D7 的口径一次聚合 6 个指标与 2 个待办列表；**待结算佣金走读时派生查询**（`settleable` 或已满 30 天的 `pending`），**不得在读路径调用 `PromoteEligible` 或写回台账状态**；「本月」按服务器本地时区；「30 天内到期」与园所列表 `soonest_expire_at` 抽同一查询函数；用测试构造两所园所与订单，核对概览与明细一致，并覆盖「未打开台账页也计入待结算佣金」且台账状态保持 pending
- [ ] 4.6 `PlatformLogRepo.List` 支持 `from/to/operator/module/kindergarten_id` 与分页；`PlatformService.ListLogs` 与 handler 接通；在实现说明中记录「审计至少保留 3 年，本期不自动清理」；用测试验证按园所与时间过滤
- [ ] 4.7 给 `POST /platform/auth/login` 挂严格限流（10 次/分钟/IP）：现有 `RateLimit` 在 Redis 出错时失败开放，需加严格模式或改默认；用中间件测试覆盖超限 429 与 Redis 故障时的行为
- [ ] 4.8 在 `docs/tests/TEST_CATALOG.md` 登记 `TC-NEW-PLT-0009`～`0025`（账号新建/停用/改密失效、两条账号守卫、续期、停用后续期被拒、登出后续期失效、限流、看板三例、审计三例、订单检索、佣金检索）
- [ ] 4.9 验证：`go test ./internal/service -run "Platform|Account|Dashboard|Audit" -timeout 600s` 与 `go test ./internal/middleware -timeout 600s` 全绿；Redis 相关测试使用 `app/router_test.go` 的假 Redis 或等价测试替身，**不得连开发环境 Redis**

## 5. 后端：配置、路由、Wire 与启动

- [ ] 5.1 在配置结构体与 `configs/app.ini.example` 增加 `[server] trusted_proxies`（默认 `127.0.0.1,::1`），`router.go` 改为从配置读取并调用 `SetTrustedProxies`；用 `go build ./...` 与启动日志验证
- [ ] 5.2 在 `router.go` 的 `platformG` 注册 refresh、dashboard、accounts 与审计筛选路由；给登录挂严格限流；确认 Casbin 通配策略覆盖新路由（不改 `rbac_policy.csv`），用 `make run` 启动后逐条 `curl` 返回非 403
- [ ] 5.3 更新 `internal/app/wire.go` 的 provider 与 `wire.Bind`（PlatformService 新增 Redis/rtTTL 依赖、PlatformAuth 新增 epoch 依赖），执行 `make wire`；`git diff internal/app/wire_gen.go` 只包含预期的构造变化
- [ ] 5.4 验证：`make build` 成功；`make run` 启动后 `GET /health` 与平台登录成功；新增 7 个端点（refresh/dashboard/5 个账号）均能返回预期包络；在 staging 反代后确认日志中的 `ClientIP()` 是真实客户端地址而不是网桥网关
- [ ] 5.5 验证：`go vet ./...` 与 `golangci-lint run ./...` 无新增告警

## 6. Admin 平台页

- [ ] 6.1 `admin/utils/authCookie.ts` 增加 `PLATFORM_REFRESH_COOKIE`（7 天）与常量；`admin/tests/unit/authCookie.test.ts` 补断言
- [ ] 6.2 `admin/composables/usePlatformHttp.ts` 实现单飞 refresh：401 时调 `/platform/auth/refresh`、成功后更新 access/refresh Cookie 并重放原请求、失败清态跳登录；用单元测试或本地手工验证并发 3 个 401 只触发一次 refresh
- [ ] 6.3 `admin/api/platform.ts` 增加 refresh、账号管理、看板、审计查询、订单筛选的类型与函数；`npm run build` 通过
- [ ] 6.4 `admin/utils/format.ts` 增加 `formatMoney`；平台页改用共享实现，`rg "function formatMoney" admin/pages/platform` 无残留本地副本
- [ ] 6.5 `admin/pages/platform/index.vue` **删除 8 行重定向，改为运营看板**：6 张 `StatCard` + 2 个待办列表；用 headless CDP 或本地页面验证数字与接口返回一致
- [ ] 6.6 新增 `admin/pages/platform/accounts/index.vue`：列表（`enabled`/`disabled`）、新建、停用/启用、重置密码、改自己密码；停用自己/最后一个账号时前端提示与后端错误一致；重置成功后当前账号若被重置应被登出
- [ ] 6.7 新增 `admin/pages/platform/logs/index.vue`：时间范围、操作人、模块、园所筛选 + 分页；验证筛选条件变化会重置到第 1 页
- [ ] 6.8 `pages/platform/kindergartens/index.vue`：补状态筛选、搜索重置页码、停用二次确认、详情抽屉刷新；验证停用操作有确认弹窗且取消不发送请求
- [ ] 6.9 `pages/platform/orders/index.vue`：补时间范围/订单号/园所/未清余额筛选；`paid`/`expired` 隐藏**确认**，`cancelled` 隐藏全部，**`expired` 且净收款为零仍允许取消**；展示 `pay_due_at` 与「信用开通」标签；确认弹窗在零收款时要求填写理由
- [ ] 6.10 `pages/platform/plans/index.vue` 接入真实分页与状态筛选；`pages/platform/commissions/index.vue` 补分销商筛选与明细抽屉（复用 `getCommission`）；验证分页切换后列表与总数变化
- [ ] 6.11 `layouts/platform.vue` 增加「看板 / 账号 / 审计」菜单；`pages/platform/login.vue` 处理 429 与已登录跳转；验证菜单高亮与退出登录
- [ ] 6.12 验证：`cd admin; npm run generate` 成功（生产静态产物与 `postgenerate` 检查走 generate，不是 build）；`node scripts/check-tokens.mjs admin` 无新增违规；用真实登录 + 真实点击完成看板、账号、审计、订单筛选四条路径（不能用页面加载代替交互）

## 7. 集成验收

- [ ] 7.1 更新 `docs/tests/TEST_CATALOG.md`：`TC-NEW-PLT-0001/0002/0003/0004/0007/0008` 标为已验证；`0005/0006` 保持未验证并注明由 change 2 重写；新增 `ORD-0014～0018`、`PLT-0009～0025` 全部有对应实现
- [ ] 7.2 在 staging 用 `cmd/platformadmin` 造平台账号，并把 `access_ttl_min` 调到 1 后完成端到端：登录 → 等 access 过期 → 自动续期 → 停用后旧 refresh 被拒 → 登出后 refresh 被拒 → 看板 → 新建账号 → 重置密码 → 审计查询 → 订单筛选 → 零收款确认；记录每步响应与截图
- [ ] 7.3 逐条对照 `specs/platform-console/spec.md` 与 `specs/billing-orders/spec.md` 的每个 Scenario，确认通过；任何一条不通过则停下修 spec 或实现，不得带病宣布完成
- [ ] 7.4 验证门：`make check`（vet + lint + test，测试加 `-timeout 600s`）与 `cd admin; npm run generate` 全绿；`git diff --stat` 核对只触达 design 列出的层；契约门逐条对照路由清单与 `backend/api/api-platform/`
- [ ] 7.5 生成 checkpoint 摘要：实际触达层 vs design 触点清单、迁移是否已执行、`trusted_proxies` 在 staging 的实际取值、未验证项与原因；确认无残留后端进程后结束

## 执行记录（T9 收口，2026-11-19）

### 已完成

- 7.1 已登记 `TC-NEW-ORD-0014～0018`、`TC-NEW-PLT-0009～0025`；仅「实际跑过并通过」的用例标 ✅。
- 7.4 本地门禁：`go vet ./...`、`golangci-lint run ./...`、`go test ./... -race -timeout 600s`（DB 用例 SKIP）、`cd admin && npm run generate`、`cd admin && npm run test`、`node scripts/check-tokens.mjs admin` 全部通过；`platformG` 32 条路由与 `backend/api/api-platform/` 的 32 个接口逐条对应，无未文档化路由、无反向缺口。
- 7.5 checkpoint 摘要已随 #122 交付说明给出：实际触达层与 design `跨端触点清单` 一致（admin 布局/页面、后端 middleware/config/router、契约文档、ADR、测试目录、部署文档）；迁移 `00008～00011` 已在树内但**未执行**（本地 MySQL 未运行）；`trusted_proxies` 配置默认为 `127.0.0.1,::1`，staging 实际取值未验证。

### 阻塞未完成

- **7.2 staging 端到端未执行**：本机 MySQL(3316)/Redis(6379)/后端(8080) 均无监听，且当前 worktree 无 `server/configs/app.ini`；没有可用平台账号与迁移后的库，无法真实登录、续期、看板与订单操作。**不得以本地 mock 或页面加载代替。**
- **7.3 spec 逐 Scenario 验收未完成**：依赖 7.2 的真实运行结果与 DB 用例（本机无 DB，相关 service/repository 测试 SKIP）。

### 恢复执行所需的命令

```bash
# 1) 起依赖：MySQL 监听 127.0.0.1:3316、Redis 监听 6379（本机服务/容器均可）
# 2) 准备配置（不要提交）：
cd server
cp configs/app.ini.example configs/app.ini   # 填 MYSQL_DSN / REDIS_ADDR / JWT_SECRET
make migrate && make seedbase

# 3) 造平台账号（账号已存在时只重置密码）
PLATFORM_ADMIN_USERNAME=ops PLATFORM_ADMIN_PASSWORD="<至少8位字母数字>" go run ./cmd/platformadmin

# 4) 为验证续期，临时把 [jwt] access_ttl_min 调为 1，再启动后端
CONFIG=configs/app.ini make run

# 5) 管理端静态产物 + 反代到 8080
cd ../admin && npm run generate
# 用 nginx 或等价方式把 /api/v1 反代到 127.0.0.1:8080

# 6) 真实交互（headless Chrome CDP，不能只用页面加载）：
#    登录 → 等 access 过期 → 自动续期 → 停用后旧 refresh 被拒 → 登出后 refresh 被拒
#    → 看板 → 新建账号 → 重置密码 → 审计查询 → 订单筛选 → 零收款确认（填理由）

# 7) 回归 DB 用例
cd ../server && go test ./internal/service ./internal/repository -run "Platform|Order|Billing" -timeout 600s -v
```

> staging 部署侧另需确认 `[server] trusted_proxies` 实际取值（反代/网桥网关地址），
> 并在日志中核对 `ClientIP()` 为真实客户端地址。
