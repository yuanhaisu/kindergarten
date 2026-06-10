# 13. 通知管理

> 对应前端页面：`admin/pages/notices/index.vue`
>
> 基础路径：`/api/v1/admin`，需 `Authorization: Bearer <token>`，角色 `admin`。
>
> 状态：✅ 6 个接口已实现（含 POST 发布、PUT 编辑）；前端已封装并可在通知页使用。

---

## GET `/admin/notices`

> 全园通知列表（管理后台可看所有班级；对应前端 `getNotices`）

```json
// Query: ?class_id=&category=&status=&keyword=&page=1&page_size=20

// Response
{
  "list": [
    {
      "id": 1,
      "category": "homework",
      "category_label": "作业",
      "title": "周末作业通知",
      "class_name": "小一班",
      "class_id": 1,
      "creator": { "id": 1, "nickname": "李老师" },
      "status": 2,
      "status_label": "已发布",
      "read_rate": "83%",
      "published_at": "2026-03-19T09:00:00Z",
      "created_at": "2026-03-19T08:50:00Z"
    }
  ],
  "total": 580,
  "page": 1,
  "page_size": 20
}
```

---

## GET `/admin/notices/:id`

> 通知详情（同老师端 `GET /teacher/notices/:id`，额外包含管理字段；对应前端 `getNotice`）

```json
// Response
{
  "id": 1,
  "category": "homework",
  "category_label": "作业",
  "title": "周末作业通知",
  "content": "各位家长您好，本周作业如下...",
  "class_id": 1,
  "class_name": "小一班",
  "need_confirm": true,
  "status": 2,
  "published_at": "2026-03-19T09:00:00Z",
  "attachments": [],
  "read_stats": {
    "read_count": 25,
    "unread_count": 5,
    "confirmed_count": 20,
    "total_count": 30,
    "read_rate": "83%"
  },
  "creator": { "id": 1, "nickname": "李老师", "avatar": "..." },
  "audit_status": "passed",
  "auditor": { "id": 100, "nickname": "张园长" },
  "audited_at": "2026-03-19T08:55:00Z",
  "created_at": "2026-03-19T08:50:00Z"
}
```

---

## DELETE `/admin/notices/:id`

> 删除通知（管理员强制删除 / 撤回通知；对应前端 `deleteNotice`）

```json
// Response
{}
```

---

## PUT `/admin/notices/:id/audit`

> 审核通知（需要审核时；对应前端 `auditNotice`）

```json
// Request
{
  "action": "pass",             // pass=通过 / reject=驳回
  "remark": ""                   // 驳回时填写原因
}

// Response
{
  "id": 1,
  "audit_status": "passed",
  "audited_at": "2026-03-19T08:55:00Z"
}
```

---

## POST `/admin/notices`

> 发布通知（管理端；对应前端 `createNotice`）。HTTP **201**。

```json
// Request
{
  "class_id": 1,
  "category": "homework",
  "title": "周末作业通知",
  "content": "正文…",
  "need_confirm": true,
  "scheduled_at": null,
  "attachments": []
}

// Response
{
  "id": 1,
  "status": 2,
  "created_at": "2026-03-19T08:50:00Z"
}
```

---

## PUT `/admin/notices/:id`

> 编辑通知（对应前端 `updateNotice`）。字段均可选（至少传一项或 `attachments`）。

```json
// Request
{
  "title": "修订后的标题",
  "content": "修订后的正文"
}

// Response
{
  "id": 1
}
```
