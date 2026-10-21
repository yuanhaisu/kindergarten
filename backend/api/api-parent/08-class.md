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

> 班级日历（家长可见视图）

```json
// Query: ?student_id=1&start=2026-03-01&end=2026-03-31

// Response
{
  "list": [
    {
      "id": 1,
      "title": "春季出游活动",
      "description": "请各位家长提醒孩子准备出游物品，明天早上8点在幼儿园门口集合",
      "event_type": "activity",
      "event_type_label": "活动",
      "event_date": "2026-03-25",
      "start_time": "09:00",
      "end_time": "15:00",
      "location": "城市公园",
      "is_all_day": false,
      "color": "#10B981",
      "change_summary": null
    },
    {
      "id": 2,
      "title": "体检",
      "description": "全员体检，请当天按时到园",
      "event_type": "checkup",
      "event_type_label": "体检",
      "event_date": "2026-03-28",
      "start_time": "09:00",
      "end_time": "11:00",
      "location": "幼儿园医务室",
      "is_all_day": false,
      "color": "#EF4444",
      "change_summary": "时间 09:00 → 08:30"
    }
  ],
  "total": 5
}
```

> 行为说明：
> - 仅返回**已发布**（`status = published`）且受众命中的事件：本班事件 + 园级事件；草稿一律不出现。
> - `change_summary` 仅在事件处于**变更提示窗口内**（近 7 天内被修订、且尚未发生）时非空，家长端据此在日历顶部显示变更提示；其余情况为 `null`。
> - `event_type_label` 与 `color` 由服务端按园所字典 `calendar_event_type` 解析下发（家长端无 dicts 接口）。
> - 内部字段（`kindergarten_id`、`creator_id`、`is_published`、`notice_id` 等）不出现在响应中。
