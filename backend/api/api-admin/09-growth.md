# 09. 成长记录管理

> 对应前端页面：`admin/pages/growth/index.vue`
>
> 基础路径：`/api/v1/admin`，需 `Authorization: Bearer <token>`，角色 `admin`。
>
> 状态：✅ 5 个接口已实现（审核走 `content_audits.target_type=growth`；无审核记录时 `audit_status` 为 `pending`）。

---

## GET `/admin/growth/logs`

> 全园成长日志列表（对应前端 `getGrowthLogs`）

```json
// Query: ?class_id=&teacher_id=&date=&page=1&page_size=20

// Response
{
  "list": [
    {
      "id": 1,
      "date": "2026-03-19",
      "class_name": "小一班",
      "content_preview": "今天的美术课主题是《我心中的春天》...",
      "media_count": 6,
      "likes_count": 28,
      "comments_count": 12,
      "creator": { "id": 1, "nickname": "李老师" },
      "audit_status": "passed",
      "created_at": "2026-03-19T11:30:00Z"
    }
  ],
  "total": 3200,
  "page": 1,
  "page_size": 20
}
```

---

## GET `/admin/growth/logs/:id`

> 成长日志详情（同老师端，额外含审核信息；对应前端 `getGrowthLog`）

```json
// Response
{
  "id": 1,
  "date": "2026-03-19",
  "class_id": 1,
  "class_name": "小一班",
  "content": "今天的美术课主题是《我心中的春天》...",
  "media": [
    { "id": 1, "file_url": "...", "thumb_url": "...", "file_type": "image" }
  ],
  "media_count": 6,
  "likes_count": 28,
  "comments_count": 12,
  "creator": { "id": 1, "nickname": "李老师", "avatar": "..." },
  "audit_status": "passed",
  "auditor": { "id": 100, "nickname": "张园长" },
  "audited_at": "2026-03-19T11:35:00Z",
  "created_at": "2026-03-19T11:30:00Z"
}
```

---

## DELETE `/admin/growth/logs/:id`

> 删除成长日志（管理员强制删除；对应前端 `deleteGrowthLog`）

```json
// Response
{}
```

---

## PUT `/admin/growth/logs/:id/audit`

> 审核成长日志（对应前端 `auditGrowthLog`）

```json
// Request
{
  "action": "pass",             // pass / reject
  "remark": "内容合规，审核通过"
}

// Response
{
  "id": 1,
  "audit_status": "passed",
  "audited_at": "2026-03-19T11:35:00Z"
}
```

---

## GET `/admin/growth/stats`

> 成长记录统计（管理端；对应前端 `getGrowthStats`，`/statistics` 页面也使用）

```json
// Query: ?start_date=2026-03-01&end_date=2026-03-31

// Response
{
  "period": { "start": "2026-03-01", "end": "2026-03-31" },
  "total_logs": 320,
  "total_media": 2800,
  "total_likes": 4500,
  "total_comments": 1200,
  "by_class": [
    { "class_name": "小一班", "teacher": "李老师", "logs": 68, "media": 580 },
    { "class_name": "小二班", "teacher": "王老师", "logs": 45, "media": 380 }
  ],
  "daily_trend": [
    { "date": "2026-03-01", "logs": 8, "media": 65 },
    { "date": "2026-03-02", "logs": 12, "media": 98 }
  ]
}
```
