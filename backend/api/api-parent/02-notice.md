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
      "confirm_deadline": "2026-03-20T18:00:00Z",
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
  "confirm_deadline": "2026-03-20T18:00:00Z",
  "is_read": true,
  "read_at": "2026-03-19T09:15:00Z",
  "is_confirmed": false,
  "confirmed_at": null,
  "attachments": [
    {
      "id": 10,
      "file_url": "https://cdn.example.com/uploads/notice/1.jpg", // 1 小时签名 URL
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

> `confirm_deadline` 仅 `need_confirm=true` 时有值（带时区 RFC3339）。客户端用它与当前时间判断「待确认 / 已截止」；服务端在确认接口做最终校验。

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

> 当前时间 `>=` `confirm_deadline` 时返回 `409`（业务码 `40900`，`确认已截止`），不得写入 `is_confirmed` / `confirmed_at`；家长端应隐藏确认栏并显示「已截止」。`need_confirm=false` 的通知不展示确认入口。
