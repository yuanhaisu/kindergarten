# Tasks

## 1. 契约（先改）

- [x] 1.1 `backend/api/api-teacher/09-class-and-tasks.md`：补 `GET /teacher/class/list` 文档与 `parent_count` 字段说明（口径：在读学生绑定的去重家长用户数）
- [x] 1.2 对照 design「契约字段溯源」核对，确认只加 `parent_count`

## 2. 后端

- [x] 2.1 `teacher_portal_service.go` `ListTeacherClasses`：一条 `GROUP BY class_id` 批量统计家长数（`students.status=1`、双方未软删、`COUNT(DISTINCT student_parents.user_id)`）
- [x] 2.2 空班级返回 0，不返回 null；不改变既有字段与排序
- [x] 2.3 [P] 确认未改通知投递逻辑、未改 `student_count`、无表结构变化

## 3. 老师端页面

- [x] 3.1 `teacher/src/pages/notice/create.vue`：班级弹层行显示「班级名 · N 位家长」
- [x] 3.2 底部接收范围显示「N 个班级 · M 位家长」（M=已选班级之和），文案不承诺与通知实际接收人逐人一致
- [x] 3.3 班级接口失败/字段缺失时不阻塞选择，回退只显示班级名

## 4. 测试

- [x] 4.1 service 测试：多班批量计数、同一家长两个孩子同班只计 1、非在读学生不计、无家长为 0、班级隔离
- [x] 4.2 在 `docs/tests/TEST_CATALOG.md` 登记 `TC-NEW-CLS-0001`–`0005`

## 5. 验证

- [x] 5.1 `cd server && go build ./...` exit 0；定向 `go test ./internal/service/ -run 'Class'` exit 0
- [x] 5.2 真实 HTTP：`/teacher/class/list` 每班含 `parent_count`，与 SQL 复核一致；N 班只有一条聚合查询（日志或 SQL 计数）
- [x] 5.3 老师端 DevTools：通知创建页弹层与底部文案真实点击验证；字段缺失回退；Console 0 error
- [x] 5.4 `node scripts/check-tokens.mjs teacher` 不高于基线；`node scripts/check-secrets.mjs --files <manifest>` 0 findings
- [x] 5.5 逐条对照 `specs/class-roster/spec.md` 验收，未验证不得宣布完成
