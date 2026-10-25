# 养成星球 · Admin 端接口文档（分模块）

> 管理后台（Web 端）全部接口，基础路径：`/api/v1/admin`
>
> 所有接口均需 `Authorization: Bearer <token>`，且用户角色为 `admin`。
>
> 以下 Response 均为 `data` 部分，省略外层 `{ "code": 0, "message": "success" }` 包装。

---

## 🚀 开始实现前必读（执行计划）

本目录不仅是接口规格，也是实现计划的主场地。采用 **"一模块一会话"** 工作流，5 份文档配套使用：

| 文档 | 作用 | 更新频率 |
|---|---|---|
| [`KICKOFF_PROMPT.md`](./KICKOFF_PROMPT.md) | **新会话开场提示词模板**（用户复制粘贴即可） | 长期稳定 |
| [`EXECUTION_PLAN.md`](./EXECUTION_PLAN.md) | 方法论、Sprint 划分、Recipe、DoD、约定 | 长期稳定 |
| [`PROGRESS.md`](./PROGRESS.md) | 95 行 API 逐条进度表 + Sprint 总览 | 每完成 1 个接口勾一次 |
| [`SESSION_HANDOFF.md`](./SESSION_HANDOFF.md) | 全局会话接力卡：`Next Action`、`Blockers` | 每次会话开始/结束 |
| [`workbooks/`](./workbooks/) | 每模块 1 份工作簿，记录实现细节、决策、问题 | 模块开工到完成 |

### 新会话如何启动？

打开新 Chat，复制下面任意一句：

- **自动续接**：`请按 backend/api/api-admin/KICKOFF_PROMPT.md 的 "自动续接流程" 继续执行 admin 接口计划。`
- **指定模块**：`请按 backend/api/api-admin/KICKOFF_PROMPT.md 开始做 02 classes 模块（只做这一个模块，完成后结账）。`

AI 会自动读文档、确认状态、开工、结账。不需要每次重新解释背景。

---

## 模块索引

本目录是原 [`api-admin.md`](../api-admin.md) 按业务模块切分后的子文件，与 Admin 前端页面一一对应，便于后端按模块并行开发。

| # | 文件 | 模块 | 对应前端页面 | 接口数 | 状态 |
|---|---|---|---|---:|---|
| 01 | [`01-dashboard.md`](./01-dashboard.md) | 数据看板 | `admin/pages/index.vue` | 4 | ✅ 已实现 |
| 02 | [`02-classes.md`](./02-classes.md) | 班级管理 | `admin/pages/classes/index.vue` | 6 | ✅ 已实现 |
| 03 | [`03-students.md`](./03-students.md) | 学生管理 | `admin/pages/students/index.vue` | 8 | ✅ 已实现 |
| 04 | [`04-teachers.md`](./04-teachers.md) | 老师管理 | `admin/pages/teachers/index.vue` | 6 | ✅ 已实现 |
| 05 | [`05-parents.md`](./05-parents.md) | 家长管理 | `admin/pages/parents/index.vue` | 5 | ✅ 已实现 |
| 06 | [`06-attendance.md`](./06-attendance.md) | 出勤管理 | `admin/pages/attendance/index.vue` | 3 | ✅ |
| 07 | [`07-leaves.md`](./07-leaves.md) | 请假管理 | `admin/pages/leaves/index.vue` | 3 | ✅ |
| 08 | [`08-pickup.md`](./08-pickup.md) | 接送管理 | `admin/pages/pickup/index.vue` | 4 | ✅ |
| 09 | [`09-growth.md`](./09-growth.md) | 成长记录 | `admin/pages/growth/index.vue` | 5 | ✅ |
| 10 | [`10-tasks.md`](./10-tasks.md) | 亲子任务 | `admin/pages/tasks/index.vue` | 13 | ✅ |
| 11 | [`11-points.md`](./11-points.md) | 积分管理 | `admin/pages/points/index.vue` | 5 | ✅ 已实现 |
| 12 | [`12-achievements.md`](./12-achievements.md) | 成就管理 | `admin/pages/achievements/index.vue` | 4 | ✅ 已实现 |
| 13 | [`13-notices.md`](./13-notices.md) | 通知管理 | `admin/pages/notices/index.vue` | 6 | ✅ 已实现 |
| 14 | [`14-announcements.md`](./14-announcements.md) | 园所公告 | `admin/pages/announcements/index.vue` | 6 | ✅ 已实现 |
| 15 | [`15-audit.md`](./15-audit.md) | 内容审核 | `admin/pages/audit/index.vue` | 3 | ✅ |
| 16 | [`16-admins.md`](./16-admins.md) | 管理员 | `admin/pages/admins/index.vue` | 5 | ✅ |
| 17 | [`17-roles.md`](./17-roles.md) | 角色权限 | `admin/pages/roles/index.vue` | 5 | ✅ |
| 18 | [`18-settings.md`](./18-settings.md) | 系统设置 / 园所信息 | `admin/pages/settings/index.vue` | 6 | ✅ |
| 19 | [`19-logs.md`](./19-logs.md) | 操作日志 | `admin/pages/logs/index.vue` | 1 | ✅ |
| 20 | [`20-export.md`](./20-export.md) | 数据导出 | `admin/pages/export/index.vue` | 4 | ✅ |
| 21 | [`21-calendar.md`](./21-calendar.md) | 园级日历与课表 | `admin/pages/calendar/index.vue`、`admin/pages/timetable/index.vue` | 11 | 🔲 未实现（变更 `align-class-calendar`） |

**合计：** `PROGRESS.md` 表内共 **95** 行 Admin API；完成数以该文件顶部「总体进度」为准。

> **后端缺失对照（最新）**：[`admin/tests/playwright-reports/20260422/BACKEND_MISSING_APIS.md`](../../../admin/tests/playwright-reports/20260422/BACKEND_MISSING_APIS.md)（结案：**0** 缺失，以 `PROGRESS.md` 为准）  
> 历史基线（2026-04-20）：[`admin/tests/playwright-reports/20260420/BACKEND_MISSING_APIS.md`](../../../admin/tests/playwright-reports/20260420/BACKEND_MISSING_APIS.md)

---

## 后端已实现的接口（对照参考）

以下 18 个接口已在 `server/internal/app/router.go` 中注册并可用，不在本目录范围内：

| 方法 | 路径 | 功能 |
|---|---|---|
| GET | `/health` | 健康检查 |
| GET | `/api/v1/system/status` | 系统初始化状态 |
| POST | `/api/v1/system/init` | 系统首次初始化 |
| POST | `/api/v1/auth/sms/send` | 发送短信验证码 |
| POST | `/api/v1/auth/login/sms` | 短信登录 |
| POST | `/api/v1/auth/login/wechat` | 微信登录 |
| POST | `/api/v1/auth/login/password` | 密码登录（admin 登录可用） |
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

## 推荐实现优先级

| 优先级 | 模块 | 接口数 | 原因 |
|---|---|---:|---|
| **P0** | 01 数据看板 | 4 | 登录后第一眼的页面 |
| **P0** | 02 班级管理 | 6 | 其他业务（学生/出勤/任务）的基础 |
| **P0** | 03 学生管理 | 8 | 核心主数据 |
| **P1** | 04-05 老师 / 家长 | 11 | 人员管理 |
| **P1** | 18 系统设置 / 园所信息 | 6 | 园所基本信息 |
| **P1** | 16-17 管理员 / 角色 | 10 | 权限配置 |
| **P2** | 06-08 出勤 / 请假 / 接送 | 9 | 依赖学生/班级数据 |
| **P2** | 09-12 成长 / 任务 / 积分 / 成就 | 18 | 成长业务闭环 |
| **P3** | 13-15 通知 / 公告 / 审核 | 13 | 内容业务，较独立 |
| **P3** | 19-20 日志 / 导出 | 5 | 运维辅助 |

---

## 前端待补（小项）

以下是前端 `api/` 封装缺失、但页面按钮暗示应有的接口（若后端实现，需一并在前端封装）：

| 模块 | 缺失的前端函数 | 预期后端 |
|---|---|---|
| 07 请假管理 | 审批请假 | `PUT /admin/leaves/:id/audit` |
