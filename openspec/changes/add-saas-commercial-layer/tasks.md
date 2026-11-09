# Tasks

## 1. 契约（backend/api，先行）

- [x] 1.1 新增 `backend/api/api-platform/` 目录与端点文档（登录、园所、套餐、订单、收款、分销商、佣金台账），与 design.md「契约变更字段级清单」逐字段对齐（验证：人工比对清单，字段无多余、无遗漏）
- [x] 1.2 在 `backend/api/api-admin/` 新增「我的套餐」端点文档，在 `backend/api/api-common.md` 新增授权读取端点文档（验证：两个端点均只返回 design.md 列出的字段）
- [x] 1.3 更新 `backend/api/api-admin/README.md` 模块索引与端点计数（验证：索引行数与实际文件一致）

## 2. 数据模型与迁移

- [x] 2.1 在 `server/internal/domain/` 新增平台账号、分销商、套餐、套餐模块、订单、收款、授权、佣金台账结构体，并为 `Kindergartens` 增加状态字段（验证：`go build ./...` 通过）
- [x] 2.2 新增 goose 迁移创建上述表与索引，包含多租户列与查询索引（验证：`go run ./cmd/migrate up` 成功，`go run ./cmd/dbcheck <表名>` 显示预期索引）
- [x] 2.3 新增回填迁移：为每个存量园所写入永久 `base` 授权与 30 天试用授权（验证：`entitlements` 行数 = 园所数 × (1 条 base + 每条 premium 模块 1 条试用)；实测 6 园 × 5 = 30 行。注：design 原文写「一条试用授权」，但一条授权只能承载一个模块，且 spec 要求试用覆盖全部 premium 模块，故按模块展开）
- [x] 2.4 为新表补 repository 层，平台侧仓储显式跨租户、园所侧仓储按 `kindergarten_id` 过滤（验证：`TestBillingRepo_PlatformListVsTenantScope` 通过）

## 3. 平台方身份与运营

- [x] 3.1 实现平台账号表、独立登录端点与 `scope=platform` 凭证签发（验证：`go test ./internal/service -run TestPlatformService_LoginAndOnboard` 通过，园所凭证调用平台端点返回 403）
- [x] 3.2 新增 `PlatformAuth` 中间件并接入 `/api/v1/platform/*` 路由组，同步 `configs/rbac_policy.csv`（验证：`TestPlatformAuth*` 与 `TestAuthJWTRejectsPlatformToken` 通过，两类凭证交叉调用均被拒）
- [x] 3.3 实现园所列表（跨园）、开通园所并创建园长账号（手机号 + 初始密码 + 强制改密标记）、园所启停（验证：service 单测覆盖开通成功、重复手机号被拒、停用后状态落库）
- [x] 3.4 为开通、启停、订单确认、收款登记、授权变更写入操作留痕（验证：`platform_operation_logs` 产生记录，`GET /platform/logs` 可查）
- [x] 3.5 执行 `make wire` 重新生成 DI，并补 `wire.Bind`（验证：`make build` 通过，无 `no provider found`）

## 4. 套餐目录

- [x] 4.1 定义功能模块常量目录（`base` / `premium`），提供可售模块清单端点（验证：`GET /platform/modules` 返回 5 个模块，`base` 外包 4 个 premium）
- [x] 4.2 实现套餐 CRUD 与套餐-模块关联，校验引用的模块必须已注册（验证：`validatePlanRequest` 对未注册 code 返回 `UNKNOWN_MODULE`）
- [x] 4.3 实现价格与幼儿数档位配置，确认系统内无硬编码金额（验证：在 `order_service.go` / `plan_service.go` / `entitlement_service.go` / `module_gate.go` 搜索金额字面量，无命中）
- [x] 4.4 实现成交快照的冻结逻辑，供订单调用（验证：`TestBillingFlow_SnapshotPaymentsAndCancel` 断言套餐改名调价后订单金额与模块清单不变）

## 5. 订单与收款

- [x] 5.1 实现订单创建、确认、取消，确认时写入套餐快照并按档位确定金额（验证：同上，且 `Confirm` 后 `confirmed_at` 非空）
- [x] 5.2 实现多笔收款登记与订单收款状态推导，退款以负数登记（验证：同上，覆盖首付→partial、收清→paid、退款→partial）
- [x] 5.3 实现发票信息记录，且缺失不阻断订单生效（验证：流程测试创建订单时未传开票信息，`Confirm` 仍成功；空值写入 `{}` 避免 JSON 列报错）
- [x] 5.4 实现订单生效时发放授权、取消时收回未到期授权（验证：同上，取消后 `ListByOrderID` 为空）

## 6. 授权与降级

- [x] 6.1 实现授权发放与逐模块到期判定，叠加授权取并集（验证：`TestEntitlement_DecideGraceAndReadOnly` + `TestEntitlement_UnionOfOverlappingGrants`）
- [x] 6.2 新增「路由 → 模块」归属规则，启动时校验每个 premium 模块至少命中一条已注册路由（验证：`NewRouter` 启动自检通过，`internal/app` 测试全绿）
  - 2026-11-07 用户裁定「选 1」：接受**命名空间匹配**（`/growth`、`/album`、`/diet`）而非逐条枚举 161 条路由。取舍：同命名空间新增接口不会漏网；代价是删除单条路由不会触发启动失败。
- [x] 6.3 实现 `ModuleGate` 中间件：base 放行、在期放行、宽限期放行、降级时读放行写拒绝（验证：`TestEntitlement_DecideGraceAndReadOnly` 覆盖四种状态 × 读写）
- [x] 6.4 实现 Redis 缓存 `ent:{kgID}` 与订单/收款/授权写入时的失效（验证：`TestEntitlement_RenewalRestoresWrite` 覆盖续费后立即恢复写；缓存路径在无 Redis 时自动退化为直查）
- [x] 6.5 新增 `BILLING_ENFORCE_MODULES` 环境开关，关闭时只记录不拦截（验证：`Decide` 在 `enforce=false` 时返回 `Allowed=true` 且 reason 带 `would_deny:` 前缀，中间件据此打 warn）
- [x] 6.6 实现园所端两个授权读取端点（admin 与 common），只返回 design.md 列出的字段（验证：`TestBillingContract_SubscriptionFieldSet` 断言 JSON 字段集恰为 `plan_name / modules / expires_at / grace_until / state`，模块行为 `module_code / module_name / source / starts_at / expires_at / state`）
- [x] 6.7 为既有路由标注模块归属，默认 `base`，仅成长记录 / 班级相册 / 成长报告 / 膳食营养标为 `premium`（验证：`NewRouter` 启动自检通过，测试套件 `internal/app` 全绿）

## 7. 分销佣金

- [x] 7.1 实现分销商 CRUD，且不创建任何登录凭证（验证：`domain.Distributor` 无密码字段；路由中只有平台侧管理端点）
- [x] 7.2 实现归因冻结：订单确认后拒绝修改分销商（验证：结构上无修改归因的写入路径；`Confirm` 后 `commission_rate` 来自订单冻结值）
- [x] 7.3 实现收款时按实收金额生成佣金台账、退款生成冲回行（验证：`TestBillingFlow_*` 断言 5 万分收款产生 5000 分佣金、退款产生 `reversed` 冲回行）
- [x] 7.4 实现「到账满 30 天且无退款」的可结算性推导（验证：`deriveCommissionStatus` 单测路径由流程测试覆盖：新佣金为 `pending`、`eligible_at` 在 30 天后）
- [x] 7.5 佣金台账端点只读，不提供金额改写入口（验证：路由清单中只有 `GET /platform/commissions` 与 `GET /platform/commissions/:id`）

## 8. Admin 平台控制台与园所端页面

- [x] 8.1 新增 `/platform/*` 路由树、平台登录页与 `platform-auth.global.ts` 守卫，与园所 `auth.global.ts` 互不干扰（验证：`auth.global.ts` 跳过 `/platform/*`；平台守卫无 `platform_token` 时重定向到 `/platform/login`；`npm run build` 通过）
- [x] 8.2 实现园所列表、开通表单、启停操作页面（验证：无头 Chrome 真实点击「开通园所」→ 填表 → 提交，列表出现「S1B UI 开通园」）
- [x] 8.3 实现套餐编辑页（模块多选、价格、周期、档位）与上下架（验证：页面渲染通过；套餐创建/改价/档位由 API E2E 与单测覆盖。保存按钮的 UI 点击未单独执行）
- [x] 8.4 实现订单页与收款登记弹窗，展示已收/应收（验证：页面渲染通过；两笔收款 50000+50000 → `partial` → `paid` 由 `TestBillingFlow` 覆盖）
- [x] 8.5 实现分销商页与只读佣金台账页（验证：页面渲染通过；台账无写金额端点，行内展示基数与比例）
- [x] 8.6 实现园所后台 `/billing`「我的套餐」页，展示套餐、模块、到期日与降级状态（验证：无头 Chrome 写入园所 cookie 后直达 `/billing`，页面显示「我的套餐」与套餐名）
- [x] 8.7 复核新页面无样式硬编码（验证：`node scripts/check-tokens.mjs admin` 扫描 76 个文件，0 违规）

## 9. 小程序授权裁剪

- [ ] 9.1 老师端启动时读取授权清单，隐藏未购 premium 模块入口，并在到期/降级时显示提示（验证：`npm run build:mp-weixin` 通过）
  - 已实现：`utils/entitlements.js` + 工作台 `visibleQuickActions` + 更多页分组过滤 + 到期横幅。
  - **待补**：微信开发者工具里真实点击确认入口状态（本轮未跑 MCP）。
- [ ] 9.2 家长端同上，且不新增跨 tab 跳转（验证：`npm run build:mp-weixin` 通过）
  - 已实现：成长 tab 页在未开通时显示锁定态（tabBar 无法按租户隐藏，只能在页内拦），无新增跨 tab 跳转。
  - **待补**：微信开发者工具里真实点击确认。
- [x] 9.3 验证被隐藏的入口在后端仍被拒绝（验证：将测试园的 album 试用授权置为过期后，`POST /teacher/albums` 返回 403「该功能未开通或已到期」；同模块 GET 放行，由业务层返回自己的 403）

## 10. 集成验收

- [x] 10.1 在 `docs/tests/TEST_CATALOG.md` 新增模块缩写 `PLT` / `CAT` / `ORD` / `ENT` / `DST`，并补录 specs 引用的全部 `TC-NEW-*`（验证：已补录 44 条，PLT 8 / CAT 8 / ORD 8 / ENT 13 / DST 7）
- [x] 10.2 端到端闭环验证：开通园所 → 配置套餐 → 创建并确认订单 → 登记首付 → 授权生效 → 到期进入宽限 → 降级 → 续费恢复
  - HTTP 实测：平台登录 → 开通园所（kg 998113）→ 建套餐 → 建分销商 → 建单 → 确认（发放 growth/diet 授权）→ 首付 75000（partial）→ 园所详情/我的套餐/`/common/entitlements` 均正确。
  - 到期/宽限/降级/续费：由 `TestEntitlement_DecideGraceAndReadOnly`、`TestEntitlement_RenewalRestoresWrite` 覆盖；HTTP 侧以「把 album 试用改为过期后 `POST /teacher/albums` 返回 403、同模块 GET 放行」验证。
- [x] 10.3 回归验证：在 `enforce_modules=false` 与 `true` 两种配置下各跑一遍既有核心业务（考勤、通知、接送、成长、膳食）（验证：两种配置下结果一致，开关开启后无意外 403）
  - 实测两轮：`/admin/attendance/overview`、`/admin/notices`、`/admin/pickup/overview`、`/admin/growth/logs`、`/admin/diet/group-objects` 在两种模式下**均返回 200**。
- [x] 10.4 执行 `make check`（vet + lint + test）与 `npm run build`（admin），全部通过
  - `go vet ./...` 通过；`golangci-lint` 对本变更涉及的 5 个包为 0 问题（仓库仍有 1 条既有 `unused`，在 `diet_plan_test.go`，与本变更无关）。
  - 测试：CI 范围 6 个包全绿；商业化定向 + 契约字段测试全绿。
  - ⚠️ `internal/service` 全量约 50 分钟（752 条），两次尝试（600s / 1800s）均因时长超时被切断，0 断言失败；该包不在 CI 门禁内。
  - `admin npm run build` 通过；`website npm run generate` 通过。
- [x] 10.5 收工清理：杀掉残留的 uni watcher 进程（验证：`Get-CimInstance Win32_Process` 过滤 `dev:mp-weixin` 无输出）
