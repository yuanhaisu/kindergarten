# 11. 积分管理

> 对应前端页面：`admin/pages/points/index.vue`
>
> 基础路径：`/api/v1/admin`，需 `Authorization: Bearer <token>`，角色 `admin`。
>
> 状态：✅ 5 个接口已实现（后端 `PointRepo` / `PointService`；规则读写 `kindergarten_app_settings.extra_config.points`）。

---

## GET `/admin/points/overview`

> 积分系统概览（对应前端 `getPointsOverview`）

```json
// Response
{
  "total_points_in_circulation": 385000,
  "total_points_awarded_this_month": 15800,
  "total_points_spent_this_month": 2300,
  "top_earners": [
    { "student": { "id": 2, "name": "李小花", "class_name": "小一班" }, "points": 1580 },
    { "student": { "id": 1, "name": "王小明", "class_name": "小一班" }, "points": 1245 }
  ]
}
```

---

## GET `/admin/points/records`

> 积分流水（对应前端 `getPointRecords`）

```json
// Query: ?student_id=&class_id=&type=earn&page=1&page_size=20

// Response
{
  "list": [
    {
      "id": 1,
      "student": { "id": 1, "name": "王小明", "class_name": "小一班" },
      "type": "earn",
      "source": "task_checkin",
      "source_label": "任务打卡",
      "amount": 5,
      "description": "完成任务【每日阅读10分钟】",
      "operator": null,
      "created_at": "2026-03-19T20:30:00Z"
    }
  ],
  "total": 38000,
  "page": 1,
  "page_size": 20
}
```

---

## POST `/admin/points/adjust`

> 手动调整积分（对应前端 `adjustPoints`）

```json
// Request
{
  "student_id": 1,
  "type": "earn",               // earn=增加 / spend=扣减
  "amount": 50,
  "reason": "参加园庆表演奖励"
}

// Response
{
  "id": 38001,
  "balance": 1295,
  "created_at": "2026-03-19T10:00:00Z"
}
```

---

## GET `/admin/points/rules`

> 积分规则配置（对应前端 `getPointRules`）

```json
// Response
{
  "rules": [
    { "id": 1, "source": "task_checkin", "label": "任务打卡", "points": 5, "editable": true },
    { "id": 2, "source": "full_attendance_week", "label": "周全勤", "points": 10, "editable": true },
    { "id": 3, "source": "full_attendance_month", "label": "月全勤", "points": 50, "editable": true },
    { "id": 4, "source": "growth_featured", "label": "成长日志被推荐", "points": 20, "editable": true }
  ]
}
```

---

## PUT `/admin/points/rules`

> 更新积分规则（对应前端 `updatePointRules`）

```json
// Request
{
  "rules": [
    { "id": 1, "points": 10 },
    { "id": 2, "points": 15 }
  ]
}

// Response
{
  "updated_count": 2,
  "updated_at": "2026-03-19T10:00:00Z"
}
```
