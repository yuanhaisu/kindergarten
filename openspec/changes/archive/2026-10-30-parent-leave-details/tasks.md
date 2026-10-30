# Tasks

## 1. 契约（先改）

- [x] 1.1 `api-parent/03-leave.md`：列表补 `stats{all,pending,approved}` 与 `reviewer.nickname`；详情补 `leave_type/leave_type_label/class_name/reviewer/attachments/emergency_contact`
- [x] 1.2 创建请求补 `leave_type/emergency_contact/attachments`；撤销限 pending
- [x] 1.3 状态标签统一为待审批/已同意/已拒绝/已取消

## 2. Domain / 迁移

- [x] 2.1 `domain/leave.go`：`LeaveType`（sick/personal/other）、`EmergencyContact` 列；新增 `LeaveAttachment`
- [x] 2.2 `provider.go` 登记新模型；启动后 `go run ./cmd/dbcheck leave_requests`、`dbcheck leave_attachments` 核实

## 3. Repository / Service / Handler

- [x] 3.1 创建：写类型（缺省 other）、接送人、附件（≤3 图片、单张 ≤20MB）
- [x] 3.2 列表：`stats` 一次聚合（不受筛选影响）、`reviewer` 批量昵称、标签统一
- [x] 3.3 详情：`class_name`、`reviewer`、`attachments` 排序、`emergency_contact`、`leave_type`
- [x] 3.4 撤销：仅 pending，其余业务错误
- [x] 3.5 [P] 确认老师端审批读写不受影响

## 4. 测试

- [x] 4.1 service 测试：类型缺省/显式、附件 2 张与 4 张、审批人与班级、统计不随筛选、撤销四种状态、标签一致、接送人
- [x] 4.2 在 `docs/tests/TEST_CATALOG.md` 登记 `TC-NEW-LV-0001`–`0009`

## 5. 验证

- [x] 5.1 `cd server && go build ./...` exit 0；定向 `go test ./internal/service/ -run 'Leave'` exit 0
- [x] 5.2 真实 HTTP：创建带类型/接送人/附件 → 列表 stats 与 reviewer → 详情字段 → 撤销守卫
- [x] 5.3 老师端请假审批回归（列表/同意/拒绝）通过
- [x] 5.4 secret 0；`dbcheck` 输出核实；测试 0 残留
- [x] 5.5 逐条对照 `specs/parent-leave/spec.md` 验收
