# Workbook · 08 pickup

> Status: `Done`

## 1. 元信息

- **模块编号**: `08`
- **Sprint**: `S3`（P2a）
- **接口数**: `4`
- **规格文档**: [`../08-pickup.md`](../08-pickup.md)
- **Finished**: `2026-04-21`

## 2. Domain

- `PickupRecord`：已有 `is_abnormal`、`abnormal_reason`、`operator_id`、`recorded_at` 等。
- `PickupAuthorization`：新增 `review_status`（0 待审 / 1 通过 / 2 拒绝，默认 1 兼容历史）、`audited_at`、`audit_remark`。

## 4. Interface Progress

| 路径 | 完成 |
|---|---|
| GET `/admin/pickup/overview` | [x] |
| GET `/admin/pickup/records` | [x] |
| GET `/admin/pickup/abnormal` | [x] |
| PUT `/admin/pickup/authorized-persons/:id/audit` | [x] |

## 6. Notes

- 概览：`total_students` 为在读学生数；`not_picked_up` 为当日有送入且无接走的去重学生数；`by_class` 与全园班级列表对齐并补零。
- 授权审核：仅 `review_status = 0` 可审；历史数据默认 `review_status = 1`。

## 9. DoD

- [x] `kindergarten_id` 过滤
- [x] 审核写 `operation_logs`
- [x] `go build ./...`
