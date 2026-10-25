# Design

## Context

See `proposal.md` — Why.

现状约束：

- `ParentTask` 只有班级范围（`class_id`），没有个体目标；`TaskCheckIn` 只有打卡内容与 `points`，没有老师回应。
- 家长端任务列表只查 `status=1`，不校验 `start_date/end_date`；打卡只校验任务 `status=1`，不校验班级、日期、频率。
- 家长端列表不传 `student_id`，服务端默认第一个孩子；打卡却用 `CurrentStudent`，多孩家庭会串孩子。
- 老师端 `tasks/index.vue` 只有创建表单（标题/说明）与任务列表，没有调用已有的 `stats` / `checkins` 接口。
- 上传通道 `POST /common/upload` 已支持 image/video/audio，`biz_type` 已有 `task`，语音可直接复用。
- 站内消息表 `messages` 已有 `task` 类型，但任务创建与回应都没有投递。
- 现有积分 `PointService` 的 `task_checkin_points` 规则没有任何写入方；打卡只写 `TaskCheckIn.Points`，不写 `PointTransaction` / `PointAccount`。
- 家长端首页 `HomeOverview` 已返回 `pending_tasks`，但统计的是「进行中任务数」，不是「当前孩子还没提交的任务数」，页面也没有使用。
- 老师端与家长端是两个不同 AppID；`WeChatClient.GetWXACodeUnlimit` 固定走老师端凭据，不能直接生成家长端小程序码。

## Goals / Non-Goals

**Goals**

- 建立「发布 → 提交 → 老师内部回应 → 家长质量反馈 → 孩子感知」的闭环。
- 支持个体任务与班级任务；默认私密，班级活动经老师审核后公开。
- 老师回应成为园所内部管理要求，但不在家长端暴露「待回应」。
- 习惯任务用周期汇总回应，避免每条提交都单独回应。
- 星星罐支持按任务、按班级两种范围。
- 从任务流程移除积分，修正多孩、跨班、过期、进度、字段与软删除问题。
- 管理端可创建活动、查看参与与回应统计。
- 老师可生成家长端小程序码/任务海报，转发到班级微信群。

**Non-Goals**

- 不做积分/排行榜/成就/礼物商城。
- 不做成长时光自动沉淀、全园活动墙、模板库。
- 不做微信订阅消息；P0 用站内消息 + 首页卡片 + 老师群海报/小程序码。
- 不做教师回应超时自动升级、SLA。
- 不做媒体文件物理删除与清理任务（P0 只做归属校验与软删除）。

## Decisions

### D1：个体/班级范围用 `target_student_id` 表达

- `ParentTask.TargetStudentID = 0` 表示班级任务；`> 0` 表示个体任务。
- 个体任务的可见性、可提交性、可回应性都以 `target_student_id` 为准；班级任务以 `class_id` 为准。
- 创建与更新为个体任务时都必须强制 `AllowPublic=false`，并撤销已有公开状态；不能只在校验创建路径。
- 不新增 `scope` 枚举；避免在 `design-data/`、`dict.go`、现有常量都没有来源的情况下臆造取值。

### D2：习惯/活动复用 `frequency`

- `daily` / `weekly`：习惯类，允许周期内多次提交，老师按周期汇总回应。
- `once`：一次性活动或个别支持，老师逐条回应。
- 个体任务优先按 `target_student_id > 0` 判定为逐条回应，不受 `frequency` 影响。

### D3：老师回应是内部闭环，家长端不显示待回应

- 新增 `task_responses` 表：一次回应可以覆盖一条或多条提交。
- `TaskCheckIn.ResponseID` 保存“主回应”，用于待回应判定；主回应只由该字段表达，不再写 link 表。
- 新增 `task_response_supplements(response_id, checkin_id)` 只存补充回应关系，唯一 `(response_id, checkin_id)`。
- `TaskCheckIn.ResponseID = 0` 表示待回应；非 0 表示已有主回应。
- 家长端状态只有「待完成 / 已提交 / 已回应」，**不显示「等待老师回应」**。
- 老师端有内部「待回应」队列；家长端不暴露内部管理状态。
- 回应类型：
  - `single`：对一条提交回应。
  - `period`：对某个周期内全部待回应提交做一次汇总回应（习惯任务）。
- 批量/周期回应写入一条 `TaskResponse`，把该批次内待回应提交的 `ResponseID` 指向它；部分失败返回失败 ID，不得静默成功。
- 汇总回应之后，老师仍可通过 `task_response_supplements` 给单个孩子追加补充贴纸/语音；补充回应不改变 `ResponseID`，家长端按“主回应 + 补充回应”展示；补充回应的 `response_type` 固定为 `single`；只支持贴纸/语音，`class_star` MUST 被拒绝；未显式指定仪式时默认贴纸，MUST NOT 继承任务默认仪式。
- 回应不可由家长编辑；老师撤回回应不在 P0。

### D4：回应质量信号

- 家长可对老师回应做一键反馈：`child_happy` / `received` / `thanks`；`child_happy` 是孩子感知质量信号，`received` / `thanks` 是家长确认。
- 新增 `task_response_reactions(response_id, user_id, reaction, created_at, updated_at)`，唯一 `(response_id, user_id)`；`TaskResponse` 不再持有单个 `ParentReaction`。
- 每位家长只能对自己的回应反馈；一个班级汇总回应可被多个家庭分别反馈；同一家长可更新自己的反馈。
- 主回应指标按 `(response_id, parent_user_id)` 对计算，不能按“回应数”做分母：一个班级汇总回应可覆盖多个家长，否则 20 个家长反馈会让比率超过 100%。
- 应反馈对数 = 每个主回应覆盖的去重家长数之和（`notified_count + notify_failed_count`，不能只算成功通知）；`feedback_coverage` = 有反馈对数 / 应反馈对数；`child_happy_rate` = `child_happy` 反馈数 / 应反馈对数；`ack_rate` = (`received` + `thanks`) / 应反馈对数。
- 补充回应单独统计，不进入上述三个指标；家长可对补充回应反馈，但不重复计入主回应指标。
- 不做家长评分/星级打分，避免把家园沟通变成服务评价。

### D5：三种回应仪式

| 仪式 | 老师动作 | 存储 | 家长/孩子感知 |
| --- | --- | --- | --- |
| 贴纸 | 点选一枚贴纸 | `sticker_key`（客户端常量，服务端限长） | 大贴纸卡片 |
| 语音 | 录 5–15 秒语音并上传 | `media_file_id`（`/common/upload`，`biz_type=task`） | 播放老师语音 |
| 班级星星 | 一键加星 | `TaskResponse.Ritual=class_star` | 星星罐进度 |

- 任务创建时选默认仪式；老师回应时可调整。
- 贴纸键值不建服务端枚举；服务端限制长度与字符集，客户端负责渲染。
- 语音时长上限 15 秒、大小上限沿用 `/common/upload` 的 20MB；超限由客户端拦截。

### D6：星星罐支持按任务、按班级两种范围

- `ParentTask.StarScope`：`task` | `class`，默认 `task`。
- `task`：星星罐属于该任务；进度 = 该任务中**主回应为 `class_star` 的提交数** / `StarGoal`（period 回应覆盖 N 条提交就加 N 星）。
- `class`：星星罐属于班级当前学期；新增 `class_star_jars` 表保存 `(kindergarten_id, class_id, term, goal)`；进度 = 该班当前学期内所有 `StarScope=class` 任务中**主回应为 `class_star` 的提交数** / goal。
- 计数 join `TaskCheckIn` 与 `TaskResponse` 时必须过滤两者的 `DeletedAt IS NULL`；已删除提交/回应不计入。
- 星星罐进度展示封顶 100%；目标下调到低于当前星星数时显示“已超额”，不出现 >100% 的进度条。
- 学期口径：`Class.AcademicYear` + `KindergartenAppSetting.CurrentSemester`；为空时用自然年。
- 班级星星罐目标独立配置：新增 `GET/PUT /teacher/class/star-jar`（按学期）与 `GET/PUT /admin/classes/:id/star-jar`；两者都校验园所与班级归属，目标必须 >0，0 视为非法。任务 `StarScope=class` 时忽略任务 `StarGoal`，只使用班级罐目标。
- 创建 `StarScope=class` 任务时必须已有班级罐目标；未配置时拒绝创建并返回“请先配置班级星星罐”的引导；teacher/admin/activity 三个创建入口必须共用同一 service 校验，不允许各写一套；该只读校验 helper 放在 `server/internal/service/task_rules.go`，A 切片先实现（2.9），C 的配置端点（3.8）复用，避免 A 依赖 C；UI 在 class scope 下隐藏任务 `StarGoal`，未配置时不允许选班级星星仪式。
- 只统计**主回应**中实际仪式为 `class_star` 的回应；贴纸/语音回应不计入；补充回应不计入星星罐。
- `StarGoal=0` 表示按任务星星罐不启用；任务选择班级星星仪式且 `StarScope=task` 时要求目标 > 0，否则创建/回应被拒绝。
- 不展示个人星星排名；班级罐只展示集体进度。

### D7：公开墙必须老师审核

- `ParentTask.AllowPublic`：任务是否允许申请公开；默认 `false`；个体任务强制 `false`。
- `TaskCheckIn.IsPublicRequested`：家长申请公开；`PublicScopeConfirmed`：老师勾选“内容中只有提交者孩子”；`PublicApprovedAt/By`：老师审核；`PublicRevokedAt`：撤回。
- 可见条件：`ParentTask.AllowPublic && ParentTask.DeletedAt IS NULL && ParentTask.Status=1 && ParentTask.EndDate >= 今天 && ParentTask.ClassID = 当前孩子班级 && IsPublicRequested && PublicScopeConfirmed && PublicApprovedAt != nil && PublicRevokedAt == nil`；任务结束后不再展示，跨班不可见；批准时 `PublicScopeConfirmed` 必须为 true，并记录审核人/时间，便于追责。
- 任务 `AllowPublic` 改为 false 时，在同一事务内批量清空该任务提交的 `PublicApprovedAt` / `PublicScopeConfirmed`、`IsPublicRequested=false`，并写 `PublicRevokedAt`；重新打开 `AllowPublic` 不得恢复旧批准，家长必须重新申请、老师重新确认。
- 状态转换：
  - 申请：`IsPublicRequested=true`，`PublicScopeConfirmed=false`，`PublicApprovedAt=nil`，`PublicRevokedAt=nil`。
  - 批准：`PublicScopeConfirmed=true`，`PublicApprovedAt=now`，`PublicApprovedBy=teacher`。
  - 家长取消 / 老师撤回：`IsPublicRequested=false`，`PublicScopeConfirmed=false`，`PublicApprovedAt=nil`，`PublicApprovedBy=0`，`PublicRevokedAt=now`。
  - 再次申请：从私密重新进入待审核，必须重新批准并重新确认；不能沿用旧 `PublicApprovedAt` 或旧 `PublicScopeConfirmed`。
- 家长只能申请自己的提交；老师只能审核本任务/本班提交；管理端可查看与撤回。
- 老师审核时必须确认「内容中只有提交者孩子」（图片/语音/文字均适用）；含其他孩子的内容不得批准公开；公开墙只展示提交者孩子的内容，不做集体合照墙。
- 公开范围：本班家长可见（P0 不做全园墙）。
- 个体任务不提供公开入口；服务端拒绝个体任务的公开申请；更新后 `target_student_id>0` 时强制 `AllowPublic=false` 并调用 `revokePublicForTask`。
- 公开内容读取时实时过滤，不依赖缓存；撤回后下一次读取即不可见。

### D8：任务流程完全移除积分

- 任务创建/编辑/列表/详情不再接收或返回 `points`；家长端/老师端/管理端不再展示任务积分。
- 提交不写 `PointTransaction`、不改 `PointAccount`；`TaskCheckIn.Points` 与新表不再写入。
- `PointService.GetPointRules` 移除「任务打卡」规则；`UpdatePointRules` 不再接受该规则。
- `TaskService.GetTaskStats` 移除 `total_points_awarded`；`SumCheckinPointsInDateRange` 不再被任务统计使用。
- `parent_tasks.points` / `task_check_ins.points` 直接删除；开发阶段无历史数据，不做迁移/兼容。
- 积分页只保留积分体系自身的历史流水，不再有任务来源。

### D9：通知与触达

- 任务发布：给目标家庭写站内消息（`type=task`，`reference_id=task_id`），同一任务对同一家长至多一条；`notified_count` / `notify_failed_count` 按家长用户去重，由创建任务响应返回。
- 老师主回应：按 `TaskResponse` 聚合，同一个家长在一次回应中无论覆盖多少条提交，只收到一条消息（`reference_id=response_id`）；计数同样按家长去重。
- 补充回应不单独发通知；家长在任务详情内看到，任务列表/详情必须有未读标记，避免个别表扬被错过。发布通知计数按家长去重；回应通知计数按家长去重，且只统计主回应（不含 supplement）。
- 新增 `task_read_marks(kindergarten_id, task_id, user_id, student_id, last_read_at)`，唯一 `(task_id, user_id, student_id)`；`POST /parent/tasks/:id/read` 必须带 `student_id`，校验家长绑定、任务可见且未删除，返回 `{ last_read_at }`；读标记用 upsert（`ON DUPLICATE KEY UPDATE` / 捕获唯一冲突重试），并发不 500；未读 = 该孩子在该任务下存在补充回应且 `last_read_at IS NULL OR supplement.created_at >= last_read_at`（用 `>=` 避免同秒漏判）。
- `messages` 增加 `action_path`；任务发布/主回应消息写入 `/pages/tasks/detail?id=<task_id>&student_id=<student_id>`，家长端消息列表按 `action_path` 跳转；`reference_id` 继续用于幂等。
- `action_path` 只允许 `/pages/` 开头的内部路径；个体任务用目标孩子，班级任务用该家长在该班的一个确定孩子，回应消息用覆盖提交中的第一个孩子；家长端详情在“当前孩子不是消息目标孩子”时提示可切换孩子，避免多孩家庭误以为看错内容。
- 点击消息时若任务已删除/已结束，客户端显示“任务已结束/不存在”；未登录时保留 `action_path`，登录后回跳。
- 通知失败不回滚任务/回应；老师回应响应返回 `notified_count` / `notify_failed_count`。
- 老师端提供「家长端小程序码 / 任务海报」：新增 `WeChatClient.GetParentWXACodeUnlimit(ctx, scene, page, envVersion)`（走家长端凭据），生成 `pages/tasks/detail` 的小程序码；`envVersion` 读 `wechat.code_env_version`（默认 `release`，开发/测试用 `trial`/`develop`），`check_path=false`；`scene` 用 `t=<task_id>`（≤32 字符）；家长端 `onLoad` 解析 `scene`，缺省回退 query `id`。
- `notified_count` / `notify_failed_count` 按家长用户去重，不按提交条数；一次主回应覆盖同一家长多条提交只计一次。
- 不做微信订阅消息。
- 家长端首页增加「今晚一件事」卡片与任务 Tab 红点；`pending_tasks` 口径见 D16。

### D10：多孩与班级边界

- 家长端所有任务接口都传 `student_id`（列表、详情、提交、历史、公开申请、反馈）。
- 任务相关接口在 `student_id` 缺失/非法时明确拒绝，**不回退到第一个孩子**。
- 提交校验：任务属于该孩子的班级，或 `target_student_id == 该孩子`；否则拒绝。
- 老师端任务操作限定 `tc.ClassID`；创建或更新个体任务时校验目标孩子属于该班级与园所；回应/审核校验提交属于该任务与班级。
- 文案/进度/已提交状态都按该孩子计算。

### D11：有效期与滚动 7 天频率

- 任务有效：`status=1 AND start_date <= 今天 <= end_date`；查询时按有效状态过滤，不依赖定时任务。
- 过期任务不可提交；家长端归入「已结束」。
- `daily`：上海时区当天最多一次。
- `weekly`：**滚动 7 天**（今天往前 6 天，共 7 个自然日）最多一次，不是自然周。
- `once`：任务周期内最多一次。
- 新增 `task_checkin_states(task_id, student_id, last_check_in_date)`，唯一 `(task_id, student_id)`；`last_check_in_date` 必须可空（首次提交前为 NULL）；空库直接建表；已有 NOT NULL 列执行 `docs/migrations/2026-10-21-task-checkin-state-nullable.sql`。
- 提交在事务内 `SELECT ... FOR UPDATE` 该状态行（不存在则创建；并发创建冲突时回滚重试并锁定已存在行），按频率校验 `last_check_in_date`，写入提交后更新状态；`(task_id, student_id, date)` 唯一索引兜底同日重复。
- 不能只靠“先查后写”；`daily/weekly/once` 都必须走同一状态行串行化。
- 日期统一按 Asia/Shanghai；写入 `date` 使用上海时区当天零点。

### D12：管理端活动与统计

- 管理端任务页增加创建/编辑表单：标题、说明、`learning_link`、班级多选、频率、起止日期、仪式、星星范围/目标、允许公开。
- 多班活动由服务端生成 `activity_id`，新增 `POST /admin/task-activities` 创建（共享字段 + 班级列表），`PUT /admin/task-activities/:activity_id` 批量编辑，`DELETE /admin/task-activities/:activity_id` 整组删除；批量编辑只允许标题/说明/学习关联/公开开关，不允许改频率/日期/班级。
- `POST /admin/task-activities` 请求形状：标题/说明/`learning_link`/频率/起止日期/仪式/星星范围与目标/`allow_public` + `class_ids`；`class_ids` 必填、去重、最多 50 个，全部属于当前园所；空列表或重复班级按校验失败处理。
- 活动端点必须校验所有班级属于当前园所；创建/删除使用事务，一个班级失败整组回滚；活动任务禁止走单条 `DELETE /admin/tasks/:id`，单条 `PUT /admin/tasks/:id` 也只允许安全字段或直接拒绝。
- `POST /admin/tasks` 强制忽略/拒绝客户端传入的 `activity_id`；只有 `/admin/task-activities` 能设置活动分组。
- 任务产生第一条提交后，编辑只允许标题/说明/`learning_link`/`allow_public`；`frequency`、起止日期、`target_student_id`、`class_id`、`star_scope`、`star_goal` 一律拒绝修改。要改结构必须新建任务。
- `TeacherPortalService.CreateTask` 与 `TaskService.CreateTask`（`POST /admin/tasks` 独立任务）必须持久化 `AllowPublic` / `RecognitionRitual` / `StarScope` / `StarGoal`；`AllowPublic` 默认 false，个体任务强制 false；`StarScope=class` 创建走 3.8 的统一班级罐校验，不能只依赖 DTO 或 UI。
- 更新规则必须同时落在 `TeacherPortalService.UpdateTask` 与管理端 `TaskService.UpdateTask`；不能只做 UI 或只做管理端。`allow_public=false` 的批量撤销也在两个更新入口共用。
- 共享规则统一放在 `server/internal/service/task_rules.go`：`validateTargetStudent` / `validateClassStarJar` / `assertTaskEditable` / `revokePublicForTask`；两个 service 只做权限、DTO、事务编排，不复制规则。
- `TaskService.UpdateTask` 与 `TeacherPortalService.UpdateTask` 在 `star_scope` 变为 `class` 时也必须调用 `validateClassStarJar`，不能只校验创建路径；在 `target_student_id` 变为个体时必须强制 `allow_public=false` 并调用 `revokePublicForTask`；`target_student_id` 创建/更新都必须调用 `validateTargetStudent`。
- 空 `activity_id` 视为独立任务；统计增加 `by_activity`，空 `activity_id` 按任务 ID 分组，不能全塞进一个“空活动”。
- `GET /admin/tasks/stats` 增加：参与率、回应率、待回应数、`pending_over_48h`、平均回应时长、`child_happy_rate`、`ack_rate`、`feedback_coverage`、按班级与按活动明细。
- `pending_over_48h` = 主回应待回应且 `TaskCheckIn.CreatedAt < now-48h` 的提交数；老师端 `GET /teacher/tasks/:id/checkins?status=pending` 由后端按 `created_at ASC` 排序，管理端统计使用同一口径；不做自动升级。
- 统计公式：`response_rate` = 有主回应的提交数 / 总提交数；`participation_rate` = 至少有一条提交的目标学生数 / 目标学生数（个体任务分母为 1，班级任务为目标学生数，不按家长用户数）；`avg_response_hours` = 仅 `single` 主回应的“提交到回应”中位数；`period` 汇总不计入该指标；补充回应不计入。
- 下线旧 `completion_rate` / `total_possible` / `checkin_count` / `not_checkin_count`；任务列表/详情/老师统计统一使用新口径。
- 管理端可查看任务列表与统计；不直接代替老师回应。
- 管理端可撤回公开内容。

### D13：媒体归属与保留

- 任务照片/语音统一走 `POST /common/upload`（`biz_type=task`），客户端提交 `media_file_id`，不直接传外链。
- 服务端校验：`files.id` 存在、`user_id` 为当前上传者、`biz_type=task`、`file_type` 为 image/audio。
- `TaskCheckIn.MediaFileID` / `TaskResponse.MediaFileID` 保存引用；`TaskCheckIn.MediaURL` / `TaskResponse.MediaURL` 作为读取缓存；不保留 `ImageURL`。
- 任务删除时级联软删除提交与回应；媒体文件保留，API 不再返回。
- 物理清理：学期结束 + 30 天后由运维任务处理（P1）。

### D14：软删除级联

- `TaskCheckIn` 增加 `DeletedAt`；`TaskResponse` 使用软删除。
- `task_check_ins` 空库直接建唯一 `(task_id, student_id, date)`；不引入 `ActiveKey`、迁移、回填或 feature flag。
- 删除任务时在同一事务内软删除其提交与回应，并硬删除该任务的 `task_read_marks`、`task_checkin_states`、`task_response_supplements`、`task_response_reactions` 行，避免孤儿；不提供单条提交恢复。
- 所有任务相关查询默认过滤软删除；统计排除已删除数据。

### D15：内容可读

- `ParentTask.LearningLink` 必填（服务层校验；数据库列保留可空以兼容历史）。
- `requirements` 从 JSON 字符串改为数组返回；家长端逐条渲染。
- 任务详情展示学习关联、要求、老师回应、家长反馈入口。
- 家长端回应卡片提供「给孩子看」全屏展示态：放大贴纸、播放语音、显示老师的话；不新增孩子独立账号。

### D16：`pending_tasks` 口径

- 首页 `pending_tasks` 只统计当前孩子**当前周期内尚未提交**的有效任务：
  - `daily`：今天没有提交。
  - `weekly`：滚动 7 天内没有提交。
  - `once`：整个任务周期内没有提交。
- 不再统计「所有进行中任务数」；已提交、已回应、已结束不计入。

## Delivery Slices

schema 一次到位；A/B/C 只是实现与验收顺序，不是可独立上线/回滚的交付单元。

| 切片 | 范围 | 门禁 |
| --- | --- | --- |
| A · 核心闭环 | 任务范围、提交、主回应、多孩/频率、移除积分、家长/老师核心页面、基础契约 | 两端构建通过；个体/班级隔离、多孩、频率、无积分、主回应闭环真实点击通过 |
| B · 隐私与感知 | 公开审核、媒体归属、补充回应、给孩子看、未读读标记、分享码/家长端小程序码、消息深链 | 公开墙边界、补充未读、孩子展示态、扫码进入家长端、消息跳转通过 |
| C · 集体与管理 | 班级星星罐、管理端活动分组、管理端统计/反馈指标、班级罐设置 | 活动事务/跨园所校验、星星罐 task/class、统计公式、看板真实数据通过 |

A 未通过不得进入 B，B 未通过不得进入 C；C 完成后才 archive。

## 状态机

```text
任务有效状态（派生，不落库）
  [未开始] --start_date 到--> [进行中] --end_date 过 / status=0--> [已结束]

提交（家长可见）
  [待完成] --家长提交--> [已提交] --主回应--> [已回应]
  [已回应] --补充回应--> [已回应 + 补充]

老师内部队列（家长不可见）
  [待回应] --单条回应 / 周期汇总--> [已回应]
       |
       +-- 超时 --> 仍为 [待回应]（老师待办）

公开
  [私密] --家长申请--> [待审核] --老师批准--> [班级墙可见]
  [待审核] --老师拒绝 / 家长取消--> [私密]
  [班级墙可见] --家长取消 / 老师撤回--> [私密，清空批准]
  [私密] --allow_public=false--> [私密，清空申请/批准/确认]
  [班级墙可见] --任务结束--> [不展示]
  [私密] --再次申请--> [待审核]（必须重新审核）

星星罐
  task：按任务中主回应为 class_star 的提交数 / 任务目标（period 覆盖 N 条加 N 星）
  class：按班级当前学期内 StarScope=class 任务中主回应为 class_star 的提交数 / 班级目标
```

## 跨端触点清单

| 层 | 是否修改 | 说明 |
| --- | --- | --- |
| Go domain | **会改** | `task.go`：`ParentTask` 增加 `ActivityID` / `TargetStudentID` / `LearningLink` / `AllowPublic` / `RecognitionRitual` / `StarScope` / `StarGoal`，删除 `Points`；`TaskCheckIn` 增加 `MediaFileID` / `MediaURL` / `IsPublicRequested` / `PublicScopeConfirmed` / `PublicApprovedAt` / `PublicApprovedBy` / `PublicRevokedAt` / `ResponseID` / `DeletedAt`，删除 `Points` / `ImageURL`；新增 `TaskResponse`、`TaskResponseSupplement`、`TaskResponseReaction`、`ClassStarJar`、`TaskCheckInState`、`TaskReadMark`；`message.go` 的 `Message` 增加 `ActionPath` |
| repository | **会改** | `task_repo.go`：个体/班级/活动查询、待回应/周期汇总、按孩子统计、星星罐、公开审核、软删除；旧完成率/打卡人数/任务积分统计下线 |
| service | **会改** | `parent_portal_service.go`：任务列表/详情/提交/历史/公开申请/反馈/未读标记/孩子展示态数据；`teacher_portal_service.go`：个体任务、待回应、单条/周期汇总/补充回应、公开审核、班级星星罐设置、分享码；`task_service.go`：管理端活动分组/创建/统计、移除旧指标与任务积分；新增 `task_rules.go`：班级罐校验、结构锁、公开撤销共享 helper |
| handler | **会改** | `parent_portal.go`、`teacher_portal.go`、`task.go`：字段、校验、回应/审核/反馈/未读/分享码/班级星星罐/统计 |
| Casbin policy | **会改** | 新增老师回应、批量回应、补充回应、公开审核、班级星星罐、分享码、家长公开申请/反馈/未读、管理端活动/统计的路由策略；核对 `/teacher/*`、`/parent/*`、`/admin/*` 通配是否覆盖 |
| router.go | **会改** | 注册新端点（见契约），含 `POST /parent/tasks/:id/read` |
| backend/api | **会改（契约变更）** | `api-parent/06-task.md`、`api-teacher/09-class-and-tasks.md`、`api-admin/10-tasks.md`、`api-common.md`：字段、端点、移除任务积分与旧统计、新增活动分组/班级星星罐/统计/未读标记与读接口、消息 `action_path` |
| 家长端页面 | **会改** | `parent/src/pages/tasks/index.vue`（未读红点）、`detail.vue`（含「给孩子看」展示态、打开清除未读）、`pages/index/index.vue`、`pages/messages/index.vue`（按 `action_path` 跳转）、`api/parent-task.js` |
| 老师端页面 | **会改** | `teacher/src/pages/tasks/index.vue`、新增 `detail.vue`、`api/teacher-tasks.js`、`pages.json` |
| 管理端 | **会改** | `admin/pages/tasks/index.vue`、`admin/api/tasks.ts`、`admin/pages/points`：活动分组/创建/编辑、统计、班级星星罐、移除任务积分规则 |
| 微信能力 | **会改** | `auth/wechat.go` 新增家长端凭据的 `GetParentWXACodeUnlimit` 与 `code_env_version` 配置；`teacher_service.go` 生成家长端小程序码 |
| design token | **不改值** | 新组件使用既有语义 token；`check-tokens` 验证 |
| 数据 | **会改（AutoMigrate）** | 空库直接建：`parent_tasks` 加 `activity_id` 并删 `points`；`task_check_ins` 加 `media_file_id`/`media_url`/公开/回应/`deleted_at`，删 `points`/`image_url`，唯一 `(task_id, student_id, date)`；新增 `task_responses` / `task_response_supplements` / `task_response_reactions` / `task_checkin_states` / `class_star_jars` / `task_read_marks`；无手工迁移/回填；用 `go run ./cmd/dbcheck` 核实 |
| Wire | **不改** | 不新增 repository/service 类型；新表读写复用现有 `TaskRepo` / service 的 `db` |
| 外部依赖 | **会改** | 语音复用 `/common/upload`；家长端小程序码复用微信 API，但需新凭据方法；不接订阅消息 |

## 契约字段溯源

| 字段 | 设计稿 / 业务场景 | 不能派生的理由 |
| --- | --- | --- |
| `activity_id` | 服务端生成的多班活动分组；设计稿没有活动实体 | 无法从标题/日期可靠推断哪些班级任务属于同一次活动；客户端不得自拟 |
| `target_student_id` | 个体任务：老师针对某个孩子创建；设计稿只有班级任务，个体任务是本次新增场景 | 无法从班级或提交记录推断任务原本发给谁 |
| `learning_link` | 任务必须说明与幼儿园学习的关系；见 proposal Why | 标题/描述不保证包含该关系 |
| `allow_public` | 班级活动可申请公开；设计稿无公开字段 | 默认私密是规则 |
| `recognition_ritual` | 贴纸/语音/班级星星 | 是老师配置 |
| `star_scope` / `star_goal` | 按任务/按班级星星罐 | 是老师配置 |
| `class_star_jar.goal` | 班级星星罐目标独立配置 | 任务 `StarGoal` 只适用于按任务范围，不能同时代表班级罐目标 |
| `is_public_requested` / `public_scope_confirmed` / `public_approved_at` / `public_approved_by` / `public_revoked_at` | 家长申请、老师审核（含「内容中只有提交者孩子」确认）与撤回 | 是双方动作；确认与撤回时间必须独立落库，不能由批准时间派生 |
| `response_id`（契约派生 `responded` / `responses`） | 老师回应是闭环状态；家长列表的「已回应」由本任务当前周期提交的 `response_id` 派生 | 是闭环状态本身，不能由提交时间或通知推断 |
| `content` / `media_url` / `sticker_key`（回应与提交同名字段） | 老师回应的三种载体；提交的照片/语音通过 `task_check_ins.media_file_id` + `media_url` 落库 | 老师/家长输入；媒体 URL 由文件表派生，`media_file_id` 才是归属校验依据 |
| `response_type` / `period_start` / `period_end` | 主回应是逐条（`single`）还是周期汇总（`period`），以及汇总覆盖的周期 | 回应关闭的提交集合由写入时决定，无法从提交时间反推 |
| `my_reaction`（存储于 `task_response_reactions`） | 家长质量反馈，按家长维度；契约只返回**当前登录家长**自己的反馈 | 家长动作；同一回应可被多个家庭分别反馈，故不能把反馈做成回应上的一个字段 |
| `notify_target_count` | 主回应承诺投递的家长数（义务集，先写义务再投递，含失败） | 通知结果不回写；用它当反馈率分母，才能避免「失败的那位家长被当成没有反馈」 |
| `supplements`（补充回应视图） | 汇总回应之后对单条提交的追加（贴纸/语音） | 补充回应是独立 `TaskResponse`，不被 `response_id` 引用，必须单独建模才能与主回应区分 |
| `action_path` | 任务发布/主回应消息的站内跳转路径，只允许内部 `/pages/` | 由业务动作决定，客户端不得自拟；必须带 `student_id` 才能落到多孩家庭正确孩子 |
| `points` | 既有字段；本变更从任务契约、页面与数据库移除 | 开发阶段无历史数据，直接删除字段与列 |
| `has_unread_supplement` | 未读模型：补充回应 `created_at > task_read_marks.last_read_at` | 未读是家长读状态，不能由提交/回应时间独立派生 |
| 统计字段（`response_rate` / `participation_rate` / `pending_over_48h` / `median_response_hours` / `feedback_coverage` / `child_happy_rate` / `ack_rate` / `notify_baseline_missing`） | 管理端统计与老师端反馈汇总的只读派生值 | 口径见 D12 与 §20.10-6/7；派生值不落库，避免与底层行漂移 |

## 数据与建表

- `parent_tasks` 新增：`activity_id VARCHAR(64) NOT NULL DEFAULT ''`、`target_student_id BIGINT NOT NULL DEFAULT 0`、`learning_link TEXT`、`allow_public TINYINT(1) NOT NULL DEFAULT 0`、`recognition_ritual VARCHAR(20)`、`star_scope VARCHAR(20) NOT NULL DEFAULT 'task'`、`star_goal INT NOT NULL DEFAULT 0`。
- `parent_tasks` 删除 `points`；`task_check_ins` 删除 `points` / `image_url`。
- `task_check_ins` 新增：`media_file_id BIGINT NOT NULL DEFAULT 0`、`media_url VARCHAR(500)`、`is_public_requested TINYINT(1) NOT NULL DEFAULT 0`、`public_scope_confirmed TINYINT(1) NOT NULL DEFAULT 0`、`public_approved_at DATETIME NULL`、`public_approved_by BIGINT NOT NULL DEFAULT 0`、`public_revoked_at DATETIME NULL`、`response_id BIGINT NOT NULL DEFAULT 0`、`deleted_at DATETIME NULL`；唯一 `(task_id, student_id, date)`。
- 新表 `task_responses`：`BaseModel` + 软删除；`kindergarten_id` / `task_id` / `teacher_id` / `ritual` / `sticker_key` / `content` / `media_file_id` / `media_url` / `response_type` / `period_start` / `period_end`；不持有 `parent_reaction`。
- 新表 `task_response_supplements`：`BaseModel`（保留 `created_at` 供未读比较）/ `kindergarten_id` / `response_id` / `checkin_id`；唯一 `(response_id, checkin_id)`；只存补充回应，主回应由 `TaskCheckIn.ResponseID` 表达。
- 新表 `task_response_reactions`：`kindergarten_id` / `response_id` / `user_id` / `reaction` / `created_at` / `updated_at`；唯一 `(response_id, user_id)`。
- 新表 `task_checkin_states`：`task_id` / `student_id` / `last_check_in_date`；唯一 `(task_id, student_id)`；空库直接建，无回填。
- 新表 `class_star_jars`：`kindergarten_id` / `class_id` / `term` / `goal` / `created_by` / `updated_by`；唯一索引 `(kindergarten_id, class_id, term)`。
- 新表 `task_read_marks`：`kindergarten_id` / `task_id` / `user_id` / `student_id` / `last_read_at`；唯一 `(task_id, user_id, student_id)`；未读 = 该孩子在该任务下存在补充回应且 `last_read_at IS NULL OR supplement.created_at > last_read_at`。
- `messages` 新增 `action_path VARCHAR(255)`；任务发布/主回应消息写入 `/pages/tasks/detail?id=<task_id>`。
- 索引：`parent_tasks (kindergarten_id, target_student_id)`、`(kindergarten_id, activity_id)`；`task_check_ins (task_id, response_id)`、`(task_id, deleted_at)`；`task_check_ins` 唯一 `(task_id, student_id, date)`；`task_responses (task_id, created_at)`；`task_response_supplements (checkin_id)`；`task_read_marks` 唯一 `(task_id, user_id, student_id)`。
- `class_star_jars` 目标独立配置；任务的 `star_goal` 只在 `star_scope=task` 时写入/生效。
- 空库直接 AutoMigrate 建表；开发库如已有旧表，直接 drop/recreate（无数据），不写迁移/回填；用 `go run ./cmd/dbcheck` 核实。

## 测试与验证策略

- 单元测试：
  - 个体/班级可见性、多孩隔离、跨班提交/回应被拒。
  - 待回应/单条回应/周期汇总；部分失败可见；汇总后追加个别补充回应；补充回应 `response_type=single`。
  - 补充回应不进入回应率/质量指标，不单独发通知。
  - 同一周期汇总回应被多个家长分别反馈，互不覆盖；跨家庭反馈被拒；`feedback_coverage` / `child_happy_rate` / `ack_rate` 按 `(response_id, parent_user_id)` 对计算。
  - 三种仪式；星星罐 task/class 按**主回应覆盖的提交数**计数，period 回应覆盖 N 条加 N 星；过滤软删除提交/回应；补充回应不计入；补充回应 `class_star` 被拒、未指定仪式默认贴纸；班级罐目标独立于任务 `StarGoal`。
  - 任务有第一条提交后，改 `frequency`/日期/`target_student_id`/`class_id`/`star_scope`/`star_goal` 被拒；改标题/说明/`learning_link`/`allow_public` 允许；`allow_public=false` 时公开内容立即消失、申请被清空且重新打开不复活；老师端与管理端调用同一 `task_rules.go` helper，行为一致。
  - 班级罐目标下调后进度封顶 100% 并显示“已超额”。
  - `POST /admin/task-activities` 的 `class_ids` 去重、上限 50、空列表/重复班级校验。
  - 公开申请/审核/撤回；取消或撤回时清空 `PublicScopeConfirmed` 与 `PublicApprovedBy/At`；再次申请必须重新审核并重新确认；个体任务不可公开；跨家庭申请被拒；含其他孩子的内容不得批准公开；未勾选 `PublicScopeConfirmed` 不得批准；任务 `allow_public` 关闭时批量撤销公开并清空申请/审核人，任务结束后不展示，跨班不可见，重新打开不恢复旧批准。
  - 班级罐目标必须 >0；设为 0 被拒。
  - 管理端 `POST /admin/tasks` 忽略/拒绝客户端 `activity_id`；活动任务单条 unsafe 编辑被拒。
  - 创建 `StarScope=class` 任务在 teacher/admin/activity 三个入口都校验班级罐目标；未配置一律拒绝。
  - 老师端待回应由后端按 `created_at ASC` 排序；管理端 `pending_over_48h` 与老师队列使用同一口径（主回应、超过 48h）。
  - 提交不写积分；任务统计不含积分；积分规则无任务项；老师列表/打卡列表/家长历史无 `points`。
  - 旧 `completion_rate` / `total_possible` / `checkin_count` / `not_checkin_count` 不再返回；新参与率/回应率口径一致。
  - 滚动 7 天频率；过期不可提交；并发提交只成功一次（状态行锁）；软删除仅随任务级联，不提供单条恢复。
  - 空库直接建 schema：唯一 `(task_id, student_id, date)`、`task_read_marks`、`task_response_supplements` 均按新定义创建，无迁移/回填。
  - 任务删除级联：提交/回应软删除，`task_read_marks` / `task_checkin_states` / `task_response_supplements` / `task_response_reactions` 硬删除，不留孤儿。
  - 媒体归属校验；软删除级联；`pending_tasks` 口径。
  - 通知计数按家长去重；`child_happy` / `ack` / `feedback_coverage` 分开统计；补充回应不通知，但任务有未读标记。
  - 未读模型：`task_read_marks` 按 `(task_id, user_id, student_id)` 独立；首次未读在 `last_read_at IS NULL` 时成立；同秒用 `created_at >= last_read_at` 不漏判；`POST /parent/tasks/:id/read` 带 `student_id`、校验归属、upsert 返回 `last_read_at`；并发读不 500；多家长/多孩互不影响。
  - 消息深链：任务发布/主回应消息的 `action_path` 带 `student_id`，能打开对应孩子/任务详情；只允许内部 `/pages/` 路径；任务删除/结束有提示；未登录保留路径并在登录后回跳；回应消息也能解析到 task_id。
  - 通知计数：任务发布与主回应都按家长去重，`notified_count + notify_failed_count` 口径一致；创建任务响应返回发布计数，回应响应返回回应计数。
  - 统计公式：`response_rate` / `participation_rate` / `avg_response_hours` 按 D12 定义；`avg_response_hours` 只含 single 主回应，period 不计入；参与率分母为目标学生数。
  - 管理端活动：班级归属校验、创建/删除事务、活动任务禁止单条删除与单条 unsafe 编辑。
- 契约测试：逐字段比对 handler/service DTO 与 `backend/api/`；确认任务积分字段已移除。
- 客户端：
  - `npm run build:mp-weixin`（parent/teacher）通过；`npm run build`（admin）通过。
  - `node scripts/check-tokens.mjs parent` / `teacher` 对改动文件无新增违规。
- 真实交互：
  - 老师创建个体任务 → 目标家庭可见、其他家庭不可见。
  - 家长提交 → 老师端待回应 → 单条/周期汇总回应 → 家长看到回应并反馈。
  - 班级活动家长申请公开 → 老师审核 → 班级墙可见 → 撤回后消失。
  - 管理端创建多班活动 → 按 `activity_id` 编辑一次全部班级 → 统计出现活动聚合与参与/回应数据。
  - 老师生成家长端小程序码 → 开发/测试环境扫码进入对应版本家长端任务详情。
  - 家长打开「给孩子看」展示态 → 看到大贴纸/播放语音/老师的话。
- 收工：杀掉残留 `dev:mp-weixin` watcher；`openspec validate --all` 通过。

## 风险与回滚

- **老师内部回应队列积压**：待回应队列必须显眼；习惯任务用周期汇总；P1 再加超时提醒。
- **语音上传失败**：允许退回贴纸；不得让回应因上传失败而丢失。
- **公开墙隐私**：老师审核后公开；个体任务强制私密；公开内容不含其他孩子信息；可撤回。
- **家长端小程序码**：必须走家长端凭据；否则扫码进入老师端，触达失败。
- **移除积分**：开发阶段直接清理；上线前确认没有旧页面仍读取任务 `points`。
- **移除旧统计**：`completion_rate` / `total_possible` / `checkin_count` / `not_checkin_count` 下线后，管理端/老师端必须同步切到新口径，否则页面空白。
- **未读模型**：`task_read_marks` 必须按 `(task_id, user_id, student_id)` 独立；不要用补充回应的单字段 `read_at`，否则双家长或双胞胎会互相清除；首次未读必须处理 `last_read_at IS NULL`。
- **消息深链**：回应消息的 `reference_id` 是 response_id，不能直接当 task_id；必须通过 `action_path` 或 response→task 解析，否则消息点不开任务。
- **分享码环境**：`code_env_version` 配置错误会扫码进入错误版本；开发/测试必须显式设置 `trial`/`develop`。
- **新表/新列**：AutoMigrate 只增不改；索引冲突用 `dbcheck` 核实后手工处理。
- **回滚**：按文件回滚；新增列/表可保留；关闭任务入口即可停止使用。
