# 08. 接送管理

> 对应前端页面：`admin/pages/pickup/index.vue`
>
> 基础路径：`/api/v1/admin`，需 `Authorization: Bearer <token>`，角色 `admin`。
>
> 状态：✅ 已实现。

---

## GET `/admin/pickup/overview`

> 全园接送概览（对应前端 `getPickupOverview`）

```json
// Query: ?date=2026-03-19

// Response
{
  "date": "2026-03-19",
  "total_students": 280,
  "arrived": 280,
  "picked_up": 120,
  "not_picked_up": 160,
  "abnormal_count": 2,
  "by_class": [
    { "class_id": 1, "class_name": "小一班", "arrived": 28, "picked_up": 12, "not_picked_up": 16 },
    { "class_id": 2, "class_name": "小二班", "arrived": 29, "picked_up": 15, "not_picked_up": 14 }
  ]
}
```

---

## GET `/admin/pickup/records`

> 全园接送记录（对应前端 `getPickupRecords`）

```json
// Query: ?class_id=&date=2026-03-19&is_abnormal=&page=1&page_size=20

// Response
{
  "list": [
    {
      "id": 301,
      "student": { "id": 1, "name": "王小明", "class_name": "小一班" },
      "type": 2,
      "type_label": "接走",
      "method": 1,
      "method_label": "扫码",
      "pickup_name": "爸爸（王明）",
      "operator": { "nickname": "李师傅" },
      "is_abnormal": false,
      "recorded_at": "2026-03-19T17:30:25Z"
    }
  ],
  "total": 560,
  "page": 1,
  "page_size": 20
}
```

---

## GET `/admin/pickup/abnormal`

> 异常接送记录（对应前端 `getPickupAbnormal`）

```json
// Query: ?date=&page=1&page_size=20

// Response
{
  "list": [
    {
      "id": 305,
      "student": { "id": 5, "name": "张小强", "class_name": "中一班" },
      "type": 2,
      "type_label": "接走",
      "pickup_name": "陌生人",
      "is_authorized": false,
      "abnormal_reason": "非授权人员接送，已拒绝并通知家长",
      "operator": { "nickname": "李师傅" },
      "recorded_at": "2026-03-19T17:45:00Z"
    }
  ],
  "total": 2
}
```

---

## PUT `/admin/pickup/authorized-persons/:id/audit`

> 审核授权接送人（对应前端 `auditPickupAuthorizedPerson`）

```json
// Request
{
  "action": "approve",           // approve / reject
  "remark": ""
}

// Response
{
  "id": 4,
  "status": 1,
  "status_label": "已审核",
  "audited_at": "2026-03-19T10:00:00Z"
}
```
