# 养成星球 · 数据结构设计

## Go 模型源文件（按业务域）

| 文件 | 域 |
|------|-----|
| `base.go` | 基础模型（`BaseModel` / `SoftDeleteModel`） |
| `organization.go` | 园所、班级 |
| `user.go` | 统一用户、角色、班级老师 |
| `student.go` | 学生、家长关联 |
| `notice.go` | 班级通知与模板 |
| `leave.go` | 请假 |
| `attendance.go` | 出勤 |
| `pickup.go` | 接送与授权 |
| `growth.go` | 成长日志、媒体、报告 |
| `calendar.go` | 班级日历 |
| `parent_task.go` | 亲子任务与打卡 |
| `points.go` | 积分与成就 |
| `announcement.go` | 园所公告 |
| `content_audit.go` | 内容审核 |
| `admin_rbac.go` | 管理后台 RBAC |
| `messaging.go` | 站内消息、微信推送 |
| `file.go` | 统一文件记录 |
| `operation_log.go` | 操作审计 |

以上均为 **`package model`**，引用方式不变（例如 `model.User`）。

## 架构概览

三端（老师端小程序、家长端小程序、管理后台网页端）共用一个 Go 后端，统一用户体系，通过角色区分权限。

```
┌─────────────────────────────────────────────────────────────────┐
│                        统一后端 (Go)                             │
├─────────────────────────────────────────────────────────────────┤
│  老师端 API (/api/teacher/*)                                     │
│  家长端 API (/api/parent/*)                                      │
│  管理后台 API (/api/admin/*)                                     │
└───────────────────────────┬─────────────────────────────────────┘
                            │
                    ┌───────┴───────┐
                    │   统一数据库    │
                    └───────────────┘
```

## 核心实体关系

```
Kindergarten (幼儿园)
 ├── Class (班级) 1:N
 │    ├── ClassTeacher (班级老师) N:M
 │    └── Student (学生) 1:N
 │         ├── StudentParent (家长关联) N:M
 │         └── PickupAuthorization (授权接送人) 1:N
 │
 ├── User (统一用户) 
 │    └── UserRole (角色: admin/teacher/parent) 1:N
 │
 ├── Notice (通知) 1:N
 │    ├── NoticeAttachment (附件) 1:N
 │    └── NoticeReadLog (已读记录) 1:N
 │
 ├── LeaveRequest (请假) 1:N
 ├── Attendance (出勤) 1:N
 ├── PickupRecord (接送记录) 1:N
 │
 ├── GrowthDailyLog (成长日志) 1:N
 │    ├── GrowthMedia (媒体文件) 1:N
 │    ├── GrowthComment (评论) 1:N
 │    └── GrowthLike (点赞) 1:N
 │
 ├── CalendarEvent (日历事件) 1:N
 ├── ParentTask (亲子任务) 1:N
 │    └── TaskCheckIn (打卡记录) 1:N
 │
 ├── Announcement (园所公告) 1:N
 └── ContentAudit (内容审核) 1:N
```

## 模型分层说明

### 第一层：组织与人员（基础）

| 模型 | 说明 | 三端使用情况 |
|------|------|-------------|
| `Kindergarten` | 幼儿园（顶层租户隔离） | 管理后台创建，三端共用 |
| `Class` | 班级 | 管理后台创建，老师端/家长端关联 |
| `User` | 统一用户表 | 三端共用，微信 OpenID 登录 |
| `UserRole` | 用户角色 | 区分 admin/teacher/parent |
| `ClassTeacher` | 班级-老师关联 | 管理后台配置，老师端使用 |
| `Student` | 学生 | 管理后台录入，三端共用 |
| `StudentParent` | 学生-家长关联 | 管理后台/家长端绑定 |

### 第二层：通知与沟通

| 模型 | 说明 | 写入端 | 读取端 |
|------|------|--------|--------|
| `Notice` | 班级通知 | 老师端 | 家长端、管理后台 |
| `NoticeAttachment` | 通知附件 | 老师端 | 家长端 |
| `NoticeReadLog` | 已读记录 | 家长端（自动） | 老师端、管理后台 |
| `NoticeTemplate` | 通知模板 | 老师端 | 老师端 |
| `LeaveRequest` | 请假申请 | 家长端 | 老师端审批、管理后台统计 |
| `Announcement` | 园所公告 | 管理后台 | 家长端、老师端 |

### 第三层：出勤与接送

| 模型 | 说明 | 写入端 | 读取端 |
|------|------|--------|--------|
| `Attendance` | 出勤记录 | 老师端/系统 | 管理后台统计 |
| `PickupRecord` | 接送记录 | 老师端扫码/手动 | 家长端通知、管理后台查询 |
| `PickupAuthorization` | 授权接送人 | 家长端 | 老师端核验 |
| `PickupQRCode` | 接送二维码 | 系统生成 | 家长端展示、老师端扫码 |

### 第四层：成长记录

| 模型 | 说明 | 写入端 | 读取端 |
|------|------|--------|--------|
| `GrowthDailyLog` | 每日成长日志 | 老师端 | 家长端时间轴 |
| `GrowthMedia` | 照片/视频 | 老师端上传 | 家长端浏览 |
| `GrowthMediaStudent` | 媒体-学生关联 | AI/老师端 | 家长端按孩子筛选 |
| `GrowthComment` | 评论 | 家长端 | 老师端、家长端 |
| `GrowthLike` | 点赞 | 家长端 | 老师端、家长端 |
| `GrowthReport` | AI 成长报告 | 系统生成 | 家长端 |

### 第五层：任务与积分

| 模型 | 说明 | 写入端 | 读取端 |
|------|------|--------|--------|
| `ParentTask` | 亲子任务 | 老师端布置 | 家长端 |
| `TaskCheckIn` | 打卡记录 | 家长端 | 老师端统计 |
| `PointAccount` | 积分账户 | 系统 | 家长端 |
| `PointTransaction` | 积分流水 | 系统 | 家长端 |
| `Achievement` | 成就定义 | 管理后台 | 家长端 |
| `StudentAchievement` | 已获成就 | 系统 | 家长端 |

### 第六层：管理后台专用

| 模型 | 说明 |
|------|------|
| `ContentAudit` | 内容审核（通知/成长记录） |
| `AdminRole` | 后台角色定义 |
| `AdminPermission` | 权限定义 |
| `AdminRolePermission` | 角色-权限关联 |
| `UserAdminRole` | 用户-角色关联 |
| `OperationLog` | 操作审计日志 |

### 通用支撑

| 模型 | 说明 |
|------|------|
| `Message` | 站内消息（三端） |
| `WechatPushLog` | 微信推送记录 |
| `FileRecord` | 统一文件管理 |

## 关键设计决策

1. **多租户隔离**：所有业务表都包含 `kindergarten_id`，支持未来多园所 SaaS 部署
2. **统一用户表**：一个人可以同时是老师和家长（通过 `UserRole` 多角色）
3. **软删除**：核心业务数据使用 `SoftDeleteModel`，保留审计追溯能力
4. **接送安全**：二维码动态生成 + 授权人管理 + 异常标记，三重保障
5. **成长记录解耦**：日志、媒体、学生关联分离，支持 AI 自动识别和手动标记
6. **积分体系独立**：积分账户 + 流水表，为后续兑换商城预留扩展空间
