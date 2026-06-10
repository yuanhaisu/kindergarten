# Workbook · 09 growth

> Status: `Done`

---

## 1. 元信息

- **模块编号**: `09`
- **模块名**: `growth`（中文：成长记录）
- **Sprint**: `S4 · P2b`
- **接口数**: `5`
- **规格文档**: [`../09-growth.md`](../09-growth.md)
- **前端 API**: `admin/api/growth.ts`
- **前端页面**: `admin/pages/growth/index.vue`
- **前端类型**: `admin/types/api.d.ts`（`GrowthLogItem` 等）
- **Status**: `Done`
- **Started**: `2026-04-21`
- **Finished**: `2026-04-21`

---

## 2. Domain 核对

| 涉及实体 | 文件 | 说明 |
|---|---|---|
| `GrowthDailyLog` 等 | `domain/growth.go` | 列表/详情/删除基于 `growth_daily_logs` |
| `ContentAudit` | `domain/audit.go` | `target_type=growth` 表示审核状态；无记录 → API 返回 `pending` |
| `GrowthMedia` / `GrowthLike` / `GrowthComment` | `domain/growth.go` | 计数与详情媒体列表 |

**是否需要新建表**：否（沿用 `content_audits`）

---

## 3. Interface Progress

| # | 方法 | 路径 | 前端函数 | 完成 | 完成日期 | 备注 |
|---|---|---|---|:---:|---|---|
| 1 | GET | `/admin/growth/logs` | `getGrowthLogs` | [x] | 2026-04-21 | |
| 2 | GET | `/admin/growth/logs/:id` | `getGrowthLog` | [x] | 2026-04-21 | |
| 3 | DELETE | `/admin/growth/logs/:id` | `deleteGrowthLog` | [x] | 2026-04-21 | 软删 + operation_logs |
| 4 | PUT | `/admin/growth/logs/:id/audit` | `auditGrowthLog` | [x] | 2026-04-21 | `action`: pass/reject |
| 5 | GET | `/admin/growth/stats` | `getGrowthStats` | [x] | 2026-04-21 | 与 dashboard 分接口；`start_date`/`end_date` 须成对或同时省略（默认当月） |

---

## 4. Questions

- 无 `content_audits` 记录时列表/详情 `audit_status` 为 `pending`（待审），与历史「免审即发布」数据如需默认 `passed` 可再议。

---

## 5. DoD checklist（模块）

- [x] 五接口实现 + `kindergarten_id` 过滤
- [x] `go build ./...`
- [x] Wire：`wire_gen.go` 手改注入 `GrowthService`（本机 `go generate` 缺依赖时）
- [x] `PROGRESS.md` / `README.md` / `SESSION_HANDOFF.md` / 规格头状态
- [x] `samples/09-growth/growth.http`

**Final Summary**: 管理端成长模块已接 `content_audits` 审核流；统计接口字段对齐 `09-growth.md`（`period` / `by_class` / `daily_trend`）。
