# Admin 接口实现 · 会话接力卡

> **用途**：会话之间的"接力棒"。每次开始/结束会话都要读/改这份文件。
>
> **体量越大越重要**：这是保证中断可恢复、方向不漂移的唯一机制。
>
> **工作流**：采用"一模块一会话"模式，详见 [`KICKOFF_PROMPT.md`](./KICKOFF_PROMPT.md) 和 [`EXECUTION_PLAN.md`](./EXECUTION_PLAN.md) §0。

---

## 当前状态（2026-04-22 更新）

- **Current Sprint**: `S6 · 回归测试`（**进行中**：文档与缺失清单已结案；E2E 待接入 Playwright）
- **Current Module**: `—`（admin 表内 95 接口已齐）
- **Active Workbook**: —
- **Completed So Far**: `95 / 95` 表内 API 行 + `7 / 8` 脚手架任务（缺可选 S0-7）
- **Overall Progress**: `脚手架 87.5% · 接口 100%（表内行）· S6 文档项 3/4`

---

## 🎯 Next Action（下次会话第一件事）

> **做什么（二选一）**：
> 1. **接 Playwright**：在 `admin/` 增加 `@playwright/test`、示例 `tests/e2e/*.spec.ts`，再跑全套并把 `PROGRESS.md` §S6 第一项勾上；或  
> 2. **补 S0-7**：本地 seed（1 园所 + 3 班 + 10 学生 + 1 admin），便于手工回归。
>
> **可选**：本机 Wire：`go mod tidy` 后 `go generate ./internal/app`。

**说明**：`20260422/BACKEND_MISSING_APIS.md` 已结案为「0 缺失」；`README.md` 已链到该文件。历史 `20260420` 报告保留作基线。

---

## 📋 本会话规则

每次进入 admin 接口实现相关的会话时，必须：

1. **开场**：
   - 读本文件 → 看 `Next Action`
   - 读 `PROGRESS.md` → 确认进度
   - 读 `EXECUTION_PLAN.md` §5（Recipe）和 §9（约定），保持一致性
2. **干活时**：
   - 每完成 1 个接口 / 1 个 S0 任务，立即在 `PROGRESS.md` 打勾
   - 遇到需要决策的问题，先写入下方 `Blockers`，向用户确认再往下走
3. **收尾**：
   - 更新本文件的 `Current Sprint / Module / Next Action / Last Modified Files / Completed This Session / Blockers`
   - 如代码阶段性稳定，提醒用户 commit

---

## 📜 Session Log（按时间倒序）

### Session #21 · 2026-04-22（S6 · 文档回归部分结案）

- **Completed**:
  - 新增 `admin/tests/playwright-reports/20260422/BACKEND_MISSING_APIS.md`（**0** 缺失，指向 `PROGRESS.md` 95/95；保留 `20260420` 为历史）
  - `backend/api/api-admin/README.md`：最新缺失对照链接 + 历史基线说明
  - `PROGRESS.md` §S6：3/4 项打勾；Sprint 表 **S6** → `3/4`、🔄
- **Not done（需后续）**: `admin` 未安装 Playwright，无法在本仓库执行 `playwright-*` 全套 E2E
- **Next Action**: 接入 Playwright **或** 做 S0-7 seed
- **Blockers**: 无
- **Last Modified Files**:
  - `admin/tests/playwright-reports/20260422/BACKEND_MISSING_APIS.md`
  - `backend/api/api-admin/README.md`
  - `backend/api/api-admin/PROGRESS.md`
  - `backend/api/api-admin/SESSION_HANDOFF.md`

### Session #20 · 2026-04-22（S5 · 20 export 全 4 接口）

- **Completed**:
  - `domain.ExportTask` + AutoMigrate；`ExportTaskRepo`；`ExportService`（POST 入队 + goroutine 写临时目录 xlsx；GET 状态；`GET /admin/export/download/:taskId` 流式下载）
  - 复用 `AttendanceService.ExportReport`、`StudentService.ExportStudentsXLSX`；新增 `GrowthService.ExportGrowthLogsXLSX` + `GrowthRepo.ListLogsForExportDateRange`
  - `handler/http/export.go`、`registerAdminRoutes`、`wire_gen.go`
  - 前端：`admin/api/export.ts` `downloadExportBlob`；`admin/pages/export/index.vue` 按钮触发 Blob 下载（Bearer）
  - `PROGRESS` **95/95**、S5 **20/20** ✅；`20-export.md` / `README`；`workbooks/20-export.md`、`samples/20-export/export.http`
- **Next Action**: S6 回归 / 可选 S0-7 seed
- **Blockers**: 无（`go build ./...` 通过）
- **Last Modified Files**:
  - `server/internal/domain/export_task.go`
  - `server/internal/repository/export_task_repo.go`
  - `server/internal/repository/growth_repo.go`
  - `server/internal/service/export_task_service.go`
  - `server/internal/service/growth_service.go`
  - `server/internal/service/wire.go`
  - `server/internal/repository/wire.go`
  - `server/internal/handler/http/export.go`
  - `server/internal/handler/http/handler.go`
  - `server/internal/app/router.go`
  - `server/internal/app/wire_gen.go`
  - `server/internal/app/provider.go`
  - `admin/api/export.ts`
  - `admin/pages/export/index.vue`
  - `backend/api/api-admin/PROGRESS.md`
  - `backend/api/api-admin/README.md`
  - `backend/api/api-admin/20-export.md`
  - `backend/api/api-admin/SESSION_HANDOFF.md`
  - `backend/api/api-admin/workbooks/20-export.md`
  - `backend/api/api-admin/samples/20-export/export.http`

### Session #19 · 2026-04-22（S5 · 19 logs · GET /admin/logs）

- **Completed**:
  - `OperationLogRepo.ListForKindergarten`（分页、模块/操作人/日期过滤，`COALESCE` 昵称）
  - `OperationLogService`：`module` 查询与库内 `students`/`teachers` 等 plural 映射；`module_label` / `action_label` / `target` / `detail` 组装
  - `handler/http/logs.go` · `registerAdminRoutes` · `GET /admin/logs` · `Handler.oplogSvc` · `wire_gen.go`
  - `PROGRESS` `91/95`、`S5 16/20`；`19-logs.md` / `README`；`workbooks/19-logs.md`、`samples/19-logs/logs.http`
- **Next Action**: `POST /admin/export/attendance`（20 export）
- **Blockers**: 无（`go build ./...` 通过）
- **Last Modified Files**:
  - `server/internal/repository/operation_log_repo.go`
  - `server/internal/service/operation_log_service.go`
  - `server/internal/service/wire.go`
  - `server/internal/handler/http/logs.go`
  - `server/internal/handler/http/handler.go`
  - `server/internal/app/router.go`
  - `server/internal/app/wire_gen.go`
  - `backend/api/api-admin/PROGRESS.md`
  - `backend/api/api-admin/README.md`
  - `backend/api/api-admin/19-logs.md`
  - `backend/api/api-admin/SESSION_HANDOFF.md`
  - `backend/api/api-admin/workbooks/19-logs.md`
  - `backend/api/api-admin/samples/19-logs/logs.http`

### Session #18 · 2026-04-21（S5 · 15 audit 全 3 接口）

- **Completed**:
  - `domain`：`AuditTargetComment`；`AuditRepo`（列表/统计/更新/幂等补 pending）；`AuditService`（`need_audit` 读 `kindergarten_app_settings.extra_config`）；`handler/http/audit.go`；`/admin/audit/*` 路由；`wire` / `wire_gen`
  - 列表 `id` 为 `content_audits.id`；园所开启 notice/growth 需审时，对尚无审核行的已排期/已发通知与成长日志补 `status=0` 行，便于统一 PUT
  - `PROGRESS` `90/95`、`S5 15/20`；`15-audit.md` / `README`；`workbooks/15-audit.md`、`samples/15-audit/audit.http`
- **Next Action**: `GET /admin/logs`（19 logs）
- **Blockers**: 无（`go build ./...` 通过）
- **Last Modified Files**:
  - `server/internal/domain/audit.go`
  - `server/internal/repository/audit_repo.go`
  - `server/internal/service/audit_service.go`
  - `server/internal/handler/http/audit.go`
  - `server/internal/handler/http/handler.go`
  - `server/internal/repository/wire.go`
  - `server/internal/service/wire.go`
  - `server/internal/app/router.go`
  - `server/internal/app/wire_gen.go`
  - `backend/api/api-admin/PROGRESS.md`
  - `backend/api/api-admin/README.md`
  - `backend/api/api-admin/15-audit.md`
  - `backend/api/api-admin/SESSION_HANDOFF.md`
  - `backend/api/api-admin/workbooks/15-audit.md`
  - `backend/api/api-admin/samples/15-audit/audit.http`

### Session #17 · 2026-04-21（S5 · 14 announcements 全 6 接口）

- **Completed**:
  - `domain.Announcement` 扩展字段 + `AnnouncementAttachment`；`AnnouncementRepo` / `AnnouncementService` / `handler/http/announcements.go`
  - `registerAdminRoutes` `/announcements/*`；`Handler.announcementSvc`、`repository`/`service` `wire.go`、`wire_gen.go`；`provider` AutoMigrate 附件表
  - 前端 `admin/types/api.d.ts`（`AnnouncementDetail`）、`admin/api/announcements.ts`；`14-announcements.md` / `PROGRESS`（`87/95`）/ `README` / `workbooks/14-announcements.md` / `samples/14-announcements/announcements.http`
- **Next Action**: `GET /admin/audit/list`（15 audit）
- **Blockers**: 无（`go build ./...` 通过）
- **Last Modified Files**:
  - `server/internal/domain/announcement.go`
  - `server/internal/repository/announcement_repo.go`
  - `server/internal/service/announcement_service.go`
  - `server/internal/handler/http/announcements.go`
  - `server/internal/handler/http/handler.go`
  - `server/internal/repository/wire.go`
  - `server/internal/service/wire.go`
  - `server/internal/app/router.go`
  - `server/internal/app/wire_gen.go`
  - `server/internal/app/provider.go`
  - `admin/types/api.d.ts`
  - `admin/api/announcements.ts`
  - `backend/api/api-admin/14-announcements.md`
  - `backend/api/api-admin/PROGRESS.md`
  - `backend/api/api-admin/README.md`
  - `backend/api/api-admin/SESSION_HANDOFF.md`
  - `backend/api/api-admin/workbooks/14-announcements.md`
  - `backend/api/api-admin/samples/14-announcements/announcements.http`

### Session #16 · 2026-04-21（S5 · 13 notices 全 6 接口 + 前端）

- **Completed**:
  - `NoticeRepo` / `NoticeService` / `handler/http/notices.go`；`registerAdminRoutes` `/notices/*`；`Handler.noticeSvc`、`wire.go` / `wire_gen.go`
  - 前端：`admin/api/notices.ts`（`createNotice` / `updateNotice`）、`admin/types/api.d.ts`、`admin/pages/notices/index.vue`（搜索、发布/编辑弹窗）
  - `workbooks/13-notices.md`、`samples/13-notices/notices.http`；`13-notices.md` / `PROGRESS.md` / `README.md`
- **Next Action**: `GET /admin/announcements`（14 announcements）
- **Blockers**: 无（`go build ./...` 通过；`go generate` 环境限制同前）
- **Last Modified Files**:
  - `server/internal/repository/notice_repo.go`
  - `server/internal/service/notice_service.go`
  - `server/internal/handler/http/notices.go`
  - `server/internal/handler/http/handler.go`
  - `server/internal/repository/wire.go`
  - `server/internal/service/wire.go`
  - `server/internal/app/router.go`
  - `server/internal/app/wire_gen.go`
  - `admin/api/notices.ts`
  - `admin/types/api.d.ts`
  - `admin/pages/notices/index.vue`
  - `backend/api/api-admin/13-notices.md`
  - `backend/api/api-admin/PROGRESS.md`
  - `backend/api/api-admin/README.md`
  - `backend/api/api-admin/SESSION_HANDOFF.md`
  - `backend/api/api-admin/workbooks/13-notices.md`
  - `backend/api/api-admin/samples/13-notices/notices.http`

### Session #15 · 2026-04-21（S4 · 12 achievements 全 4 接口）

- **Completed**:
  - `domain.Achievement` 扩展：`kindergarten_id`、`category`、`target_value`、`status`；列表 `earned_count` 子查询
  - `AchievementRepo` / `AchievementService` / `handler/http/achievements.go`；`Handler` / `router` / `wire_gen.go` / `repository`+`service` `wire.go`
  - `workbooks/12-achievements.md`、`samples/12-achievements/achievements.http`；`PROGRESS` / `README` / `12-achievements.md`；**S4 已闭环**
- **Next Action**: `GET /admin/notices`（13 notices）
- **Blockers**: 无（`go build ./...` 通过；`go generate ./internal/app` 仍缺 wire 间接 go.sum）
- **Last Modified Files**:
  - `server/internal/domain/points.go`
  - `server/internal/repository/achievement_repo.go` (new)
  - `server/internal/service/achievement_service.go` (new)
  - `server/internal/handler/http/achievements.go` (new)
  - `server/internal/handler/http/handler.go`
  - `server/internal/repository/wire.go`
  - `server/internal/service/wire.go`
  - `server/internal/app/router.go`
  - `server/internal/app/wire_gen.go`
  - `backend/api/api-admin/PROGRESS.md`
  - `backend/api/api-admin/README.md`
  - `backend/api/api-admin/12-achievements.md`
  - `backend/api/api-admin/SESSION_HANDOFF.md`
  - `backend/api/api-admin/workbooks/12-achievements.md` (new)
  - `backend/api/api-admin/samples/12-achievements/achievements.http` (new)

### Session #14 · 2026-04-21（S4 · 11 points 全 5 接口 + 接线）

- **Completed**:
  - `PointRepo` / `PointService` / `handler/http/points.go` 已接入：`Handler.pointSvc`、`repository`/`service` `wire.go`、`registerAdminRoutes` `/points/*`、`wire_gen.go` 注入
  - `mergePointsPatchIntoExtra` / `readPointsFromExtra`：非法 `extra_config` 时回退默认 JSON
  - `workbooks/11-points.md`、`samples/11-points/points.http`；`PROGRESS` / `README` / `11-points.md` / 本文件
- **Next Action**: `GET /admin/achievements`（12 achievements）
- **Blockers**: 无（`go build ./...` 通过）
- **Last Modified Files**:
  - `server/internal/handler/http/handler.go`
  - `server/internal/handler/http/points.go`
  - `server/internal/service/point_service.go`
  - `server/internal/repository/point_repo.go`
  - `server/internal/repository/wire.go`
  - `server/internal/service/wire.go`
  - `server/internal/app/router.go`
  - `server/internal/app/wire_gen.go`
  - `backend/api/api-admin/PROGRESS.md`
  - `backend/api/api-admin/README.md`
  - `backend/api/api-admin/11-points.md`
  - `backend/api/api-admin/SESSION_HANDOFF.md`
  - `backend/api/api-admin/workbooks/11-points.md` (new)
  - `backend/api/api-admin/samples/11-points/points.http` (new)

### Session #13 · 2026-04-21（S4 · 09 growth 全 5 接口）

- **Completed**:
  - `GrowthRepo`：列表（子查询媒体/赞/评/审核）、详情预加载、媒体列表、软删、`content_audits` 单条 upsert；统计（总量、按班、按日）
  - `GrowthService` + `handler/http/growth.go`：五路由；`audit_status` 无审核记录为 `pending`；删除/审核写 `operation_logs`
  - `registerAdminRoutes`：`/growth/stats`、`/growth/logs`…；`Handler` / `wire_gen.go` 注入 `GrowthService`；`repository`/`service` `wire.go` 追加 `GrowthRepo`/`GrowthService`
  - `workbooks/09-growth.md`、`samples/09-growth/growth.http`；`PROGRESS` / `README` / `09-growth.md`
- **Decisions**:
  - 审核态：`content_audits`（`target_type=growth`）；无记录 → API `audit_status: pending`
  - `GET /admin/growth/stats`：`start_date` 与 `end_date` 须同时传或同时省略（省略则默认当月）；响应字段对齐 `09-growth.md`（非 dashboard 结构）
- **Next Action**: `GET /admin/tasks`（10 tasks）
- **Blockers**: 无（本环境 `go generate ./internal/app` 因 wire 间接依赖缺 go.sum 项失败，已手改 `wire_gen.go`）
- **Last Modified Files**:
  - `server/internal/repository/growth_repo.go` (new)
  - `server/internal/service/growth_service.go` (new)
  - `server/internal/handler/http/growth.go` (new)
  - `server/internal/handler/http/handler.go`
  - `server/internal/repository/wire.go`
  - `server/internal/service/wire.go`
  - `server/internal/app/router.go`
  - `server/internal/app/wire_gen.go`
  - `backend/api/api-admin/PROGRESS.md`
  - `backend/api/api-admin/README.md`
  - `backend/api/api-admin/09-growth.md`
  - `backend/api/api-admin/SESSION_HANDOFF.md`
  - `backend/api/api-admin/workbooks/09-growth.md` (new)
  - `backend/api/api-admin/samples/09-growth/growth.http` (new)

### Session #12 · 2026-04-21（S3 · 08 pickup 全 4 接口）

- **Completed**:
  - `PickupRepo`：概览统计（去重送入/接走、未接走 NOT EXISTS、异常条数、按班聚合）；记录/异常列表 JOIN；授权审核查询与更新
  - `PickupAuthorization` 域模型：`review_status`、`audited_at`、`audit_remark`（默认 `review_status=1` 兼容历史）
  - `PickupService` + `handler/http/pickup.go`：四个路由；审核写 `operation_logs`
  - `workbooks/08-pickup.md`、`samples/08-pickup/pickup.http`；`PROGRESS` / `README` / `08-pickup.md`
  - `Handler` / `wire_gen.go` 注入 `PickupService`
- **Decisions**:
  - 概览 `not_picked_up`：当日有「送入」且无「接走」的学生去重；`by_class` 与班级列表对齐补零
  - 异常列表 `is_authorized`：`pickup_person_id != nil` 视为已关联授权
  - 审核：`action` approve/reject；仅 `review_status=0` 可操作
- **Next Action**: `GET /admin/growth/logs`（09 growth）
- **Blockers**: 无
- **Last Modified Files**:
  - `server/internal/domain/pickup.go`
  - `server/internal/repository/pickup_repo.go` (new)
  - `server/internal/service/pickup_service.go` (new)
  - `server/internal/handler/http/pickup.go` (new)
  - `server/internal/handler/http/handler.go`
  - `server/internal/repository/wire.go`
  - `server/internal/service/wire.go`
  - `server/internal/app/router.go`
  - `server/internal/app/wire_gen.go`
  - `backend/api/api-admin/PROGRESS.md`
  - `backend/api/api-admin/README.md`
  - `backend/api/api-admin/08-pickup.md`
  - `backend/api/api-admin/workbooks/08-pickup.md` (new)
  - `backend/api/api-admin/samples/08-pickup/pickup.http` (new)
  - `backend/api/api-admin/SESSION_HANDOFF.md`

### Session #11 · 2026-04-21（S3 · 07 leaves 全 3 接口）

- **Completed**:
  - `LeaveRepo`：列表 JOIN（学生/班级/申请人/`student_parents` 关系、审批人）、区间重叠筛选、统计（按状态/班级/原因聚合）
  - `LeaveService` + `handler/http/leave.go`：`GET /admin/leaves`、`GET /admin/leaves/stats`（未传日期默认当月 Asia/Shanghai）、`PUT /admin/leaves/:id/audit`；审批写 `operation_logs`
  - `admin/api/leaves.ts` 增加 `auditLeave`；`admin/pages/leaves/index.vue` 待审批行「同意/驳回」
  - `workbooks/07-leaves.md`、`samples/07-leaves/leaves.http`；`07-leaves.md` 补充 PUT 审批规格
  - `PROGRESS.md` / `README.md`；`Handler` / `wire_gen.go` 注入 `LeaveService`
- **Decisions**:
  - `by_reason` 按关键词归入「生病 / 家事 / 其他」；与 MD 示例一致
  - 审批请求体：`status` 1/2，`review_remark` 可选
- **Next Action**: `GET /admin/pickup/overview`（08 pickup）
- **Blockers**: 无
- **Last Modified Files**:
  - `server/internal/repository/leave_repo.go` (new)
  - `server/internal/service/leave_service.go` (new)
  - `server/internal/handler/http/leave.go` (new)
  - `server/internal/handler/http/handler.go`
  - `server/internal/repository/wire.go`
  - `server/internal/service/wire.go`
  - `server/internal/app/router.go`
  - `server/internal/app/wire_gen.go`
  - `admin/api/leaves.ts`
  - `admin/pages/leaves/index.vue`
  - `backend/api/api-admin/07-leaves.md`
  - `backend/api/api-admin/workbooks/07-leaves.md` (new)
  - `backend/api/api-admin/samples/07-leaves/leaves.http` (new)
  - `backend/api/api-admin/PROGRESS.md`
  - `backend/api/api-admin/README.md`
  - `backend/api/api-admin/SESSION_HANDOFF.md`

### Session #10 · 2026-04-21（S3 · 06 attendance 全 3 接口）

- **Completed**:
  - `AttendanceRepo`（按班聚合、明细 JOIN、导出范围查询；`deleted_at` 过滤）
  - `AttendanceService` + `Asia/Shanghai` 日界；概览/分页明细/xlsx 导出（excelize）；`DashboardRepo.CountActiveStudents` + `ClassRepo` 班级在读人数
  - `handler/http/attendance.go`、`/admin/attendance/overview|records|report`；`Handler` / `wire_gen.go` 注入；报表 Blob 响应头与学员导出一致
  - `workbooks/06-attendance.md`、`samples/06-attendance/attendance.http`；`PROGRESS` / `README` / `06-attendance.md`
- **Decisions**:
  - 全园 `absent` = `total_students - present - leave`（与班级行一致）；迟到计入出勤与 `present` 筛选
  - 导出须 `start_date`+`end_date`；`pickup_time` 暂无库字段，返回空串
- **Next Action**: `GET /admin/leaves`（07 leaves）
- **Blockers**: 无
- **Last Modified Files**:
  - `server/internal/repository/attendance_repo.go` (new)
  - `server/internal/service/attendance_service.go` (new)
  - `server/internal/handler/http/attendance.go` (new)
  - `server/internal/handler/http/handler.go`
  - `server/internal/repository/wire.go`
  - `server/internal/service/wire.go`
  - `server/internal/app/router.go`
  - `server/internal/app/wire_gen.go`
  - `backend/api/api-admin/workbooks/06-attendance.md` (new)
  - `backend/api/api-admin/samples/06-attendance/attendance.http` (new)
  - `backend/api/api-admin/PROGRESS.md`
  - `backend/api/api-admin/README.md`
  - `backend/api/api-admin/06-attendance.md`
  - `backend/api/api-admin/SESSION_HANDOFF.md`

### Session #9 · 2026-04-21（S2 · 17 roles 全 5 接口）

- **Completed**:
  - `RoleService` + `role_catalog.go`（与 `17-roles.md` 一致的权限树；种子 `admin_permissions` 含 `*`）
  - `AdminRepo` 扩展：角色列表/计数、权限码、按码取 ID、替换关联（`Unscoped` 物理删关联行）、按名校验、软删角色
  - `handler/http/role.go`：`/admin/roles` CRUD + `GET /admin/permissions`；`Handler` / `wire_gen.go` 注入 `RoleService`
  - 「超级管理员」无权限行时列表返回 `permissions: ["*"]`；系统内置角色不可删、不可改名
  - `workbooks/17-roles.md`、`samples/17-roles/roles.http`；`PROGRESS` / `README` / `17-roles.md` 状态
- **Decisions**:
  - Casbin 仍为 admin 全放行；权限码主要为前端配置与后续细粒度校验预留
- **Next Action**: `GET /admin/attendance/overview`（06 attendance）
- **Blockers**: 无
- **Last Modified Files**:
  - `server/internal/service/role_catalog.go` (new)
  - `server/internal/service/role_service.go` (new)
  - `server/internal/service/wire.go`
  - `server/internal/repository/admin_repo.go`
  - `server/internal/handler/http/role.go` (new)
  - `server/internal/handler/http/handler.go`
  - `server/internal/app/router.go`
  - `server/internal/app/wire_gen.go`
  - `backend/api/api-admin/workbooks/17-roles.md` (new)
  - `backend/api/api-admin/samples/17-roles/roles.http` (new)
  - `backend/api/api-admin/PROGRESS.md`
  - `backend/api/api-admin/README.md`
  - `backend/api/api-admin/17-roles.md`
  - `backend/api/api-admin/SESSION_HANDOFF.md`

### Session #8 · 2026-04-21（S2 · 16 admins 全 5 接口）

- **Completed**:
  - `AdminRepo`（列表 JOIN、`ResolveAdminRoleID`、园所无角色时自动建「园所管理员」）、`UserRoleRepo` 扩展（管理员查询/计数/删除）
  - `AdminService` + `handler/http/admin.go`；`/admin/admins` CRUD + `PUT .../reset-password`
  - `system_service.Initialize` 增加首条 `admin_roles` + `user_admin_roles` 与首管理员绑定（新装库一致）
  - 列表分页 `OKList`、手机脱敏、`operation_logs`；删前校验「至少一名管理员」「不可删自己」
  - `workbooks/16-admins.md`、`samples/16-admins/admins.http`；`admin/pages/admins/index.vue` 默认 `role_id: 1`；`PROGRESS` / `README` / `16-admins.md`
  - `wire_gen.go` 手改注入 `AdminService` / `AdminRepo`
- **Decisions**:
  - `last_login_at` 暂无表字段，列表不返回该键（`omitempty`）
  - 已初始化但历史无 `admin_roles` 的园所：首个有效 `POST` 在 `role_id` 无法匹配且园所角色数为 0 时自动建默认角色再绑定
- **Next Action**: `GET /admin/roles`（17 roles）
- **Blockers**: 无
- **Last Modified Files**:
  - `server/internal/repository/admin_repo.go` (new)
  - `server/internal/repository/user_role_repo.go`
  - `server/internal/repository/wire.go`
  - `server/internal/service/admin_service.go` (new)
  - `server/internal/service/system_service.go`
  - `server/internal/service/wire.go`
  - `server/internal/handler/http/admin.go` (new)
  - `server/internal/handler/http/handler.go`
  - `server/internal/app/router.go`
  - `server/internal/app/wire_gen.go`
  - `admin/pages/admins/index.vue`
  - `backend/api/api-admin/workbooks/16-admins.md` (new)
  - `backend/api/api-admin/samples/16-admins/admins.http` (new)
  - `backend/api/api-admin/PROGRESS.md`
  - `backend/api/api-admin/README.md`
  - `backend/api/api-admin/16-admins.md`
  - `backend/api/api-admin/SESSION_HANDOFF.md`

### Session #7 · 2026-04-21（S2 · 18 settings 全 6 接口）

- **Completed**:
  - 扩展 `Kindergarten` 档案字段；新表 `academic_years`、`kindergarten_app_settings`（JSON 存出勤/通知/成长/接送/积分）；`AutoMigrate` 已注册
  - `AcademicYearRepo`、`KindergartenAppSettingRepo`；`SettingsService`（`GetSettings`/`UpdateSettings`/学年 CRUD/园所信息）；`handler/http/settings.go`；`/admin/settings/*`、`/admin/kindergarten/info` 路由
  - `basic` 学年块来自学年表 + `current_semester`；首个学年 `POST` 自动 `is_current`；写操作 `operation_logs`
  - `workbooks/18-settings.md`、`samples/18-settings/settings.http`；`PROGRESS` / `README` / `18-settings.md` 状态
  - `wire_gen.go` 手改注入 `SettingsService` 与两个新 Repo
- **Decisions**:
  - 切换「当前学年」的专用 API 未在规格中：仅首个创建学年标为 current；后续学年 `is_current=false`
  - `semester`（spring/fall）持久化在 `kindergarten_app_settings`，当前无单独接口修改（默认 `spring`）
- **Next Action**: `GET /admin/admins`（16 admins）
- **Blockers**: 无
- **Last Modified Files**:
  - `server/internal/domain/kindergarten.go`
  - `server/internal/domain/academic_year.go` (new)
  - `server/internal/domain/kindergarten_app_setting.go` (new)
  - `server/internal/repository/academic_year_repo.go` (new)
  - `server/internal/repository/kindergarten_app_setting_repo.go` (new)
  - `server/internal/repository/kindergarten_repo.go`
  - `server/internal/repository/class_repo.go`
  - `server/internal/repository/student_repo.go`
  - `server/internal/repository/user_role_repo.go`
  - `server/internal/repository/wire.go`
  - `server/internal/service/settings_service.go` (new)
  - `server/internal/service/wire.go`
  - `server/internal/handler/http/settings.go` (new)
  - `server/internal/handler/http/handler.go`
  - `server/internal/app/router.go`
  - `server/internal/app/provider.go`
  - `server/internal/app/wire_gen.go`
  - `backend/api/api-admin/workbooks/18-settings.md` (new)
  - `backend/api/api-admin/samples/18-settings/settings.http` (new)
  - `backend/api/api-admin/PROGRESS.md`
  - `backend/api/api-admin/README.md`
  - `backend/api/api-admin/18-settings.md`
  - `backend/api/api-admin/SESSION_HANDOFF.md`

### Session #6 · 2026-04-21（S2 · 05 parents 全 5 接口）

- **Completed**:
  - 新增 `ParentProfile` + `parent_profiles` AutoMigrate；`ParentRepo`、`ParentService`、`handler/http/parent.go`；`/admin/parents/*` 路由
  - 家长：`users` + `user_roles(role=parent)` + `parent_profiles` + `student_parents`；列表/详情/写操作均 `kindergarten_id` 过滤；写操作 `operation_logs`
  - `UserRoleRepo.DeleteParentRole`；默认新家长密码 **`KindyParent#1`**（与 `KindyTeacher#1` 对称）
  - `workbooks/05-parents.md`、`samples/05-parents/parents.http`；`PROGRESS` / `README` / `05-parents.md` 状态
  - `wire_gen.go` 手改注入（本会话 `go generate` 因 goproxy 502 未成功；网络恢复后请补跑 `go mod tidy` + `go generate ./internal/app` 校验）
- **Decisions**:
  - 管理接口家长主键为 **`users.id`**；`last_login_at` 依赖 `parent_profiles.last_login_at`（当前多为空字符串，后续可在家长登录流程写入）
  - 历史仅有 `user_roles` 无 `parent_profiles` 的数据不会出现在列表（INNER JOIN）；需数据时走本模块创建或写迁移补全
- **Next Action**: `GET /admin/settings`（18 settings）
- **Blockers**: 无
- **Last Modified Files**:
  - `server/internal/domain/parent.go` (new)
  - `server/internal/repository/parent_repo.go` (new)
  - `server/internal/repository/user_role_repo.go`
  - `server/internal/repository/wire.go`
  - `server/internal/service/parent_service.go` (new)
  - `server/internal/service/wire.go`
  - `server/internal/handler/http/parent.go` (new)
  - `server/internal/handler/http/handler.go`
  - `server/internal/app/router.go`
  - `server/internal/app/provider.go`
  - `server/internal/app/wire_gen.go`
  - `backend/api/api-admin/workbooks/05-parents.md` (new)
  - `backend/api/api-admin/samples/05-parents/parents.http` (new)
  - `backend/api/api-admin/PROGRESS.md`
  - `backend/api/api-admin/README.md`
  - `backend/api/api-admin/05-parents.md`
  - `backend/api/api-admin/SESSION_HANDOFF.md`

### Session #5 · 2026-04-20（S2 · 04 teachers 全 6 接口 + 文档同步 03）

- **Completed**:
  - `TeacherProfile` 域模型 + `AutoMigrate`；`TeacherRepo`、`TeacherService`、`handler/http/teacher.go`；`/admin/teachers/*` 路由与 Wire
  - 老师以 `users` + `user_roles(role=teacher)` + `teacher_profiles` + `class_teachers` 表达；列表/详情 `kindergarten_id` 过滤；写操作 `operation_logs`
  - `UserRoleRepo.FindTeacherInKindergarten` / `DeleteTeacherRole`；修正 `FindByUserIDAndRole` 使用 `errors.Is`
  - `README` / `PROGRESS` / `04-teachers.md` 状态；模块 03 README 标 ✅（与已实现代码对齐）
- **Decisions**:
  - 管理接口中的老师主键为 **`users.id`**（与班级里 `teacher_ids[].user_id` 一致）
  - 详情 `stats.avg_parent_satisfaction` 暂无数据源，固定返回 **0**
- **Next Action**: `GET /admin/parents`（05 parents）
- **Blockers**: 无
- **Last Modified Files**:
  - `server/internal/domain/teacher.go`
  - `server/internal/repository/teacher_repo.go` (new)
  - `server/internal/repository/user_role_repo.go`
  - `server/internal/repository/wire.go`
  - `server/internal/service/teacher_service.go` (new)
  - `server/internal/service/wire.go`
  - `server/internal/handler/http/teacher.go` (new)
  - `server/internal/handler/http/handler.go`
  - `server/internal/app/router.go`
  - `server/internal/app/wire_gen.go`
  - `server/internal/app/provider.go`
  - `backend/api/api-admin/PROGRESS.md`
  - `backend/api/api-admin/README.md`
  - `backend/api/api-admin/04-teachers.md`
  - `backend/api/api-admin/SESSION_HANDOFF.md`

### Session #4 · 2026-04-20（S1 · 02 classes 全 6 接口）

- **Completed**:
  - `domain.Class` 扩展字段；`ClassRepo`、`OperationLogRepo`、`ClassService`、`handler/http/class.go`
  - `GET/POST /admin/classes`、`GET/PUT/DELETE /admin/classes/:id`、`POST /admin/classes/:id/adjust`
  - 写操作写 `operation_logs`；删除仅空班；调班校验学生归属
  - `workbooks/02-classes.md`、`samples/02-classes/list.http`；`PROGRESS` / `README` 模块 02 ✅
- **Decisions**:
  - `move_out`：body `from_class_id` 表示**目标班级**（学生从 URL `:id` 班转出到该班）
- **Next Action**: `GET /admin/students`（03 students）
- **Blockers**: 无
- **Last Modified Files**:
  - `server/internal/domain/kindergarten.go`
  - `server/internal/repository/class_repo.go` (new)
  - `server/internal/repository/operation_log_repo.go` (new)
  - `server/internal/repository/wire.go`
  - `server/internal/service/class_service.go` (new)
  - `server/internal/service/wire.go`
  - `server/internal/handler/http/class.go` (new)
  - `server/internal/handler/http/handler.go`
  - `server/internal/app/router.go`
  - `server/internal/app/wire_gen.go`
  - `backend/api/api-admin/workbooks/02-classes.md` (new)
  - `backend/api/api-admin/samples/02-classes/list.http` (new)
  - `backend/api/api-admin/PROGRESS.md`
  - `backend/api/api-admin/README.md`
  - `backend/api/api-admin/SESSION_HANDOFF.md`

### Session #3 · 2026-04-20（S1 · 01 dashboard 全 4 接口）

- **Completed**:
  - `GET /admin/dashboard/overview|attendance-trend|class-ranking|growth-stats`
  - `KindergartenRepo.FindByID`、`DashboardRepo`、`DashboardService`、`handler/http/dashboard.go`、`registerAdminRoutes` 挂载 `/dashboard/*`
  - `samples/01-dashboard/overview.http`；`workbooks/01-dashboard.md` 结账
  - `PROGRESS.md` / `README.md` 模块 01 标 ✅
- **Decisions**:
  - 统计口径以 **Asia/Shanghai** 自然日/月；`new_notices` 用当日 `notices.created_at`
  - 班级排行按 **present/total** 降序
- **Next Action**: `GET /admin/classes`（02 classes 模块）
- **Blockers**: 无
- **Last Modified Files**:
  - `server/internal/repository/kindergarten_repo.go`
  - `server/internal/repository/dashboard_repo.go` (new)
  - `server/internal/repository/wire.go`
  - `server/internal/service/dashboard_service.go` (new)
  - `server/internal/service/wire.go`
  - `server/internal/handler/http/dashboard.go` (new)
  - `server/internal/handler/http/handler.go`
  - `server/internal/app/router.go`
  - `server/internal/app/wire_gen.go`
  - `backend/api/api-admin/workbooks/01-dashboard.md` (new)
  - `backend/api/api-admin/samples/01-dashboard/overview.http` (new)
  - `backend/api/api-admin/PROGRESS.md`
  - `backend/api/api-admin/README.md`
  - `backend/api/api-admin/SESSION_HANDOFF.md`

### Session #2 · 2026-04-20（S0 脚手架收尾：路由组 + 核对 + REST 模板）

- **Completed**:
  - S0-2：`registerAdminRoutes` + `authed.Group("/admin")` 挂载于 `server/internal/app/router.go`
  - S0-4：核对 `repository` / `service` / `handler/http` 的 `wire.NewSet` 导出
  - S0-5：`Handler` 结构体上增加聚合器扩展约定注释（`handler.go`）
  - S0-6：核对 `ProvideGorm` 内 `AutoMigrate` 实体列表（`provider.go`），修正接力卡中 migration 文件路径说明
  - S0-8：新增 `backend/api/api-admin/samples/_login.http`
- **Decisions**:
  - AutoMigrate 真源为 `internal/app/provider.go`（与旧文档写的 `system_service` 不一致处已以代码为准）
  - 本会话未改 Wire 注入图；本机首次 `go generate ./internal/app` 若报 wire 工具链缺 `go.sum`，在项目根执行 `go mod tidy` 后再 `go generate ./internal/app`
- **Next Action**: S1 `GET /admin/dashboard/overview` 或可选 S0-7 seed
- **Blockers**: 无
- **Last Modified Files**:
  - `server/internal/app/router.go`
  - `server/internal/handler/http/handler.go`
  - `backend/api/api-admin/samples/_login.http` (new)
  - `backend/api/api-admin/PROGRESS.md`
  - `backend/api/api-admin/SESSION_HANDOFF.md`

### Session #1 · 2026-04-20（方案 B：JWT `kindergarten_id` + S0 部分）

- **Completed**:
  - JWT：`TenantID` 移除，`AccessClaims.KindergartenID uint64`，`json:"kindergarten_id,omitempty"`；`SignAccess` 增加参数并在登录/刷新时写入
  - `AuthService`：`issueTokens` / `issueTokensFromRoles`，按 `user_roles` 推导园所 ID（优先 `admin` 角色的 `kindergarten_id`）
  - `middleware/context.go`：`GetUserID` / `GetKindergartenID` / `GetRoles`
  - `pkg/pagination/pagination.go`（S0-1）
- **Decisions**:
  - **方案 B**：不再使用 JWT 字段 `tid`，统一用声明式 `kindergarten_id`（与 DB / API 一致）；旧 token 需重新登录才含园所 ID
- **Next Action**: S0-2 `registerAdminRoutes`
- **Blockers**: 无
- **Last Modified Files**:
  - `server/internal/auth/jwt.go`
  - `server/internal/service/auth_service.go`
  - `server/internal/middleware/context.go`
  - `server/pkg/pagination/pagination.go`
  - `backend/api/api-admin/PROGRESS.md`
  - `backend/api/api-admin/SESSION_HANDOFF.md`

---

### Session #0 · 2026-04-20（计划启动）

- **Completed**:
  - 创建 `EXECUTION_PLAN.md`
  - 创建 `PROGRESS.md`
  - 创建 `SESSION_HANDOFF.md`
- **Decisions**:
  - Casbin 策略对 admin 全放行（已有 `p, admin, /api/v1/*, *`），不需要每模块加策略
  - 按 Sprint 串行，不并行
  - 每个模块严格按 CRUD 顺序
  - 前端 `admin/api/*.ts` 已就绪，除 5 个待补函数外不主动改前端
- **Next Action**: S0-1 新建 `server/pkg/pagination/pagination.go`
- **Blockers**: 无
- **Last Modified Files**:
  - `backend/api/api-admin/EXECUTION_PLAN.md` (new)
  - `backend/api/api-admin/PROGRESS.md` (new)
  - `backend/api/api-admin/SESSION_HANDOFF.md` (new)

---

## 🚧 Blockers（需要用户决策 / 暂时卡住）

当前无 blocker。

> 如出现 blocker，格式：
> - **[YYYY-MM-DD] 标题**
>   - 背景：...
>   - 可选方案 A / B / C
>   - 需要用户选择

---

## 🔑 关键上下文（长期记忆，新会话必读）

1. **后端栈**：Go + Gin + GORM + Wire + Casbin + Redis + OSS，入口 `server/cmd/api/main.go`。
2. **响应封装**：统一用 `server/pkg/response/response.go` 的 `OK / OKList / Created / ErrXxx`。
3. **分页**：Query `page / page_size / keyword`；Response `{list, total, page, page_size}`。
4. **多租户**：所有业务接口在 Service 层用 `kindergarten_id` 兜底过滤。
5. **DB migration**：GORM AutoMigrate，新字段直接加 domain 即可；**新表** 要在 `server/internal/app/provider.go` 的 `ProvideGorm` → `AutoMigrate` 列表里追加。
6. **Wire**：改了 `*/wire.go` 的 `Set` 后必须 `go generate ./internal/app`。
7. **测试账号**：`13800138000 / Admin@123`（来自 `BACKEND_MISSING_APIS.md`）。
8. **前端启动**：`admin/` 下 `npm run dev`（默认 3000 端口）。
9. **后端启动**：`cd server && go run cmd/api/main.go`（默认 8080 端口）。

---

## 📎 文档间关系

```
KICKOFF_PROMPT.md     — 会话入口（提示词模板，用户复制粘贴）
       ↓
EXECUTION_PLAN.md     — 方法论 & 约定（长期稳定）
       ↓
PROGRESS.md           — 87 个接口进度表（每次打勾）
       ↓
SESSION_HANDOFF.md    — 全局会话接力卡（每次会话开始/结束更新）
       ↓
workbooks/<XX>-*.md   — 模块工作簿（每模块一份，会话级状态）
       ↓
01~20-*.md            — 接口规格（只读，真源）
samples/*.http        — REST Client 样例（实现时创建）
```

## 🔁 新会话启动方式（给用户）

打开新 Chat，复制下面任意一句：

```
请按 backend/api/api-admin/KICKOFF_PROMPT.md 的 "自动续接流程" 继续执行 admin 接口计划。
```

或指定模块：

```
请按 backend/api/api-admin/KICKOFF_PROMPT.md 开始做 02 classes 模块（只做这一个模块，完成后结账）。
```
