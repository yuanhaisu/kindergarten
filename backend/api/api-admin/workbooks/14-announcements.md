# Workbook · 14 announcements

- **Status**: `Done`
- **规格**: [`../14-announcements.md`](../14-announcements.md)
- **Finished**: 2026-04-21

## Interface Progress

| 接口 | 完成 |
|---|---|
| GET /admin/announcements | [x] |
| GET /admin/announcements/:id | [x] |
| POST /admin/announcements | [x] |
| PUT /admin/announcements/:id | [x] |
| DELETE /admin/announcements/:id | [x] |
| PUT /admin/announcements/:id/top | [x] |

## 实现要点

- `domain.Announcement` 扩展：`cover_url`、`is_important`、`target_roles`（JSON 文本）、`view_count`、`scheduled_at`；状态 0/1/2/3 与 MD 已发布=2 对齐。
- `AnnouncementAttachment` 表；`AnnouncementRepo` + `AnnouncementService` + `handler/http/announcements.go`；`kindergarten_id` 过滤；写操作 `operation_logs`。
- `POST` 返回 HTTP 201；列表 `creator` 仅 `nickname`（与 MD 一致）。

## DoD

- [x] 字段对齐 `14-announcements.md`
- [x] `go build ./...`
- [x] `PROGRESS` / `README` / `SESSION_HANDOFF` / 样例 `.http`
