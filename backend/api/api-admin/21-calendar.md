# 21. 园级日历与课表

> 对应前端页面：`admin/pages/calendar/index.vue`（园级日历管理）、`admin/pages/timetable/index.vue`（课表管理）
>
> 基础路径：`/api/v1/admin`，需 `Authorization: Bearer <token>`，角色 `admin`。
>
> 权限码：`calendar:view`（查看园级日历）、`calendar:manage`（管理园级事件）、`timetable:view`（查看课表）、`timetable:manage`（课表与同步）。**全部接口须登记进管理端权限登记表**（漏登记会使服务启动失败）。
>
> 状态：🔲 待实现（变更 `align-class-calendar`）。
>
> 类型字典：日历事件类型为园所字典 `calendar_event_type`（内置 5 类，由本变更 seed）；其字典项维护走既有 `/admin/dicts/*` 接口，事件响应的 `event_type_label`/`color` 由服务端按该字典解析。

## 接口清单（11 个）

园级日历事件：

- `GET /admin/calendar/events`
- `POST /admin/calendar/events`
- `PUT /admin/calendar/events/:id`
- `DELETE /admin/calendar/events/:id`

课表：

- `GET /admin/timetable/templates/:grade`
- `PUT /admin/timetable/templates/:grade`
- `GET /admin/timetable/classes/:class_id`
- `PUT /admin/timetable/classes/:class_id`
- `GET /admin/timetable/setting`
- `PUT /admin/timetable/setting`
- `POST /admin/timetable/sync`

---

## 园级日历事件

### GET `/admin/calendar/events`

> 园级事件列表（仅 `scope = kindergarten` 的事件）

```json
// Query: ?start=2026-04-01&end=2026-04-30&status=published
// status: draft / published，缺省返回全部

// Response
{
  "list": [
    {
      "id": 3,
      "title": "清明节假期",
      "event_type": "holiday",
      "event_type_label": "假期",
      "status": "published",
      "event_date": "2026-04-04",
      "is_all_day": true,
      "color": "#F97316"
    }
  ],
  "total": 2
}
```

---

### POST `/admin/calendar/events`

> 创建园级事件（仅限园内固定角色：园长类角色、保健医走医生端）

```json
// Request
{
  "title": "清明节假期",
  "description": "4月4日-4月6日放假，4月7日正常入园",
  "event_type": "holiday",
  "event_date": "2026-04-04",
  "is_all_day": true,
  "status": "published",
  "notify": true
}

// Response
{
  "id": 3,
  "created_at": "2026-03-25T08:00:00Z"
}
```

> 行为说明：与老师端创建一致——`status` 可选（默认 `published`，草稿不通知）；`notify = true` 时生成面向**全园家长**的通知；`event_type_label` 与 `color` 由服务端按字典 `calendar_event_type` 解析。

---

### PUT `/admin/calendar/events/:id`

> 编辑园级事件

```json
// Request（与 POST 同构）

// Response
{
  "id": 3
}
```

> 修改已发布事件时可选择通知家长；服务端自动记录变更要旨（家长端显示变更提示）。

---

### DELETE `/admin/calendar/events/:id`

> 删除园级事件（软删除；生效后全端不再可见）

```json
// Response
{}
```

---

## 课表

### GET `/admin/timetable/templates/:grade`

> 年级课表模板（周模式）

```json
// :grade = 小班

// Response
{
  "grade": "小班",
  "entries": [
    { "weekday": 3, "period": 2, "title": "美术课", "location": "美术教室", "start_time": "10:00", "end_time": "10:40" },
    { "weekday": 3, "period": 3, "title": "音乐课", "location": "音乐教室", "start_time": "15:00", "end_time": "15:40" }
  ]
}
```

> `weekday`：1-7（周一 = 1）；`period`：第几节（正整数）；`start_time`/`end_time` 可选（不填则为无具体时刻的条目）。

---

### PUT `/admin/timetable/templates/:grade`

> 保存年级模板（**全量替换**该年级条目）

```json
// Request
{
  "entries": [
    { "weekday": 3, "period": 2, "title": "美术课", "location": "美术教室", "start_time": "10:00", "end_time": "10:40" }
  ]
}

// Response
{
  "grade": "小班",
  "entries": [ "..." ]
}
```

> 保存后自动触发未来滚动窗口的重算：未覆盖的未来实例按新模板更新，已发生日期与已覆盖实例保持不变。

---

### GET `/admin/timetable/classes/:class_id`

> 班级微调（覆盖年级模板的周模式条目）

```json
// Response
{
  "class_id": 1,
  "overrides": [
    { "weekday": 3, "period": 2, "action": "replace", "title": "户外活动", "location": "操场", "start_time": "10:00", "end_time": "10:40" },
    { "weekday": 5, "period": 4, "action": "remove" },
    { "weekday": 1, "period": 1, "action": "add", "title": "升旗仪式", "location": "操场" }
  ]
}
```

> `action`：`replace`（替换模板同槽位）/ `remove`（删除模板同槽位）/ `add`（新增模板没有的槽位）。

---

### PUT `/admin/timetable/classes/:class_id`

> 保存班级微调（**全量替换**该班微调项）

```json
// Request
{
  "overrides": [
    { "weekday": 3, "period": 2, "action": "replace", "title": "户外活动", "location": "操场" }
  ]
}

// Response
{
  "class_id": 1,
  "overrides": [ "..." ]
}
```

---

### GET `/admin/timetable/setting`

> 课表开关

```json
// Response
{
  "timetable_enabled": true
}
```

---

### PUT `/admin/timetable/setting`

> 更新课表开关

```json
// Request
{
  "timetable_enabled": false
}

// Response
{
  "timetable_enabled": false
}
```

> 关闭后：课表来源的日历事件不再在老师端/家长端呈现（数据保留），手动事件不受影响；开启时自动触发同步。

---

### POST `/admin/timetable/sync`

> 立即同步（排障用；与每日滚动任务等价，幂等）

```json
// Response
{
  "synced": true,
  "created": 120,
  "updated": 4,
  "deleted": 2,
  "skipped_overridden": 3
}
```
