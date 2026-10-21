# 模块 8：班级日历 API

> 对应开发计划：模块 8：班级日历
>
> 基础路径：`/api/v1/teacher`
>
> 认证：`Authorization: Bearer <token>`，用户角色为 `teacher`
>
> 以下 Response 均为 `data` 部分，省略外层 `{ "code": 0, "message": "success" }` 包装。

## 语义要点

- **受众范围**：`scope = class`（本班）或 `kindergarten`（园级）。园级事件对全园老师可见但**只读**——修改/删除返回明确错误。
- **发布状态**：`status = draft | published`。草稿仅老师端可见并带标记，家长端不可见。
- **类型字典**：`event_type` 为园所字典键（字典类型 `calendar_event_type`，内置 5 类不可删、可改名/停用）；`event_type_label` 与 `color` 由服务端按该字典解析下发，**客户端不得硬编码类型色映射**。
- **类型选择器数据来源**：`GET /teacher/dicts/calendar_event_type`（字典接口，返回类型键/名称/颜色/图标）。
- **通知**：创建或修改已发布事件时可选择是否通知家长（`notify`）；确认后经通知模块生成面向本班家长的通知，已读可查。草稿不产生通知。

## 接口清单

- `GET /teacher/calendar/events`
- `GET /teacher/calendar/events/:date`
- `GET /teacher/calendar/events/detail/:id`
- `POST /teacher/calendar/events`
- `PUT /teacher/calendar/events/:id`
- `DELETE /teacher/calendar/events/:id`
- `GET /teacher/calendar/dots`

---

### GET `/teacher/calendar/events`

> 日历事件列表（月视图/区间视图，含本班事件与园级事件）

```json
// Query: ?start=2026-03-01&end=2026-03-31

// Response
{
  "list": [
    {
      "id": 1,
      "title": "春季出游活动通知",
      "description": "请各位家长提醒孩子准备出游物品，明天早上8点在幼儿园门口集合",
      "event_type": "notice",
      "event_type_label": "通知",
      "scope": "class",
      "status": "published",
      "event_date": "2026-03-19",
      "start_time": "09:00",
      "end_time": null,
      "location": "",
      "is_all_day": false,
      "color": "#3B82F6"
    },
    {
      "id": 2,
      "title": "美术课：春天的小树",
      "description": "带领小朋友们用画笔描绘春天的景色，发挥想象力",
      "event_type": "activity",
      "event_type_label": "活动",
      "scope": "class",
      "status": "draft",
      "event_date": "2026-03-19",
      "start_time": "10:00",
      "end_time": "11:00",
      "location": "教室",
      "is_all_day": false,
      "color": "#10B981"
    },
    {
      "id": 3,
      "title": "清明节假期",
      "event_type": "holiday",
      "event_type_label": "假期",
      "scope": "kindergarten",
      "status": "published",
      "event_date": "2026-04-04",
      "is_all_day": true,
      "location": "",
      "color": "#F97316"
    }
  ],
  "total": 15
}
```

---

### GET `/teacher/calendar/events/:date`

> 某天的事件列表

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
      "scope": "class",
      "status": "published",
      "start_time": "09:00",
      "end_time": null,
      "location": "",
      "color": "#3B82F6"
    }
  ]
}
```

---

### GET `/teacher/calendar/events/detail/:id`

> 单条事件详情

```json
// Response
{
  "event": {
    "id": 2,
    "title": "美术课：春天的小树",
    "description": "带领小朋友们用画笔描绘春天的景色，发挥想象力",
    "event_type": "activity",
    "event_type_label": "活动",
    "scope": "class",
    "status": "published",
    "event_date": "2026-03-19",
    "start_time": "10:00",
    "end_time": "11:00",
    "location": "教室",
    "is_all_day": false,
    "color": "#10B981"
  }
}
```

---

### POST `/teacher/calendar/events`

> 创建日历事件（仅班级级；园级事件由管理端/医生端发布）

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
  "status": "published",
  "notify": true
}

// Response
{
  "id": 2,
  "created_at": "2026-03-18T16:00:00Z"
}
```

> 行为说明：
> - `status` 可选，默认 `published`；`status = draft` 时 `notify` 被忽略（草稿不产生通知）。
> - `notify = true` 时生成面向本班家长的通知；`notify = false` 时事件照常按发布状态生效，但无通知。

---

### PUT `/teacher/calendar/events/:id`

> 编辑日历事件（园级事件不可编辑，返回明确错误）

```json
// Request（与 POST 同构）
{
  "title": "美术课：春天的小树（调整）",
  "description": "带领小朋友们用画笔描绘春天的景色",
  "event_type": "activity",
  "event_date": "2026-03-19",
  "start_time": "10:30",
  "end_time": "11:30",
  "location": "美术教室",
  "is_all_day": false,
  "status": "published",
  "notify": true
}

// Response
{
  "id": 2
}
```

> 行为说明：
> - 修改**已发布**事件时，`notify = true` 生成一条变更通知；修改草稿不产生通知。
> - 服务端自动记录变更要旨（家长端据此显示变更提示），客户端**不回传**该字段。

---

### DELETE `/teacher/calendar/events/:id`

> 删除日历事件（软删除；园级事件不可删除）

```json
// Response
{}
```

---

### GET `/teacher/calendar/dots`

> 月视图日期标记点：日期 → 当天事件的类型键集合（去重），客户端按字典颜色渲染标记点

```json
// Query: ?start=2026-03-01&end=2026-03-31

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

---

### 医生端（老师端小程序内）：`/doctor/calendar/*`

> 保健医发布的**园级事件**（体检等）。认证同老师端 `<token>`（角色 `doctor`），不经管理端权限码。
> 请求/响应与上方事件的字段结构同构；作用域固定为园级（全园可见，`class_id` 为 NULL）。

- `GET /doctor/calendar/events`（Query: `?start=&end=&status=`）
- `POST /doctor/calendar/events`（与老师端创建同构：`status` 默认 `published`；`notify = true` 时面向**全园家长**生成通知）
- `PUT /doctor/calendar/events/:id`
- `DELETE /doctor/calendar/events/:id`
