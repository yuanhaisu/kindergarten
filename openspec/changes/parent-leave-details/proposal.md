# Proposal

## Why

家长请假设计（`parent/06-leave-index`、`07-leave-create`、`08-leave-detail`）要求请假类型（病假/事假/其他）、审批人姓名、班级、附件、接送人、全量统计与撤销守卫；契约 `api-parent/03-leave.md` 已声明其中多数，但实现只落学生/事由/日期：无 `leave_type`、无附件与接送人、列表/详情不返回审批人与班级、无统计，且撤销不校验状态（已同意/已驳回也能撤销，违反契约）。P06 三页因此无法按设计落地。

## What Changes

- 新增请假类型 `leave_type`：`sick` 病假 / `personal` 事假 / `other` 其他；创建与编辑可写，列表/详情返回代码与中文标签。
- 详情补 `class_name`、`reviewer{nickname}`、`attachments[]`、`emergency_contact`；列表补 `reviewer.nickname`（审批意见署名）。
- 附件：最多 3 张图片、单张 ≤20MB，新表 `leave_attachments` 存储；创建时写入，详情按顺序返回。
- 列表返回全量 `stats{all,pending,approved}`（该家长所有孩子的请假计数，不随筛选变化）。
- 撤销仅允许 `pending`；其它状态返回业务错误。
- 状态标签统一为：待审批 / 已同意 / 已拒绝 / 已取消；详情页如需「已撤销」由端上映射。
- **本次将修改 `backend/api/api-parent/03-leave.md`，属于契约变更。**
- 明确不做（out of scope）：
  - 不改老师端审批流程与页面；不改审批通知。
  - 不做请假附件在老师端的展示（本片只做家长端落库与回显）。
  - 不做 `cancelled_at` 字段（设计明确无时间）。
  - 不改管理端。

## Capabilities

### New Capabilities

- `parent-leave`: 家长请假的类型、附件、审批信息、统计与撤销规则。

### Modified Capabilities

（无）

## Impact

- 契约：`backend/api/api-parent/03-leave.md`（新增 leave_type、stats、attachments、emergency_contact、reviewer、class_name；撤销规则）。
- 后端：`domain/leave.go`（新列/新表）、`repository`（附件）、`service/parent_portal_service.go`、`handler/http/parent_portal.go`、`provider.go`（AutoMigrate）。
- 家长端：P06 三页消费；本片不改页面（P06 紧随其后）。
- 数据：`leave_requests` 增可空列；新表 `leave_attachments`；AutoMigrate 只增不改，需 `dbcheck` 核实。
