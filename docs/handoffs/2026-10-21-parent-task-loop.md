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

- `ListTasks`：返回 `pending_responses`。
- `GetTask`：任务详情，供编辑页使用。
- `CreateTask`：校验标题、学习关联、目标孩子、频率、日期、星星罐；个体任务强制 `allow_public=false`。
- `UpdateTask`：有提交后只允许改标题/说明/学习关联/公开开关；改个体或关公开时撤回公开。
- `DeleteTask`：事务级联删除并写操作日志。
- `TaskStats`：提交数 / 已回应 / 待回应。
- `TaskCheckins`：`pending` / `responded`。
- `RespondTask`：只回应 `response_id=0` 的提交。
- `BatchRespondTask`：一次回应覆盖全部待回应，返回 `failed_ids`。

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

> 该目录目前**未跟踪**，清理工作区前先确认不要丢。

## 6. 当前验证状态

| 项 | 状态 |
|---|---|
| `go build ./...` | 此前通过 |
| `go vet ./...` | 此前通过 |
| `go test ./internal/service/ -run 'TestValidTaskFrequency|TestSameTaskDay'` | 此前通过 |
| `parent npm run build:mp-weixin` | 此前通过 |
| `teacher npm run build:mp-weixin` | 此前通过 |
| nullable SQL | **未执行** |
| 核心链路 | **未跑** |
| TC 0101–0146 | **未执行，全部 🔲** |

## 7. 下一步

### 7.1 先确认数据库列可空

```powershell
cd D:\yuanhaisu\personal_projects\kindergarten\server
go run ./cmd/dbcheck --query "SHOW COLUMNS FROM task_checkin_states LIKE 'last_checkin_date'"
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
	if err := db.Exec("ALTER TABLE task_checkin_states MODIFY last_checkin_date DATE NULL").Error; err != nil {
		log.Fatal(err)
	}
	log.Println("done")
}
```

```powershell
go run ./cmd/tmpnullable
go run ./cmd/dbcheck --query "SHOW COLUMNS FROM task_checkin_states LIKE 'last_checkin_date'"
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
- **nullable**：`task_checkin_states.last_checkin_date` 首次提交为 NULL；GORM AutoMigrate 只增不改，必须手工迁移。
- **时区**：统一 `Asia/Shanghai`；日期比较用 `shanghaiLoc()`。
- **范围**：`taskScopeQuery` 以 `pc.CurrentStudent` 为准；多孩家庭所有任务请求都要带 `student_id`。
- **唯一键**：`task_check_ins` 是 `(task_id, student_id, date)`；不是只按 `date`。
- **媒体**：`media_file_id` 必须属于当前用户且 `biz_type=task`；不要接受任意外链。
- **路由**：批量回应用 `POST /teacher/tasks/:id/batch-response`，避免 Gin 的 `:id` 与 `checkins` 冲突。
- **编辑**：老师编辑页 `status` 用 `?? 1`，不要用 `|| 1`，否则已结束任务会被重新打开。
- **OpenSpec 目录未跟踪**：`openspec/changes/2026-10-20-redesign-parent-task-loop/` 不要在工作区清理时删掉。

## 10. 回滚

- 任务代码已提交；按提交回滚：`git revert <commit>`。
- 或只回滚文件：`git checkout <commit> -- <path>`。
- nullable SQL 是向后兼容的，不需要回滚。
- OpenSpec change 目录未跟踪，回滚代码前先备份。

## 11. 新会话开场建议

```text
读 docs/handoffs/2026-10-21-parent-task-loop.md，按里面的“下一步”继续：
先执行 nullable SQL，再跑核心链路，最后执行 TC-NEW-TSK-0101～0146。
不要动 B/C 切片，不要新增积分，不要改通知实现。
```
