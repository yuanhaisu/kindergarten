# 04. 订单与收款

> 对应前端页面：`admin/pages/platform/orders/index.vue`
>
> 基础路径：`/api/v1/platform`

---

## GET `/platform/orders`

> 订单列表

```json
// GET /platform/orders?page=1&page_size=20&kindergarten_id=7&status=partial
// 可选筛选：order_no（分号匹配）、start_date / end_date（按订单创建时间，含当日）、
// outstanding_only（只看未收清）、distributor_id

// Response
{
  "list": [
    {
      "id": 12,
      "order_no": "ORD202611061000001234567891234",
      "kindergarten_id": 7,
      "kindergarten_name": "阳光幼儿园",
      "plan_name": "旗舰版",
      "student_tier": 2,
      "amount_due_cents": 195000,
      "amount_paid_cents": 100000,
      "status": "partial",
      "starts_at": "2026-11-06",
      "expires_at": "2027-05-05",
      "pay_due_at": "2026-11-06",
      "credit_granted": true,
      "distributor_name": "华东渠道",
      "created_at": "2026-11-06T10:00:00+08:00"
    }
  ],
  "total": 1,
  "page": 1,
  "page_size": 20
}
```

> `amount_paid_cents` 由收款登记推导（单次聚合，排除已软删收款），不允许人工设置。
> `status` 读时推导：取消 → 已收清 → 已过付款到期日（`expired`）→ 部分收款 → 待收款。
> `pay_due_at` 是付款到期日，独立于服务期 `starts_at` / `expires_at`；缺省等于服务起始日。
> `credit_granted` 标记该订单是零收款确认的信用开通；信用开通理由**只在详情返回**，列表不返回。
>
> **筛选参数**
>
> | 参数 | 说明 |
> | --- | --- |
> | `status` | `unpaid` / `partial` / `paid` / `expired` / `cancelled`。`expired` 匹配「已过付款到期日且未收清」的订单（含存储状态为 `expired` 的记录）；`unpaid` / `partial` 只匹配尚未过付款到期日的订单 |
> | `order_no` | 订单号模糊匹配 |
> | `start_date` / `end_date` | 订单创建时间的日期范围，`YYYY-MM-DD`，`end_date` 含当日；非法格式或结束早于开始返回 422 |
> | `outstanding_only` | `true` 时仅返回未收清且未取消的订单 |
> | `kindergarten_id` / `distributor_id` | 精确匹配 |

---

## POST `/platform/orders`

> 创建订单（草稿）。确认后才会发放授权。

```json
// Request
{
  "kindergarten_id": 7,
  "plan_id": 2,
  "distributor_id": 3,
  "pay_due_at": "2026-11-20",
  "invoice_info": {
    "title": "阳光幼儿园",
    "tax_no": "91330100XXXXXXXX"
  }
}

// Response
{
  "id": 12,
  "order_no": "ORD202611061000001234567891234",
  "status": "unpaid",
  "amount_due_cents": 195000,
  "student_tier": 2,
  "starts_at": "2026-11-06",
  "expires_at": "2027-05-05",
  "pay_due_at": "2026-11-20",
  "plan_snapshot": { "module_codes": ["attendance", "notice", "pickup", "growth", "diet"] }
}
```

> 创建时读取该园当前在园幼儿数确定档位；档位在确认时冻结，之后人数变动不改档位。
> `plan_snapshot` 在创建时冻结，套餐目录的后续修改不影响该订单。
> `pay_due_at` 可选，`YYYY-MM-DD`；缺省等于服务起始日，服务端保证非空。
> `invoice_info` 可缺省，缺失不阻断确认。
> `pay_due_at` 格式不合法返回 422。
> `order_no` 由服务端生成、全局唯一，客户端不可传入：格式为
> `ORD` + `yyyyMMddHHmmss`（建单时刻）+ 9 位纳秒 + 4 位随机数，定长 30 字符（列宽 32）。
> 并发建单若撞上唯一索引，服务端自动换号重试，调用方无需处理。
>
> **档位无法匹配（422）**：套餐配置了档位，但按当前在园幼儿数落不到任何一档（通常是历史遗留的
> 不完整配置，例如区间有缺口或最高档有界）。此时不生成订单，返回：
>
> ```json
> {
>   "code": 42200,
>   "message": "按在园幼儿数无法落入该套餐的任何档位：档位配置不完整，请先修正套餐档位后重试",
>   "data": null
> }
> ```
>
> 未配置任何档位的套餐不受影响，按平价（`price_multiplier = 100`）成交。

---

## GET `/platform/orders/:id`

> 订单详情（含收款流水与授权预览）

```json
// Response
{
  "id": 12,
  "order_no": "ORD202611061000001234567891234",
  "kindergarten_id": 7,
  "kindergarten_name": "阳光幼儿园",
  "status": "partial",
  "amount_due_cents": 195000,
  "amount_paid_cents": 100000,
  "starts_at": "2026-11-06",
  "expires_at": "2027-05-05",
  "pay_due_at": "2026-11-06",
  "confirmed_at": "2026-11-06T10:00:00+08:00",
  "credit_granted": true,
  "credit_reason": "合同已签署，对公转账到账后补录",
  "plan_snapshot": { "module_codes": ["attendance", "notice", "pickup", "growth", "diet"] },
  "distributor": { "id": 3, "name": "华东渠道", "commission_rate": 10 },
  "invoice_info": { "title": "阳光幼儿园", "tax_no": "91330100XXXXXXXX" },
  "payments": [
    {
      "id": 21,
      "amount_cents": 100000,
      "method": "bank_transfer",
      "received_at": "2026-11-06T14:00:00+08:00",
      "voucher_url": "https://...",
      "remark": "首付",
      "recorded_by": "ops",
      "refund_of_payment_id": null
    },
    {
      "id": 25,
      "amount_cents": -40000,
      "method": "bank_transfer",
      "received_at": "2026-11-20T10:00:00+08:00",
      "voucher_url": "",
      "remark": "部分退费",
      "recorded_by": "ops",
      "refund_of_payment_id": 21
    }
  ]
}
```

> `credit_reason` 只在详情返回：非信用开通订单为 `false` / 空串。
> 列表只暴露 `credit_granted` 标记，不返回理由。
> `confirmed_at` 为 null 表示尚未确认；`status` 只由收付款金额与付款到期日派生，不包含确认状态。
> 管理端据 `confirmed_at` 与金额决定是否显示「确认订单」。

---

## POST `/platform/orders/:id/confirm`

> 确认订单：冻结快照、发放模块授权。净收款为零时确认即「信用开通」，必须填写理由；
> 部分收款一律不可确认。

```json
// Request（净收款为零时 reason 必填；已收清时可省略；部分收款不可确认）
{ "reason": "合同已签署，对公转账到账后补录" }

// Response
{
  "id": 12,
  "status": "unpaid",
  "entitlements": [
    { "module_code": "growth", "source": "subscription", "expires_at": "2027-05-05" }
  ]
}
```

> 确认前按事务内重算的净收款（`已收净额 = 正数收款 − 负数退款`）判定：
>
> - **净收款为零** → 信用开通，`reason` 必填；缺失或纯空白返回 422
>   `CREDIT_REASON_REQUIRED`。确认成功后订单被标记 `credit_granted=true` 并保存
>   `credit_reason`（理由只在详情返回），同时写入审计（操作账号、时间与理由）。
> - **`0 < 净收款 < 应收`** → 一律拒绝，返回 422 `ORDER_UNDERPAID` 与 `data.remaining_cents`。
>   理由不能替代收清：带理由也同样拒绝，且不写 `confirmed_at`、不发授权、不写确认审计。
> - **`净收款 >= 应收`**（含超收）→ 无需理由，正常确认。
>
> 信用开通不改金额口径：应收全额仍计入未收余额。
> `reason` 超过 255 个字符返回 422 `CREDIT_REASON_TOO_LONG`。
> 确认后不得再修改分销商归因。
> 取消订单会收回其产生且尚未到期的授权。

**零收款确认缺理由（422）**

```json
{
  "code": 42200,
  "message": "零收款确认必须填写信用开通理由",
  "data": null
}
```

**部分收款不可确认（422）**

```json
{
  "code": 42200,
  "message": "订单未收清，尚欠 999.99 元；部分收款不可确认，请先收清或退款后取消订单",
  "data": { "remaining_cents": 99999 }
}
```

- `data.remaining_cents` 为尚欠金额（分，= 应收 − 已收净额），管理端据此展示。

---

## POST `/platform/orders/:id/cancel`

> 取消订单。取消前先计算已收净额（正数收款之和 − 负数退款之和）。

```json
// Request
{ "reason": "园所暂缓采购" }

// Response
{ "cancelled_at": "2026-11-07T09:00:00+08:00" }
```

> 已产生授权的订单取消后，相关授权立即失效。
> 未收款的订单可直接取消，行为不变。

**未退款收款阻断取消（409）**

已收净额大于 0 时拒绝取消，返回待退金额；退款登记完毕、净额归零后才允许取消：

```json
{
  "code": 40900,
  "message": "订单仍有未退款收款 600.00 元，请先登记退款后再取消",
  "data": { "unrefunded_amount_cents": 60000 }
}
```

- `data.unrefunded_amount_cents` 为待退金额（分，等于已收净额），管理端据此展示。
- 拒绝取消时订单状态与授权均不变。
- 已取消订单不能再登记收款（`ORDER_CANCELLED`）。

---

## POST `/platform/orders/:id/payments`

> 登记一笔收款（退款传负数）

```json
// Request（正常收款）
{
  "amount_cents": 100000,
  "method": "bank_transfer",
  "received_at": "2026-11-06T14:00:00+08:00",
  "voucher_url": "https://...",
  "remark": "首付"
}

// Request（退款：负数金额 + 被退的原收款）
{
  "amount_cents": -40000,
  "refund_of_payment_id": 21,
  "remark": "部分退费"
}

// Response
{
  "payment_id": 21,
  "order_status": "partial",
  "amount_paid_cents": 100000
}
```

> `method` 取值：`bank_transfer` 对公转账 / `wechat` 微信 / `cash` 现金 / `other` 其他。
> 退款以负数金额登记，同时冲减订单已收金额与对应佣金。
> 佣金结算时钟使用**服务端登记时间**（登记时刻 + 30 天），`received_at` 只用于账务展示与对账。

**幂等键（防重复记账）**

请求头 `Idempotency-Key`（推荐，管理端收款弹窗在打开时生成一次，窗口内重试复用）：

```http
POST /platform/orders/12/payments
Idempotency-Key: 3f1c2a9e-8b7d-4c6f-9a10-2d5e8f7a6b4c
```

- 也可放在 JSON body 的 `idempotency_key` 字段；两者同时出现时以请求头为准。
- 服务端按 `(order_id, idempotency_key)` 唯一约束落库。相同键重复提交**返回首次收款结果**（同一 `payment_id`），不新增收款，也不重复计佣。
- 不传幂等键的旧请求仍可写入，但没有幂等保护（写 `NULL`，多个旧请求互不冲突）。

**金额上限**

- 正数收款：`当前已收净额 + 本次金额 ≤ amount_due_cents`，超出即拒绝，已收金额不变。
- 负数退款：`|本次金额| ≤ 当前已收净额`，同时不得超过被退收款的可退金额，超出即拒绝。

**时间可信**

- `received_at` 必须不晚于当前时间，未来时间返回 `RECEIVED_AT_IN_FUTURE`。
- 回填历史 `received_at` 允许落库，但佣金 `eligible_at` 仍按服务端登记时间计算，不会被提前「催熟」。
- 所有校验与收款、佣金写入在同一事务内；任一步失败不会留下收款或佣金行。

**退款链接规则（负数收款）**

- `refund_of_payment_id` 必填，指向**同一订单**的一笔正数收款；正数收款携带此字段会被拒绝。
- 可退上限为该笔收款金额减去已退金额，超出即拒绝。
- 佣金冲回精确作用于 `refund_of_payment_id` 所指向收款产生的佣金行，与该订单下「最近一条佣金」无关；每笔退款生成一条指向该佣金行的冲回记录。
- 被退收款的佣金累计冲回金额达到原始佣金时，净额归零并置为 `reversed`，即使此前已可结算也不再可结算。

**错误码**

| code | HTTP | 含义 |
| --- | --- | --- |
| `PAYMENT_EXCEEDS_AMOUNT_DUE` | 422 | 正数收款导致已收超过应收 |
| `REFUND_EXCEEDS_PAID_AMOUNT` | 422 | 退款金额超过订单当前已收净额 |
| `RECEIVED_AT_IN_FUTURE` | 422 | `received_at` 晚于当前时间 |
| `INVALID_IDEMPOTENCY_KEY` | 422 | 幂等键超过 64 个字符 |
| `REFUND_PAYMENT_REQUIRED` | 400 | 负数收款未指定 `refund_of_payment_id` |
| `REFUND_PAYMENT_NOT_ALLOWED` | 400 | 正数收款携带了 `refund_of_payment_id` |
| `REFUND_PAYMENT_INVALID` | 400 | 被退收款不是正数收款（如指向退款行） |
| `REFUND_PAYMENT_NOT_FOUND` | 404 | 被退收款不存在或不属于该订单 |
| `REFUND_EXCEEDS_PAYMENT` | 400 | 退款金额超过该笔收款的可退金额 |

金额类错误附带结构化金额，管理端可直接展示，不必解析文案：

```json
// 正数收款超额（code 42200）
{
  "code": 42200,
  "message": "收款金额超过订单应收，超出 100.00 元",
  "data": { "excess_cents": 10000, "remaining_cents": 40000 }
}

// 退款超过已收（code 42200）
{
  "code": 42200,
  "message": "退款金额超过订单已收，最多可退 500.00 元",
  "data": { "available_cents": 50000, "requested_cents": 50001 }
}
```

---

## GET `/platform/orders/:id/payments`

> 收款流水

```json
// Response
{
  "list": [
    {
      "id": 21,
      "amount_cents": 100000,
      "method": "bank_transfer",
      "received_at": "2026-11-06T14:00:00+08:00",
      "refund_of_payment_id": null
    }
  ]
}
```
