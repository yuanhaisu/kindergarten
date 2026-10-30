# Tasks

## 1. 契约

- [ ] 1.1 新建 `backend/api/api-teacher/10-album.md`：stats/list/detail/create 四端点与字段、分类四值、校验规则

## 2. Domain / 迁移

- [ ] 2.1 新建 `domain/album.go`（Album/AlbumPhoto，软删与索引）
- [ ] 2.2 `app/provider.go` 登记 AutoMigrate；`dbcheck albums`、`dbcheck album_photos` 核实

## 3. Repository / Service / Handler / Router

- [ ] 3.1 `album_repo.go`：统计（总数/本月/相册数）、分类列表分页、详情+照片、创建（事务）
- [ ] 3.2 `teacher_album_service.go`：校验与映射（category_label）、班级/园所收敛
- [ ] 3.3 `teacher_portal.go` 四个 handler；`router.go` 注册；Wire 注入
- [ ] 3.4 [P] 确认 Casbin 无需改动（teacher 前缀）

## 4. 测试

- [ ] 4.1 service 测试：创建 3 张顺序、无照片/超 50 拒绝、非法分类拒绝、统计跨月、分类筛选、详情顺序、跨班/跨园隔离
- [ ] 4.2 在 `docs/tests/TEST_CATALOG.md` 登记 `TC-NEW-ALB-0001`–`0007`

## 5. 验证

- [ ] 5.1 `cd server && go build ./...` exit 0；定向 `go test ./internal/service/ -run 'Album'` exit 0
- [ ] 5.2 真实 HTTP：创建→统计→筛选→详情；非法入参 400；跨班 404；临时数据硬删 0 残留
- [ ] 5.3 `dbcheck` 输出核实；secret 0
- [ ] 5.4 逐条对照 `specs/class-album/spec.md` 验收
