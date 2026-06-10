# Workbook · 13 notices

- **Status**: `Done`
- **规格**: [`../13-notices.md`](../13-notices.md)
- **Finished**: 2026-04-21

## Interface Progress

| 接口 | 完成 |
|---|---|
| GET /admin/notices | [x] |
| GET /admin/notices/:id | [x] |
| POST /admin/notices | [x] |
| PUT /admin/notices/:id | [x] |
| DELETE /admin/notices/:id | [x] |
| PUT /admin/notices/:id/audit | [x] |

## 实现要点

- `NoticeRepo` + `NoticeService` + `handler/http/notices.go`；`kindergarten_id` 在列表/详情/写路径校验；`read_rate` 基于 `notice_read_logs` 与班级家长 `student_parents` 去重统计。
- 写操作写 `operation_logs`；审核写 `content_audits`（`target_type=notice`）。
- 前端：`createNotice` / `updateNotice`；通知页支持关键词、发布/编辑弹窗、列表操作。

## DoD

- [x] 接口与 MD 字段对齐
- [x] `go build ./...` 通过
- [x] `PROGRESS` / `README` / `SESSION_HANDOFF` / 样例 `.http`
