# Design

## Context

见 `proposal.md` — Why。设计受以下既有事实约束：

- 全链路以 `kindergarten_id` 隔离，`middleware/context.go` 从 JWT 取当前园所；仓储默认按园所过滤。
- 网关鉴权两层：Casbin 做角色 → 路由前缀粗放行，`middleware/admin_perm.go` 做路由 → 权限码细校验，且启动时强制登记表全覆盖。
- JWT 只带 `UserID` / `Roles` / `KindergartenID`，没有任何订阅或模块 claim。
- Schema 由 goose 迁移管理，应用启动不再执行 `AutoMigrate`（ADR-0008）。
- Wire DI 需要 `wire.go` + `wire_gen.go` 同步；service 接口依赖必须补 `wire.Bind`。

## Goals / Non-Goals

**Goals:**

- 一条可运行闭环：平台方开通园所 → 配置套餐 → 成交订单 → 登记收款 → 发放模块授权 → 到期降级 → 续费恢复。
- 授权判定必须能在**一次请求内**完成，且引入的延迟可接受。
- 存量园所在上线后不因授权收紧而突然不可用。

**Non-Goals:**

- 不做支付网关、不做对账、不做开票状态机。
- 不做分销商账号、多级分佣、提现。
- 不做计量配额（调用次数 / 存储 / 短信）。
- 不做园所自助注册与自助购买。

## Decisions

### D1 — 平台方是独立信任域（ADR-0014）

平台账号独立表、独立登录端点、独立 JWT 声明 `scope=platform`，不含 `kindergarten_id`。中间件 `RequirePlatform` 只认 `scope=platform`；既有的 `AuthJWT` 只认园所凭证。

替代方案「`users` 表加平台角色 + 空园所」被拒：会让每个下游中间件都出现 null 分支，越权边界只靠 if 维持。

### D2 — 授权按模块存行，不存园所级单一状态

运行时授权是一组 `(kindergarten_id, module_code, starts_at, expires_at, source, order_id)` 行。判定规则：存在任一在期行即可用。园所整体到期日仅用于展示，取所有在期行的最早 `expires_at`。

不建 `subscriptions` 聚合表：聚合状态完全可以由授权行推导，多一张表就多一个必须保持一致的状态。

### D3 — 模块目录是服务端常量，路由归属需要登记表

功能模块（`base` / `premium`）由 Go 常量维护，经平台接口暴露给套餐编辑器，不建表。

每条路由必须声明所属模块。沿用 `admin_perm.go` 的做法：建立「路由 → 模块」登记表，**启动时校验全覆盖，缺口使启动失败**。默认归属 `base`，仅成长记录 / 班级相册 / 成长报告 / 膳食营养标为 `premium`。

### D4 — 降级按 HTTP 方法区分读写，不做逐接口标注

`GET` / `HEAD` 视为读，`POST` / `PUT` / `PATCH` / `DELETE` 视为写。降级状态下：读放行，写拒绝。这样无需为 161 条路由逐一标注读写属性，且语义稳定。

例外（若有 GET 触发写入的接口）在登记表中显式标为写，作为白名单。

### D5 — 授权检查走缓存，写路径失效

```
request → RequireModule(code)
        → Redis ent:{kgID}
           ├ hit  → 判定
           └ miss → 查 entitlements → 回填（TTL 5 min）→ 判定
```

订单确认 / 取消、收款登记、授权人工变更后立即 `DEL ent:{kgID}`，保证续费即时恢复（spec: 续费即时恢复）。

### D6 — 订单快照存 JSON，授权行由快照派生

`orders` 保存成交时冻结的模块清单、幼儿数档位、起止日期与金额的 JSON 快照。订单生效时按快照逐模块生成授权行；订单取消时按 `order_id` 收回未到期授权。

加购与升级一律新开订单、全价购买整周期，不做 proration（ADR-0013）。

### D7 — 订单收款状态由金额推导

`payments` 表每行是一笔到账，退款为负数。订单状态不落库为人工可改字段，而是由 `sum(payments.amount) vs orders.amount_due` 推导。这样「多笔收款」「退款」都不需要额外的状态机代码。

### D8 — 佣金台账在收款确认时生成，可结算性是推导值

每笔正数收款按订单的分销商与佣金比例生成一条台账行；负数收款生成一条冲回行。可结算性 = `received_at + 30d <= now` 且该订单期间无退款，读取时推导，不依赖定时任务。

### D9 — 平台方跨租户查询是显式行为

平台仓储单独实现，不复用按 `kindergarten_id` 自动过滤的园所仓储。平台查询必须显式接收园所 ID 或明确全量查询；代码评审与测试均以此为准。

### D10 — 前端：同应用、独立路由树、独立登录态

复用现有 Nuxt 应用，新增 `/platform/*` 路由树与 `platform_token` cookie，`middleware/platform-auth.global.ts` 与现有 `auth.global.ts` 互不干扰。园所后台新增 `/billing` 页面展示本园套餐与到期状态。

### D11 — 授权强制开关化，先观察后收紧

新增配置 `billing.enforce_modules`（默认 `false`）。关闭时中间件只记录「本应拒绝」的请求并放行；开启后才真正拒绝。上线后观察一周，确认路由归属无遗漏再开启。这是对「存量园所被误锁」的唯一有效防线。

## 跨端触点清单

| 层 | 会改 / 不改 | 说明 |
|---|---|---|
| Go domain | 会改 | 平台账号、分销商、套餐、套餐模块、订单、收款、授权、佣金台账；`kindergartens` 增加状态字段 |
| Go repository | 会改 | 新增平台侧仓储；授权与订单仓储 |
| Go service | 会改 | 平台运营、套餐目录、订单收款、授权核算、佣金台账 |
| Go handler | 会改 | 新增平台端点与园所端授权读取端点 |
| Go middleware | 会改 | `RequirePlatform`、`RequireModule`（含只读降级与开关） |
| Casbin policy | 会改 | 新增平台角色与前缀规则；园所角色不变 |
| router.go | 会改 | 新增 `/api/v1/platform/*` 分组；`api-common` 新增授权读取 |
| backend/api/ | 会改 | 新增 `api-platform/`；`api-admin/` 与 `api-common.md` 各增一个端点，见下节 |
| backend/model/ | 不改 | 已停更，不引用 |
| admin 页面 | 会改 | 新增 `/platform/*` 六个页面；园所后台新增 `/billing` |
| teacher 小程序 | 会改 | 按授权隐藏 premium 入口；到期提示。无 tabBar，不涉及 `switchTab` |
| parent 小程序 | 会改 | 按授权隐藏 premium 入口；到期提示。跳转 tabBar 页仍用 `uni.switchTab`，本次不新增跨 tab 跳转 |
| design token | 不改 | 新页面复用既有 token；若到期提示需要新语义色，先改 `tokens.json` 再引用 |
| migrations | 会改 | 新增 goose 迁移；为存量园所回填 base 授权与试用 |
| Wire DI | 会改 | 新增 provider 与 `wire.Bind`，执行 `make wire` |

## 契约变更字段级清单

> 字段只保留页面或业务场景真正会用到的；无设计稿的新页面以业务场景作为溯源依据。

### 平台端点（新增 `backend/api/api-platform/`）

| 端点 | 字段 | 溯源（页面元素 / 业务场景） |
|---|---|---|
| 登录 | `username` / `password` → `token` | 平台登录页；与园所后台登录页同构但独立 |
| 园所列表 | `id` / `name` / `principal` / `status` / `student_count` / `active_plan_name` / `soonest_expire_at` | 园所列表页需要一眼看出「哪家快到期、哪家停用、哪家超规模」 |
| 开通园所 | 入参 `name` / `principal` / `phone` / `address?`；出参 `kindergarten_id` / `admin_phone` | 开通表单；出参用于回显「已创建园长账号」 |
| 园所状态 | `status` | 列表行内启停操作 |
| 套餐 | `code` / `name` / `tier` / `price_cents` / `term_months` / `module_codes` / `status` | 套餐编辑页：组合模块、定价、上下架 |
| 订单 | `kindergarten_id` / `plan_code` / `student_tier` / `amount_due_cents` / `amount_paid_cents` / `status` / `starts_at` / `expires_at` / `distributor_id` / `invoice_info` | 订单页与订单详情；`amount_paid_cents` 由收款推导，供详情展示 |
| 收款登记 | `amount_cents` / `method` / `received_at` / `voucher_url` / `remark` | 收款登记弹窗；`voucher_url` 用于对公转账凭证留档 |
| 分销商 | `name` / `contact` / `phone` / `commission_rate` / `status` | 分销商名单页 |
| 佣金台账 | `order_no` / `distributor_name` / `base_amount_cents` / `rate` / `amount_cents` / `status` / `eligible_at` | 台账页需要展示「怎么算出来的」，因此保留基数与比例 |

### 园所端新增端点

| 端点 | 字段 | 溯源（页面元素 / 业务场景） |
|---|---|---|
| `GET /admin/billing/subscription` | `plan_name` / `modules[]` / `expires_at` / `grace_until` / `state` | 「我的套餐」页：显示当前套餐、到期日、是否在宽限/降级 |
| `GET /common/entitlements` | `modules[]` / `expires_at` | 两端小程序启动时裁剪入口，避免进入后再被 403 |

两个端点均不返回内部订单号与金额，园所端不需要看到平台侧账务。

## 状态机

订单与收款：

```
              confirm               收款 < 应收              收款 = 应收
  [草稿] ────────────────> [生效·待收款] ──────────> [部分收款] ──────────> [已收清]
                               │                        │
                               │ 取消                    │ 取消
                               v                        v
                           [已取消]                  [已取消]
                               │
                               │ 到期日到达且未收清
                               v
                           [已到期]
```

授权生命周期：

```
  [在期] ──到期日到达──> [宽限期 15d] ──宽限结束──> [降级·停写保读]
    ^                                                    │
    └──────────────── 续费（新授权生效）─────────────────┘
```

请求路径：

```
  client → AuthJWT → CasbinRBAC → RequirePermission → RequireModule
                                                        ├ base      → 放行
                                                        ├ 在期      → 放行
                                                        ├ 宽限期    → 放行
                                                        ├ 降级 + 读 → 放行
                                                        └ 降级 + 写 → 403（开关关闭时仅记录）
```

## Migration Plan

1. **新增 goose 迁移**：创建平台账号、分销商、套餐、套餐模块、订单、收款、授权、佣金台账表；`kindergartens` 增加状态列。不修改任何既有业务表字段。
2. **回填**：为每个存量园所写入一条永久 `base` 授权（来源 `gift`）与一条 30 天试用授权（来源 `trial`）。这样上线后现有环境功能不变。
3. **先观察后收紧**：`billing.enforce_modules=false` 上线，收集「本应拒绝」的日志；确认路由归属无遗漏后改为 `true`。
4. **回滚**：把开关改回 `false` 即可恢复放行；所有新表都是附加表，回滚不需要删数据。

## Risks / Trade-offs

- **路由归属遗漏导致误锁** → 启动时登记表全覆盖校验 + 强制开关先关闭观察。
- **平台接口越权读取跨园数据** → 平台仓储独立实现、显式园所参数、专门的越权测试。
- **缓存导致续费不即时** → 订单与收款写入路径强制失效 `ent:{kgID}`。
- **幼儿数上限阻断招生** → 只提示、不阻断新增；超限仅作为升级线索，避免「孩子已经来了却录不进去」的教学事故。
- **低定价与分销经济学不成立** → 价格全配置化；本设计的验收标准是「闭环可运行」，不是收入。
- **平台方只有单一权限等级** → v1 接受；若平台团队超过 2 人再引入平台角色体系。

## Open Questions

无。四组状态枚举与幼儿数超限策略已于 2026-11-06 确认，取值见 `proposal.md` — 已确认的产品决策。

## 实施记录（2026-11-06）

实施过程中对设计做的四处增补，均已同步进 `backend/api/`：

1. **平台凭证 TTL 固定 24 小时**：平台方没有 refresh 流程，不能复用园所 15 分钟的 access TTL。
2. **开通园所请求增加 `admin_password`**：运营在表单里设置初始密码，避免把明文密码放在响应里。
3. **新增 `GET /platform/logs` + `platform_operation_logs` 表**：spec 要求「平台方可查询」敏感操作，复用园所 `operation_logs` 会让平台行为混入园所审计视图。
4. **新增 `PUT /platform/plans/:id/tiers` 与 `PUT /common/user/password`**：前者用于配置幼儿数档位，后者用于清除 `must_change_password`。

两处与原文的已知偏差：

- **路由 → 模块归属采用命名空间匹配**（`/growth`、`/growth/report`、`/album`、`/diet`），不逐条枚举 161 条路由；启动自检断言「每个 premium 模块至少命中一条已注册路由」。好处是同命名空间的新接口不会漏网，代价是删除单条路由不会触发启动失败。
- **`must_change_password` 是软要求**：登录响应返回标记，前端提示改密，服务端未强制阻断未改密账号的其它请求。
