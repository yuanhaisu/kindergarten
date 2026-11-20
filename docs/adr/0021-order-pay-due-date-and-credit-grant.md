# 0021 — 订单付款到期日独立于服务期，零收款确认即信用开通

- Status: Accepted
- Date: 2026-11-19

## Context

平台订单的收款状态此前用 `expires_at`（服务期结束日）判定逾期：一张 6 个月的订单在整个服务期内都显示 `unpaid`，看板的「未收余额」没有逾期含义，运营无法区分「还没到付款期」和「已经拖欠」。订单也没有独立的付款期限字段。

另一侧，零收款确认订单会直接发出整期授权，且不留任何标记。确认操作本身有审计，但审计不能代替订单上可查询的业务状态：后续收款到账后，无法再判断这笔订单当初是不是信用开通。

## Decision

### 1. `orders.pay_due_at` 独立于服务期

订单增加 `pay_due_at`，创建时确定，默认取 `starts_at`，创建请求可显式指定。调整服务起止日期不改变 `pay_due_at`。服务端通过三层保证非空：

- `Order.BeforeCreate` 钩子在 `PayDueAt` 为零值时取 `StartsAt`；
- `deriveOrderStatus` 对零值再兜底回退到 `StartsAt`；
- goose 迁移把存量 `pay_due_at` 回填为 `starts_at` 后置 `NOT NULL`（AutoMigrate 只加列、不回填，开发库也必须先跑迁移）。

收款状态改为读时派生，不写回数据库：

```
cancelledAt != nil              -> cancelled
amount_due > 0 && paid >= due   -> paid
today > date(pay_due_at) && paid < due -> expired
paid > 0                        -> partial
                                -> unpaid
```

到期日按日期比较：当天仍可付款，次日零时起才算逾期。`expires_at` 只表示服务期，不再参与收款状态。列表筛选在 SQL 中翻译状态（`expired` 匹配「存储状态为 unpaid/partial/expired 且已过付款到期日」，`unpaid`/`partial` 只匹配未过付款到期日的记录）。

### 2. 零收款确认 = 信用开通，必须留理由

确认请求增加可选 `reason`；净收款为零时必填，否则 422。确认成功后写 `credit_granted=true` 与 `credit_reason`：

- 列表返回 `credit_granted` 作为催收标识；
- 详情返回 `credit_reason`，列表不返回；
- 信用开通订单仍计入未收余额。

## Consequences

- 需要新增 goose 迁移（`00009_orders_pay_due_at.sql`、`00011_orders_credit_grant.sql`），并在发布顺序里先迁移后启动。
- 订单列表改为单次 `LEFT JOIN` 聚合已收金额，去掉逐行 `SumByOrder`；状态筛选必须在 SQL 中翻译，不能只按存储 `status` 过滤。
- 存储的 `status` 可能与派生状态不一致；所有展示与筛选以派生状态为准，handler 不得直接返回存储状态。
- 零收款确认的语义从「无标记放行」变为「必须说明理由的信用授予」，审计与订单字段同时留痕。
- 信用开通是业务状态，不是审计的替代；二者都必须存在。

## Alternatives considered

- **继续用 `expires_at` 当付款期限**：服务期长达 6 个月，未付款订单永远显示 `unpaid`，逾期不可识别，被否。
- **新增定时任务批量刷新订单状态**：系统当前是单实例，定时任务不在本期范围；读时派生 + SQL 筛选已足够，且避免在读取路径写业务表。
- **完全禁止零收款确认**：对公转账常发生在合同签署后、开课前，禁止会让园所无法开始使用。
- **零收款确认只写审计、不落订单字段**：后续收款到账后无法再判断该订单当初是否信用开通，被否。
