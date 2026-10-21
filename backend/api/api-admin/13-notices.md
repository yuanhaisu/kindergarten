# 13. 通知管理

> 对应前端页面：`admin/pages/notices/index.vue`
>
> 基础路径：`/api/v1/admin`，需 `Authorization: Bearer <token>`，角色 `admin`。
>
> 状态：✅ 6 个接口已实现（含 POST 发布、PUT 编辑）；前端已封装并可在通知页使用。

> ⚠️ **契约变更（notice-multitarget，2026-10-09）**：
> - 发布/编辑入参 `class_id` 替换为 `class_ids`（多班级目标，必填非空）；新增可选 `grades`（年级圈选，取值 小班/中班/大班，发布时展开为班级快照，与 `class_ids` 并集去重）
> - 列表项/详情 `class_id`/`class_name` 替换为 `class_names`（目标班级名列表，对应「发布范围」列）
> - `read_stats` / `read_rate` 的 `total_count` 分母：**形状不变，语义变**——统一为「目标范围去重家长数」，与老师端数值一致

---

## GET `/admin/notices`

> 全园通知列表（管理后台可看所有班级；对应前端 `getNotices`）

```json
// Query: ?class_id=&category=&status=&keyword=&page=1&page_size=20
// class_id: 按目标班级过滤（命中目标集即返回）

// Response
{
  "list": [
    {
      "id": 1,
      "category": "homework",
      "category_label": "作业",
      "title": "周末作业通知",
      "class_names": ["小一班", "小二班"],
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
  "class_names": ["小一班", "小二班"],
  "need_confirm": true,
  "status": 2,
  "published_at": "2026-03-19T09:00:00Z",
  "attachments": [],
  "read_stats": {
    "read_count": 25,
    "unread_count": 5,
    "confirmed_count": 20,
    "total_count": 30,            // 分母=目标范围去重家长数（多孩只计一人）；形状不变，口径已变
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
  "class_ids": [1, 2],            // 目标班级，必填非空；与 grades 并集去重
  "grades": ["大班"],             // 可选，年级圈选（小班/中班/大班）；发布时展开为该年级班级快照
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

> 编辑通知（对应前端 `updateNotice`）。字段均可选（至少传一项或 `attachments`）。传 `class_ids`/`grades` 时为**显式重选目标集**（整体替换，不做增量）。

```json
// Request
{
  "title": "修订后的标题",
  "content": "修订后的正文",
  "class_ids": [1, 2],
  "grades": ["大班"]
}

// Response
{
  "id": 1
}
```
