# Tasks

> 分组按触点切，每组自带该组产出的测试/文档与验证门，不堆到最后。
> 标 `[P]` 的任务可与同组其它 `[P]` 任务并行。
> **MCP 验证一律串行**；UI 交互验证必须真实点击（`wechat_automator` tap/input 或真机），
> 不能用 `wechat_navigate` 替代。家长端 pickup 是 tabBar 页，子页跳转用 `navigateTo`。

## 1. 契约（backend/api/）— 契约先行

- [x] 1.1 [P] `backend/api/api-parent/05-pickup.md`：新增 `POST /pickup/designate`（请求 `student_id` 走 query、`authorization_id` 走 body，沿用既有 `student_id` 放 query 的约定）；`GET /pickup/qrcode` 响应新增 `person:{name,photo}`、`student` 补 `photo`；授权人列表响应字段改为与实现一致（`qr_code`/脱敏 phone/`review_status_label`，**以实现为准**，不暴露证件号）。验证：文档内字段清单与 design.md 契约字段级清单逐条对上
- [x] 1.2 [P] `backend/api/api-teacher/07-pickup.md`：`POST /scan` 请求收敛为仅 `qr_code`（移除 `student_id`）+ 可选 `remark`，响应新增 `person:{name,photo}` 与 `student.photo`；`POST /manual` 请求新增 `authorization_id`(可空)/`is_authorized_override`/`override_name`，`remark` 注明重复接走与提前接走必填；`GET /students` 响应每人新增 `is_overdue`/`pickup_deadline`/`last_remark`/`is_authorized_override`；`GET /overview` 新增 `overdue_not_picked`；`GET /records/:id` 改为 `{"record":{...}}` 包装（以实现为准）；补 `POST /notify-overdue`；补 `PUT /authorized-persons/:id/approve|reject`。验证：同上，且明确标注本次为契约变更
- [x] 1.3 [P] `backend/api/api-admin/08-pickup.md`：补 `GET /authorized-persons`（既有端点漏写）；新增 `GET /overdue`（滞留名单 + 各班超时未接计数）；记录明细的「核验人」列名对齐 `admin-05` 设计稿。验证：`grep` 确认 4 个既有端点 + 1 个新端点全部在文档中
- [x] 1.4 【本组验证】契约变更声明完整；逐字段核对 design.md「契约字段级清单」——凡清单里没有的字段不得出现在文档；`git diff backend/api/` 只含本次相关改动

## 2. 后端 domain + repository

- [x] 2.1 [P] `server/internal/domain/pickup.go`：`PickupQRCode` 加 `AuthorizationID`；`PickupRecord` 更新注释（`PickupPersonID` 真正落值、`OperatorID` 语义为核验人）；**新增** `PickupDailyDesignation` struct（`KindergartenID`+`StudentID`+`AuthorizationID`+`Date`，`UNIQUE(student_id,date)`）。验证：`go build ./...` 通过；新 struct 含 `kindergarten_id`（不复制 E1 旧缺陷）
- [x] 2.2 [P] `server/internal/domain/class.go`：`Class` 加 `PickupDeadline`（time 型，可空）。验证：`go build ./...` 通过
- [x] 2.3 `server/internal/repository/pickup_repo.go`：指定记录按 `(student_id, date)` upsert 与读取；按 `code` 联查 authorization；防重查询改为「取当日最后一条」；失效该生当日全部码；滞留名单与超时未接计数查询。验证：单测覆盖 upsert 覆盖语义、联查命中/未命中、连续同向判定
- [x] 2.4 【本组验证】`go vet ./...` 与 `go test ./repository/...` 通过；`go run ./cmd/dbcheck pickup_qr_codes`、`go run ./cmd/dbcheck classes`、`go run ./cmd/dbcheck pickup_daily_designations` 三条**逐条**核实列/表真实落地（AutoMigrate 只增不改，加列/建表需实测确认）

## 3. 后端 service + handler + router + Casbin

- [x] 3.1 `teacher_portal_service.go`：重写 `CreatePickupRecord` 校验链——从码解出学生与接走人 → 授权二次校验（`status=1 AND review_status=1` 且未过期）→ 连续同向校验 → 理由必填判定（重复接走 / 提前接走，判定集中一处见 design D5）→ 落 `PickupPersonID` 与例外标志 → 码置失效 → 若 `NotifyOnPickup` 则写 Message（失败仅记日志，不回滚登记）。验证：单测覆盖 spec 中扫码 3 个场景 + 例外通道 3 个场景 + 交替序列 3 个场景 + 重复理由 4 个场景 + 提前理由 4 个场景
- [x] 3.2 `teacher_portal_service.go`：`PickupStudents`/`PickupOverview` 状态派生改为「看最后一条方向」，输出 `is_overdue`/`pickup_deadline`/`last_remark`/`is_authorized_override` 与 `overdue_not_picked` 计数（按学生计）。验证：单测覆盖 spec「状态以最后一次转移为准」「超时未接被计数」「随后被接走不重复计数」
- [x] 3.3 `parent_portal_service.go`：指定今日接走人（校验授权有效、覆盖旧指定、失效该生当日旧码）；`GetQRCode` 从指定记录取 `authorization_id` 并再校验、建码绑人、响应补接走人姓名照片；`ListPickupRecords` 响应补 `is_abnormal`（修既有漂移：契约已列、实现未返回）；接走通知接收方为**该孩家长**（非接走人本人，见 spec）。验证：单测覆盖 spec「指定成功/拒绝未审核/拒绝已撤销/覆盖旧值」4 个场景 + 凭证 5 个场景；`is_abnormal` 有/无异常各一条
- [x] 3.4 `pickup_service.go`（管理端）：滞留名单、超时未接计数、一键通知复用同一 Message 写入路径。验证：单测覆盖 spec「管理员看到全园滞留名单」「通知留痕可查」
- [x] 3.5 `handler/http/teacher_portal.go`、`parent_portal.go`、`admin_*.go`：按 1.1–1.3 的契约调整入参与响应；**新增**家长指定端点、老师一键通知端点、管理滞留/计数端点。验证：`go build ./...`；请求只带 `qr_code` 的扫码能跑通（不再因 `StudentID==0` 报参数错）
- [x] 3.6 `server/internal/app/router.go` 注册新路由；`configs/rbac_policy.csv` 核对新路由是否落在角色通配前缀内，落不进则补策略。验证：`go build ./...`；用三类 token 试打新端点不出现 403（角色通配覆盖的应放行）
- [x] 3.7 【本组验证】`go vet ./...` 与 `go test ./...` 全部通过；`make wire`（若新增 repo/service/handler 依赖则必须，接口依赖补 `wire.Bind`）；对照 `specs/pickup-custody/spec.md` 逐条核对单测已覆盖 13 条 Requirement 的主路径

## 4. 家长端页面

- [x] 4.1 [P] `parent/src/pages/pickup/authorized.vue`：增「设为今日接走人」操作与当前指定展示；换人时提示「将使今日已生成的凭证失效」。验证：`npm run dev:mp-weixin` 构建通过；真机/automator 点击可完成指定与换人
- [x] 4.2 [P] `parent/src/pages/pickup/qrcode.vue`：渲染真实二维码图（替换现有文本占位框），显示孩子与接走人姓名/照片。验证：构建通过；扫码得到的码可被老师端识别
- [x] 4.3 [P] `parent/src/api/parent-pickup.js`：新增指定函数；`student_id` 继续放 query（既有约定，勿放 body）；响应字段按 1.1 对齐。验证：构建通过
- [x] 4.4 【本组验证】`npm run build:mp-weixin` 通过；UI 交互用 `wechat_automator` tap/input 或真机走通「指定 → 取码 → 显示人名照片」；**不得**用 `wechat_navigate` 代替点击

## 5. 老师端页面

- [x] 5.1 `teacher/src/pages/pickup/index.vue`：扫码只传凭证内容；手动登记加「选授权人 / 标例外」二选一与理由输入；滞留名单高亮；一键通知按钮。验证：`npm run dev:mp-weixin` 构建通过；automator 点击走通扫码登记与手动例外两条路径
- [x] 5.2 `teacher/src/pages/pickup/records.vue`：显示理由与例外标记；重复/提前接走的失败提示文案。验证：构建通过；触发一次缺理由的重复接走可见明确提示
- [x] 5.3 `teacher/src/api/teacher-pickup.js`：扫码入参收敛；新增一键通知函数；手动登记入参加例外与理由。验证：构建通过
- [x] 5.4 【本组验证】`npm run build:mp-weixin` 通过；对照 spec「扫码接走登记受授权约束」「手动登记接走区分授权与例外」逐条用真实点击验证；老师端无 tabBar，跳转一律 `navigateTo`（不涉及 `switchTab`）

## 6. 管理端页面

- [x] 6.1 [P] `admin/pages/pickup/index.vue`：滞留名单与超时未接计数展示；核验人列名对齐 `admin-05` 设计稿。验证：`npm run build` 通过
- [x] 6.2 [P] `admin/api/pickup.ts`：新增滞留/计数接口；补 `getAuthorizedPersons` 等既有函数的契约对齐。验证：`npm run build` 通过
- [x] 6.3 【本组验证】`npm run build` 通过；管理端能按班级查看滞留名单与超时未接计数

## 7. design token

- [x] 7.1 滞留高亮、例外标记优先复用既有语义色（`$color-semantic-warning` / `danger` 系）。验证：`node scripts/check-tokens.mjs teacher parent admin` 无违规；`<style>` 中无新增硬编码色值
- [x] 7.2 【本组验证】若确需新视觉语义：改 `shared-design-tokens/tokens.json`（px 值）→ `cd shared-design-tokens && npm run build` → `git diff build/` 核对产物；**不得**只改生成产物

## 8. 测试用例补录与逐条验收

- [x] 8.1 `docs/tests/TEST_CATALOG.md` 补录 `TC-NEW-PKU-001`~`122`（对应 spec 13 条 Requirement 的全部场景），并修订与新行为冲突的既有 TC：`TC-TCH-PKUT-014`（不再假设请求带 student_id）、`TC-TCH-PKUT-019`/`020`（同日同类型唯一改为连续同向限制）。验证：catalog 中 TC ID 与 spec 场景数一致、无重复
- [x] 8.2 对照 `specs/pickup-custody/spec.md` **13 条 Requirement 逐条**验收，每条记录通过/失败。**未验证不得宣布完成、不得 archive**。UI 场景用真实点击，双端同时验证用 `scripts/dual-ide.ps1`（家长端 auto_port=9420/cdp=9222，老师端 auto_port=9421/cdp=9223）
- [x] 8.3 明确记录：**送入不记交接人**与 `design-data/admin-05` 的分歧已在 proposal 分歧清单声明，验收时按 spec（不记人）判定，不按设计稿示例判定
- [x] 8.4 明确记录：双人核验流程、非授权接送尝试事件、迟到接走类型、请假接入、租户列补齐**不在本次验收范围**（proposal out-of-scope）

## 9. 收工清理

- [x] 9.1 按 PID 杀掉残留 uni watcher（`dev:mp-weixin` 是 watch 进程，`TaskStop` 只终止 shell 包装）：
  ```powershell
  Get-CimInstance Win32_Process -Filter "Name='node.exe'" |
    Where-Object { $_.CommandLine -match "dev:mp-weixin|kindergarten\\(teacher|parent)\\node_modules" } |
    ForEach-Object { Stop-Process -Id $_.ProcessId -Force }
  ```
  验证：`Get-CimInstance` 复查无匹配进程，工作区无 watch 进程弄脏的产物
- [x] 9.2 双 IDE 实例收工：`powershell -ExecutionPolicy Bypass -File scripts/dual-ide.ps1 -Kill`。验证：无残留 `wechatdevtools` / `cli login` 进程占用 3799 端口

## 回滚预案（不作为任务，仅供应急）

1. 新列/新表均为增量，回滚只需回退代码，不需数据回迁
2. 回退后旧逻辑对 `authorization_id` 为空的码会拒绝扫码，属预期行为
3. 契约文档若已提交，回滚需一并回退 `backend/api/`，避免文档与实现二次漂移
