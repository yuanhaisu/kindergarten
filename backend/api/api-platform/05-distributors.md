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
      "status": "settleable",
      "received_at": "2026-11-06T14:00:00+08:00",
      "eligible_at": "2026-12-06"
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
> 负数收款会生成一条 `reversed` 行，冲回对应佣金。

---

## GET `/platform/commissions/:id`

> 单条佣金明细（含推导过程）

```json
// Response
{
  "id": 41,
  "order_no": "ORD20261106001",
  "distributor": { "id": 3, "name": "华东渠道" },
  "payment": { "id": 21, "amount_cents": 100000, "received_at": "2026-11-06T14:00:00+08:00" },
  "rate": 10,
  "amount_cents": 10000,
  "status": "settleable",
  "eligible_at": "2026-12-06",
  "reversal_of": null
}
```

> `reversal_of` 指向被冲回的佣金行；普通佣金为 `null`。
