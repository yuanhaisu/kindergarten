# Proposal

## Why

班级相册目前只有老师自己能看到：家长端不可见、管理端无相册管理、无编辑/删除；它实际上是一个老师私人照片库，和成长记录高度重复。同时家长设置里的 `allow_growth_share`（允许展示成长记录）与 `show_phone_to_teacher` 只存值、没有任何读取方，是死开关，家长会误以为已经生效。Phase 1 要把班级相册变成**家长可见的班级活动相册**，并把共享/隐私开关真正接入展示路径。

## What Changes

- **隐私设置拆分与迁移**：把单一 `allow_growth_share` 拆成 `班内共享`（默认开、家长可撤回）与 `园外分享`（默认关、需显式开启）；新增家长端设置 UI；旧值迁移到 `班内共享`，`园外分享` 默认关。`show_phone_to_teacher` 接入老师端家长联系方式展示。
- **班级相册生命周期**：从“创建即落库”改为草稿 → 预览 → 发布；支持编辑、下架、软删除；本班任一老师可操作并写审计日志；园长（admin）只读且可下架。
- **照片-幼儿关联**：相册照片支持关联幼儿。从成长记录选图时自动使用 `growth_log_students`；新上传按批次选参与幼儿，可逐张微调。`album_photos` 补 `file_id`，遵循素材引用统一字段契约；对既有相册回填相册级参与者。
- **家长端班级相册**：新增家长端班级相册列表/详情，仅展示已发布相册；默认「我的孩子」按照片级关联过滤；家长关闭 `班内共享` 后，其他家长视图中含该孩子的照片整张隐藏，该孩子自己的家长仍可见。
- **留存**：毕业后家长端只读保留 90 天、可导出；转园/退园立即失去访问；软删除保留 180 天审计。
- **指标**：埋点家长周查看率、每班每月发布相册/照片数、家长保存/分享率，以及 opt-out 率、相册投诉护栏。
- **契约变更声明**：本次将修改 `backend/api/`，属于契约变更（`api-teacher/10-album.md`、`api-teacher/09-class-and-tasks.md`、`api-parent/11-settings.md`、新增家长端班级相册契约）。

**明确不做（Phase 2）**：园所「全班」视图开关、视频、园外分享/下载、发布通知/周报、导出/打印、人脸模糊、园长发布审核。

## Capabilities

### New Capabilities

- `parent-class-album`: 家长端查看已发布的班级相册，按「我的孩子」过滤、按共享开关隐藏照片、毕业后 90 天只读/转园失去访问。
- `parent-privacy-settings`: 班内共享与园外分享两个开关的默认值、迁移、撤回语义，以及 `show_phone_to_teacher` 对老师端联系方式展示的约束。

### Modified Capabilities

- `class-album`: 老师端相册从创建即发布改为草稿/发布/下架/软删；任一老师可操作并审计；照片关联幼儿与 `file_id`；园长只读+可下架；保留/清理规则。
- `growth-visibility`: 明确可见性按业务引用判定——同一文件被班级相册独立引用时，按班级相册的可见性规则生效，不改变成长记录引用的可见性。

## Impact

- **Go 后端**：`domain`（Album 状态字段、AlbumPhoto.file_id、AlbumPhotoStudent、ParentSetting 两开关）、repository、service（老师相册生命周期/家长相册/设置迁移/留存清理/审计）、handler、router、Wire 绑定；新增家长端班级相册与 admin 相册只读/下架路由。
- **契约**：`backend/api/api-teacher/10-album.md`、`09-class-and-tasks.md`（家长联系方式）、`api-parent/11-settings.md`、新增家长端班级相册契约。
- **老师端**：`teacher/src/pages/album/index.vue`（草稿/发布/下架、参与者批量选择、发布前未授权提示）、`class/index.vue` 或学生详情（家长联系方式）。
- **家长端**：新增班级相册列表/详情页（复用老师相册视觉与家长端 token，无现成设计稿）、`settings/index.vue`（两个开关）、`class/index.vue` 与成长页入口。
- **管理端**：新增只读相册列表/详情 + 下架（无现成设计稿，按现有 admin 列表模式）。
- **数据**：`albums` 新字段、`album_photos.file_id`、新表 `album_photo_students`、`parent_settings` 迁移；均为 AutoMigrate 只增不改，需手工迁移 SQL 并用 `go run ./cmd/dbcheck` 核实。
- **测试/指标**：新增 `TC-NEW-ALB-*`、`TC-NEW-PCA-*`、`TC-NEW-PPS-*` 用例并登记 `TEST_CATALOG.md`；埋点事件在 design/tasks 中定义。
