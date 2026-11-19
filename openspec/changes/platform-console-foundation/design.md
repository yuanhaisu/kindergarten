# Design

## Context

见 `proposal.md` — Why。当前实现的关键约束：

- 平台方是独立信任域（ADR-0014）：独立表、独立登录、`scope=platform`、独立路由树；平台账号在 `platform_admins`，平台审计在 `platform_operation_logs`。
- **平台访问凭证在 `ProvidePlatformService` 中硬编码为 24 小时**（注入的 `atTTL` 被 `_ = atTTL` 丢弃），契约 `01-auth.md` 也写 `expires_in: 86400`；当前没有续期端点。`usePlatformHttp` 在 401 时直接清 Cookie 回登录页。管理端的 `useHttp` 也**没有**自动续期——它只保存 `admin_refresh`，从未调用 `/auth/refresh`。因此平台端的单飞续期是第一次落地的客户端流程，没有现成先例。
- 订单状态 `expired` 由 `deriveOrderStatus` 用 `ExpiresAt`（服务期结束日）判定；`Order` 没有付款到期日。`deriveOrderStatus` 对零值时间没有任何兜底，`now > 零值` 恒真。
- 审计写入全部是事务提交后的 `_ = logs.Create(...)`；`PlatformOperationLog` 没有 `kindergarten_id`、`request_id`，也没有 `result` 字段。失败操作会回滚，因此审计天然只覆盖成功操作。
- `OrderService.List` 对每行单独 `SumByOrder`，是 N+1；`OrderFilter` 只有园所/状态/分销商。
- `PlatformAdminRepo.Count()` 统计全部账号（含已停用、已软删），不能直接用于「最后一个可用账号」判定。
- 佣金 `settleable` 状态靠 `DistributeService.ListCommissions/GetCommission` 里的 `PromoteEligible` 惰性推进；直接查台账仓储的看板会看到未推进的 `pending`。
- `router.go:14-17` 硬编码 `SetTrustedProxies(127.0.0.1, ::1)`。API 跑在容器、Nginx 跑在宿主机时，容器看到的来源是网桥地址而非 127.0.0.1，`ClientIP()` 会得到网关 IP；限流退化为全站共享配额，审计 IP 失真。
- 迁移由 goose 管理，`AutoMigrate` 不参与生产（ADR-0008）；最新迁移号 `00007`。**AutoMigrate 只加列，不回填**，`pay_due_at` 的非空回填必须走 goose。

## Goals / Non-Goals

**Goals**

- 把「付款到期」变成独立于服务期的真实口径，`expired` 与看板、筛选、详情共用同一判定；任何建单路径都不会产生空的付款到期日。
- 财务类与账号类敏感操作的审计与业务同成同败；审计只覆盖成功操作，可按时间/操作人/模块/园所查询。
- 平台账号可管理、可停用、可改密；停用/改密/重置同时使访问与续期凭证失效，续期不能绕过停用。
- 看板只聚合一处，指标定义在服务端固化；待结算佣金不依赖台账页是否被访问。
- 平台页复用现有 `DataTable` / `SearchBar` / `StatCard`，新页直接采用，旧页只改触达部分。

**Non-Goals**

- 不改变平台方的信任域边界（ADR-0014），不引入平台侧多角色 RBAC。
- 不做 ADR-0018 的停用只读、导出与清理（change 2）。
- 不做授权人工调整、佣金结算、发票状态（change 3）。
- 不改老师端与家长端。
- 不新增 design token 或独立视觉。

## 跨端触点清单

| 层 | 会改 / 不改 | 内容 |
|---|---|---|
| Go domain | **会改** | `Order` 增加 `PayDueAt`、`CreditGranted`、`CreditReason`；`PlatformAdmin` 增加 `Role`（内部保留）；`PlatformOperationLog` 增加 `KindergartenID`、`RequestID`；`Order` 增加 `BeforeCreate` 兜底 |
| repository | **会改** | `OrderRepo`：筛选条件扩展、列表改为单次聚合已收金额；`PlatformLogRepo`：`CreateTx` + 多维筛选；`PlatformAdminRepo`：列表、状态更新、改密、可用账号计数（`status=1 AND deleted_at IS NULL`） |
| service | **会改** | `PlatformService`：refresh、账号管理、看板、审计查询；`OrderService`：付款到期/信用开通/审计入事务；`AuthService`：refresh 键按作用域隔离、平台登出吊销续期；相关构造参数（Redis、rtTTL）与接口同步更新 |
| handler | **会改** | `platform.go`、`platform_billing.go`；新增账号、看板、refresh、审计查询的 handler；从 gin context 取 `request_id` 传入 service；平台登出接收 refresh token |
| Casbin policy | **不改** | 现有 `p, platform, /api/v1/platform/*, *` 已覆盖新增路由；平台分组不经过 `adminRoutePermission` 登记表，不需要新增 policy |
| router.go | **会改** | `platformG` 注册新路由；`POST /platform/auth/login` 挂严格限流；可信代理改为从配置读取 |
| backend/api/ | **会改（契约变更）** | `01-auth.md`（refresh、登录响应、TTL）、`02-kindergartens.md`（必填口令、时间格式）、`03-plans.md`（真分页）、`04-orders.md`（付款到期、信用开通、筛选）、新增 `06-audit.md`、`07-accounts.md`、README 端点数 |
| admin 页面 | **会改** | `pages/platform/index.vue` **改为看板**（删除重定向）、新增 `pages/platform/accounts/index.vue`、`pages/platform/logs/index.vue`；`pages/platform/kindergartens/index.vue`、`orders/index.vue`、`plans/index.vue`、`commissions/index.vue`、`login.vue`；`layouts/platform.vue`、`api/platform.ts`、`composables/usePlatformHttp.ts`、`utils/authCookie.ts`、`utils/format.ts`（补 `formatMoney`） |
| teacher / parent | **不改** | 本次不触碰小程序；续期与只读语义都不涉及两端 |
| design token | **不改** | 复用现有 token 与 Tailwind 类名，无新增颜色/间距/圆角 |
| migrations | **会改** | 新增 `00008_platform_console_audit.sql`、`00009_orders_pay_due_at.sql`、`00010_platform_admin_role.sql`、`00011_orders_credit_grant.sql`（见 Migration Plan） |
| config | **会改** | 新增 `[server] trusted_proxies`（逗号分隔，默认 `127.0.0.1,::1`）；限流阈值仍为路由层常量 |

## Decisions

### D1. 付款到期日独立于服务期，且保证非空

`orders.pay_due_at` 在创建时确定，默认等于 `starts_at`，创建请求可显式指定。`deriveOrderStatus` 改为：

```
cancelledAt != nil            -> cancelled
amount_due > 0 && paid >= due -> paid
today > date(pay_due_at) && paid < due -> expired
paid > 0                      -> partial
                              -> unpaid
```

到期日按日期比较：当天仍可付款，次日零时起才算逾期。`expires_at` 只表示服务期，不再参与收款状态。

**非空保证**（防止零值把老订单全判成过期）：`Order` 增加 `BeforeCreate` 钩子，`PayDueAt` 为零值时取 `StartsAt`；`deriveOrderStatus` 对零值再做一次兜底（回退到 `StartsAt`）。任何绕过 `OrderService.Create` 的建单路径（测试、种子、历史数据）都不会因此被判成过期。开发库必须执行 `make migrate` 回填——AutoMigrate 只加列，不回填。

**备选**：继续用 `expires_at` 当付款期限——被否。服务期长达 6 个月，未付款订单会一直显示 `unpaid`，运营无法识别逾期。

**备选**：新增定时任务批量刷新状态——被否。系统是单实例且定时任务不是本次范围；读时派生 + SQL 筛选已足够，且不会在读取路径写业务表。

### D2. 零收款确认 = 信用开通，必须留理由

`POST /platform/orders/:id/confirm` 增加可选 `reason`。当 `SUM(payments.amount_cents) == 0` 时 `reason` 必填，否则 422。确认时写入 `credit_granted=true` 与 `credit_reason`。订单列表展示「信用开通」标签；理由只在详情返回（列表/详情字段分离）。

**备选**：完全禁止零收款确认——被否。对公转账常发生在合同签署后、开课前，禁止会让园所无法开始使用。

**备选**：不落库，只写审计——被否。后续收款到账后无法再判断该订单当初是否信用开通。

### D3. 订单列表单次聚合，状态读时派生

`OrderRepo.List` 改为 `orders LEFT JOIN (SELECT order_id, SUM(amount_cents) paid FROM payments WHERE deleted_at IS NULL GROUP BY order_id)`，返回订单 + `paid_cents`；service 用 D1 的规则派生状态。彻底去掉逐行 `SumByOrder`。

状态筛选在 SQL 中翻译：

- `expired` → `status IN ('unpaid','partial','expired') AND pay_due_at < 当天零点`
- `unpaid` → `status='unpaid' AND pay_due_at >= 当天零点`
- `partial` → `status='partial' AND pay_due_at >= 当天零点`
- `paid` / `cancelled` → 存储状态直接匹配

**备选**：读路径顺手把 `status` 写回数据库——被否。项目已明确禁止在读取路径写业务表。

**风险**：存储的 `status` 与派生状态可能不一致；所有展示与筛选都走派生，存储值只作为 `paid/cancelled` 的事实来源。需要在 `OrderService` 中统一出口，禁止 handler 直接返回 `domain.Order.Status`。

### D4. 平台账号、改密与停用

新增 5 个端点：账号列表、新建、停用/启用、重置密码、修改自己的密码。守卫与规则：

- 「可用账号」定义：`status=1 AND deleted_at IS NULL`；禁用账号不计入。不能停用当前登录账号（用 `claims.UserID` 比对）；停用前统计可用账号，只剩一个时拒绝。
- 新建与重置密码的口令由操作者提供，复用既有 `validatePassword`：至少 8 位，且同时包含大写字母、小写字母与数字；响应不回显口令。口令策略在服务端校验，前端只做提示。
- 重置他人密码必须同时使该账号的访问与续期凭证失效（写 epoch，见 D5）。
- 账号的新建、停用/启用、重置密码必须写审计；审计与账号写入同事务，epoch 写入失败则操作失败。
- 账号列表不返回 `password_hash`；`role` 列内部保留，不在响应中暴露、不参与鉴权。`status` 在契约中暴露为 `enabled` / `disabled`（来源：`PlatformAdmin.Status` 的既有注释 `1 enabled / 0 disabled`）。
- 首个账号仍由 `cmd/platformadmin` 引导；UI 负责后续账号。

**为什么保留 `role`**：已确认的决策是预留列；当前全部为 `platform`，不暴露契约、不参与 Casbin，避免未来加角色时再做一次迁移。design 明确标注它是保留列，防止误以为已生效。

**备选**：现在就做超管/运营/财务三角色——被否。当前 1–2 名使用者，没有分工。

### D5. 会话：短访问凭证 + 可失效的续期

- **访问凭证 TTL 改为 `jwt.access_ttl_min`（默认 15 分钟）**，删除 `ProvidePlatformService` 里硬编码的 24 小时。这是显式行为变更：平台会话从 24 小时缩短到 15 分钟，必须靠续期维持。
- 登录与续期都签发 refresh token，存 Redis：`refresh:platform:{jti}`，值为 `adminID:issuedAtUnix`，TTL 取 `jwt.refresh_ttl_day`（7 天）；园所保持 `refresh:{jti}`，两类键不重叠。
- `POST /platform/auth/refresh` 只消费 `refresh:platform:*`；园所 `/auth/refresh` 只消费 `refresh:*`。避免平台 refresh 被园所端点换出园所凭证。
- **续期必须重新校验账号状态**：账号 `status != 1` 时拒绝；`issuedAt < epoch` 时拒绝。否则停用/改密后，旧 refresh 仍能换出 `iat > epoch` 的新 access，epoch 形同虚设。
- 续期旋转：消费旧 jti 后签发新 access + 新 refresh。
- 改密/停用/重置他人密码写 Redis epoch：`platform:auth_epoch:{adminID} = now`（TTL = refresh TTL + access TTL + 1 天）。**写 epoch 失败时操作 MUST 失败**，不得返回成功。`PlatformAuth` 与平台续期端点都检查 epoch；Redis 不可用时 fail closed（503），与现有 revoker 行为一致。
- **平台退出登录必须吊销续期凭证**：登出请求带 refresh token，服务端删除 `refresh:platform:{jti}` 并把 jti 写入吊销名单。现有 `AuthService.Logout` 的 `RefreshRemainingTTL` 写死 `refresh:` 前缀，需要新增按作用域读取 TTL 的方法。
- 登录限流：10 次/分钟/IP，**失败时 fail closed**。现有 `RateLimit` 在 Redis 出错时 `c.Next()` 失败开放，需要加严格模式或改默认（目前无其他调用方，可直接改为严格）。
- 前端靠 refresh 续期；`useHttp`（园所后台）并没有自动续期，平台端是第一次落地这个客户端流程，没有现成先例。

**备选**：只吊销当前 jti——不满足「此前签发的凭证全部失效」。
**备选**：给 `platform_admins` 加 `token_version` 并在每请求查库——多一次 DB 查询；Redis epoch 更轻。
**备选**：保留 24 小时 access 只加 refresh——安全收益低，且与「短 access + refresh」的园所会话模型不一致。

### D6. 审计与业务同事务

`PlatformLogRepo` 增加 `CreateTx(ctx, tx, row)`。**强一致清单**：开通园所、停用/启用园所、确认订单、取消订单、登记收款/退款、变更授权（change 3），以及平台账号的新建/停用/重置密码——日志与业务同事务，日志失败即回滚。**best-effort 清单**：套餐与分销商目录变更——失败只记 `log.Error`，不阻断业务。

审计只覆盖成功操作：业务失败会回滚，不产生记录；因此不引入 `result` 字段，规格里的「结果」删除或明确为「成功」。

日志新增：

- `kindergarten_id`：目标园所；账号/套餐类操作为 0。
- `request_id`：从 gin context `request_id` 传入，用于关联 `RequestContextLogger` 的链路日志。

查询：`GET /platform/logs` 支持 `from/to/operator/module/kindergarten_id`，分页沿用现有包络；索引 `(kindergarten_id, created_at)` 与 `(operator, created_at)`。

### D7. 看板一次聚合，定义写死在服务端

`GET /platform/dashboard` 返回：

| 指标 | 口径 |
|---|---|
| 园所总数 | 未删除园所计数 |
| 在营园所 | `status='active'` 计数 |
| 30 天内到期 | 在营园所中，存在未删除且 `starts_at<=now<expires_at<=now+30d` 的授权 |
| 本月实收 | `payments.received_at` 落在当前自然月的 `amount_cents` 合计（含负数退款） |
| 未收余额 | 未取消订单的 `max(amount_due - paid, 0)` 合计；`paid` 只计未删除收款 |
| 待结算佣金 | `commission_ledger` 中 `reversal_of IS NULL` 且满足派生条件（`status='settleable'`，或 `status='pending'` 且 `eligible_at<=now` 且净额>0）的 `amount_cents - reversed_amount_cents` 合计 |
| 7 天内到期待办 | 在营园所中最早到期授权落在 7 天内的列表 |
| 可结算佣金待办 | 按分销商聚合的上述派生条件金额列表 |

- 「本月」按服务器本地时区（与现有 `time.Now()` 一致），不引入 UTC 换算。
- 待结算佣金是**读时派生查询**：SQL 条件同时覆盖已推进为 `settleable` 的行与到账已满 30 天但仍为 `pending` 的行；**读路径不写台账状态**，不得在看板里调用 `PromoteEligible`。满 30 天但从未打开过佣金台账页的记录同样计入。
- 「30 天内到期」与园所列表的 `soonest_expire_at` 抽同一查询函数，避免两套口径。

**备选**：前端拼多个列表接口——被否。请求数多且口径会漂移。

### D8. 前端刷新采用单飞（single-flight）

`usePlatformHttp` 响应拦截器遇到 401 时：

1. 若请求本身是 `/platform/auth/refresh` 或重试过，直接清态跳登录。
2. 否则复用同一个 `refreshPromise` 调续期；成功后更新两个 Cookie 并重放原请求；失败则清态跳登录。

`utils/authCookie.ts` 增加 `PLATFORM_REFRESH_COOKIE`（7 天）与对应常量。Cookie 仍不设 HttpOnly（现有架构决定，见文件头注释），靠短 access TTL 降低泄露窗口。

### D9. 页面模式

新增页面必须使用 `admin/components/common/DataTable.vue`、`SearchBar.vue`、`StatCard.vue`；金额格式化统一放 `admin/utils/format.ts` 的 `formatMoney`。`pages/platform/index.vue` 由 8 行重定向改为看板页。旧页面本次只改触达部分（筛选、守卫、操作按钮条件、空/错状态），不做纯风格重写。

### D10. 可信代理改为配置

新增 `[server] trusted_proxies`（逗号分隔，默认 `127.0.0.1,::1`），`router.go` 从配置读取并调用 `SetTrustedProxies`。容器部署必须把网桥网关地址加入，否则 `ClientIP()` 得到网关 IP：限流会退化为全站共享配额，审计 IP 也会失真。上线前在 staging 用真实请求核对 `ClientIP()`（可在日志中打印一次）。

**备选**：限流键改成 `username + IP`——仍依赖正确 IP，不能替代代理配置；可作为补充，不替代。

## 契约字段溯源

| 端点 | 新增/修改字段 | 业务场景 | 不能由已有字段派生的原因 |
|---|---|---|---|
| `POST /platform/orders`、`GET /platform/orders`、`GET /platform/orders/:id` | `pay_due_at` | 财务看板的逾期判定、未收余额筛选 | `expires_at` 是服务期结束日；付款期限与服务期是两件事 |
| `GET /platform/orders`（列表） | `credit_granted` | 列表标识信用开通订单，便于催收 | 收款到账后无法从金额反推确认时的状态 |
| `GET /platform/orders/:id`（详情） | `credit_reason` | 详情/审计展示信用开通理由 | 只在详情渲染，不进列表 |
| `POST /platform/orders/:id/confirm` | `reason` | 零收款确认的审计理由 | 无其他字段可承载 |
| `GET /platform/logs` | `kindergarten_id` | 按园所检索审计 | `target_type/target_id` 指向订单/账号，不能稳定映射到园所 |
| `GET /platform/logs` | `request_id` | 与后端链路日志关联排查 | 现有表无此信息 |
| `GET /platform/logs` | `from/to/operator/module/kindergarten_id` 查询参数 | 审计检索 | 现有只支持 `module` 与分页 |
| `GET /platform/dashboard` | 6 个指标 + 2 个待办列表 | 运营概览与到期/结算待办 | 无现有端点提供跨园聚合 |
| `GET/POST/PATCH /platform/admins`、`POST /platform/admins/:id/reset-password`、`PUT /platform/admins/me/password` | `id/username/name/status/created_at/password` | 账号管理、停用、重置；`status` 取 `enabled`/`disabled`（来源：`PlatformAdmin.Status` 注释） | 现有只有 CLI，无查询与变更入口 |
| `POST /platform/auth/login`、`POST /platform/auth/refresh` | `refresh_token` | 15 分钟 access 过期后的续期 | 现有登录不返回续期凭证 |
| `GET /platform/plans` | `page/page_size/total` | 套餐目录增长后的真实分页 | 契约已声明分页，handler 目前返回全量并伪造 `page_size=total` |
| `POST /platform/kindergartens` | `admin_password` | 开通时交付园长初始口令 | 契约漏写，UI 与后端已要求 |

接口粒度：`credit_reason` 仅详情返回；`request_id` 仅审计列表返回；不把订单详情才用的 `plan_snapshot`、`payments` 塞进列表。平台账号响应**不包含** `password_hash` 与 `role`。

## Migration Plan

实际拆成四个迁移文件，各自 `-- +goose NO TRANSACTION`（MySQL DDL 隐式提交，不包事务）：

**1. `server/migrations/00008_platform_console_audit.sql`** — `platform_operation_logs` 增加 `kindergarten_id`（目标园所，账号/套餐操作为 0）与 `request_id`（链路关联），并建两个检索索引：

```sql
-- +goose Up
ALTER TABLE `platform_operation_logs`
  ADD COLUMN `kindergarten_id` bigint unsigned NOT NULL DEFAULT '0' AFTER `target_id`,
  ADD COLUMN `request_id` varchar(64) NOT NULL DEFAULT '' AFTER `ip`,
  ADD KEY `idx_platform_log_kg_time` (`kindergarten_id`, `created_at`) USING BTREE,
  ADD KEY `idx_platform_log_operator_time` (`operator`, `created_at`) USING BTREE;

-- +goose Down
ALTER TABLE `platform_operation_logs`
  DROP KEY `idx_platform_log_operator_time`,
  DROP KEY `idx_platform_log_kg_time`,
  DROP COLUMN `request_id`,
  DROP COLUMN `kindergarten_id`;
```

**2. `server/migrations/00009_orders_pay_due_at.sql`** — `orders.pay_due_at` 先加为可空列，用 `starts_at` 回填后再置 `NOT NULL` 并建索引（AutoMigrate 只加列不回填）：

```sql
-- +goose Up
ALTER TABLE `orders` ADD COLUMN `pay_due_at` datetime(3) NULL AFTER `expires_at`;
UPDATE `orders` SET `pay_due_at` = `starts_at` WHERE `pay_due_at` IS NULL;
ALTER TABLE `orders`
  MODIFY COLUMN `pay_due_at` datetime(3) NOT NULL,
  ADD KEY `idx_orders_pay_due_at` (`pay_due_at`) USING BTREE;

-- +goose Down
ALTER TABLE `orders`
  DROP KEY `idx_orders_pay_due_at`,
  DROP COLUMN `pay_due_at`;
```

**3. `server/migrations/00010_platform_admin_role.sql`** — `platform_admins.role` 预留列（当前全部 `platform`，不暴露契约、不参与鉴权）：

```sql
-- +goose Up
ALTER TABLE `platform_admins`
  ADD COLUMN `role` varchar(32) NOT NULL DEFAULT 'platform' AFTER `status`;

-- +goose Down
ALTER TABLE `platform_admins` DROP COLUMN `role`;
```

**4. `server/migrations/00011_orders_credit_grant.sql`** — 订单信用开通标记与理由（依赖 00009 已加的 `pay_due_at`）：

```sql
-- +goose Up
ALTER TABLE `orders`
  ADD COLUMN `credit_granted` tinyint(1) NOT NULL DEFAULT '0' AFTER `pay_due_at`,
  ADD COLUMN `credit_reason` varchar(255) NOT NULL DEFAULT '' AFTER `credit_granted`;

-- +goose Down
ALTER TABLE `orders`
  DROP COLUMN `credit_reason`,
  DROP COLUMN `credit_granted`;
```

部署顺序：`make migrate` → `make seedbase`（如需）→ 启动后端 → `go run ./cmd/dbcheck orders`、`go run ./cmd/dbcheck platform_operation_logs` 核对索引与行数。**开发库与测试基座也必须先跑 `make migrate`**：AutoMigrate 只加列，不会回填 `pay_due_at`，否则老订单会被判成过期。回滚先回代码再按 00011 → 00010 → 00009 → 00008 逆序执行 Down；`pay_due_at` 回滚会丢失付款期限，需在回滚前导出。

配置新增（`configs/app.ini.example`，不提交真实值）：

```ini
[server]
trusted_proxies = 127.0.0.1,::1
```

容器部署时把网桥网关地址（如 `172.17.0.1`）加入；上线前核对一次 `ClientIP()`。

## Risks / Trade-offs

- [访问凭证从 24 小时缩短到 15 分钟] → 续期成为强依赖；前端单飞必须一次成功，续期接口故障会导致频繁重新登录。上线前先在 staging 验证续期链路。
- [续期绕过停用/改密] → 已在 D5 修复：续期校验账号状态与 `issuedAt < epoch`；用 middleware + service 测试覆盖。
- [`pay_due_at` 零值把老订单判成过期] → `BeforeCreate` + `deriveOrderStatus` 兜底 + 开发库执行 goose 回填；三者缺一不可。
- [看板待结算佣金少算] → 聚合查询按派生条件同时命中 `settleable` 与已满 30 天的 `pending`；用「未打开台账页」的测试场景验证，并断言读路径不写回台账状态。
- [审计入事务扩大失败面] → 财务与账号操作本就不该在无审计时成功；失败返回明确错误，由前端提示重试。
- [Redis 不可用时平台接口不可用] → 与现有 token revoker 一致，fail closed；健康检查已覆盖 Redis。限流同样改为 fail closed，避免 Redis 故障时暴力破解无限制。
- [限流 IP 取到网桥地址] → `[server] trusted_proxies` 配置 + staging 实测；否则所有用户共享一个配额。
- [停用最后一个账号的并发竞争] → 事务内计数 + 禁用操作串行化；当前 1–2 名使用者，残余风险极低。
- [refresh token 存在非 HttpOnly Cookie] → 沿用现有架构；access TTL 15 分钟限制泄露窗口；后续若引入 BFF 再改 HttpOnly。
- [`role` 保留列被误认为已生效] → 在 domain 注释、design 与迁移注释三处标注「保留，不参与鉴权」。
- [看板 30 天内到期口径与列表 `soonest_expire_at` 不一致] → 两者都用「在营 + 活跃授权的最早到期日」；实现时抽同一查询函数。

## Open Questions

- 平台账号找回密码目前只能由另一个平台账号重置；是否需要短信/邮件自助找回，待有外部运营人员时再定。
