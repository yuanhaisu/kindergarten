# 02. 园所与开通

> 对应前端页面：`admin/pages/platform/kindergartens/index.vue`
>
> 基础路径：`/api/v1/platform`

---

## GET `/platform/kindergartens`

> 园所列表（跨园，不受单一园所范围过滤）

```json
// GET /platform/kindergartens?page=1&page_size=20&keyword=阳光&status=active

// Response
{
  "list": [
    {
      "id": 4,
      "name": "阳光幼儿园",
      "principal": "王园长",
      "status": "active",
      "student_count": 180,
      "active_plan_name": "旗舰版",
      "soonest_expire_at": "2027-06-30",
      "created_at": "2026-11-06T10:00:00+08:00"
    }
  ],
  "total": 1,
  "page": 1,
  "page_size": 20
}
```

> `student_count` 为当前在园幼儿数，用于判断档位是否超限。
> `soonest_expire_at` 取该园所有在期授权中最早的到期日；无授权时为 `null`。

---

## POST `/platform/kindergartens`

> 开通园所，同时创建该园第一个园长管理员账号

```json
// Request
{
  "name": "阳光幼儿园",
  "principal": "王园长",
  "phone": "13800001234",
  "address": "杭州市西湖区"
}

// Response
{
  "kindergarten_id": 7,
  "admin_phone": "13800001234",
  "must_change_password": true,
  "trial_expires_at": "2026-12-06"
}
```

> 开通即发放 30 天全功能试用（来源 `trial`）。
> 手机号在该园所已有管理员时返回 `409`，园所不被创建。
> 初始密码由系统生成，通过响应外的运营渠道告知园长；园长首次登录必须改密。

---

## GET `/platform/kindergartens/:id`

> 园所详情：基本信息 + 当前授权 + 最近订单

```json
// Response
{
  "id": 7,
  "name": "阳光幼儿园",
  "principal": "王园长",
  "status": "active",
  "student_count": 180,
  "entitlements": [
    {
      "module_code": "growth",
      "module_name": "成长记录",
      "source": "trial",
      "starts_at": "2026-11-06",
      "expires_at": "2026-12-06",
      "state": "active"
    }
  ],
  "recent_orders": [
    { "id": 12, "order_no": "ORD20261106001", "amount_due_cents": 150000, "status": "unpaid" }
  ]
}
```

> `state` 取值：`active` 在期 / `grace` 宽限期 / `expired` 已降级。

---

## PATCH `/platform/kindergartens/:id/status`

> 启用或停用园所

```json
// Request
{ "status": "suspended" }

// Response
{ "updated_at": "2026-11-06T10:30:00+08:00" }
```

> 停用后该园全部园所端会话立即不可用；重新启用后数据与授权保持不变。
