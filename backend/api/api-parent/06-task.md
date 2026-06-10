# 家长端 API — 亲子任务

> 基础路径：`/api/v1/parent` | 鉴权：`Authorization: Bearer <token>`（角色 `parent`）
>
> Response 均为 `data` 部分，省略外层 `{ "code": 0, "message": "success" }` 包装。

---

### GET `/parent/tasks`

> 任务列表

```json
// Query: ?student_id=1&status=1&page=1&page_size=20
// status: 1=进行中 0=已结束

// Response
{
  "list": [
    {
      "id": 1,
      "title": "每日阅读10分钟",
      "description": "每天陪孩子阅读10分钟绘本，培养阅读习惯",
      "frequency": "daily",
      "frequency_label": "每日任务",
      "start_date": "2026-03-19",
      "end_date": "2026-03-25",
      "points": 5,
      "status": 1,
      "today_checked_in": false,
      "streak_days": 4,
      "my_completion_rate": "57%",
      "created_at": "2026-03-18T10:00:00Z"
    }
  ],
  "total": 5,
  "page": 1,
  "page_size": 20
}
```

---

### GET `/parent/tasks/:id`

> 任务详情

```json
// Response
{
  "id": 1,
  "title": "每日阅读10分钟",
  "description": "每天陪孩子阅读10分钟绘本，培养阅读习惯",
  "intro": "为了培养孩子良好的阅读习惯，请家长每天陪孩子阅读10分钟绘本，并上传一张阅读照片作为记录。",
  "requirements": [
    "每天阅读时长不少于10分钟",
    "家长需陪同阅读",
    "打卡时上传一张阅读照片",
    "可重复阅读同一本书"
  ],
  "rewards": [
    { "type": "points", "value": 5, "desc": "每次打卡 +5积分" },
    { "type": "badge", "name": "阅读小达人", "desc": "累计打卡30天解锁" }
  ],
  "frequency": "daily",
  "frequency_label": "每日任务",
  "start_date": "2026-03-19",
  "end_date": "2026-03-25",
  "points": 5,
  "status": 1,
  "today_checked_in": false,
  "streak_days": 4,
  "my_completion_rate": "57%",
  "my_checkin_count": 4,
  "total_days": 7,
  "creator": {
    "id": 1,
    "nickname": "李老师",
    "avatar": "https://cdn.example.com/avatar/1.jpg"
  },
  "created_at": "2026-03-18T10:00:00Z"
}
```

---

### POST `/parent/tasks/:id/checkin`

> 打卡

```json
// Request
{
  "student_id": 1,
  "content": "今天和爸爸一起读了《小王子》第三章",
  "image_url": "https://cdn.example.com/task/checkin/1.jpg"   // 可选
}

// Response
{
  "id": 1,
  "points_earned": 5,
  "streak_days": 5,
  "total_points": 1245,
  "message": "+5积分！已连续打卡5天，太棒了！",
  "created_at": "2026-03-19T20:30:00Z"
}
```

---

### GET `/parent/tasks/:id/checkin-history`

> 打卡历史

```json
// Query: ?student_id=1&page=1&page_size=20

// Response
{
  "list": [
    {
      "id": 1,
      "date": "2026-03-19",
      "content": "今天和爸爸一起读了《小王子》第三章",
      "image_url": "https://cdn.example.com/task/checkin/1.jpg",
      "points": 5,
      "created_at": "2026-03-19T20:30:00Z"
    },
    {
      "id": 2,
      "date": "2026-03-18",
      "content": "一起阅读了《猜猜我有多爱你》",
      "image_url": null,
      "points": 5,
      "created_at": "2026-03-18T20:00:00Z"
    }
  ],
  "total": 4,
  "streak_days": 4
}
```
