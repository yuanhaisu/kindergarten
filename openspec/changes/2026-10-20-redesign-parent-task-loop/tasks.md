# Tasks

## Delivery Slices

本次仍是一个 OpenSpec change，但按 A/B/C 推进；A 未通过不得进入 B，B 未通过不得进入 C，C 完成后才 archive。

| 切片 | 范围 | 任务组 |
| --- | --- | --- |
| A · 核心闭环 | 任务范围、提交、主回应、多孩/频率、移除积分、家长/老师核心页面、基础契约 | 1.1-1.6, 2.1-2.11, 3.1-3.4, 6, 8.1-8.7, 9.1-9.4, 9.6, 10.1-10.4, 12.1-12.2 |
| B · 隐私与感知 | 公开审核、媒体归属、补充回应、给孩子看、反馈汇总、未读读标记、分享码/家长端小程序码、消息深链 | 3.5-3.6, 3.11, 4.1-4.8, 5.1-5.7, 9.5, 9.7, 10.5-10.8, 12.3-12.4, 12.6 |
| C · 集体与管理 | 班级星星罐、管理端活动分组、管理端统计/反馈指标、班级罐设置 | 3.7-3.10, 7.0-7.6, 11.1-11.4, 12.5, 12.7-12.8 |

> 标记 `[P2]` 的任务是收尾项，不阻塞 A/B/C 门禁；可在对应切片内顺手处理，未列入切片范围的也不影响交付。
>
> 班级罐校验：2.9 在 A 先实现只读 `validateClassStarJar` helper；3.8 在 C 只做配置端点/UI 并复用该 helper，避免 A 依赖 C。

## 1. 后端 · 领域与数据

- [x] 1.1 在 `server/internal/domain/task.go` 为 `ParentTask` 增加 `ActivityID`、`TargetStudentID`、`LearningLink`、`AllowPublic`、`RecognitionRitual`、`StarScope`、`StarGoal`，删除 `Points`；为 `TaskCheckIn` 增加 `MediaFileID`、`MediaURL`、`IsPublicRequested`、`PublicScopeConfirmed`、`PublicApprovedAt`、`PublicApprovedBy`、`PublicRevokedAt`、`ResponseID`、`DeletedAt`，删除 `Points` / `ImageURL`；空库直接定义唯一 `(task_id, student_id, date)`；为 `Message` 增加 `ActionPath`；验证：`go build ./...` 通过
      <br>落地 2026-10-24 前（C-1/C-2）。`ParentTask` 新字段见 `domain/task.go:45,47,53,57-60`；`TaskCheckIn` 见 `:85-92`（含 `ResponseID`）；`Message.ActionPath` 见 `domain/message.go:24`；`Points`/`ImageURL` 在 `domain/task.go` 已零命中（2026-10-25 复核）。
- [x] 1.2 新增 `TaskResponse`、`TaskResponseSupplement`、`TaskResponseReaction`、`ClassStarJar`、`TaskCheckInState`、`TaskReadMark` domain struct 与表名；`TaskResponse` 使用软删除；`TaskResponseSupplement` 使用 `BaseModel`（需要 `created_at` 供未读比较），唯一 `(response_id, checkin_id)`，含 `kindergarten_id`；`TaskResponseReaction` 唯一 `(response_id, user_id)`；`ClassStarJar` 唯一 `(kindergarten_id, class_id, term)`；`TaskCheckInState` 唯一 `(task_id, student_id)`；`TaskReadMark` 唯一 `(task_id, user_id, student_id)`；验证：`go build ./...` 通过
      <br>落地 2026-10-24 前（C-1/B4）。六个 struct 见 `domain/task.go:101`（TaskResponse）、`:126`（Supplement）、`:136`（Reaction）、`:147`（CheckInState）、`:157`（ClassStarJar）、`:170`（ReadMark）；唯一索引写在 tag 上（如 `:129` `idx_task_response_supplement`、`:139` `idx_task_response_reaction`）。
- [ ] 1.3 空库直接 AutoMigrate 建表：开发库如已有旧表，直接 `DROP TABLE parent_tasks, task_check_ins` 后重建（无数据、不写迁移）；确认 `task_check_ins` 唯一 `(task_id, student_id, date)`、`task_read_marks` 表存在且唯一 `(task_id, user_id, student_id)`、`parent_tasks`/`task_check_ins` 无 `points` 列、`task_check_ins` 无 `image_url`、有 `public_scope_confirmed`、`messages` 有 `action_path`；若 `task_checkin_states.last_check_in_date` 曾以 NOT NULL 建过，执行 `docs/migrations/2026-10-21-task-checkin-state-nullable.sql`；执行 `go run ./cmd/dbcheck parent_tasks`、`task_check_ins`、`task_responses`、`task_response_supplements`、`task_response_reactions`、`task_checkin_states`、`class_star_jars`、`task_read_marks`、`messages` 核实；验证：无手工回填文件，nullable 列已确认
- [x] 1.4 删除任务时在同一事务内软删除该任务的提交与回应，并硬删除该任务的 `task_read_marks` / `task_checkin_states` / `task_response_supplements` / `task_response_reactions` 行；所有查询与统计默认过滤软删除；不提供单条提交恢复；验证：单测断言删除后列表/统计/公开墙都不返回，且无 read/state/supplement/reaction 孤儿
      <br>落地 2026-10-24 前（C-1/B4）。级联在 `task_rules.go:566-601` 的删除事务内：`:574` supplement、`:577` reaction、`:587` checkin_state、`:590` read_mark 均 `Unscoped` 硬删，提交与回应软删。
- [ ] [P2] 1.5 删除 `Rewards` 字段：domain/契约/输入 DTO 均不再使用；验证：`grep` 无任务 `Rewards` 读写
- [x] 1.6 验证：`points` 字段与列已删除，任务接口/页面无 `points`；不新增 repo/service 类型；不引用 `backend/model/`
      <br>落地 2026-10-24 前。2026-10-25 复核：`domain/task.go`、`task_service.go`、`teacher_portal_service.go`、`parent/src/pages/tasks/` 对 `"points"`/`Points:` 均零命中；任务路径未新增 repo/service 类型。

## 2. 后端 · 范围、多孩、有效期与频率

- [ ] 2.1 在 `task_repo.go` 增加按 `target_student_id` 与班级的查询：班级任务按 `class_id`，个体任务按 `target_student_id`；两者都带 `kindergarten_id`；验证：单测覆盖班级/个体/其他班三类
- [ ] 2.2 家长端任务列表/详情/提交/历史/公开申请/反馈统一要求 `student_id`；缺失或不属于该家长时拒绝，不回退到第一个孩子；验证：单测覆盖缺失、非法、多孩切换
- [ ] 2.3 提交校验：任务属于该孩子班级，或 `target_student_id == 该孩子`；不满足返回明确拒绝；验证：单测断言跨班提交不落库
- [x] 2.4 老师创建/更新个体任务时校验目标孩子属于 `tc.ClassID` 与园所；回应/审核校验提交属于该任务与班级；验证：单测覆盖跨班目标、跨班回应、更新目标跨班被拒
      <br>落地 2026-10-24 前。四个包级 helper 已就位且两端共用：`task_rules.go:87` `validateTargetStudent`、`:328` `validateClassStarJar`、`:544` `assertTaskEditable`、`:609` `revokePublicForTask`。跨班目标用例即 `TC-NEW-TSK-0146`（`docs/tests/TEST_CATALOG.md:729`）。
- [ ] 2.5 任务有效状态按 `status=1 AND start_date <= 今天 <= end_date` 判定；列表、详情、提交统一使用；验证：单测覆盖未开始、进行中、已结束
- [ ] 2.6 频率校验：`daily` 上海时区当天一次；`weekly` 滚动 7 天一次；`once` 周期内一次；提交在事务内 `SELECT ... FOR UPDATE` `task_checkin_states` 状态行（不存在则创建，创建冲突时回滚重试并锁定已存在行），校验 `last_check_in_date` 后写入并更新状态；`(task_id, student_id, date)` 唯一索引兜底同日重复；验证：单测覆盖三种频率、跨日/跨周、并发双击只成功一次
- [ ] 2.7 修正 `HomeOverview.pending_tasks`：只统计当前孩子当前周期内尚未提交的有效任务；验证：单测覆盖 daily/weekly/once、已提交、已结束
- [x] 2.8 创建/编辑任务时校验 `learning_link` 必填；老师端与管理端表单都标记必填；验证：单测与真实提交空值被拒
      <br>落地 2026-10-24 前。后端必填见 `teacher_portal_service.go:2970`（创建）与 `:3058`（更新），均为空则拒绝；老师端表单字段见 `teacher/src/pages/tasks/index.vue`，管理端见 `admin/pages/tasks/index.vue`（C-3）。
- [x] 2.9 改造 `TeacherPortalService.CreateTask` / `UpdateTask`：CreateTask 持久化 `allow_public` / `recognition_ritual` / `star_scope` / `star_goal`，`allow_public` 默认 false、个体任务强制 false；**在 `server/internal/service/task_rules.go` 实现 `validateTargetStudent` / `validateClassStarJar` / `assertTaskEditable` / `revokePublicForTask` 四个包级 helper**，两个 service 共用；UpdateTask 有提交后只允许标题/说明/`learning_link`/`allow_public`，`allow_public=false` 时批量清空批准/确认/审核人/申请，`StarScope=class` 变更调用同一 helper；更新后 `target_student_id>0` 时强制 `allow_public=false` 并调用 `revokePublicForTask`，且 `target_student_id` 变更调用 `validateTargetStudent`；验证：单测覆盖创建字段不被丢弃、个体任务公开强制 false、class scope 校验、老师端改结构字段被拒、改标题允许、关闭公开撤销、班级改个体后公开关闭、更新目标跨班被拒
      <br>落地 2026-10-24 前（C-1/B4）。创建路径字段落库见 `teacher_portal_service.go:3040`（`LearningLink` 等），必填校验 `:2970`/`:3058`，更新按安全字段逐项比较见 `:3121` 起；四个 helper 见 `task_rules.go:87/328/544/609`，老师端与管理端共用同一份。
- [ ] [P2] 2.10 编辑边界补充：`RecognitionRitual` 可改但只影响未来回应；`Status=0` 结束任务需二次确认；验证：单测/真实点击
- [ ] 2.11 验证：`go build ./...`、`go vet ./...` 通过；对照 specs 的范围/多孩/有效期/频率/pending/learning_link/结构锁场景

## 3. 后端 · 提交、回应、质量反馈与星星罐

- [ ] 3.1 家长提交接口改用 `media_file_id`；服务端校验文件存在、`user_id` 为上传者、`biz_type=task`、类型为 image/audio；提交不写积分；验证：单测覆盖合法/非法文件、积分不变
- [ ] 3.2 老师端 `GET /teacher/tasks/:id/checkins` 支持 `status=pending|responded`，`pending` 由后端按 `created_at ASC` 排序，返回提交、家长、时间、回应、公开状态；验证：单测覆盖筛选与排序
- [x] 3.3 新增 `POST /teacher/tasks/:id/checkins/:checkin_id/response`：创建 `TaskResponse`（`single`），把 `TaskCheckIn.ResponseID` 指向该回应；验证：单测覆盖三种仪式
      <br>落地 2026-10-24 前（B）。路由见 `router.go:196`（`TeacherRespondTask`）；三种仪式常量在 `domain/task.go:21-23`，测试侧共 89 处引用覆盖（`task_response_supplement_test.go` 41 处为主）。
- [ ] 3.4 新增 `POST /teacher/tasks/:id/checkins/batch-response`：创建一条 `TaskResponse`（`period`），把周期内待回应提交的 `ResponseID` 指向它；部分失败返回失败 ID；验证：单测覆盖全部成功、部分失败、空列表、跨任务 ID
- [x] 3.5 汇总后个别补充：新增 `POST /teacher/tasks/:id/checkins/:checkin_id/supplement`，创建 `TaskResponse`（`response_type=single`）并写 `task_response_supplements`；必须已有主回应，不改变 `ResponseID`；只允许贴纸/语音，`class_star` 返回明确错误；未指定仪式时默认贴纸，不继承任务默认仪式；验证：单测断言汇总后仍可追加贴纸/语音，家长端返回主回应 + 补充回应，pending 提交不能补，class_star 被拒，默认贴纸
- [x] 3.6 家长反馈：新增 `POST /parent/tasks/responses/:response_id/reaction`，写入 `task_response_reactions`；唯一 `(response_id, user_id)`；取值 `child_happy` / `received` / `thanks`；只允许对“与自己孩子提交相关”的回应反馈——提交人本人**或**与该孩子有绑定关系的另一位家长，跨家庭拒绝；验证：单测覆盖三种反馈、同一家长更新、co-parent 可反馈、跨家庭拒绝、多个家长分别反馈不互相覆盖；补充回应的反馈不进入主回应指标（指标属 7.2，落地时补断言）
- [ ] [P1] 3.11 老师端反馈汇总：`GET /teacher/tasks/:id/feedback-summary`（或并入任务详情）返回 `feedback_coverage` / `child_happy_rate` / `ack_rate` 与按孩子的反馈计数；口径与 7.2 一致（按 `(response_id, parent_user_id)` 对计算，分母为回应覆盖的去重家长数，含通知失败；补充回应的反馈不计入）。规格「老师端与管理端 SHALL 能看到反馈汇总」的老师端一半目前无人承接；验证：单测覆盖无反馈、部分反馈、co-parent 两条反馈、补充反馈不计入

- [x] 3.7 星星罐：`StarScope=task` 统计该任务中**主回应为 `class_star` 的提交数** / `StarGoal`；`StarScope=class` 按班级当前学期统计同类提交数 / `class_star_jars.goal`；period 回应覆盖 N 条提交就加 N 星；join 时过滤 `TaskCheckIn.DeletedAt` 与 `TaskResponse.DeletedAt`；进度展示封顶 100%，目标下调时显示“已超额”；补充回应不计入；验证：单测覆盖 task/class、period 加 N 星、软删除不计入、贴纸/语音不计入、补充回应不计入、满罐/超额
      <br>分两次落地：**task 侧**随 B4-b（`taskStarCounts` / `star_count`）；**class 侧**在 C-4 本轮补上（`classStarCountInTerm` + `GetClassStarJarWithProgress`，按提交日期落在 term 内计数）。证据见 handoff §24.4：5 条新单测（scope/period/软删/贴纸/补充/超额/封顶/跨班/跨园所/学期边界）+ 老师端卡片真实渲染 `已集 3 / 30`、`已超额 3/2`。未配置时 `progress=null`（不是 0）是**有意决定**，理由见 handoff §24.3-6。
- [x] 3.8 班级星星罐目标独立配置：新增 `GET/PUT /teacher/class/star-jar`（按学期），目标必须 >0；**复用 `task_rules.go` 的 `validateClassStarJar` helper**，本任务只做配置端点/UI，不重复写校验；teacher/admin/activity 三个创建入口共用该 helper；任务 `StarScope=class` 时忽略任务 `StarGoal`，UI 隐藏任务目标；验证：单测覆盖三个入口未配置均拒绝、目标 0 被拒、设置、修改、跨班拒绝
      <br>端点与校验随 C-1 落地；**老师端 UI 在 C-4 补齐**（按班级可选/未配置时给可读原因、目标设置入口、detail 口径文案）。C-4 同时修掉 C-1 的一个映射缺陷：老师端建/改任务曾把「班级罐缺失」折叠成「请求参数无效」，现在三个入口返回同一句「请先配置班级星星罐」（handoff §24.2 / §24.4 接口层表）。UI 上「目标设 0 的 toast」未在运行中的小程序里抓到，C-5 补验（§24.5-1）。
- [x] [P2] 3.9 班级罐 `term` 跨年边界：`Class.AcademicYear + CurrentSemester` 为空/跨学期时给出 fallback；验证：单测
      <br>落地 2026-10-24（C-1 + C-4）。`task_rules.go:209` `classStarJarTermOf` / `:304` `classStarJarTermWindow`；无学年班级按自然年切分。单测见 `task_admin_c1_test.go:208-276`（spring/fall/跨学期/班级不存在）与 `task_star_jar_progress_test.go:232-233`（8/1–12/31 无缝不重叠）。C-4 补的「后一学期起始 = 前一学期结束 + 1 天」也在此断言。
- [x] 3.10 验证：`go test ./internal/service -run 'Task|ParentTask'` 通过；对照 specs 的提交/回应/反馈/星星场景
      <br>落地 2026-10-25（C-4 收尾）。证据：`go test ./internal/service -p 1 -parallel 2 -count=1` 全量通过，`ok kindergarten-server/internal/service 220.271s`——该命令是真要求的超集（含 `-run 'Task|ParentTask'` 全部用例）。场景对照见 handoff §24.4 与本节 3.5/3.7/3.8 注记。

## 4. 后端 · 公开审核

- [x] 4.1 家长公开申请/取消：新增 `POST /parent/tasks/checkins/:checkin_id/public-request` 与取消接口；仅当任务 `allow_public=true` 且非个体任务、且提交属于当前孩子时允许；申请时清空 `PublicApprovedAt` / `PublicApprovedBy` / `PublicScopeConfirmed`；取消时清空批准/审核人、`PublicScopeConfirmed=false` 并写 `PublicRevokedAt`；验证：单测覆盖允许、任务不允许、个体任务拒绝、跨家庭拒绝、取消后确认与审核人清除
- [x] 4.2 老师审核：新增 `PUT /teacher/tasks/:id/checkins/:checkin_id/public`，校验提交属于该任务与班级，并要求老师勾选 `PublicScopeConfirmed`（内容中只有提交者孩子，图片/语音/文字均适用）；批准时该字段必须为 true，写 `PublicApprovedAt/By`；拒绝/撤回清空批准/审核人、`PublicScopeConfirmed=false` 并写 `PublicRevokedAt`；再次申请必须重新批准并重新确认；含其他孩子的内容不得批准；验证：单测覆盖批准、拒绝、撤回、取消/撤回后再次申请不沿用旧批准或旧确认、未勾选/含他人内容被拒、审核留痕
- [x] 4.3 公开墙查询：join 任务，只返回 `Task.AllowPublic && Task.DeletedAt IS NULL && Task.Status=1 && Task.EndDate>=今天 && Task.ClassID = 当前孩子班级 && IsPublicRequested && PublicScopeConfirmed && PublicApprovedAt != nil && PublicRevokedAt == nil` 的提交；读取时实时过滤；任务结束后不展示，跨班不可见；只展示提交者孩子的内容；验证：单测覆盖默认私密、批准可见、撤回不可见、allow_public 关闭不可见、任务结束不可见、跨班不可见、含他人内容不展示
- [x] 4.4 管理端撤回：管理员可撤回公开内容；验证：单测覆盖撤回后不可见
- [ ] [P2] 4.5 一键完成（无文字无媒体）不可申请公开；验证：单测/真实点击
- [ ] [P2] 4.6 任务结束后公开内容的老师回看/导出能力：明确不提供或加只读入口；验证：产品确认
- [ ] [P2] 4.7 `PublicApprovedBy` 清空后的审计历史：写入只读操作日志或明确不保留；验证：审计页
- [ ] 4.8 验证：对照 specs 的公开场景；公开墙不返回其他孩子姓名/班级以外的信息

## 5. 后端 · 通知、分享码与微信能力

- [x] 5.1 任务发布通知：给目标家庭写站内消息（`type=task`，`reference_id=task_id`，`action_path=/pages/tasks/detail?id=<task_id>&student_id=<student_id>`），同一任务对同一家长至多一条；个体任务用目标孩子，班级任务用该家长在该班的一个确定孩子；只允许内部 `/pages/` 路径；`notified_count` / `notify_failed_count` 按家长去重，并由创建任务响应返回；验证：单测覆盖班级/个体范围、多孩 action_path、内部路径校验、发布通知计数
- [x] 5.2 主回应通知聚合：按 `TaskResponse` 与家长聚合，一个家长一次主回应只收一条消息（`reference_id=response_id`，`action_path=/pages/tasks/detail?id=<task_id>&student_id=<student_id>`）；覆盖多条提交不重复发送；`notified_count` / `notify_failed_count` 按家长用户去重；补充回应不单独发通知；验证：单测覆盖单条/周期汇总/多孩/同一家长多条提交/补充不通知/action_path 带 student_id
- [x] 5.3 通知失败不回滚任务/回应；老师回应响应返回按家长去重的 `notified_count` / `notify_failed_count`，只统计主回应；验证：单测覆盖部分失败
- [ ] 5.4 新增 `WeChatClient.GetParentWXACodeUnlimit(ctx, scene, page, envVersion)`（走家长端凭据）；新增 `wechat.code_env_version` 配置（默认 `release`，开发/测试 `trial`/`develop`）；老师端 `GET /teacher/tasks/:id/share-card` 生成 `pages/tasks/detail` 的小程序码/海报 URL，`scene=t=<task_id>`；验证：单测断言使用家长端凭据与 envVersion；真实扫码进入对应版本家长端
- [x] 5.5 补充回应未读提示：新增 `task_read_marks(task_id, user_id, student_id, last_read_at)` 唯一 `(task_id, user_id, student_id)` 与 `POST /parent/tasks/:id/read`；读接口带 `student_id`，校验家长绑定、任务可见且未删除，upsert（唯一冲突重试/ON DUPLICATE）并返回 `{ last_read_at }`；未读 = `last_read_at IS NULL OR 补充回应.created_at >= last_read_at`；任务列表/详情显示红点，打开后更新读标记；验证：单测与真实点击，首次未读、同秒、并发读不 500、多家长、多孩互不影响
      <br>落地 2026-10-24 前（B）。表见 `domain/task.go:170`；`POST /parent/tasks/:id/read` 见 `router.go:504`，实现 `parent_portal_service.go:1686` `MarkTaskRead`；未读谓词 `:1745`（`last_read_at IS NULL OR r.created_at >= last_read_at`，同秒不漏判）；列表/详情字段 `:1573` / `:1678`。
- [ ] [P2] 5.6 `task_read_marks.last_read_at` 与 supplement `created_at` 使用同一存储时区/精度；验证：单测同秒/跨时区
- [x] 5.7 验证：消息文案包含任务标题与入口提示；不接微信订阅消息；不新增 `MessageType`

## 6. 后端 · 移除任务积分

- [x] 6.1 从 domain、任务创建/编辑/列表/详情契约与 DTO 中删除 `points` 字段/列；逐端点改造，不能直接返回 domain struct：老师任务列表（`TeacherListTasks`）、老师打卡列表（`TaskCheckins`）、家长打卡历史（`ParentCheckinHistory`）、管理端任务列表；验证：接口响应无 `points`/内部字段，数据库中无 `points` 列
      <br>落地 2026-10-24 前。2026-10-25 复核：`teacher_portal_service.go`、`task_service.go`、`parent/src/pages/tasks/` 对 `"points"`/`Points:` 零命中；四个端点均改为手写响应 map，不直接返回 domain struct。列级删除另见 1.3。
- [ ] 6.2 提交不写 `PointTransaction`、不改 `PointAccount`、不写 `TaskCheckIn.Points`；验证：单测断言提交前后积分账户与流水不变
- [x] 6.3 `PointService.GetPointRules` 移除「任务打卡」规则；`UpdatePointRules` 拒绝该规则；管理端积分规则页不再展示任务项；验证：单测与页面检查
      <br>落地 2026-10-24 前。`point_service.go:268-272` 只返回 id 2/3/4（周全勤/月全勤/成长推荐），无任务项；`UpdatePointRules` 的 `switch` 无 `case 1`，落入 `default: BAD_RULE_ID`（`:297-306`）；`admin/pages/points` 对 `task` 零命中。**残留（非本任务范围）**：`task_checkin_points` 仍被 `readPointsFromExtra`（`:232`）读入但返回值已丢弃，`settings_service.go:97` 仍保留该配置字段——属死配置，不影响规则展示。
- [x] 6.4 `TaskService.GetTaskStats` 移除 `total_points_awarded`；`SumCheckinPointsInDateRange` 不再被任务统计使用；验证：`grep` 确认无任务路径调用
      <br>落地 2026-10-24（C-2）。`task_admin_c2_stats_test.go:356` 断言响应不含 `avg_completion_rate` / `total_points_awarded`；任务统计路径无 `SumCheckinPointsInDateRange` 调用（该函数仅在积分类使用）。
- [x] 6.5 管理端积分规则页（`admin/pages/points` 或对应设置页）移除「任务打卡」规则项；验证：页面不再出现任务积分规则，保存其他规则不受影响
      <br>落地 2026-10-24 前。`admin/pages/points` 对 `任务打卡`/`task` 零命中（2026-10-25 复核）；其余规则保存路径未改动（`UpdatePointRules` 仅接受 id 2/3/4）。
- [x] 6.6 验证：`points` 字段与列已删除；家长端积分页不展示任务来源；任务页无积分文案
      <br>落地 2026-10-24 前。代码侧零 `points` 命中见 6.1/1.6；`pointSourceLabel`（`point_service.go:33-48`）仍保留 `task_checkin → 任务打卡` 的**历史流水展示标签**，用于回溯旧数据，不代表任务页仍有积分；列级删除见 1.3。

## 7. 后端 · 管理端活动与统计

- [x] 7.0 改造 `TaskService.CreateTask`（`POST /admin/tasks` 独立任务）：持久化 `allow_public` / `recognition_ritual` / `star_scope` / `star_goal`；`allow_public` 默认 false；调用 `task_rules.go` 的 `validateTargetStudent` / `validateClassStarJar`；验证：创建请求字段不被丢弃、默认值正确、目标孩子归属与 class scope 校验
- [x] 7.1 新增管理端活动端点：`POST /admin/task-activities`（服务端生成 `activity_id`，共享字段 + `class_ids`；`class_ids` 必填、去重、最多 50、全部属于当前园所；事务创建）、`PUT /admin/task-activities/:activity_id`（只允许标题/说明/学习关联/公开开关；C-2 追加：`status` 允许批量改，作为活动唯一结束路径，见 handoff §21.8）、`DELETE /admin/task-activities/:activity_id`（整组删除，事务级联）；`POST /admin/tasks` 强制忽略/拒绝客户端 `activity_id`；活动任务禁止走单条 `DELETE /admin/tasks/:id`，单条 `PUT /admin/tasks/:id` 只允许安全字段或直接拒绝；`allow_public=false` 时调用 `task_rules.go` 的 `revokePublicForTask`；`star_scope` 创建或更新为 `class` 时调用 `validateClassStarJar`；`target_student_id` 创建/更新调用 `validateTargetStudent`；更新后 `target_student_id>0` 时强制 `allow_public=false` 并调用 `revokePublicForTask`；空 `activity_id` 为独立任务；任务有第一条提交后，单条/批量编辑都调用 `assertTaskEditable`；验证：单测覆盖创建/批量编辑/整组删除/非法字段被拒/跨园所班级被拒/空列表与重复班级被拒/超过 50 被拒/一个班级失败整组回滚/活动任务单删被拒/活动任务 unsafe 单编辑被拒/独立任务自设 activity_id 被拒/有提交后结构字段被拒/关闭 allow_public 撤销并清空申请/审核人/更新为 class scope 未配置目标被拒/班级改个体后公开关闭/更新目标跨班被拒
- [x] 7.2 `GET /admin/tasks/stats` 增加参与率、回应率、待回应数、`pending_over_48h`、平均回应时长、`child_happy_rate`、`ack_rate`、`feedback_coverage`、按班级与按活动明细；`response_rate` = 有主回应提交数 / 总提交数；`participation_rate` = 至少一条提交的目标学生数 / 目标学生数（个体任务分母 1）；平均回应时长 = 仅 `single` 主回应的“提交到回应”中位数，`period` 不计入（契约字段名 `median_response_hours`，见 decision §20.10-7）；`pending_over_48h` = 主回应待回应且 `CreatedAt < now-48h` 的提交数，与老师端 `pending` 队列同口径；主回应指标按 `(response_id, parent_user_id)` 对计算，分母为覆盖的去重家长数（`task_responses.notify_target_count` 义务集：先写义务再投递，含通知失败；decision §20.10-6）；补充回应不含在内；验证：单测覆盖无数据、部分参与、全部回应、多个家长反馈不互相覆盖、比率不超过 100%、通知失败仍计入分母、period 不计入 single 均值、超 48h 计数与排序一致
- [x] 7.3 管理端可查看公开内容并撤回；验证：接口与权限
- [x] 7.4 下线旧 `completion_rate` / `total_possible` / `checkin_count` / `not_checkin_count`：任务列表、任务详情、老师统计、管理端统计、看板统一新口径；验证：`grep` 确认无旧字段返回，前后端同步（后端已下线；管理端页面接新字段 / 移除旧列属 C-3 的 11.1/11.4）
- [x] 7.5 管理端班级星星罐设置：新增 `GET/PUT /admin/classes/:id/star-jar`，按班级/学期设置目标，校验园所归属，目标必须 >0；验证：接口与权限，目标 0 被拒，任务 `StarScope=class` 使用该目标
- [ ] [P2] 7.6 `participation_rate` 分母是否排除没有绑定家长的学生；选定口径并加单测

## 8. 契约

- [x] 8.1 更新 `backend/api/api-parent/06-task.md`：列表/详情新增 `target_student_id`、`learning_link`、`allow_public`、`recognition_ritual`、`star_scope`、`star_goal`、`responses`（每条含 `my_reaction`，取值为 `child_happy` / `received` / `thanks`）、`public_*`（含 `public_scope_confirmed`）、`has_unread_supplement`；新增 `POST /parent/tasks/:id/read`，返回 `{ last_read_at }`；`requirements` 改为数组；移除 `points` 与旧统计字段
      <br>落地 2026-10-24 前（B）。2026-10-25 复核：上述字段在 `api-parent/06-task.md` 共命中 24 处（`target_student_id` / `learning_link` / `allow_public` / `recognition_ritual` / `star_scope` / `star_goal` / `has_unread_supplement` / `last_read_at` / `my_reaction`）；`points` 只出现在三处「已移除」声明（`:53`/`:116`/`:161`），无字段定义。
- [x] 8.2 更新 `backend/api/api-teacher/09-class-and-tasks.md`：创建/编辑新增字段；创建任务响应返回发布通知的 `notified_count` / `notify_failed_count`（按家长去重）；打卡列表支持 `status`；新增单条/周期汇总/个别补充回应、公开审核、班级星星罐设置、分享码端点；回应通知计数按家长去重、只统计主回应；移除 `points` 与旧统计字段
      <br>落地 2026-10-24 前（B）。2026-10-25 复核：`notified_count` / `batch-response` / `supplement` / `public` / `star-jar` / `share-card` / `status=pending` 在该文件共命中 55 处；端点与 `teacher/src/api/teacher-tasks.js:31-85` 逐条对应；`points` 只出现在四处「已移除」声明（`:223`/`:287`/`:333`/`:371`）。
- [x] 8.3 更新 `backend/api/api-admin/10-tasks.md`：`POST /admin/tasks` 创建独立任务新增 `allow_public` / `recognition_ritual` / `star_scope` / `star_goal` 字段与默认值；新增活动端点（`POST/PUT/DELETE /admin/task-activities`，`activity_id` 服务端生成、安全字段批量编辑、整组删除、事务与班级归属校验）、`GET/PUT /admin/classes/:id/star-jar`、统计字段（参与率/回应率/待回应/`pending_over_48h`/`child_happy_rate`/`ack_rate`/`feedback_coverage`/按活动）、公开撤回；移除任务积分与旧统计字段
- [ ] 8.4 更新 `backend/api/api-common.md` 消息字段：新增 `action_path`（带 `student_id`，仅内部 `/pages/` 路径）；更新 `api-parent/README.md` / `api-teacher/README.md` / `api-admin/README.md` 索引（如有）
- [x] [P2] 8.5 `POST /admin/task-activities` 契约补字段级示例（含 `class_ids` 上限/去重）
      <br>落地 2026-10-24（C-3）。字段级示例见 `backend/api/api-admin/10-tasks.md:401`（`"class_ids": [1, 2]`）；约束行见 `:418`——「**是** / 非空、**不得重复**、**≤50**、全部属于当前园所；任一不满足 → 422」。
- [ ] [P2] 8.6 主回应 `response_type` 契约说明：`single|period` 的语义与 supplement 固定 `single`
- [ ] 8.7 验证：契约字段与 design 溯源表一一对应；无 design 未声明字段；`points` 已从任务契约移除

## 9. 老师端

- [x] 9.1 更新 `teacher/src/api/teacher-tasks.js`：创建/编辑支持新字段；新增待回应、单条/周期汇总/个别补充回应、公开审核、分享码封装；验证：调用方无字段/类型错误
      <br>落地 2026-10-24 前（B/C-4）。封装齐备：`:27` 打卡列表（带 `params`，可传 `status`）、`:31` 单条回应、`:39` 周期汇总、`:49` 个别补充、`:59` 公开审核、`:69` 班级罐读取、`:74` 目标设置、`:80` 分享码；创建/编辑 `:11`/`:15` 直传对象，无字段裁剪。
- [x] 9.2 改造 `teacher/src/pages/tasks/index.vue`：创建表单增加学习关联、范围（班级/个体）、学生选择、频率、起止日期、仪式、星星范围/目标、允许公开；增加班级星星罐目标设置；`StarScope=class` 时隐藏任务 `StarGoal`，未配置班级罐时禁用班级星星选项；任务有提交后，编辑表单禁用频率/日期/学生/星星范围等结构字段，只允许标题/说明/学习关联/公开开关；关闭 `allow_public` 时提示“已公开内容将全部撤下”；列表展示待回应数量与星星进度，待回应按提交时间升序；移除积分与旧完成率
      <br>落地 2026-10-24（B + C-4）。2026-10-25 复核：`star-jar` / `待回应` / `star_scope` / `allow_public` / `learning_link` 在 `teacher/src/pages/tasks/` 共命中 55 处（`index.vue` 21、`detail.vue` 34）。C-4 本轮补的是班级罐卡片与进度（`index.vue` 目标设置、列表 `已集 N / M`），结构锁的只读态在 `detail.vue:933-970`。「目标设 0 的 toast」仍未在运行中小程序里抓到，见 3.8 注记与 C-5 待验清单。
- [ ] 9.3 新增 `teacher/src/pages/tasks/detail.vue`：提交列表、待回应/已回应筛选、贴纸/语音/星星回应、周期汇总、汇总后个别补充（班级星星不可选）、公开审核（含「内容中只有提交者孩子」确认）；语音用 `uni.getRecorderManager` 录制后上传 `/common/upload`（`biz_type=task`）；验证：真实点击完成贴纸、语音、周期汇总、汇总后补充，pending 提交不能补，未勾确认不能批准公开
- [ ] 9.4 在 `teacher/src/pages.json` 注册 `pages/tasks/detail`；核对老师端无 tabBar，跳转使用 `uni.navigateTo`；验证：真实点击进入详情
- [ ] 9.5 分享码：详情页调用 `GET /teacher/tasks/:id/share-card`，保存图片到相册并提供「转发到班级群」提示；验证：真实扫码进入家长端
- [x] 9.6 `TeacherTaskInput` 明确包含 `allow_public` / `recognition_ritual` / `star_scope` / `star_goal`，避免前端传入被后端丢弃（A 切片依赖）；验证：老师端创建/编辑请求字段不被丢弃，`go build` 通过
      <br>落地 2026-10-24 前。字段见 `teacher_portal_service.go:2862-2865`（`AllowPublic`/`RecognitionRitual`/`StarScope`/`StarGoal`，指针类型以区分「未传」与「传 false/0」）。`go build ./...`、`go vet ./...`、`golangci-lint` 在 C-4 收尾时均通过（handoff §24.4）。
- [ ] 9.7 验证：`npm run build:mp-weixin` 通过；`node scripts/check-tokens.mjs teacher` 对改动文件无新增违规；真实点击走完「创建个体任务 → 待回应 → 回应 → 审核公开 → 分享」

## 10. 家长端

- [ ] 10.1 更新 `parent/src/api/parent-task.js`：所有任务接口传 `student_id`；提交使用 `media_file_id`；新增公开申请/取消、家长反馈、读标记封装；任务详情/首页入口解析小程序码 `scene=t=<task_id>`，缺省回退 query `id`；验证：多孩切换后参数正确、扫码能定位任务
- [ ] 10.2 改造 `parent/src/pages/tasks/index.vue`：按当前孩子展示；状态为「待完成 / 已提交 / 已回应」；进度按该孩子计算；移除积分；任务 Tab 红点使用修正后的 `pending_tasks`；有补充回应未读时显示任务级红点
- [x] 10.3 改造 `parent/src/pages/tasks/detail.vue`：展示 `learning_link`、要求数组、主回应 + 补充回应（贴纸/语音/文字）、每条回应的家长反馈按钮、公开申请开关；打开详情清除补充回应未读标记；当前孩子不是消息目标孩子时提示可切换；新增「给孩子看」全屏展示态（放大贴纸/播放语音/老师的话）；移除积分与旧完成率；修复 `checked_in_today` / `requirements` 字段错误
      <br>落地 2026-10-23（B4-a/B4-b）。`requirements` 数组见 `parent/src/pages/tasks/detail.vue:447-448`，`learning_link` 见 `:452-456`；「给孩子看」见 `:480`/`:509`（全屏态 `:563`）；打开即 `markTaskRead` 见 `:198`（`markTaskRead` 从 `:13` 引入）；`my_reaction` 回显见 `:488`/`:516`。真实点击见 handoff §16.3：#3 进详情（`requirements` 渲染成 1./2. 两条、学习关联卡正常）、#6 打开详情写 `task_read_marks.last_read_at` 并渲染补充回应、#9「给孩子看」全屏态、#16 消息目标孩子≠当前孩子时提示 + 「切换查看」（§16.7 补验切换真的生效）。
- [ ] 10.4 在 `parent/src/pages/index/index.vue` 增加「今晚一件事」卡片：取当前孩子最近一条待完成任务；验证：真实点击进入详情
- [x] 10.5 家长反馈：主回应卡片下提供「孩子很开心 / 收到了 / 谢谢老师」三个一键反馈（`child_happy` / `received` / `thanks`）；补充回应可单独反馈但不进入主回应指标；同一家长可更新自己的反馈；验证：真实点击写入并回显，多个家长分别反馈不互相覆盖
      <br>落地 2026-10-23（B4-a）。三个选项见 `parent/src/pages/tasks/detail.vue:24-26`，主/补充回应各一组按钮见 `:488-492` / `:516-520`，回显依赖 `my_reaction`。真实点击见 handoff §16.3 #8：点「孩子很开心」→ 库 `task_response_reactions` 落 `(699, 61, child_happy)`、按钮 `active`、重进详情回显；§16.7 补验补充回应单独反馈写 `(700, 61, received)`，两组 active 互不串。
- [ ] 10.6 消息深链：家长端 `pages/messages/index.vue` 按 `action_path` 跳转；任务发布/主回应消息点击进入对应任务/孩子详情；只接受 `/pages/` 开头的内部路径；任务删除/结束显示提示；未登录保留路径，登录后回跳；验证：真实点击消息列表，多孩家庭进入正确孩子
- [x] [P2] 10.7 点消息打开任务详情时，同时把对应站内消息标记已读；若产品决定独立，写清不联动
      <br>落地 2026-10-23（B4-c 提前做，handoff §16.8 #1）。`parent/src/pages/messages/index.vue:35` 在跳转前调 `markMessageRead(item.id)`，失败只记日志不阻断跳转。真实点击复验：消息页「2条未读」→ 点任务消息 → 库 `is_read=1, read_at` 有值 → 返回显示「1条未读」（§16.8）。
- [x] 10.8 验证：`npm run build:mp-weixin` 通过；`node scripts/check-tokens.mjs parent` 对改动文件无新增违规；真实点击走完「提交 → 看到老师回应 → 反馈 → 申请公开 → 消息跳转」
      <br>落地 2026-10-23（B4-a）。handoff §16.3 全链真实点击：#4 一键提交 → #6 看到老师回应与补充回应 → #8 反馈落库 → #10/#11/#12 申请/取消/重新申请公开 → #13 消息页点任务卡片跳到 `detail?id=4322&student_id=27`；§16.3 末段记录 `npm run build:mp-weixin` 通过、`check-tokens.mjs parent` 对改动文件 0 违规。**不含真机**：真机扫码与真机多孩见 12.6。

## 11. 管理端

- [x] 11.1 改造 `admin/pages/tasks/index.vue`：新增创建/编辑表单（学习关联、班级多选、频率、日期、仪式、星星范围/目标、允许公开、班级星星罐目标）；独立任务创建与多班活动创建共用同一字段/默认值；多班活动调用活动端点，`activity_id` 由服务端返回；任务列表按 `activity_id` 分组展示；任务有提交后，编辑表单禁用频率/日期/班级/学生/星星范围等结构字段；关闭 `allow_public` 时提示“已公开内容将全部撤下”；班级罐进度封顶 100%；移除积分列与旧统计列
      <br>落地 2026-10-24（C-3，证据见 handoff §23.3）。两处口径按最新决策调整：①**班级罐目标是「班级管理 → 星星罐」的设置项**，任务表单只读取并前置校验，不在表单里写目标；②**「班级罐进度封顶 100%」未做**——管理端不展示星星进度（jar 视图不含进度，decision §20.7-5 归 C-4 老师端）。C-5 复核时以本节注记为准。
- [x] 11.2 接入增强后的 `GET /admin/tasks/stats`：参与率、回应率、待回应、平均回应时长、`child_happy_rate`、`ack_rate`、`feedback_coverage`、按班级/按活动明细；验证：真实数据与任务列表一致
      <br>落地 2026-10-24（C-3）。「平均回应时长」后端只提供 `median_response_hours`（中位数），页面显示中位数且不在前端自算平均。
- [x] 11.3 多班活动：调用 `POST /admin/task-activities` 创建，编辑只允许安全字段，删除整组；验证：选择 2 个班级后生成 2 条任务，改一次标题两条都变，删除活动两条都消失，统计按活动聚合
      <br>落地 2026-10-24（C-3）：验证用 **3 个班**（3 条任务同 `activity_id`），改标题三条同变、结束活动三条 `status=0`、组行只给「删除整个活动」。
- [x] 11.4 验证：`npm run build` 通过；管理端无任务积分/旧完成率展示；公开内容可撤回
      <br>落地 2026-10-24（C-3）：`npm run build` 通过 + CDP 真实点击 45 项断言全绿，撤回后 `public_status=revoked`、`public_visible=false`。

## 12. 测试与验证

- [x] 12.1 在 `docs/tests/TEST_CATALOG.md` 新增「亲子任务闭环 (TSK-LOOP)」模块，登记 `TC-NEW-TSK-0101` 至 `TC-NEW-TSK-0146`；验证：编号与 specs 引用一一对应
      <br>落地 2026-10-24 前。46 条已登记：`docs/tests/TEST_CATALOG.md:684`（`TC-NEW-TSK-0101` 班级任务对本班家庭可见）至 `:729`（`TC-NEW-TSK-0146` 更新个体任务目标跨班被拒），连续无缺号。
- [ ] 12.2 后端单测：范围/多孩/跨班/有效期/滚动 7 天/并发状态行/回应/周期汇总/汇总后补充（class_star 被拒、默认贴纸、response_type=single）/补充不进入指标/补充未读与读标记（首次 NULL、同秒 >=、按孩子、归属校验、upsert 并发、返回 last_read_at）/反馈按家长/星星 task/class 按主回应覆盖提交数、period 加 N 星、软删除不计入、补充不计入、超额封顶/班级罐三入口统一校验、目标 0 被拒、创建/更新为 class scope 未配置目标被拒/公开审核（内容边界、PublicScopeConfirmed、留痕、取消/撤回清空确认与审核人、allow_public 关闭清空批准/申请/审核人、任务结束不展示、跨班不可见、重开不复活）/取消后重审/创建任务字段持久化（老师端+管理端）与个体任务公开强制 false、班级改个体后公开关闭、更新目标跨班被拒/有提交后结构字段锁定（老师端与管理端共用 `task_rules.go` helper）/无积分/旧统计下线/通知按家长去重、创建任务响应返回发布计数、回应计数只统计主回应、action_path（student_id、内部路径、任务删除/结束、登录回跳）/空库 schema 唯一键/统计公式（single/period 分开，avg 不含 period）/pending 后端升序/`pending_over_48h`/活动分组与事务、class_ids 去重/上限/`POST /admin/tasks` 忽略 activity_id/安全字段批量编辑与单编辑限制/删除级联 read+state+supplement+reaction；验证：`go test ./internal/service -run 'Task|ParentTask' -timeout 600s` 通过
- [ ] 12.3 集成验证：老师创建个体任务 → 目标家庭可见、其他家庭不可见 → 家长提交 → 老师单条/周期汇总回应 → 家长看到回应并反馈 → 老师追加个别补充 → 家长看到任务红点并打开「给孩子看」→ 读标记清除；验证：真实点击 + 本地库核对 `response_id` / `task_response_supplements` / `task_response_reactions` / `task_checkin_states` / `task_read_marks`，并确认 `points` 字段/列不存在
- [x] 12.4 公开验证：班级活动默认私密 → 家长申请 → 老师审核 → 班级墙可见 → 撤回后消失 → 再次申请必须重新审核；个体任务无法公开；验证：真实点击 + 接口返回
      <br>落地 2026-10-23（B4-a/B4-b/B4-c）。真实点击与库核对见 handoff §18.3 #10–#14：不勾确认不能批准（库 `scope_confirmed` 仍 0）、批准写 `approved_at/by`、一次点击不撤（停「确认撤下？」）、确认后 `revoked_at` 有值、家长重新申请后勾选框回到 ☐；个体任务无法公开见 #7（`allow_public` 被后端强制为 0）。§16.6 #1 复验「老师关公开后家长仍能撤下」。班级墙页面 `parent/src/pages/tasks/wall.vue` 于 §16.9 补齐。
      <br>说明：本节真实点击均在 IDE 内完成，**真机扫码/真机公开墙仍属 12.6 的待办**，不因本勾而视为已验。
- [ ] 12.5 管理端验证：通过 `POST /admin/task-activities` 创建多班活动（含一个跨园所班级时整组失败）→ 按 `activity_id` 编辑安全字段一次全部班级 → 活动任务单删被拒 → 删除活动整组消失 → 统计出现活动聚合与参与/回应/反馈数据；撤回公开内容；验证：真实点击
- [ ] 12.6 分享码验证：老师生成家长端小程序码 → 开发/测试环境扫码进入对应版本家长端任务详情；验证：真实扫码，不从老师端进入；未配置 `code_env_version` 时有明确错误提示
- [ ] 12.7 回归验证：成长时光、通知、接送、积分页、管理端看板不受影响；任务流程不再产生积分流水；旧完成率/旧打卡人数不再出现；验证：关键页面 MCP 导航 + 真实点击抽查
- [ ] 12.8 收工验证：`go build ./...`、`go vet ./...`、两端 `npm run build:mp-weixin`、管理端 `npm run build`、`openspec validate --all` 通过；杀掉残留 `dev:mp-weixin` watcher；逐条对照 specs 全部场景，未通过项不得归档
