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

- `GET /teacher/class/list`
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
- `GET /teacher/tasks/:id/feedback-summary`
- `GET /teacher/tasks/:id/checkins`
- `POST /teacher/tasks/:id/checkins/:checkin_id/response`
- `POST /teacher/tasks/:id/checkins/:checkin_id/supplement`
- `POST /teacher/tasks/:id/batch-response`
- `PUT /teacher/tasks/:id/checkins/:checkin_id/public`
- `GET /teacher/tasks/:id/share-card`
- `GET /teacher/class/star-jar`
- `PUT /teacher/class/star-jar`

---

## 7. 班级学生

### GET `/teacher/class/list`

> 当前老师在该园所**任教**的班级列表（`class_teachers.status=1`，按任教关系 id 升序）。
>
> 老师端「发布通知」的接收班级选择器在用。无 query 参数。

```json
// Response（data 为数组，无分页包络）
[
  {
    "class_id": 1,
    "class_name": "小一班",
    "title": "head",
    "title_label": "主班老师",
    "parent_count": 28
  }
]
```

- `title` / `title_label`：当前老师在该班的职务码与中文标签（`head`/`assistant`/`nursery`，其它按「老师」）。
- `parent_count`：该班**潜在触达家长数**——本班**在读学生**（`students.status=1` 且未删除）
  绑定的**去重家长用户数**（`COUNT(DISTINCT student_parents.user_id)`，`student_parents` 未删除）。
  同一家长有两个孩子同班只计 1；已毕业/退学/待入学学生与其家长不计入。
- **与通知实际接收人不是同一口径**：本字段只表示班级潜在触达范围，不随某条通知的
  target 选择、已读/确认变化。某条通知的实际接收人数以通知详情为准。
- 没有在读学生或没有绑定家长 → `0`（不是 `null`）。
- 计数查询失败时返回 `null`（不是 `0`），且有服务端日志；客户端应回退为不显示人数，
  不能把 `null` 当成「0 位家长」。
- 列表级一次 `GROUP BY class_id` 聚合，不逐班查询。

---

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
// Query: ?page=1&page_size=20&status=active|ended   // status 可选（大小写不敏感）；缺省/其它值 = 全部

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
      "star_count": 12,
      "participant_count": 18,
      "pending_responses": 3,
      "pending_period": 3,
      "period_start": "2026-03-25",
      "period_end": "2026-03-25",
      "created_at": "2026-03-18T10:00:00Z"
    }
  ],
  "stats": { "all": 8, "active": 5, "ended": 3 },
  "total": 8,
  "page": 1,
  "page_size": 20
}
```

- `status`（query，可选）：任务状态视图筛选，**在 SQL 中过滤**，分页语义不变（`total` 随筛选结果）。
  - `active` = `status=1` 且未过 `end_date`（`end_date >= 今天`，Asia/Shanghai）；
  - `ended` = 其余全部（`status=0` 或已过 `end_date`）；
  - 缺省及其它值 = 全部（`active` / `ended` 大小写不敏感、首尾空白忽略）。按实时日期判定，不只看 `status`：后台把任务标成进行中但日期已过时，不算「进行中」。
- `stats`：当前老师**当前班级全部任务**的计数，**不受 `status` 筛选与分页影响**（页签计数必须稳定）：
  `all` = 全部任务数，`active` / `ended` 口径同上，三者满足 `all = active + ended`。一次聚合查询。
- `participant_count`：该任务**已提交过的去重学生数**（`COUNT(DISTINCT student_id)`，软删提交不计）。
  与详情 stats 的 `submission_count`（打卡行数）不同：同一学生多次打卡只算 1 人。
  列表级一次 `GROUP BY task_id` 聚合，不逐任务查询；查询失败时整个列表失败（不返回 `null` 或 0）。
- `pending_responses` 是当前任务下全部 `response_id=0` 的提交数（含旧周期遗留）。
- `pending_period` 是**本周期内**待回应的条数，与批量回应接口的筛选条件逐字一致
  （`task_id=? AND response_id=0 AND date BETWEEN period_start AND period_end`）：
  每日/每周的班级任务按区间统计；一次性、个体任务以及未知 `frequency` 不支持汇总回应，
  恒等于 `pending_responses`（这些任务调用 `POST /teacher/tasks/:id/batch-response` 会得到 422，
  所以该字段不能当作「可以一键关闭」的承诺）。
- `period_start` / `period_end` 与 `GET /teacher/tasks/:id` 同义（`taskPeriodRange`）。
  列表页只在两者不等时额外提示「本周期 N」，避免与详情页的关闭条数对不上。
- `star_count`：**本任务**已集星数 = 主回应 `ritual=class_star` 且未删除的提交数。
  `period` 回应覆盖 N 条提交就计 N 星；补充回应不改变 `checkin.response_id`，不计入。
  `StarScope=class` 时这个字段仍只表示本任务的贡献，班级罐总进度（含学期目标）由 C 切片端点提供，
  不要拿它冒充班级罐进度。
  **星数查询失败时列表与详情都返回 `null`**（不是 0）：星数是装饰性聚合，不能让任务列表/详情整体不可用；
  客户端应隐藏进度而不是显示 0。`StarScope=class` 的统计**不做学期过滤**，只是本任务全生命周期的贡献，
  C 切片计算班级罐时不得直接累加 `star_count`（需按学期重新统计）。
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
  "star_count": 12,
  "status": 1,
  "period_start": "2026-03-21",
  "period_end": "2026-03-21",
  "created_at": "2026-03-18T10:00:00Z"
}
```

- `period_start` / `period_end`：当前汇总回应周期（每日=当天；每周=滚动 7 天；任务结束后取最后一天/最后一周）。前端用它区分「本周期待回应」与旧周期遗留。
- `star_count`：口径同列表；星数查询失败时返回 `null`（不阻塞详情），客户端应隐藏「已集 N 星」。

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
- `notified_count` / `notify_failed_count`：发布站内消息的投递结果，**按家长用户去重**。班级任务覆盖该班全部家庭（同一家长有两个孩子只计一次，消息 `action_path` 固定取该家长在该班 id 最小的孩子）；个体任务只通知目标孩子的家庭。文案含任务标题；通知失败不回滚任务，只计数并写日志。

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
- `star_goal` 在创建与编辑两条路径都必须 ≥0（负数返回参数错误）；`recognition_ritual=class_star` 且 `star_scope=task` 时还必须 >0。
- **编辑时可选字段缺省 = 保留原值**：`star_goal` / `allow_public` / `description` 不传不会清零目标、不会关闭公开、不会清空说明（与 `api-admin/10-tasks.md` 同口径）。因此上面的最小编辑示例可以直接用。
- `star_scope` 只接受 `task` / `class`，`recognition_ritual` 只接受 `sticker` / `voice` / `class_star`（首尾空白会被 trim）；非法值返回参数错误，不会落库成非规范值。
- `allow_public=false` 时批量撤销公开（B 切片字段，A 先保留）。
- **活动任务（`activity_id` 非空）只允许 `title` / `description` / `learning_link` / `allow_public`**：
  频率、起止日期、`target_student_id`、`recognition_ritual`、`star_scope`、`star_goal`、`status`
  任一变化 → **422**「活动任务只能修改标题、说明、学习关联与公开开关，请让管理员整组编辑」。
  活动任务的「一个班级一条、共享 `activity_id`」只能由管理端整组维护（design D12）。

### DELETE `/teacher/tasks/:id`

```json
{}
```

- **活动任务不能单条删除** → **422**「活动任务不能单独删除，请让管理员删除整个活动」
  （单删会让活动分组与统计聚合悬空）。独立任务照常级联删除。

### GET `/teacher/tasks/:id/stats`

```json
{
  "task_id": 1,
  "total_students": 30,
  "submission_count": 21,
  "participant_count": 18,
  "responded_count": 12,
  "pending_count": 9
}
```

- `submission_count`：打卡**行数**（同一孩子跨天多次提交会多次计），用于待回应/回应进度。
- `participant_count`：**已提交过的去重学生数**（`COUNT(DISTINCT student_id)`），用于「已参与 N/M」。
  与 `submission_count` 语义不同：同一学生多次打卡只算 1 人，客户端不要把行数当人数。
- `total_students`：当前班**在读**学生数（`students.status=1` 且未删除）。
  **查询失败返回 500，不得静默返回 0**——那会把整班显示成「没有孩子」。
- 计数限定当前老师当前班级；任务不属于当前班级/园所 → 404。
- 不再返回 `checkin_count` / `not_checkin_count` / `total_points_awarded`。

### GET `/teacher/tasks/:id/feedback-summary`

老师端反馈汇总。口径与管理端 `GET /admin/tasks/stats` 的
`feedback_coverage` / `child_happy_rate` / `ack_rate` **完全一致**，不是另一套公式：

- 分母 = 被本任务提交引用的**主回应**所承诺的义务家长数之和
  （`task_responses.notify_target_count`：先写义务再投递，含通知失败）；
- 分子 = 去重后的 `(回应, 家长)` 反馈对；补充回应不被任何提交引用，其反馈不计入；
- `notify_target_count` 为 NULL（未知义务）的回应整条排除，并在
  `notify_baseline_missing` 计数；不能当 0，否则会把整体覆盖率拖低；
- 比率是 `"NN%"` 且封顶 100%；分母为 0 时是 `null`，不是 `"0%"`。

```json
{
  "task_id": 1,
  "feedback_coverage": "50%",
  "child_happy_rate": "50%",
  "ack_rate": "0%",
  "feedback_counts": { "pairs": 1, "child_happy": 1, "received": 0, "thanks": 0 },
  "notify_baseline_missing": 0,
  "unattributed_feedback": 0,
  "children": [
    { "student_id": 11, "student_name": "王小明", "child_happy": 1, "received": 0, "thanks": 0 }
  ]
}
```

- `children` 覆盖本任务**有提交**的每个孩子（无反馈时计数为 0），按 `student_id` 升序。
- 归属规则：反馈计入同一条回应覆盖的孩子，满足其一即可 —— 该孩子那条提交的提交人
  就是反馈人，或反馈人与该孩子有 `student_parents` 绑定（co-parent）。一条反馈命中
  同一家长的多个孩子时按孩子各计一次；**总量仍按 `(回应, 家长)` 对只计一次**。
- 两者都不命中的反馈计入 `unattributed_feedback`（可观测，不静默丢弃）。
- 任务不属于当前老师班级/园所 → **404**。不返回家长身份信息，不提供评分或排名。

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
      "created_at": "2026-03-19T20:30:00Z",
      "supplements": []
    }
  ],
  "total": 1
}
```

- `status=pending` 由后端按 `created_at ASC` 排序。
- `public_status`：`none` / `pending` / `approved` / `revoked`，供审核徽标与筛选使用。
- `response_id > 0` 时额外返回 `response` 对象（`id` / `ritual` / `sticker_key` / `content` / `media_url` / `response_type` / `created_at`）。
  `response_type`：`single` = 逐条回应（只覆盖一条提交）；`period` = 周期汇总回应（一次覆盖一个周期内的多条提交）。
  两者都是主回应；**补充回应固定 `single` 且只出现在 `supplements`**，不改变 `response_id`，也不计入主回应指标。
- `supplements`：该提交上追加的补充回应，**始终是数组**（空数组而非 `null`），按 `created_at ASC, id ASC`。
  每项形状 `{ id, checkin_id, ritual, sticker_key, content, media_url, created_at }`；`ritual` 只会是 `sticker` / `voice`。
  只返回未删除的关系行/提交/回应。老师端用它显示「已补充 N 条」，避免下次进入以为没补过而重复追加。
  **读取补充回应失败时整个 checkins 请求失败**（不返回空数组）：`[]` 只能表示「确实没补过」。
- 不返回 `points`。
- **任务结束后仍可回看**：本端点不按任务有效期过滤，任务 `status=0` 或已过 `end_date` 时老师仍能读取提交与回应，用于事后复盘；**不提供导出**（产品决定 2026-10-25，见 `tasks.md` 4.6 注记）。

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
  "created_at": "2026-03-19T20:35:00Z",
  "notified_count": 1,
  "notify_failed_count": 0
}
```

- `ritual`：`sticker` / `voice` / `class_star`。
- 回应后提交 `response_id` 指向该回应。
- `notified_count` / `notify_failed_count`：按**家长用户**去重（提交人及其同孩子的另一位绑定家长都会收到），同一家长只计一次；只统计主回应，补充回应不发通知。通知失败不回滚回应。
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
  "failed_ids": [],
  "notified_count": 18,
  "notify_failed_count": 0
}
```

- 只对**当前周期**内（每日=当天；每周=滚动 7 天；任务已结束时取最后一天/最后一周）仍为 `response_id=0` 的提交创建一条 `period` 回应，并写入 `period_start` / `period_end`；旧周期的待回应提交需逐条回应。
- `notified_count` / `notify_failed_count` 按家长用户去重：一次汇总回应覆盖同一家长多条提交只计一次；失败的提交（`failed_ids`）不通知。空操作（`id=0`）时两个计数都是 0。
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
  "task_id": 1,
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
- 前置：**批准**要求任务 `allow_public=true` 且不是个体任务，并且提交者孩子**当前仍在该任务班级**（转班或学生记录已删除后拒绝，否则只会造出一个永远不上墙的「已公开」状态），否则 **422**。**拒绝/撤回不受此限**——任务被关掉公开开关、改成个体任务、孩子已转班或学生记录已删除后，老师仍必须能把残留的公开状态清干净，此时 `approve=false` 照常返回 200。
- 任务不属于当前老师班级 → **404**；提交不属于该任务 → **422**；行锁等待超时/死锁 → **503**。
- 再次申请必须重新批准、重新勾确认：取消/撤回会把 `public_scope_confirmed` 清成 `false`，不会沿用。

### GET `/teacher/tasks/:id/share-card`

> 生成**家长端**任务详情的小程序码（PNG），供老师保存到相册后转发班级群

```json
// Response（HTTP 200）
{
  "task_id": 1,
  "page": "pages/tasks/detail",
  "scene": "t=1",
  "env_version": "release",
  "image_base64": "iVBORw0KGgoAAAANSUhEUg..."
}
```

- 小程序码**必须用家长端凭据**生成：用老师端凭据会生成老师端小程序码，家长扫码进错端、任务触达不到。
- `env_version` 取 `wechat.code_env_version`（默认 `release`；开发/测试用 `trial` / `develop`）。配置非法（不是这三个值）→ **500**，错误信息里明确说明。
- `scene=t=<task_id>`（≤32 字符，只含任务 id）；`page=pages/tasks/detail`；`check_path=false`（页面尚未发布到线上版本时也能生成）。
- 任务不属于当前老师班级或已删除 → **404**。
- 家长端 `onLoad` 解析 `scene` 得到 `t`（缺省回退 query `id`），并带上当前孩子的 `student_id`。

### GET `/teacher/class/star-jar`

> 读取**当前班级当前学期**的班级星星罐目标与进度。班级取自登录态，不接受 query 指定班级。
> 管理端的同名能力是 `GET /admin/classes/:id/star-jar`，**响应不含 `stars` / `progress` / `over_goal`**（管理端不展示进度）。

```json
// Response（HTTP 200）
{
  "class_id": 1,
  "term": "2031-2032-spring",
  "term_start": "2031-01-21",
  "term_end": "2031-07-10",
  "goal": 30,
  "configured": true,
  "updated_at": "2026-10-23T09:00:00Z",
  "stars": 12,
  "progress": 40,
  "over_goal": false
}
```

- `term` = `<班级 academic_year>-<当前学期>`；班级没有学年时用 `<自然年>-<学期>`。
- 当前学期取 `kindergarten_app_settings.current_semester`（`spring` / `fall`，非 `fall` 一律按 `spring`）。
- `term_start` / `term_end` 口径：`academic_years` 有配置就用配置；有学年但没配置起止 → 自然学期兜底（`spring` = 次年 2/1–7/31，`fall` = 当年 9/1–次年 1/31）；**班级没有学年**（或学年不是 `YYYY-YYYY`、起止年倒序）→ 自然年按学期切分（`spring` = 1/1–7/31，`fall` = 8/1–12/31）。
- **两学期无缝且不重叠**（2026-10-24 决策）：按起点排序后，后一个学期的起点强制等于前一个结束的次日。配置里 spring 与 fall 之间留了暑假空档时，`fall` 起点会被前移到 `spring_end + 1`；配置里 fall 在 spring 之前且留了寒假空档时，`spring` 起点会被前移到 `fall_end + 1`。**`term` 键不变**，只有区间收口。这样学期之间的提交不会掉进「两边都不算」。
- 未配置时返回 `goal=0`、`configured=false`、`updated_at=null`（仍带 `term`，客户端可据此展示「请先配置」）。
- 未配置时 `stars` **仍如实返回**（历史 class scope 任务可能已经集过星），只有 `progress` / `over_goal` 退化成 `null` / `false`；客户端应据 `configured` 决定是否展示进度，而不是据 `stars>0`。
- 学期之间的目标互相独立：上学期配置过不会让本学期任务通过校验。

**进度字段（`stars` / `progress` / `over_goal`）**

- `stars`：当前学期已集星数。口径与任务列表的 `star_count` 同一套 SQL 语义，只是范围从「本任务」换成「本班」：
  - 只算 **`star_scope=class`** 的任务（`star_scope=task` 的任务只进自己的任务罐）；
  - 只算**主回应**：`task_check_ins.response_id` 指向的回应 `ritual=class_star` 才计入；一次 `period` 回应覆盖 N 条提交就计 N 星；
  - 补充回应不写 `response_id`，**不计入**；贴纸 / 语音不计入；
  - 提交、回应、任务三层都过滤软删除；
  - 「当前学期」按**提交日期**（`task_check_ins.date`）落在 `term_start..term_end` 内判定：罐按学期重置，跨学期的长任务按提交日拆到两边。
- `progress`：`floor(stars * 100 / goal)`，**封顶 100**；`goal<=0`（未配置）时为 `null`（不是 `0`，避免把「没有目标」画成 0%）。
- `over_goal`：`stars > goal` 时为 `true`，客户端显示「已超额 N/G」而不是 >100% 的进度条；未配置时恒为 `false`。
- 契约 MUST NOT 提供个人星星排名或个人完成率排名字段。

### PUT `/teacher/class/star-jar`

```json
// Request
{ "goal": 30 }

// Response（HTTP 200）
{ "class_id": 1, "term": "2031-2032-spring", "term_start": "2031-01-21", "term_end": "2031-07-10", "goal": 30, "configured": true, "updated_at": "2026-10-23T09:01:00Z" }
```

- **响应与 GET 不同**：PUT 只回目标配置（上图字段），**不含** `stars` / `progress` / `over_goal`；写入后要展示进度，客户端必须再调一次 GET。
- 目标 **必须 > 0**：`0` / 负数 → **422**「班级星星罐目标必须大于 0」。`0` 是「未配置」，不是「关闭」。
- 按 `(kindergarten_id, class_id, term)` upsert；重复设置同一学期为覆盖，不新增行。
- 任务 `star_scope=class` 的创建/更新只认**当前学期**这里有 `goal>0`；未配置 → **422**「请先配置班级星星罐」。
- 写入 `operation_logs`（`module=tasks`、`action=set_star_jar`、`target_type=class_star_jar`）。

## 说明

- 公开审核见上方 `PUT /teacher/tasks/:id/checkins/:checkin_id/public`；公开墙本身只在家长端。
- 分享码见上方 `GET /teacher/tasks/:id/share-card`；补充回应未读由家长端 `POST /parent/tasks/:id/read` 清除（家长端契约）。
- 班级罐配置见上方 `GET|PUT /teacher/class/star-jar`；管理端同能力见 `api-admin/10-tasks.md`。任务 `star_scope=class` 时忽略任务 `star_goal`。
- 任务不发放积分。

