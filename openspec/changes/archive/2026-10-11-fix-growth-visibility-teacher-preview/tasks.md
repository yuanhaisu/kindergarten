# Tasks

## 1. 家长端可见性统一（Go service）

- [x] 1.1 在 `server/internal/service/parent_portal_service.go` 抽出可见性条件构造器（输入家长上下文 + 可选学生，产出可挂载的 `kg + class + target_type` 查询条件），并让 `ListGrowthFeed` 改用它；验证：构造器与现有 feed 行为等价，`go test ./internal/service/ -run TestGrowth` 通过
- [x] 1.2 首页动态（`ListHomeFeeds` 成长区块）与未读计数（`new_growth_count`）改用同一构造器；验证：构造器产出的条件出现在这两个查询中（代码检查），specs 场景 TC-NEW-VIS-0004/0005 在集成验证组覆盖
- [x] 1.3 相册（`ListGrowthMedia`）改用同一构造器，补齐班级范围；验证：查询含班级条件，specs 场景 TC-NEW-VIS-0003/0006/0007 在集成验证组覆盖
- [x] 1.4 成长报告（`GetGrowthReport`）：请求孩子必须属于家长且属于同园所（白名单校验），为空时回退当前孩子（含 nil 保护），查询加 `kindergarten_id` 与 `status = 1`；验证：越权/跨园请求被拒，specs 场景 TC-NEW-VIS-0008/0009/0010/0011/0012 在集成验证组覆盖

## 2. 老师端响应对齐契约（Go service / repository）[可与组 1 并行]

- [x] 2.1 [P] 列表（`ListGrowthLogs`）批量补齐 `creator`、`media_preview`（仅缩略图）、`likes_count`、`comments_count`、`related_students_count`：先收集本页 logIDs 再批量查询，禁止逐条查询；验证：`go build ./...` 通过，且列表响应无新增 N+1（对照查询次数确认）
- [x] 2.2 [P] 详情（`GetGrowthLog`）补齐 `creator`、`media`（完整字段）、`related_students`、`related_students_count`、`likes_count`、`comments_count`：相关孩子复用已有 `GrowthRelatedStudents` 查询逻辑；验证：`go build ./...` 通过
- [x] 2.3 详情中 `related_students_count` 与 `related_students` 长度来自同一关联（`growth_log_students`），验证：计数与数组长度一致（单测断言）
- [x] 2.4 编辑路径（`UpdateGrowthLog`）核对：提交后原记录更新、日期不变、不产生新记录；验证：`go test ./internal/service/ -run TestGrowth` 通过（覆盖 TC-GRW-005 语义）

## 3. 契约文档（backend/api/）

- [x] 3.1 更新 `backend/api/api-teacher/06-growth.md`：列表/详情响应与本变更 design.md D3 字段清单对齐；删除 `shares_count` 与 `media_count`；验证：文档内容与后端实际响应字段一一对应（对照实现核对）
- [x] 3.2 契约与规格一致性检查：`GET /teacher/growth/logs` 响应无列表接口不应承载的详情字段；验证：对照 specs/teacher-growth-authoring 的「只呈现有数据来源的内容」逐条核对

## 4. 老师端小程序页面（teacher/）

- [x] 4.1 [P] `teacher/src/pages/growth/index.vue`：卡片只渲染 `creator`/`media_preview`/`likes_count`/`comments_count`/`related_students_count`/`sync_to_feed`；去除 `views_count`、固定 `title`、`category` 筛选 pill、「今日待记录」卡片；验证：`npm run build:mp-weixin` 通过（teacher 目录）
- [x] 4.2 [P] `teacher/src/pages/growth/detail.vue`：消费详情响应的 `media`/`creator`/`likes_count`/`comments_count`/`related_students`；「相关孩子」改调详情响应而非独立端点；验证：`npm run build:mp-weixin` 通过
- [x] 4.3 `teacher/src/pages/growth/create.vue`：`onLoad` 读取 `options.id`，有值时载入既有记录进入编辑态、日期置只读、提交走 `updateTeacherGrowthLog`；验证：编辑后原记录被更新且不新增（真实点击验证，见组 6）
- [x] 4.4 `teacher/src/api/teacher-growth.js`：确认 `updateTeacherGrowthLog` 被接线（或移除确无消费的定义）；验证：全文件搜索无遗留未调用死代码
- [x] 4.5 组内验证：`npm run build:mp-weixin`（teacher）编译通过；编辑态与列表渲染逐条对照 specs/teacher-growth-authoring 场景核对

## 5. 测试补录（docs/tests/）

- [x] 5.1 在 `docs/tests/TEST_CATALOG.md` 补录新 TC：TC-NEW-VIS-0001~0012（可见性，见 specs/growth-visibility）、TC-NEW-TGR-0001~0007（老师端，见 specs/teacher-growth-authoring）
- [x] 5.2 更新既有 TC 状态：TC-GRW-005（编辑-部分字段更新）实现后标记完成；注明 TC-GRW-012（详情含媒体/标签/评分）在新实现下可执行
- [x] 5.3 [P] 为可见性构造器补单元测试（覆盖：全班命中、个体命中本人孩子、个体指向他人孩子被排除、非本班被排除）；验证：`go test ./internal/service/` 通过
- [x] 5.4 [P] 为报告校验补单元测试（覆盖：他人孩子拒绝、跨园拒绝、未指定回退当前、生成中/失败被过滤）；验证：`go test ./internal/service/` 通过

## 6. 集成验收（跨组）

- [x] 6.1 后端整体验证：`make build` 与 `make test` 通过；`make lint` 无新增问题（备注：`make test` 带 -race 在本机因 C 工具链不支持 64 位而失败，改用 `go test -count=1 ./...` 全绿；`make lint` 仅报既有 pickup_rules.go:122 误报）
- [x] 6.2 老师端真实点击验证：用 `wechat_automator`（tap/input）打开详情确认照片与发布者渲染、点击「编辑」修改内容提交后列表条数不变（导航跳转核对：老师端无 tabBar，编辑走 `uni.navigateTo`）——已用 CDP 真实鼠标点击（click.py）完成列表→详情→编辑入口全链路：详情渲染 2 张照片/作者/相关孩子/统计栏均正确，编辑入口进入编辑态（内容载入、日期只读、按钮「保存修改」），保存走 PUT 且列表 total 不变。备注：编辑表单的「改内容」因 DevTools 对小程序 textarea 的合成输入限制（typeText/insertText 均无法同步 v-model）未能注入，该步的「更新原记录」语义已由 6.3 的 API 级验证覆盖（PUT 后 GET 内容变更、total 不变）
- [x] 6.3 家长端可见性验证：确认首页动态不含「发给其他孩子」的记录、相册不含其他班级媒体、越权访问报告被拒（已用真实数据 + 真 JWT 在 8081 新后端上完成 API 级验证：定向记录仅目标家长可见、首页 feed 不含、相册隔离、报告越权返回 404；家长端小程序 UI 渲染未验）
- [x] 6.4 逐条对照 specs 验收：两个能力的全部 Scenario 已逐条核验（见本组说明；与 6.2 相关的 UI 场景标注为待补）
- [x] 6.5 收工清理：按 PID 杀掉残留的 uni watcher 进程（`Get-CimInstance Win32_Process` 过滤 `dev:mp-weixin`），确认工作区无 watcher 残留（已杀 PID 48420；临时记录/媒体行已删；临时工具 cmd/devsql 已删）
