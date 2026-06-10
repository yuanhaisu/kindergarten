# 家长端 API — 通知

> 基础路径：`/api/v1/parent` | 鉴权：`Authorization: Bearer <token>`（角色 `parent`）
>
> Response 均为 `data` 部分，省略外层 `{ "code": 0, "message": "success" }` 包装。

---

### GET `/parent/notices`

> 通知列表

```json
// Query: ?category=homework&is_read=false&page=1&page_size=20
// category: homework / bring / activity / health / task / other

// Response
{
  "list": [
    {
      "id": 201,
      "category": "homework",
      "category_label": "作业",
      "title": "周末作业通知",
      "content_preview": "各位家长您好，本周作业如下：1. 复习本周学过的儿歌...",
      "need_confirm": true,
      "is_read": false,
      "is_confirmed": false,
      "attachments_count": 2,
      "creator": { "nickname": "李老师", "avatar": "..." },
      "published_at": "2026-03-19T09:00:00Z"
    }
  ],
  "total": 24,
  "page": 1,
  "page_size": 20
}
```

---

### GET `/parent/notices/:id`

> 通知详情

```json
// Response
{
  "id": 201,
  "category": "homework",
  "category_label": "作业",
  "title": "周末作业通知",
  "content": "各位家长您好，本周作业如下：1. 复习本周学过的儿歌；2. 完成手工制作《我的家》；3. 请家长和孩子一起阅读绘本30分钟。请于下周一前完成，感谢配合！",
  "need_confirm": true,
  "is_read": true,
  "read_at": "2026-03-19T09:15:00Z",
  "is_confirmed": false,
  "confirmed_at": null,
  "attachments": [
    {
      "id": 10,
      "file_url": "https://cdn.example.com/uploads/notice/1.jpg",
      "file_type": "image",
      "file_name": "活动照片1.jpg",
      "file_size": 512000
    }
  ],
  "creator": {
    "id": 1,
    "nickname": "李老师",
    "avatar": "https://cdn.example.com/avatar/1.jpg"
  },
  "published_at": "2026-03-19T09:00:00Z"
}
```

---

### PUT `/parent/notices/:id/confirm`

> 确认通知

```json
// Request
{}

// Response
{
  "id": 201,
  "is_confirmed": true,
  "confirmed_at": "2026-03-19T09:20:00Z"
}
```
