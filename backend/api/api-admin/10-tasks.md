# 10. 亲子任务管理

> 对应前端页面：`admin/pages/tasks/index.vue`
>
> 基础路径：`/api/v1/admin`，需 `Authorization: Bearer <token>`，角色 `admin`。
>
> **状态（C-2，2026-10-23）**：`points` 已从 domain/契约移除；建/改任务的结构字段、班级星星罐、多班活动端点、新口径统计、公开内容查看**均已实现**。旧 `completion_rate` / `total_possible` / `checkin_count` 已下线（7.4）。管理端页面接新字段属 C-3。

---

## POST `/admin/tasks`

> 新建**独立任务**（`creator_id` 取当前登录用户）。多班活动请用 `POST /admin/task-activities`。
>
> 独立任务的 `activity_id` 恒为空串；本端点**不接受**客户端指定的活动分组。

```json
// Request
{
  "title": "每日阅读10分钟",
  "description": "每天陪孩子阅读10分钟绘本",
  "learning_link": "延续本周班级的春天主题",
  "class_id": 1,
  "target_student_id": 0,
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
  "notified_count": 12,
  "notify_failed_count": 0
}
```

| 字段 | 必填 | 默认 | 校验 |
| --- | --- | --- | --- |
| `title` | 是 | — | ≤200 |
| `description` | 否 | `""` | — |
| `learning_link` | **是** | — | 空 → 422「学习关联必填」（D15） |
| `class_id` | 是 | — | 必须属于当前园所，否则 404 |
| `target_student_id` | 否 | `0`（班级任务） | >0 时必须是该班在读学生，否则 422 |
| `frequency` | 是 | — | `daily` / `weekly` / `once` |
| `start_date` / `end_date` | 是 | — | `YYYY-MM-DD`，`end ≥ start` |
| `allow_public` | 否 | `false` | `target_student_id>0` 时强制为 `false` |
| `recognition_ritual` | 否 | `sticker` | `sticker` / `voice` / `class_star` |
| `star_scope` | 否 | `task` | `task` / `class` |
| `star_goal` | 否 | `0` | 必须 ≥0（负数 → 422「星星目标不能为负数」）；`class_star` + `task` 范围时必须 >0（编辑时仅在本次动到目标/仪式/范围时校验，历史任务改标题不会被挡住） |
| `activity_id` | — | — | **必须缺省**；传非空 → 422。只有 `/admin/task-activities` 能生成分组 |

- `star_scope=class` 时要求**当前学期**已有班级罐目标（`GET/PUT /admin/classes/:id/star-jar`），未配置 → **422**「请先配置班级星星罐」。
- 创建成功后按家长去重给目标家庭发站内消息（与老师端同一实现）；通知失败不回滚任务：逐收件人失败计入 `notify_failed_count`，**收件人查询整体失败**只记服务端日志、计数保持 `0`（不伪造失败数，与 B3 同口径）。
- 拒绝原因以中文 `message` 说明，统一包络 `code=42200`（不额外下发机器错误码）。

---

## PUT `/admin/tasks/:id`

> 编辑任务

```json
// Request
{
  "title": "每日阅读10分钟",
  "description": "…",
  "learning_link": "延续本周班级的春天主题",
  "class_id": 1,
  "target_student_id": 0,
  "frequency": "daily",
  "start_date": "2026-03-19",
  "end_date": "2026-03-25",
  "status": 1,
  "allow_public": false,
  "recognition_ritual": "sticker",
  "star_scope": "task",
  "star_goal": 20
}
```

`status`：`0` 已结束，`1` 进行中；**缺省表示不改**。

- `star_goal` 必须 ≥0，负数 → **422**「星星目标不能为负数」（只校验本次请求给的值，历史负值不会挡住改标题）。
- `target_student_id` / `allow_public` / `star_goal` / `description` 用**指针**语义：缺省 = 保留原值。老客户端不传 `allow_public` 不会把公开悄悄关掉并批量撤回，不传 `description` 也不会清空已有说明。
- `learning_link` 缺省或空 = 保留原值；原值也为空 → 422「学习关联必填」。
- `recognition_ritual` / `star_scope` 空串 = 保留原值。
- **有提交后**只允许 `title` / `description` / `learning_link` / `allow_public`（`status` 不受限）：
  `frequency`、起止日期、`class_id`、`target_student_id`、`star_scope`、`star_goal` 任一变化 → **422**
  「任务已有提交，只能修改标题、说明、学习关联与公开开关」（`TASK_STRUCTURAL_LOCKED`）。
- `target_student_id` 从 0 变为个体，或 `allow_public` 由 true 变 false → 同一事务内撤销该任务全部公开状态
  （清确认/审核人、写 `public_revoked_at`），与老师端 `UpdateTask` 同一 `revokePublicForTask`。
- `star_scope` 更新为 `class` 时同样要求当前学期班级罐目标，未配置 → 422。
- **活动任务（`activity_id` 非空）只允许 `title` / `description` / `learning_link` / `allow_public`**：
  频率、起止日期、班级、个体目标、仪式、星星范围/目标、`status` 任一变化 → **422**
  「活动任务只能修改标题、说明、学习关联与公开开关，请使用活动端点整组编辑」。
  与「有提交后的结构锁」独立：活动任务即使还没有提交也不允许单条改结构。
- 锁等待超时 / 死锁 → **503**。

---

## GET `/admin/tasks`

> 全园任务列表

```json
// Query: ?class_id=&status=&page=1&page_size=20

// Response
{
  "list": [
    {
      "id": 1,
      "title": "每日阅读10分钟",
      "class_name": "小一班",
      "class_id": 1,
      "learning_link": "延续本周班级的春天主题",
      "target_student_id": 0,
      "frequency": "daily",
      "frequency_label": "每日任务",
      "start_date": "2026-03-19",
      "end_date": "2026-03-25",
      "status": 1,
      "allow_public": false,
      "recognition_ritual": "sticker",
      "star_scope": "task",
      "star_goal": 20,
      "activity_id": "",
      "creator": { "nickname": "李老师" },
      "created_at": "2026-03-18T10:00:00Z"
    }
  ],
  "total": 80,
  "page": 1,
  "page_size": 20
}
```

- `points` 已移除。
- `activity_id` 为空串表示独立任务。
- **不含** `completion_rate` / `total_possible` / `checkin_count`（7.4 下线）；每任务口径请用
  `GET /admin/tasks/:id`（单任务）或 `GET /admin/tasks/stats`（按班级/活动聚合）。

---

## GET `/admin/tasks/:id`

> 任务详情

```json
{
  "id": 1,
  "title": "每日阅读10分钟",
  "description": "每天陪孩子阅读10分钟绘本",
  "class_id": 1,
  "class_name": "小一班",
  "learning_link": "延续本周班级的春天主题",
  "target_student_id": 0,
  "frequency": "daily",
  "frequency_label": "每日任务",
  "start_date": "2026-03-19",
  "end_date": "2026-03-25",
  "status": 1,
  "allow_public": false,
  "recognition_ritual": "sticker",
  "star_scope": "task",
  "star_goal": 20,
  "activity_id": "",
  "submission_count": 120,
  "responded_count": 96,
  "pending_count": 24,
  "response_rate": "80%",
  "participation_rate": "86%",
  "notify_target_count": 21,
  "creator": { "id": 1, "nickname": "李老师", "avatar": "..." },
  "created_at": "2026-03-18T10:00:00Z"
}
```

- 不返回 `points`。
- **已下线** `checkin_count` / `total_possible` / `completion_rate`（7.4）。替代口径：
  - `submission_count` = 该任务全部未删除提交数（不按时间窗过滤，任务详情是累计视角）；
  - `responded_count` = 其中主回应已齐（`response_id <> 0`）的提交数；
  - `pending_count` = `submission_count - responded_count`（主回应待回应）；
  - `response_rate` = `responded_count / submission_count`，无提交 → `null`；
  - `participation_rate` = 至少有一条提交的目标孩子数 / 目标孩子数（个体任务分母 1，班级任务为班级在读学生数），无目标孩子 → `null`；
  - 比率一律是 `"<整数>%"` 字符串且封顶 `"100%"`，无法计算时为 `null`（不是 `"0%"`）。
- `notify_target_count`：发布时按家长去重的**通知义务集**大小；`null` = 未知（收件人查询失败或历史任务）。
  它衡量「该告诉多少家长」，不是「成功送出多少条」。

---

## DELETE `/admin/tasks/:id`

> 删除任务（管理员强制删除；级联删除提交/回应/读标记/状态）

```json
{}
```

- **活动任务（`activity_id` 非空）不能单条删除** → **422**「活动任务不能单独删除，请删除整个活动」。
  多班活动用 `DELETE /admin/task-activities/:activity_id` 整组删除。
- 删除为软删除任务 + 级联软删提交/回应，硬删读标记/频率状态/补充/反馈（D14）。

---

## GET `/admin/tasks/:id/checkins`

> 查看某任务的公开内容（7.3 的「查看」入口）+ 撤回所需的 `checkin_id`

```
// Query: ?status=public|revoked|all   （缺省 public）
//        &page=1&page_size=20        （缺省 1 / 20，page_size 上限 100）
```

```json
// Response（HTTP 200）
{
  "list": [
    {
      "checkin_id": 42,
      "task_id": 1,
      "student_id": 7,
      "student_name": "小明",
      "content": "今天读了绘本",
      "media_url": "https://...",
      "created_at": "2026-03-20T12:00:00Z",
      "is_public_requested": true,
      "public_scope_confirmed": true,
      "public_approved_at": "2026-03-21T09:00:00Z",
      "public_approved_by": 3,
      "public_revoked_at": null,
      "public_status": "approved",
      "public_visible": true
    }
  ],
  "total": 1,
  "page": 1,
  "page_size": 20
}
```

- **分页**：`total` 是过滤后的总数，不是本页条数。`status=all` 对学期长的每日任务
  可能一次命中几千行，必须分页；`page_size` 超过 100 会被压到 100。
- `status=public`（缺省）：`is_public_requested = true` 的提交，即 `public_status` 为 `pending` 或 `approved`；
  `status=revoked`：`is_public_requested = false AND public_revoked_at IS NOT NULL`；`status=all`：不过滤。
  其他取值 → **422**「status 须为 public / revoked / all」。
- `public_status` / `public_visible` 由服务端计算（复用公开墙同一套条件），客户端不得自己拼：
  `public_visible=true` 表示**此刻**确实在班级公开墙上可见（任务已结束、任务关闭公开、孩子转班、
  任务改为个体任务都会让它变 false，即使仍处于已批准状态）。
- 只读，不改变任何公开状态；撤回请用 `PUT /admin/tasks/checkins/:checkin_id/public`。
- 任务不属于当前园所 → **404**。权限码：`task:view`。

---

## PUT `/admin/tasks/checkins/:checkin_id/public`

> 管理员强制撤回一条已公开（或待审核）的提交

```json
// Request
{ "revoke": true }

// Response（HTTP 200）
{
  "checkin_id": 42,
  "task_id": 1,
  "is_public_requested": false,
  "public_scope_confirmed": false,
  "public_approved_at": null,
  "public_approved_by": 0,
  "public_revoked_at": "2026-03-21T09:00:00Z",
  "public_status": "revoked",
  "public_visible": false
}
```

- 只有 `revoke=true` 是合法请求；缺省或 `false` → **422**「仅支持撤回操作」（管理端不得代替家长发起申请，也不得绕过老师审核直接批准）。
- 撤回后清空 `public_scope_confirmed` / `public_approved_at` / `public_approved_by`，写 `public_revoked_at`，置 `is_public_requested=false`；公开墙下一次读取即不可见。
- 提交不属于当前园所 → **404**；行锁等待超时/死锁 → **503**。
- 撤回动作写操作日志（`module=tasks`、`action=revoke_public`、`target_type=task_check_in`）。
- 权限码：`task:manage`。

---

## GET `/admin/tasks/stats`

> 任务统计（新口径，7.2）

```json
// Query: ?start_date=2026-03-01&end_date=2026-03-31

// Response
{
  "period": { "start": "2026-03-01", "end": "2026-03-31" },
  "total_tasks": 25,
  "active_tasks": 8,
  "completed_tasks": 17,
  "total_checkins": 3200,
  "response_rate": "75%",
  "participation_rate": "88%",
  "pending_count": 800,
  "pending_over_48h": 120,
  "median_response_hours": 5.5,
  "median_response_negative_excluded": 1,
  "feedback_coverage": "42%",
  "child_happy_rate": "18%",
  "ack_rate": "24%",
  "notify_baseline_missing": 3,
  "by_class": [
    {
      "class_id": 1,
      "class_name": "小一班",
      "tasks": 5,
      "total_checkins": 600,
      "response_rate": "80%",
      "participation_rate": "90%",
      "pending_count": 120,
      "pending_over_48h": 10,
      "median_response_hours": 4.5,
      "median_response_negative_excluded": 0,
      "feedback_coverage": "40%",
      "child_happy_rate": "20%",
      "ack_rate": "20%",
      "notify_baseline_missing": 0
    }
  ],
  "by_activity": [
    { "activity_id": "act_1793...", "task_id": 0, "title": "春天阅读周", "tasks": 2, "...": "同 by_class 的指标字段" },
    { "activity_id": "", "task_id": 31, "title": "每日阅读10分钟", "tasks": 1, "...": "同上" }
  ]
}
```

**统计范围**：与 `[start_date, end_date]` 有交集的任务（`parent_tasks` 区间重叠），
且**提交/回应只统计 `task_check_ins.date` 落在该区间内的行**。默认区间为当月。

**字段口径**

| 字段 | 定义 | 无法计算时 |
| --- | --- | --- |
| `total_checkins` | 区间内未删除提交数 | `0` |
| `response_rate` | 有主回应的提交数 / 总提交数（`response_id <> 0`） | `null` |
| `participation_rate` | 区间内**至少有一条提交的目标孩子去重数** / 范围内任务覆盖的目标孩子去重数（个体任务计 1，班级任务计班级在读学生） | `null` |
| `pending_count` | 主回应待回应（`response_id = 0`）的提交数 | `0` |
| `pending_over_48h` | 上述提交中 `created_at < now - 48h` 的数量 | `0` |
| `median_response_hours` | 仅 `response_type=single` 主回应的「提交→回应」时长中位数；偶数取两中值平均，四舍五入 1 位小数 | `null` |
| `median_response_negative_excluded` | 因时长为负被剔除、未参与中位数的样本数 | `0` |
| `feedback_coverage` | 去重 `(response_id, parent_user_id)` 反馈对数 / 分母 | `null` |
| `child_happy_rate` | 其中 `child_happy` 的对数 / 分母 | `null` |
| `ack_rate` | 其中 `received`+`thanks` 的对数 / 分母（家长确认，与质量信号分开） | `null` |
| `notify_baseline_missing` | 参与反馈聚合、但 `notify_target_count IS NULL` 的**回应数**（已从分母排除） | `0` |

**参与率边界**：分子只算仍在目标范围内的提交——孩子提交后转班/离园，其历史提交
仍留在库里，但不能再让参与率超过目标孩子数。分母是去重后的学生集合，不是
「每任务比例求平均」，也不是家长数。

**反馈类分母（decision §20.10-6）**：不是 `messages` 行数，也不是实时 join `student_parents`，
而是**回应发出那一刻确定的义务集** `task_responses.notify_target_count`（按家长去重，
发送前先落库）。`NULL` 表示义务未知，**排除**而不是当 0。
聚合起点是 `task_check_ins.response_id`：补充回应同样是 `TaskResponse`，但不被任何提交引用，
因此天然不进分母（B1 已定「补充不计入」）。

**`period` 回应**：计入 `response_rate` / `pending_count`（覆盖即算已回应），但**不进**中位数
（中位数只算逐条 `single`）。补充回应两个指标都不进。

**比率**：一律为 `"<整数>%"` 且封顶 `"100%"`（分母小于分子时截断，例如通知义务集
小于实际反馈家长数）；`null` 表示没有可计算的分母，前端显示「—」。

**`by_activity` 分组**：`activity_id` 非空按活动聚合（`task_id=0`）；`activity_id` 为空的
**每个独立任务各占一条**（`task_id` 为该任务 id），不得合并成一个「空活动」。

- 已下线 `avg_completion_rate` / `by_class[].avg_rate`（7.4）。
- `total_points_awarded` 已移除。
- 权限码：`task:view`。

---

## POST `/admin/task-activities`

> 多班活动：一个活动 = 每个班级一条任务，共享服务端生成的 `activity_id`（design D12）

```json
// Request
{
  "title": "春天阅读周",
  "description": "亲子共读三本绘本",
  "learning_link": "配合本周春天主题",
  "frequency": "daily",
  "start_date": "2026-03-19",
  "end_date": "2026-03-25",
  "allow_public": true,
  "recognition_ritual": "sticker",
  "star_scope": "class",
  "star_goal": 0,
  "class_ids": [1, 2]
}

// Response（HTTP 201）
{
  "activity_id": "act_1793260800123456789_7f3a9c2b1d4e5f60718293a4",
  "task_ids": [11, 12],
  "notified_count": 24,
  "notify_failed_count": 0
}
```

| 字段 | 必填 | 默认 | 校验 |
| --- | --- | --- | --- |
| `title` | 是 | — | ≤200，纯空白 → 422 |
| `description` | 否 | `""` | — |
| `learning_link` | **是** | — | 空 → 422「学习关联必填」 |
| `class_ids` | **是** | — | 非空、**不得重复**、**≤50**、全部属于当前园所；任一不满足 → 422 |
| `frequency` / `start_date` / `end_date` | 是 | — | 同 `POST /admin/tasks` |
| `allow_public` | 否 | `false` | 活动没有个体目标，可自由开启 |
| `recognition_ritual` | 否 | `sticker` | `sticker` / `voice` / `class_star` |
| `star_scope` | 否 | `task` | `task` / `class`；`class` 时**每个班级**都要有当前学期班级罐目标 |
| `star_goal` | 否 | `0` | ≥0；`class_star` + `task` 范围时 >0 |
| `activity_id` | — | — | **必须缺省**，传非空 → 422（由服务端生成，客户端不得指定） |

- `activity_id` 形如 `act_<unixnano>_<24位随机hex>`，服务端生成、不可猜、不连续。
- **事务性**：班级归属校验（对 `classes` 行加 `FOR SHARE`）、逐班结构校验与建任务
  在同一事务内进行——并发软删的班级会让整组失败，而不是留下一条在列表/统计里
  永远看不到的任务。任一班级（含班级罐未配置）失败整组回滚，不留半组任务。
- 创建后逐任务给目标家庭发通知（与 `POST /admin/tasks` 同一实现）；响应里的
  `notified_count` / `notify_failed_count` 是**各任务之和**：同一家长如果有两个孩子在
  两个班级，会收到两条消息（对应两条任务），计 2 次。
- 活动任务没有个体目标，`target_student_id` 恒为 0。
- 权限码：`task:manage`。

---

## PUT `/admin/task-activities/:activity_id`

> 批量编辑活动的**安全字段**（标题/说明/学习关联/公开开关）与整组结束状态（`status`）

```json
// Request（只发要改的字段）
{
  "title": "春天阅读周（第 2 版）",
  "allow_public": false
}

// Response（HTTP 200）
{ "activity_id": "act_...", "updated_tasks": 2 }
```

- 只允许 `title` / `description` / `learning_link` / `allow_public` / `status`；缺省 = 不改。
- `status`：`0` 已结束 / `1` 进行中。这是活动**唯一的结束路径**——`status` 不是结构字段
  （C-1 口径），所以有提交后仍可整组结束；单条 `PUT /admin/tasks/:id` 对活动任务仍拒绝
  `status`，避免一个活动里部分班级已结束、部分还在进行。
- `updated_tasks` = 本活动下的任务数（本次覆盖到几条），不是「值真的变化了」的条数；
  重复提交同一份修改也返回同样的数字。
- **显式传入任何非空结构字段**（`frequency` / `start_date` / `end_date` / `class_ids` /
  `target_student_id` / `recognition_ritual` / `star_scope` / `star_goal` /
  `activity_id`）→ **422**「活动只能批量修改标题、说明、学习关联、公开开关与结束状态」，
  即使取值与当前相同也拒绝（避免调用方误以为改生效了）。
- 空值口径按字段类型区分：
  - **字符串 / 数组**：空串、`class_ids: []` 视为「未提供」，不报错也不会删班
    （客户端整表单回传时结构位是空的，不该因此 422）。
  - **数值 / 指针**（`star_goal` / `target_student_id`）：**只要出现非 `null` 的键就拒绝，
    `0` 也算出现**——这两个字段的 `0` 都是合法业务值（0 星目标、非个体任务），无法区分
    「回传当前值」与「想改成 0」。显式 `null` 与缺省同义（不拒绝）。客户端整表单回传时
    应省略这两个键。`status` 不在此列（见上，它是允许改的）。
- `allow_public` 由 true 变 false → 同一事务内撤销该活动**全部任务**的公开状态
  （清确认/审核人、写 `public_revoked_at`），与单任务编辑同一 `revokePublicForTask`。
- `learning_link` 传空串按「未提供」处理（保留原值），与单任务编辑同口径。
- 活动不存在（该园所下没有任何该 `activity_id` 的任务）→ **404**。
- 加锁顺序：按任务 id 升序锁 `parent_tasks` 行，再动 `task_check_ins`，与单任务/删除同一锁序。
  锁等待超时 / 死锁 → **503**。
- 权限码：`task:manage`。

---

## DELETE `/admin/task-activities/:activity_id`

> 整组删除活动（事务级联）

```json
// Response（HTTP 200）
{ "activity_id": "act_...", "deleted_tasks": 2 }
```

- 每个任务走与 `DELETE /admin/tasks/:id` 相同的级联（软删任务/提交/回应，硬删读标记/
  频率状态/补充/反馈），全部在同一事务内；中途失败整组回滚（不会出现「删了一半」）。
- 活动不存在 → **404**。权限码：`task:manage`。

---

## GET `/admin/tasks/stats` 与活动的关系

- 活动的统计以 `activity_id` 聚合，见上方 `by_activity`；独立任务按任务 id 各自成组。

---

## GET `/admin/classes/:id/star-jar`

> 读取班级**当前学期**的班级星星罐目标

```json
// Response（HTTP 200）
{
  "class_id": 1,
  "term": "2031-2032-spring",
  "term_start": "2031-01-21",
  "term_end": "2031-07-10",
  "goal": 30,
  "configured": true,
  "updated_at": "2026-10-23T09:00:00Z"
}
```

- `term` = `<classes.academic_year>-<当前学期>`；班级无学年时用 `<自然年>-<学期>`。
- 学期取 `kindergarten_app_settings.current_semester`（`spring` / `fall`，非 `fall` 按 `spring`）。
- 起止口径分三种：
  1. `academic_years` 里有该学年、且对应学期起止可解析 → 用配置值；
  2. 有该学年但未配置起止 → 自然学期兜底（`spring` = 次年 2/1–7/31，`fall` = 当年 9/1–次年 1/31）；
  3. **班级没有学年**（或学年不是 `YYYY-YYYY`、起止年倒序）→ 自然年按学期切分（`spring` = 1/1–7/31，`fall` = 8/1–12/31）。
- **两学期无缝且不重叠**（与老师端同一口径，2026-10-24）：按起点排序后，后一个学期的起点 = 前一个结束的次日。配置里的暑假/寒假空档会被收口（暑假归 `fall`、寒假归 `spring`）；`term` 键不变。示例里 `term_start` 之所以是 `2031-01-21`，是因为该学年的 fall 结束于 `2031-01-20`。
- 未配置 → `goal=0`、`configured=false`、`updated_at=null`（仍返回 `term`）。
- 班级不属于当前园所 → **404**。权限码：`task:view`。

## PUT `/admin/classes/:id/star-jar`

```json
// Request
{ "goal": 30 }

// Response（HTTP 200）
{ "class_id": 1, "term": "2031-2032-spring", "term_start": "2031-01-21", "term_end": "2031-07-10", "goal": 30, "configured": true, "updated_at": "2026-10-23T09:01:00Z" }
```

- 目标必须 **> 0**；`0` / 负数 → **422**「班级星星罐目标必须大于 0」（`STAR_JAR_GOAL_INVALID`）。
- 按 `(kindergarten_id, class_id, term)` upsert；命中软删行会复活并清 `deleted_at`。
- 写 `operation_logs`（`module=tasks`、`action=set_star_jar`、`target_type=class_star_jar`）。
- 权限码：`task:manage`。

- 任务 `star_scope=class` 只认**当前学期** `goal>0`；上学期配置不会放行本学期任务。
- 任务 `star_scope=class` 时忽略任务 `star_goal`，一律使用班级罐目标。
