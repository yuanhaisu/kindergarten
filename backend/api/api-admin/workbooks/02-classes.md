# Workbook · 02 classes

> Status: `Done`

## 1. 元信息

- **模块编号**: `02` · **模块名**: `classes`（班级管理）· **Sprint**: `S1`
- **接口数**: `6` · **规格**: [`../02-classes.md`](../02-classes.md)
- **前端**: `admin/api/classes.ts` · `admin/pages/classes/index.vue` · `admin/types/api.ts`
- **Status**: `Done` · **Started / Finished**: `2026-04-20`

## 2. Domain

- `domain.Class` 已扩展：`grade`, `academic_year`, `capacity`, `status`（AutoMigrate 加列）
- 调班：`move_in` 时 `from_class_id` 为来源班；`move_out` 时 `from_class_id` 表示**目标班**（与规格字段名一致，语义见下）

## 3. Interface Progress

| # | 方法 | 路径 | 完成 |
|---|-----|------|:---:|
| 1 | GET | `/admin/classes` | [x] |
| 2 | GET | `/admin/classes/:id` | [x] |
| 3 | POST | `/admin/classes` | [x] |
| 4 | PUT | `/admin/classes/:id` | [x] |
| 5 | DELETE | `/admin/classes/:id` | [x] |
| 6 | POST | `/admin/classes/:id/adjust` | [x] |

## 4. Decisions

- `move_out`：学生从路径 `:id` 班级**调出至** body 的 `from_class_id`（目标班）；`move_in`：从 `from_class_id` **调入**路径 `:id` 班级。
- 写操作写入 `operation_logs`（`OperationLogRepo`）。
- 删除班级前先 `Unscoped` 清理 `class_teachers`，且仅 `student_count==0` 可删。

## 5. DoD

- [x] JSON 与规格 / 前端类型一致 · [x] `list` 空为 `[]` · [x] `kindergarten_id` 过滤 · [x] 写操作日志 · [x] `go build ./...`

## 6. Final Summary

- 建议 commit：`feat(admin): complete classes module (10/87)`
