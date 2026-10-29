# Design

## Context

见 `proposal.md - Why`。现状约束：

- `GET /teacher/class/list` 由 `ListTeacherClasses` 返回 `class_id/class_name/title/title_label`，**未写进契约**。
- 老师端仅 `notice/create.vue` 消费该端点；班级选择弹层目前只显示班级名 + 职务。
- 家长计数已有先例：通知/任务投递侧用 `COUNT(DISTINCT student_parents.user_id)`，只过滤双方软删，不过滤学生状态。
- 班级学生数 `student_count` 只对单班 `class/info` 生效，且过滤 `students.status=1`。

## Goals / Non-Goals

**Goals:**

- 列表级一次查询给出每个班的家长人数。
- 口径可解释、与「家长将收到通知」的直觉一致。
- 端点进入契约，后续可被其它老师端页面复用。

**Non-Goals:**

- 不改通知投递目标计算；不保证与某条通知的实际接收人数完全一致。
- 不做跨班家长去重。
- 不改 `student_count` 口径，不改管理端。

## Decisions

### D1 家长人数 = 在读学生绑定的去重家长用户数

计数条件：`students.status = 1` 且双方未软删，`COUNT(DISTINCT student_parents.user_id)`。

理由：班级列表表达的是「当前这个班的家长」，毕业/转出的学生家长不应计入。备选「沿用通知侧的不过滤学生状态」被否决——那会把已离园学生的家长算进触达范围，与课堂语境不符。

风险：与通知实际接收人集合可能不同（通知侧按任务/通知目标计算）。已在 spec 中明确两者口径不同，页面文案不承诺逐人一致。

### D2 列表级批量查询

`ListTeacherClasses` 一次 `GROUP BY class_id` 聚合，禁止逐班 `COUNT`（N+1）。返回顺序按现有班级排序，不影响计数。

### D3 契约补文档化

`backend/api/api-teacher/09-class-and-tasks.md` 增加 `GET /teacher/class/list` 与 `parent_count` 字段说明；字段溯源见下表。不改变响应包络。

### D4 老师端展示

`teacher/src/pages/notice/create.vue` 的班级弹层行显示「班级名 · N 位家长」；底部接收范围显示「N 个班级 · M 位家长」，M 为已选班级 `parent_count` 之和。跨班同一家长可能被重复计数，作为已知限制记录，不在本片解决。

## 跨端触点清单

| 层 | 是否改 | 说明 |
| --- | --- | --- |
| Go domain | 不改 | 计数为查询结果，无新表/字段 |
| repository | 可能新增 | 可复用现有 repo 或在 service 内一条聚合查询；不新增表 |
| service | 会改 | `teacher_portal_service.go` `ListTeacherClasses` 增加批量计数 |
| handler | 会改 | 列表响应增加字段（包络不变） |
| Casbin policy | 不改 | 端点已在 teacher 策略内 |
| router.go | 不改 | 路由已存在 |
| backend/api 契约 | 会改（契约变更） | `api-teacher/09-class-and-tasks.md` 新增端点与 `parent_count` |
| 端页面 | teacher 会改 | `teacher/src/pages/notice/create.vue`；parent/admin 不改 |
| design token | 不改 | 仅文案与既有行布局，无新样式 |
| 迁移 | 不改 | 无表结构变化 |

### 契约字段溯源

| 字段 | 出处 |
| --- | --- |
| `parent_count` | 设计 `teacher/03-notice-create` 接收范围「1 个班级 · 28 位家长将收到通知」与班级弹层行人数 |
| `GET /teacher/class/list` | 现有未被文档化的端点，老师端通知创建页在用；本次补文档，不新增能力 |

## Risks / Trade-offs

- [与通知实际接收人数不一致] → spec 明确区分；页面文案用「位家长」表示潜在触达，不复述通知接收人。
- [跨班同家长重复计数] → 已知限制；如后续需要精确合计，新增按班级集合去重的接口。
- [学生状态变化导致数字波动] → 实时查询，不做缓存；与班级学生数口径一致（status=1）。
- [列表性能] → 一条 GROUP BY，索引走 `students(class_id)` 与 `student_parents(student_id)`；如慢再评估覆盖索引。

## Migration Plan

无 schema 变更，无迁移。回滚仅需回退代码与契约。
