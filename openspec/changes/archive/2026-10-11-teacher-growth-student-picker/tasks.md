# Tasks

## 1. 后端：候选范围与关联持久化（Go service）

- [x] 1.1 `classStudentIDs`（`server/internal/service/teacher_portal_service.go`）的查询补 `status = 1`，并在方法注释写明口径（本班在读）；验证：`go build ./...` 通过，且接送页/成长记录两处调用方共用新口径（对照查询串确认）——**核实为既有代码已满足**（HEAD 即含 `status = 1` 与「本班全部在读学生」注释；两处调用方 :1136/:1202 共用）
- [x] 1.2 `UpdateGrowthLog`：`target_type=2` 时校验 `student_ids` 非空（空则返回 `ErrTeacherInvalidParams`）；验证：`go test ./internal/service/ -run TestGrowth` 通过
- [x] 1.3 `UpdateGrowthLog` 在事务内重建 `growth_log_students`：`target_type=2` 先删后插、`target_type=1` 清空关联；验证：`go build ./...` 通过，事务包裹（主表更新与关联重建同成败）

## 2. 契约文档（backend/api/）

- [x] 2.1 更新契约文档：`GET /teacher/class/students` 的 Query 补 `status`（候选范围=在读）——实际落在 **`backend/api/api-teacher/09-class-and-tasks.md`**（非任务原写的 06-growth.md），并顺带校正了该处文档与实现的漂移（`student_id` 键/接送向字段/全量返回）；`PUT /teacher/growth/logs/:id` 的 `student_ids` 语义（按提交值重建；改为全班时清除）补在 `06-growth.md`。验证：两处文档与实现逐字段核对，无响应字段增删——已在组 2 完成，勾选为补录

## 3. 老师端页面（teacher/）

- [x] 3.1 [P] `create.vue` 选择器字段对齐：`:key` 与 `:class` 统一读 `s.student_id`（含授徽章面板的 `student-row` 选择器与 `grant.student_ids`）；验证：`npm run build:mp-weixin`（teacher 目录）通过
- [x] 3.2 `create.vue` 选择器可用性：加姓名搜索框（本地过滤）、「已选 N 人」计数、空态区分（班级暂无学生 / 未找到匹配的学生）；验证：`npm run build:mp-weixin` 通过
- [x] 3.3 编辑态回填核对：`loadExistingLog` 把详情 `related_students[].id` 映射进 `form.student_ids`，配合 3.1 的字段对齐后已选学生呈现选中态；验证：本地构建通过 + 集成验证组用真实数据回显
- [x] 3.4 组内验证：`npm run build:mp-weixin`（teacher）编译通过；逐条对照 specs/growth-target-students 的交互场景自查

## 4. 测试补录（docs/tests/）

- [x] 4.1 在 `docs/tests/TEST_CATALOG.md` 补录新 TC：TC-NEW-TGP-0001~0011（见 specs/growth-target-students）
- [x] 4.2 [P] 为「搜索过滤」抽出纯函数并补单元测试——**核实后判定不适合**：过滤的生产实现在前端 `create.vue` 的 `filteredStudents`（本地 computed），Go 侧不存在对应逻辑；在 Go 层造一个仅供测试的函数属于「为测试而造死代码」，违背仓库惯例。过滤语义由 UI 验证（5.3）覆盖。原函数已删除，构建通过

## 5. 集成验收（跨组）

- [x] 5.1 后端整体验证：`make build` 与 `go test -count=1 ./...` 通过；`make lint` 无新增问题
- [x] 5.2 API 级验证（真实数据 + JWT）：个体记录创建 → `related_students` 正确；PUT 改学生名单后 GET 反映新名单且无重复；PUT 改为全班后 `related_students` 为空；重复 PUT 同名单幂等（TC-NEW-TGP-0009/0010/0011）——**全部 PASS**，另覆盖「复活软删行」与「空名单拒绝状态不变」
- [x] 5.3 老师端真实交互验证（click.py / evaluate.py 走 CDP；输入用 `Input.insertText`）：候选渲染 4 人、点击高亮（selected 0→1→2→1）、搜索「刘」→ 仅刘晨晨、无结果空态「未找到匹配的学生」、已选计数实时更新、编辑态回显 2 人已选（TC-NEW-TGP-0003~0008）——**全部 PASS**
- [x] 5.4 逐条对照 specs/growth-target-students 验收：11 条场景逐项核对通过（含 TC 引用对应关系）
- [x] 5.5 收工清理：临时记录已删除；uni watcher 已杀；`devsql` 已于上一变更删除
