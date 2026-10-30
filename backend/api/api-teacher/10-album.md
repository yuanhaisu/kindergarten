# 模块 10：班级相册 API

> 基础路径：`/api/v1/teacher`
>
> 认证：`Authorization: Bearer <token>`，用户角色为 `teacher`
>
> 以下 Response 均为 `data` 部分，省略外层 `{ "code": 0, "message": "success" }` 包装。
>
> 相册只属于老师当前任教班级（由登录态 `TeacherPortalContext` 决定，不接受请求里的班级参数）。
> 跨班/跨园一律按不存在处理（404），不泄露其它班级数据是否存在。

## 接口清单

- `GET /teacher/albums/stats` — 照片统计卡
- `GET /teacher/albums` — 相册列表（分类筛选 + 分页）
- `GET /teacher/albums/:id` — 相册详情（含照片列表）
- `POST /teacher/albums` — 创建相册（先上传，后关联已上传 URL）

---

## 分类取值

| 值 | 中文标签 | 说明 |
| --- | --- | --- |
| `activity` | 活动 | 郊游、运动会等班级活动 |
| `daily` | 日常 | 日常在园生活 |
| `work` | 作品 | 手工、绘画等幼儿作品 |
| `festival` | 节日 | 节日主题活动 |

> 设计稿的第五个胶囊「全部」是列表筛选值（不传 `category`），**不是**分类，不入库。
> 分类中文标签由服务端在 `category_label` 返回，端上不自行映射。

---

## GET `/teacher/albums/stats`

> 统计卡三项。`monthly_new` 按相册的 `album_date` 所在自然月统计（不是创建时间）。

```json
// Response
{
  "total_photos": 386,
  "monthly_new": 42,
  "album_count": 12
}
```

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `total_photos` | int | 本班全部相册的照片数之和 |
| `monthly_new` | int | `album_date` 在本月的相册照片数之和 |
| `album_count` | int | 本班相册数 |

---

## GET `/teacher/albums`

> 相册列表，按 `album_date` 倒序（同日按 `id` 倒序，顺序稳定）。

### 请求参数

| 参数 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `category` | string | 否 | `activity` / `daily` / `work` / `festival`；不传返回全部 |
| `page` | int | 否 | 页码，默认 1 |
| `page_size` | int | 否 | 每页条数，默认 20，最大 50 |

> 非法 `category` 返回 422，不静默当作「全部」。

### 响应

```json
{
  "list": [
    {
      "id": 12,
      "title": "秋日郊游",
      "category": "activity",
      "category_label": "活动",
      "album_date": "2026-10-24",
      "photo_count": 12,
      "cover_url": "https://cdn.example.com/album/cover.jpg",
      "cover_urls": [
        "https://cdn.example.com/album/1.jpg",
        "https://cdn.example.com/album/2.jpg",
        "https://cdn.example.com/album/3.jpg"
      ]
    }
  ],
  "total": 12,
  "page": 1,
  "page_size": 20
}
```

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `id` | uint64 | 相册 ID |
| `title` | string | 标题 |
| `category` | string | 分类值 |
| `category_label` | string | 分类中文标签 |
| `album_date` | string | 相册日期，`YYYY-MM-DD` |
| `photo_count` | int | 照片数（创建时冗余写入） |
| `cover_url` | string | 封面，创建时取第一张照片的 `file_url` |
| `cover_urls` | string[] | 封面候选，最多 3 张，按 `sort_num` 升序；无照片时为空数组 `[]`（不是 `null`） |

---

## GET `/teacher/albums/:id`

> 相册详情与大图浏览。照片按创建时的提交顺序返回（`sort_num` 升序）。

```json
// Response
{
  "id": 12,
  "title": "秋日郊游",
  "category": "activity",
  "category_label": "活动",
  "album_date": "2026-10-24",
  "photo_count": 12,
  "cover_url": "https://cdn.example.com/album/cover.jpg",
  "photos": [
    {
      "id": 101,
      "file_url": "https://cdn.example.com/album/1.jpg",
      "thumb_url": "https://cdn.example.com/album/1_thumb.jpg",
      "width": 1200,
      "height": 900
    }
  ]
}
```

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `photos[].id` | uint64 | 照片 ID |
| `photos[].file_url` | string | 原图 URL |
| `photos[].thumb_url` | string | 缩略图 URL，可为空串 |
| `photos[].width` / `height` | int | 像素尺寸；未知时为 0 |

不存在、已删除、跨班或跨园：`404`。

---

## POST `/teacher/albums`

> 创建相册。照片需先走 `POST /common/upload` 上传，再以 URL 与尺寸关联。
> 服务端整单校验，任一字段非法则整单拒绝，不落半条数据。

### 请求

```json
{
  "title": "秋日郊游",
  "category": "activity",
  "album_date": "2026-10-24",
  "photos": [
    {
      "file_url": "https://cdn.example.com/album/1.jpg",
      "thumb_url": "https://cdn.example.com/album/1_thumb.jpg",
      "width": 1200,
      "height": 900
    }
  ]
}
```

| 字段 | 类型 | 必填 | 规则 |
| --- | --- | --- | --- |
| `title` | string | 是 | 去首尾空格后 1–30 个字符 |
| `category` | string | 是 | 仅 `activity` / `daily` / `work` / `festival` |
| `album_date` | string | 是 | `YYYY-MM-DD`，必须为合法日期 |
| `photos` | array | 是 | 1–50 条，按下标写入 `sort_num` |
| `photos[].file_url` | string | 是 | `http://` 或 `https://` URL |
| `photos[].thumb_url` | string | 否 | 非空时必须是 `http(s)` URL |
| `photos[].width` / `height` | int | 否 | 非负；未知写 0 |

### 响应（201）

```json
{
  "id": 12
}
```

### 错误

| 场景 | HTTP | code |
| --- | --- | --- |
| 标题为空/超 30、分类非法、日期非法、照片数不在 1–50、URL 非 http(s)、尺寸为负 | 422 | 42200 |
| 未绑定老师班级 | 403 | 40300 |
| 服务端异常 | 500 | 50000 |

---

## 实现约束

- 封面 `cover_url` 与 `photo_count` 在创建时一次写入；本期无编辑/删除接口，没有漂移来源。
- 列表的 `cover_urls` 由一次 `album_photos` 批量查询（`album_id IN (...)`，按 `sort_num` 升序）后按相册取前 3 张，不做逐相册查询（禁止 N+1）。
- 写路径一律带 `kindergarten_id` 与 `class_id = 当前老师班级`；读路径同样收敛，跨班/跨园返回 404。
- 本期不支持：编辑/删除相册、家长端浏览、视频、跨班级相册、管理端相册管理。
