# Proposal

## Why

老师端手动登记弹窗的「选授权人」模式只有 tab 高亮切换，没有选人列表 UI——老师无法从授权人中选择来接者。后端 `POST /pickup/manual` 的 `authorization_id` 参数与校验链已就绪，repo 层查询也已存在，缺的是中间的 API 端点和前端选人交互。同时，送入（入园）没有独立登记入口（弹窗硬编码 `type: 2`），学生未入园时只能收到「先登记入园再接走」的冷提示。

## What Changes

- **新增** `GET /teacher/pickup/students/:id/authorized-persons` 端点：按学生返回有效授权接送人列表（已审核 + 未过期 + status=1），供弹窗选人使用。**本次将修改 `backend/api/`，属于契约变更**
- **补全**手动登记弹窗「选授权人」模式的选人 UI：radio 风格卡片列表（头像 + 姓名 + 关系 + 脱敏手机号），点选记录 `authorization_id`，切换到「标例外」时清空选人状态
- **新增**送入（入园）快捷按钮：学生状态为 `not_arrived` 时显示「入园」一键按钮，直接调 `POST /pickup/manual`（`type: 1`，不走弹窗——送入不记交接人、不要理由）
- **优化**错误提示：「先登记入园再接走」→「该生今日尚未入园，请先登记入园（送入）后再登记接走」

## Capabilities

### New Capabilities

- `teacher-pickup-registration`: 老师端手动接送登记的交互行为——包括按学生查看有效授权人、选人/标例外二选一、送入快捷登记、错误提示可操作性

### Modified Capabilities

（无——`pickup-custody` 尚未合入 `openspec/specs/`，本变更不修改已有 capability）

## Impact

### 契约（本次将修改 backend/api/）

| 端点 | 变更 | 字段溯源 |
|---|---|---|
| `GET /teacher/pickup/students/:id/authorized-persons` | **新增** | 响应字段 `id`/`name`/`relation`/`avatar`/`phone`/`auth_type` 与 `GET /parent/pickup/authorized-persons` 对齐（`backend/api/api-parent/05-pickup.md` 已定义同名字段），来源 `domain.PickupAuthorization` |

### Go 后端

- `server/internal/repository/pickup_repo.go`：复用 `ListAuthorizationsByStudentIDs`（已有），无需新查询
- `server/internal/service/teacher_portal_service.go`：新增 `StudentAuthorizedPersons(ctx, studentID)` 方法（过滤有效授权人）
- `server/internal/handler/http/teacher_portal.go`：新增 handler
- `server/internal/app/router.go`：注册路由
- `configs/rbac_policy.csv`：确认 `/teacher/*` 通配已覆盖

### 老师端

- `teacher/src/api/teacher-pickup.js`：新增 `fetchStudentAuthPersons(studentId)`
- `teacher/src/pages/pickup/index.vue`：弹窗选人 UI + 送入快捷按钮 + 错误提示优化

### 不动的东西

- `domain/pickup.go`：数据结构不变
- `parent/` / `admin/`：不涉及
- `POST /pickup/manual`：入参不变（`authorization_id` 已就绪）
