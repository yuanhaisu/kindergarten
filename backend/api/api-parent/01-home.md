# 家长端 API — 首页

> 基础路径：`/api/v1/parent` | 鉴权：`Authorization: Bearer <token>`（角色 `parent`）
>
> Response 均为 `data` 部分，省略外层 `{ "code": 0, "message": "success" }` 包装。

---

### GET `/parent/home/overview`

> 首页概览（当前孩子的今日动态）

```json
// Response
{
  "date": "2026-03-19",
  "weekday": "星期四",
  "current_student": {
    "id": 1,
    "name": "王小明",
    "avatar": "https://cdn.example.com/avatar/s1.jpg",
    "class_name": "小一班",
    "class_id": 1
  },
  "students": [
    { "id": 1, "name": "王小明", "avatar": "...", "class_name": "小一班" },
    { "id": 5, "name": "王小花", "avatar": "...", "class_name": "大二班" }
  ],
  "today_summary": {
    "attendance_status": "present",
    "attendance_label": "已入园",
    "arrive_time": "07:52",
    "meal_rating": 3,
    "meal_label": "很好",
    "nap_duration": 120,
    "sleep_quality": 3,
    "sleep_label": "很好",
    "mood_rating": 3,
    "mood_label": "很好",
    "performance_tag": "小小艺术家",
    "new_growth_count": 2,
    "new_notice_count": 1,
    "pending_tasks": 3,
    "points_today": 10,
    "leave_status": null
  }
}
```

---

### PUT `/parent/home/switch-student`

> 切换当前孩子（多孩家庭）

```json
// Request
{
  "student_id": 5
}

// Response
{
  "current_student": {
    "id": 5,
    "name": "王小花",
    "avatar": "https://cdn.example.com/avatar/s5.jpg",
    "class_name": "大二班",
    "class_id": 3
  }
}
```

---

### GET `/parent/home/announcements`

> 首页公告轮播

```json
// Query: ?limit=5

// Response
{
  "list": [
    {
      "id": 1,
      "title": "春季学期家长开放日通知",
      "cover_url": "https://cdn.example.com/banner/1.jpg",
      "is_important": true,
      "published_at": "2026-03-18T10:00:00Z"
    }
  ]
}
```

---

### GET `/parent/home/feeds`

> 首页动态流（成长记录 + 通知混合流）

```json
// Query: ?page=1&page_size=10&student_id=1

// Response
{
  "list": [
    {
      "id": 1,
      "type": "growth",
      "growth_log": {
        "id": 1,
        "content": "今天的美术课主题是《我心中的春天》...",
        "media_preview": [
          { "thumb_url": "https://cdn.example.com/thumb/1.jpg", "type": "image" },
          { "thumb_url": "https://cdn.example.com/thumb/2.jpg", "type": "image" }
        ],
        "media_count": 6,
        "likes_count": 28,
        "comments_count": 12,
        "is_liked": true,
        "creator": { "id": 1, "nickname": "李老师", "avatar": "..." },
        "date": "2026-03-19"
      },
      "created_at": "2026-03-19T11:30:00Z"
    },
    {
      "id": 2,
      "type": "notice",
      "notice": {
        "id": 201,
        "category": "homework",
        "title": "周末作业通知",
        "content_preview": "各位家长您好，本周作业如下...",
        "need_confirm": true,
        "is_read": false,
        "is_confirmed": false
      },
      "created_at": "2026-03-19T09:00:00Z"
    }
  ],
  "total": 56,
  "page": 1,
  "page_size": 10
}
```
