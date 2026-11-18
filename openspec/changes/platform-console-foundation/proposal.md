# Proposal

## Why

平台方控制台已有「开通园所 → 套餐 → 订单 → 收款 → 分销」的最小闭环，但撑不起真实运营：审计写入是发后不管（`_ = logs.Create(...)`）且没有查询页；平台账号只能靠 `cmd/platformadmin` 创建；订单没有付款到期口径，`expired` 拿服务期结束日判定，导致看板的「未收余额」没有逾期含义；平台会话 24 小时且无续期，录一笔收款中途可能被踢回登录页。即将接入真实园所，先把地基补实。

## What Changes

> **本次将修改 `backend/api/`，属于契约变更。**

- **付款到期口径**：订单新增付款到期日（创建时确定，默认等于服务起始日）；`expired` 从付款到期日推导，不再用服务期结束日。零收款确认订单必须填写理由，订单标记为「信用开通」。
- **契约补齐**：`GET /platform/logs` 契约化；开通园所补齐必填初始口令；套餐列表改为真实分页；`trial_expires_at` 统一 RFC 3339；新增平台会话续期端点与平台账号管理端点。
- **平台账号管理**：列表、新建、停用/启用、重置密码、修改自己的密码；守卫禁止停用自己或最后一个可用账号；停用、改密或重置密码同时使该账号的访问与续期凭证失效；账号操作纳入审计。
- **平台会话**：refresh token（7 天、旋转）+ 登录限流；访问凭证从硬编码 24 小时改为配置值（默认 15 分钟）——这是显式行为变更。续期必须校验账号状态与改密/停用时间，登出必须吊销续期凭证。
- **审计可靠性**：确认订单、收款/退款、取消订单、开通与停用园所、平台账号变更的审计写入与业务写入同事务，失败即操作失败；审计只覆盖成功操作；查询支持时间范围、操作人、模块、园所；记录 `request_id` 以关联后端日志。
- **运营看板**：6 张卡（园所总数、在营、30 天内到期、本月实收、未收余额、待结算佣金）+ 2 个待办列表（7 天内到期园所、可结算佣金）。
- **检索**：订单支持时间范围、订单号、园所、未清余额筛选；佣金支持分销商、月份、状态筛选。
- **页面模式**：平台端新页面统一使用 `DataTable` / `SearchBar` / `StatCard` 与统一金额/日期格式化；旧页面只改本次触达的部分。
- **数据模型**：`orders.pay_due_at`；`platform_operation_logs.kindergarten_id` + `request_id`；`platform_admins.role`（内部保留，当前不参与鉴权、不在契约中暴露）。

## Capabilities

### New Capabilities

- 无。本次全部是既有能力的行为补强，不引入新能力域。

### Modified Capabilities

- `platform-console`：增加平台账号管理、会话续期与登录限流、运营看板、订单/佣金检索；强化「敏感操作可追溯」为同事务写入并支持多维度查询。
- `billing-orders`：增加付款到期日与逾期判定；增加零收款确认订单的信用开通约束。

## Out of Scope

- ADR-0018 的「停用改只读 + 导出」、180 天保留倒计时与清理命令（独立 change `tenant-suspension-readonly`）
- 授权人工调整、佣金结算批次、只读诊断视图、发票状态跟踪（独立 change `platform-commercial-ops`）
- 独立域名与 IP 白名单（保留为部署配置开关，不在本次代码范围）
- 分销商门户、平台方代登录、在线支付、自动续费、自助注册、用量计量、平台级配置/公告、工单客服
- 老师端与家长端不做任何改动；本次只涉及 `admin/` 与 `server/`

## Impact

- **契约**：`backend/api/api-platform/01-auth.md`、`02-kindergartens.md`、`03-plans.md`、`04-orders.md`；新增 `06-audit.md`、`07-accounts.md`
- **Go**：`server/internal/domain/billing.go`；`repository/`（order / platform log / platform admin）；`service/`（platform / order / entitlement 接口签名）；`handler/http/`（platform、platform_billing）；`auth/jwt.go`；`middleware/`（trusted proxies、审计事务适配）；`server/migrations/` 新增迁移
- **Admin**：`admin/pages/platform/`（新增看板、账号、审计页；园所/订单页补筛选与守卫）、`admin/api/platform.ts`、`admin/composables/usePlatformHttp.ts`、`admin/utils/authCookie.ts`、`admin/layouts/platform.vue`
- **Casbin**：不改。现有 `p, platform, /api/v1/platform/*, *` 已覆盖新增路由
- **配置**：`configs/app.ini` 增加 `[server] trusted_proxies`（默认 `127.0.0.1,::1`；容器部署需加入网桥地址），限流阈值仍用路由层常量
- **测试**：`docs/tests/TEST_CATALOG.md` 登记新增 TC；Go service 测试；staging 真实登录验收
