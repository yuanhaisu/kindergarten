# 模块 4：通知管理 API

> 对应开发计划：模块 4：通知管理
>
> 基础路径：`/api/v1/teacher`
>
> 认证：`Authorization: Bearer <token>`，用户角色为 `teacher`
>
> 以下 Response 均为 `data` 部分，省略外层 `{ "code": 0, "message": "success" }` 包装。

## 接口清单

- `GET /teacher/notices`
- `GET /teacher/notices/:id`
- `POST /teacher/notices`
- `PUT /teacher/notices/:id`
- `DELETE /teacher/notices/:id`
- `PUT /teacher/notices/:id/revoke`
- `POST /teacher/notices/:id/remind`
- `GET /teacher/notices/:id/read-status`
- `GET /teacher/notices/templates`
- `POST /teacher/notices/templates`
- `PUT /teacher/notices/templates/:id`
- `DELETE /teacher/notices/templates/:id`

---

## 2. 通知管理

### GET `/teacher/notices`

> 通知列表

```json
// Query: ?status=2&category=homework&page=1&page_size=20
// status: 0=草稿 1=定时待发 2=已发布 3=已撤回，不传查全部
// category: homework / bring / activity / health / other，不传查全部

// Response
{
  "list": [
    {
      "id": 1,
      "category": "homework",
      "category_label": "作业",
      "title": "周末作业通知",
      "content": "各位家长您好，本周作业如下：1. 复习本周学过的儿歌...",
      "need_confirm": true,
      "status": 2,
      "status_label": "已发布",
      "read_count": 25,
      "total_count": 30,
      "read_rate": "83%",
      "attachments_count": 2,
      "published_at": "2026-03-19T09:00:00Z",
      "created_at": "2026-03-19T08:50:00Z"
    }
  ],
  "total": 36,
  "page": 1,
  "page_size": 20
}
```

---

### GET `/teacher/notices/:id`

> 通知详情（含已读统计）

```json
// Response
{
  "id": 1,
  "category": "homework",
  "category_label": "作业",
  "title": "周末作业通知",
  "content": "各位家长您好，本周作业如下：1. 复习本周学过的儿歌；2. 完成手工制作《我的家》...",
  "need_confirm": true,
  "status": 2,
  "scheduled_at": null,
  "published_at": "2026-03-19T09:00:00Z",
  "attachments": [
    {
      "id": 10,
      "file_url": "https://cdn.example.com/uploads/notice/1.jpg",
      "file_type": "image",
      "file_name": "活动照片1.jpg",
      "file_size": 512000
    }
  ],
  "read_stats": {
    "read_count": 25,
    "unread_count": 5,
    "confirmed_count": 20,
    "total_count": 30,
    "read_rate": "83%"
  },
  "creator": {
    "id": 1,
    "nickname": "李老师",
    "avatar": "https://cdn.example.com/avatar/1.jpg"
  },
  "created_at": "2026-03-19T08:50:00Z"
}
```

---

### POST `/teacher/notices`

> 创建 / 发布通知

```json
// Request
{
  "category": "homework",
  "title": "周末作业通知",
  "content": "各位家长您好，本周作业如下...",
  "attachments": [
    {
      "file_url": "https://cdn.example.com/uploads/notice/1.jpg",
      "file_type": "image",
      "file_name": "活动照片1.jpg",
      "file_size": 512000
    }
  ],
  "need_confirm": true,
  "scheduled_at": null,           // null=立即发布，传时间=定时发送
  "template_id": null             // 使用的模板 ID，可选
}

// Response
{
  "id": 1,
  "status": 2,
  "published_at": "2026-03-19T09:00:00Z"
}
```

---

### PUT `/teacher/notices/:id`

> 编辑通知（仅草稿 / 定时待发状态可编辑）

```json
// Request
{
  "category": "homework",
  "title": "周末作业通知（修改版）",
  "content": "各位家长您好，本周作业更新如下...",
  "attachments": [
    {
      "file_url": "https://cdn.example.com/uploads/notice/1.jpg",
      "file_type": "image",
      "file_name": "活动照片1.jpg",
      "file_size": 512000
    }
  ],
  "need_confirm": true,
  "scheduled_at": null
}

// Response
{
  "id": 1,
  "status": 0,
  "updated_at": "2026-03-19T09:10:00Z"
}
```

---

### DELETE `/teacher/notices/:id`

> 删除通知（仅草稿可删除）

```json
// Response
{}
```

---

### PUT `/teacher/notices/:id/revoke`

> 撤回已发布的通知

```json
// Request
{}

// Response
{
  "id": 1,
  "status": 3,
  "status_label": "已撤回"
}
```

---

### POST `/teacher/notices/:id/remind`

> 一键提醒未读家长

```json
// Request
{}

// Response
{
  "reminded_count": 5,
  "message": "已向5位未读家长发送提醒"
}
```

---

### GET `/teacher/notices/:id/read-status`

> 已读 / 未读家长名单

```json
// Query: ?status=read&page=1&page_size=50
// status: read=已读 / unread=未读 / confirmed=已确认，不传查全部

// Response
{
  "list": [
    {
      "user_id": 10,
      "parent_name": "王小明妈妈",
      "avatar": "https://cdn.example.com/avatar/10.jpg",
      "student_name": "王小明",
      "relation": "mother",
      "is_read": true,
      "read_at": "2026-03-19T09:15:00Z",
      "is_confirmed": true,
      "confirmed_at": "2026-03-19T09:16:00Z"
    },
    {
      "user_id": 11,
      "parent_name": "李小花爸爸",
      "avatar": "https://cdn.example.com/avatar/11.jpg",
      "student_name": "李小花",
      "relation": "father",
      "is_read": false,
      "read_at": null,
      "is_confirmed": false,
      "confirmed_at": null
    }
  ],
  "total": 30,
  "read_count": 25,
  "unread_count": 5,
  "confirmed_count": 20
}
```

---

### GET `/teacher/notices/templates`

> 通知模板列表

```json
// Query: ?category=homework&page=1&page_size=20

// Response
{
  "list": [
    {
      "id": 1,
      "category": "homework",
      "category_label": "作业",
      "title": "周末作业通知",
      "content": "各位家长您好，本周作业如下：1. 复习本周学过的儿歌；2. 完成手工制作《我的家》；3. 请家长和孩子一起阅读绘本30分钟。请于下周一前完成，感谢配合！",
      "usage_count": 12,
      "created_at": "2026-01-15T10:00:00Z"
    },
    {
      "id": 2,
      "category": "activity",
      "category_label": "活动",
      "title": "春季出游通知",
      "content": "各位家长您好，我们将于3月25日组织春季出游活动...",
      "usage_count": 3,
      "created_at": "2026-02-20T14:00:00Z"
    }
  ],
  "total": 8
}
```

---

### POST `/teacher/notices/templates`

> 创建通知模板

```json
// Request
{
  "category": "homework",
  "title": "每日作业通知",
  "content": "各位家长您好，今日作业如下：..."
}

// Response
{
  "id": 3,
  "created_at": "2026-03-19T10:00:00Z"
}
```

---

### PUT `/teacher/notices/templates/:id`

> 编辑模板

```json
// Request
{
  "category": "homework",
  "title": "每日作业通知（更新版）",
  "content": "各位家长您好，今日作业更新如下：..."
}

// Response
{
  "id": 3,
  "updated_at": "2026-03-19T10:30:00Z"
}
```

---

### DELETE `/teacher/notices/templates/:id`

> 删除模板

```json
// Response
{}
```
