# 04. 订单与收款

> 对应前端页面：`admin/pages/platform/orders/index.vue`
>
> 基础路径：`/api/v1/platform`

---

## GET `/platform/orders`

> 订单列表

```json
// GET /platform/orders?page=1&page_size=20&kindergarten_id=7&status=partial

// Response
{
  "list": [
    {
      "id": 12,
      "order_no": "ORD20261106001",
      "kindergarten_id": 7,
      "kindergarten_name": "阳光幼儿园",
      "plan_name": "旗舰版",
      "student_tier": 2,
      "amount_due_cents": 195000,
      "amount_paid_cents": 100000,
      "status": "partial",
      "starts_at": "2026-11-06",
      "expires_at": "2027-05-05",
      "distributor_name": "华东渠道",
      "created_at": "2026-11-06T10:00:00+08:00"
    }
  ],
  "total": 1,
  "page": 1,
  "page_size": 20
}
```

> `amount_paid_cents` 由收款登记推导，不允许人工设置。
> `status` 由应收与已收推导；到期且未收清时变为 `expired`。

---

## POST `/platform/orders`

> 创建订单（草稿）。确认后才会发放授权。

```json
// Request
{
  "kindergarten_id": 7,
  "plan_id": 2,
  "distributor_id": 3,
  "invoice_info": {
    "title": "阳光幼儿园",
    "tax_no": "91330100XXXXXXXX"
  }
}

// Response
{
  "id": 12,
  "order_no": "ORD20261106001",
  "status": "unpaid",
  "amount_due_cents": 195000,
  "student_tier": 2,
  "starts_at": "2026-11-06",
  "expires_at": "2027-05-05",
  "plan_snapshot": { "module_codes": ["attendance", "notice", "pickup", "growth", "diet"] }
}
```

> 创建时读取该园当前在园幼儿数确定档位；档位在确认时冻结，之后人数变动不改档位。
> `plan_snapshot` 在创建时冻结，套餐目录的后续修改不影响该订单。
> `invoice_info` 可缺省，缺失不阻断确认。

---

## GET `/platform/orders/:id`

> 订单详情（含收款流水与授权预览）

```json
// Response
{
  "id": 12,
  "order_no": "ORD20261106001",
  "kindergarten_id": 7,
  "kindergarten_name": "阳光幼儿园",
  "status": "partial",
  "amount_due_cents": 195000,
  "amount_paid_cents": 100000,
  "starts_at": "2026-11-06",
  "expires_at": "2027-05-05",
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
      "recorded_by": "ops"
    }
  ]
}
```

---

## POST `/platform/orders/:id/confirm`

> 确认订单：冻结快照、发放模块授权

```json
// Response
{
  "id": 12,
  "status": "unpaid",
  "entitlements": [
    { "module_code": "growth", "source": "subscription", "expires_at": "2027-05-05" }
  ]
}
```

> 确认后不得再修改分销商归因。
> 取消订单会收回其产生且尚未到期的授权。

---

## POST `/platform/orders/:id/cancel`

> 取消订单

```json
// Request
{ "reason": "园所暂缓采购" }

// Response
{ "cancelled_at": "2026-11-07T09:00:00+08:00" }
```

> 已产生授权的订单取消后，相关授权立即失效。

---

## POST `/platform/orders/:id/payments`

> 登记一笔收款（退款传负数）

```json
// Request
{
  "amount_cents": 100000,
  "method": "bank_transfer",
  "received_at": "2026-11-06T14:00:00+08:00",
  "voucher_url": "https://...",
  "remark": "首付"
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
> 到账后 30 天且无退款的部分才可进入结算。

---

## GET `/platform/orders/:id/payments`

> 收款流水

```json
// Response
{
  "list": [
    { "id": 21, "amount_cents": 100000, "method": "bank_transfer", "received_at": "2026-11-06T14:00:00+08:00" }
  ]
}
```
