# Workbook · 18 settings

> Status: **Done**

---

## 1. 元信息

- **模块编号**: `18`
- **模块名**: `settings`（中文：系统设置 / 园所信息）
- **Sprint**: `S2`（P1）
- **接口数**: `6`
- **规格文档**: [`../18-settings.md`](../18-settings.md)
- **前端 API**: `admin/api/settings.ts` + `admin/api/kindergarten.ts`
- **前端页面**: `admin/pages/settings/index.vue`
- **前端类型**: `admin/types/api.d.ts`（`SystemSettings` / `KindergartenInfo`）
- **Started**: `2026-04-21`
- **Finished**: `2026-04-21`

---

## 2. Domain 核对

| 涉及实体 | 文件 | 说明 |
|---|---|---|
| `Kindergarten` | `domain/kindergarten.go` | 扩展园所档案字段（电话、园长、logo、许可证、容量、创办日、简介） |
| `AcademicYear` | `domain/academic_year.go` | 新表：学年 + 春秋学期起止、`is_current` |
| `KindergartenAppSetting` | `domain/kindergarten_app_setting.go` | 新表：`current_semester` + `extra_config` JSON（出勤/通知/成长/接送/积分） |

---

## 3. Interface Progress

| # | 方法 | 路径 | 前端函数 | 完成 | 完成日期 |
|---|---|---|---|:---:|---|
| 1 | GET | `/admin/settings` | `getSettings` | [x] | 2026-04-21 |
| 2 | PUT | `/admin/settings` | `updateSettings` | [x] | 2026-04-21 |
| 3 | GET | `/admin/settings/academic-years` | `getAcademicYears` | [x] | 2026-04-21 |
| 4 | POST | `/admin/settings/academic-years` | `createAcademicYear` | [x] | 2026-04-21 |
| 5 | GET | `/admin/kindergarten/info` | `getKindergartenInfo` | [x] | 2026-04-21 |
| 6 | PUT | `/admin/kindergarten/info` | `updateKindergartenInfo` | [x] | 2026-04-21 |

---

## 4. Decisions & Notes

- `[2026-04-21]` `basic` 块：`academic_year` 与学期日期来自 `academic_years`（优先 `is_current`，否则最新一条）；`semester` 默认 `spring`，存 `kindergarten_app_settings.current_semester`（当前无单独 API 修改，仅占位）。
- `[2026-04-21]` 首个学年的 `POST` 自动 `is_current=true`，后续学年 `is_current=false`（规格未提供切换当前学年接口）。
- `[2026-04-21]` `current_count` 为在读学生数（`status=1`）；`class_count` / `teacher_count` 为园所维度统计。
- `[2026-04-21]` 写操作：`update_settings`、`create_academic_year`、`update_kindergarten_info` 写入 `operation_logs`。

---

## 5. Test Samples

- [x] `samples/18-settings/settings.http`

---

## 6. DoD Checklist

- [x] Response 与 `18-settings.md` 字段一致（蛇形、嵌套）
- [x] 学年列表空为 `[]`
- [x] Service 层以 JWT `kindergarten_id` 为界（无 kg 则 `NO_KINDERGARTEN`）
- [x] 写操作 `operation_logs`
- [x] `go build ./...` 通过

---

## 7. Final Summary

- **实现**: `SettingsService` + `handler/http/settings.go` + `registerAdminRoutes`；新表 AutoMigrate；`wire_gen.go` 手改注入。
