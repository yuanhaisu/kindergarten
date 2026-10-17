# Tasks

> 分组按触点切，每组自带该组产出的测试与验证门。
> 标 `[P]` 的任务可与同组其它 `[P]` 任务并行。
> **MCP 验证一律串行**；UI 交互验证必须真实点击（`wechat_automator` tap/input 或真机），不能用 `wechat_navigate` 替代。
> 本变更不涉及跳转与 tabBar（详见 design.md 跳转核对），页面验证以表单交互与列表展示为主。

## 1. 契约（backend/api/）— 契约先行

- [x] 1.1 [P] `backend/api/api-teacher/04-notices.md`：发布/编辑入参 `class_id` 替换为 `class_ids`（多班级，均须为任教班）；列表项与详情的 `class_id`/`class_name` 替换为 `class_names`。验证：文档字段清单与 design.md「契约字段级清单」逐条对上，无清单外字段
- [x] 1.2 [P] `backend/api/api-admin/13-notices.md`：发布/编辑入参 `class_id` 替换为 `class_ids`，新增可选 `grades`（小班/中班/大班）；列表/详情 `class_id`/`class_name` 替换为 `class_names`。验证：同上；显式标注本次为契约变更
- [x] 1.3 【本组验证】契约变更声明完整；`read_stats` 等分母字段标注口径变化（形状不变、语义变）；`git diff backend/api/` 只含本次相关改动；无 `target_count` 等派生字段夹带

## 2. 后端 domain + repository + 迁移

- [x] 2.1 [P] `server/internal/domain/notice.go`：新增 `NoticeTarget`（`NoticeID`+`ClassID`，联合唯一索引，含 `kindergarten_id` 不复制 E1 旧缺陷）；`Notice.ClassID` 标注弃用注释。验证：`go build ./...` 通过
- [x] 2.2 [P] 回填 SQL 落 `docs/migrations/`（幂等 `INSERT ... SELECT ... WHERE NOT EXISTS`，SQL 写法见 design.md Migration Plan）。验证：SQL 文件存在且幂等条件完整
- [x] 2.3 `server/internal/repository/notice_repo.go`：目标集读写；家长/老师可见性改 EXISTS；分母统计改为「目标范围 `COUNT(DISTINCT sp.user_id)`」单一实现；已读/确认仍按 user 去重。验证：单测覆盖——多孩家长分母只计 1、两班目标的命中/未命中、迁移等价（单目标=原单班）
- [x] 2.4 【本组验证】`go vet ./...` 与 `go test ./repository/...` 通过；执行回填后 `go run ./cmd/dbcheck notice_targets` 核实表与唯一索引；对账 SQL（design.md 步骤 3）行数相等

## 3. 后端 service + handler

- [x] 3.1 `teacher_portal_service.go`：发布/编辑接收 `class_ids`，逐个校验任教关系，空集拒绝；列表/详情可见性改「命中任教班级集」；统计换共用分母。验证：单测覆盖 spec「老师一次发给两个班」「非任教班被拒」「目标为空被拒」「老师可查命中通知」「非目标班不可见」
- [x] 3.2 `notice_service.go`（管理端）：发布/编辑接收 `class_ids`+`grades`（展开为班级快照，与手工勾选并集去重）；列表/详情响应 `class_names` 与新分母。验证：单测覆盖「年级快照展开」「混用去重」「两端已读率一致」
- [x] 3.3 `parent_portal_service.go`：列表/首页动态/新通知计数改 EXISTS 命中判定，一条通知只出一行（多孩合并）；读/确认维持按用户一次。验证：单测覆盖「双孩分属两目标班只出一条」「仅一孩命中也可见」「家长看不到无关通知」「确认一次对双孩生效」
- [x] 3.4 `handler/http/`（`teacher_portal.go`/`notices.go`/`parent_portal.go`）：入参与响应形状随 1.1/1.2 契约。验证：`go build ./...`；用旧入参（单 `class_id`）打发布端点应明确报参数错而非静默单班
- [x] 3.5 【本组验证】`go vet ./...` 与 `go test ./...` 全部通过；对照 `specs/notice-targeting/spec.md` 逐条核对 6 条 Requirement 主路径有单测；管理端与老师端对同一通知的 `read_rate`/`total_count` 数值一致（造 1 条双班通知+双孩家长后断言）

## 4. 端页面

- [x] 4.1 [P] `teacher/src/pages/notice/create.vue` + `api/teacher-notices.js`：「接收范围」单选改多选（任教班级多选），提交 `class_ids`。验证：`npm run dev:mp-weixin` 构建通过；automator tap 完成「选两班 → 发布」
- [x] 4.2 [P] `admin/pages/notices/index.vue` + `api/notices.ts`：发布对话框班级多选 + 年级快捷圈选（小班/中班/大班）；列表「发布范围」列展示 `class_names`。验证：`npm run build` 通过；浏览器走通「圈大班+勾一个班 → 发布 → 列表见多班」
- [x] 4.3 【本组验证】两端构建通过；家长端打开通知列表**不出现重复卡片**（多孩场景若无双孩测试号，用同班双通知反向排除重复渲染问题）；UI 交互均真实点击（`wechat_automator` tap 或真机/浏览器），不以 MCP 导航充数；收工按 PID 杀残留 uni watcher

## 5. 测试目录补录 + 收尾

- [x] 5.1 `docs/tests/TEST_CATALOG.md` 登记本次新 TC（NTC 模块）：TC-NEW-NTC-0001 多班发布、0002 非任教班拒绝、0003 目标为空拒绝、0004 年级快照展开、0005 混用去重、0006 双孩合并展示、0007 单孩命中可见、0008 家长不越界可见、0009 老师命中可查、0010 非目标老师不可见、0011 双孩分母计一、0012 两端已读率一致、0013 存量迁移行为等价。验证：catalog 中 13 条 TC 与 spec 场景一一对应
- [x] 5.2 【最终验收】对照 `specs/notice-targeting/spec.md` 逐条勾验（6 Requirement / 13 Scenario），全部通过才可进入 archive；`openspec validate --changes --strict` 通过；未逐条验证不得宣布完成
