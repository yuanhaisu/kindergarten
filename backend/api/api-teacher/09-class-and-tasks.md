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
- `GET /teacher/class/present-students`
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

> 班级学生列表（接送向；成长记录创建页的候选学生也取自本接口）
>
> 候选范围：**本班在读**（`status=1`）学生，按学生 ID 升序，一次全量返回（无分页）。

```json
// Query: ?status=            // 可选，接送状态过滤（arrived/picked_up/not_picked_up/not_arrived）；不传为全部

// Response
{
  "list": [
    {
      "student_id": 1,
      "name": "王小明",
      "avatar": "https://cdn.example.com/avatar/s1.jpg",
      "authorized_persons": "爷爷、奶奶",
      "arrive_time": "08:15",
      "pickup_status": "not_picked_up",
      "is_overdue": false,
      "pickup_deadline": "16:00",
      "last_remark": "",
      "is_authorized_override": false
    }
  ],
  "total": 30
}
```

> 说明：响应为接送页专用形状（字段以 `student_id` 而非 `id` 为主键）；成长记录创建页只消费 `student_id` / `name` / `avatar` 三个字段。

---

### GET `/teacher/class/present-students`

> 指定日期的**到场学生名单**（照片点名「全班」一键的名单来源）
>
> 口径：本班在读 ∧ **未被明确排除**（无已批准请假覆盖当日，且当日考勤非缺勤/请假）。
> **无请假/考勤数据时保守视为到场**——家长端「漏标」（看不到孩子的照片）的成本远高于「误标」。
> 阈值与口径由服务端判定，客户端不得重复实现。

```json
// Query: ?date=2026-10-16    // 可选，缺省为今天（Asia/Shanghai）
// 格式错误或无法解析时按今天处理

// Response
{
  "list": [1, 2, 3, 5],
  "total": 4
}
```

> `list` 为学生 ID 数组（升序不保证，客户端按 ID 匹配）；只含**本班**在读学生。

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

> A 切片：班级/个体任务、提交、老师单条/周期回应、无积分。公开审核、补充回应、分享码属于 B 切片。
>
> 老师端任务操作限定当前班级。

### GET `/teacher/tasks`

```json
// Query: ?page=1&page_size=20

// Response
{
  "list": [
    {
      "id": 1,
      "title": "每日阅读10分钟",
      "description": "每天陪孩子阅读10分钟绘本",
      "intro": "任务详细介绍",
      "learning_link": "延续本周班级的春天主题",
      "target_student_id": 0,
      "frequency": "daily",
      "start_date": "2026-03-19",
      "end_date": "2026-03-25",
      "status": 1,
      "allow_public": false,
      "recognition_ritual": "sticker",
      "star_scope": "task",
      "star_goal": 20,
      "pending_responses": 3,
      "created_at": "2026-03-18T10:00:00Z"
    }
  ],
  "total": 8,
  "page": 1,
  "page_size": 20
}
```

- `pending_responses` 是当前任务下 `response_id=0` 的提交数。
- `points` 已移除。

### GET `/teacher/tasks/:id`

> 任务详情（编辑页按 id 拉取，不再依赖列表分页）

```json
{
  "id": 1,
  "title": "每日阅读10分钟",
  "description": "每天陪孩子阅读10分钟绘本",
  "intro": "任务详细介绍",
  "learning_link": "延续本周班级的春天主题",
  "target_student_id": 0,
  "frequency": "daily",
  "start_date": "2026-03-19",
  "end_date": "2026-03-25",
  "allow_public": false,
  "recognition_ritual": "sticker",
  "star_scope": "task",
  "star_goal": 20,
  "status": 1,
  "created_at": "2026-03-18T10:00:00Z"
}
```

### POST `/teacher/tasks`

```json
// Request
{
  "title": "每日阅读10分钟",
  "description": "每天陪孩子阅读10分钟绘本",
  "intro": "任务详细介绍",
  "requirements": ["每天阅读10分钟", "上传一张阅读照片"],
  "target_student_id": 0,
  "learning_link": "延续本周班级的春天主题",
  "frequency": "daily",
  "start_date": "2026-03-19",
  "end_date": "2026-03-25",
  "allow_public": false,
  "recognition_ritual": "sticker",
  "star_scope": "task",
  "star_goal": 20
}

// Response（HTTP 201）
{
  "id": 1,
  "created_at": "2026-03-18T10:00:00Z",
  "notified_count": 0,
  "notify_failed_count": 0
}
```

- `learning_link` 必填。
- `target_student_id > 0` 时必须是当前班级在读学生；个体任务 `allow_public` 强制 false。
- `star_scope=class` 时必须已有班级罐目标，否则拒绝。
- 不接收 `points`。

### PUT `/teacher/tasks/:id`

```json
// Request：字段同创建；有提交后只允许 title/description/learning_link/allow_public
{
  "title": "每日阅读15分钟",
  "description": "时间延长",
  "learning_link": "延续本周班级的春天主题",
  "allow_public": false
}
```

- 有提交后修改 `frequency`、起止日期、`target_student_id`、`star_scope`、`star_goal` 返回参数错误。
- `allow_public=false` 时批量撤销公开（B 切片字段，A 先保留）。

### DELETE `/teacher/tasks/:id`

```json
{}
```

### GET `/teacher/tasks/:id/stats`

```json
{
  "task_id": 1,
  "total_students": 30,
  "submission_count": 21,
  "responded_count": 12,
  "pending_count": 9
}
```

- 不再返回 `checkin_count` / `not_checkin_count` / `total_points_awarded`。

### GET `/teacher/tasks/:id/checkins`

```json
// Query: ?status=pending|responded
{
  "list": [
    {
      "id": 1,
      "student_id": 1,
      "student": { "id": 1, "name": "王小明", "avatar": "..." },
      "date": "2026-03-19",
      "content": "今天和爸爸一起读了《小王子》",
      "media_file_id": 1001,
      "media_url": "https://cdn.example.com/task/checkin/1.jpg",
      "is_public_requested": false,
      "public_scope_confirmed": false,
      "response_id": 0,
      "created_at": "2026-03-19T20:30:00Z"
    }
  ],
  "total": 1
}
```

- `status=pending` 由后端按 `created_at ASC` 排序。
- 不返回 `points`。

### POST `/teacher/tasks/:id/checkins/:checkin_id/response`

```json
// Request
{
  "ritual": "sticker",
  "sticker_key": "star",
  "content": "今天读得真认真",
  "media_file_id": 0
}

// Response（HTTP 201）
{
  "id": 10,
  "created_at": "2026-03-19T20:35:00Z"
}
```

- `ritual`：`sticker` / `voice` / `class_star`。
- 回应后提交 `response_id` 指向该回应。

### POST `/teacher/tasks/:id/batch-response`

```json
// Request：周期汇总回应
{
  "ritual": "class_star",
  "sticker_key": "",
  "content": "本周大家都很棒",
  "media_file_id": 0
}

// Response
{
  "id": 11,
  "responded_count": 20,
  "failed_ids": []
}
```

- 对当前任务全部 `response_id=0` 的提交创建一条 `period` 回应。
- 部分失败返回 `failed_ids`。

## 说明

- 公开墙审核、补充回应、分享码、未读属于 B 切片。
- 班级罐配置端点属于 C 切片；A 只做创建时只读校验。
- 任务不发放积分。
