# 家长端 API — 成长时光

> 基础路径：`/api/v1/parent` | 鉴权：`Authorization: Bearer <token>`（角色 `parent`）
>
> Response 均为 `data` 部分，省略外层 `{ "code": 0, "message": "success" }` 包装。

---

### GET `/parent/growth/feed`

> 成长时光流（家长端时间线）

```json
// Query: ?student_id=1&year=2026&month=3&page=1&page_size=10

// Response
{
  "list": [
    {
      "id": 1,
      "date": "2026-03-19",
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
          "file_url": "https://cdn.example.com/growth/2.mp4",
          "thumb_url": "https://cdn.example.com/growth/2_thumb.jpg",
          "file_type": "video",
          "duration": 30
        }
      ],
      "media_count": 6,
      "likes_count": 28,
      "comments_count": 12,
      "is_liked": true,
      "creator": {
        "id": 1,
        "nickname": "李老师",
        "avatar": "https://cdn.example.com/avatar/1.jpg"
      },
      "created_at": "2026-03-19T11:30:00Z"
    }
  ],
  "total": 236,
  "page": 1,
  "page_size": 10
}
```

---

### GET `/parent/growth/feed/:id`

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
    }
  ],
  "media_count": 6,
  "likes_count": 28,
  "comments_count": 12,
  "is_liked": true,
  "related_students": [
    { "id": 1, "name": "王小明", "avatar": "..." }
  ],
  "creator": {
    "id": 1,
    "nickname": "李老师",
    "avatar": "https://cdn.example.com/avatar/1.jpg"
  },
  "created_at": "2026-03-19T11:30:00Z"
}
```

---

### POST `/parent/growth/feed/:id/like`

> 点赞成长日志

```json
// Request
{}

// Response
{
  "is_liked": true,
  "likes_count": 29
}
```

---

### DELETE `/parent/growth/feed/:id/like`

> 取消点赞

```json
// Response
{
  "is_liked": false,
  "likes_count": 28
}
```

---

### GET `/parent/growth/feed/:id/comments`

> 评论列表

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
      "content": "谢谢李老师的表扬和鼓励！小明回家特别开心 😊",
      "parent_id": null,
      "replies": [
        {
          "id": 5,
          "user": { "id": 1, "nickname": "李老师", "avatar": "..." },
          "content": "小明今天表现确实很棒！",
          "parent_id": 1,
          "created_at": "2026-03-19T13:10:00Z"
        }
      ],
      "created_at": "2026-03-19T13:05:00Z"
    }
  ],
  "total": 12
}
```

---

### POST `/parent/growth/feed/:id/comments`

> 发表评论

```json
// Request
{
  "content": "谢谢李老师的表扬和鼓励！小明回家特别开心 😊",
  "parent_id": null            // 回复某条评论传 ID，一级评论传 null
}

// Response
{
  "id": 1,
  "created_at": "2026-03-19T13:05:00Z"
}
```

---

### GET `/parent/growth/album`

> 成长相册（按月份分组的照片视频列表）

```json
// Query: ?student_id=1&year=2026&month=3

// Response
{
  "year": 2026,
  "month": 3,
  "groups": [
    {
      "date": "2026-03-19",
      "media": [
        {
          "id": 1,
          "file_url": "https://cdn.example.com/growth/1.jpg",
          "thumb_url": "https://cdn.example.com/growth/1_thumb.jpg",
          "file_type": "image",
          "log_id": 1
        }
      ]
    }
  ],
  "total": 28
}
```

---

### GET `/parent/growth/report`

> 成长报告（周报 / 月报 / 学期报告）

```json
// Query: ?student_id=1&type=weekly&date=2026-03-17
// type: weekly / monthly / term

// Response
{
  "student": { "id": 1, "name": "王小明", "avatar": "..." },
  "period": {
    "type": "weekly",
    "start_date": "2026-03-17",
    "end_date": "2026-03-23"
  },
  "attendance": {
    "present_days": 4,
    "leave_days": 1,
    "total_days": 5,
    "rate": "80%"
  },
  "meals": {
    "breakfast_avg": 2.8,
    "lunch_avg": 2.5,
    "dinner_avg": 3.0
  },
  "sleep": {
    "avg_nap_duration": 108,
    "avg_sleep_quality": 2.6
  },
  "mood": {
    "morning_avg": 2.7,
    "afternoon_avg": 2.5
  },
  "top_performance_tags": [
    { "tag": "乐于助人", "count": 3 },
    { "tag": "积极发言", "count": 2 },
    { "tag": "遵守纪律", "count": 4 }
  ],
  "growth_media_count": 15,
  "points_earned": 45,
  "tasks_completed": 8,
  "tasks_total": 12,
  "dimensions": [
    { "name": "语言", "score": 85, "max_score": 100, "comment": "表达能力强，词汇量丰富" },
    { "name": "认知", "score": 78, "max_score": 100, "comment": "逻辑思维良好" },
    { "name": "社交", "score": 90, "max_score": 100, "comment": "与同伴相处融洽" },
    { "name": "运动", "score": 72, "max_score": 100, "comment": "大运动发展良好" },
    { "name": "艺术", "score": 95, "max_score": 100, "comment": "绘画色彩感强" },
    { "name": "自理", "score": 68, "max_score": 100, "comment": "独立穿衣待加强" },
    { "name": "情绪", "score": 82, "max_score": 100, "comment": "情绪稳定，善于表达" }
  ],
  "highlights": [
    "美术作品《春天的树》被选为班级展示作品",
    "本周全勤，并获得全勤之星",
    "主动帮助同学收拾玩具"
  ],
  "concerns": [
    "午睡时间偏短，建议关注晚间睡眠",
    "蔬菜摄入量偏少，鼓励尝试新种类"
  ],
  "teacher_comment": "王小明本周整体表现优秀，在美术课上表现特别积极，希望继续保持！"
}
```
