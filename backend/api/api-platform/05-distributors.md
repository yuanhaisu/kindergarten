# 05. 分销商与佣金台账

> 对应前端页面：`admin/pages/platform/distributors/index.vue`、`admin/pages/platform/commissions/index.vue`
>
> 基础路径：`/api/v1/platform`
>
> **分销商不登录系统**，只是平台方维护的一份名单；佣金按实收回款计。

---

## GET `/platform/distributors`

> 分销商列表

```json
// Response
{
  "list": [
    {
      "id": 3,
      "name": "华东渠道",
      "contact": "李经理",
      "phone": "13900001234",
      "commission_rate": 10,
      "status": "active",
      "created_at": "2026-11-01T10:00:00+08:00"
    }
  ],
  "total": 1,
  "page": 1,
  "page_size": 20
}
```

> `commission_rate` 为百分比整数（10 = 10%）。

---

## POST `/platform/distributors`

> 新建分销商

```json
// Request
{
  "name": "华东渠道",
  "contact": "李经理",
  "phone": "13900001234",
  "commission_rate": 10
}

// Response
{ "id": 3 }
```

---

## PATCH `/platform/distributors/:id`

> 修改分销商。**不回算历史订单的佣金。**

```json
// Request
{ "commission_rate": 12, "status": "inactive" }

// Response
{ "updated_at": "2026-11-07T10:00:00+08:00" }
```

> 佣金比例在订单确认时冻结，之后修改只影响新订单。

---

## GET `/platform/commissions`

> 佣金台账（只读）

```json
// GET /platform/commissions?month=2026-11&distributor_id=3&status=settleable

// Response
{
  "list": [
    {
      "id": 41,
      "order_no": "ORD20261106001",
      "kindergarten_name": "阳光幼儿园",
      "distributor_name": "华东渠道",
      "payment_id": 21,
      "base_amount_cents": 100000,
      "rate": 10,
      "amount_cents": 10000,
      "reversed_amount_cents": 0,
      "net_amount_cents": 10000,
      "status": "settleable",
      "received_at": "2026-11-06T14:00:00+08:00",
      "eligible_at": "2026-12-06",
      "reversal_of": null
    }
  ],
  "total": 1,
  "page": 1,
  "page_size": 20,
  "summary": {
    "pending_cents": 0,
    "settleable_cents": 10000,
    "settled_cents": 0,
    "reversed_cents": 0
  }
}
```

> 台账金额由「收款金额 × 订单确认时冻结的比例」推导，**不提供手工改写金额的接口**。
> `net_amount_cents = amount_cents - reversed_amount_cents`，结算条件为「到账满 30 天且净额大于 0」。
> 列表只返回原始佣金行（`reversal_of` 为 `null`）：退款产生的冲回行金额已累计进原始行的 `reversed_amount_cents`，再列一次会重复扣减；冲回行仅在单条明细的 `reversals` 中查看。
> `status` 取值 `pending` / `settleable` / `settled` / `reversed`；净额归零即 `reversed`，即使此前已可结算也不会再进入结算。

**汇总口径（summary）**

| 字段 | 口径 |
| --- | --- |
| `pending_cents` | 待结算原始行的净额合计 |
| `settleable_cents` | 可结算原始行的净额合计 |
| `settled_cents` | 已结算原始行的净额合计 |
| `reversed_cents` | 期间累计冲回金额（正数），即原始行 `reversed_amount_cents` 合计 |

---

## GET `/platform/commissions/:id`

> 单条佣金明细（含推导过程）

```json
// Response
{
  "id": 41,
  "order_no": "ORD20261106001",
  "kindergarten_name": "阳光幼儿园",
  "distributor_name": "华东渠道",
  "payment_id": 21,
  "base_amount_cents": 100000,
  "rate": 10,
  "amount_cents": 10000,
  "reversed_amount_cents": 4000,
  "net_amount_cents": 6000,
  "status": "settleable",
  "received_at": "2026-11-06T14:00:00+08:00",
  "eligible_at": "2026-12-06",
  "reversal_of": null,
  "reversals": [
    {
      "id": 52,
      "payment_id": 25,
      "base_amount_cents": 40000,
      "amount_cents": -4000,
      "status": "reversed",
      "created_at": "2026-11-20T10:00:00+08:00"
    }
  ]
}
```

> `reversal_of` 指向被冲回的原始佣金行；列表返回的原始佣金行恒为 `null`。
> `reversals` 为该原始行对应的冲回记录，仅在单条明细中返回：每笔退款一条，`payment_id` 为退款收款，`amount_cents` 为负数。
> 冲回金额累计达到原始佣金时，该行 `status` 置为 `reversed`、`net_amount_cents` 为 0。
