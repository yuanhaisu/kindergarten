# 10. 亲子任务管理

> 对应前端页面：`admin/pages/tasks/index.vue`
>
> 基础路径：`/api/v1/admin`，需 `Authorization: Bearer <token>`，角色 `admin`。
>
> **A 切片状态**：`points` 已从 domain/契约移除；活动分组、新字段统计、班级罐配置属于 C 切片，下列标注 `[C]` 的端点/字段尚未实现。

---

## POST `/admin/tasks`

> 新建独立任务（`creator_id` 取当前登录用户）

```json
// Request
{
  "title": "每日阅读10分钟",
  "description": "每天陪孩子阅读10分钟绘本",
  "class_id": 1,
  "frequency": "daily",
  "start_date": "2026-03-19",
  "end_date": "2026-03-25"
}

// Response（HTTP 201）
{
  "id": 1,
  "created_at": "2026-03-18T10:00:00Z"
}
```

`frequency`：`daily` | `weekly` | `once`。不接收 `points`。

> `[C]` 后续新增：`learning_link`、`target_student_id`、`allow_public`、`recognition_ritual`、`star_scope`、`star_goal`。

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
  "status": 1
}
```

`status`：`0` 已结束，`1` 进行中。不接收 `points`。

---

## GET `/admin/tasks`

> 全园任务列表

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

- `points` 已移除。
- `[C]` 旧 `completion_rate` 将下线，改为参与率/回应率。

---

## GET `/admin/tasks/:id`

> 任务详情

```json
{
  "id": 1,
  "title": "每日阅读10分钟",
  "description": "每天陪孩子阅读10分钟绘本",
  "class_id": 1,
  "class_name": "小一班",
  "frequency": "daily",
  "start_date": "2026-03-19",
  "end_date": "2026-03-25",
  "status": 1,
  "completion_rate": "57%",
  "checkin_count": 120,
  "total_possible": 210,
  "creator": { "id": 1, "nickname": "李老师", "avatar": "..." },
  "created_at": "2026-03-18T10:00:00Z"
}
```

- 不返回 `points`。

---

## DELETE `/admin/tasks/:id`

> 删除任务（管理员强制删除；级联删除提交/回应/读标记/状态）

```json
{}
```

---

## GET `/admin/tasks/stats`

> 任务统计

```json
// Query: ?start_date=2026-03-01&end_date=2026-03-31

// Response
{
  "period": { "start": "2026-03-01", "end": "2026-03-31" },
  "total_tasks": 25,
  "active_tasks": 8,
  "completed_tasks": 17,
  "total_checkins": 3200,
  "avg_completion_rate": "62%",
  "by_class": [
    { "class_name": "小一班", "tasks": 5, "avg_rate": "68%" }
  ]
}
```

- `total_points_awarded` 已移除。
- `[C]` 后续新增参与率、回应率、`pending_over_48h`、反馈率、按活动统计。

---

## `[C]` 活动与班级罐端点（未实现）

- `POST/PUT/DELETE /admin/task-activities`
- `GET/PUT /admin/classes/:id/star-jar`

这些属于 C 切片，A 不实现。
