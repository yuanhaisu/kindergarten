# Proposal

## Why

亲子任务目前是一个「任务板 + 自报打卡」的半成品：老师创建任务后没有通知、没有查看打卡与回应的入口；家长打卡只写 `task_check_ins`，不产生积分流水、不更新积分账户；老师端页面不调用已有的统计/打卡接口；多孩家庭的任务列表与打卡归属会串孩子；过期任务仍可打卡。结果是家长不知道、老师看不到、孩子感知不到，家园共育的链路没有闭合。

本变更的业务目标不是留存或招生，而是建立一条低频、高质量的家园共育渠道：让家长知道幼儿园在教什么、在家怎么配合；让老师看见并回应；让孩子获得具体、即时、社交的认可。老师回应是园所内部管理要求，家长端不展示「等待老师回应」这种内部状态。

## What Changes

- **任务范围**：支持班级任务与个体任务。个体任务只对目标孩子的家庭可见；班级任务对本班家庭可见；多孩家庭按当前孩子隔离任务与提交。
- **互动记录**：家长可一键完成、写一句话或上传一张照片；默认仅本家庭与班级老师可见。
- **老师回应（内部管理）**：家长提交后进入老师端内部「待回应」队列；家长端只显示「已提交」或「老师已回应」，不显示待回应。回应按任务类型执行：
  - 个体任务、一次性活动：逐条回应。
  - 每日/每周习惯：按周期汇总回应，一次回应关闭该周期内全部待回应提交。
- **回应质量信号**：家长可对老师回应做一键反馈（孩子很开心 / 收到了 / 谢谢老师）；反馈按家长维度保存，一次班级汇总回应可被多个家庭分别反馈。
- **汇总与个别补充**：周期汇总回应后，老师仍可对单个孩子的提交追加个别贴纸/语音；补充回应存独立表，不改变主回应关闭状态；补充回应只支持贴纸/语音，`class_star` 被拒绝，避免“加了星但不计星”的空操作。
- **回应仪式**：贴纸 / 5 秒语音 / 班级星星；老师创建任务时选择默认仪式，回应时可调整。
- **星星罐范围**：支持按任务、按班级两种范围；按班级的星星罐跨任务累积、按学期重置、目标独立配置；创建按班级范围任务时必须已有班级罐目标，否则拒绝创建并返回引导；teacher/admin/activity 三个创建入口统一在 service 校验；星星罐按**被主回应覆盖的提交数**计数（period 回应覆盖 N 条就加 N 星），补充回应不计入。
- **活动授权公开**：班级活动默认私密；家长可申请公开，老师审核通过后才在班级墙可见；可撤回；取消或撤回后再次申请必须重新审核，不能沿用旧批准；个体任务永不公开。
- **移除积分**：任务流程不再发放积分；任务接口、页面、积分规则与任务统计中移除任务积分字段与「任务打卡」规则。开发阶段不做版本兼容。
- **通知与触达**：任务发布通知家长；批量回应按家长汇总为一条消息；老师可生成「家长端小程序码 / 任务海报」转发到班级微信群。
- **管理端**：增加活动创建/编辑入口；多班活动由服务端生成 `activity_id` 分组，编辑只允许安全字段（标题/说明/学习关联/公开开关），删除按活动整组处理，统计按活动聚合。
- **统计口径**：下线旧 `completion_rate` / `total_possible` / `checkin_count` / `not_checkin_count`；统一为参与率、回应率、待回应数、`child_happy_rate`、`ack_rate` 与 `feedback_coverage`；补充回应单独统计，不进入主回应指标。
- **班级星星罐**：目标独立配置；任务 `StarGoal` 只用于按任务范围，按班级范围不再由任务 upsert 目标。
- **孩子感知**：家长端回应卡片提供「给孩子看」展示态（大贴纸/播放语音/老师的话）。
- **通知计数**：任务发布与主回应的 `notified_count` / `notify_failed_count` 都按家长用户去重，不按提交条数；补充回应不单独发通知，但必须在任务上产生未读提示，家长下次打开能看到。
- **反馈语义**：`child_happy` 作为孩子感知质量信号；`received` / `thanks` 作为家长确认；增加 `feedback_coverage`，避免“只有 1 条反馈 = 100% 正向”的误读。
- **统计公式**：`response_rate` = 有主回应的提交数 / 总提交数；`participation_rate` = 至少有一条提交的目标学生数 / 目标学生数；`avg_response_hours` = 仅 `single` 主回应的“提交到回应”中位数；`period` 汇总不计入该指标。
- **管理端活动**：`POST/PUT/DELETE /admin/task-activities` 必须校验班级归属、创建/删除使用事务；活动任务禁止走单条 `DELETE /admin/tasks/:id`；班级星星罐通过 `GET/PUT /admin/classes/:id/star-jar` 维护。
- **提交唯一性**：空库直接按 `unique (task_id, student_id, date)` 建 `task_check_ins`；每周/一次性频率由 `task_checkin_states` 状态行串行化；不引入 `ActiveKey`、迁移、回填或 feature flag。任务删除时级联软删除提交/回应，不提供单条恢复。
- **未读模型**：新增 `task_read_marks(task_id, user_id, student_id, last_read_at)`，唯一 `(task_id, user_id, student_id)`；`POST /parent/tasks/:id/read` 必须带 `student_id` 并校验归属；未读 = `last_read_at IS NULL OR 补充回应.created_at > last_read_at`，多孩各自独立。
- **消息深链**：`messages` 增加 `action_path`；任务发布/主回应消息写入 `/pages/tasks/detail?id=<task_id>&student_id=<student_id>`，确保多孩家庭打开正确孩子；`action_path` 只允许 `/pages/` 开头的内部路径；保留 `reference_id` 用于幂等。
- **公开墙边界**：老师审核公开时必须勾选确认「内容中只有提交者孩子」（图片/语音/文字均适用）；系统记录 `PublicScopeConfirmed` 与审核人/时间；含其他孩子的照片不得进入班级墙。公开墙只展示提交者孩子的内容，并 MUST 按当前孩子所在班级过滤，跨班不可见。家长取消或老师撤回时 MUST 同时清空 `PublicScopeConfirmed`；再次申请必须重新确认。任务 `allow_public` 改为 false 时 MUST 撤销该任务全部公开批准/确认、清空 `IsPublicRequested`；公开墙查询 MUST 校验任务级 `allow_public` 且任务仍在进行；重新打开不得恢复旧批准，家长必须重新申请。
- **老师待回应兜底**：老师端待回应按提交时间排序；管理端统计超过 48h 未回应数，不做自动升级。
- **创建与编辑边界**：创建时必须持久化 `allow_public` / `recognition_ritual` / `star_scope` / `star_goal`，`allow_public` 默认 false、个体任务强制 false；更新为个体任务（`target_student_id > 0`）时同样强制 `allow_public=false` 并撤销公开；`target_student_id` 创建或更新时都必须校验目标孩子属于当前班级/园所；任务产生第一条提交后，锁定 `frequency`、起止日期、`target_student_id`、`class_id`、`star_scope`、`star_goal`；只允许改标题/说明/`learning_link`/`allow_public`。要改结构必须新建任务；老师端与管理端创建/更新入口共用同一校验，不能只做 UI。
- **共享规则归属**：班级罐校验、结构锁、公开撤销收敛到 `server/internal/service/task_rules.go` 的包级 helper，`TeacherPortalService` 与 `TaskService` 都调用，不在两个 service 各写一套；创建与更新为 `StarScope=class` 时都必须校验班级罐目标。
- **删除级联**：任务删除时清理 `task_read_marks` 与 `task_checkin_states`，不留孤儿。
- **媒体**：任务照片/语音统一通过 `/common/upload` 上传，接口使用 `media_file_id`；服务端校验文件归属；任务删除时级联软删除提交与回应。
- **修正闭环前置缺陷**：`learning_link` 必填、多孩 `student_id`、班级归属校验、有效期与滚动 7 天频率、周期提交并发状态行、过期不可打卡、进度按孩子、`requirements` 数组、首页 `pending_tasks` 口径、软删除孤儿数据。
- **本次将修改 `backend/api/`，属于契约变更**（家长端/老师端/管理端亲子任务契约；新增回应、公开审核、分享码、统计字段；移除任务积分字段）。

### Out of Scope

- 积分体系、排行榜、成就自动解锁、礼物兑换商城。
- 成长时光自动沉淀（P1）。
- 全园共享活动墙（P0 只做班级墙）。
- 园所级内容模板库（P1）。
- 微信订阅消息（P0 用站内消息 + 首页卡片 + 老师群海报/小程序码）。
- 家长端任务推荐、AI 生成内容。
- 教师回应超时自动升级（P0 只做待回应队列与提醒标识）。
- 任务附件、材料清单、家庭多成员协作。
- 媒体文件的物理删除与清理任务（P0 做归属校验与级联软删除；物理清理 P1）。

## Capabilities

### New Capabilities

- `parent-task-loop`：亲子任务从发布、家长提交、老师内部回应、家长质量反馈到孩子感知的闭环；包含个体/班级范围、私密/审核公开、星星罐多范围、任务积分移除、通知聚合与管理端统计。

### Modified Capabilities

（无。`media-assets` 的引用契约不在本期修改；任务媒体复用 `/common/upload` 的 `file_id`，不新增媒体引用表。）

## Open Decisions / Conflicts

1. **回应仪式、回应类型与星星范围取值**：`sticker` / `voice` / `class_star`、`single` / `period`、`task` / `class` 是本次新增的产品语义，`design-data/`、`dict.go` 与现有常量均无来源；来源为 2026-10-20 产品决策。贴纸具体键值由客户端常量维护，服务端只存字符串。补充回应不再使用 `link_type` 枚举，单独表表达。
2. **家长反馈取值**：`child_happy` / `received` / `thanks` 为本次新增的产品语义，来源为 2026-10-20 产品决策；`child_happy` 是孩子感知质量信号，`received` / `thanks` 是家长确认，统计时分开。
3. **星星罐学期口径**：按班级的星星罐按当前学期重置；学期取 `KindergartenAppSetting.CurrentSemester`，为空时按自然年。
4. **公开审核**：P0 由老师审核家长公开申请；管理端可查看与撤回，不做独立审核队列。
5. **管理端多班活动**：服务端生成 `activity_id`，为每个班级创建一条任务并共享该 ID；批量编辑只允许标题/说明/学习关联/公开开关，删除按活动整组处理；空 `activity_id` 视为独立任务；统计按活动聚合。
6. **家长端小程序码**：老师端与家长端是不同 AppID，现有 `GetWXACodeUnlimit` 固定走老师端凭据，不能直接复用；需要新增走家长端凭据的生成方法，否则扫码会进入老师端。
7. **媒体保留**：P0 只做归属校验与软删除；物理文件保留到学期结束 + 30 天，清理任务 P1。
8. **反馈与补充回应**：反馈按家长维度存 `task_response_reactions`；汇总回应后允许 `task_response_supplements` 追加个别补充回应，主回应仍用于待回应判定；补充回应不进入主回应指标、不单独发通知。
9. **并发**：周期提交用 `task_checkin_states` 状态行在事务内串行化，不依赖单一日期唯一索引。
10. **班级星星罐目标**：独立配置在 `class_star_jars`，通过老师/管理端班级设置维护；目标必须 >0，设为 0 视为非法；任务的 `StarGoal` 只在 `StarScope=task` 时生效；创建 `StarScope=class` 任务时必须已有班级罐目标，否则拒绝创建并返回“请先配置班级星星罐”的引导。
11. **活动分组**：`ParentTask.ActivityID` 为空表示独立任务；多班活动共享同一 `activity_id`，编辑与统计按该字段聚合。
12. **分享码环境**：新增 `wechat.code_env_version`（`release` / `trial` / `develop`），开发/测试环境使用对应版本；家长端解析 `scene=t=<task_id>`，缺省回退 query `id`。
13. **孩子展示态**：家长端提供全屏「给孩子看」卡片，放大贴纸/播放语音/显示老师的话；不新增孩子独立账号。
14. **提交唯一性**：`task_check_ins` 唯一 `(task_id, student_id, date)`；`weekly` / `once` 由 `task_checkin_states` 行锁串行化；空库直接建 schema，不引入 `ActiveKey` / 迁移 / 回填。
15. **质量指标分母**：按 `(response_id, parent_user_id)` 对计算，避免一个班级汇总回应被 20 个家长反馈时比率超过 100%。`feedback_coverage` = 有反馈的 (回应, 家长) 对数 / 应反馈的 (回应, 家长) 对数；`child_happy_rate` = `child_happy` 反馈数 / 应反馈对数；`ack_rate` = (`received` + `thanks`) / 应反馈对数。应反馈对数 = 主回应覆盖的去重家长数之和（`notified_count + notify_failed_count`，含通知失败）。补充回应不进入这三个指标。
16. **补充回应**：`task_response_supplements(response_id, checkin_id)` 只存补充关系；主回应只由 `TaskCheckIn.ResponseID` 表达；补充回应的 `response_type` 固定为 `single`；只支持贴纸/语音，`class_star` 被拒绝；未显式指定仪式时默认贴纸，不继承任务默认仪式；不单独发通知，不进入回应率/质量指标/星星罐；任务列表/详情必须有未读标记。
17. **统计公式**：`response_rate` / `participation_rate` / `avg_response_hours` 按 proposal What Changes 的公式实现；`participation_rate` 分母为目标学生数；`avg_response_hours` 仅统计 `single` 主回应，`period` 不计入；补充回应不计入。
18. **管理端活动边界**：班级归属校验、创建/删除事务、活动任务禁止单条删除与单条 unsafe 编辑；`POST /admin/tasks` 强制忽略/拒绝客户端传入的 `activity_id`；一个班级失败时整组回滚，不产生半个活动。
19. **补充回应提示**：不单独推送，但任务列表/详情必须有未读标记；家长下次打开能看到，不依赖推送。
20. **未读模型**：`task_read_marks(task_id, user_id, student_id, last_read_at)`，唯一 `(task_id, user_id, student_id)`；`POST /parent/tasks/:id/read` 必须带 `student_id` 并校验归属，返回 `{ last_read_at }`；读标记用 upsert（`ON DUPLICATE KEY UPDATE` / 捕获唯一冲突重试），并发不 500；未读 = `last_read_at IS NULL OR 补充回应.created_at >= last_read_at`；多孩各自独立。
21. **消息深链**：`messages` 增加 `action_path`；任务发布/主回应消息写入 `/pages/tasks/detail?id=<task_id>&student_id=<student_id>`；个体任务用目标孩子，班级任务用该家长在该班的一个确定孩子，回应消息用覆盖提交中的第一个孩子；`action_path` 只允许 `/pages/` 开头；任务已删除/已结束时客户端提示“任务已结束/不存在”；未登录时保留 `action_path`，登录后回跳；`reference_id` 继续用于幂等。
22. **公开墙边界**：老师审核时必须确认「内容中只有提交者孩子」（图片/语音/文字均适用）；含其他孩子的内容不得公开；公开墙只展示提交者孩子的内容；可见性查询 MUST 同时满足任务 `allow_public=true`、任务未删除、仍在进行，且 `Task.ClassID = 当前孩子班级`。
23. **公开确认留痕**：`TaskCheckIn.PublicScopeConfirmed` 由老师勾选；批准公开时 MUST 为 true，并记录 `PublicApprovedBy/At`；家长取消/老师撤回时 MUST 置 false 并清空 `PublicApprovedBy/At`；任务 `allow_public` 关闭时 MUST 批量清空批准/确认/审核人、`IsPublicRequested=false` 并写 `PublicRevokedAt`，重新打开不得恢复。
24. **待回应兜底**：老师端待回应按提交时间升序；管理端统计 `pending_over_48h`；不做自动升级，但园长可见。
25. **有提交后锁结构字段**：任务产生第一条提交后，`frequency` / 起止日期 / `target_student_id` / `class_id` / `star_scope` / `star_goal` 一律只读；只允许标题/说明/`learning_link`/`allow_public`；更新为个体任务时强制 `allow_public=false` 并撤销公开。
26. **共享规则归属**：`validateTargetStudent` / `validateClassStarJar` / `assertTaskEditable` / `revokePublicForTask` 放在 `server/internal/service/task_rules.go` 的包级 helper；老师端与管理端 service 共用，禁止复制规则；创建与更新为 `StarScope=class` 都必须走 `validateClassStarJar`，创建与更新 `target_student_id` 都必须走 `validateTargetStudent`。

## Delivery Slices

本次仍是一个 OpenSpec change，schema 一次到位；A/B/C **只是实现与验收顺序，不是可独立上线/回滚的交付单元**。A 未通过不得进入 B，B 未通过不得进入 C，C 完成后才 archive。

| 切片 | 范围 | 验收门 |
| --- | --- | --- |
| A · 核心闭环 | 任务范围（班级/个体）、提交、主回应、多孩/频率、移除积分、家长/老师核心页面、基础契约 | 两端构建通过；个体/班级隔离、多孩、频率、无积分、主回应闭环真实点击通过 |
| B · 隐私与感知 | 公开审核、媒体归属、补充回应、给孩子看、未读读标记、分享码/家长端小程序码、消息深链 | 公开墙审核/撤回、补充未读、孩子展示态、扫码进入家长端、消息跳转通过 |
| C · 集体与管理 | 班级星星罐、管理端活动分组、管理端统计/反馈指标、班级罐设置 | 活动事务/跨园所校验、星星罐 task/class、统计公式、看板真实数据通过 |

> 如果排期紧张，优先保 A；B/C 可以延后，但不要在一个 PR 里混合验收。

## Contract Field Provenance

| 字段 | 来源 | 动作 | 为什么不能派生 |
| --- | --- | --- | --- |
| `target_student_id` | 产品决策：个体任务；现有 `GrowthDailyLog.TargetType` 的个体语义可类比 | 新增 | 无法从班级或打卡记录推断任务原本只发给哪个孩子 |
| `learning_link` | 产品决策：任务必须说明与幼儿园学习的关系；见本 change 的 Why | 新增，必填 | 标题/描述不保证包含这层关系，客户端无法可靠生成 |
| `allow_public` | 产品决策：活动可申请公开；`design-data/04-家长端-亲子任务页.json` 没有公开字段 | 新增 | 默认私密是行为规则，不能由班级或任务类型推断 |
| `recognition_ritual` | 2026-10-20 产品决策：贴纸 / 语音 / 班级星星 | 新增 | 老师的选择是任务配置，不能从 `frequency` 派生 |
| `star_scope` | 2026-10-20 产品决策：按任务 / 按班级 | 新增 | 范围是老师配置，不能从任务类型推断 |
| `star_goal` | 产品决策：星星罐目标 | 新增 | 目标由老师设置，不能由班级人数或任务天数推断 |
| `is_public_requested` | 产品决策：家长申请公开 | 新增（提交） | 申请是家长动作，不能由任务 `allow_public` 自动推导 |
| `public_approved_at` / `public_approved_by` | 产品决策：老师审核公开 | 新增（提交） | 审核是老师动作，不能由申请自动推导 |
| `public_scope_confirmed` | 产品决策：老师确认内容中只有提交者孩子 | 新增（提交） | 是老师的确认动作，不能由审批时间推断 |
| `response_id` | 产品决策：老师回应是闭环状态 | 新增（提交） | 是闭环状态本身，不能从打卡时间派生 |
| `response_content` / `response_media_file_id` / `sticker_key` | 产品决策：老师回应的三种载体 | 新增（回应） | 回应内容是老师输入，不能派生 |
| `parent_reaction` | 产品决策：家长质量反馈，按家长维度保存；`child_happy` / `received` / `thanks` | 新增（回应反馈） | 是家长动作；同一回应可被多个家庭分别反馈，不能存在回应单字段上 |
| `activity_id` | 服务端生成的管理端多班活动分组；现有设计稿没有活动实体 | 新增 | 无法从标题/日期可靠推断哪些班级任务属于同一次活动；客户端不得自拟 |
| `class_star_jar.goal` | 产品决策：班级星星罐目标独立配置 | 新增（班级设置） | 任务 `StarGoal` 只适用于按任务范围，不能同时代表班级罐目标 |
| `points` | 既有字段；本变更从任务契约、页面与数据库删除 | 删除（字段与列） | 开发阶段无历史数据，直接移除 |
| `has_unread_supplement` | 未读模型：`last_read_at IS NULL OR 补充回应.created_at > last_read_at`，按当前孩子 | 新增（列表/详情展示） | 未读是家长读状态，不能由提交/回应时间独立派生 |
| `action_path` | 消息深链：任务/回应消息跳转任务详情，带 `student_id`，仅允许内部 `/pages/` 路径 | 新增（messages） | 仅靠 `type` + `reference_id` 无法区分回应消息对应的任务与孩子 |

## Impact

| 层 | 位置 | 变更 |
| --- | --- | --- |
| Go domain | `server/internal/domain/task.go` / `message.go` | `ParentTask` 增加 `ActivityID` / `TargetStudentID` / `LearningLink` / `AllowPublic` / `RecognitionRitual` / `StarScope` / `StarGoal`，删除 `Points`；`TaskCheckIn` 增加 `MediaFileID` / `MediaURL` / `IsPublicRequested` / `PublicScopeConfirmed` / `PublicApprovedAt` / `PublicApprovedBy` / `PublicRevokedAt` / `ResponseID` / `DeletedAt`，删除 `Points` / `ImageURL`；新增 `TaskResponse`、`TaskResponseSupplement`、`TaskResponseReaction`、`ClassStarJar`、`TaskCheckInState`、`TaskReadMark`；`Message` 增加 `ActionPath` |
| repository | `server/internal/repository/task_repo.go` | 个体/班级/活动查询、待回应统计、按孩子统计、星星罐进度、公开审核、软删除级联；旧统计与任务积分统计下线 |
| service | `server/internal/service/parent_portal_service.go` / `teacher_portal_service.go` / `task_service.go` / 新增 `task_rules.go` | 任务范围、提交、主回应/补充回应、周期汇总、家长反馈、公开审核、班级星星罐、通知聚合、活动分组、分享码、移除积分与旧统计；共享规则收敛到 `task_rules.go` |
| handler | `server/internal/handler/http/parent_portal.go` / `teacher_portal.go` / `task.go` | 任务接口字段与校验；回应/补充回应/公开审核/反馈/分享码；管理端活动与统计 |
| router / Casbin | `server/internal/app/router.go` / `configs/rbac_policy.csv` | 新增老师回应、补充回应、公开审核、班级星星罐、家长反馈、分享码、管理端活动/统计路由与策略 |
| API 契约 | `backend/api/api-parent/06-task.md` / `api-teacher/09-class-and-tasks.md` / `api-admin/10-tasks.md` / `api-common.md` | 新增字段与端点；补充回应；移除任务积分与旧统计；新增管理端活动分组/班级星星罐/统计；消息增加 `action_path` |
| 家长端 | `parent/src/pages/tasks/index.vue` / `detail.vue` / `pages/index/index.vue` / `pages/messages/index.vue` / `api/parent-task.js` | 列表/详情、首页卡片、公开申请、老师回应与家长反馈、未读红点、消息 `action_path` 跳转、多孩 `student_id`、无积分 |
| 老师端 | `teacher/src/pages/tasks/index.vue` / 新增 `detail.vue` / `api/teacher-tasks.js` / `pages.json` | 个体任务、学习关联、待回应/周期汇总、贴纸/语音/星星、公开审核、分享码 |
| 管理端 | `admin/pages/tasks/index.vue` / `admin/api/tasks.ts` / `admin/pages/points` | 活动创建/编辑、`activity_id` 分组、参与/回应/反馈统计；移除任务积分规则 |
| 微信能力 | `server/internal/auth/wechat.go` / `configs/app.ini` / `teacher_service.go` | 新增走家长端凭据的 `GetWXACodeUnlimit` 与 `code_env_version` 配置；老师端生成家长端小程序码 |
| design token | — | 不新增 token；新组件使用既有语义 token，`check-tokens` 验证 |
| 数据 | `parent_tasks` / `task_check_ins` / `messages` / 新增 `task_responses` / `task_response_supplements` / `task_response_reactions` / `task_checkin_states` / `class_star_jars` / `task_read_marks` | 空库直接 AutoMigrate 建表；`task_check_ins` 唯一 `(task_id, student_id, date)`、加 `public_scope_confirmed`；`messages` 加 `action_path`；无手工迁移/回填；用 `go run ./cmd/dbcheck` 核实 |

**风险**：老师内部回应队列如果不可见或操作过重，会卡住闭环；P0 用待回应队列 + 周期汇总 + 一键贴纸降低摩擦。公开墙涉及儿童影像，必须老师审核后公开。老师端与家长端 AppID 不同，分享码必须走家长端凭据。开发阶段无历史数据，积分字段/列直接删除，不留兼容代码。
