# 模块 8：班级日历 API

> 对应开发计划：模块 8：班级日历
>
> 基础路径：`/api/v1/teacher`
>
> 认证：`Authorization: Bearer <token>`，用户角色为 `teacher`
>
> 以下 Response 均为 `data` 部分，省略外层 `{ "code": 0, "message": "success" }` 包装。

## 接口清单

- `GET /teacher/calendar/events`
- `GET /teacher/calendar/events/:date`
- `POST /teacher/calendar/events`
- `PUT /teacher/calendar/events/:id`
- `DELETE /teacher/calendar/events/:id`
- `GET /teacher/calendar/dots`

---

## 4. 班级日历

### GET `/teacher/calendar/events`

> 日历事件列表（月视图）

```json
// Query: ?year=2026&month=3

// Response
{
  "list": [
    {
      "id": 1,
      "title": "春季出游活动通知",
      "description": "请各位家长提醒孩子准备出游物品，明天早上8点在幼儿园门口集合",
      "event_type": "notice",
      "event_type_label": "通知",
      "event_date": "2026-03-19",
      "start_time": "09:00",
      "end_time": null,
      "location": "",
      "is_all_day": false,
      "color": "#3B82F6",
      "notice_id": 201
    },
    {
      "id": 2,
      "title": "美术课：春天的小树",
      "description": "带领小朋友们用画笔描绘春天的景色，发挥想象力",
      "event_type": "activity",
      "event_type_label": "活动",
      "event_date": "2026-03-19",
      "start_time": "10:00",
      "end_time": "11:00",
      "location": "教室",
      "is_all_day": false,
      "color": "#10B981",
      "notice_id": null
    }
  ],
  "total": 15
}
```

---

### GET `/teacher/calendar/events/:date`

> 某天的事件详情列表

```json
// :date = 2026-03-19

// Response
{
  "date": "2026-03-19",
  "weekday": "星期四",
  "events": [
    {
      "id": 1,
      "title": "春季出游活动通知",
      "description": "请各位家长提醒孩子准备出游物品...",
      "event_type": "notice",
      "event_type_label": "通知",
      "start_time": "09:00",
      "end_time": null,
      "location": "",
      "color": "#3B82F6"
    },
    {
      "id": 2,
      "title": "美术课：春天的小树",
      "description": "带领小朋友们用画笔描绘春天的景色",
      "event_type": "activity",
      "event_type_label": "活动",
      "start_time": "10:00",
      "end_time": "11:00",
      "location": "教室",
      "color": "#10B981"
    },
    {
      "id": 3,
      "title": "音乐课：学习新儿歌",
      "description": "教小朋友们唱《春天在哪里》，配合动作表演",
      "event_type": "activity",
      "event_type_label": "活动",
      "start_time": "15:00",
      "end_time": "15:40",
      "location": "音乐教室",
      "color": "#8B5CF6"
    }
  ]
}
```

---

### POST `/teacher/calendar/events`

> 创建日历活动

```json
// Request
{
  "title": "美术课：春天的小树",
  "description": "带领小朋友们用画笔描绘春天的景色，发挥想象力",
  "event_type": "activity",
  "event_date": "2026-03-19",
  "start_time": "10:00",
  "end_time": "11:00",
  "location": "教室",
  "is_all_day": false,
  "color": "#10B981"
}

// Response
{
  "id": 2,
  "created_at": "2026-03-18T16:00:00Z"
}
```

---

### PUT `/teacher/calendar/events/:id`

> 编辑日历活动

```json
// Request
{
  "title": "美术课：春天的小树（调整）",
  "description": "带领小朋友们用画笔描绘春天的景色",
  "event_type": "activity",
  "event_date": "2026-03-19",
  "start_time": "10:30",
  "end_time": "11:30",
  "location": "美术教室",
  "is_all_day": false,
  "color": "#10B981"
}

// Response
{
  "id": 2,
  "updated_at": "2026-03-19T08:00:00Z"
}
```

---

### DELETE `/teacher/calendar/events/:id`

> 删除日历活动

```json
// Response
{}
```

---

### GET `/teacher/calendar/dots`

> 月视图日期标记点（日历上显示的彩色圆点）

```json
// Query: ?year=2026&month=3

// Response
{
  "dots": {
    "2026-03-19": ["notice", "activity"],
    "2026-03-20": ["activity"],
    "2026-03-21": ["holiday"],
    "2026-03-23": ["meeting"],
    "2026-03-25": ["checkup"]
  }
}
```
