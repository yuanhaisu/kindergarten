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

记录型任务 P0 增量（**契约先行，服务端待实现**，详见 §9）：

- `POST /teacher/tasks/:id/checkins/batch-response`
- `POST /teacher/tasks/:id/checkins/:checkin_id/read`
- `POST /teacher/tasks/:id/end`
- `GET /teacher/tasks/:id/participants`
- `POST /teacher/tasks/:id/remind`

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

> 学生详情（含家长联系方式、健康备注）
>
> 跨班/跨园读取一律 **404**。`student` 为学生对象原样返回（字段以 `students` 表为准，
> 与既有响应一致）；本次变更只在同层新增 `parents`。

```json
// Response
{
  "student": {
    "id": 1,
    "kindergarten_id": 4,
    "class_id": 10,
    "name": "王小明",
    "gender": 1,
    "birthday": "2021-09-15T00:00:00Z",
    "avatar": "https://cdn.example.com/avatar/s1.jpg",
    "student_no": "2025001",
    "id_card_number": "",
    "address": "",
    "health_note": "对花生过敏",
    "enroll_date": "2025-09-01T00:00:00Z",
    "status": 1,
    "status_changed_at": null,
    "created_at": "2025-08-20T10:00:00+08:00",
    "updated_at": "2025-08-20T10:00:00+08:00"
  },
  "parents": [
    {
      "name": "王明",
      "relation": "father",
      "relation_label": "爸爸",
      "phone": "138****8888"
    },
    {
      "name": "李丽",
      "relation": "mother",
      "relation_label": "妈妈"
    }
  ]
}
```

#### `parents[]` 规则

- 每项字段：`name`（用户昵称）、`relation`（9 码闭集）、`relation_label`
  （服务端按枚举映射的中文，未知 code 原样返回）、`phone`（**脱敏**，如 `138****8888`）。
- **手机号逐家长过滤**：只有该家长在**当前园所**的 `show_phone_to_teacher=true` 时才返回
  `phone`；关闭时该键**缺省**，不是空串。没有 `parent_settings` 行按默认 `true` 处理。
- 同一幼儿有多个家长时逐个判断：一人关闭不影响另一人的手机号是否返回。
- 只返回该幼儿未删除的 `student_parents` 绑定；其中 `users` 行缺失（账号已注销/数据缺失）
  的绑定直接跳过，不出现在数组里。
- 不会混入同班其他幼儿的家长；无任何绑定返回空数组 `[]`（不是 `null`）。
- 该开关只影响手机号，不影响 `student` 中的姓名、班级与健康备注。

---

## 8. 亲子任务

> A 切片：班级/个体任务、提交、老师单条/周期回应、无积分。B 切片已补：汇总后个别补充回应、公开审核。分享码仍待补。
>
> 老师端任务操作限定当前班级。

### GET `/teacher/tasks`

```json
// Query: ?page=1&page_size=20&filter=all|active|pending|todo|ended|completed
// filter 可选（大小写不敏感）；缺省/未知值 = all。旧参数 status=active|ended 仍兼容（见下）

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
      "allow_late_submission": false,
      "requirements": ["每天阅读10分钟", "上传一张阅读照片"],
      "recognition_ritual": "sticker",
      "star_scope": "task",
      "star_goal": 20,
      "star_count": 12,
      "participant_count": 18,
      "student_count": 28,
      "unsubmitted_count": 10,
      "pending_responses": 3,
      "pending_period": 3,
      "period_start": "2026-03-25",
      "period_end": "2026-03-25",
      "created_at": "2026-03-18T10:00:00Z"
    }
  ],
  "stats": { "all": 8, "active": 5, "pending": 3, "todo": 4, "ended": 3, "completed": 2, "pending_responses": 21 },
  "total": 8,
  "page": 1,
  "page_size": 20
}
```

- `filter`（query，可选）：任务视图筛选，**在 SQL 中过滤**，分页语义不变（`total` 为筛选后的任务总数）。取值大小写不敏感、首尾空白忽略；缺省或未知值按 `all` 处理。
  - `all`：当前班级全部任务，不附加条件（默认行为）。
  - `active`：有效进行中 = `status=1` **且** 未有效结束：`allow_late_submission=false` 时要求 `end_date >= 今天`（Asia/Shanghai），`true` 时只要未手动结束。按实时日期判定，不只看 `status`：后台把任务标成进行中但日期已过时，不算「进行中」（§9.5）。
  - `ended`：有效结束 = `status=0`，或 `allow_late_submission=false` 且 `end_date < 今天`。它包含「已完成」「已结束 · 未提交」以及「已结束但仍有待回应」三种情况，**不等于**已完成。
  - `pending`：存在待回应提交 = `pending_responses > 0`（`response_id=0` **且** `teacher_read_at IS NULL`）。**不限任务是否已结束**：已结束但仍有未处理提交的任务也进这个视图，否则这些提交会失去入口；因此它与 `active` / `ended` 会重叠。
  - `todo`：待提交 = `active` 且 `unsubmitted_count > 0`。按 §9.1 的口径用 `unsubmitted_count`，**不用**产品 3.3 写的 `participant_count < student_count`：`participant_count` 含已转班/离班的历史提交者，会漏掉实际未提交的在读孩子（§9.7）。
  - `completed`：已完成 = `ended` **且** `submission_count > 0` **且** `pending_responses = 0`。三条缺一不可：只看 `pending_responses=0` 会把「已结束但无人提交」误判为已完成（产品 3.3 指出的问题）；只看 `ended` 会把仍有积压的任务算进来。`pending_responses` 已排除标记已阅的提交，所以「仅标记已阅」可让任务进入已完成；若产品要求「已完成」必须全部有主回应（`response_id > 0`），需另行确认（§9.11）。
- `status`（query，**旧参数，兼容保留**）：仅接受 `active|ended`，语义与同名 `filter` 取值一致；`filter` 存在时忽略；缺省/未知值 = 全部。新代码一律使用 `filter`。
- 分页：`filter` 只改变 `WHERE`，不改变列表既有排序；`total` 是筛选后的总数，不是全班任务数。
  设计稿当前在客户端派生「待提交 / 已完成」（09 原型按 `pending_responses` 本地过滤），那只是原型做法。接入时必须使用服务端 `filter`：客户端二次过滤只能覆盖已加载页，`total`、页签计数与跨页结果都会错，大数据量下还要求全量拉取。
- `stats`：当前老师**当前班级全部任务**的计数，**不受 `filter` 筛选与分页影响**（页签计数必须稳定），一次聚合查询：
  - `all` / `active` / `pending` / `todo` / `ended` / `completed`：各视图的**任务数**（不是提交数），口径与同名 `filter` 取值逐一相同。
  - `all = active + ended` 恒成立；`completed ⊆ ended`；`pending` 与 `todo` 是**交叉视图**，可能同时命中 `active` 或 `ended`，不要用它们参与分组求和。
  - `pending_responses`：当前班全部任务的 `pending_responses` 之和（09 顶部「M 条待回应」），与 `pending`（任务数）区分；同样不受筛选影响。
- `participant_count`：该任务**已提交过的去重学生数**（`COUNT(DISTINCT student_id)`，软删提交不计）。
  与详情 stats 的 `submission_count`（打卡行数）不同：同一学生多次打卡只算 1 人。
  列表级一次 `GROUP BY task_id` 聚合，不逐任务查询；查询失败时整个列表失败（不返回 `null` 或 0）。
- `student_count` / `unsubmitted_count`：班级任务取当前班在读学生数，以及其中**没有未删除提交**的人数；个体任务按目标孩子计：目标仍在当前班在读时 `student_count=1`，否则 0；`unsubmitted_count` 同理（0/1）。`unsubmitted_count` 是 09「待提交」筛选依据；
  **不要用 `participant_count < student_count` 代替**——`participant_count` 是历史去重提交者
  （含已转班/离班孩子），转班场景两者会不等（见 §9.7）。列表级一次聚合，不逐任务查询。
- `requirements`：提交要求，**始终数组**（`[]` 表示无要求），与创建请求同名字段一致；供 09 卡片、10 详情与分享卡使用。
- `allow_late_submission`：截止后是否允许补交，**始终布尔**，默认 `false`；筛选与结束口径见 §9.5。
- `pending_responses` 是当前任务下全部**待回应**提交数 = `response_id=0` **且** `teacher_read_at` 为空（含旧周期遗留）。
  标记已阅的提交不算待回应（§9.3）。
- `pending_period` 是**本周期内**待回应的条数，与批量回应接口的筛选条件逐字一致
  （`task_id=? AND response_id=0 AND teacher_read_at IS NULL AND date BETWEEN period_start AND period_end`）：
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
  "target_student": null,
  "frequency": "daily",
  "start_date": "2026-03-19",
  "end_date": "2026-03-25",
  "allow_public": false,
  "allow_late_submission": false,
  "requirements": ["每天阅读10分钟", "上传一张阅读照片"],
  "recognition_ritual": "sticker",
  "star_scope": "task",
  "star_goal": 20,
  "star_count": 12,
  "status": 1,
  "ended_at": null,
  "ended_by": 0,
  "period_start": "2026-03-21",
  "period_end": "2026-03-21",
  "created_at": "2026-03-18T10:00:00Z"
}
```

- `target_student`：`target_student_id > 0` 时返回 `{ "id", "name", "avatar" }`，班级任务为 `null`。详情头部与个体任务分享卡直接用它，不需要客户端再拉全班名单。
- `requirements` / `allow_late_submission`：口径同列表（§9.1 / §9.5 / §9.6）。
- `ended_at` / `ended_by`：仅**手动结束**时非空（§9.4）；到期自动结束保持 `null`，客户端用 `end_date` + `allow_late_submission` 判断。
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
  "allow_late_submission": false,
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
- `requirements` 可选，字符串数组，缺省 `[]`；空字符串/纯空白条目 → 422（§9.6）。
- `allow_late_submission` 可选，默认 `false`；语义见 §9.5。
- `target_student_id > 0` 时必须是当前班级在读学生；个体任务 `allow_public` 强制 false。
- `star_scope=class` 时必须已有班级罐目标，否则拒绝。
- 不接收 `points`。
- `notified_count` / `notify_failed_count`：发布站内消息的投递结果，**按家长用户去重**。班级任务覆盖该班全部家庭（同一家长有两个孩子只计一次，消息 `action_path` 固定取该家长在该班 id 最小的孩子）；个体任务只通知目标孩子的家庭。文案含任务标题；通知失败不回滚任务，只计数并写日志。

### PUT `/teacher/tasks/:id`

```json
// Request：字段同创建；有提交后只允许 title/description/requirements/learning_link/allow_public/allow_late_submission
{
  "title": "每日阅读15分钟",
  "description": "时间延长",
  "learning_link": "延续本周班级的春天主题",
  "allow_public": false
}

// 需要同时调整提交要求 / 补交开关时追加：
{
  "requirements": ["每天阅读15分钟", "上传一张阅读照片"],
  "allow_late_submission": true
}
```

- 有提交后修改 `frequency`、起止日期、`target_student_id`、`star_scope`、`star_goal` 返回参数错误。
- `requirements` / `allow_late_submission` 有提交后仍可修改（与 `description` 同组）：只影响后续提交提示与到期判定，不改变已提交内容，也不追溯已手动结束的任务。
- `star_goal` 在创建与编辑两条路径都必须 ≥0（负数返回参数错误）；`recognition_ritual=class_star` 且 `star_scope=task` 时还必须 >0。
- **编辑时可选字段缺省 = 保留原值**：`star_goal` / `allow_public` / `description` / `requirements` / `allow_late_submission` 不传不会清零目标、不会关闭公开、不会清空说明/要求、不会重置补交开关（与 `api-admin/10-tasks.md` 同口径）。因此上面的最小编辑示例可以直接用。
- `star_scope` 只接受 `task` / `class`，`recognition_ritual` 只接受 `sticker` / `voice` / `class_star`（首尾空白会被 trim）；非法值返回参数错误，不会落库成非规范值。
- `allow_public=false` 时批量撤销公开（B 切片字段，A 先保留）。
- **活动任务（`activity_id` 非空）只允许 `title` / `description` / `requirements` / `learning_link` / `allow_public`**：
  频率、起止日期、`target_student_id`、`recognition_ritual`、`star_scope`、`star_goal`、`allow_late_submission`、`status`
  任一变化 → **422**「活动任务只能修改标题、说明、提交要求、学习关联与公开开关，请让管理员整组编辑」。
  活动任务的「一个班级一条、共享 `activity_id`」只能由管理端整组维护（design D12）。
  任务级结束不使用本接口的 `status` 表达，请用 `POST /teacher/tasks/:id/end`（§9.4）。

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
  "total_students": 28,
  "submission_count": 21,
  "participant_count": 18,
  "responded_count": 12,
  "pending_count": 9,
  "unsubmitted_count": 10
}
```

- `submission_count`：打卡**行数**（同一孩子跨天多次提交会多次计），用于待回应/回应进度。
- `participant_count`：**已提交过的去重学生数**（`COUNT(DISTINCT student_id)`），用于「已参与 N/M」。
  与 `submission_count` 语义不同：同一学生多次打卡只算 1 人，客户端不要把行数当人数。
- `total_students`：班级任务 = 当前班**在读**学生数（`students.status=1` 且未删除）；个体任务 = 目标孩子仍在当前班在读时返回 1，否则返回 0。
  **查询失败返回 500，不得静默返回 0**——那会把整班显示成「没有孩子」。
- `responded_count`：已回应的提交数 = `response_id > 0` **或** `teacher_read_at` 非空的提交数（标记已阅计入，§9.3）。
- `pending_count`：`submission_count - responded_count`。
- `unsubmitted_count`：班级任务 = 当前在读且没有未删除提交的学生数；个体任务 = 目标孩子仍在读且没有未删除提交时返回 1，否则 0。它按当前 scope 计算，不能用 `total_students - participant_count` 反推（`participant_count` 含已转班/离班的历史提交者）。
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
      "media_url": "https://cdn.example.com/task/checkin/1.jpg", // 1 小时签名 URL
      "is_public_requested": false,
      "public_scope_confirmed": false,
      "public_approved_at": null,
      "public_revoked_at": null,
      "public_status": "none",
      "public_visible": false,
      "response_id": 0,
      "teacher_read_at": null,
      "created_at": "2026-03-19T20:30:00Z",
      "supplements": []
    }
  ],
  "total": 1
}
```

- `status=pending|responded` 的判定含标记已阅：`responded` = `response_id > 0` **或** `teacher_read_at` 非空；`pending` 是其余。
  `response_id=0` 且 `teacher_read_at` 非空表示「老师已阅、未回应」（§9.3）。
- `status=pending` 由后端按 `created_at ASC` 排序。
- `teacher_read_at`：老师标记已阅的时间，未标记为 `null`；不写主回应、不改变 `response_id`、不通知家长。
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

- 只对**当前周期**内（每日=当天；每周=滚动 7 天；任务已结束时取最后一天/最后一周）仍为 `response_id=0` **且未标记已阅**的提交创建一条 `period` 回应，并写入 `period_start` / `period_end`；旧周期的待回应提交需逐条回应，已标记已阅的提交不会被周期汇总重新回应（§9.3）。
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
- **本接口只返回小程序码，不返回整张海报**：海报由客户端 Canvas 合成、原生转发走 `onShareAppMessage`，尺寸与字段要求见 §9.9。

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

---

## 9. 记录型任务 P0 增量（2026-11-21）

> **契约先行**：本节端点与字段用于老师端 09 / 10 / 10a 的记录型任务 P0，**服务端实现是后续工作**，未实现前客户端不得调用。
> 产品依据：2026-11-21 记录型任务 P0 产品决策（原产品文档未入库，本文 §9 为唯一契约口径）；字段落库以 `server/internal/domain/` 为准。
> 本节不改动既有端点的已实现行为；新增字段对旧客户端是兼容的（多返回字段不应导致报错）。

### 9.1 任务字段增量（创建 / 更新 / 列表 / 详情）

| 字段 | 类型 | 创建 | 更新 | 列表 | 详情 | 说明 |
| --- | --- | --- | --- | --- | --- | --- |
| `requirements` | string[] | 可选 | 可选 | 返回 | 返回 | 提交要求；缺省 `[]`，始终数组 |
| `allow_late_submission` | bool | 可选 | 可选 | 返回 | 返回 | 截止后允许补交；默认 `false` |
| `ended_at` | string\|null | — | — | — | 返回 | 仅手动结束非空 |
| `ended_by` | int | — | — | — | 返回 | 手动结束老师 id；未结束为 0 |
| `target_student` | object\|null | — | — | — | 返回 | 个体任务目标孩子；班级任务 `null` |
| `student_count` | int | — | — | 返回 | — | 班级任务=当前班在读学生数；个体任务=目标孩子在读 1/0 |
| `unsubmitted_count` | int | — | — | 返回 | — | 班级任务=当前在读且无未删除提交人数；个体任务=目标孩子未提交 1/0 |

- `requirements`：字符串数组，原样存取；`[]` 表示无提交要求。创建缺省 `[]`；更新不传 = 保留原值。有提交后仍可修改，只影响后续家长的提交提示，不改变已提交内容（§9.6）。
- `allow_late_submission`：创建缺省 `false`；更新不传 = 保留原值。有提交后仍可修改；只影响后续到期判定，不追溯已手动结束的任务（§9.5）。
- `student_count`：班级任务取当前班在读学生数；个体任务取 `1`（目标孩子仍在当前班在读）或 `0`（目标已不在当前班在读）。列表级一次班级查询，不逐任务查询。
- `unsubmitted_count`：班级任务取当前班在读学生中**没有**未删除提交的人数；个体任务按目标孩子计（0/1；目标已不在当前班在读 → 0）。它是 09「待提交」筛选的依据，**不能用 `participant_count < student_count` 代替**：`participant_count` 是历史去重提交者（含已转班/离班孩子），转班场景两者会不等（§9.7）。
- `target_student`：仅详情返回，形状 `{ "id", "name", "avatar" }`；`target_student_id=0` 时为 `null`。用于详情头部与个体任务分享卡，客户端不需要再拉全班名单。

### 9.2 选中提交批量回应

#### POST `/teacher/tasks/:id/checkins/batch-response`

> 对**老师选中的若干条提交**批量应用同一条回应（待回应队列多选）。与周期汇总 `POST /teacher/tasks/:id/batch-response` 是不同端点、不同语义。

```json
// Request
{
  "checkin_ids": [401, 403, 405],
  "ritual": "sticker",
  "sticker_key": "star",
  "content": "今天读得真认真",
  "media_file_id": 0
}

// Response（HTTP 201）
{
  "responded_count": 3,
  "failed_ids": [],
  "responses": [
    { "checkin_id": 401, "response_id": 21 },
    { "checkin_id": 403, "response_id": 22 },
    { "checkin_id": 405, "response_id": 23 }
  ],
  "notified_count": 3,
  "notify_failed_count": 0
}
```

- `checkin_ids`：必填，1–100 个；重复 id 去重后处理；空数组或超过 100 个 → **422**。
- 每条选中提交**各创建一条 `response_type=single` 的主回应**，不共享 `period` 回应、不写 `period_start` / `period_end`；因此可以跨周期选择，也适用于个体 / 一次性 / 已结束任务。
- 写前校验：任一 id 不存在、已被软删、或不属于本任务 → **422**「选中的提交不属于该任务，请刷新后重试」，整批不写。
- 并发关闭：某条已被其他回应关闭（`response_id > 0`）→ 计入 `failed_ids`，其余照常写入；全部都已关闭 → **409**「所选提交均已回应，请刷新后重试」，仍返回 `responded_count=0` 与 `failed_ids`，不创建回应。
- 部分成功 → **201**，`responded_count` 只计成功条数；有候选但一条都没写成功（非并发关闭）→ **500**，提交保持原状态。
- 仪式载体校验与单条回应完全一致：`sticker` 必须带 `sticker_key`；`voice` 必须带 `media_file_id` 且文件类型为音频；`class_star` 不需要载体。非法 → **422**，整批不写。
- `class_star` 按**每条成功提交**各计 1 星（与逐条回应一致；周期汇总回应覆盖 N 条提交的计星方式不同）。
- `notified_count` / `notify_failed_count`：按家长用户去重，同一家长选中多条提交只计一次；`failed_ids` 不通知；不发补充回应通知。
- 任务不属于当前老师班级/园所 → **404**；行锁等待超时/死锁 → **503**「操作冲突，请稍后重试」。

**与周期汇总的区别**

| | `POST /teacher/tasks/:id/batch-response`（周期汇总） | `POST /teacher/tasks/:id/checkins/batch-response`（选中批量） |
| --- | --- | --- |
| 选择范围 | 当前周期内全部待回应（`response_id=0` 且未标记已阅）的提交 | 老师显式选中的提交，可跨周期 |
| 回应类型 | 一条 `period` 回应覆盖 N 条提交 | 每条提交一条 `single` 回应 |
| 周期字段 | 写 `period_start` / `period_end` | 不写 |
| 适用任务 | 仅每日/每周的班级任务 | 任意任务 |
| 空操作 | 200 + `id=0` | 409（全部已被关闭）或 422（没有有效 id） |
| 家长展示 | 一条「本周期汇总」 | 每条提交各自一条回应 |

### 9.3 标记已阅

#### POST `/teacher/tasks/:id/checkins/:checkin_id/read`

> 老师「仅标记已阅」：不选贴纸、不写内容，也不通知家长。

```json
// Request：无 body
{}

// Response（HTTP 200）
{
  "checkin_id": 401,
  "task_id": 1,
  "teacher_read_at": "2026-11-21T10:00:00Z",
  "read_by": 11,
  "responded_count": 13,
  "pending_count": 8
}
```

- 只写提交上的已阅标记（`teacher_read_at` / `read_by`，字段名以 `server/internal/domain/` 落库为准），**不创建 `TaskResponse`**，不改变 `response_id`，不通知家长。
- 老师端统计口径：`responded_count` = `response_id > 0` **或** `teacher_read_at` 非空的提交数；因此标记已阅会让 `responded_count +1`、`pending_count -1`，`submission_count` 不变。响应回传当前累计值，便于客户端就地更新。
- 幂等：同一条重复调用返回 **200** 与首次的 `teacher_read_at`，不刷新时间、不重复计数。
- 标记已阅不是终态：之后仍可调用单条回应 / 选中批量回应写入主回应；写入后 `response_id > 0`，以主回应为准。对已标记已阅的提交调用 `POST .../response` 正常执行，不返回 409。
- 已有主回应（`response_id > 0`）的提交调用本接口 → **409**「该提交已有主回应，无需标记已阅」。
- 提交不存在、已被软删或不属于本任务 → **422**；任务不属于当前老师班级/园所 → **404**；行锁等待超时/死锁 → **503**。
- **家长端不感知**：家长端 `responded` 仍按 `response_id > 0` 判定，标记已阅不会让家长端显示「老师已回应」。如果产品要求家长看到「老师已阅」，需要家长端契约另开变更（§9.10）。
- 批量标记已阅不在本次增量；需要时对同一端点扩展 `ritual=read` 或新增 batch-read，先补契约。

### 9.4 任务级结束

#### POST `/teacher/tasks/:id/end`

> 手动结束整个任务。与「回应关闭一条提交」是两件事。

```json
// Request：无 body
{}

// Response（HTTP 200）
{
  "task_id": 1,
  "status": 0,
  "ended_at": "2026-11-21T10:00:00Z",
  "ended_by": 11,
  "already_ended": false
}
```

- 写 `status=0`、`ended_at=now`、`ended_by=当前老师`；**不发送家长通知**。
- 幂等：已手动结束（`status=0` 且 `ended_at` 非空）→ **200** 原值返回，`already_ended=true`，不刷新 `ended_at`。
- 与回应级关闭的区别：回应关闭的是**一条提交**（`checkin.response_id`）；本接口结束的是**整个任务**，只阻止新的家长提交，不关闭已有提交。结束后老师仍可查看并回应已有提交（逐条或选中批量均可）。
- **手动结束优先于补交**：不论 `allow_late_submission` 为何值，手动结束后家长都不能再提交。
- 到期自动结束：`allow_late_submission=false` 时，`end_date` 次日 00:00（Asia/Shanghai）起视为**有效结束**，不写库、`ended_at` 保持 `null`；`allow_late_submission=true` 时不自动结束，直到老师手动结束（§9.5）。
- 活动任务（`activity_id` 非空）不能由老师单条结束 → **422**「活动任务不能单独结束，请让管理员结束整个活动」（与 DELETE 同口径）。
- 任务不属于当前老师班级/园所 → **404**；行锁等待超时/死锁 → **503**。
- 任务级结束不使用 `PUT /teacher/tasks/:id` 的 `status` 字段表达；`PUT` 不新增结束语义。

### 9.5 截止后补交 `allow_late_submission`

- 创建缺省 `false`；更新不传 = 保留原值。详情 / 列表始终返回布尔值（不是 `null`）。
- `false`（默认）：`end_date` 次日 00:00（Asia/Shanghai）起任务**有效结束**：
  - 家长不能再提交；
  - `/teacher/tasks?status=ended` 命中、`status=active` 不再命中（按实时日期判定，不只看 `status` 列）；
  - `ended_at` 仍为 `null`（没有手动结束动作）。
- `true`：`end_date` 之后仍可补交，任务保持进行中，直到老师调用 `POST /teacher/tasks/:id/end` 手动结束。
- **有效结束的统一判定**：`status=0` **或**（`allow_late_submission=false` 且 `今天 > end_date`）。老师端筛选、家长端状态、公开墙条件都复用该口径，不各自实现。
- 切换到 `false` 且 `end_date` 已过 → 任务立即有效结束，没有宽限期；切换到 `true` 只对**未手动结束**的任务生效，不会重新开放已手动结束的任务。
- 个体任务与班级任务同口径。
- 家长端契约需要同步修改（§9.10）：`POST /parent/tasks/:id/checkin` 在有效结束后返回 **409**「任务已结束，不能再提交」；`allow_late_submission=true` 且 `status=1` 时，`end_date` 之后仍允许提交。

### 9.6 提交要求 `requirements`

- 类型：字符串数组，原样存取；`[]` 表示无提交要求，不用 `null`。
- 创建可选，缺省 `[]`；更新不传 = 保留原值；有提交后仍可改（与 `description` 同组），因为它只影响后续家长的提交提示，不改变已提交内容。
- 数组内的空字符串 / 纯空白条目 → **422**，避免家长端出现空白要求。
- 列表 / 详情始终返回数组，供 09 卡片、10 详情、分享卡与家长端提交页使用；客户端展示与分享卡截断由客户端负责，契约不返回拼接后的字符串。
- 活动任务允许修改 `requirements`（内容型字段），但频率、起止日期、`allow_late_submission` 等仍按活动任务限制处理（见 `PUT /teacher/tasks/:id`）。

### 9.7 未提交名单

#### GET `/teacher/tasks/:id/participants`

> 任务维度的参与名单；09「未提交 N 人」点进去的名单来源。

```json
// Query: ?status=unsubmitted    // unsubmitted（默认）| submitted | all
// Response
{
  "list": [
    { "student_id": 11, "name": "王小明", "avatar": "https://cdn.example.com/avatar/s11.jpg" }
  ],
  "total": 10,
  "submitted_count": 18,
  "unsubmitted_count": 10
}
```

- 范围：任务所属班级的**当前在读**学生（`students.status=1` 且未删除）；个体任务只有 `target_student_id` 一个孩子；目标孩子已不在当前班在读时返回空列表，且 `total=0`、`submitted_count=0`、`unsubmitted_count=0`（stats 的 `total_students` 也为 0）。不扫描其它班级、不跨园。
- `total`：当前 `status` 筛选后的名单条数（`list.length`），不是 `total_students`；`unsubmitted` 时等于 `unsubmitted_count`，`submitted` 时等于 `submitted_count`，`all` 时等于两者之和。
- `submitted` = 至少有一条未删除提交；`unsubmitted` = 没有；`all` = 两者并集。
- 排序：`student_id ASC`；不分页（班级规模有限，一次全量）。
- `submitted_count` / `unsubmitted_count`：当前 scope 内已提交 / 未提交人数；班级任务按当前在读，个体任务按目标孩子 0/1。
- `submitted_count + unsubmitted_count = total_students`，与 `GET /teacher/tasks/:id/stats` 的 `total_students` 同口径；个体任务的 `total_students` 同样按目标孩子是否在读取 1/0。
- 与 `stats.participant_count` 的差异：后者是**历史去重提交者**，包含已转班/离班孩子；`submitted_count` 只算当前在读。转班场景 `participant_count` 可能大于 `submitted_count`。09 的「参与 N/M」如果要求严格 `N ≤ M`，应改用 `submitted_count / total_students`（待确认，§9.11）。
- 任务不属于当前老师班级/园所 → **404**；`status` 非法 → **422**。
- 实现为当前任务 + 当前班级的一次查询，不新增全班逐人扫描端点。

### 9.8 手动提醒

#### POST `/teacher/tasks/:id/remind`

> **手动**提醒未提交家庭。契约不提供任何自动 / 定时提醒。

```json
// Request
{
  "client_request_id": "8f3c2a1e-4d1b-4a63-9f60-2f5f4b8d9c11",
  "scope": "unsubmitted",
  "student_ids": []
}

// Response（HTTP 200）
{
  "notified_count": 10,
  "notify_failed_count": 0,
  "deduplicated": false
}
```

- `scope`：`unsubmitted`（默认）或 `student_ids`；其它值 → **422**。
  - `unsubmitted`：服务端按 §9.7 的 `unsubmitted` 口径实时计算名单；`student_ids` 必须缺省或空数组，否则 **422**。
  - `student_ids`：必须 1–100 个，且全部在当前未提交名单内（本班在读学生；个体任务只接受目标孩子）。任一学生已提交或不在范围 → **422**「所选学生已提交或不在未提交名单，请刷新后重试」，整批不发送。
- 只发**站内消息**（与创建任务同一渠道），不发明细 / 短信 / 模板消息；按家长用户去重，同一家长多个孩子只发一条；文案含任务标题与「待提交」，`action_path` 指向家长端任务详情。
- 没有绑定家长的学生不计入 `notified_count` / `notify_failed_count`（没有可投递对象）；客户端不要把 `notified_count` 当学生数。
- **只支持手动调用**：没有「发布后未提交提醒」「待回应提醒」的定时任务；老师不点，系统不发。
- 幂等：`client_request_id` 可选，建议每次用户动作生成、重试沿用。同一 `(teacher_id, task_id, client_request_id)` 24 小时内返回首次结果并置 `deduplicated=true`，不重复投递；未传 `client_request_id` 时，服务端按请求参数做 60 秒防连点去重。幂等只作用于「同一动作的重试」，不是发送频率上限。
- 没有可提醒对象（未提交名单为空）→ **200** `{ "notified_count": 0, "notify_failed_count": 0, "deduplicated": false }`，不发送。
- 部分失败 → **200**，两个计数如实返回；全部失败（`notified_count=0` 且 `notify_failed_count>0`）→ **500**，通知失败不回滚任何业务状态。
- 任务有效结束（§9.5）→ **422**「任务已结束，不能提醒」；`allow_late_submission=true` 且已过 `end_date` 但未手动结束的任务仍可提醒。
- 任务不属于当前老师班级/园所 → **404**；行锁等待超时/死锁 → **503**。

### 9.9 分享卡与原生转发

> `GET /teacher/tasks/:id/share-card` 的形状**不变**：只返回家长端小程序码 `image_base64`，不返回整张海报。

- 小程序码：仍必须用**家长端凭据**生成（见上方端点）；客户端把 `image_base64` 绘制到海报，二维码绘制尺寸 **≥200×200 px**，四周加白色 quiet zone（**≥8 px**），不得拉伸变形。
- 海报 Canvas：宽度 **≥750 px**（按 750 设计稿）；导出用 `canvasToTempFilePath`，保存相册用 `saveImageToPhotosAlbum`。任务表没有 `cover_url`，v1 封面用客户端预置插画，不新增字段。
- 海报内容取自任务详情与班级列表，不新增接口：

  | 元素 | 字段 |
  | --- | --- |
  | 标题 | `title` |
  | 提交要求 | `requirements`（多行展示，过长截断；完整要求以家长端为准） |
  | 日期与频率 | `start_date` / `end_date` / `frequency` |
  | 回应方式 | `recognition_ritual`（只展示 `sticker` / `class_star`，不展示 `voice`） |
  | 星星目标 | `star_scope=task` 时用 `star_goal`；`class` 时用班级罐目标或隐藏 |
  | 班级与家长数 | `GET /teacher/class/list` 的 `class_name` / `parent_count` |
  | 个别幼儿 | `target_student` 的 `name` 与「仅该家长可见」 |

- 原生转发用 `onShareAppMessage` 返回：
  - `title`：任务标题 + 提交要求摘要（如「每日阅读10分钟 · 每天阅读10分钟」）；无要求时只放标题。
  - `imagePath`：Canvas 合成海报的临时文件路径（建议 5:4，如 750×600 px）。
  - `path`：`pages/tasks/detail?id=<task_id>`（与小程序码 `scene=t=<task_id>` 指向同一页面；家长端 `onLoad` 在无 `scene` 时回退 query `id`，见上方 share-card）。
- 个体任务的分享仍是「私发给该家长」的产品约定，但微信原生转发无法限制接收人；服务端必须继续按 `target_student_id` 拦截其他家庭（家长端详情 404）。
- `share-card` 失败 / 超时 → 分享卡不可用；不要把空白二维码画进海报，客户端给出「重新生成」。

### 9.10 跨端影响（家长端契约后续修改）

> 本节只记录影响；`backend/api/api-parent/06-task.md` 的同步修改是后续工作。未改前家长端实现不得假设 `allow_late_submission` 存在。

- `allow_late_submission` 需要家长端契约同步：
  - `POST /parent/tasks/:id/checkin`：有效结束后 → **409**「任务已结束，不能再提交」；`allow_late_submission=true` 且 `status=1` 时，`end_date` 之后仍允许提交（直到老师手动结束）。
  - `GET /parent/tasks` 的 `status` / `scope=active`、`GET /parent/tasks/:id` 的 `status`：由「`end_date` 已过即结束」改为「有效结束」判定（§9.5）。
  - `GET /parent/tasks/public-wall` 等所有以 `end_date >= 今天` 判定任务未结束的条件，同样改用有效结束；否则允许补交的任务在到期后会从公开墙消失。
- 老师端 `/teacher/tasks?filter=active|ended`（旧 `status` 参数兼容）也使用同一有效结束口径；`allow_late_submission=true` 且已过 `end_date` 的任务仍算 `active`。
- 标记已阅（§9.3）不改变家长端：`responded` 仍按 `response_id > 0`；若产品要求家长看到「老师已阅」，需在家长端新增展示状态。
- 新增字段对家长端是**可选**的：家长端未升级时忽略 `requirements` / `allow_late_submission` 不应报错（服务端响应加字段不构成破坏性变更）。

### 9.11 与产品文档的差异 / 待确认

- **产品文档**：记录型任务的产品原文未入库；本 PR 不新增该文件，以 §9 的决策口径为准。
- **手动提醒 vs 产品 2.2**：产品 2.2（2026-11-21 决策，原产品文档未入库）写「不做发布后未提交提醒、待回应提醒」，但 2026-11-21 用户决定保留**手动**未提交提醒、不做自动提醒。本节按后者编写；产品文档需要回写这一条。
- **仍待定：标记已阅对家长是否可见**。产品 3.2 要求「仅标记已阅」，但未说明家长端看到什么。本契约把它定义为老师端效率字段（家长端 `responded` 不变）；若要让家长看到「老师已阅」，需要家长端新增状态。同时老师端 `responded_count` 与 `completed` 都把已阅视为已处理；如果产品要求「已完成」必须有主回应（`response_id > 0`），这两处要一起改。
- **活动任务能否单独结束**：本契约按 DELETE 同口径禁止（422）。如果活动任务允许老师单条结束，需要产品确认后放宽。
- **仍待定：结束任务是否通知家长**。本契约定义为**不通知**（与产品 2.2 不做提醒一致）。如果要在结束时推「任务已结束」，需要另补通知文案与计数。
- **`participant_count` 与 `submitted_count` 的口径**：转班 / 离班历史提交者会让 `participant_count > submitted_count`。09「参与 N/M」如果必须 `N ≤ M`，应改用 `submitted_count`（待确认是否调整 stats 语义）。
- **09 筛选参数（已收口）**：`GET /teacher/tasks` 已定义 `filter=all|active|pending|todo|ended|completed` 与 `stats` 各计数（见 §8）。相对产品 3.3 的差异：`todo` 用 `unsubmitted_count > 0` 而非 `participant_count < student_count`；「已完成」要求 `ended` + 至少 1 条提交 + `pending_responses = 0`，不能只拿 `pending_responses=0` 判定。
- **批量标记已阅**：本轮只定义单条 `read`；产品 3.2 的队列批量「仅标记已阅」尚未有批量端点（§9.3 末条）。
- **仍待定：`requirements` 的长度 / 条数上限**。本契约未设上限；若需要，按产品规则补 422。
- **仍待定：到期自动结束是否落库**。本契约定义为计算口径（`status` 可仍为 1，`ended_at=null`），不依赖定时任务。如果运营需要在后台按「已结束」筛选 / 导出，需要决定是否在 `end_date` 后写 `status=0`。

## 说明

- 公开审核见上方 `PUT /teacher/tasks/:id/checkins/:checkin_id/public`；公开墙本身只在家长端。
- 分享码见上方 `GET /teacher/tasks/:id/share-card`；补充回应未读由家长端 `POST /parent/tasks/:id/read` 清除（家长端契约）。
- 班级罐配置见上方 `GET|PUT /teacher/class/star-jar`；管理端同能力见 `api-admin/10-tasks.md`。任务 `star_scope=class` 时忽略任务 `star_goal`。
- 任务不发放积分。
- 记录型任务 P0 增量（选中批量回应、标记已阅、任务级结束、未提交名单、手动提醒、`allow_late_submission`、`requirements`）见 §9；**契约先行，服务端待实现**。

