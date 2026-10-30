# Design

## Context

见 `proposal.md - Why`。现状：`leave_requests` 只有学生/事由/日期/天数/审批字段；家长侧 `parentLeaveMap` 只映射基本字段；附件、接送人、类型、审批人、班级、统计均缺失；撤销只校验归属。老师端审批按 `status` 流转，新增列不得影响其读写。

## Goals / Non-Goals

**Goals:**

- 用最小 schema 变更支撑 P06 三页全部字段。
- 撤销状态守卫与契约一致。
- 统计口径为「家长全部孩子」，与筛选解耦。

**Non-Goals:**

- 不做老师端附件展示、审批通知、`cancelled_at`、管理端。
- 不重构请假审批流。

## Decisions

### D1 类型与接送人用可空列

`leave_requests` 增 `leave_type varchar(16) default 'other'`、`emergency_contact varchar(100) default ''`。选列而非新表：单值字段，无需关联。存量行按默认值「其他」/空。

### D2 附件用独立表

新增 `leave_attachments(id, leave_id, file_url, file_type, file_name, file_size, sort_num)`，仿 `notice_attachments`。理由：与既有附件模式一致，便于按序返回与后续扩展；单列 JSON 在 GORM/迁移上更易出错。

### D3 统计由后端一次 GROUP BY 给出

列表响应增 `stats{all,pending,approved}`：对家长所有孩子 `GROUP BY status` 计数，不受 `status` 筛选影响。备选「前端取全量再算」被否决：`page_size` 上限 50 会截断。

### D4 审批人/班级补齐

- `reviewer`：按 `reviewer_id` 批量取用户昵称（列表一次 IN 查询，避免 N+1）。
- `class_name`：详情按学生当前班级取；学生转班以当前班级为准。

### D5 撤销守卫与标签

- `CancelLeave` 仅 `status=pending`；否则业务错误（400 语义）。
- 后端标签统一：`pending=待审批`、`approved=已同意`、`rejected=已拒绝`、`canceled=已取消`；详情「已撤销」由端上按 canceled 映射，后端不特判。

## 跨端触点清单

| 层 | 是否改 | 说明 |
| --- | --- | --- |
| Go domain | 会改 | `leave.go` 增两列；新增 `LeaveAttachment` 模型 |
| repository | 会改 | 附件 CRUD、审批人批量查询、统计聚合 |
| service | 会改 | 创建写类型/接送人/附件；列表/详情映射；撤销守卫；标签统一 |
| handler | 会改 | 创建绑定新字段与附件；列表无需新参数 |
| Casbin policy | 不改 | 现有家长请假路由已覆盖 |
| router.go | 不改 | 端点不变 |
| backend/api 契约 | 会改（契约变更） | `api-parent/03-leave.md`：新字段与撤销规则 |
| 端页面 | 本片不改 | P06 紧随其后消费；teacher/admin 不涉及 |
| design token | 不改 | 无 UI |
| 迁移 | 会改 | AutoMigrate 增列/建表；`dbcheck leave_requests` 与 `dbcheck leave_attachments` 核实 |

### 契约字段溯源

| 字段 | 出处 |
| --- | --- |
| `leave_type` | 设计 `parent/07-leave-create` 三类型卡（病假/事假/其他） |
| `attachments` | 同页「病假条、就诊记录等照片」最多 3 张 |
| `emergency_contact` | 同页「接送人（选填）」 |
| `reviewer.nickname` | 设计 06 审批意见署名、08 时间线「王老师」 |
| `class_name` | 设计 07「为 乐乐 · 小一班」、08 学生信息 |
| `stats` | 设计 06 统计 3 等分（全部/待审批/已同意） |

## Risks / Trade-offs

- [新增列对老师端读写影响] → 列可空/有默认；老师端查询不 SELECT 新列不受影响；回归老师请假测试。
- [AutoMigrate 只增不改] → 仅新增列与新表；用 `go run ./cmd/dbcheck` 核实。
- [附件孤儿] → 请假删除为软删时附件保留；本片不做清理，记录为 P2。
- [标签统一影响既有页面] → 家长端现有页面用 `status_label`，新值更规范；P06 会重做。

## Migration Plan

`AutoMigrate` 增列 `leave_type`/`emergency_contact` 与建表 `leave_attachments`；启动后 `dbcheck leave_requests` / `dbcheck leave_attachments` 核实。回滚需手工删列（可选，数据可保留）。
