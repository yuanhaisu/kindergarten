# Workbook · 10 tasks

> Status: `Done`

## 1. 元信息

- **模块编号**: `10` · **Sprint**: `S4 · P2b` · **接口数**: `6`
- **规格**: [`../10-tasks.md`](../10-tasks.md) · **API**: `admin/api/tasks.ts` · **页面**: `admin/pages/tasks/index.vue`
- **Status**: `Done` · **Finished**: `2026-04-21`

## 2. Domain

- `ParentTask` / `TaskCheckIn` — `domain/task.go`
- 完成率：`在读学生数` × 按 `frequency`（daily/weekly/once）推算的理论打卡上限 vs 实际 `task_check_ins` 条数

## 3. Interface Progress

| # | 路径 | 完成 |
|---|-----|:---:|
| GET /admin/tasks | [x] |
| GET /admin/tasks/:id | [x] |
| DELETE /admin/tasks/:id | [x] |
| GET /admin/tasks/stats | [x] |
| POST /admin/tasks | [x] |
| PUT /admin/tasks/:id | [x] |

## 4. Notes

- `creator_id` 新建时为当前 admin 用户；`PUT` 全量更新含 `status`。
- `GET /admin/tasks/stats`：`start_date`/`end_date` 须成对或同时省略（默认当月）。
