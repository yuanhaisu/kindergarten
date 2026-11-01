# Design

## Context

现状与约束（动机见 `proposal.md`，行为验收见 `specs/`）：

- 班级相册已有 `albums`/`album_photos` 与老师端列表/详情/创建，但无发布状态、无编辑/删除、照片不关联幼儿；`album_photos` 只存 URL，不符合 `media-assets` 的引用统一字段。
- 家长端已有「成长相册」（成长记录媒体的派生视图），没有班级相册入口。
- `parent_settings` 只有单个 `allow_growth_share`，默认 false，且无读取方；`show_phone_to_teacher` 同样无读取方；老师端目前没有家长手机号展示面。
- `students.status` 有 0 待入学 / 1 在读 / 2 毕业 / 3 退学，但没有状态变更时间，无法计算 90/180 天窗口。
- 管理端没有相册模块；家长端没有班级相册页；两端都没有对应设计稿，Phase 1 复用现有相册视觉与设计 token。
- `media-assets` 已明确：同一文件可被多个业务引用，可见性按引用判定。班级相册必须是**独立引用**，不能改变成长记录引用的可见性。

## Goals / Non-Goals

**Goals**

- 班级相册成为可发布、可撤回、可审计的班级资产，家长只看到已发布且与自己孩子相关的内容。
- 隐私开关从“存储值”变成真实生效的展示约束，并保留家长撤回权。
- 同一张照片可在成长记录与班级相册中各有一条引用，互不覆盖可见性。
- 既有相册安全迁移：不直接暴露历史内容，老师复核后再发布。

**Non-Goals**

- Phase 2 的园所「全班」视图、视频、园外分享/下载、通知/周报、打包导出、人脸模糊、园长发布审核。
- 不改成长记录的发布/编辑流程；不改成长相册的派生视图语义。
- 不做跨班/跨园共享，不做家长上传。

## Decisions

### D1. 班级相册照片是独立引用，文件层共享

`album_photos` 增加 `file_id`（可空，兼容历史 URL），保留 `file_url`/`thumb_url` 作为读取缓存；从成长记录选图时复用同一个文件行，但新建 `album_photos` 引用。可见性按引用判定，符合 `media-assets` 的“同一文件多条引用各自生效”。

替代方案：复制文件行（存储翻倍、同一图两套 URL）；把成长媒体直接挂到相册（会把成长记录的可见性带进相册，冲突）。

### D2. 数据模型

- `albums`：新增 `status`（`draft` / `published` / `unpublished`）、`published_at`、`published_by`；沿用 `SoftDeleteModel`。`photo_count`/`cover_url` 仍是写入时冗余，编辑/删除照片时在同一事务重算。
- `album_students`：相册级参与幼儿（批量上传的默认参与者、未逐张标记照片的兜底）。
- `album_photo_students`：照片级关联幼儿（唯一键 `photo_id + student_id`）；从成长记录选图时由 `growth_log_students` 自动写入。
- `parent_settings`：新增 `allow_class_share`（默认 true）与 `allow_external_share`（默认 false）；保留旧 `allow_growth_share` 列用于迁移与回滚，API 不再返回旧键。
- `students`：新增 `status_changed_at`（可空）。状态变为 2/3 时写入；历史数据无值按“已过期”处理（不放开访问），避免用 `updated_at` 猜。

替代方案：把参与者只放在相册级（无法做“我的孩子”照片级过滤）；只放照片级（新上传工作量过大）；单独建 `album_photo_student` 之外再建相册参与者表（重复）。

### D3. 发布状态机

```
draft --publish--> published --unpublish--> unpublished
  ^                    |                        |
  |                    +------ edit ------------+
  +------------------ (edit 不改变状态) --------+
任何状态 --soft delete--> deleted（180 天后清理）
```

- 创建默认 `draft`；发布要求至少 1 张照片、标题/分类/日期合法；下架后家长立即不可见。
- 任一老师可操作，`operation_logs` 记录 `album.publish` / `album.unpublish` / `album.edit` / `album.delete`。
- 园长只能读与下架，不能创建/编辑；下架同样写审计。

### D4. 家长可见性查询

家长班级相册列表/详情只查 `status=published`、同园同班、未软删的相册。照片可见性按以下顺序：

1. 取当前家长孩子的集合（`pc.Students`），以及每个孩子的 `allow_class_share`（任一监护人关闭即视为关闭）。
2. 照片有 `album_photo_students` 时：孩子在其中 → 可见；否则不可见。
3. 照片没有逐张关联时：孩子在该相册的 `album_students` 中 → 可见。
4. 最终过滤：若照片关联到任何 `allow_class_share=false` 的幼儿，且该幼儿不是当前家长的孩子 → **整张隐藏**；当前家长自己的孩子始终可见。

列表只返回相册元信息与 `visible_photo_count`（由详情查询聚合，避免列表 N+1）；详情返回可见照片，顺序仍按 `sort_num, id`。

替代方案：整本相册隐藏（伤及全班）；人脸模糊（Phase 2）。

### D5. 多监护人口径

同一幼儿有多个监护人时，班内共享按**任一监护人关闭即关闭**处理（隐私优先）；老师端手机号按每个家长各自的 `show_phone_to_teacher` 过滤。替代方案：任一开启即开启（会覆盖另一位监护人的撤回，不采用）。

### D6. 隐私设置迁移与 UI

- 无 `parent_settings` 行：`allow_class_share=true`、`allow_external_share=false`。
- 已有行：`allow_class_share = allow_growth_share`（保留旧值），`allow_external_share=false`；不把旧值静默改成开启。
- Phase 1 设置页只展示「班内共享」；`allow_external_share` 只在数据模型/契约中存在，等待园外分享能力上线后再展示（避免第二个死开关）。
- PUT 设置同时接受旧键 `allow_growth_share`（映射到班内共享）与新键，返回新键；旧键保留一个版本用于回滚。

### D7. 老师端家长联系方式

在老师端班级学生详情返回 `parents[]`（姓名、关系、手机号）；手机号按每个家长的 `show_phone_to_teacher` 过滤，关闭时字段缺省而不是空串。班级页在学生信息卡中展示，不新增独立页面。

### D8. 留存与清理

- 读取时按学生状态与 `status_changed_at` 计算：毕业 90 天内只读；退学/转园立即不可见；`status_changed_at` 为空的历史毕业/退学学生按已过期处理。
- 「导出」在 Phase 1 定义为家长可逐张保存原图（复用小程序预览/保存能力），不做打包导出；打包导出留 Phase 2。
- 软删除 180 天清理用 `cmd/albumretention` 维护命令（与现有 `cmd/` 辅助命令同模式），不新增常驻 worker；调度由运维定时执行。读取时也会用 `deleted_at + 180d` 兜底过滤。

### D9. 前端页面与导航

- 家长端新增 `pages/album/index.vue`（列表）与 `pages/album/detail.vue`（网格 + 预览）；非 tabBar 页，使用 `navigateTo`。入口：班级页与成长相册页。
- 老师端沿用 `pages/album/index.vue`，增加状态徽标、发布/下架、参与者批量选择与发布预览；无 tabBar，使用 `navigateTo`/页内弹层。
- 管理端新增 `admin/pages/albums/index.vue` 与详情抽屉，只读 + 下架；不新增编辑表单。
- 没有现成设计稿：家长端复用老师相册的卡片/网格视觉与家长端 token；管理端复用现有列表/详情模式。视觉验收按现有设计系统，不新增 token；如出现新的状态色，使用语义 token。

### D10. 指标

Phase 1 用结构化 slog 事件（不建分析表）：`album_publish`、`album_parent_view`、`album_photo_save`、`album_opt_out`、`album_complaint`；字段含 `kindergarten_id`、`class_id`、`album_id`、`role`，不记录手机号/openid。后续接分析平台时再从日志聚合。

## 跨端触点清单

| 层 | 是否修改 | 内容 |
| --- | --- | --- |
| Go domain | 会改 | `Album` 状态字段；`AlbumPhoto.file_id`；新增 `AlbumStudent`、`AlbumPhotoStudent`；`ParentSetting` 两开关；`Student.status_changed_at` |
| repository | 会改 | `AlbumRepo` 发布/编辑/软删/家长查询；`ParentSetting` 读写；`Student` 状态时间；审计日志复用现有 repo |
| service | 会改 | 老师相册生命周期与参与者；家长班级相册与过滤；设置迁移；家长联系方式过滤；留存窗口 |
| handler | 会改 | 老师相册新端点；家长相册；admin 相册；设置；班级学生联系方式 |
| Casbin policy | 不改 | 现有 `teacher/*`、`parent/*`、`admin/*` 通配已覆盖；新增路由仅需 router 注册与契约登记 |
| router.go | 会改 | 新增老师发布/下架/编辑/删除；家长班级相册；admin 相册只读/下架 |
| backend/api/ | 会改 | `api-teacher/10-album.md`、`09-class-and-tasks.md`、`api-parent/11-settings.md`、新增家长端班级相册契约；本次属于契约变更 |
| 老师端页面 | 会改 | `teacher/src/pages/album/index.vue`（状态/发布/参与者/预览）、`teacher/src/pages/class/index.vue`（家长联系方式） |
| 家长端页面 | 会改 | 新增 `parent/src/pages/album/{index,detail}.vue`；`settings/index.vue`（班内共享）；`class/index.vue`、`growth/album.vue` 入口 |
| 管理端页面 | 会改 | 新增 `admin/pages/albums/index.vue` + 详情抽屉（只读/下架） |
| design token | 不改 | 复用现有颜色/间距/圆角；新增状态徽标只用语义 token |
| 测试 | 会改 | Go 单测/集成；`TEST_CATALOG.md` 新增 ALB/PCA/PPS 用例；DevTools 双端 + admin 真实交互 |
| 指标 | 会改 | slog 事件；无新表 |

## 契约字段溯源

- `status` / `published_at` / `published_by`：ADR 0003 的发布流程；无设计稿，属于明确业务场景。
- `participants` / `photos[].students`：ADR 0003 的照片-幼儿混合关联；不能由已有字段派生（当前照片无幼儿信息）。
- `file_id`：`media-assets` 的引用统一字段；已有 `file_url` 不能替代文件行标识。
- `allow_class_share` / `allow_external_share`：ADR 0003 的共享模型；单一 `allow_growth_share` 无法表达两种风险。
- `visible_photo_count`：列表不能返回全量照片，家长需要知道“这本相册里有几张是我孩子的”；由详情聚合，不在列表 N+1。
- 家长端相册字段（title/category/album_date/photo_count/cover_urls）复用老师端相册已有语义，不新增重复字段。

## Migration Plan

1. **SQL 迁移**（AutoMigrate 只增不改，必须手工执行并用 `go run ./cmd/dbcheck <table>` 核实）：
   - `albums` 加 `status`（默认 `draft`）、`published_at`、`published_by`；索引 `(kindergarten_id, class_id, status, album_date)`。
   - `album_photos` 加 `file_id`（可空）+ 索引；按 `file_url` 关联 `files` 回填，未命中保留空。
   - 新建 `album_students`（唯一 `album_id + student_id`）与 `album_photo_students`（唯一 `photo_id + student_id`）。
   - `parent_settings` 加 `allow_class_share`（默认 1）、`allow_external_share`（默认 0）；`UPDATE ... SET allow_class_share = allow_growth_share`。
   - `students` 加 `status_changed_at`；历史 2/3 留空（按已过期处理）。
2. **既有相册安全迁移**：所有既有 `albums.status` 置 `draft`，不自动发布；老师复核参与者后再发布，避免历史照片直接暴露。
3. **上线顺序**：先执行 schema 迁移，再一次性部署包含设置默认值的新后端（4.1）→ 老师端（可见草稿/发布）→ 家长端（入口）→ 管理端；家长端入口在发布前为空态，不阻塞。**在设置默认值（4.1）完成前不要用新模型启动服务**——旧 `UpdateSettings` 建行时不会写 `allow_class_share`，新模型无 GORM default 标签会把零值 false 落库。
4. **回滚**：回滚代码即可；旧 `allow_growth_share` 列与旧键解析保留一个版本。新表/新列若保留，**必须先把 `allow_class_share` / `allow_external_share` 的 DDL 默认值恢复为 1/0**（AutoMigrate 会按新模型去掉默认值），否则旧代码 INSERT `parent_settings` 会因新列无默认而失败；或者直接删除两个新列。既有相册仍是 draft，不会回退成已发布。
5. **验证**：`go build ./...`、`go test`、两端 `build:mp-weixin`、admin `npm run build`；DevTools 真实点击；`dbcheck` 核对索引/行数；数据回填残留数为 0。

## Risks / Trade-offs

- **历史相册全部变草稿** → 老师需要逐本复核发布；上线初期家长端可能为空。用老师端列表状态徽标与空态说明缓解。
- **旧 `allow_growth_share=false` 的家长保持关闭** → 其孩子的照片不会出现在其他家长视图；需要一次性设置提示，避免老师误以为发布后全班可见。
- **未逐张标记的合影可能包含其他孩子** → 相册级兜底与发布前未授权提示只能降低风险，不能完全消除；老师发布前需人工确认。Phase 2 再评估人脸模糊。
- **同一幼儿多监护人** → 任一关闭即隐藏，可能比另一位监护人的意愿更保守；这是隐私优先的刻意选择。
- **家长端/管理端无设计稿** → 复用现有视觉，可能出现像素级差异；视觉验收按设计系统而非新稿。
- **`file_id` 回填失败** → 保留 URL 兜底；新写入一律带 `file_id`。
- **留存清理依赖 `status_changed_at`** → 历史数据无时间戳按已过期处理，可能让部分历史毕业家长立即失去访问；符合数据最小化，但需在上线说明中告知。
- **导出 = 逐张保存原图** → 不提供打包导出；若产品要求 ZIP/成长册，Phase 2 另立切片。

## Open Questions

- 打包导出/成长册的产品形态（Phase 2，不影响 Phase 1 规格）。
- 园所「全班」视图的默认开关与审批流程（Phase 2）。
- 人脸模糊的技术方案与成本（Phase 2）。
