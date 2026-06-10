# Workbook · 15 audit

- **Status**: `Done`
- **规格**: [`../15-audit.md`](../15-audit.md)
- **Finished**: 2026-04-21

## Interface Progress

| 接口 | 完成 |
|---|---|
| GET /admin/audit/list | [x] |
| PUT /admin/audit/:id | [x] |
| GET /admin/audit/stats | [x] |

## 实现要点

- `AuditRepo`：`content_audits` 左联 `notices` / `growth_daily_logs` / `users` / `classes`；筛选 `type`、`status`（pending/passed/rejected）。
- 园所 `kindergarten_app_settings.extra_config` 中 `notice.need_audit` / `growth.need_audit` 为 true 时，在列表/统计前 **幂等** 补插待审行（已排期/已发通知 `status IN (1,2)`；全部未删成长日志），使列表 `id` 恒为 `content_audits.id`，与 `PUT /admin/audit/:id` 一致。
- `target_type=comment` 预留域名常量；无业务写入时列表为空，统计 `by_type.comment` 为 0。
- 写操作记 `operation_logs`（`module=audit`, `action=decide`）。

## DoD

- [x] 对齐 `15-audit.md` 响应形状
- [x] `go build ./...`
- [x] `PROGRESS` / `README` / `SESSION_HANDOFF` / 样例 `.http`
