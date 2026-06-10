# Workbook · 01 dashboard

> Status: `Done`

---

## 1. 元信息

- **模块编号**: `01`
- **模块名**: `dashboard`（中文：数据看板）
- **Sprint**: `S1`（P0）
- **接口数**: `4`
- **规格文档**: [`../01-dashboard.md`](../01-dashboard.md)
- **前端 API**: `admin/api/dashboard.ts`
- **前端页面**: `admin/pages/index.vue`
- **前端类型**: `admin/types/api.ts`（DashboardOverview 等）
- **Status**: `Done`
- **Started**: `2026-04-20`
- **Finished**: `2026-04-20`

---

## 2. Domain 核对

| 涉及实体 | 文件 | 当前字段够吗 | 备注 |
|---|---|---|---|
| Kindergarten / Student / Class / Attendance 等 | 已有 | ✅ | 统计聚合 |

**是否需要新建表**：否

---

## 3. Interface Progress

| # | 方法 | 路径 | 前端函数 | 完成 | 完成日期 |
|---|---|---|---|:---:|---|
| 1 | GET | `/admin/dashboard/overview` | `getDashboardOverview` | [x] | 2026-04-20 |
| 2 | GET | `/admin/dashboard/attendance-trend` | `getAttendanceTrend` | [x] | 2026-04-20 |
| 3 | GET | `/admin/dashboard/class-ranking` | `getClassRanking` | [x] | 2026-04-20 |
| 4 | GET | `/admin/dashboard/growth-stats` | `getGrowthStats` | [x] | 2026-04-20 |

---

## 4. Decisions & Notes

- `[2026-04-20]` 「今日」「本月」、趋势日序：统一 **Asia/Shanghai**。
- `[2026-04-20]` `not_picked_up_count` = max(0, present − picked_up)。
- `[2026-04-20]` 班级排行按 **出勤率（present/total）** 降序，同分按班级名。

---

## 5. Questions

- 无

---

## 6. Test Samples

- [x] `samples/01-dashboard/overview.http`

---

## 9. DoD Checklist

### 接口级
- [x] Response JSON 字段与 MD 规格一致
- [x] 空列表返回 `[]`（`class-ranking.list`、`attendance-trend.data` 等）
- [x] Service 层以 `kindergarten_id` 为界统计（Repo 查询均带园所条件）
- [x] 只读接口，豁免 operation_logs
- [x] `go build ./...` 通过

### 模块级
- [x] 所有接口 Interface Progress 全勾
- [ ] `admin/pages/index.vue` 整页 CRUD（看板只读；需用户本地起 server+admin 点验）
- [x] `PROGRESS.md` 该模块状态 ✅
- [x] `README.md` 模块索引状态 ✅
- [x] `SESSION_HANDOFF.md` 已更新

---

## 11. Final Summary

- **Git commits**: （请本地提交）建议 `feat(admin): complete dashboard module (4/87)`
