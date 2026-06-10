# 15. 内容审核

> 对应前端页面：`admin/pages/audit/index.vue`
>
> 基础路径：`/api/v1/admin`，需 `Authorization: Bearer <token>`，角色 `admin`。
>
> 状态：✅ 3 个接口已实现（`AuditRepo` / `AuditService` / `handler/http/audit.go`）。

---

## GET `/admin/audit/list`

> 待审核内容列表（对应前端 `getAuditList`）

```json
// Query: ?type=notice&status=pending&page=1&page_size=20
// type: notice / growth / comment
// status: pending / passed / rejected

// Response
{
  "list": [
    {
      "id": 1,
      "type": "notice",
      "type_label": "通知",
      "reference_id": 201,
      "title": "周末作业通知",
      "content_preview": "各位家长您好...",
      "submitter": { "id": 1, "nickname": "李老师", "class_name": "小一班" },
      "status": "pending",
      "status_label": "待审核",
      "submitted_at": "2026-03-19T08:50:00Z"
    }
  ],
  "total": 5,
  "page": 1,
  "page_size": 20
}
```

---

## PUT `/admin/audit/:id`

> 审核操作（对应前端 `auditContent`）

```json
// Request
{
  "action": "pass",             // pass / reject
  "remark": ""
}

// Response
{
  "id": 1,
  "status": "passed",
  "audited_at": "2026-03-19T09:00:00Z"
}
```

---

## GET `/admin/audit/stats`

> 审核统计（对应前端 `getAuditStats`）

```json
// Response
{
  "pending": 5,
  "passed_today": 12,
  "rejected_today": 1,
  "total_this_month": 350,
  "by_type": {
    "notice": { "pending": 2, "passed": 200, "rejected": 5 },
    "growth": { "pending": 2, "passed": 130, "rejected": 3 },
    "comment": { "pending": 1, "passed": 15, "rejected": 0 }
  }
}
```
