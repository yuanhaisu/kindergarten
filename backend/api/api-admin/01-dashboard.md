# 01. 数据看板 (Dashboard)

> 对应前端页面：`admin/pages/index.vue`
>
> 基础路径：`/api/v1/admin`，需 `Authorization: Bearer <token>`，角色 `admin`。
>
> 状态：🚫 4 个接口全部待后端实现。

---

## GET `/admin/dashboard/overview`

> 首页数据概览（对应前端 `getDashboardOverview`，用于 4 个统计卡：在园 / 请假 / 出勤率 / 已接走）

```json
// Response
{
  "kindergarten": {
    "id": 1,
    "name": "阳光幼儿园"
  },
  "realtime": {
    "present_count": 280,
    "leave_count": 12,
    "total_student_count": 300,
    "attendance_rate": "93.3%",
    "picked_up_count": 45,
    "not_picked_up_count": 235
  },
  "today": {
    "new_notices": 5,
    "pending_leaves": 3,
    "growth_logs": 12,
    "new_tasks": 2
  },
  "this_month": {
    "avg_attendance_rate": "95.2%",
    "total_growth_logs": 320,
    "total_points_awarded": 15800,
    "new_students": 3,
    "new_parents": 5
  }
}
```

---

## GET `/admin/dashboard/attendance-trend`

> 出勤趋势图（对应前端 `getAttendanceTrend`，近 30 日出勤趋势图）

```json
// Query: ?period=30d
// period: 7d / 30d / 12m

// Response
{
  "period": "30d",
  "data": [
    { "date": "2026-02-18", "present": 278, "leave": 15, "absent": 7, "rate": "92.7%" },
    { "date": "2026-02-19", "present": 285, "leave": 10, "absent": 5, "rate": "95.0%" },
    { "date": "2026-03-19", "present": 280, "leave": 12, "absent": 8, "rate": "93.3%" }
  ]
}
```

---

## GET `/admin/dashboard/class-ranking`

> 班级出勤率排行（对应前端 `getClassRanking`）

```json
// Response
{
  "list": [
    { "class_id": 1, "class_name": "小一班", "present": 28, "total": 30, "rate": "93.3%" },
    { "class_id": 2, "class_name": "小二班", "present": 29, "total": 30, "rate": "96.7%" },
    { "class_id": 3, "class_name": "中一班", "present": 32, "total": 35, "rate": "91.4%" }
  ]
}
```

---

## GET `/admin/dashboard/growth-stats`

> 成长记录统计（对应前端 `getGrowthStats`，与 [`09-growth.md`](./09-growth.md) 的 `/admin/growth/stats` 可共享实现）

```json
// Query: ?period=30d

// Response
{
  "period": "30d",
  "total_logs": 320,
  "total_media": 2800,
  "total_likes": 4500,
  "total_comments": 1200,
  "daily_avg_logs": 10.7,
  "top_classes": [
    { "class_name": "小一班", "logs_count": 68, "teacher_name": "李老师" },
    { "class_name": "中二班", "logs_count": 55, "teacher_name": "王老师" }
  ]
}
```
