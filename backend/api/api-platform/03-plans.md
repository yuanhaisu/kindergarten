# 03. 套餐目录与模块

> 对应前端页面：`admin/pages/platform/plans/index.vue`
>
> 基础路径：`/api/v1/platform`

---

## GET `/platform/modules`

> 可售功能模块清单（服务端常量，不建表）

```json
// Response
{
  "list": [
    { "code": "attendance", "name": "考勤与请假", "category": "base", "sellable": false },
    { "code": "growth", "name": "成长记录与相册", "category": "premium", "sellable": true },
    { "code": "diet", "name": "膳食营养", "category": "premium", "sellable": true }
  ]
}
```

---

## GET `/platform/plans`

> 套餐列表

```json
// Response
{
  "list": [
    {
      "id": 1,
      "code": "basic",
      "name": "基础版",
      "tier": 1,
      "price_cents": 100000,
      "term_months": 6,
      "module_codes": ["attendance", "notice", "pickup", "growth"],
      "status": "active",
      "created_at": "2026-11-06T10:00:00+08:00"
    }
  ],
  "total": 1,
  "page": 1,
  "page_size": 20
}
```

> `price_cents` 为每个计费周期、每所园的应收金额（分）。价格只影响之后创建的订单。
> `status`：`active` 可售 / `archived` 已下架。

---

## POST `/platform/plans`

> 新建套餐

```json
// Request
{
  "code": "flagship",
  "name": "旗舰版",
  "tier": 2,
  "price_cents": 150000,
  "term_months": 6,
  "module_codes": ["attendance", "notice", "pickup", "growth", "diet"]
}

// Response
{ "id": 2 }
```

> `code` 全局唯一；引用未注册的模块返回 `422`。

---

## PATCH `/platform/plans/:id`

> 修改套餐。**不改变已成交订单的快照。**

```json
// Request（可部分更新）
{
  "name": "旗舰版（2027）",
  "price_cents": 168000,
  "module_codes": ["attendance", "notice", "pickup", "growth", "diet"]
}

// Response
{ "updated_at": "2026-11-06T11:00:00+08:00" }
```

---

## GET `/platform/plans/:id/tiers`

> 幼儿数档位配置

```json
// Response
{
  "list": [
    { "tier": 1, "min_students": 0, "max_students": 150, "price_multiplier": 100 },
    { "tier": 2, "min_students": 151, "max_students": 0, "price_multiplier": 130 }
  ]
}
```

> `price_multiplier` 为百分比（100 = 不加价）。档位边界由平台配置，变更只影响之后创建的订单。
> `max_students = 0` 表示无上限；最高档必须无上限。

---

## PUT `/platform/plans/:id/tiers`

> 整体替换幼儿数档位。**不改变已成交订单的快照。**

```json
// Request
{
  "list": [
    { "tier": 1, "min_students": 0, "max_students": 150, "price_multiplier": 100 },
    { "tier": 2, "min_students": 151, "max_students": 0, "price_multiplier": 130 }
  ]
}

// Response
{ "updated_at": "2026-11-14T10:00:00+08:00" }
```

> 保存前校验覆盖完整性，任一条不满足即整体拒绝（422），原有档位保持不变：
>
> | 规则 | 错误文案（`message`） |
> | --- | --- |
> | 至少一档，且第一档下限为 0 | 至少需要配置一个档位，且第一档必须从 0 人开始 / 档位必须从 0 人开始：第一档的人数下限应为 0 |
> | 区间连续：后一档下限 = 前一档上限 + 1 | 档位区间不连续：后一档的人数下限必须等于前一档上限 + 1 |
> | 区间不重叠 | 档位区间重叠：后一档的人数下限必须等于前一档上限 + 1 |
> | 只有最高档可以 `max_students = 0` | 只有最高档可以无上限：无上限的档位必须排在最后 |
> | 最高档必须 `max_students = 0` | 最高档必须无上限：请把最高档的人数上限填为 0 |
>
> 保存合法配置后，建单按园所在园幼儿数落入对应档位；档位变更只影响之后创建的订单。
> 档位为空（从未配置）的套餐按 `price_multiplier = 100` 平价成交。
