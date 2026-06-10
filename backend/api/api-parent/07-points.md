# 家长端 API — 积分与成就

> 基础路径：`/api/v1/parent` | 鉴权：`Authorization: Bearer <token>`（角色 `parent`）
>
> Response 均为 `data` 部分，省略外层 `{ "code": 0, "message": "success" }` 包装。

---

### GET `/parent/points/overview`

> 积分总览

```json
// Query: ?student_id=1

// Response
{
  "student": { "id": 1, "name": "王小明", "avatar": "..." },
  "total_points": 1245,
  "level": 5,
  "level_name": "超级小星星",
  "level_icon": "https://cdn.example.com/levels/5.png",
  "next_level_points": 1500,
  "rank_in_class": 3,
  "class_total": 30,
  "this_week": 45,
  "this_month": 180
}
```

---

### GET `/parent/points/history`

> 积分流水

```json
// Query: ?student_id=1&type=earn&page=1&page_size=20
// type: earn=获取 / spend=消费 / all=全部

// Response
{
  "list": [
    {
      "id": 1,
      "type": "earn",
      "source": "task_checkin",
      "source_label": "任务打卡",
      "amount": 5,
      "balance": 1245,
      "description": "完成任务【每日阅读10分钟】",
      "reference_id": 1,
      "created_at": "2026-03-19T20:30:00Z"
    }
  ],
  "total": 128,
  "page": 1,
  "page_size": 20
}
```

---

### GET `/parent/points/leaderboard`

> 积分排行榜

```json
// Query: ?student_id=1&period=weekly
// period: weekly / monthly / all

// Response
{
  "period": "weekly",
  "my_rank": 3,
  "my_points": 45,
  "list": [
    {
      "rank": 1,
      "student": { "id": 2, "name": "李小花", "avatar": "..." },
      "points": 60,
      "is_self": false
    },
    {
      "rank": 2,
      "student": { "id": 3, "name": "张小强", "avatar": "..." },
      "points": 50,
      "is_self": false
    },
    {
      "rank": 3,
      "student": { "id": 1, "name": "王小明", "avatar": "..." },
      "points": 45,
      "is_self": true
    }
  ]
}
```

---

### GET `/parent/achievements`

> 成就勋章列表

```json
// Query: ?student_id=1

// Response
{
  "earned": [
    {
      "id": 1,
      "name": "全勤之星",
      "description": "连续一个月全勤",
      "icon_url": "https://cdn.example.com/badges/attendance.png",
      "category": "attendance",
      "earned_at": "2026-03-01T08:00:00Z"
    }
  ],
  "not_earned": [
    {
      "id": 3,
      "name": "艺术小天才",
      "description": "参与10次美术/手工活动",
      "icon_url": "https://cdn.example.com/badges/art_locked.png",
      "category": "growth",
      "progress": {
        "current": 6,
        "target": 10,
        "rate": "60%"
      }
    }
  ],
  "earned_count": 8,
  "total_count": 20
}
```
