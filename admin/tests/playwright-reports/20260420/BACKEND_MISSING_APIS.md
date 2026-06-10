# Admin 端后端接口缺失清单

> 生成日期：2026-04-20
> 测试工具：Playwright（Chromium）
> 测试账号：`13800138000 / Admin@123`（角色：`super_admin`）
> 前端地址：`http://localhost:3000`
> 后端地址：`http://localhost:8080/api/v1`

---

## 一、总览

| 项 | 数量 |
|---|---|
| Admin 前端页面 | 22（登录 + 看板 + 20 个业务页） |
| 前端已封装的业务接口 | **约 87 个** |
| 后端已实现的接口 | 18 个（auth / common / system 基础） |
| **后端缺失接口** | **87 个（几乎全部 admin 业务接口）** |
| 受影响的前端页面 | **20 个业务页面全部受影响** |

### 关键结论

> **前端 UI 已完成 95%+，后端业务层仅完成了"登录 + 文件上传 + 消息 + 用户资料"这几个公共模块，所有管理后台业务接口均未实现。** 当前 admin 端每个业务页面打开后都会收到 `Request failed with status code 404` 错误，表格全部显示空态。

---

## 二、后端已实现的接口（对照参考）

以下 18 个接口已在 `server/internal/app/router.go` 中注册并可用：

| 方法 | 路径 | 功能 |
|---|---|---|
| GET | `/health` | 健康检查 |
| GET | `/api/v1/system/status` | 系统初始化状态 |
| POST | `/api/v1/system/init` | 系统首次初始化 |
| POST | `/api/v1/auth/sms/send` | 发送短信验证码 |
| POST | `/api/v1/auth/login/sms` | 短信登录 |
| POST | `/api/v1/auth/login/wechat` | 微信登录 |
| POST | `/api/v1/auth/login/password` | 密码登录 ✅（admin 登录可用） |
| POST | `/api/v1/auth/refresh` | 刷新 Token |
| POST | `/api/v1/auth/logout` | 退出登录 |
| POST | `/api/v1/common/upload` | 单文件上传 |
| POST | `/api/v1/common/upload/batch` | 批量上传 |
| GET | `/api/v1/common/user/profile` | 获取用户资料 |
| PUT | `/api/v1/common/user/profile` | 更新用户资料 |
| PUT | `/api/v1/common/user/avatar` | 更新头像 |
| GET | `/api/v1/common/messages` | 消息列表 |
| GET | `/api/v1/common/messages/unread-count` | 未读数 |
| PUT | `/api/v1/common/messages/:id/read` | 标记已读 |
| PUT | `/api/v1/common/messages/read-all` | 全部已读 |

---

## 三、按模块列出缺失接口

### 🚫 1. 数据看板 `/` → `admin/pages/index.vue`（4 个缺失）

| 方法 | 路径 | 前端函数 | 用途 |
|---|---|---|---|
| GET | `/admin/dashboard/overview` | `getDashboardOverview` | 首页 4 个统计卡（在园/请假/出勤率/已接走） |
| GET | `/admin/dashboard/attendance-trend?period=30d` | `getAttendanceTrend` | 近 30 日出勤趋势图 |
| GET | `/admin/dashboard/class-ranking` | `getClassRanking` | 班级出勤排行 |
| GET | `/admin/dashboard/growth-stats?period=30d` | `getGrowthStats` | 成长记录统计 |

---

### 🚫 2. 班级管理 `/classes` → `admin/pages/classes/index.vue`（6 个缺失）

| 方法 | 路径 | 前端函数 | 用途 |
|---|---|---|---|
| GET | `/admin/classes` | `getClasses` | 班级列表（支持分页、年级筛选） |
| GET | `/admin/classes/:id` | `getClassDetail` | 班级详情 |
| POST | `/admin/classes` | `createClass` | 新建班级 |
| PUT | `/admin/classes/:id` | `updateClass` | 编辑班级 |
| DELETE | `/admin/classes/:id` | `deleteClass` | 删除班级 |
| POST | `/admin/classes/:id/adjust` | `adjustClassStudents` | 调整班级学生（调入/调出） |

---

### 🚫 3. 学生管理 `/students` → `admin/pages/students/index.vue`（8 个缺失）

| 方法 | 路径 | 前端函数 | 用途 |
|---|---|---|---|
| GET | `/admin/students` | `getStudents` | 学生列表（分页、班级、状态筛选） |
| GET | `/admin/students/:id` | `getStudent` | 学生详情 |
| POST | `/admin/students` | `createStudent` | 新建学生 |
| PUT | `/admin/students/:id` | `updateStudent` | 编辑学生 |
| DELETE | `/admin/students/:id` | `deleteStudent` | 删除学生 |
| PUT | `/admin/students/:id/status` | `updateStudentStatus` | 更新学生状态（在读/毕业/退学） |
| POST | `/admin/students/import` | `importStudents` | 批量导入（Excel） |
| GET | `/admin/students/export` | `exportStudentsBlob` | 导出名册（Blob） |

---

### 🚫 4. 老师管理 `/teachers` → `admin/pages/teachers/index.vue`（6 个缺失）

| 方法 | 路径 | 前端函数 | 用途 |
|---|---|---|---|
| GET | `/admin/teachers` | `getTeachers` | 老师列表 |
| GET | `/admin/teachers/:id` | `getTeacher` | 老师详情 |
| POST | `/admin/teachers` | `createTeacher` | 新建老师 |
| PUT | `/admin/teachers/:id` | `updateTeacher` | 编辑老师 |
| DELETE | `/admin/teachers/:id` | `deleteTeacher` | 删除老师 |
| PUT | `/admin/teachers/:id/status` | `updateTeacherStatus` | 更新状态 |

---

### 🚫 5. 家长管理 `/parents` → `admin/pages/parents/index.vue`（5 个缺失）

| 方法 | 路径 | 前端函数 | 用途 |
|---|---|---|---|
| GET | `/admin/parents` | `getParents` | 家长列表 |
| GET | `/admin/parents/:id` | `getParent` | 家长详情 |
| POST | `/admin/parents` | `createParent` | 新建家长 |
| PUT | `/admin/parents/:id` | `updateParent` | 编辑家长 |
| PUT | `/admin/parents/:id/status` | `updateParentStatus` | 更新家长状态 |

---

### 🚫 6. 出勤管理 `/attendance` → `admin/pages/attendance/index.vue`（3 个缺失）

| 方法 | 路径 | 前端函数 | 用途 |
|---|---|---|---|
| GET | `/admin/attendance/overview?date=` | `getAttendanceOverview` | 出勤概览（按日期、按班级） |
| GET | `/admin/attendance/records` | `getAttendanceRecords` | 出勤明细（分页） |
| GET | `/admin/attendance/report` | `exportAttendanceReport` | 导出出勤报表（Blob） |

---

### 🚫 7. 请假管理 `/leaves` → `admin/pages/leaves/index.vue`（2 个缺失）

| 方法 | 路径 | 前端函数 | 用途 |
|---|---|---|---|
| GET | `/admin/leaves` | `getLeaves` | 请假列表 |
| GET | `/admin/leaves/stats` | `getLeaveStats` | 请假统计 |

> ⚠️ 页面显示的"审批通过/驳回"按钮尚未在前端接入接口（前端 `api/leaves.ts` 未封装审批接口），若后端实现请留意此点。

---

### 🚫 8. 接送记录 `/pickup` → `admin/pages/pickup/index.vue`（4 个缺失）

| 方法 | 路径 | 前端函数 | 用途 |
|---|---|---|---|
| GET | `/admin/pickup/overview?date=` | `getPickupOverview` | 接送概览 |
| GET | `/admin/pickup/records` | `getPickupRecords` | 接送明细 |
| GET | `/admin/pickup/abnormal` | `getPickupAbnormal` | 异常接送 |
| PUT | `/admin/pickup/authorized-persons/:id/audit` | `auditPickupAuthorizedPerson` | 授权人审核 |

---

### 🚫 9. 成长记录 `/growth` → `admin/pages/growth/index.vue`（5 个缺失）

| 方法 | 路径 | 前端函数 | 用途 |
|---|---|---|---|
| GET | `/admin/growth/logs` | `getGrowthLogs` | 成长记录列表 |
| GET | `/admin/growth/logs/:id` | `getGrowthLog` | 成长记录详情 |
| DELETE | `/admin/growth/logs/:id` | `deleteGrowthLog` | 删除记录 |
| PUT | `/admin/growth/logs/:id/audit` | `auditGrowthLog` | 审核（通过/驳回） |
| GET | `/admin/growth/stats` | `getGrowthStats` | 成长统计（`/statistics` 页面也用） |

---

### 🚫 10. 亲子任务 `/tasks` → `admin/pages/tasks/index.vue`（4 个缺失）

| 方法 | 路径 | 前端函数 | 用途 |
|---|---|---|---|
| GET | `/admin/tasks` | `getTasks` | 任务列表 |
| GET | `/admin/tasks/:id` | `getTask` | 任务详情 |
| DELETE | `/admin/tasks/:id` | `deleteTask` | 删除任务 |
| GET | `/admin/tasks/stats` | `getTaskStats` | 任务统计 |

> ⚠️ 前端 `api/tasks.ts` 没有封装"新建任务"接口，但页面按钮文案暗示存在该功能，需和后端一起对齐。

---

### 🚫 11. 积分管理 `/points` → `admin/pages/points/index.vue`（5 个缺失）

| 方法 | 路径 | 前端函数 | 用途 |
|---|---|---|---|
| GET | `/admin/points/overview` | `getPointsOverview` | 积分概览 |
| GET | `/admin/points/records` | `getPointRecords` | 积分流水 |
| POST | `/admin/points/adjust` | `adjustPoints` | 手动调整积分 |
| GET | `/admin/points/rules` | `getPointRules` | 积分规则 |
| PUT | `/admin/points/rules` | `updatePointRules` | 更新规则 |

---

### 🚫 12. 成就管理 `/achievements` → `admin/pages/achievements/index.vue`（4 个缺失）

| 方法 | 路径 | 前端函数 | 用途 |
|---|---|---|---|
| GET | `/admin/achievements` | `getAchievements` | 成就列表 |
| POST | `/admin/achievements` | `createAchievement` | 新建成就 |
| PUT | `/admin/achievements/:id` | `updateAchievement` | 编辑成就 |
| DELETE | `/admin/achievements/:id` | `deleteAchievement` | 删除成就 |

---

### 🚫 13. 通知管理 `/notices` → `admin/pages/notices/index.vue`（4 个缺失）

| 方法 | 路径 | 前端函数 | 用途 |
|---|---|---|---|
| GET | `/admin/notices` | `getNotices` | 通知列表 |
| GET | `/admin/notices/:id` | `getNotice` | 通知详情 |
| DELETE | `/admin/notices/:id` | `deleteNotice` | 删除/撤回通知 |
| PUT | `/admin/notices/:id/audit` | `auditNotice` | 审核通知 |

> ⚠️ 前端 `api/notices.ts` 没有封装"发布通知"接口（POST），需要后端补齐；同时前端也需补调用。

---

### 🚫 14. 园所公告 `/announcements` → `admin/pages/announcements/index.vue`（6 个缺失）

| 方法 | 路径 | 前端函数 | 用途 |
|---|---|---|---|
| GET | `/admin/announcements` | `getAnnouncements` | 公告列表 |
| GET | `/admin/announcements/:id` | `getAnnouncement` | 公告详情 |
| POST | `/admin/announcements` | `createAnnouncement` | 发布公告 |
| PUT | `/admin/announcements/:id` | `updateAnnouncement` | 编辑公告 |
| DELETE | `/admin/announcements/:id` | `deleteAnnouncement` | 删除公告 |
| PUT | `/admin/announcements/:id/top` | `setAnnouncementTop` | 置顶/取消置顶 |

---

### 🚫 15. 内容审核 `/audit` → `admin/pages/audit/index.vue`（3 个缺失）

| 方法 | 路径 | 前端函数 | 用途 |
|---|---|---|---|
| GET | `/admin/audit/list` | `getAuditList` | 待审核列表 |
| PUT | `/admin/audit/:id` | `auditContent` | 审核（通过/驳回） |
| GET | `/admin/audit/stats` | `getAuditStats` | 审核统计 |

---

### 🚫 16. 数据统计 `/statistics` → `admin/pages/statistics/index.vue`（复用接口）

| 方法 | 路径 | 依赖 |
|---|---|---|
| GET | `/admin/leaves/stats` | 与"请假管理"共用 |
| GET | `/admin/growth/stats` | 与"成长记录"共用 |
| GET | `/admin/tasks/stats` | 与"亲子任务"共用 |

---

### 🚫 17. 管理员 `/admins` → `admin/pages/admins/index.vue`（5 个缺失）

| 方法 | 路径 | 前端函数 | 用途 |
|---|---|---|---|
| GET | `/admin/admins` | `getAdmins` | 管理员列表 |
| POST | `/admin/admins` | `createAdmin` | 新建管理员 |
| PUT | `/admin/admins/:id` | `updateAdmin` | 编辑管理员 |
| DELETE | `/admin/admins/:id` | `deleteAdmin` | 删除管理员 |
| PUT | `/admin/admins/:id/reset-password` | `resetAdminPassword` | 重置密码 |

---

### 🚫 18. 角色权限 `/roles` → `admin/pages/roles/index.vue`（5 个缺失）

| 方法 | 路径 | 前端函数 | 用途 |
|---|---|---|---|
| GET | `/admin/roles` | `getRoles` | 角色列表 |
| POST | `/admin/roles` | `createRole` | 新建角色 |
| PUT | `/admin/roles/:id` | `updateRole` | 编辑角色 |
| DELETE | `/admin/roles/:id` | `deleteRole` | 删除角色 |
| GET | `/admin/permissions` | `getPermissionTree` | 权限树（分配权限用） |

---

### 🚫 19. 系统设置 `/settings` → `admin/pages/settings/index.vue`（6 个缺失）

| 方法 | 路径 | 前端函数 | 用途 |
|---|---|---|---|
| GET | `/admin/settings` | `getSettings` | 系统设置 |
| PUT | `/admin/settings` | `updateSettings` | 更新设置 |
| GET | `/admin/settings/academic-years` | `getAcademicYears` | 学年列表 |
| POST | `/admin/settings/academic-years` | `createAcademicYear` | 新建学年 |
| GET | `/admin/kindergarten/info` | `getKindergartenInfo` | 园所信息 |
| PUT | `/admin/kindergarten/info` | `updateKindergartenInfo` | 更新园所信息 |

---

### 🚫 20. 操作日志 `/logs` → `admin/pages/logs/index.vue`（1 个缺失）

| 方法 | 路径 | 前端函数 | 用途 |
|---|---|---|---|
| GET | `/admin/logs` | `getOperationLogs` | 操作日志列表 |

---

### 🚫 21. 数据导出 `/export` → `admin/pages/export/index.vue`（4 个缺失）

| 方法 | 路径 | 前端函数 | 用途 |
|---|---|---|---|
| POST | `/admin/export/attendance` | `exportAttendance` | 出勤异步导出 |
| POST | `/admin/export/students` | `exportStudentsTask` | 学生异步导出 |
| POST | `/admin/export/growth` | `exportGrowthTask` | 成长异步导出 |
| GET | `/admin/export/:taskId` | `getExportTask` | 查询导出任务 |

---

## 四、推荐的后端实现优先级

| 优先级 | 模块 | 影响页面 | 接口数 | 原因 |
|---|---|---|---|---|
| **P0** | Dashboard | 看板（首页） | 4 | 登录后第一眼的页面，必须能看数据 |
| **P0** | Classes | 班级管理 | 6 | 其他业务（学生/出勤/任务）的基础 |
| **P0** | Students | 学生管理 | 8 | 核心主数据 |
| **P1** | Teachers / Parents | 老师/家长管理 | 11 | 人员管理 |
| **P1** | Settings / Kindergarten | 系统设置 | 6 | 园所基本信息 |
| **P1** | Roles / Admins | 角色/管理员 | 10 | 权限配置 |
| **P2** | Attendance / Leaves / Pickup | 日常业务 | 9 | 依赖学生/班级数据存在后才有意义 |
| **P2** | Growth / Tasks / Points / Achievements | 成长业务 | 18 | 同上 |
| **P3** | Notices / Announcements / Audit | 内容业务 | 13 | 较独立，可后置 |
| **P3** | Logs / Export | 运维辅助 | 5 | 依赖其他模块产生数据后才有值 |

**合计：约 87 个业务接口。**

---

## 五、前端待补（小项）

以下是前端 `api/` 封装缺失、但页面按钮暗示应有的接口（若后端实现需一并在前端封装）：

| 模块 | 缺失的前端函数 | 预期后端 |
|---|---|---|
| 请假管理 | 审批请假 | `PUT /admin/leaves/:id/audit` |
| 亲子任务 | 新建任务 | `POST /admin/tasks` |
| 亲子任务 | 编辑任务 | `PUT /admin/tasks/:id` |
| 通知管理 | 发布通知 | `POST /admin/notices` |
| 通知管理 | 编辑通知 | `PUT /admin/notices/:id` |

---

## 六、已修复的前端 Bug（本次测试中发现并修正）

| # | 文件 | 问题 | 修复 |
|---|---|---|---|
| 1 | `admin/nuxt.config.ts` | Nuxt 默认对嵌套 `components/` 子目录自动加前缀，导致 `StatCard`/`TodoList`/`AttendanceTrendChart`/`ClassRankingTable`/`GrowthStatsCard` 无法解析 | 增加 `components: [{ path: '~/components', pathPrefix: false }]` |
| 2 | `admin/layouts/default.vue` | 使用了带前缀的 `LayoutSidebar` / `LayoutTopbar` / `LayoutBreadcrumb` | 改为 `Sidebar` / `Topbar` / `Breadcrumb` |
| 3 | `admin/pages/statistics/index.vue` | 使用了 `ChartsPieChart` / `ChartsBarChart` 带前缀名 | 改为 `PieChart` / `BarChart` |

---

## 七、交付物目录

```
admin/tests/
├── PLAYWRIGHT_TEST_PLAN.md              # 测试计划（之前保存）
└── playwright-reports/
    └── 20260420/
        ├── BACKEND_MISSING_APIS.md      # 本文件：后端缺失接口清单
        └── screenshots/
            ├── 01-login-page-*.png
            └── 02-dashboard-*.png
```
