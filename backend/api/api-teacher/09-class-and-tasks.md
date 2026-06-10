# 模块 9：班级与亲子任务 API

> 对应开发计划：模块 9：班级、亲子任务与补充页
>
> 基础路径：`/api/v1/teacher`
>
> 认证：`Authorization: Bearer <token>`，用户角色为 `teacher`
>
> 以下 Response 均为 `data` 部分，省略外层 `{ "code": 0, "message": "success" }` 包装。

## 接口清单

班级学生：

- `GET /teacher/class/info`
- `GET /teacher/class/students`
- `GET /teacher/class/students/:id`

亲子任务：

- `GET /teacher/tasks`
- `POST /teacher/tasks`
- `PUT /teacher/tasks/:id`
- `DELETE /teacher/tasks/:id`
- `GET /teacher/tasks/:id/stats`
- `GET /teacher/tasks/:id/checkins`

---

## 7. 班级学生

### GET `/teacher/class/info`

> 当前班级信息

```json
// Response
{
  "id": 1,
  "name": "小一班",
  "grade": "小班",
  "academic_year": "2025-2026",
  "capacity": 35,
  "student_count": 30,
  "teachers": [
    { "id": 1, "nickname": "李老师", "avatar": "...", "title": "head", "title_label": "主班老师" },
    { "id": 2, "nickname": "张老师", "avatar": "...", "title": "assistant", "title_label": "副班老师" },
    { "id": 3, "nickname": "刘老师", "avatar": "...", "title": "nursery", "title_label": "生活老师" }
  ]
}
```

---

### GET `/teacher/class/students`

> 班级学生列表

```json
// Query: ?keyword=&page=1&page_size=50

// Response
{
  "list": [
    {
      "id": 1,
      "name": "王小明",
      "avatar": "https://cdn.example.com/avatar/s1.jpg",
      "gender": 1,
      "birthday": "2021-09-15",
      "age": "4岁半",
      "student_no": "2025001",
      "status": 1,
      "parents": [
        { "user_id": 10, "name": "王明", "relation": "father", "is_primary": true },
        { "user_id": 11, "name": "李丽", "relation": "mother", "is_primary": false }
      ]
    }
  ],
  "total": 30
}
```

---

### GET `/teacher/class/students/:id`

> 学生详情（含家长信息、健康备注）

```json
// Response
{
  "id": 1,
  "name": "王小明",
  "avatar": "https://cdn.example.com/avatar/s1.jpg",
  "gender": 1,
  "birthday": "2021-09-15",
  "age": "4岁半",
  "student_no": "2025001",
  "health_note": "对花生过敏",
  "enroll_date": "2025-09-01",
  "class_name": "小一班",
  "status": 1,
  "parents": [
    {
      "user_id": 10,
      "name": "王明",
      "phone": "138****8888",
      "relation": "father",
      "relation_label": "爸爸",
      "is_primary": true,
      "avatar": "https://cdn.example.com/avatar/10.jpg"
    },
    {
      "user_id": 11,
      "name": "李丽",
      "phone": "139****6666",
      "relation": "mother",
      "relation_label": "妈妈",
      "is_primary": false,
      "avatar": "https://cdn.example.com/avatar/11.jpg"
    }
  ],
  "pickup_authorizations": [
    { "id": 1, "name": "爷爷（王建国）", "relation": "grandfather", "phone": "136****5555", "status": 1 }
  ]
}
```

---

## 8. 亲子任务

### GET `/teacher/tasks`

> 任务列表

```json
// Query: ?status=1&page=1&page_size=20
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
      "completion_rate": "57%",
      "checkin_count": 120,
      "total_possible": 210,
      "created_at": "2026-03-18T10:00:00Z"
    },
    {
      "id": 2,
      "title": "手工制作：春天的小树",
      "description": "和孩子一起用彩纸和剪刀制作一棵春天的小树",
      "frequency": "once",
      "frequency_label": "一次任务",
      "start_date": "2026-03-18",
      "end_date": "2026-03-23",
      "points": 15,
      "status": 1,
      "completion_rate": "0%",
      "checkin_count": 0,
      "total_possible": 30,
      "created_at": "2026-03-17T14:00:00Z"
    }
  ],
  "total": 8
}
```

---

### POST `/teacher/tasks`

> 布置新任务

```json
// Request
{
  "title": "每日阅读10分钟",
  "description": "每天陪孩子阅读10分钟绘本，培养阅读习惯",
  "frequency": "daily",         // daily=每日 / weekly=每周 / once=一次性
  "start_date": "2026-03-19",
  "end_date": "2026-03-25",
  "points": 5
}

// Response
{
  "id": 1,
  "created_at": "2026-03-18T10:00:00Z"
}
```

---

### PUT `/teacher/tasks/:id`

> 编辑任务

```json
// Request
{
  "title": "每日阅读15分钟",
  "description": "每天陪孩子阅读15分钟绘本，培养阅读习惯（时间延长）",
  "end_date": "2026-03-30",
  "points": 5
}

// Response
{
  "id": 1,
  "updated_at": "2026-03-19T10:00:00Z"
}
```

---

### DELETE `/teacher/tasks/:id`

> 删除任务

```json
// Response
{}
```

---

### GET `/teacher/tasks/:id/stats`

> 任务完成率统计

```json
// Response
{
  "task": {
    "id": 1,
    "title": "每日阅读10分钟",
    "frequency": "daily",
    "start_date": "2026-03-19",
    "end_date": "2026-03-25",
    "points": 5
  },
  "overall": {
    "completion_rate": "57%",
    "total_checkins": 120,
    "total_possible": 210,
    "total_points_awarded": 600
  },
  "student_stats": [
    {
      "student_id": 1,
      "student_name": "王小明",
      "avatar": "...",
      "checkin_count": 4,
      "total_days": 7,
      "completion_rate": "57%",
      "streak_days": 4
    },
    {
      "student_id": 2,
      "student_name": "李小花",
      "avatar": "...",
      "checkin_count": 7,
      "total_days": 7,
      "completion_rate": "100%",
      "streak_days": 7
    }
  ]
}
```

---

### GET `/teacher/tasks/:id/checkins`

> 任务打卡记录

```json
// Query: ?student_id=&date=&page=1&page_size=20

// Response
{
  "list": [
    {
      "id": 1,
      "student": { "id": 1, "name": "王小明", "avatar": "..." },
      "parent": { "id": 10, "nickname": "王小明爸爸" },
      "date": "2026-03-19",
      "content": "今天和爸爸一起读了《小王子》第三章",
      "image_url": "https://cdn.example.com/task/checkin/1.jpg",
      "points": 5,
      "created_at": "2026-03-19T20:30:00Z"
    }
  ],
  "total": 120
}
```
