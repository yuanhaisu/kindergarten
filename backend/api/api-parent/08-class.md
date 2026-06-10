# 家长端 API — 班级

> 基础路径：`/api/v1/parent` | 鉴权：`Authorization: Bearer <token>`（角色 `parent`）
>
> Response 均为 `data` 部分，省略外层 `{ "code": 0, "message": "success" }` 包装。

---

### GET `/parent/class/info`

> 班级信息

```json
// Query: ?student_id=1

// Response
{
  "id": 1,
  "name": "小一班",
  "grade": "小班",
  "academic_year": "2025-2026",
  "student_count": 30,
  "teachers": [
    { "id": 1, "nickname": "李老师", "avatar": "...", "title_label": "主班老师", "phone": "138****1234" },
    { "id": 2, "nickname": "张老师", "avatar": "...", "title_label": "副班老师", "phone": "139****5678" },
    { "id": 3, "nickname": "刘老师", "avatar": "...", "title_label": "生活老师", "phone": "137****9012" }
  ]
}
```

---

### GET `/parent/class/calendar`

> 班级日历（家长端简化版）

```json
// Query: ?student_id=1&year=2026&month=3

// Response
{
  "list": [
    {
      "id": 1,
      "title": "春季出游活动",
      "event_type": "activity",
      "event_type_label": "活动",
      "event_date": "2026-03-25",
      "start_time": "09:00",
      "end_time": "15:00",
      "location": "城市公园",
      "color": "#10B981"
    },
    {
      "id": 2,
      "title": "体检",
      "event_type": "checkup",
      "event_type_label": "体检",
      "event_date": "2026-03-28",
      "start_time": "09:00",
      "end_time": "11:00",
      "location": "幼儿园医务室",
      "color": "#EF4444"
    }
  ],
  "total": 5
}
```
