# Workbook · 12 achievements

> Status: `Done`

## 1. 元信息

- **模块编号**: `12` · **Sprint**: `S4 · P2b` · **接口数**: `4`
- **规格**: [`../12-achievements.md`](../12-achievements.md) · **API**: `admin/api/achievements.ts` · **页面**: `admin/pages/achievements/index.vue`
- **Status**: `Done` · **Finished**: `2026-04-21`

## 2. Domain / 存储

- `Achievement` — `domain/points.go`（表 `achievements`）：新增 `kindergarten_id`、`category`、`target_value`、`status`；`icon` ↔ API `icon_url`
- `StudentAchievement` — 列表 `earned_count` 子查询：`student_achievements` JOIN `students` 且 `students.kindergarten_id = achievements.kindergarten_id`

## 3. Interface Progress

| # | 路径 | 完成 |
|---|-----|:---:|
| GET /admin/achievements | [x] |
| POST /admin/achievements | [x] |
| PUT /admin/achievements/:id | [x] |
| DELETE /admin/achievements/:id | [x] |

## 4. Notes

- `category_label` 由服务端根据 `category` 映射（attendance/growth/points/task 等），未知分类回传原串或「其他」。
- `PUT` 仅处理 `name` / `description` / `target_value`；`icon_url` / `category` / `condition` 若前端后续要编辑可再扩展。
- 旧库若已有 `achievements` 行无 `kindergarten_id`，迁移后需数据修复或置默认园所。

## 5. Final Summary

- `AchievementRepo` / `AchievementService` / `achievements.go`；`wire_gen.go` 手改注入；`go generate ./internal/app` 仍可能因 go.sum 缺间接依赖失败。
