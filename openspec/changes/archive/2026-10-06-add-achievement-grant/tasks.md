# Tasks

> 分组对应 design.md 的「跨端触点清单」。每组末尾是**验证**任务，未验证不得进入下一组、不得 archive。
> `[P]` = 与其它标记 `[P]` 的任务可并行。

## 1. 后端 — 数据结构与授予逻辑

- [x] 1.1 `server/internal/domain/points.go`：`StudentAchievement` 加 `LogID *uint64`（见证记录，可空；注释说明语义）
- [x] 1.2 `server/internal/service/achievement_service.go`：新增 `GrantToStudent`，按 design D2 的校验顺序执行——徽章启用 → 跨园所（徽章/学生）→ 见证记录可见性 → 重复授予
- [x] 1.3 服务层语义化错误：徽章未启用 / 跨园所 / 见证无效 / 重复授予 各自返回可区分错误，供 handler 映射 400/403/409（**不得吞错返回 500**）
- [x] 1.4 `server/internal/service/parent_portal_service.go`：`ListAchievements` 与记录查询带出 `log_id`、徽章名称/图标/类别
- [x] 1.5 **验证（后端）**
  - [x] `go build ./...` 通过
  - [x] `go run ./cmd/dbcheck student_achievements` 确认 `log_id` 列存在且可空（`idx_student_achievements_log_id`）
  - [x] TC-ACH-001~009 由**真实 HTTP + 真实 MySQL** 验证（GrantToStudent 是 DB-bound，仓库无 sqlite 测试设施；比 mock 单测更强）。纯函数部分另有 8 个单测
  - [x] 断言重复授予返回 **409 而非 500** —— 实测 HTTP 409 / code 40910
  - [x] 对照 specs `achievement-award` 全部 4 个 Requirement

## 2. 后端 — 路由与权限

- [x] 2.1 `server/internal/app/router.go`：新增老师端授予路由（`POST /teacher/achievements/:id/grant`，挂在 teacher 分组）
- [x] 2.2 `server/configs/rbac_policy.csv`：**已核实无需改**——`p, teacher, /api/v1/teacher/*, *` 通配已覆盖新路由（任务假设不成立）
- [x] 2.3 **验证（路由/权限）**
  - [x] `go build ./...` 通过
  - [x] 无 token → 401；无 teacher 角色 → 403「无权限访问该资源」；teacher 角色 → 通过（实测 uid 65 补角色前后对比）
  - [x] `rbac_policy.csv` 无需新增行——通配符已覆盖；启动日志确认 `GET /teacher/achievements` + `POST .../:id/grant` 已注册

## 3. 契约 — `backend/api/`（与后端同一 PR）

- [x] 3.1 `backend/api/api-teacher/`：新增授予端点文档（请求 `achievement_id` / `student_ids` / `log_id?`，响应授予结果；错误码 400/403/409）
- [x] 3.2 `backend/api/api-parent/07-points.md`：`GET /parent/achievements` 条目补 `log_id`；标注「可空——未绑定见证的授予」
- [x] 3.3 家长端记录契约（`04-growth.md` 或 `07-points.md` 相应位置）：卡片/详情条目补 `badge` 字段，形状 `{name, icon, color_kind}`，**标注「仅见证记录非空」**
- [x] 3.4 契约顶部加增量变更标注：本次为**纯增量**（新增字段/端点），不删不改名，旧客户端不受影响
- [x] 3.5 **验证（契约）**
  - [x] 契约变更已在 proposal 的「What Changes」里显式声明为契约变更
  - [x] 真实请求逐字段核对：`/teacher/achievements` **完全对齐**；`/parent/achievements` 的 `log_id` 正确（见证式有值/非见证式无键）；feed `badge` 条件正确（id=25 有、id=26 无）。**发现 3 处既有漂移**（`icon` vs `icon_url`、`points` 实现返回契约未记、`description` 契约有实现无）——非本次引入，已写进契约注记，未擅自扩面
  - [x] 确认 `Condition`/`TargetValue`/`progress` 未被触碰（契约已标注 progress 属 P2 未实现）

## 4. 老师端 — 授予 UI

- [x] 4.1 `teacher/src/pages/growth/create.vue`：新增「授予徽章」区（可选）——选徽章（本园已启用）、选学生、勾选「本记录作为获奖见证」
- [x] 4.2 [P] `teacher/src/pages/growth/detail.vue`：已发布记录可补授徽章（复用同一授予组件/逻辑）
- [x] 4.3 [P] 新增老师端 API 封装（`teacher/src/api/`）：拉取本园徽章 + 授予
- [x] 4.4 **验证（老师端）**
  - [x] `npm run build:mp-weixin` 通过
  - [x] `check-tokens.mjs teacher` **0 违规**（修正了 badge-pick 的 `12rpx` 裸值 → `$spacing-sm`）
  - [x] 用 `wechat_automator` **tap/input 真实交互**验证：点徽章→展开学生选择+见证勾选→点学生→填内容→发布→自动授予。DB 确认 `log_id=28` 见证绑定正确
  - [x] 真实交互验证：重复授予→**409**；跨班→403；见证学生不可见→400（经 HTTP 实测）
  - [x] 对照 specs `achievement-award`：TC-ACH-001~008 全部通过
  - [x] 已核对：老师端无 tabBar；授予 UI 是页内交互，未新增跳转

## 5. 家长端 — 徽章呈现

- [x] 5.1 `parent/src/pages/growth/index.vue`：卡片渲染徽章（彩底+图标+文字）；`badge` 为空时不渲染徽章区、布局不变
- [x] 5.2 [P] `parent/src/pages/growth/detail.vue`：详情渲染徽章，与卡片一致
- [x] 5.3 徽章配色按 `category` 映射到 token 语义色（design D3：**不落库、不硬编码色值**）——黄 `$color-semantic-warning`/`-bg`、蓝 `$color-semantic-info`/`-bg`
- [x] 5.4 **验证（家长端）**
  - [x] `npm run build:mp-weixin` 通过
  - [x] `node scripts/check-tokens.mjs parent` growth 页面 **0 违规**（8 处全在既有 `uni.scss`）——徽章不得出现 `#hex` 或裸 `rpx`
  - [x] 真实交互验证：见证记录（id=21）显示徽章+`rc-badge--warning` class；非见证（R2/R3/R4）无徽章键；详情同规则
  - [x] 对照 specs：TC-GRW-096 / 097 / 098 全部通过

## 6. 测试用例登记与收尾

- [x] 6.1 `docs/tests/TEST_CATALOG.md`：登记 `TC-ACH-001`~`009`（成就模块）与 `TC-GRW-096`~`012`（成长卡片），回填到 spec 与本文件
- [x] 6.2 **验证（总）**
  - [x] `go vet ./...` 干净；`go test ./...` 全通过（含 8 个徽章用例）
  - [x] 逐条对照 specs 5 个 Requirement / 12 个 Scenario——**全部通过**（真实 HTTP + 真实 UI 点击 + DB 三方确认）
  - [x] 已回写 `docs/achievement-badge-review.md` §10：触点差异、Q1/Q2/Q3 落地、端到端验证、新发现漂移、未完成项
  - [x] 收工清理：已按 PID 杀掉残留 uni watcher（见下）

    ```powershell
    Get-CimInstance Win32_Process -Filter "Name='node.exe'" |
      Where-Object { $_.CommandLine -match "dev:mp-weixin|kindergarten\\(teacher|parent)\\node_modules" } |
      ForEach-Object { Stop-Process -Id $_.ProcessId -Force }
    ```
