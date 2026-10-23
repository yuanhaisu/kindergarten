# 10. 亲子任务管理

> 对应前端页面：`admin/pages/tasks/index.vue`
>
> 基础路径：`/api/v1/admin`，需 `Authorization: Bearer <token>`，角色 `admin`。
>
> **C-1 状态**：`points` 已从 domain/契约移除；建/改任务的结构字段（学习关联、范围、个体、公开、仪式、星星）与班级星星罐配置**已实现**（C-1，2026-10-23）。多班活动端点、新口径统计仍标 `[C]`，属 C-2。

---

## POST `/admin/tasks`

> 新建**独立任务**（`creator_id` 取当前登录用户）。多班活动请用 `POST /admin/task-activities`（`[C]`）。

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
      "completion_rate": "57%",
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
- `[C]` 旧 `completion_rate` 将下线，改为参与率/回应率（C-2）。

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
  "completion_rate": "57%",
  "checkin_count": 120,
  "total_possible": 210,
  "creator": { "id": 1, "nickname": "李老师", "avatar": "..." },
  "created_at": "2026-03-18T10:00:00Z"
}
```

- 不返回 `points`。
- `checkin_count` / `total_possible` / `completion_rate` 是旧口径，C-2 第 7.4 条统一下线。

---

## DELETE `/admin/tasks/:id`

> 删除任务（管理员强制删除；级联删除提交/回应/读标记/状态）

```json
{}
```

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

> 任务统计

```json
// Query: ?start_date=2026-03-01&end_date=2026-03-31

// Response
{
  "period": { "start": "2026-03-01", "end": "2026-03-31" },
  "total_tasks": 25,
  "active_tasks": 8,
  "completed_tasks": 17,
  "total_checkins": 3200,
  "avg_completion_rate": "62%",
  "by_class": [
    { "class_name": "小一班", "tasks": 5, "avg_rate": "68%" }
  ]
}
```

- `total_points_awarded` 已移除。
- `[C]` 后续新增参与率、回应率、`pending_over_48h`、反馈率、按活动统计。

---

## GET `/admin/classes/:id/star-jar`

> 读取班级**当前学期**的班级星星罐目标

```json
// Response（HTTP 200）
{
  "class_id": 1,
  "term": "2031-2032-spring",
  "term_start": "2031-02-20",
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
  3. **班级没有学年**（或学年不是 `YYYY-YYYY`、起止年倒序）→ 自然年 1/1–12/31。
- 未配置 → `goal=0`、`configured=false`、`updated_at=null`（仍返回 `term`）。
- 班级不属于当前园所 → **404**。权限码：`task:view`。

## PUT `/admin/classes/:id/star-jar`

```json
// Request
{ "goal": 30 }

// Response（HTTP 200）
{ "class_id": 1, "term": "2031-2032-spring", "term_start": "2031-02-20", "term_end": "2031-07-10", "goal": 30, "configured": true, "updated_at": "2026-10-23T09:01:00Z" }
```

- 目标必须 **> 0**；`0` / 负数 → **422**「班级星星罐目标必须大于 0」（`STAR_JAR_GOAL_INVALID`）。
- 按 `(kindergarten_id, class_id, term)` upsert；命中软删行会复活并清 `deleted_at`。
- 写 `operation_logs`（`module=tasks`、`action=set_star_jar`、`target_type=class_star_jar`）。
- 权限码：`task:manage`。

- 任务 `star_scope=class` 只认**当前学期** `goal>0`；上学期配置不会放行本学期任务。
- 任务 `star_scope=class` 时忽略任务 `star_goal`，一律使用班级罐目标。

## `[C]` 活动与统计（未实现，C-2）

- `POST/PUT/DELETE /admin/task-activities`（服务端生成 `activity_id`；`class_ids` 必填、去重、≤50）
- `GET /admin/tasks/:id/checkins?status=public`（7.3 的「查看」入口；撤回端点已实现）
- `GET /admin/tasks/stats` 新口径（参与率/回应率/`pending_over_48h`/反馈率/按活动）
