# Design

## Context

见 `proposal.md - Why`。约束：

- 老师上下文 `TeacherPortalContext` 已带 `KindergartenID/ClassID`，所有读写按此收敛。
- 现有 `GrowthMedia` 绑定成长记录，无相册/分类概念，不复用、不改动。
- 上传走现有 `POST /common/upload`，本片只接收已上传文件的引用。

## Goals / Non-Goals

**Goals:**

- 两张表支撑统计、分类列表、详情与大图浏览。
- 写入校验与班级/园所隔离。

**Non-Goals:**

- 上传、编辑、删除、家长端、视频、AI。

## Decisions

### D1 两表模型

- `albums`：`id/kindergarten_id/class_id/title/category/album_date/cover_url/photo_count/creator_id`，软删。
- `album_photos`：`id/album_id/file_url/thumb_url/width/height/sort_num`，随相册硬删。
- 封面取第一张照片 URL 冗余存 `cover_url`，列表免 join；`photo_count` 冗余计数。
- 理由：列表需要封面与张数，冗余避免 N+1；照片原文各端已有 file 记录，这里只存展示 URL。

### D2 分类四值

`activity/daily/work/festival`，中文标签由端上映射（活动/日常/作品/节日）。设计第五个胶囊「全部」是筛选值，不入库。

### D3 接口

- `GET /teacher/albums/stats` → `{total_photos, monthly_new, album_count}`（本月按 `album_date` 月份）。
- `GET /teacher/albums?category&page&page_size` → 列表（日期倒序），每项含 id/title/category/category_label/album_date/photo_count/cover_url。
- `GET /teacher/albums/:id` → 相册 + `photos[{id,file_url,thumb_url,width,height}]`（sort_num 升序）。
- `POST /teacher/albums` → `{title, category, album_date, photos[]}`，返回 `{id}`。
- 均在现有 teacher 路由组，Casbin `teacher /api/v1/teacher/* *` 已覆盖。

### D4 校验

- title 1–30 字符；category 必须四值之一；album_date `YYYY-MM-DD`；photos 1–50 条，每条 `file_url` 必须 http(s)。
- 非法整单拒绝；照片顺序按数组下标写 `sort_num`。

### D5 隔离

读写一律带 `kindergarten_id` 与 `class_id = tc.ClassID`；跨班/跨园返回不存在。

## 跨端触点清单

| 层 | 是否改 | 说明 |
| --- | --- | --- |
| Go domain | 会改 | 新增 `album.go`：Album / AlbumPhoto |
| repository | 会改 | 新增 `album_repo.go`：统计/列表/详情/创建 |
| service | 会改 | 新增 `teacher_album_service.go` + Wire 注入 |
| handler | 会改 | `teacher_portal.go` 增加 4 个 handler |
| Casbin policy | 不改 | teacher 前缀已覆盖 |
| router.go | 会改 | 注册 4 条路由 |
| backend/api 契约 | 会改（新增文件） | `api-teacher/10-album.md` |
| 端页面 | 本片不改 | T06b 消费 |
| design token | 不改 | 无 UI |
| 迁移 | 会改 | AutoMigrate 两表；`dbcheck albums` / `dbcheck album_photos` |

### 契约字段溯源

| 字段 | 出处 |
| --- | --- |
| `stats.total_photos/monthly_new/album_count` | 设计 16 统计卡三项 |
| `category` | 设计 16 五个胶囊（全部 + 活动/日常/作品/节日） |
| `album_date/photo_count/cover_url` | 设计 16 相册卡（日期/张数/封面） |
| `photos[]` | 设计 16 大图层（第 N 张 / 共 M 张） |

## Risks / Trade-offs

- [冗余 cover/count 与实际照片不一致] → 创建时一次写入；本期无编辑删除，无漂移来源。
- [照片 URL 非本地上传] → 只校验 http(s)，不做域名白名单；记录 P2。
- [本月口径] → 按 `album_date` 月份，非创建时间，符合「本月新增照片」语义。
- [大量照片] → 单相册上限 50，列表分页 20。

## Migration Plan

AutoMigrate 建两表；`go run ./cmd/dbcheck albums`、`dbcheck album_photos` 核实。回滚删表即可（无存量依赖）。
