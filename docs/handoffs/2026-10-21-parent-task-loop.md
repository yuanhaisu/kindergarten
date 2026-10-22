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
读 docs/handoffs/2026-10-21-parent-task-loop.md，按里面的“下一步”继续。
A 切片与 B1（3.5 补充回应 + 3.6 家长反馈）已落地；B 剩下的按 §12 的窗口表逐组推进，
一次只做一组，做完一组要先做限制性对抗性审查再验证，通过后才进下一组。
不要新增积分，不要改通知实现。
```

---

## 12. B 切片进展（2026-10-22）

B 按任务组拆成 4 个窗口推进，**窗口内只做一组**，每组走完「改 → 限制性对抗性审查 → 验证」再进下一组：

| 窗口 | 任务组 | 状态 |
| --- | --- | --- |
| **B1** | 3.5 补充回应 + 3.6 家长反馈 | ✅ 2026-10-22 完成（后端） |
| **B2** | 4.1–4.4 公开审核 + 公开墙 | ✅ 2026-10-22 完成（后端，无界面） |
| B3 | 5.1–5.5 通知聚合 + 读标记 + 小程序码 | 未开始 |
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
