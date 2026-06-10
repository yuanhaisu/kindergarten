# Workbook · 20 export

> Status: **Done**

## 1. 元信息

- **模块**: `20 export` · Sprint **S5**
- **规格**: [`../20-export.md`](../20-export.md)
- **Status**: `Done` · **Finished**: `2026-04-22`

## 2. Interface Progress

| # | 路径 | 完成 |
|---|------|:---:|
| POST `/admin/export/attendance` | [x] |
| POST `/admin/export/students` | [x] |
| POST `/admin/export/growth` | [x] |
| GET `/admin/export/:taskId` | [x] |

**实现说明**：`export_tasks` 表；POST 入队后 `goroutine` 生成 xlsx（复用 `AttendanceService.ExportReport`、`StudentService.ExportStudentsXLSX`、`GrowthService.ExportGrowthLogsXLSX`）；`GET /admin/export/download/:taskId` 带 JWT 下载（前端 `downloadExportBlob`）。

## 3. DoD

- [x] `kindergarten_id` 隔离
- [x] `go build ./...`
- [x] `samples/20-export/export.http`
