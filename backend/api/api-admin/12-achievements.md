# 12. 成就管理

> 对应前端页面：`admin/pages/achievements/index.vue`
>
> 基础路径：`/api/v1/admin`，需 `Authorization: Bearer <token>`，角色 `admin`。
>
> 状态：✅ 4 个接口已实现（`achievements.kindergarten_id` 隔离；`earned_count` 为园内学生获得数）。

---

## GET `/admin/achievements`

> 成就勋章列表（对应前端 `getAchievements`）

```json
// Query: ?page=1&page_size=20

// Response
{
  "list": [
    {
      "id": 1,
      "name": "全勤之星",
      "description": "连续一个月全勤",
      "icon_url": "https://cdn.example.com/badges/attendance.png",
      "category": "attendance",
      "category_label": "出勤",
      "condition": "monthly_full_attendance",
      "earned_count": 45,
      "status": 1,
      "created_at": "2025-09-01T08:00:00Z"
    }
  ],
  "total": 20
}
```

---

## POST `/admin/achievements`

> 创建成就（对应前端 `createAchievement`）

```json
// Request
{
  "name": "手工小达人",
  "description": "参与20次手工制作活动",
  "icon_url": "https://cdn.example.com/badges/craft.png",
  "category": "growth",
  "condition": "craft_activities_20",
  "target_value": 20
}

// Response
{
  "id": 21,
  "created_at": "2026-03-19T10:00:00Z"
}
```

---

## PUT `/admin/achievements/:id`

> 编辑成就（对应前端 `updateAchievement`）

```json
// Request
{
  "name": "手工小达人",
  "description": "参与15次手工制作活动（降低门槛）",
  "target_value": 15
}

// Response
{
  "id": 21,
  "updated_at": "2026-03-19T10:30:00Z"
}
```

---

## DELETE `/admin/achievements/:id`

> 删除成就（对应前端 `deleteAchievement`）

```json
// Response
{}
```
