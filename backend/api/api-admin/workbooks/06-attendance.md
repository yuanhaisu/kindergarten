# Workbook · 06 attendance

> Status: **Done**

## 1. 元信息

- **模块编号**: `06` · **Sprint**: `S3`
- **规格**: [`../06-attendance.md`](../06-attendance.md)
- **前端**: `admin/api/attendance.ts` · `admin/pages/attendance/index.vue`
- **Finished**: `2026-04-21`

## 2. Domain

- `attendances`（`kindergarten_id` + `class_id` + `student_id` + `date` + `status`）；统计口径 **Asia/Shanghai** 自然日（与 dashboard 一致）
- 迟到（`AttendanceLate`）计入出勤人数与 `present` 筛选；全园汇总 `absent = total_students - present - leave`（与班级公式对齐）

## 3. Interface Progress

| # | 接口 | 完成 |
|---|------|:---:|
| 1 | GET `/admin/attendance/overview` | [x] |
| 2 | GET `/admin/attendance/records` | [x] |
| 3 | GET `/admin/attendance/report` | [x] |

## 4. Decisions

- 导出：`start_date` / `end_date` 必填；`format` 默认 xlsx；Blob 不走 Envelope
- 明细 `pickup_time` 暂无字段，固定 `""`
- 读接口不写 `operation_logs`（规格未要求）

## 5. Samples

- [x] `samples/06-attendance/attendance.http`
