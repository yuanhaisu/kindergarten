# 亲子任务闭环 A 切片 Handoff

> 用途：把当前实现状态、未完成项、验证步骤交给新会话，避免在 37 MB / 3 次 compact 的旧会话里继续。
> 写于：2026-10-21
> 分支：`feat/parent-growth-scope`
> 写文档时 HEAD：`67fbb51`（仓库有并发提交，执行前先 `git log -3 --oneline` 确认）

## 1. 目标

把亲子任务从「任务板 + 打卡 + 积分」改成「家园共育回应链」：

> 幼儿园给一个明确的家庭动作，家长做完留下记录，老师回应，记录沉淀到孩子的成长里。

A 切片只做**闭环成立所需的最小集合**：班级/个体任务、提交、老师回应、删除级联、无积分、频率与范围校验。

## 2. 已定产品决策

| 决策 | 结论 |
|---|---|
| 业务目标 | 家园共育渠道，联动幼儿园学习与家庭行为，增进家长对幼儿园教育的认可 |
| 留存 | 不作为目标；不做每日签到、连续打卡、积分排行榜 |
| 任务类型 | 习惯与活动都存在；习惯按周/低频，活动一次性 |
| 老师反馈 | 必须；A 切片用「待回应」状态承载，园所考核未实现 |
| 内容来源 | 老师按个别学生创建；园所创建活动（活动属后续切片） |
| 积分 | 从亲子任务移除；不写积分流水、不更新积分账户 |
| 公开 | 默认私密；班级任务可申请公开；个体任务不可公开 |
| 星星 | 任务星星 / 班级星星两种范围；A 只开放按任务 |

## 3. A 切片范围

已纳入：

- 班级任务、个体任务（`target_student_id`）
- 学习关联 `learning_link` 必填
- 提交：一键、文字、照片（`media_file_id`）
- 频率：每日 / 每周（滚动 7 天）/ 一次性
- 老师：待回应列表、单条回应、批量回应、贴纸/语音/班级星星
- 默认私密与公开开关（申请/审核链路未做）
- 删除级联
- 无积分
- 家长端任务列表/详情/首页“今晚一件事”
- 老师端任务列表/创建/详情
- 契约、TC 登记、nullable 迁移文件

不纳入 A：

- 发布通知与 `action_path`
- 公开申请/老师审核
- 补充回应、家长反馈、未读标记
- 班级星星罐管理入口
- 园所活动创建 / 多班活动 / 活动统计
- 管理端任务页面重构

## 4. 已实现

### 4.1 数据模型

`server/internal/domain/task.go`：

- `ParentTask`：`TargetStudentID`、`LearningLink`、`AllowPublic`、`RecognitionRitual`、`StarScope`、`StarGoal`、`ActivityID`；删除 `Points`。
- `TaskCheckIn`：`MediaFileID`、`MediaURL`、公开申请字段、`ResponseID`；删除 `Points` / `ImageURL`。
- 新表：
  - `TaskResponse`：老师回应（single / period）
  - `TaskResponseSupplement`：汇总回应后的个别补充（A 预留）
  - `TaskResponseReaction`：家长对回应的反馈（A 预留）
  - `TaskCheckInState`：频率串行化状态
  - `ClassStarJar`：班级星星罐（A 预留）
  - `TaskReadMark`：补充回应未读（A 预留）

`server/internal/domain/message.go`：新增 `ActionPath`。

`server/internal/app/provider.go`：AutoMigrate 已加入上述新表。

### 4.2 家长端后端

`server/internal/service/parent_portal_service.go`：

- `taskScopeQuery`：班级任务按当前孩子班级；个体任务只按目标孩子。
- `ListTasks`：只返回有效任务（`status=1` 且在起止日期内）；返回 `submitted_current_period`、`checked_in_today`（同值）、`checkin_count`（当前孩子累计）。
- `GetTask`：校验有效期和孩子范围；`requirements` 反序列化为数组；返回老师回应列表。
- `CheckinTask`：
  - 频率：每日一天一次；每周滚动 7 天一次；一次性只能一次。
  - `task_checkin_states`：`OnConflict DoNothing` + `SELECT ... FOR UPDATE`。
  - `media_file_id` 必须属于当前用户、`biz_type=task`、image/audio。
  - 不写积分；返回“已提交，等待老师回应”。
- `CheckinHistory`：返回 `media_url`、`response_id`。

### 4.3 老师端后端

`server/internal/service/teacher_portal_service.go`：

- `ListTasks`：返回 `pending_responses`（全量待回应）与 `pending_period`（**本周期内**的待回应条数，与批量的筛选条件一致）、`period_start` / `period_end`；两者不等时列表显示「待回应 N（本周期 M）」。不支持汇总回应的任务（一次性、个体、未知频率）`pending_period` 恒等于全量，不能当成「可以一键关闭」。
- `GetTask`：任务详情，供编辑页使用。
- `CreateTask`：校验标题、学习关联、目标孩子、频率、日期、星星罐；个体任务强制 `allow_public=false`。
- `UpdateTask`：有提交后只允许改标题/说明/学习关联/公开开关；改个体或关公开时撤回公开。
- `DeleteTask`：事务级联删除并写操作日志。
- `TaskStats`：提交数 / 已回应 / 待回应。
- `TaskCheckins`：`pending` / `responded`。
- `RespondTask`：只回应 `response_id=0` 的提交。
- `BatchRespondTask`：只覆盖**当前周期**（每日=当天；每周=滚动 7 天；任务结束后取最后一天/最后一周）的待回应提交，仅限每日/每周的班级任务；返回 `failed_ids`，全失败返回 500，无候选返回 200 + `id=0` 空操作。
- `GetTask` 返回 `period_start` / `period_end`；老师端把区间外的待回应标注为「旧周期 · 请逐条回应」。
- `task_check_ins` 新增 `(task_id, response_id, date)` 复合索引，缩小 `FOR UPDATE` 的锁范围。

### 4.4 共享规则

`server/internal/service/task_rules.go`：

- `validateTargetStudent`
- `validateClassStarJar`
- `assertTaskEditable`
- `revokePublicForTask`
- `deleteTaskCascade`
- `sameTaskDay`
- `validTaskFrequency`

### 4.5 前端

家长端：

- `parent/src/api/parent-task.js`：支持 `student_id`、`media_file_id`。
- `parent/src/pages/tasks/index.vue`：按当前孩子取数；卡片显示“已提交/待完成”；无积分。
- `parent/src/pages/tasks/detail.vue`：学习关联、要求数组、老师回应、一键提交、文字、照片上传。
- `parent/src/pages/index/index.vue`：首页“今晚一件事”。

老师端：

- `teacher/src/api/teacher-tasks.js`：详情、checkins、单条回应、批量回应。
- `teacher/src/pages/tasks/index.vue`：创建表单含范围/频率/日期/仪式/星星目标/公开开关；列表显示待回应数。
- `teacher/src/pages/tasks/detail.vue`：编辑、待回应/已回应、贴纸、语音、批量送星。
- `teacher/src/pages.json`：注册 `pages/tasks/detail`。

### 4.6 契约、测试、迁移

- `backend/api/api-parent/06-task.md`
- `backend/api/api-teacher/09-class-and-tasks.md`
- `backend/api/api-admin/10-tasks.md`
- `backend/api/api-common.md`（消息 `action_path`）
- `docs/tests/TEST_CATALOG.md`：TC-NEW-TSK-0101～0146
- `server/internal/service/task_rules_test.go`
- `docs/migrations/2026-10-21-task-checkin-state-nullable.sql`
- `docs/migrations/2026-10-21-drop-task-points-columns.sql`（已执行）

## 5. 关键文件

后端：

```
server/internal/domain/task.go
server/internal/domain/message.go
server/internal/service/task_rules.go
server/internal/service/parent_portal_service.go
server/internal/service/teacher_portal_service.go
server/internal/service/task_service.go
server/internal/service/point_service.go
server/internal/handler/http/parent_portal.go
server/internal/handler/http/teacher_portal.go
server/internal/handler/http/task.go
server/internal/app/router.go
server/internal/app/provider.go
server/internal/app/wire_gen.go
```

前端：

```
parent/src/api/parent-task.js
parent/src/pages/tasks/index.vue
parent/src/pages/tasks/detail.vue
parent/src/pages/index/index.vue
teacher/src/api/teacher-tasks.js
teacher/src/pages/tasks/index.vue
teacher/src/pages/tasks/detail.vue
teacher/src/pages.json
```

OpenSpec：

```
openspec/changes/2026-10-20-redesign-parent-task-loop/
```

> 该目录已入库（文件全部跟踪，2026-10-22 核对）。早前「未跟踪、清理工作区前先备份」的提示已过期。

## 6. 当前验证状态

| 项 | 状态 |
|---|---|
| `go build ./...` | ✅ 2026-10-21 通过 |
| `go vet ./...` | ✅ 2026-10-21 通过 |
| `go test ./internal/service/ -run 'TestValidTaskFrequency|TestSameTaskDay'` | ✅ 通过 |
| `go test ./internal/service/`（全包） | ✅ 通过（约 280s） |
| `parent npm run build:mp-weixin` | ✅ 通过（含本轮 student_id 改动） |
| `teacher npm run build:mp-weixin` | ✅ 2026-10-21 通过（含 onShow 修复） |
| nullable 列 | ✅ `last_check_in_date` 存在且 `Null=YES`，无需迁移 |
| 核心链路 | ✅ API 级 14 步已跑（见下） |
| 提交并发（0107 频率约束） | ✅ 2026-10-21 补测：10 并发提交只成功 1 次（201×1 / 422×9，无 5xx），历史仅 1 条。**不是 0111** |
| UI 交互（家长端） | ✅ 微信开发者工具 + 真实点击（CDP 鼠标事件）：首页卡片→详情→一键提交→返回刷新→文字提交→老师回应渲染 |
| UI 交互（老师端） | ✅ 列表/详情渲染、单条贴纸回应、一键送星（本周期，两条提交共享 response）、返回刷新；创建表单实际提交与语音回应未覆盖 |
| TC 0101–0146 | A 切片 ✅，但 **0111 仅并发路径已验证（⚠️）**；B/C ⏸；明细见 `docs/tests/TEST_CATALOG.md`「A 切片执行记录」 |

本轮实测发现并修复 6 个缺陷：

1. `CheckinTask` 更新了不存在的列 `last_checkin_date`（实际列 `last_check_in_date`），所有打卡 500。已修，并同步修正迁移 SQL、OpenSpec 与本 handoff。
2. 家长端任务接口在 `student_id` 不属于该家长时静默回退到第一个孩子，违反规格「多孩家庭按当前孩子隔离」。已新增 `parentTaskCtx`：列表/详情/提交/历史统一要求 `student_id` 且必须归属当前家长；前端首页与任务页先用 overview 落盘当前孩子再取任务。
3. `CheckinHistory` 未套任务可见范围：请求别的孩子的个体任务历史返回 200 空列表。已改为先过 `taskScopeQuery`，跨家庭直接访问返回 404。
4. 本地缓存的孩子被解绑后任务页持续 403、详情页仍渲染提交入口。已改为 403 时刷新 overview 重试一次；详情增加错误态，无 `detail.id` 时不渲染提交入口。
5. 任务列表用 `onLoad` 取数，从详情页提交后返回不刷新，卡片停在「待完成 / 0N次」。已改为 `onShow`，返回即刷新（H5 点击验证：提交后返回，卡片立刻转为已提交口径）。
6. 老师端任务列表同样 `onLoad`，详情页回应后返回「待回应 N」不刷新；别处创建的任务也不会出现（本页 `addTask` 创建后会主动重拉，不受影响）。已改为 `onLoad(loadStudents)` + `onShow(loadTasks)`，并在微信开发者工具复验。

核心链路证据（curl，2026-10-21）：班级任务 11、个体任务 12；家长 11（孩子 4）与 20（孩子 1，同班）可见性；家长 12（孩子 21，异班）不可见；一键/文字/照片提交；每日/每周/一次性重复拒绝；未开始/已结束拒绝；老师单条贴纸、语音、班级星星回应；批量回应一次覆盖 2 个家庭（`responded_count=2`，同一 `response_id`）；删除级联（任务/提交/回应软删、状态硬删）；`point_transactions` 无新增。

UI 验证于 2026-10-21 在微信开发者工具完成（详见 TEST_CATALOG「UI 交互验证」）。工具备注：本机 `wechat_automator` 的 `page_data`/`element_info`/`screenshot` 与 `wechat_navigate` 持续 90–120s 超时，`system_info`/`evaluate` 正常；改用 `_verify/wx_cdp.mjs`（CDP `Runtime.evaluate` + `Input.dispatchMouseEvent`）读渲染文案并派发真实点击。触摸事件不触发 tap，鼠标事件有效。

仍未覆盖：照片选图（`wx.chooseMedia` 拉系统对话框）、真机渲染、老师端创建表单的逐项填写提交与语音回应。

数据库旧列已清理：`docs/migrations/2026-10-21-drop-task-points-columns.sql` 已执行，`parent_tasks.points`、`task_check_ins.points`、`task_check_ins.image_url` 三列已不存在；执行前后行数不变（parent_tasks 17 / task_check_ins 10）。

## 7. 下一步

> 2026-10-21 已全部执行：7.1 列可空（YES）、7.2 后端起在 `:8080`、7.3 账号可用、7.4 核心链路跑通、7.5 TC 已登记。以下保留为复跑步骤。

### 7.1 先确认数据库列可空

```powershell
cd D:\yuanhaisu\personal_projects\kindergarten\server
go run ./cmd/dbcheck --query "SHOW COLUMNS FROM task_checkin_states LIKE 'last_check_in_date'"
```

期望 `Null = YES`。若为 `NO`，执行迁移。

本机没有 mysql 客户端时，用一次性 Go 程序（执行后删除，**不要提交**）：

```go
// server/cmd/tmpnullable/main.go
package main

import (
	"log"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"kindergarten-server/internal/config"
)

func main() {
	cfg, err := config.Load("configs/app.ini")
	if err != nil {
		log.Fatal(err)
	}
	db, err := gorm.Open(mysql.Open(cfg.MySQL.DSN), &gorm.Config{})
	if err != nil {
		log.Fatal(err)
	}
	if err := db.Exec("ALTER TABLE task_checkin_states MODIFY last_check_in_date DATE NULL").Error; err != nil {
		log.Fatal(err)
	}
	log.Println("done")
}
```

```powershell
go run ./cmd/tmpnullable
go run ./cmd/dbcheck --query "SHOW COLUMNS FROM task_checkin_states LIKE 'last_check_in_date'"
Remove-Item -Recurse -Force ./cmd/tmpnullable
```

### 7.2 启动后端

```powershell
cd D:\yuanhaisu\personal_projects\kindergarten\server
$env:CONFIG_PATH='configs/app.ini'
go run ./cmd/api/main.go
```

确认 AutoMigrate 建出新表：

```
task_responses
task_response_supplements
task_response_reactions
task_checkin_states
class_star_jars
task_read_marks
```

### 7.3 准备测试账号

- 家长测试账号：`13900010001` / `13900010002`；密码用 `go run ./cmd/fixpwd` 重置（脚本内固定值，**不要把密码写进文档或提交**）。
- 老师账号：走现有测试登录流程；不要新增绕过鉴权的测试后门，不要把 token 写进仓库。
- 上传：`POST /api/v1/common/upload`，`biz_type=task`。

### 7.4 核心链路清单

按顺序执行并记录证据：

1. 老师创建班级任务（`target_student_id=0`，`learning_link` 非空）。
2. 老师创建个体任务（`target_student_id=学生A`，`allow_public` 传 true 应被强制为 false）。
3. 家长 A 列表：两个任务都可见。
4. 家长 B（同班）列表：只见班级任务，不见个体任务。
5. 家长 A 一键提交班级任务；列表 `submitted_current_period=true`。
6. 同一天再次提交：应被拒绝。
7. 家长 A 提交个体任务（文字或照片）；非法 `media_file_id` 应被拒绝。
8. 老师 `GET /teacher/tasks/:id/checkins?status=pending`：看到待回应。
9. 老师单条贴纸回应；再测批量回应。
10. 家长详情：看到老师回应。
11. 频率：每日重复拒绝；每周 7 天内重复拒绝；一次性第二次拒绝。
12. 删除任务：`task_check_ins` / `task_responses` / `task_checkin_states` 级联清理；任务软删除。
13. 积分：`point_transactions` / `point_accounts` 无新增。
14. 跨班：家长不能通过任务 ID 看到其他班任务。

可用接口：

```
POST /api/v1/teacher/tasks
GET  /api/v1/teacher/tasks/:id
GET  /api/v1/teacher/tasks/:id/checkins?status=pending
POST /api/v1/teacher/tasks/:id/checkins/:checkin_id/response
POST /api/v1/teacher/tasks/:id/batch-response

GET  /api/v1/parent/tasks?student_id=...
GET  /api/v1/parent/tasks/:id?student_id=...
POST /api/v1/parent/tasks/:id/checkin?student_id=...
GET  /api/v1/parent/tasks/:id/checkin-history?student_id=...
```

创建任务请求示例：

```json
{
  "title": "今晚一起收玩具",
  "description": "让孩子自己把玩具放回收纳箱",
  "learning_link": "本周班级主题是‘我会整理’",
  "target_student_id": 0,
  "frequency": "once",
  "start_date": "2026-10-21",
  "end_date": "2026-10-28",
  "allow_public": false,
  "recognition_ritual": "sticker",
  "star_scope": "task",
  "star_goal": 0
}
```

### 7.5 执行 TC

逐条执行 `TEST_CATALOG.md` 的 TC-NEW-TSK-0101～0146：

- 属于 A 切片的，按实际结果标 ✅ / ❌；
- 属于 B/C 切片（通知、公开审核、补充回应、活动、班级罐等）的，标 ⏸ 或注明“未实现”，**不要为了全绿而假通过**。

## 8. 已知缺口

| 项 | 状态 |
|---|---|
| 发布通知 / `action_path` | 未实现；`CreateTask` 返回 `notified_count: 0` |
| 批量回应并发 | ✅ 已修：事务内 `FOR UPDATE` + `RowsAffected` 判定；输家返回 0 且回滚，无孤儿回应 |
| 周期汇总边界 | ✅ 已修：`taskPeriodRange` 按频率取区间并写入 `PeriodStart/PeriodEnd`；区间外提交不被关闭 |
| 公开申请与老师审核 | 未实现；只有 `allow_public` 与撤回 |
| 补充回应 / 家长反馈 / 未读标记 | 只有模型，无接口 |
| 班级星星罐管理 | 只有模型与校验，无入口；A 只开放按任务 |
| 园所活动 / 多班活动 / 活动统计 | 未实现 |
| 管理端任务页面 | 仍旧列表/删除，未接新字段 |
| 家长列表 Tab | 仍旧“进行中/已打卡/全部” |
| 老师回应文字 | 后端支持 `content`，前端只有贴纸/语音 |
| `has_unread_supplement` | 硬编码 false |
| 任务状态自动结束 | 未做；家长查询按日期过滤，但 `status` 仍为 1 |
| `TaskStats` 周期口径 | 目前是全量提交，不按周期 |
| `TaskService` 管理端 | 已去积分、已接级联删除；新字段展示仍需核对 |
| admin 前端类型 | 仍可能有 `task_checkin_points` 旧类型 |

## 9. 已知坑

- **并发提交**：写文档期间 HEAD 连续前进（`c5f7704 → 77c91e5 → 67fbb51`）。新会话开始前先 `git status` + `git log -3`，确认没有另一个会话正在改同一批文件。
- **旧会话不要继续**：37.8 MB / 11,819 行 / 3 次 compact / 最近一次压缩约 38 分钟；rewind 后当前分支是回退叶子，细节可能不在上下文。
- **nullable**：`task_checkin_states.last_check_in_date` 首次提交为 NULL；GORM AutoMigrate 只增不改，必须手工迁移。
- **时区**：统一 `Asia/Shanghai`；日期比较用 `shanghaiLoc()`。
- **范围**：`taskScopeQuery` 以 `pc.CurrentStudent` 为准；多孩家庭所有任务请求都要带 `student_id`。
- **唯一键**：`task_check_ins` 是 `(task_id, student_id, date)`；不是只按 `date`。
- **媒体**：`media_file_id` 必须属于当前用户且 `biz_type=task`；不要接受任意外链。
- **路由**：批量回应用 `POST /teacher/tasks/:id/batch-response`，避免 Gin 的 `:id` 与 `checkins` 冲突。
- **编辑**：老师编辑页 `status` 用 `?? 1`，不要用 `|| 1`，否则已结束任务会被重新打开。
- **OpenSpec 目录已入库**：`openspec/changes/2026-10-20-redesign-parent-task-loop/` 下的文件都在版本控制里，按普通代码对待即可（此前「未跟踪、别误删」的提醒已过期）。

## 10. 回滚

- 任务代码已提交；按提交回滚：`git revert <commit>`。
- 或只回滚文件：`git checkout <commit> -- <path>`。
- nullable SQL 是向后兼容的，不需要回滚。
- OpenSpec change 目录已入库，随代码一起 `git revert` 即可。

## 11. 新会话开场建议

```text
读 docs/handoffs/2026-10-21-parent-task-loop.md 的 §12（B1/B2 已完成）与 §13，然后开工 B3：
OpenSpec change 2026-10-20-redesign-parent-task-loop 的任务 5.1–5.5（通知聚合 + 读标记 + 小程序码）。

注意：B3 会改通知实现，旧提示「不要改通知实现」只适用于 A/B1 阶段，已作废。
先读 design.md D9 与 backend/api/ 的相关契约，再动代码；任务写路径先锁 parent_tasks 行。
按 §12 的窗口纪律：一次只做一组，做完一组先做限制性对抗性审查再验证，通过后才进下一组。
不要动 A/B1/B2 已落地的语义；不要提交 docs/competitive/*、.impeccable/、skills-lock.json。
```

---

## 12. B 切片进展（2026-10-22）

B 按任务组拆成 4 个窗口推进，**窗口内只做一组**，每组走完「改 → 限制性对抗性审查 → 验证」再进下一组：

| 窗口 | 任务组 | 状态 |
| --- | --- | --- |
| **B1** | 3.5 补充回应 + 3.6 家长反馈 | ✅ 2026-10-22 完成（后端） |
| **B2** | 4.1–4.4 公开审核 + 公开墙 | ✅ 2026-10-22 完成（后端，无界面） |
| **B3** | 5.1–5.5 通知聚合 + 读标记 + 小程序码 | ⚠️ 2026-10-23 后端完成（5.1–5.3 勾选；5.4/5.5 的界面与真机扫码待 10.x/12.6） |
| B4 | 9.5/9.7 + 10.5–10.8 前端 + 12.3/12.4/12.6 | 未开始 |

### 12.1 B1 落地内容

- `POST /teacher/tasks/:id/checkins/:checkin_id/supplement` → `TeacherPortalService.SupplementResponse`
- `POST /parent/tasks/responses/:response_id/reaction` → `ParentPortalService.ReactToResponse`
- `ParentPortalService.GetTask` 的 `responses[]` 新增 `supplements[]`（只含当前孩子），`my_reaction` 按家长账号回填
- 契约：`backend/api/api-teacher/09-class-and-tasks.md`、`backend/api/api-parent/06-task.md`
- 测试：`server/internal/service/task_response_supplement_test.go`（17 条，真实 MySQL）；每条关键断言都做过变异验证（删掉被守护的条件后该测试确实失败）

### 12.2 B1 的语义决定（后续不要改错）

- **补充回应不改 `ResponseID`**：它是一条独立 `TaskResponse(single)` + 一行关系表，提交的「已回应」状态与主回应都不动。
- **补充的 `ritual` 不继承任务默认仪式**：空值一律贴纸。否则一个默认 `class_star` 的任务会让所有补充请求 422。
- **补充禁止 `class_star`**：一条提交只有一个主回应能计入星星罐（3.7 的口径依赖这一点）。
- **反馈权限按「自己孩子」而不是「自己提交」**：提交人本人算，与该孩子有绑定关系的另一位家长也算（规格原文「家长只能对与自己孩子提交相关的回应反馈」）。判定时主回应走 `checkin.response_id`、补充回应走 `task_response_supplements` 关系，两条路径都要有；`student_parents` 绑定作为 co-parent 的放行条件。反馈仍记在**家长账号**维度，一条 period 回应可能覆盖两个孩子的提交，不按当前孩子收窄。
- **所有任务写入方都先锁任务行（`SELECT ... FOR UPDATE` on `parent_tasks`）**：老师端 `DeleteTask` / `RespondTask` / `BatchRespondTask` / `SupplementResponse`、家长端 `CheckinTask` / `ReactToResponse`、管理端 `TaskService.DeleteTask`。这是唯一能同时防止「孤儿关系行」和「1213 死锁」的锁序：
  - 孤儿：级联用当前读（`FOR UPDATE` Pluck）取回应 id，能看见并发补充刚提交的回应；
  - 死锁：级联的顺序是「锁回应 → 删 checkin」，而回应路径是「锁 checkin → 插回应」，不先拿任务锁就是 A→B/B→A。
  - 任何**新增**的任务写路径都必须先锁任务行，否则死锁会以 500 形式出现（删除类已映射 503）。
- **语音只能挂音频**：三个回应入口共用同一判断（`ErrTeacherRitualMediaType`），否则会写入家长端播不出来的语音回应。
- **仪式载体必填**：贴纸必须有 `sticker_key`，语音必须有音频媒体。主回应/批量回应/补充回应三个入口共用 `validateRitualPayload`。
- **`BaseModel` 自带 `DeletedAt`**，所有表默认软删除。凡是手写 `.Table(...)` 的查询都要自己补 `deleted_at IS NULL`；upsert 命中软删行时要清 `deleted_at`（同 `growth_likes` 的历史回归）。

### 12.3 B1 的路由结论（推翻 §9 的一条旧结论）

§9 写的「避免 Gin 的 `:id` 与静态段冲突」在 gin 1.11 上**不成立**：`/parent/tasks/responses/:response_id/reaction` 与 `/parent/tasks/:id` 可以共存，静态段优先匹配。`go test ./internal/app` 会构造整张路由表，冲突会直接 panic。

### 12.4 B1 未做的部分

- 老师端界面看不到某条提交已经追加过哪些补充（`TeacherTaskCheckins` 未返回 supplements）——归 9.3。
- 家长端反馈按钮、补充回应渲染、「给孩子看」：归 10.3 / 10.5。
- 指标（`feedback_coverage` / `child_happy_rate` / `ack_rate`）：管理端归 7.2；**老师端反馈汇总此前无人承接**，已登记为 3.11（P1）。落地时补一条「补充回应的反馈不进入主回应指标」的断言。

### 12.5 B1 UI 回归证据（2026-10-22）

实测路径（微信开发者工具 + CDP 直连）：

| # | 端 | 操作 | 结果 |
| --- | --- | --- | --- |
| 1 | 家长 | 打开任务详情 | 正常渲染任务标题/学习关联/提交区/打卡历史 |
| 2 | 家长 | 一键提交（真实鼠标事件） | 页面转为「打卡历史 2026-10-22 已提交」 |
| 3 | 老师 | 微信授权登录（含协议勾选，真实点击） | 进入工作台，`teacher_token` 落盘 |
| 4 | 老师 | 打开任务详情 | 待回应列表出现提交；贴纸选项 star/heart/thumb/录音 齐全 |
| 5 | 老师 | 点击 star 贴纸回应 | 待回应清空；「已回应」标签页出现该提交 |
| 6 | 家长 | 重新打开该任务 | 渲染「老师回应 / 贴纸 / 老师回应了你」，console 0 error |

工具层坑（复现记录，下次别再踩）：

- MCP 的 `wechat_navigate` / `wechat_automator(page_data)` / `wechat_screenshot` 在本机均 **120s 超时**；全程改用 `_verify/wx_cdp.mjs`（CDP 直连 `127.0.0.1:9222`）做导航、读文本、真实点击。
- `wx_cdp.mjs click <sel>` 默认 `touch` 模式 **点不动 uni 的 @click**（没有任何请求发出）；必须传第二个参数 `mouse`。
- DOM 标签是 `wx-view` / `wx-text` / `wx-button`（不是 `uni-*`），选择器要按前者写。
- 老师端与家长端共用同一个 IDE 实例：用 `wechat_ide(open)` 切项目后，**另一端的 automator 会失效**，需要重新 `start`；切换后 storage 仍在，但页面可能停在 login，重新走一次授权登录即可。

未覆盖（前端尚未实现，不是回归遗漏）：老师端补充回应入口（9.3）、家长端反馈按钮与补充回应渲染（10.3/10.5）。

---

## 12.6 B2 落地内容（2026-10-22）

起点 `393a3e3`。按 §13.2 先写契约再写代码；**没有界面**，全部 API/服务层验证。

新增端点（契约已在 `backend/api/`）：

- `POST /parent/tasks/checkins/:checkin_id/public-request` / `DELETE …/public-request`（取消；二选一里选了 DELETE）
- `GET /parent/tasks/public-wall`
- `PUT /teacher/tasks/:id/checkins/:checkin_id/public`（`{approve, scope_confirmed}`）
- `PUT /admin/tasks/checkins/:checkin_id/public`（只接受 `{revoke:true}`，已登记 `task:manage`）

读路径补 `public_*` / `public_status` / `public_visible`（`checkin-history`、老师端 `checkins`）。`public_visible` 由服务端按公开墙同一套条件算出。

### 12.7 B2 的语义决定（后续不要改错）

- **取消/撤回不受任务状态限制**：任务关了公开开关或已结束，家长仍必须能撤回自己的内容；只有「申请」要求 `allow_public=true`、非个体任务、任务有效。最初的实现把两者都套上这组前置，导致撤回入口被任务状态锁死——已修，`TestPublicCancelWorksAfterPublicDisabled/AfterTaskEnded` 守护。
- **重复申请不降级**：已批准且未撤回时再点申请是幂等空操作，避免手滑把墙上记录打回待审核。
- **空操作不写时间戳/日志**：从未申请时取消不写 `revoked_at`；管理端撤回空操作不写 `operation_logs`。
- **个体任务三重拒绝**：申请 422、审核 422、公开墙查询侧再排除一次（即使 2.9 回归留下 `allow_public=true` 也不展示）。
- **归属按「自己孩子」**：同一孩子的另一位家长（有绑定）也能申请/取消，与 3.6 反馈权限口径一致；不是「必须本人提交」。
- **按 checkin_id 进来的写路径**先用**非锁定读**定位 task_id，再锁 `parent_tasks`、最后锁 `task_check_ins`——锁序仍是任务行优先，非锁定读不参与死锁顺序。
- **审核允许在任务最后一天/结束后进行**，但公开墙按 `end_date >= 今天` 实时过滤，响应里的 `public_visible` 会如实返回 false。
- **公开墙多一条安全补强：`st.class_id = t.class_id`**。D7 的九条只约束任务侧，不约束提交者侧；少了它，孩子转班后内容会留在原班级墙上，管理员把任务改派到别的班时原班孩子的照片会漏给新班家长。
- **取消不套 `taskScopeQuery`**：孩子转班后原班级任务不再匹配，套上去会让家长撤不回自己的内容。归属只看「提交属于当前孩子」；「任务是否可见」只在申请时校验。

### 12.9 B2 限制性对抗性审查结论（2026-10-22）

审查方独立复核（未参与实现）后给出的 CONFIRMED / POSSIBLE，逐条处置：

| # | 级别 | 问题 | 处置 |
| --- | --- | --- | --- |
| 1 | CONFIRMED | 转班后撤回入口 404，内容仍留在原班级墙 | ✅ 已修：取消改为按 `id + kindergarten_id` 锁任务；新增 `TestPublicContentFollowsStudentClassMove` |
| 2 | POSSIBLE | 管理端改派任务班级 → 原班孩子内容出现在新班墙 | ✅ 已修：公开墙加 `st.class_id = t.class_id`；新增 `TestPublicWallHidesWhenTaskReassignedToOtherClass` |
| 3 | POSSIBLE | 老师审核锁内缺班级/园所条件（TOCTOU） | ✅ 已修：锁定读补 `kindergarten_id + class_id`（串行测试不可观测，未做变异） |
| 4 | POSSIBLE | `UpdateTask` 对不公开任务无条件撤销，反复给从未申请的提交盖撤回时间 | ✅ 已修：只在 `allow_public` true→false 或班级任务改个体时撤销；新增 `TestEditingNonPublicTaskDoesNotStampRevokedAt` |
| 5 | POSSIBLE | 测试缺 `approved_at IS NOT NULL`、提交软删、排序守卫 | ✅ 已补 3 条用例 + 3 项变异 |

审查确认无问题的部分：状态机逐条对应 D7、幂等语义、锁序无反向、手写 SQL 的软删过滤、权限边界。

### 12.10 第二轮审查（针对修复本身，2026-10-22）

对「修完的那版」再做一轮对抗审查，发现 3 个问题（均已修）：

| # | 问题 | 修法 |
| --- | --- | --- |
| 6 | 关闭 `allow_public` 的撤销只有 helper 级测试，**真实编辑路径没被覆盖**（spec 0143 可能回归而全绿） | 新增 `TestTeacherUpdateTaskClosingPublicRevokes` + 变异「彻底不撤销公开」 |
| 7 | `public_visible` 在老师/管理端按 0 计算，不校验提交者班级，与契约「等价于公开墙」不符 | 家长申请/取消与管理端撤回用 `locateCheckin` 带出学生当前班级；老师审核直接载入提交者班级；老师列表用 `Preload("Student")` 取班级。新增 `TestTeacherCheckinsPublicVisibleFollowsStudentClass` |
| 8 | 提交者已转班仍可批准 → 造出永远不上墙的「已公开」状态 | 批准时校验「提交者班级 = 任务班级」，不满足 422；拒绝/撤回仍可用（`TestTeacherApproveRejectedAfterStudentLeavesClass`） |
| 9 | `revokePublicForTask` 给**从未申请**的提交也盖 `public_revoked_at`，`public_status` 与撤回时间失真 | WHERE 收窄为「仍有公开状态」的行（申请过/确认过/批准过）；D7 的「清空」在这些行上本就是空操作，可见性不变；`TestTeacherUpdateTaskClosingPublicRevokes` 断言同伴提交保持 `none` |
| 10 | `UpdateTask` 撤销判定用的是原始 `req.AllowPublic`，落库值却被「个体任务强制 false」夹过 | 改用 `effectiveAllowPublic = allowPublic && target==0` 判定（防御 7.1 放宽结构锁后的耦合） |

第二轮变异项从 22 条增加到 25 条，全部 detected。另按审查建议修正了两处测试口径：

- 0143：改为由**真实编辑路径**的用例守护（此前只有 helper 直调）。
- 0145：注明「有提交后 `assertTaskEditable` 拒绝改 target，转换路径不可达」，已验证的只是公开墙侧排除个体任务，不能说成端到端已验证。
- 「老师审核锁内再校验园所/班级」是 TOCTOU 兜底，串行测试观测不到，代码保留但不计入变异。

### 12.11 第三轮审查（契约 / 接线 / 可见性等价，2026-10-22）

这轮换了切入面：不再只看 service 语义，而是逐字核对 `backend/api/` 契约、Casbin/路由接线，并把 `public_visible` 的判定与公开墙 SQL 做逐条对照。发现 2 个 CONFIRMED（同一根因的两面）、1 个契约分歧、3 处契约漂移：

| # | 问题 | 后果 | 修法 |
| --- | --- | --- | --- |
| 11 | 提交者**学生记录被软删**后，`locateCheckin`/`Preload` 拿到的班级为 0，而 `publicCheckinVisible` 把 0 当成「跳过班级校验」 | 老师 `checkins` 列表报 `public_visible=true`，公开墙的 `st.class_id = t.class_id` 却把该行排除——违反契约「`public_visible` 等价于公开墙会不会返回它」 | `submitterClassID == 0` 一律判不可见；补 `TestTeacherCheckinsPublicVisibleHidesDeletedStudent` |
| 12 | 同一状态下老师**撤回**返回 422（`TEACHER_INVALID_PARAMS`），因为提交者载入失败被当成参数错误 | 违反契约「撤回永远可用」；老师清不掉残留公开状态（管理端仍可撤） | 只在查库真出错时中断；学生缺失时班级为 0，批准自然被拒、撤回照常；补 `TestTeacherRevokeWorksAfterStudentDeleted` |
| 13 | §12.10 行 7 把「老师列表取班级」写成用 `locateCheckin`，实际是 `Preload("Student")` | 文档夸大了覆盖范围，正是本轮暴露 #11 的原因 | 已更正为真实实现 |
| 14 | 家长契约把 `public_status=approved` 定义为「已批准**且可见**」，但任务结束/关闭公开/转班后仍是 approved 而 `public_visible=false` | 客户端若用 `status` 判断上墙，会在任务结束后继续显示「已公开」 | 契约改为「审核状态」，并写明「是否在墙只以 `public_visible` 为准」 |
| 15 | 转班后申请公开返回 **404**（`ErrParentNotFound`），而契约把「任务仍属于孩子当前班级」列为允许条件并写明「不满足 → 422」 | 客户端按 404 显示「任务不存在」，与契约语义不符；测试还把错误码固化了 | 班级不再匹配 → `ErrTaskPublicNotAllowed`（422）；个体任务指向别的孩子仍按 404（不可见，不泄漏存在性）；更新测试断言 |
| 16 | 契约漂移：老师审核响应漏写 `task_id`；老师 checkins 漏写 B1 的 `response` 对象；家长申请/取消漏写 503 | 契约不再是准确的 source of truth | 三处契约补齐（实现不变） |

**复核对确认无问题的部分**：

- 公开墙与 `publicCheckinVisible` 的 `end_date` 口径等价（DATE 列 + 上海时区，`taskDayOf` 与字符串比较同一天）。
- 读者维度（`t.class_id = 读者孩子班级`）只属于墙查询，不属于 `public_visible`；后者回答的是「在该任务所属班级的墙上会不会出现」。
- 学生 `status`（在读/毕业/退学）两侧都不过滤——行为一致；若产品要求「毕业生不上墙」，必须同时改两处，否则会新造一条漂移。
- 公开墙不排除读者自己孩子的提交，D7 没有此要求（「本班家长可见」含自己在内）。
- Casbin 用 `keyMatch2` + `/api/v1/{parent,teacher,admin}/*` 通配，5 条新路由都在既有 policy 覆盖内；管理端权限缺口已在 `admin_perm.go` 登记并通过 `router_perm_test`。
- 路由无参数名冲突：`/public-wall`、`/checkins/:checkin_id/...` 与既有的 `/:id` 共存，`router_perm_test` 会构建真实路由树。
- 「编辑时无条件调用 revoke」已不是可观测缺陷（收窄后的 WHERE 清空后再调匹配不到），`UpdateTask` 里的条件只为少写无用 UPDATE；不变量由 `TestEditingTaskDoesNotRefreshRevokedAt` 守护。

### 12.8 B2 验证状态

- `go build ./...`、`go vet ./...`、`go test ./... -timeout 900s` 全绿（2026-10-22；跑全量前先停掉 `:8080` 的 `kg-server.exe`，否则识别队列测试会被抢任务）。
- `server/internal/service/task_public_test.go`：47 个真实 MySQL 用例；`_verify/mutate_b2.py` 28 项变异全部 detected。
- HTTP 冒烟（真实网关 + 中间件 + handler，2026-10-22）：公开墙 200、缺 `student_id` 422、不存在提交 404、管理端 `revoke=false` 422、跨班任务 404。
- TEST_CATALOG 的 0119/0120/0121/0134/0143/0144/0145 已从 ⏸ 改为「后端已验证 ⚠️」，UI 仍待 9.3/10.x/11.4。
- 未做：审核/申请/公开墙界面（9.3、10.x、11.4）；「含他人内容」只能靠老师勾选，无自动识别；`GET /parent/tasks` 与详情尚未返回 `public_*`（归 8.1）；`CheckinHistory` 仍按任务可见范围收窄（转班后家长查不到旧任务历史，但撤回接口可用）。

---

## 13. B2 开工清单（公开审核 + 公开墙）

### 13.1 起点状态

- B1 已提交并推送：`393a3e3 feat(tasks): B1 补充回应与家长反馈`（分支 `feat/parent-growth-scope`）。
- `openspec/.../tasks.md` 已是 3/89 勾选（3.5、3.6，另新增 3.11 待做）；B2 = **4.1–4.4**（4.5–4.7 是 P2）。
- 全量测试在 B1 收口时全绿；`go test ./internal/service -run 'TestSupplement|TestParentGetTask|TestReactToResponse|TestRitualPayload|TestDeleteTask'` 约 55s。
- 本地后端跑在 `:8080`（新构建），家长端 IDE 登录 user 61 / 学生 27，老师端登录 user 11（**小一班 class 1**）——两者不同班，跨端 UI 验证要造数，见 §12.5。

### 13.2 第一步：先把端点写进契约

`backend/api/` 目前**只声明**「公开相关属于 B 切片」，没有任何具体端点。按仓库铁律「改接口先改这里」，B2 的第一件事是把形状定下来，建议：

| 用途 | 建议路径 | 备注 |
| --- | --- | --- |
| 家长申请公开 | `POST /parent/tasks/checkins/:checkin_id/public-request?student_id=` | 校验提交属于当前孩子 |
| 家长取消申请 | `POST /parent/tasks/checkins/:checkin_id/public-cancel?student_id=` | 也可用 DELETE，二选一并写进契约 |
| 老师审核 | `PUT /teacher/tasks/:id/checkins/:checkin_id/public` | body `{approve, scope_confirmed}`；批准时 `scope_confirmed` 必须 true |
| 公开墙 | `GET /parent/tasks/public-wall?student_id=` | 只返回本班、仅含提交者孩子的内容 |
| 管理端撤回 | `PUT /admin/tasks/checkins/:checkin_id/public` body `{revoke:true}` | 走管理端权限登记表 |

> 管理端新增路由必须登记到权限注册表，否则 `NewRouter` 直接报错（`router_perm_test.go` 会红）。

### 13.3 状态机（design.md D7，逐字实现）

- 申请：`is_public_requested=true`、`public_scope_confirmed=false`、`public_approved_at=nil`、`public_revoked_at=nil`
- 批准：`scope_confirmed=true`、`public_approved_at=now`、`public_approved_by=teacher`
- 取消 / 撤回 / 关闭 allow_public：`is_public_requested=false`、`scope_confirmed=false`、`approved_*=清空`、`public_revoked_at=now`
- 再次申请：必须重新批准、重新勾确认，**不得沿用**旧批准或旧确认
- 可见条件（公开墙逐字照抄）：`allow_public && task.deleted_at IS NULL && task.status=1 && end_date >= 今天 && task.class_id = 当前孩子班级 && is_public_requested && public_scope_confirmed && public_approved_at IS NOT NULL && public_revoked_at IS NULL`
- 个体任务强制不可公开；一键完成（无文字无媒体）不可申请（P2 4.5）

### 13.4 必须遵守的既有约束

- **锁序**：任何任务写路径的第一条语句都是 `SELECT ... FOR UPDATE` 锁 `parent_tasks` 行，否则与 `deleteTaskCascade`（锁回应 → 删 checkin）成反序死锁。
- `BaseModel` 自带 `DeletedAt`，手写 `.Table()` 查询要自己补 `deleted_at IS NULL`；upsert 命中软删行要清 `deleted_at`。
- 已有 helper：`revokePublicForTask`（`task_rules.go`，清申请/批准/确认/审核人并写 revoked_at）——关闭 `allow_public` 时直接复用，不要重写。
- 老师审核的权限校验照 `SupplementResponse` 的写法：任务 `id + kindergarten_id + class_id`，提交 `id + task_id`。
- 错误码：参数/状态 422、跨家庭 403、并发关闭 409、锁冲突 503。

### 13.5 验证怎么做

- 后端：真实 MySQL 集成测试（照 `task_response_supplement_test.go` 的结构：固定 ID 夹具 + `Unscoped` 清理 + `clean()` 先跑一次）。**每条关键断言都要做变异验证**。
- 场景清单见 spec.md「活动公开需老师审核」的 7 个 Scenario（默认私密、申请后审核可见、跨班不可见、撤回与个体任务不可公开、关闭开关、更新为个体强制关闭、取消后重审）。
- UI：家长申请/老师审核的界面属 9.3、10.x，B2 仍然**没有界面可点**；只能走 HTTP + 必要时造数。造数用一次性 `cmd/tmp*` 程序，**用完立即删除**。

---

## 14. B3 落地内容（2026-10-23）

起点 `b3f2028`。按 §13.2 先改契约再写代码；只有后端，**没有界面**。

### 14.1 新增 / 变更

- 通知（5.1–5.3）：`server/internal/service/task_notify.go`
  - `POST /teacher/tasks` 发布后给目标家庭写 `type=task` 消息，`reference_id=task_id`；
  - `POST /teacher/tasks/:id/checkins/:checkin_id/response` 与 `POST /teacher/tasks/:id/batch-response` 按家长聚合写 `reference_id=response_id` 的消息；
  - 响应新增 `notified_count` / `notify_failed_count`（创建任务原本已有字段，现在返回真实值）。
  - 公共写入路径：`pickup_notify.go` 的 `sendMessageToRecipients`（支持每个收件人各自的 `action_path`）；`messages.action_path` 现在会通过 `/common/messages` 下发（`MessageItem.ActionPath`）。
- 小程序码（5.4）：`wechat.code_env_version`（默认 `release`；`server/configs/app.ini` 被 .gitignore 忽略，开发/测试要扫体验版/开发版时需在各环境本地配置）；`WeChatClient.GetParentWXACodeUnlimit(ctx, scene, page, envVersion)` 走家长端凭据；`GET /teacher/tasks/:id/share-card` → `{task_id, page, scene, env_version, image_base64}`（PNG base64，不落 OSS）。
- 补充回应未读（5.5）：`POST /parent/tasks/:id/read?student_id=`（upsert，返回 `{last_read_at}`）；任务列表/详情的 `has_unread_supplement` 现在按 `(任务, 家长, 孩子)` 实时计算。
- 契约：`backend/api/api-teacher/09-class-and-tasks.md`（回应计数 + share-card）、`backend/api/api-parent/06-task.md`（read 端点 + 未读口径）。
- 测试：`server/internal/service/task_notify_test.go`（18 条真实 MySQL）；`server/internal/auth/wechat_test.go` 新增 2 条（家长端凭据 / envVersion / 非法值）；`config_test.go` 新增默认版本断言；`_verify/mutate_b3.py` 15 项变异全部 detected（每条都人工确认是断言失败，不是编译失败）。

### 14.2 B3 的语义决定（后续不要改错）

- **消息判重带 title**：`type=task` 同时承载发布（`reference_id=task_id`）与回应（`reference_id=response_id`），两张表自增 id 会撞数；只按 `(user,type,reference_id)` 判重会吞掉正常通知。`MessageRepo.Exists` 现在额外匹配完整标题，两类前缀不同（`新亲子任务：` / `老师回应了：`）。
- **通知对象是「家庭」不是「提交人」**：主回应通知发给被覆盖孩子的全部家长（co-parent 也收）；计数按家长去重。
- **深链的孩子选择**：发布时班级任务取该家长在本班 `student_id` 最小的孩子（确定、稳定）；回应时取**第一个被覆盖的提交**所属的孩子（按 checkin id 升序），与 D9 逐字一致。
- **批量回应只通知写成功的提交**（`RowsAffected=1` 的那些）；失败提交继续保持待回应、不通知。
- **action_path 只允许 `/pages/` 开头且不含 `://`**；写路径会拒绝并计入失败，不会落库。
- **通知失败不回滚**：所有通知在事务提交之后调用，失败只记日志 + `notify_failed_count`。
- **读标记时间截断到毫秒**：列是 `datetime(3)`，MySQL 对不足毫秒的部分四舍五入；不截断时 `.0007s` 的读会存成下一秒，误清同毫秒的补充未读。
- **读标记 upsert 要复活软删行**：唯一索引包含软删行，`ON DUPLICATE KEY UPDATE` 必须同时清 `deleted_at`，否则红点清了又回来。
- **未读用 `>=`**：补充回应 `created_at >= last_read_at` 即未读（同毫秒不漏判）。
- **读接口的家长绑定由 handler 的 `parentTaskCtx` 保证**：它走 `ResolveContext`（只加载当前家长绑定的孩子）并校验 `CurrentStudent.ID == student_id`；service 与 `GetTask` 一样信任已解析的 `ParentPortalContext`。
- **share-card 必须走家长端凭据**；`envVersion` 空值取配置，非法值在发 HTTP 前报错（静默回退会生成另一个版本的码）。

### 14.3 限制性对抗性审查（2026-10-23）

| # | 级别 | 问题 | 处置 |
| --- | --- | --- | --- |
| 1 | CONFIRMED | 家长端 `pages/tasks/detail.vue` 的 `onLoad` 不解析 `scene`，扫码进来只有 `scene` 没有 `id` | 未修：属 10.1（前端），记入 14.5 |
| 2 | CONFIRMED | 家长端全仓不消费 `has_unread_supplement`，也不调用 `POST /parent/tasks/:id/read` | 未修：属 10.2/10.3（前端），记入 14.5 |
| 3 | CONFIRMED | `pages/messages/index.vue` 不消费 `action_path`，消息卡片没有点击跳转 | 未修：属 10.6（B4），记入 14.5 |
| 4 | POSSIBLE | `last_read_at` 亚毫秒精度被 MySQL 四舍五入，可能误判同毫秒补充为已读 | ✅ 已修：写库前 `Truncate(time.Millisecond)` |
| 5 | POSSIBLE | 目标家长查询失败时返回 `0/0`，与「班里没有家长」不可区分 | 保留：错误已 `log.Error`（含任务/回应 id），且不得把通知故障升级成业务失败；API 计数不伪造 |
| 6 | NOTE | 用标题当判重命名空间较脆弱，长期应改成 `reference_type + reference_id` | 记录；当前两类前缀稳定，改动需同步 domain/表结构 |
| 7 | NOTE | 5.3 的「部分失败」原先只覆盖全失败（关掉消息库） | ✅ 已补 `TestSendMessagePartialFailureCountsPerRecipient`：用超长 `action_path` 让一个收件人失败、另一个成功，断言 sent/failed 分别计数且失败方不留半条消息 |
| 8 | NOTE | 管理端 `POST /admin/tasks` 发布不发通知 | 属 C 切片（`task_service.go` 无消息依赖）；口径待 7.x 确认 |
| 9 | NOTE | 班级任务通知不按学生 `status` 过滤 | 与家长端任务可见性一致（也只按班级），若产品要求「毕业生不通知」，两处必须同时改 |

#### 第二轮（干净 subagent，2026-10-23）

| # | 级别 | 问题 | 处置 |
| --- | --- | --- | --- |
| 10 | CONFIRMED | 变异脚本有两条「假检出」：把 `isInternalActionPath` 改成 `return true` 会让 `strings` 变成未使用，把详情红点改成 `false` 会让 `unread` 未使用——两者都是编译失败，却被当成测试检出 | ✅ 已修：变体改成仍使用 `strings` / `unread`（`false && unread[...]`）；新增「列表红点」变体；重跑 15/15，并人工确认两条变体输出是 `Error Trace`（断言失败）而非 build error |
| 11 | POSSIBLE | `share-card` handler 把所有错误折叠成固定 500 且不打日志；契约写明「配置非法时错误信息要说明」，但具体原因被丢弃 | ✅ 已修：`auth.ErrInvalidCodeEnvVersion` 哨兵 + handler 分别返回「版本配置非法」；所有非 404 错误先 `log.Error` 再回 500 |
| 12 | NOTE | `POST /parent/tasks/:id/read` 的 default 分支静默回 500 | ✅ 已修：handler 记录 `parent mark task read failed`（含 task id 与 err）后再回 500 |
| 13 | NOTE | 周期汇总回应之后再追加补充的未读路径此前没有用例 | ✅ 已补 `TestUnreadSupplementUnderPeriodResponse`（A1 未读、A2 不受影响） |
| 14 | NOTE | `GenerateTaskShareCard` 没有服务层用例（只有 auth 包的凭据/env 用例） | ✅ 已补 `TestTaskShareCardRejectsForeignOrDeletedTask`（跨班 404、软删 404；正向生成依赖真实微信接口，仍不在单测内） |
| 15 | NOTE | `Truncate(time.Millisecond)` 的精度修复没有测试；目标查询失败仍返回 `0/0`；任务创建允许 `end_date` 已过但仍立即发通知，家长点进去 404 直到窗口开始 | 记录：精度修复靠 `datetime(3)` 事实 + 代码审查；`0/0` 已由日志兜底；过期日期属创建校验（2.x/C 切片），本次不改 |
| 16 | NOTE | 判重加入 title 也改变了晨检通知的幂等口径（同一 check 若结论变化会重发） | 现有 `SubmitCheck` 是一次性的（重复提交直接拒绝），无实际回归；记录在案 |
| 17 | NOTE | 每个夹具 `setupB1/2/3` 都新开一个 `gorm.Open` 连接池且不关闭，全量 suite 会累积连接 | 既有模式，非本次引入；记录 |

### 14.4 验证状态

- `go build ./...`、`go vet ./...` 通过（2026-10-23）。
- `go test ./internal/service -run 'TestTaskPublish|TestBatchResponse|TestSingleResponse|TestTaskAndResponseSurvive|TestSupplementUnread|TestReadMark|TestTaskActionPath|TestSendMessage|TestMessageDedupe'` 全绿。
- `go test ./internal/auth ./internal/config ./internal/app` 全绿（含新路由表的冲突检查）。
- `_verify/mutate_b3.py`：15 项变异全部 detected；其中「放宽 `/pages/` 校验」「列表红点恒 false」两条已人工回读输出，确认是断言失败而不是包编译失败。
- 全量 `go test ./... -timeout 900s`：**全绿**（2026-10-23 第二轮修复后重跑，exit 0；`internal/service` 492s，`internal/ai/face` 1.9s、`internal/app` 3.9s、`internal/auth` 3.8s、`internal/config` 0.5s、`internal/middleware` 3.0s、`internal/rbac` 1.3s、`internal/repository` 10.0s 均 ok；跑前已确认本机 :8080 无 LISTEN）。
- TEST_CATALOG 0124/0125/0126/0140 已从 ⏸ 改为「后端已验证 ⚠️」。

### 14.5 B3 未做的部分（不要当成已验证）

- **家长端**：`scene` 解析（10.1）、任务列表/详情红点与打开清除（10.2/10.3）、消息点击按 `action_path` 跳转（10.6，B4）。
- **老师端**：保存分享码到相册 / 转发提示（9.5，B4）、真机扫码进入对应版本（12.6）。
- **管理端**：管理端创建任务的发布通知（C 切片）。
- 目标家长查询失败的 API 级可观测（见 14.3 #5）；任务级「一个家长成功、一个失败」未做（共享写路径已由 #7 覆盖）。

---

## 15. 下一窗口开工（B4 前端收口，2026-10-23 准备）

### 15.1 起点状态

- 分支 `feat/parent-growth-scope`，HEAD `26a853c`（已 push；远端此前停在 `393a3e3`，本次连带 B1/B2 一起推了）。
- 后端：5.1–5.3、5.7 已勾选；**5.4 / 5.5 的后端已完成但未勾选**——它们的验收包含家长端 `scene` 解析、红点/读标记、真机扫码，这些在本窗口闭合后再勾。
- 全量 `go test ./... -timeout 900s` 全绿（service 492s）；B3 用例 18 条；`_verify/mutate_b3.py` 15/15 **真实**检出。
- 前端是**旧 A 切片页面**：B1（补充回应/反馈）、B2（公开审核/公开墙）、B3（未读/分享码）的界面都还没接。
- 本地后端当前未跑；改前端后必须先 `npm run dev:mp-weixin` 再验证（IDE 读 `dist/dev/mp-weixin`，不是源码目录）。

### 15.2 前端现状（2026-10-23 逐文件核实，别凭任务描述假设）

| 文件 | 已有 | 缺 |
| --- | --- | --- |
| `teacher/src/pages/tasks/detail.vue` | 单条/批量回应、贴纸/语音/星星、待回应/已回应筛选、编辑、下拉刷新 | 补充回应入口、公开审核、分享码、星星罐目标设置 |
| `teacher/src/pages/tasks/index.vue` | 列表、进入详情 | 创建/编辑表单的新字段（学习关联、范围、频率、仪式、星星、公开）→ 9.2 |
| `teacher/src/api/teacher-tasks.js` | 列表/详情/建改删/统计/checkins/回应/批量 | `supplement`、`public` 审核、`share-card` → 9.1 |
| `parent/src/pages/tasks/index.vue` | 列表、按孩子取数（`withStudentScope`）、进度 | 未渲染 `has_unread_supplement` 红点；状态/进度口径待核对 → 10.2 |
| `parent/src/pages/tasks/detail.vue` | 详情、提交、打卡历史、主回应简版 | `scene` 解析、补充回应渲染、`my_reaction`、公开申请/取消、打开调 `read`、「给孩子看」→ 10.1/10.3 |
| `parent/src/pages/messages/index.vue` | 消息列表 | 卡片点击、`action_path` 跳转、任务不存在/结束提示 → 10.6 |
| `parent/src/api/parent-task.js` | 列表/详情/提交/历史（已带 `student_id`） | `markTaskRead`、`reactResponse`、公开申请/取消/公开墙、`share-card` 相关无 |

> ⚠️ 两端 `components/` 会被 `scripts/sync-ui.js` 覆盖；**改组件只改 `shared-ui/`**，再同步。

### 15.3 建议窗口拆分（一次只做一组，组间先审查再验证）

| 窗口 | 范围 | 目的 |
| --- | --- | --- |
| **B4-a（建议下一个开）** | 10.1、10.2、10.3、10.6 + 9.1 的 `share-card` 封装 + 9.5 分享按钮 | 闭合 5.4/5.5 验收，并把 B1/B2 的家长端界面补上 |
| B4-b | 9.1 余下、9.2、9.3（补充回应 + 公开审核）、9.7 | 老师端完整闭环 |
| B4-c | 10.4、10.5、10.7、11.x | 首页卡片、反馈按钮、管理端 C 切片 |
| 收尾 | 8.x 勾选、12.3/12.4/12.6/12.7/12.8 | 真机/回归/归档前验证 |

### 15.4 B4-a 实现要点

1. **10.1 API 与 scene**
   - 新增 `markTaskRead(taskId, studentId)` → `POST /parent/tasks/:id/read?student_id=`；
     `reactResponse(responseId, reaction, studentId)` → `POST /parent/tasks/responses/:response_id/reaction?student_id=`；
     `requestPublic(checkinId, studentId)` / `cancelPublic(checkinId, studentId)` / `fetchPublicWall`。
   - `onLoad(opts)` 解析小程序码：`scene` 形如 `t=123`（可能 URL-encoded）→ `decodeURIComponent` → 取 `t`；
     `id = Number(opts?.id || sceneTaskId || 0)`，两者都没有才报错。
   - 首页入口若也带 `scene`，同样处理（10.1 原文包含首页入口）。
2. **10.2 列表红点**
   - 用 `item.has_unread_supplement` 显示任务级红点（与消息未读是两套，不要合并）。
   - `onShow` 重新拉列表（详情返回后红点要消失）；切换孩子后所有请求带新 `student_id`。
3. **10.3 详情**
   - 渲染 `responses[].supplements`：贴纸 key → 图形/文案、语音 → 播放、文字 → 文本；`my_reaction` 回显。
   - 反馈三选一（`child_happy` / `received` / `thanks`），同一按钮再点即更新；反馈按家长账号，不按孩子收窄。
   - 公开开关只在 `allow_public && target_student_id === 0` 时显示；申请后用 `public_status`（审核状态）与
     `public_visible`（此刻是否在墙）**分别**表达，不要用 status 冒充「已上墙」。
   - **数据加载成功后**调 `POST read`，失败只 toast/日志、不阻塞渲染；多孩时用当前 `student_id`。
   - 「给孩子看」全屏态：放大贴纸/播放语音/老师的话（10.3 原文）。
4. **10.6 消息深链**
   - 只接受 `/pages/` 开头且不含 `://` 的 `action_path`；跳转时传 `id` 与 `student_id`。
   - 任务页不是 tabBar 页 → `uni.navigateTo`；家长端 tabBar 只有 index/growth/pickup/profile。
   - 任务删除/结束（详情 404）显示「任务已结束/不存在」，不要白屏。
5. **9.5 分享码**
   - `fetchTaskShareCard(taskId)` → `{ task_id, page, scene, env_version, image_base64 }`。
   - base64 先用 `uni.getFileSystemManager().writeFile({ encoding: "base64" })` 落临时文件，再 `uni.saveImageToPhotosAlbum`；
     失败提示「请检查家长端微信配置」，不要把微信错误直接抛给老师。
   - 码必须由后端家长端凭据生成（已保证）；真机扫码验证属 12.6，需要本地 `wechat.code_env_version` 配 `trial`/`develop`。

### 15.5 验证与收工

- `npm run build:mp-weixin`（teacher + parent）通过；`node scripts/check-tokens.mjs teacher|parent` 对改动文件零新增违规。
- 交互必须真实点击（`wechat_automator` / CDP），不能用 `wechat_navigate` 代替；导航跳转的 `switchTab`/`navigateTo` 差异只在真实点击下暴露。
- 至少走通：家长任务红点 → 打开 → read 清除 → 看到补充回应 → 反馈 → 申请公开；
  老师生成分享码 → 保存相册 → 真机扫码进入**家长端**任务详情（12.6）。
- 不提交 `docs/competitive/*`、`skills-lock.json`、`.impeccable/`。
- 完成后：更新本文件 §16、勾选 5.4/5.5（只有在 UI 与扫码都验证后）及对应 9.x/10.x，再考虑 archive。

### 15.6 已知坑（沿用，别重踩）

- `wx_cdp.mjs click <sel> mouse` 必须带 `mouse`，否则点不动 uni 的 `@click`；DOM 标签是 `wx-view`/`wx-text`/`wx-button`。
- 单 IDE 实例下双端不能并行；要跨端联调走 `scripts/dual-ide.ps1`（两个微信号、9222/9223、9420/9421）。
- 老师端**没有 tabBar**；家长端 tabBar 是 index/growth/pickup/profile，跳这些页面必须 `switchTab`。
- 改 AppID / `project.config.json` 后必须重新构建；占位 AppID 会让自动化端口静默不监听。
- 前端组件改 `shared-ui/`，不要改两端 `components/`（会被同步脚本覆盖）。

### 15.7 新窗口开工提示词（直接复制）

```text
读 docs/handoffs/2026-10-21-parent-task-loop.md 的 §15，按 §15.3 的 B4-a 范围开工：
家长端 10.1 / 10.2 / 10.3 / 10.6（scene 解析、补充回应渲染与反馈、任务红点与读标记、消息 action_path 跳转），
外加老师端 9.1 的 share-card 封装与 9.5 分享按钮。

开工前先读：design.md 的 D9/D10、backend/api/api-parent/06-task.md、backend/api/api-teacher/09-class-and-tasks.md、
以及 openspec/changes/2026-10-20-redesign-parent-task-loop/tasks.md 的 9.x/10.x。

约束：
- 不要动 A/B1/B2/B3 已落地的后端语义（尤其任务锁序、公开状态机、通知按家长聚合、读标记口径）。
- 前端组件只改 shared-ui/，不要改两端 components/；样式只能引用 shared-design-tokens 的 token。
- 家长端所有任务接口必须带 student_id；老师端没有 tabBar，家长端 tabBar 只有 index/growth/pickup/profile。
- 交互验证必须真实点击（wechat_automator / CDP），不能用 wechat_navigate 代替；改完先 npm run dev:mp-weixin。
- 别提交 docs/competitive/*、skills-lock.json、.impeccable/。
- 完成后：两端 build + check-tokens，更新 handoff §16，并在 UI 与真机扫码都验证后再勾选 5.4/5.5 与对应 9.x/10.x。
```

---

## 16. B4-a 落地内容（2026-10-23）

### 16.1 改了什么（逐个文件）

| 文件 | 变更 |
| --- | --- |
| `parent/src/api/parent-task.js` | 新增 `markTaskRead` / `reactResponse` / `requestPublicCheckin` / `cancelPublicCheckin` / `fetchPublicWall`；`student_id` 一律拼在 query（body 里的会被后端丢弃），读标记走 `silent`（失败不影响渲染） |
| `parent/src/pages/tasks/detail.vue` | `onLoad` 解析小程序码 `scene`（`t=123`，支持 URL 编码）→ 缺省回退 `id`；渲染 `responses[].supplements`（贴纸图形/语音播放/文字）；主回应与补充回应各三个反馈按钮并回显 `my_reaction`；打卡历史的公开申请/撤下（含 `public_visible` 与审核状态分开表达）；「给孩子看」全屏态；加载成功后调 `POST read`；消息目标孩子与当前孩子不一致时给提示 + 可切换 |
| `parent/src/pages/tasks/index.vue` | `has_unread_supplement` 任务级红点 + 文案；状态改「待完成/已提交」；进度按「本周期是否已提交」；`累计打卡 N 次`；进行中 tab 待办数；移除本地 `$purple-*`，全部改 token |
| `parent/src/pages/messages/index.vue` | 卡片按 `action_path` 跳转；只接受 `/pages/` 且不含 `://`；tabBar 四页走 `switchTab`，其余 `navigateTo`（失败提示「该内容已不存在」）；`onLoad` → `onShow` |
| `parent/src/pages/index/index.vue` | `quickActions` 改 `computed`，「亲子任务」红点取后端修正后的 `pending_tasks` |
| `teacher/src/api/teacher-tasks.js` | 新增 `fetchTaskShareCard` |
| `teacher/src/pages/tasks/detail.vue` | 分享卡：生成家长端小程序码 → base64 落临时文件 → 存相册；写入/相册各有超时封顶 |

### 16.2 语义决定（后续不要改错）

1. **列表不显示「已回应」**：契约 `06-task.md` 的任务列表只返回 `submitted_current_period` / `checkin_count`，没有回应字段。10.2 原文要三态，但补这个字段属于后端 + 契约改动，不在 B4-a 范围，且会动已经被 B1/B2 测过的读路径。现状＝列表两态（待完成/已提交），详情页按 `responses` 表达已回应。**要补就等有人愿意同时改 `06-task.md` + service + 单测**，不要在列表里猜。
2. **撤下公开用页面内两步点击，不用 `uni.showModal`**：原生弹窗不进视图层 DOM，CDP 既看不到也点不到，「撤下已公开内容」这条流程因此无法用真实点击验证（实测第一次点确实卡住不回调）。两步点击（`取消公开` → `确认撤下？`）保持显式确认，且可自动化。
3. **任务详情时间按本地时区渲染**：后端是 UTC RFC3339，直接截字符串会显示成「凌晨」。`shortTime` 用 `new Date(iso)` 转本地。
4. **`quickActions` 必须是 computed**：原先静态数组在 setup 时 `summary` 还是空的，`badge` 表达式只求值一次，红点永远不出现。该修复同时让「通知公告」红点开始生效（同一表达式，本就是原意）。
5. **分享码保存有超时封顶**（写入 8s / 相册 15s）：相册授权是系统弹窗，用户不理它时 `saveImageToPhotosAlbum` 永不回调，按钮会一直「生成中…」。没有封顶就是死锁。
6. **公开按钮的显示口径**：只有 `allow_public && target_student_id === 0` 才给入口；一键完成（无文字无媒体）不给「申请」，但「撤下」永远可用（对齐后端「撤回永远可用」）。

### 16.3 验证证据（真实点击，非导航替代）

后端 `make run`（8080）+ IDE 单实例，CDP 走 `scripts/cdp.py` / `scripts/focus-ide.ps1`。

| # | 动作 | 结果 |
| --- | --- | --- |
| 1 | 首页点「亲子任务」（`.action-item`） | 进入 `pages/tasks/index` |
| 2 | 列表读渲染 | 新任务显示「待完成 / 本周期待提交 / 一次性任务 · 累计打卡 0 次」，进行中 tab 带待办数 1 |
| 3 | 点「查看详情」 | 进入 `detail?id=4322&student_id=27`，`requirements` 渲染成 1./2. 两条，学习关联卡正常 |
| 4 | 点「今天完成了（一键提交）」 | 提交成功，提交卡消失，历史出现当日记录 |
| 5 | 库里补主回应 + 补充回应后返回列表 | 任务卡出现 `.tc-dot`（1 个）与「老师又给你留了一句悄悄话」 |
| 6 | 打开详情 | 未读提示条 + `⭐` 贴纸 + 老师的话 + 补充回应 `🌸`；`task_read_marks.last_read_at` 被写成打开时刻 |
| 7 | 返回列表 | 红点消失（`dots=0`），onShow 重取生效 |
| 8 | 点「孩子很开心」 | `task_response_reactions` 落到 `(699, 61, child_happy)`；按钮 `active`；重进详情回显 |
| 9 | 点「给孩子看」 | 全屏遮罩显示放大 `⭐` + 老师的话；点空白处关闭（`Input.dispatchMouseEvent` 打点） |
| 10 | 待完成任务点「去提交」→ 有内容提交 → 历史点「申请上班级墙」 | 状态变「已申请，等待老师审核」，按钮变「取消公开」；库里 `is_public_requested=1` |
| 11 | 点「取消公开」→「确认撤下？」 | 状态变「已撤下」；库 `is_public_requested=0`、`public_revoked_at` 有值 |
| 12 | 点「重新申请」 | 回到待审核；库 `revoked_at` 被清空（不沿用旧批准） |
| 13 | 消息页点任务消息卡片 | 跳到 `detail?id=4322&student_id=27`；消息里带 `查看 ›`，系统消息无入口 |
| 14 | 只带 `scene=t%3D4322` 打开详情 | 正确加载任务 4322（小程序码编码串可解析） |
| 15 | 打开不存在的任务 id | 显示「任务已结束或不存在」+ 重试，不白屏 |
| 16 | 消息目标孩子 ≠ 当前孩子（`student_id=32`） | 顶部提示「这条消息是另一个孩子的任务…」+「切换查看」 |
| 17 | 首页在存在待办任务时刷新 | 「亲子任务」入口出现 `.action-badge` |
| 18 | 老师端任务详情点「生成小程序码」 | 接口 200 + `image_base64`；PNG 落到 `USER_DATA_PATH/task-share-4324.png`（82521 字节）；相册授权在 IDE 里不回调，靠 15s 封顶把按钮放回可点 |

两端 `npm run build:mp-weixin` 通过；`node scripts/check-tokens.mjs parent|teacher` 对本次改动文件 **0 违规**。

### 16.4 未验证 / 缺口（不要当成已验证）

- **5.4 / 5.5 / 9.5 / 10.1–10.3 / 10.6 都没勾**：按 §15.5 的要求，勾选要等真机扫码（12.6）。IDE 里相册弹窗不回调，扫码那一段只能在真机上做。
- **未登录保留 action_path、登录后回跳**：IDE 里重新打开项目会清 token，但没构造「带着深链被踢去登录」的用例。
- **10.7（点消息顺带标记消息已读）** 仍属 B4-c，本次没动消息读状态。
- **列表「已回应」**：见 16.2-1，需要后端补只读字段才能做。
- **本地验证数据未清理**（都在开发库，园所 4）：`parent_tasks` 4322（class 4，回应+补充回应夹具）、4323（class 4，公开申请/撤下过）、4324（class 1，分享码）、4325（class 4，待完成）；补充回应 = `task_responses` 700（`response_type=single`）+ `task_response_supplements` 指向 checkin 1953；`messages` 1171 带 `action_path`；`task_read_marks` 131。要重跑验证可直接用这些 id，要清就从后往前删。
- 后端是本窗口 `go run` 起的（8080），收工时已停；下次验证要重新 `make run`。

### 16.5 下一步

B4-b：`9.1` 余下（supplement / public 审核封装）+ `9.2`（创建/编辑表单新字段）+ `9.3`（补充回应与公开审核 UI）+ `9.7`。老师端目前只能用 A 切片的老表单，创建任务时新字段还填不了。

### 16.6 对抗性审查（2026-10-23，针对 16.1 的 diff 本身）

按「怎样让它坏」的方向逐条过自己写的代码，找到 4 条真缺陷，3 条已修并复验，1 条只报告不动手。

| # | 缺陷 | 性质 | 处置 |
| --- | --- | --- | --- |
| 1 | **关掉 `allow_public` 后家长撤不下已公开内容**：`showPublicAction` 先判 `canPublic()`，老师一关公开开关，按钮整个消失。契约 D9 / 4.1 写的是「撤回永远可用，这些条件只约束申请」 | 违反契约 | 已修：`pending/approved` 直接给撤下入口，只有「申请」受 `canPublic()` 与「有内容」约束 |
| 2 | **深链落在 404 时，「换孩子」出口跟着消失**：提示条原来写在 `<template v-else>` 里，任务对当前孩子不可见（个体任务、跨班）时详情 404，家长只看到「任务已结束或不存在」，没有任何切换入口 | 10.6 的多孩场景 | 已修：提示条移到加载/错误分支之外；`currentSid` 在 `onLoad` 先用本地当前孩子兜底 |
| 3 | **`revoked` 文案把老师驳回说成家长撤下**：后端「老师拒绝/撤回」与「家长取消」写的是同一个 `PublicRevokedAt`，UI 却写死「已撤下」 | 语义误导 | 已修：改成中性的「未公开」 |
| 4 | **分享码接口返回 200 但没图时静默返回**：`stage === "generate"` 分支默认「http 层会弹提示」，只有当接口非 2xx 才成立；`image_base64` 为空时什么都不显示 | 静默失败（违反 CLAUDE.md） | 已修：空图单独 toast「小程序码为空，请检查家长端微信配置」 |
| 5 | ~~已结束任务无法撤下公开内容~~ **（查证后撤回）**：`GetTask` 对 `!taskIsActive` 返回 404，详情进不去 | 起初判为「契约与可达性冲突」，**证据不支持** | **不改**。spec 第 178 行「任务结束后公开内容也不再展示」→ 内容本来就不在墙上；第 209 行「任务删除/结束有提示」→ 详情 404 是有意行为。`DELETE public-request` 不限制任务状态是接口层兜底（防并发/直连），不是要求 UI 给已结束任务留入口。改 `GetTask` 的可见性会推翻 A 切片语义并牵动 2.5 的单测，收益为零 |

复验（真实点击 + 库核对）：
- #1：`task 4323 allow_public=0` + `checkin 1954 approved` → 详情显示「老师已通过，暂未展示在班级墙」且**有**「取消公开」按钮；两次点击后 `is_public_requested=0 / approved=null / revoked_at 有值`，文案变「未公开」，之后再申请入口按规则隐藏。改前该场景按钮数为 0。
- #2：`detail?id=4324&student_id=32`（class 1 任务，当前孩子 class 4）→ 页面同时显示「这条消息是另一个孩子的任务…切换查看」和「任务已结束或不存在」。
- #3 / #4：随 #1、分享码链路一并观察，编译产物已含新分支。
- 回归：`detail?id=4322` 回应/补充回应/反馈回显（6 个反馈按钮，1 个 active）未受模板结构改动影响；两端 `npm run build:mp-weixin` 再次通过。
