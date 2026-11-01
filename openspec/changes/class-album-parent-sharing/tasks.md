# Tasks

## 1. 数据模型与迁移

- [x] 1.1 在 `server/internal/domain` 增加 `Album` 状态字段、`AlbumPhoto.file_id`、`AlbumStudent`、`AlbumPhotoStudent`、`ParentSetting` 两开关、`Student.status_changed_at`；`go build ./...` 通过，且 `go vet ./internal/domain/` 无新增错误
- [x] 1.2 编写 `docs/migrations/` 下的相册/设置/学生状态迁移 SQL（含索引与唯一键），执行后用 `go run ./cmd/dbcheck albums`、`dbcheck album_photos`、`dbcheck album_photo_students`、`dbcheck parent_settings` 核对索引与列
- [x] 1.3 迁移 `parent_settings`：`allow_class_share = allow_growth_share`、`allow_external_share = 0`；验证前后行数一致、抽样旧 true/false 值保持
- [x] 1.4 既有 `albums` 全部置 `status='draft'`；验证已发布数为 0、家长端接口不返回任何历史相册
- [x] 1.5 按 `file_url` 回填 `album_photos.file_id`（关联 `files`），未命中保留空；验证回填数量 + 未命中数量 = 总行数

## 2. 后端：老师相册生命周期

- [x] 2.1 `AlbumRepo` 增加发布/下架/编辑/软删除/状态筛选；`Stats` 只统计已发布；验证单测覆盖草稿/下架不计入、软删不计入
- [x] 2.2 `TeacherPortalService` 实现状态机与审计（publish/unpublish/edit/delete），发布校验至少 1 张照片、标题/分类/日期合法；验证单测覆盖合法/非法转换与操作人记录
- [x] 2.3 发布预览返回「含未授权孩子的照片数」；验证单测：有 opt-out 孩子时计数正确、无 opt-out 时为 0
- [x] 2.4 老师端 handler/router 增加 `PUT /teacher/albums/:id`、`POST /teacher/albums/:id/publish`、`POST /teacher/albums/:id/unpublish`、`DELETE /teacher/albums/:id`；验证 `go test ./internal/handler/http/` 与 `go build ./...`
- [x] 2.5 更新 `backend/api/api-teacher/10-album.md`：状态字段、发布/下架/编辑/删除、预览、参与者；验证契约示例与 handler 字段逐项对照
- [x] 2.6 在 `docs/tests/TEST_CATALOG.md` 登记 `TC-NEW-ALB-0008..0015` 并映射到新增单测；验证 TC ID 与测试名一一对应

## 3. 后端：照片-幼儿关联

- [x] 3.1 `AlbumRepo` 支持 `album_students` 与 `album_photo_students` 的批量写入/替换/查询；验证唯一键冲突与重复提交幂等
- [x] 3.2 创建/编辑相册接受 `file_id`、相册级参与幼儿、照片级幼儿；校验 file_id 属于本园、照片数 1–50；验证单测覆盖越权 file_id 与非法参与幼儿
- [x] 3.3 从成长记录选图时用 `growth_log_students` 自动带出照片级幼儿；验证单测：记录关联 A/B → 照片关联 A/B，手工覆盖后以覆盖为准
- [x] 3.4 更新 `backend/api/api-teacher/10-album.md` 的照片/参与者字段与 `00-common` 上传返回 `id` 的关联说明；验证契约与请求样例可被 handler 绑定
- [x] 3.5 在 `TEST_CATALOG.md` 登记 `TC-NEW-ALB-0016..0017`；验证对应用例存在且通过

## 4. 后端：隐私设置与老师端联系方式

- [x] 4.1 `ParentPortalService` 设置读写支持 `allow_class_share`/`allow_external_share`，兼容旧 `allow_growth_share` 入参，返回新键；验证单测覆盖新键、旧键、部分更新与迁移值
- [x] 4.2 老师端班级学生详情返回 `parents[]`（姓名/关系/手机号），手机号按每个家长的 `show_phone_to_teacher` 过滤；验证单测覆盖全开/全关/多家长部分关闭
- [x] 4.3 更新 `backend/api/api-parent/11-settings.md` 与 `backend/api/api-teacher/09-class-and-tasks.md`；验证契约字段与 handler 响应一致
- [x] 4.4 在 `TEST_CATALOG.md` 登记 `TC-NEW-PPS-0001..0008`；验证对应用例存在且通过

## 5. 后端：家长端班级相册

- [x] 5.1 新增家长班级相册查询：仅已发布、同园同班、未软删；按孩子与 `allow_class_share` 过滤照片、整张隐藏 opt-out 合影、自己孩子始终可见；验证单测覆盖多孩、opt-out、草稿/下架/删除
- [x] 5.2 列表返回 `visible_photo_count` 且不返回全量照片，详情返回可见照片并按 `sort_num,id` 排序；验证单测与查询次数（无 N+1）
- [x] 5.3 毕业后 90 天只读、退学/转园立即拒绝；`status_changed_at` 为空的历史 2/3 状态按已过期；验证单测覆盖边界日期
- [x] 5.4 新增 `GET /parent/class-albums`、`GET /parent/class-albums/:id` handler/router；验证 `go test ./internal/handler/http/` 与 401/403/404 分支
- [x] 5.5 新建 `backend/api/api-parent/12-class-albums.md`（或项目现有编号的下一个），写清发布状态、我的孩子过滤、opt-out、留存；验证契约示例与 handler 一致
- [x] 5.6 在 `TEST_CATALOG.md` 登记 `TC-NEW-PCA-0001..0010`；验证对应用例存在且通过

## 6. 后端：管理端相册只读/下架

- [x] 6.1 新增 admin 相册列表/详情/下架 service/handler/router；只读 + 下架，写审计；验证单测覆盖跨园隔离、不能编辑、下架后家长不可见
- [x] 6.2 新建 `backend/api/api-admin/22-albums.md`（21 已被 21-calendar.md 占用）；验证契约字段与 handler 一致
- [x] 6.3 在 `TEST_CATALOG.md` 登记 `TC-NEW-ALB-0018..0019`；验证对应用例存在且通过

## 7. 老师端页面

- [x] 7.1 `teacher/src/pages/album/index.vue` 增加状态徽标、草稿/已发布筛选、发布/下架/编辑/软删入口；验证 `npm run build:mp-weixin` exit 0
- [ ] 7.2 创建/编辑相册支持批量参与幼儿选择与逐张微调，发布预览展示未授权合影提示；验证 DevTools 真实点击：选 3 张、批量选 A/B、改 1 张为 A、预览提示正确
- [x] 7.3 `teacher/src/pages/class/index.vue` 学生信息卡展示家长联系方式（手机号按开关过滤）；验证 DevTools：开/关两种家长数据下手机号出现/缺省
- [ ] 7.4 两处改动只用现有 token；`node scripts/check-tokens.mjs teacher` 保持基线 3，无新增；验证 `check-secrets` 0

## 8. 家长端页面

- [ ] 8.1 新增 `parent/src/pages/album/index.vue` 与 `detail.vue`，在 `pages.json` 注册；列表仅已发布，详情网格 + 预览 + 逐张保存原图；验证 `npm run build:mp-weixin` exit 0
- [ ] 8.2 入口：`parent/src/pages/class/index.vue` 与 `growth/album.vue` 增加班级相册入口，使用 `navigateTo`（非 tabBar 页）；验证 DevTools 真实点击可达
- [ ] 8.3 默认「我的孩子」过滤与 opt-out 整张隐藏在前端正确呈现（后端已过滤，前端不二次猜测）；验证 DevTools：自己孩子可见、其他孩子照片不出现、关闭共享后合影消失且自己仍可见
- [ ] 8.4 `parent/src/pages/settings/index.vue` 将「允许展示成长记录」改为「班内共享」，不展示园外分享；验证 DevTools 开关保存后重新进入保持
- [ ] 8.5 token 基线 3 无新增；`check-secrets` 0；验证两端 `build:mp-weixin` 均 exit 0

## 9. 管理端页面

- [x] 9.1 新增 `admin/pages/albums/index.vue` 与详情抽屉（只读 + 下架），在菜单/路由注册；验证 `cd admin && npm run build` exit 0
- [x] 9.2 下架后家长端立即不可见；验证 admin 真实点击 + 家长端 DevTools 刷新
- [x] 9.3 无新 token；如使用状态色，只用语义 token；验证 `check-tokens` 无新增

## 10. 留存、清理与指标

- [ ] 10.1 新增 `server/cmd/albumretention/main.go`：清理软删满 180 天的相册/照片/关联，保留审计；验证 `go run ./cmd/albumretention` 在测试数据上执行并核对残留
- [ ] 10.2 学生状态变更为 2/3 时写 `status_changed_at`（admin 学生编辑路径）；验证单测覆盖状态变更与时间戳
- [ ] 10.3 埋点 `album_publish`/`album_parent_view`/`album_photo_save`/`album_opt_out`/`album_complaint`；验证日志字段含 kg/class/album/role 且不含手机号/openid

## 11. 集成验证与收尾

- [ ] 11.1 `cd server && go build ./... && go vet ./internal/... && go test ./internal/service/ -run 'Album|ClassAlbum|ParentSettings' -count=1 -timeout 600s` 全绿
- [ ] 11.2 两端 `npm run build:mp-weixin` exit 0；admin `npm run build` exit 0；`check-tokens` 两端 3/3；`check-secrets` 0
- [ ] 11.3 DevTools 端到端：老师创建草稿 → 选图/参与者 → 发布 → 家长「我的孩子」可见 → 关闭班内共享 → 其他家长合影消失、自己仍可见 → 老师下架 → 家长不可见 → 园长只读并下架
- [ ] 11.4 数据清理：测试相册/照片/关联/设置全部硬删，残留断言 0；审计日志按测试 ID 清理
- [ ] 11.5 对照 `specs/` 逐条勾选验收，更新 `PROGRESS.md` 与 handoff；未验证项不得标记完成
