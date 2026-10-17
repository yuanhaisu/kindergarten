# Tasks

## 1. 契约（backend/api/）

- [x] 1.1 `backend/api/api-teacher/07-pickup.md`：新增 `GET /teacher/pickup/students/:id/authorized-persons`，响应字段与 `GET /parent/pickup/authorized-persons` 对齐（id/name/relation/phone/avatar/auth_type），注明仅返回有效授权人（status=1, review_status=1, 未过期）。验证：文档内端点清单 + 字段列表完整

## 2. Go 后端

- [x] 2.1 `server/internal/service/teacher_portal_service.go`：新增 `StudentAuthorizedPersons(ctx, studentID)` 方法，复用 `ListAuthorizationsByStudentIDs` 并过滤有效授权人。验证：单测覆盖「返回有效授权人」「过滤未审核/已撤销/已过期」「空列表」3 个场景
- [x] 2.2 `server/internal/handler/http/teacher_portal.go`：新增 handler，校验学生归属本班，响应包络走 `pkg/response`。验证：`go build ./...` 通过
- [x] 2.3 `server/internal/app/router.go`：注册路由；`configs/rbac_policy.csv` 核对 `/teacher/*` 通配已覆盖。验证：`go build ./...` 通过
- [x] 2.4 `make wire`（若新增依赖则必须）。验证：`go vet ./...` 与 `go test ./internal/service/...` 通过

## 3. 老师端前端

- [x] 3.1 `teacher/src/api/teacher-pickup.js`：新增 `fetchStudentAuthPersons(studentId)`。验证：`npm run dev:mp-weixin` 构建通过
- [x] 3.2 `teacher/src/pages/pickup/index.vue`：弹窗「选授权人」模式显示 radio 卡片列表（头像+姓名+关系+脱敏手机号），点选高亮并记录 `authorization_id`；切换模式清空选人状态；未选人提交 toast 拦截。验证：构建通过
- [x] 3.3 `teacher/src/pages/pickup/index.vue`：学生卡片按状态派生按钮（not_arrived 显示「入园」+「接走」），「入园」一键直调 `POST /pickup/manual`（type=1）不走弹窗。验证：构建通过
- [x] 3.4 `teacher/src/pages/pickup/index.vue`：错误提示映射（先入园→可操作提示、重复接走缺理由→提示补理由）。验证：构建通过

## 4. 本组验证

- [x] 4.1 `npm run dev:mp-weixin` 与 `go test ./...` 全部通过；对照 spec 10 个 Scenario 逐条用真实交互验证（老师端无 tabBar，跳转一律 `navigateTo`）
