# Proposal

## Why

老师端「班级相册」（设计 `teacher/16-album-index`）需要统计卡（照片总数/本月新增/相册数）、分类筛选、
按月分组的相册卡与大图浏览，但后端**完全没有相册能力**：无 Album 域模型、无表、无契约、无路由、无 service，
`GrowthMedia` 也没有 album/category/cover 字段。页面目前只能显示假统计与空列表。

## What Changes

- 新增相册域：`albums`（班级、标题、分类、日期、封面、照片数）与 `album_photos`（相册、文件、排序）。
- 新增老师端接口：统计、相册列表（分类筛选 + 分页）、相册详情（照片列表）、创建相册（先上传后关联）。
- 分类四值：`activity` 活动 / `daily` 日常 / `work` 作品 / `festival` 节日（设计五个胶囊中「全部」为筛选值，非分类）。
- 照片通过现有 `POST /common/upload` 上传后，以 URL/尺寸引用写入 `album_photos`。
- **新增契约文件 `backend/api/api-teacher/10-album.md`，属于契约变更。**
- 明确不做（out of scope）：
  - 家长端相册浏览/复用（后续切片）。
  - 视频、AI 识别、跨班级相册、相册编辑/删除（本期只做创建与读取）。
  - 管理端相册管理。

## Capabilities

### New Capabilities

- `class-album`: 老师班级相册的存储、分类、统计与读取行为。

### Modified Capabilities

（无）

## Impact

- 契约：新增 `backend/api/api-teacher/10-album.md`。
- 后端：`domain/album.go`、`repository/album_repo.go`、`service/teacher_album_service.go`、`handler/http/teacher_portal.go`、
  `app/router.go`、`app/provider.go`（AutoMigrate）。
- 老师端：T06b 页面消费；本片不改页面。
- 数据：两张新表，AutoMigrate + `dbcheck` 核实；无存量迁移。
