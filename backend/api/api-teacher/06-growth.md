# 模块 6：成长记录 API

> 对应开发计划：模块 6：成长记录
>
> 基础路径：`/api/v1/teacher`
>
> 认证：`Authorization: Bearer <token>`，用户角色为 `teacher`
>
> 以下 Response 均为 `data` 部分，省略外层 `{ "code": 0, "message": "success" }` 包装。

## 接口清单

- `GET /teacher/growth/logs`
- `GET /teacher/growth/logs/:id`
- `POST /teacher/growth/logs`
- `PUT /teacher/growth/logs/:id`
- `DELETE /teacher/growth/logs/:id`
- `GET /teacher/growth/logs/:id/comments`
- `POST /teacher/growth/logs/:id/comments`
- `GET /teacher/growth/logs/:id/likes`
- `GET /teacher/growth/logs/:id/related-students`
- `POST /teacher/growth/media/upload`

---

## 5. 成长记录

### GET `/teacher/growth/logs`

> 成长日志列表

```json
// Query: ?date=2026-03-19&page=1&page_size=20

// Response
{
  "list": [
    {
      "id": 1,
      "date": "2026-03-19",
      "content": "今天的美术课小朋友们都太棒了！...",
      "media_count": 6,
      "media_preview": [
        { "thumb_url": "https://cdn.example.com/thumb/1.jpg", "type": "image" },
        { "thumb_url": "https://cdn.example.com/thumb/2.jpg", "type": "image" }
      ],
      "likes_count": 28,
      "comments_count": 12,
      "related_students_count": 28,
      "sync_to_feed": true,
      "target_type": 1,
      "creator": {
        "id": 1,
        "nickname": "李老师",
        "avatar": "https://cdn.example.com/avatar/1.jpg"
      },
      "created_at": "2026-03-19T11:30:00Z"
    }
  ],
  "total": 236
}
```

---

### GET `/teacher/growth/logs/:id`

> 成长日志详情

```json
// Response
{
  "id": 1,
  "date": "2026-03-19",
  "breakfast_rating": 3,
  "lunch_rating": 2,
  "dinner_rating": 3,
  "nap_duration": 120,
  "sleep_quality": 3,
  "morning_mood": 3,
  "afternoon_mood": 2,
  "performance_tags": ["乐于助人", "积极发言", "遵守纪律"],
  "content": "今天的美术课主题是《我心中的春天》，小朋友们都发挥了自己的想象力...",
  "media": [
    {
      "id": 1,
      "file_url": "https://cdn.example.com/growth/1.jpg",
      "thumb_url": "https://cdn.example.com/growth/1_thumb.jpg",
      "file_type": "image"
    },
    {
      "id": 2,
      "file_url": "https://cdn.example.com/growth/2.jpg",
      "thumb_url": "https://cdn.example.com/growth/2_thumb.jpg",
      "file_type": "image"
    }
  ],
  "sync_to_feed": true,
  "target_type": 1,
  "related_students": [
    { "id": 1, "name": "王小明", "avatar": "..." },
    { "id": 2, "name": "李小花", "avatar": "..." }
  ],
  "related_students_count": 28,
  "likes_count": 28,
  "comments_count": 12,
  "shares_count": 5,
  "creator": {
    "id": 1,
    "nickname": "李老师",
    "avatar": "https://cdn.example.com/avatar/1.jpg"
  },
  "created_at": "2026-03-19T11:30:00Z"
}
```

---

### POST `/teacher/growth/logs`

> 创建成长日志

```json
// Request
{
  "date": "2026-03-19",
  "breakfast_rating": 3,
  "lunch_rating": 2,
  "dinner_rating": 3,
  "nap_duration": 120,
  "sleep_quality": 3,
  "morning_mood": 3,
  "afternoon_mood": 2,
  "performance_tags": ["乐于助人", "积极发言", "遵守纪律"],
  "content": "今天的美术课小朋友们都太棒了！...",
  "media_ids": [1, 2, 3, 4, 5, 6],
  "sync_to_feed": true,
  "target_type": 1,
  "student_ids": []
}

// Response
{
  "id": 1,
  "created_at": "2026-03-19T11:30:00Z"
}
```

---

### PUT `/teacher/growth/logs/:id`

> 编辑成长日志

```json
// Request（同 POST，字段可选更新）
{
  "content": "今天的美术课小朋友们都太棒了！（已更新）",
  "performance_tags": ["乐于助人", "积极发言", "遵守纪律", "动手能力强"],
  "media_ids": [1, 2, 3, 4, 5, 6, 7]
}

// Response
{
  "id": 1,
  "updated_at": "2026-03-19T12:00:00Z"
}
```

---

### DELETE `/teacher/growth/logs/:id`

> 删除成长日志

```json
// Response
{}
```

---

### GET `/teacher/growth/logs/:id/comments`

> 成长日志评论列表

```json
// Query: ?page=1&page_size=20

// Response
{
  "list": [
    {
      "id": 1,
      "user": {
        "id": 10,
        "nickname": "王小明妈妈",
        "avatar": "https://cdn.example.com/avatar/10.jpg",
        "relation": "mother",
        "student_name": "王小明"
      },
      "content": "谢谢李老师的表扬和鼓励！小明回家特别开心，说今天画画特别有意思，还要继续努力呢 😊",
      "parent_id": null,
      "replies": [
        {
          "id": 5,
          "user": {
            "id": 1,
            "nickname": "李老师",
            "avatar": "https://cdn.example.com/avatar/1.jpg"
          },
          "content": "小明今天表现确实很棒，超级优秀，想象力很强！",
          "parent_id": 1,
          "created_at": "2026-03-19T13:10:00Z"
        }
      ],
      "created_at": "2026-03-19T13:05:00Z"
    },
    {
      "id": 2,
      "user": {
        "id": 11,
        "nickname": "李小花妈妈",
        "avatar": "https://cdn.example.com/avatar/11.jpg",
        "relation": "mother",
        "student_name": "李小花"
      },
      "content": "看到小花的作品了，画的真好看！谢谢老师拍照记录～",
      "parent_id": null,
      "replies": [],
      "created_at": "2026-03-19T13:12:00Z"
    }
  ],
  "total": 12
}
```

---

### POST `/teacher/growth/logs/:id/comments`

> 回复评论

```json
// Request
{
  "content": "小明今天表现确实很棒，继续加油！",
  "parent_id": 1                // 回复某条评论，传 null 或不传为一级评论
}

// Response
{
  "id": 5,
  "created_at": "2026-03-19T13:10:00Z"
}
```

---

### GET `/teacher/growth/logs/:id/likes`

> 点赞列表

```json
// Query: ?page=1&page_size=50

// Response
{
  "list": [
    {
      "user_id": 10,
      "nickname": "王小明妈妈",
      "avatar": "https://cdn.example.com/avatar/10.jpg",
      "created_at": "2026-03-19T12:00:00Z"
    },
    {
      "user_id": 11,
      "nickname": "李小花爸爸",
      "avatar": "https://cdn.example.com/avatar/11.jpg",
      "created_at": "2026-03-19T12:05:00Z"
    }
  ],
  "total": 28
}
```

---

### GET `/teacher/growth/logs/:id/related-students`

> 相关孩子列表

```json
// Response
{
  "list": [
    { "id": 1, "name": "王小明", "avatar": "https://cdn.example.com/avatar/s1.jpg" },
    { "id": 2, "name": "李小花", "avatar": "https://cdn.example.com/avatar/s2.jpg" },
    { "id": 3, "name": "张小强", "avatar": "https://cdn.example.com/avatar/s3.jpg" }
  ],
  "total": 28
}
```

---

### POST `/teacher/growth/media/upload`

> 上传成长照片 / 视频（专用上传接口，支持 AI 识别学生）

```
Content-Type: multipart/form-data
字段: files[]     — 文件数组（必填，最多20张照片/5个视频）
字段: date        — 日期（必填）
字段: auto_tag    — 是否 AI 自动识别学生（可选，默认 true）
```

```json
// Response
{
  "media": [
    {
      "id": 1,
      "file_url": "https://cdn.example.com/growth/1.jpg",
      "thumb_url": "https://cdn.example.com/growth/1_thumb.jpg",
      "file_type": "image",
      "file_name": "IMG_001.jpg",
      "file_size": 1024000,
      "tagged_students": [
        { "id": 1, "name": "王小明", "is_auto": true }
      ]
    }
  ],
  "success_count": 6,
  "fail_count": 0
}
```
