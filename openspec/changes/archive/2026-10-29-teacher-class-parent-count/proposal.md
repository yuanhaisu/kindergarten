# Proposal

## Why

老师端「发通知」需要告诉老师这条通知会触达多少家长（设计 `teacher/03-notice-create`：「1 个班级 · 28 位家长将收到通知」），但 `GET /teacher/class/list` 只返回班级名与职务，没有家长人数；该端点还未写进契约。页面目前只能显示「N 个班级」，老师无法判断触达范围。

## What Changes

- `GET /teacher/class/list` 每班新增 `parent_count`：该班**在读学生**（`students.status = 1`，未软删）绑定的**去重家长 user 数**；列表级一条 `GROUP BY` 批量查询，不逐班查询。
- 补文档化 `/teacher/class/list`（`backend/api/api-teacher/09-class-and-tasks.md`），字段逐一说明来源。**本次将修改 `backend/api/api-teacher/09-class-and-tasks.md`，属于契约变更。**
- 老师端通知创建页在班级选择弹层与接收范围文案显示「班级名 + N 位家长」，底部合计按已选班级相加。
- 明确不做（out of scope）：
  - 不改通知实际投递逻辑与接收人集合（仍沿用现有 target 计算）。
  - 不做跨班家长去重（同一家长两个孩子选两个班时，合计可能重复计数，作为已知限制记录）。
  - 不改学生人数口径（`student_count` 维持现状），不改管理端/家长端。
  - 不加表、不改表结构，无迁移。

## Capabilities

### New Capabilities

- `class-roster`: 老师端班级列表的家长触达人数口径与读取行为。

### Modified Capabilities

（无）

## Impact

- 契约：`backend/api/api-teacher/09-class-and-tasks.md`（补 `/teacher/class/list` 与 `parent_count`）。
- 后端：`server/internal/service/teacher_portal_service.go`（`ListTeacherClasses` 批量计数）、定向测试。
- 老师端：`teacher/src/pages/notice/create.vue`（弹层与底部文案）、`teacher/src/api/teacher-notices.js` 或对应封装（如无需改则不碰）。
- 其它端：parent / admin 不受影响；管理端「人数」仍为学生数。
- 数据：只读 `students`、`student_parents`，无新表。
