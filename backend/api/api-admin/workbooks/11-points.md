# Workbook · 11 points

> Status: `Done`

## 1. 元信息

- **模块编号**: `11` · **Sprint**: `S4 · P2b` · **接口数**: `5`
- **规格**: [`../11-points.md`](../11-points.md) · **API**: `admin/api/points.ts` · **页面**: `admin/pages/points/index.vue`
- **Status**: `Done` · **Finished**: `2026-04-21`

## 2. Domain / 存储

- `PointAccount`、`PointTransaction` — `domain/points.go`
- 规则：`kindergarten_app_settings.extra_config` 内 `points` 对象；键：`task_checkin_points`、`weekly_attendance_bonus`、`monthly_attendance_bonus`、`growth_featured_points`（规则 id 4）

## 3. Interface Progress

| # | 路径 | 完成 |
|---|-----|:---:|
| GET /admin/points/overview | [x] |
| GET /admin/points/records | [x] |
| POST /admin/points/adjust | [x] |
| GET /admin/points/rules | [x] |
| PUT /admin/points/rules | [x] |

## 4. Notes

- 流水 `amount`：与 DB `points` 同号（获得为正、消费为负）；`type` 由符号推导。
- `POST /adjust`：`source=admin_adjust`；余额不足 → `40041`。
- `AdjustPoints` 响应 `created_at` 为处理时刻（UTC）；若需与 DB 完全一致可改为查事务行。

## 5. Final Summary

- 路由注册在 `registerAdminRoutes` 的 `/points` 子组；Wire 注入 `PointService`。
