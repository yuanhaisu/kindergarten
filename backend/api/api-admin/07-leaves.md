# 07. 请假管理

> 对应前端页面：`admin/pages/leaves/index.vue`
>
> 基础路径：`/api/v1/admin`，需 `Authorization: Bearer <token>`，角色 `admin`。
>
> 状态：✅ 已实现（含审批接口）。

> ⚠️ 页面显示的 "审批通过 / 驳回" 按钮尚未在前端接入接口（前端 `api/leaves.ts` 未封装审批接口），若后端实现请留意补齐 `PUT /admin/leaves/:id/audit`。

---

## GET `/admin/leaves`

> 全园请假记录（对应前端 `getLeaves`）

```json
// Query: ?class_id=&status=&start_date=&end_date=&page=1&page_size=20

// Response
{
  "list": [
    {
      "id": 101,
      "student": { "id": 1, "name": "王小明", "class_name": "小一班" },
      "applicant": { "nickname": "王小明妈妈", "relation": "mother" },
      "reason": "感冒发烧",
      "start_date": "2026-03-19",
      "end_date": "2026-03-21",
      "days": 3,
      "status": 1,
      "status_label": "已同意",
      "reviewer": { "nickname": "李老师" },
      "reviewed_at": "2026-03-19T08:00:00Z",
      "created_at": "2026-03-19T07:30:00Z"
    }
  ],
  "total": 150,
  "page": 1,
  "page_size": 20
}
```

---

## GET `/admin/leaves/stats`

> 请假统计（对应前端 `getLeaveStats`，`/statistics` 页面也使用）

```json
// Query: ?start_date=2026-03-01&end_date=2026-03-31

// Response
{
  "period": { "start": "2026-03-01", "end": "2026-03-31" },
  "total": 45,
  "pending": 3,
  "approved": 40,
  "rejected": 2,
  "by_class": [
    { "class_name": "小一班", "count": 8 },
    { "class_name": "小二班", "count": 5 }
  ],
  "by_reason": [
    { "reason": "生病", "count": 30 },
    { "reason": "家事", "count": 10 },
    { "reason": "其他", "count": 5 }
  ]
}
```

---

## PUT `/admin/leaves/:id/audit`

> 审批请假（对应前端 `auditLeave`）。仅 `status = 0`（待审批）可操作。

```json
// Request
{ "status": 1, "review_remark": "" }

// status: 1 同意，2 驳回；review_remark 可选

// Response data
{}
```
