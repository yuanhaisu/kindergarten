# Admin 端接口实现 · 进度表

> **规则**：每完成 1 个接口，把该行 `[ ]` 改成 `[x]`，在"完成日期"列写 `YYYY-MM-DD`，可选写 commit hash。
>
> **模块状态**：🚫 未开始 / 🔄 进行中 / ✅ 已完成（模块所有接口都勾完 + 前端联调通过）
>
> **总体进度**：`95 / 95`

---

## Sprint 总览

| Sprint | 模块 | 进度 | 状态 |
|---|---|---:|---|
| **S0** | 脚手架 | 7 / 8 | 🔄 |
| **S1** | P0：Dashboard + Classes + Students | 18 / 18 | 🔄 |
| **S2** | P1：Teachers + Parents + Settings + Admins + Roles | 27 / 27 | ✅ |
| **S3** | P2a：Attendance + Leaves + Pickup | 9 / 9 | ✅ |
| **S4** | P2b：Growth + Tasks + Points + Achievements | 18 / 18 | ✅ |
| **S5** | P3：Notices + Announcements + Audit + Logs + Export | 20 / 20 | ✅ |
| **S6** | 回归测试 | 3 / 4 | 🔄 |

---

## S0 · 脚手架（8 项，先于所有业务模块）

| # | 任务 | 完成 | 完成日期 | 备注 |
|---|---|:---:|---|---|
| S0-1 | 新建 `server/pkg/pagination/pagination.go` | [x] | 2026-04-20 | `Query{Page,PageSize,Keyword}` + `Normalize()` + `Offset()` |
| S0-2 | 抽 `registerAdminRoutes` 到 `router.go` | [x] | 2026-04-20 | `authed.Group("/admin")` + 空壳注册函数 |
| S0-3 | 新增 `middleware/context.go` claims helper | [x] | 2026-04-20 | `GetUserID / GetKindergartenID / GetRoles`；JWT 已含 `kindergarten_id` |
| S0-4 | 核对 3 个 wire Set 正常导出 | [x] | 2026-04-20 | `repository/service/handler/http` 均导出 `Set` |
| S0-5 | Handler 聚合器字段扩展约定 | [x] | 2026-04-20 | 约定写在 `handler.go` 结构体注释 |
| S0-6 | 核对 AutoMigrate 注册表 | [x] | 2026-04-20 | 列表在 `internal/app/provider.go` `ProvideGorm`；新表追加此处 |
| S0-7 | 本地 seed 数据（可选） | [ ] | | 1 园所 + 3 班 + 10 学生 + 1 admin |
| S0-8 | REST Client 样例模板 `samples/_login.http` | [x] | 2026-04-20 | 密码登录 + Bearer 示例 |

---

## S1 · P0 模块（18 个接口）

### 01. Dashboard `/admin/dashboard/*`（4 个）

对应文档：[`01-dashboard.md`](./01-dashboard.md)　前端 API：`admin/api/dashboard.ts`　页面：`admin/pages/index.vue`　状态：✅

| # | 方法 | 路径 | 前端函数 | 完成 | 完成日期 | commit |
|---|---|---|---|:---:|---|---|
| 1 | GET | `/admin/dashboard/overview` | `getDashboardOverview` | [x] | 2026-04-20 | |
| 2 | GET | `/admin/dashboard/attendance-trend` | `getAttendanceTrend` | [x] | 2026-04-20 | |
| 3 | GET | `/admin/dashboard/class-ranking` | `getClassRanking` | [x] | 2026-04-20 | |
| 4 | GET | `/admin/dashboard/growth-stats` | `getGrowthStats` | [x] | 2026-04-20 | 与后续 `/admin/growth/stats` 可复用 `DashboardService.GrowthStats` |

### 02. Classes `/admin/classes/*`（6 个）

对应文档：[`02-classes.md`](./02-classes.md)　前端 API：`admin/api/classes.ts`　页面：`admin/pages/classes/index.vue`　状态：✅

| # | 方法 | 路径 | 前端函数 | 完成 | 完成日期 | commit |
|---|---|---|---|:---:|---|---|
| 1 | GET | `/admin/classes` | `getClasses` | [x] | 2026-04-20 | |
| 2 | GET | `/admin/classes/:id` | `getClassDetail` | [x] | 2026-04-20 | |
| 3 | POST | `/admin/classes` | `createClass` | [x] | 2026-04-20 | |
| 4 | PUT | `/admin/classes/:id` | `updateClass` | [x] | 2026-04-20 | |
| 5 | DELETE | `/admin/classes/:id` | `deleteClass` | [x] | 2026-04-20 | 仅空班级可删 |
| 6 | POST | `/admin/classes/:id/adjust` | `adjustClassStudents` | [x] | 2026-04-20 | move_out 时 `from_class_id` 为目标班 |

### 03. Students `/admin/students/*`（8 个）

对应文档：[`03-students.md`](./03-students.md)　前端 API：`admin/api/students.ts`　页面：`admin/pages/students/index.vue`　状态：✅

| # | 方法 | 路径 | 前端函数 | 完成 | 完成日期 | commit |
|---|---|---|---|:---:|---|---|
| 1 | GET | `/admin/students` | `getStudents` | [x] | 2026-04-20 | |
| 2 | GET | `/admin/students/:id` | `getStudent` | [x] | 2026-04-20 | |
| 3 | POST | `/admin/students` | `createStudent` | [x] | 2026-04-20 | |
| 4 | PUT | `/admin/students/:id` | `updateStudent` | [x] | 2026-04-20 | |
| 5 | DELETE | `/admin/students/:id` | `deleteStudent` | [x] | 2026-04-20 | |
| 6 | PUT | `/admin/students/:id/status` | `updateStudentStatus` | [x] | 2026-04-20 | 在读/毕业/退学 |
| 7 | POST | `/admin/students/import` | `importStudents` | [x] | 2026-04-20 | Excel 导入 |
| 8 | GET | `/admin/students/export` | `exportStudentsBlob` | [x] | 2026-04-20 | Blob，不走 Envelope |

---

## S2 · P1 模块（27 个接口）

### 04. Teachers `/admin/teachers/*`（6 个）

对应文档：[`04-teachers.md`](./04-teachers.md)　前端 API：`admin/api/teachers.ts`　页面：`admin/pages/teachers/index.vue`　状态：✅

| # | 方法 | 路径 | 前端函数 | 完成 | 完成日期 | commit |
|---|---|---|---|:---:|---|---|
| 1 | GET | `/admin/teachers` | `getTeachers` | [x] | 2026-04-20 | |
| 2 | GET | `/admin/teachers/:id` | `getTeacher` | [x] | 2026-04-20 | |
| 3 | POST | `/admin/teachers` | `createTeacher` | [x] | 2026-04-20 | |
| 4 | PUT | `/admin/teachers/:id` | `updateTeacher` | [x] | 2026-04-20 | |
| 5 | DELETE | `/admin/teachers/:id` | `deleteTeacher` | [x] | 2026-04-20 | |
| 6 | PUT | `/admin/teachers/:id/status` | `updateTeacherStatus` | [x] | 2026-04-20 | |

### 05. Parents `/admin/parents/*`（5 个）

对应文档：[`05-parents.md`](./05-parents.md)　前端 API：`admin/api/parents.ts`　页面：`admin/pages/parents/index.vue`　状态：✅

| # | 方法 | 路径 | 前端函数 | 完成 | 完成日期 | commit |
|---|---|---|---|:---:|---|---|
| 1 | GET | `/admin/parents` | `getParents` | [x] | 2026-04-21 | |
| 2 | GET | `/admin/parents/:id` | `getParent` | [x] | 2026-04-21 | |
| 3 | POST | `/admin/parents` | `createParent` | [x] | 2026-04-21 | 新用户默认密码 `KindyParent#1` |
| 4 | PUT | `/admin/parents/:id` | `updateParent` | [x] | 2026-04-21 | |
| 5 | PUT | `/admin/parents/:id/status` | `updateParentStatus` | [x] | 2026-04-21 | |

### 18. Settings `/admin/settings/*` + `/admin/kindergarten/info`（6 个）

对应文档：[`18-settings.md`](./18-settings.md)　前端 API：`admin/api/settings.ts` + `admin/api/kindergarten.ts`　页面：`admin/pages/settings/index.vue`　状态：✅

| # | 方法 | 路径 | 前端函数 | 完成 | 完成日期 | commit |
|---|---|---|---|:---:|---|---|
| 1 | GET | `/admin/settings` | `getSettings` | [x] | 2026-04-21 | |
| 2 | PUT | `/admin/settings` | `updateSettings` | [x] | 2026-04-21 | |
| 3 | GET | `/admin/settings/academic-years` | `getAcademicYears` | [x] | 2026-04-21 | |
| 4 | POST | `/admin/settings/academic-years` | `createAcademicYear` | [x] | 2026-04-21 | |
| 5 | GET | `/admin/kindergarten/info` | `getKindergartenInfo` | [x] | 2026-04-21 | |
| 6 | PUT | `/admin/kindergarten/info` | `updateKindergartenInfo` | [x] | 2026-04-21 | |

### 16. Admins `/admin/admins/*`（5 个）

对应文档：[`16-admins.md`](./16-admins.md)　前端 API：`admin/api/admins.ts`　页面：`admin/pages/admins/index.vue`　状态：✅

| # | 方法 | 路径 | 前端函数 | 完成 | 完成日期 | commit |
|---|---|---|---|:---:|---|---|
| 1 | GET | `/admin/admins` | `getAdmins` | [x] | 2026-04-21 | |
| 2 | POST | `/admin/admins` | `createAdmin` | [x] | 2026-04-21 | |
| 3 | PUT | `/admin/admins/:id` | `updateAdmin` | [x] | 2026-04-21 | |
| 4 | DELETE | `/admin/admins/:id` | `deleteAdmin` | [x] | 2026-04-21 | |
| 5 | PUT | `/admin/admins/:id/reset-password` | `resetAdminPassword` | [x] | 2026-04-21 | |

### 17. Roles `/admin/roles/*` + `/admin/permissions`（5 个）

对应文档：[`17-roles.md`](./17-roles.md)　前端 API：`admin/api/roles.ts`　页面：`admin/pages/roles/index.vue`　状态：✅

| # | 方法 | 路径 | 前端函数 | 完成 | 完成日期 | commit |
|---|---|---|---|:---:|---|---|
| 1 | GET | `/admin/roles` | `getRoles` | [x] | 2026-04-21 | |
| 2 | POST | `/admin/roles` | `createRole` | [x] | 2026-04-21 | |
| 3 | PUT | `/admin/roles/:id` | `updateRole` | [x] | 2026-04-21 | |
| 4 | DELETE | `/admin/roles/:id` | `deleteRole` | [x] | 2026-04-21 | |
| 5 | GET | `/admin/permissions` | `getPermissionTree` | [x] | 2026-04-21 | |

---

## S3 · P2a 模块（9 个接口）

### 06. Attendance `/admin/attendance/*`（3 个）

对应文档：[`06-attendance.md`](./06-attendance.md)　前端 API：`admin/api/attendance.ts`　页面：`admin/pages/attendance/index.vue`　状态：✅

| # | 方法 | 路径 | 前端函数 | 完成 | 完成日期 | commit |
|---|---|---|---|:---:|---|---|
| 1 | GET | `/admin/attendance/overview` | `getAttendanceOverview` | [x] | 2026-04-21 | |
| 2 | GET | `/admin/attendance/records` | `getAttendanceRecords` | [x] | 2026-04-21 | |
| 3 | GET | `/admin/attendance/report` | `exportAttendanceReport` | [x] | 2026-04-21 | Blob |

### 07. Leaves `/admin/leaves/*`（2 个 + 1 个前后端补齐）

对应文档：[`07-leaves.md`](./07-leaves.md)　前端 API：`admin/api/leaves.ts`　页面：`admin/pages/leaves/index.vue`　状态：✅

| # | 方法 | 路径 | 前端函数 | 完成 | 完成日期 | commit |
|---|---|---|---|:---:|---|---|
| 1 | GET | `/admin/leaves` | `getLeaves` | [x] | 2026-04-21 | |
| 2 | GET | `/admin/leaves/stats` | `getLeaveStats` | [x] | 2026-04-21 | |
| + | PUT | `/admin/leaves/:id/audit` | `auditLeave` | [x] | 2026-04-21 | |

### 08. Pickup `/admin/pickup/*`（4 个）

对应文档：[`08-pickup.md`](./08-pickup.md)　前端 API：`admin/api/pickup.ts`　页面：`admin/pages/pickup/index.vue`　状态：✅

| # | 方法 | 路径 | 前端函数 | 完成 | 完成日期 | commit |
|---|---|---|---|:---:|---|---|
| 1 | GET | `/admin/pickup/overview` | `getPickupOverview` | [x] | 2026-04-21 | |
| 2 | GET | `/admin/pickup/records` | `getPickupRecords` | [x] | 2026-04-21 | |
| 3 | GET | `/admin/pickup/abnormal` | `getPickupAbnormal` | [x] | 2026-04-21 | |
| 4 | PUT | `/admin/pickup/authorized-persons/:id/audit` | `auditPickupAuthorizedPerson` | [x] | 2026-04-21 | 授权表 `review_status` 0 待审 |

---

## S4 · P2b 模块（18 个接口）

### 09. Growth `/admin/growth/*`（5 个）

对应文档：[`09-growth.md`](./09-growth.md)　前端 API：`admin/api/growth.ts`　页面：`admin/pages/growth/index.vue`　状态：✅

| # | 方法 | 路径 | 前端函数 | 完成 | 完成日期 | commit |
|---|---|---|---|:---:|---|---|
| 1 | GET | `/admin/growth/logs` | `getGrowthLogs` | [x] | 2026-04-21 | |
| 2 | GET | `/admin/growth/logs/:id` | `getGrowthLog` | [x] | 2026-04-21 | |
| 3 | DELETE | `/admin/growth/logs/:id` | `deleteGrowthLog` | [x] | 2026-04-21 | 软删；写 operation_logs |
| 4 | PUT | `/admin/growth/logs/:id/audit` | `auditGrowthLog` | [x] | 2026-04-21 | `content_audits`；写 operation_logs |
| 5 | GET | `/admin/growth/stats` | `getGrowthStats` | [x] | 2026-04-21 | 与 dashboard 分接口；日期须成对或同时省略 |

### 10. Tasks `/admin/tasks/*`（6 个）

对应文档：[`10-tasks.md`](./10-tasks.md)　前端 API：`admin/api/tasks.ts`　页面：`admin/pages/tasks/index.vue`　状态：✅

| # | 方法 | 路径 | 前端函数 | 完成 | 完成日期 | commit |
|---|---|---|---|:---:|---|---|
| 1 | GET | `/admin/tasks` | `getTasks` | [x] | 2026-04-21 | |
| 2 | GET | `/admin/tasks/:id` | `getTask` | [x] | 2026-04-21 | |
| 3 | DELETE | `/admin/tasks/:id` | `deleteTask` | [x] | 2026-04-21 | 写 operation_logs |
| 4 | GET | `/admin/tasks/stats` | `getTaskStats` | [x] | 2026-04-21 | 日期须成对或同时省略 |
| 5 | POST | `/admin/tasks` | `createTask` | [x] | 2026-04-21 | HTTP 201 |
| 6 | PUT | `/admin/tasks/:id` | `updateTask` | [x] | 2026-04-21 | |

### 11. Points `/admin/points/*`（5 个）

对应文档：[`11-points.md`](./11-points.md)　前端 API：`admin/api/points.ts`　页面：`admin/pages/points/index.vue`　状态：✅

| # | 方法 | 路径 | 前端函数 | 完成 | 完成日期 | commit |
|---|---|---|---|:---:|---|---|
| 1 | GET | `/admin/points/overview` | `getPointsOverview` | [x] | 2026-04-21 | |
| 2 | GET | `/admin/points/records` | `getPointRecords` | [x] | 2026-04-21 | |
| 3 | POST | `/admin/points/adjust` | `adjustPoints` | [x] | 2026-04-21 | |
| 4 | GET | `/admin/points/rules` | `getPointRules` | [x] | 2026-04-21 | |
| 5 | PUT | `/admin/points/rules` | `updatePointRules` | [x] | 2026-04-21 | `extra_config.points.*`；规则 id 4 → `growth_featured_points` |

### 12. Achievements `/admin/achievements/*`（4 个）

对应文档：[`12-achievements.md`](./12-achievements.md)　前端 API：`admin/api/achievements.ts`　页面：`admin/pages/achievements/index.vue`　状态：✅

| # | 方法 | 路径 | 前端函数 | 完成 | 完成日期 | commit |
|---|---|---|---|:---:|---|---|
| 1 | GET | `/admin/achievements` | `getAchievements` | [x] | 2026-04-21 | |
| 2 | POST | `/admin/achievements` | `createAchievement` | [x] | 2026-04-21 | HTTP 201 |
| 3 | PUT | `/admin/achievements/:id` | `updateAchievement` | [x] | 2026-04-21 | |
| 4 | DELETE | `/admin/achievements/:id` | `deleteAchievement` | [x] | 2026-04-21 | 软删 |

---

## S5 · P3 模块（18 个接口）

### 13. Notices `/admin/notices/*`（4 个 + 2 个前后端补齐）

对应文档：[`13-notices.md`](./13-notices.md)　前端 API：`admin/api/notices.ts`　页面：`admin/pages/notices/index.vue`　状态：✅

| # | 方法 | 路径 | 前端函数 | 完成 | 完成日期 | commit |
|---|---|---|---|:---:|---|---|
| 1 | GET | `/admin/notices` | `getNotices` | [x] | 2026-04-21 | |
| 2 | GET | `/admin/notices/:id` | `getNotice` | [x] | 2026-04-21 | |
| 3 | DELETE | `/admin/notices/:id` | `deleteNotice` | [x] | 2026-04-21 | 撤回 |
| 4 | PUT | `/admin/notices/:id/audit` | `auditNotice` | [x] | 2026-04-21 | |
| + | POST | `/admin/notices` | `createNotice` | [x] | 2026-04-21 | 发布通知 |
| + | PUT | `/admin/notices/:id` | `updateNotice` | [x] | 2026-04-21 | 编辑通知 |

### 14. Announcements `/admin/announcements/*`（6 个）

对应文档：[`14-announcements.md`](./14-announcements.md)　前端 API：`admin/api/announcements.ts`　页面：`admin/pages/announcements/index.vue`　状态：✅

| # | 方法 | 路径 | 前端函数 | 完成 | 完成日期 | commit |
|---|---|---|---|:---:|---|---|
| 1 | GET | `/admin/announcements` | `getAnnouncements` | [x] | 2026-04-21 | |
| 2 | GET | `/admin/announcements/:id` | `getAnnouncement` | [x] | 2026-04-21 | |
| 3 | POST | `/admin/announcements` | `createAnnouncement` | [x] | 2026-04-21 | |
| 4 | PUT | `/admin/announcements/:id` | `updateAnnouncement` | [x] | 2026-04-21 | |
| 5 | DELETE | `/admin/announcements/:id` | `deleteAnnouncement` | [x] | 2026-04-21 | |
| 6 | PUT | `/admin/announcements/:id/top` | `setAnnouncementTop` | [x] | 2026-04-21 | |

### 15. Audit `/admin/audit/*`（3 个）

对应文档：[`15-audit.md`](./15-audit.md)　前端 API：`admin/api/audit.ts`　页面：`admin/pages/audit/index.vue`　状态：✅

| # | 方法 | 路径 | 前端函数 | 完成 | 完成日期 | commit |
|---|---|---|---|:---:|---|---|
| 1 | GET | `/admin/audit/list` | `getAuditList` | [x] | 2026-04-21 | `content_audits` 聚合；`need_audit` 时补 pending 行 |
| 2 | PUT | `/admin/audit/:id` | `auditContent` | [x] | 2026-04-21 | |
| 3 | GET | `/admin/audit/stats` | `getAuditStats` | [x] | 2026-04-21 | |

### 19. Logs `/admin/logs`（1 个）

对应文档：[`19-logs.md`](./19-logs.md)　前端 API：`admin/api/logs.ts`　页面：`admin/pages/logs/index.vue`　状态：✅

| # | 方法 | 路径 | 前端函数 | 完成 | 完成日期 | commit |
|---|---|---|---|:---:|---|---|
| 1 | GET | `/admin/logs` | `getOperationLogs` | [x] | 2026-04-22 | |

### 20. Export `/admin/export/*`（4 个）

对应文档：[`20-export.md`](./20-export.md)　前端 API：`admin/api/export.ts`　页面：`admin/pages/export/index.vue`　状态：✅

| # | 方法 | 路径 | 前端函数 | 完成 | 完成日期 | commit |
|---|---|---|---|:---:|---|---|
| 1 | POST | `/admin/export/attendance` | `exportAttendance` | [x] | 2026-04-22 | 异步；`export_tasks` + goroutine |
| 2 | POST | `/admin/export/students` | `exportStudentsTask` | [x] | 2026-04-22 | 同上 |
| 3 | POST | `/admin/export/growth` | `exportGrowthTask` | [x] | 2026-04-22 | 同上 |
| 4 | GET | `/admin/export/:taskId` | `getExportTask` | [x] | 2026-04-22 | 另：`GET /admin/export/download/:taskId` 流式下载 |

---

## S6 · 回归测试

- [ ] 跑 `admin/tests/playwright-*` 全套（**待接入**：`admin/package.json` 尚未添加 Playwright 依赖与脚本）
- [x] 生成新一期 `admin/tests/playwright-reports/YYYYMMDD/` → **`20260422/BACKEND_MISSING_APIS.md`**（结案快照）
- [x] 把 `BACKEND_MISSING_APIS.md` 更新为「无缺失」结论（见 `20260422` 文件；历史 `20260420` 保留）
- [x] 更新 `backend/api/api-admin/README.md` 里模块状态列（已为 ✅，并增加最新缺失对照链接）

---

## 附：前端待补封装

（当前无待补项；历史已完成的见各模块 `api/*.ts`。）
