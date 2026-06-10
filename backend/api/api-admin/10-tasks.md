# 10. 亲子任务管理

> 对应前端页面：`admin/pages/tasks/index.vue`
>
> 基础路径：`/api/v1/admin`，需 `Authorization: Bearer <token>`，角色 `admin`。
>
> 状态：✅ 6 个接口已实现（含新建/编辑）；`admin/api/tasks.ts` 已封装 `createTask` / `updateTask`。

---

## POST `/admin/tasks`

> 新建任务（管理端；`creator_id` 取当前登录用户）

```json
// Request
{
  "title": "每日阅读10分钟",
  "description": "每天陪孩子阅读10分钟绘本",
  "class_id": 1,
  "frequency": "daily",
  "start_date": "2026-03-19",
  "end_date": "2026-03-25",
  "points": 5
}

// Response（HTTP 201）
{
  "id": 1,
  "created_at": "2026-03-18T10:00:00Z"
}
```

`frequency`：`daily` | `weekly` | `once`。`points` 缺省或 ≤0 时按 1。

---

## PUT `/admin/tasks/:id`

> 编辑任务（全量字段）

```json
// Request
{
  "title": "每日阅读10分钟",
  "description": "…",
  "class_id": 1,
  "frequency": "daily",
  "start_date": "2026-03-19",
  "end_date": "2026-03-25",
  "points": 5,
  "status": 1
}

// Response
{ "id": 1 }
```

`status`：`0` 已结束，`1` 进行中。

---

## GET `/admin/tasks`

> 全园任务列表（对应前端 `getTasks`）

```json
// Query: ?class_id=&status=&page=1&page_size=20

// Response
{
  "list": [
    {
      "id": 1,
      "title": "每日阅读10分钟",
      "class_name": "小一班",
      "class_id": 1,
      "frequency": "daily",
      "frequency_label": "每日任务",
      "start_date": "2026-03-19",
      "end_date": "2026-03-25",
      "points": 5,
      "status": 1,
      "completion_rate": "57%",
      "creator": { "nickname": "李老师" },
      "created_at": "2026-03-18T10:00:00Z"
    }
  ],
  "total": 80,
  "page": 1,
  "page_size": 20
}
```

---

## GET `/admin/tasks/:id`

> 任务详情（含完成统计；对应前端 `getTask`）

```json
// Response
{
  "id": 1,
  "title": "每日阅读10分钟",
  "description": "每天陪孩子阅读10分钟绘本，培养阅读习惯",
  "class_id": 1,
  "class_name": "小一班",
  "frequency": "daily",
  "frequency_label": "每日任务",
  "start_date": "2026-03-19",
  "end_date": "2026-03-25",
  "points": 5,
  "status": 1,
  "completion_rate": "57%",
  "checkin_count": 120,
  "total_possible": 210,
  "creator": { "id": 1, "nickname": "李老师", "avatar": "..." },
  "created_at": "2026-03-18T10:00:00Z"
}
```

---

## DELETE `/admin/tasks/:id`

> 删除任务（管理员强制删除；对应前端 `deleteTask`）

```json
// Response
{}
```

---

## GET `/admin/tasks/stats`

> 任务统计（对应前端 `getTaskStats`，`/statistics` 页面也使用）

```json
// Query: ?start_date=2026-03-01&end_date=2026-03-31

// Response
{
  "period": { "start": "2026-03-01", "end": "2026-03-31" },
  "total_tasks": 25,
  "active_tasks": 8,
  "completed_tasks": 17,
  "total_checkins": 3200,
  "total_points_awarded": 16000,
  "avg_completion_rate": "62%",
  "by_class": [
    { "class_name": "小一班", "tasks": 5, "avg_rate": "68%" },
    { "class_name": "小二班", "tasks": 3, "avg_rate": "55%" }
  ]
}
```
