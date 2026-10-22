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
- `POST /teacher/tasks/:id/checkins/:checkin_id/response`
- `POST /teacher/tasks/:id/checkins/:checkin_id/supplement`
- `POST /teacher/tasks/:id/batch-response`

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

> A 切片：班级/个体任务、提交、老师单条/周期回应、无积分。B 切片已补：汇总后个别补充回应、公开审核。分享码仍待补。
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
      "pending_period": 3,
      "period_start": "2026-03-25",
      "period_end": "2026-03-25",
      "created_at": "2026-03-18T10:00:00Z"
    }
  ],
  "total": 8,
  "page": 1,
  "page_size": 20
}
```

- `pending_responses` 是当前任务下全部 `response_id=0` 的提交数（含旧周期遗留）。
- `pending_period` 是**本周期内**待回应的条数，与批量回应接口的筛选条件逐字一致
  （`task_id=? AND response_id=0 AND date BETWEEN period_start AND period_end`）：
  每日/每周的班级任务按区间统计；一次性、个体任务以及未知 `frequency` 不支持汇总回应，
  恒等于 `pending_responses`（这些任务调用 `POST /teacher/tasks/:id/batch-response` 会得到 422，
  所以该字段不能当作「可以一键关闭」的承诺）。
- `period_start` / `period_end` 与 `GET /teacher/tasks/:id` 同义（`taskPeriodRange`）。
  列表页只在两者不等时额外提示「本周期 N」，避免与详情页的关闭条数对不上。
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
  "period_start": "2026-03-21",
  "period_end": "2026-03-21",
  "created_at": "2026-03-18T10:00:00Z"
}
```

- `period_start` / `period_end`：当前汇总回应周期（每日=当天；每周=滚动 7 天；任务结束后取最后一天/最后一周）。前端用它区分「本周期待回应」与旧周期遗留。

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
      "public_approved_at": null,
      "public_revoked_at": null,
      "public_status": "none",
      "public_visible": false,
      "response_id": 0,
      "created_at": "2026-03-19T20:30:00Z"
    }
  ],
  "total": 1
}
```

- `status=pending` 由后端按 `created_at ASC` 排序。
- `public_status`：`none` / `pending` / `approved` / `revoked`，供审核徽标与筛选使用。
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
- 仪式载体必填：`sticker` 必须带 `sticker_key`；`voice` 必须带 `media_file_id` **且文件类型为音频**（否则 422「贴纸需要选择具体贴纸，语音需要先录制并上传」/「语音回应必须使用音频文件」）；`class_star` 不需要载体。批量回应与补充回应同口径。
- 提交不存在 → 422；提交已被其他回应关闭（并发/重复点击）→ **409**「该提交已被其他回应关闭，请刷新后重试」；行锁等待超时/死锁 → **503**「操作冲突，请稍后重试」。

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

- 只对**当前周期**内（每日=当天；每周=滚动 7 天；任务已结束时取最后一天/最后一周）仍为 `response_id=0` 的提交创建一条 `period` 回应，并写入 `period_start` / `period_end`；旧周期的待回应提交需逐条回应。
- 只有**每日/每周的班级任务**（`target_student_id=0` 且 `frequency` 为 `daily|weekly`）支持汇总回应；个体任务、一次性任务及其它频率返回 422 `一次性任务与个体任务需逐条回应`。
- 当前周期没有待回应提交时返回 **200** 与 `{"id": 0, "responded_count": 0, "failed_ids": []}`（空操作，不创建回应）；创建成功返回 201。
- 有候选提交但一条都没写成功时返回 500，提交保持待回应；部分失败返回 `failed_ids` 且 `responded_count` 只计成功条数。
- 仪式载体校验与单条回应一致（`sticker` 需 `sticker_key`、`voice` 需音频媒体）。

### POST `/teacher/tasks/:id/checkins/:checkin_id/supplement`

> 汇总回应之后，给某一条已回应的提交追加个别补充回应

```json
// Request
{
  "ritual": "sticker",
  "sticker_key": "flower",
  "content": "单独夸一句",
  "media_file_id": 0
}

// Response（HTTP 201）
{
  "id": 12,
  "created_at": "2026-03-19T20:40:00Z"
}
```

- 只能追加到**已有主回应**的提交上；待回应的提交调用本接口 → 422「该提交尚未回应，请先完成主回应」。
- 补充回应是一条独立的 `TaskResponse`（`response_type=single`）+ 一行 `task_response_supplements` 关系；**不改变**提交的 `response_id`，提交仍保持「已回应」，家长端按「主回应 + 补充回应」展示。
- `ritual` 只接受 `sticker` / `voice`；`class_star` → 422「补充回应仅支持贴纸或语音」（一条提交只有一个主回应能计入星星罐）。未传时默认 `sticker`，**不继承**任务默认仪式。
- 同一条提交可以追加多条补充；`media_file_id` 校验与主回应一致（上传者本人、`biz_type=task`、image/audio）。
- 仪式载体必填：`sticker` 必须带 `sticker_key`、`voice` 必须带 `media_file_id` 且文件类型为音频，否则 422「贴纸需要选择具体贴纸，语音需要先录制并上传」/「语音回应必须使用音频文件」。主回应与批量回应同口径。
- 提交不属于该任务 → 422；任务不属于当前老师班级 → 404；行锁等待超时/死锁 → 503。

### PUT `/teacher/tasks/:id/checkins/:checkin_id/public`

> 老师审核家长提交的公开申请（批准 / 拒绝或撤回）

```json
// Request：批准
{
  "approve": true,
  "scope_confirmed": true
}

// Response（HTTP 200）
{
  "checkin_id": 1,
  "is_public_requested": true,
  "public_scope_confirmed": true,
  "public_approved_at": "2026-03-19T21:00:00Z",
  "public_approved_by": 11,
  "public_revoked_at": null,
  "public_status": "approved",
  "public_visible": true
}
```

```json
// Request：拒绝 / 撤回
{ "approve": false }
```

- **批准**时 `scope_confirmed` 必须为 `true`（老师确认「内容中只有提交者孩子」）；缺失或为 `false` → **422**「批准公开前必须确认内容中只有提交者孩子」。图片/语音/文字均适用，含其他孩子的内容不得批准。
- 批准写 `public_scope_confirmed=true`、`public_approved_at=now`、`public_approved_by=当前老师`、`public_revoked_at=null`。
- **拒绝/撤回**（`approve=false`）清空 `public_scope_confirmed`、`public_approved_at`、`public_approved_by`，写 `public_revoked_at=now`，`is_public_requested=false`；对从未申请过的提交调用是幂等空操作。
- 只有处于**待审核**（家长已申请且未撤回）的提交可以批准；未申请 / 已取消 / 已撤回 → **409**「该提交当前不在待审核状态，请让家长重新申请」。已批准时重复批准是幂等空操作。
- 前置：**批准**要求任务 `allow_public=true` 且不是个体任务，并且提交者孩子**当前仍在该任务班级**（转班后拒绝，否则只会造出一个永远不上墙的「已公开」状态），否则 **422**。**拒绝/撤回不受此限**——任务被关掉公开开关、改成个体任务或孩子已转班后，老师仍必须能把残留的公开状态清干净，此时 `approve=false` 照常返回 200。
- 任务不属于当前老师班级 → **404**；提交不属于该任务 → **422**；行锁等待超时/死锁 → **503**。
- 再次申请必须重新批准、重新勾确认：取消/撤回会把 `public_scope_confirmed` 清成 `false`，不会沿用。

## 说明

- 公开审核见上方 `PUT /teacher/tasks/:id/checkins/:checkin_id/public`；公开墙本身只在家长端。
- 分享码、未读属于 B 切片剩余部分。
- 班级罐配置端点属于 C 切片；A 只做创建时只读校验。
- 任务不发放积分。
