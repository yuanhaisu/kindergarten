# Workbook · 07 leaves

> Status: `Done`

---

## 1. 元信息

- **模块编号**: `07`
- **模块名**: `leaves`（中文：请假管理）
- **Sprint**: `S3`（P2a）
- **接口数**: `3`（2 GET + 1 PUT）
- **规格文档**: [`../07-leaves.md`](../07-leaves.md)
- **前端 API**: `admin/api/leaves.ts`
- **前端页面**: `admin/pages/leaves/index.vue`
- **Started**: `2026-04-21`
- **Finished**: `2026-04-21`

---

## 2. Domain 核对

| 涉及实体 | 文件 | 当前字段够吗 |
|---|---|---|
| `LeaveRequest` | `server/internal/domain/leave.go` | ✅ |

**是否需要新建表**：否（`leave_requests` 已 AutoMigrate）

---

## 4. Interface Progress

| # | 方法 | 路径 | 前端函数 | 完成 | 完成日期 |
|---|---|---|---|:---:|---|
| 1 | GET | `/admin/leaves` | `getLeaves` | [x] | 2026-04-21 |
| 2 | GET | `/admin/leaves/stats` | `getLeaveStats` | [x] | 2026-04-21 |
| 3 | PUT | `/admin/leaves/:id/audit` | `auditLeave` | [x] | 2026-04-21 |

---

## 6. Decisions & Notes

- `[2026-04-21]` `by_reason` 按文案关键词归入「生病 / 家事 / 其他」，与 MD 示例一致。
- `[2026-04-21]` 列表日期筛选：同时传 `start_date`+`end_date` 时按请假区间与查询区间重叠过滤。
- `[2026-04-21]` `GET /stats` 未传日期时默认当前自然月（上海时区）。

---

## 9. DoD Checklist

### 接口级
- [x] Response JSON 字段与 MD 规格一致
- [x] 空列表返回 `[]`
- [x] Service 层 `kindergarten_id` 过滤
- [x] 审批写入 `operation_logs`
- [x] `go build ./...` 通过

### 模块级
- [x] `PROGRESS.md` / `README.md` / `SESSION_HANDOFF.md` 已更新
- [x] 前端 `auditLeave` + 列表页审批按钮

---

## 11. Final Summary

- **总会话数**: `1`
- **REST 样例**: `samples/07-leaves/leaves.http`
