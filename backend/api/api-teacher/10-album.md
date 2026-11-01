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

- `GET /teacher/albums/stats` — 照片统计卡（只统计已发布）
- `GET /teacher/albums` — 相册列表（分类/状态筛选 + 分页，老师端默认返回全部状态）
- `GET /teacher/albums/:id` — 相册详情（含状态、参与者、照片与幼儿关联、未授权计数）
- `POST /teacher/albums` — 创建相册草稿（先上传，后关联 `file_id` 或 URL）
- `PUT /teacher/albums/:id` — 编辑相册（标题/分类/日期/照片/参与者）
- `POST /teacher/albums/:id/publish` — 发布草稿或重新上架已下架相册
- `POST /teacher/albums/:id/unpublish` — 下架已发布相册
- `DELETE /teacher/albums/:id` — 软删除相册

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

## 发布状态

| 值 | 说明 | 家长端可见 |
| --- | --- | --- |
| `draft` | 新建默认状态；可编辑、可发布 | 否 |
| `published` | 已发布；编辑保存后立即对家长生效 | 是 |
| `unpublished` | 已下架；可重新发布 | 否 |

```
draft --publish--> published --unpublish--> unpublished
  ^                    |                        |
  |                    +------ edit ------------+
  +------------------ (edit 不改变状态) --------+
```

- `publish` 允许 `draft` / `unpublished` → `published`；已发布再次发布返回 422。
- `unpublish` 只允许 `published` → `unpublished`；其它状态返回 422。
- 任何非删除状态都可 `edit`，编辑不改变状态。
- `delete` 为软删除；软删后老师详情与家长端均不可见，相册行保留 180 天供审计。
- 发布要求至少 1 张照片，且标题/分类/日期合法；不满足返回 422。

---

## GET `/teacher/albums/stats`

> 统计卡三项，**只统计 `status=published` 且未软删的相册**；草稿、已下架与软删不计入。
> `monthly_new` 按相册的 `album_date` 所在自然月统计（不是创建时间）。

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
| `total_photos` | int | 本班已发布相册的照片数之和 |
| `monthly_new` | int | `album_date` 在本月的已发布相册照片数之和 |
| `album_count` | int | 本班已发布相册数 |

---

## GET `/teacher/albums`

> 相册列表，按 `album_date` 倒序（同日按 `id` 倒序，顺序稳定）。
> 老师端默认返回全部状态（管理视图）；家长端只查 `published`，两者不是同一个入口。

### 请求参数

| 参数 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `category` | string | 否 | `activity` / `daily` / `work` / `festival`；不传返回全部 |
| `status` | string | 否 | `draft` / `published` / `unpublished`；不传返回全部状态 |
| `page` | int | 否 | 页码，默认 1 |
| `page_size` | int | 否 | 每页条数，默认 20，最大 50 |

> 非法 `category` 或 `status` 返回 422，不静默当作「全部」。

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
      ],
      "status": "published",
      "published_at": "2026-10-25T02:30:00Z",
      "published_by": 61
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
| `photo_count` | int | 照片数（写入时冗余，编辑时重算） |
| `cover_url` | string | 封面，取第一张照片的 `file_url`（编辑时重算） |
| `cover_urls` | string[] | 封面候选，最多 3 张，按 `sort_num` 升序；无照片时为空数组 `[]`（不是 `null`） |
| `status` | string | `draft` / `published` / `unpublished` |
| `published_at` | string \| null | 最近一次发布时间（RFC3339 UTC）；草稿/下架为 `null` |
| `published_by` | uint64 | 最近一次发布人 user_id；未发布为 0 |

---

## GET `/teacher/albums/:id`

> 相册详情与大图浏览。照片按创建时的提交顺序返回（`sort_num` 升序）。
> 老师端**不按状态过滤**：草稿与已下架相册也要能打开编辑与预览。

```json
// Response
{
  "id": 12,
  "title": "秋日郊游",
  "category": "activity",
  "category_label": "活动",
  "album_date": "2026-10-24",
  "photo_count": 2,
  "cover_url": "https://cdn.example.com/album/1.jpg",
  "status": "published",
  "published_at": "2026-10-25T02:30:00Z",
  "published_by": 61,
  "album_students": [
    { "id": 1001, "name": "小明", "avatar": "https://cdn.example.com/avatar/1001.jpg" }
  ],
  "photos": [
    {
      "id": 101,
      "file_id": 9001,
      "file_url": "https://cdn.example.com/album/1.jpg",
      "thumb_url": "https://cdn.example.com/album/1_thumb.jpg",
      "width": 1200,
      "height": 900,
      "students": [
        { "id": 1001, "name": "小明", "avatar": "https://cdn.example.com/avatar/1001.jpg" }
      ]
    }
  ],
  "unauthorized_photo_count": 0
}
```

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `status` / `published_at` / `published_by` | — | 同列表字段 |
| `album_students` | object[] | 相册级参与幼儿；无参与者为 `[]` |
| `album_students[].id` | uint64 | 幼儿 ID |
| `album_students[].name` | string | 姓名；幼儿已软删时为空串 |
| `album_students[].avatar` | string | 头像 URL，可为空串 |
| `photos[].id` | uint64 | 照片 ID（编辑时提交该值可保留原照片） |
| `photos[].file_id` | uint64 \| null | 文件行 ID（`files.id`）；历史 URL 行为 `null` |
| `photos[].file_url` / `thumb_url` | string | 原图/缩略图 URL |
| `photos[].width` / `height` | int | 像素尺寸；未知时为 0 |
| `photos[].students` | object[] | 照片级关联幼儿；**无逐张关联时为 `[]`**（家长端可见性按相册级 `album_students` 兜底） |
| `unauthorized_photo_count` | int | 发布预览用：含未开启班内共享幼儿的照片数；照片无逐张关联时按 `album_students` 判断 |

不存在、已删除、跨班或跨园：`404`。

---

## POST `/teacher/albums`

> 创建相册**草稿**。照片需先走 `POST /common/upload` 上传；
> **上传响应里的 `id` 就是本契约的 `file_id`**（`POST /teacher/albums` 的
> `photos[].file_id` 填这个值，不是 URL、不是 `files` 以外的其它 ID）。
> 服务端整单校验，任一字段非法则整单拒绝，不落半条数据
> （相册、照片、两级幼儿关联在同一事务内写入）。

### 请求

```json
{
  "title": "秋日郊游",
  "category": "activity",
  "album_date": "2026-10-24",
  "participants": [1001, 1002],
  "photos": [
    {
      "file_id": 9001,
      "width": 1200,
      "height": 900,
      "student_ids": [1001]
    },
    {
      "source_media_id": 7001
    }
  ]
}
```

| 字段 | 类型 | 必填 | 规则 |
| --- | --- | --- | --- |
| `title` | string | 是 | 去首尾空格后 1–30 个字符 |
| `category` | string | 是 | 仅 `activity` / `daily` / `work` / `festival` |
| `album_date` | string | 是 | `YYYY-MM-DD`，必须为合法日期 |
| `participants` | uint64[] | 否 | 相册级参与幼儿（批量默认参与者）；必须属于本园本班，重复值自动去重。**全量替换语义**：提交清单即最终清单，不传或 `[]` 表示清空 |
| `photos` | array | 是 | 1–50 条，按下标写入 `sort_num` |
| `photos[].file_id` | uint64 | 二选一 | 本次上传的文件行；文件必须存在、为图片，且上传者在本园拥有老师角色 |
| `photos[].file_url` | string | 二选一 | 兼容路径：`http(s)` 绝对 URL；与 `file_id` 同时提交时以 `file_id` 的文件行为准 |
| `photos[].thumb_url` | string | 否 | 非空时必须是 `http(s)` URL |
| `photos[].width` / `height` | int | 否 | 非负；未知写 0 |
| `photos[].source_media_id` | uint64 | 否 | 从成长记录选图（`growth_media.id`）：必须同园同班、未软删、为图片；URL 以成长记录为准 |
| `photos[].student_ids` | uint64[] | 否 | 照片级关联幼儿。**不传 = 从成长记录自动带出**；传数组（含 `[]`）= 显式覆盖 |

> **文件来源优先级**：`source_media_id` > `file_id` > `file_url`。
> 从成长记录选图时服务端按 `file_url` 反查 `files` 回填 `file_id`（同一 URL 命中多行视为歧义，留空）；
> 并用该记录的 `growth_log_students` 自动写入照片级幼儿，显式 `student_ids` 优先。

### 响应（201）

```json
{
  "id": 12,
  "status": "draft"
}
```

### 错误

| 场景 | HTTP | code |
| --- | --- | --- |
| 标题为空/超 30、分类非法、日期非法、照片数不在 1–50、URL 非 http(s)、尺寸为负 | 422 | 42200 |
| `file_id` 不存在/非图片/上传者无本园老师角色 | 422 | 42200 |
| `source_media_id` 越权（跨班/跨园）、已软删、非图片 | 422 | 42200 |
| 参与者或照片级幼儿不属于本园本班 | 422 | 42200 |
| 未绑定老师班级 | 403 | 40300 |
| 服务端异常 | 500 | 50000 |

---

## PUT `/teacher/albums/:id`

> 编辑相册。请求体与创建完全一致。已发布相册保存后立即对家长生效；状态不变。
>
> **照片替换语义**：提交清单即最终清单。
> - `photos[].id` 属于本相册 → 原地更新并保留照片 ID（端上正在展示的 ID 继续有效）；
> - 不带 `id` → 新建照片；
> - 未提交的既有照片 → 从相册移除（硬删照片行及其幼儿关联）；
> - 提交其他相册的 `photos[].id` → 整单拒绝 422。
>
> **参与者替换语义**：`participants` 与照片同义——提交清单即最终清单；
> 不传或传 `[]` 表示清空相册级参与者（不会保留上一次的值）。照片级
> `student_ids` 同样按清单整体替换，不传仍表示「从成长记录自动带出/不关联」。
>
> `photo_count` 与 `cover_url` 在同一事务内按最终清单重算。

### 响应

```json
{
  "id": 12
}
```

### 错误

除与创建相同的校验错误外：

| 场景 | HTTP | code |
| --- | --- | --- |
| 相册不存在/已软删/跨班跨园 | 404 | 40400 |
| `photos[].id` 不属于本相册或重复提交 | 422 | 42200 |

---

## POST `/teacher/albums/:id/publish`

> 发布草稿或重新上架已下架相册。本班任一老师均可操作，写审计日志
> （`module=albums`、`action=publish`、`target=album`、`detail` 含标题与状态）。
>
> 发布校验：至少 1 张照片；标题 1–30 字、分类为四值之一、相册日期非零。

```json
// Response
{
  "id": 12,
  "status": "published",
  "published_at": "2026-11-01T06:00:00Z"
}
```

| 场景 | HTTP | code |
| --- | --- | --- |
| 相册不存在/已软删/跨班跨园 | 404 | 40400 |
| 已发布再次发布、没有照片、标题/分类/日期不合法 | 422 | 42200 |

---

## POST `/teacher/albums/:id/unpublish`

> 下架已发布相册。下架后家长端列表与详情立即不再返回；清空 `published_at` /
> `published_by`（审计日志保留操作人与时间）。

```json
// Response
{
  "id": 12,
  "status": "unpublished"
}
```

| 场景 | HTTP | code |
| --- | --- | --- |
| 相册不存在/已软删/跨班跨园 | 404 | 40400 |
| 当前不是 `published`（草稿/已下架） | 422 | 42200 |

---

## DELETE `/teacher/albums/:id`

> 软删除相册（任意状态）。软删后老师详情与家长端均不可见；相册行保留 180 天供审计，
> 由运维清理命令回收。写审计日志（`action=delete`）。

```json
// Response
{
  "id": 12
}
```

| 场景 | HTTP | code |
| --- | --- | --- |
| 相册不存在/已软删/跨班跨园 | 404 | 40400 |

---

## 实现约束

- 封面 `cover_url` 与 `photo_count` 是写入时冗余；创建与编辑时在同一事务内按最终照片清单重算，不存在漂移来源。
- 列表的 `cover_urls` 由一次 `album_photos` 批量查询（`album_id IN (...)`，按 `sort_num` 升序）后按相册取前 3 张，不做逐相册查询（禁止 N+1）。
- 详情用两条查询取齐照片与照片级幼儿关联（`album_photo_students` LEFT JOIN `students`），不做逐张 N+1。
- 写路径一律带 `kindergarten_id` 与 `class_id = 当前老师班级`；读路径同样收敛，跨班/跨园返回 404。
- 状态迁移用 `UPDATE ... WHERE status IN (...)` 原子完成，并发重复点击不会写出非法状态。
- 审计写失败不阻断主流程，但会打服务端错误日志（OperationLog 是事后追责依据，不允许静默丢失）。
- `album_students` / `album_photo_students` 的重复提交按唯一键硬删后重插，保证幂等，不留下占用唯一索引的软删行。

## 字段溯源

| 字段 | 来源 |
| --- | --- |
| `status` / `published_at` / `published_by` | design D2/D3（发布流程与状态机） |
| `album_students`（请求名 `participants`） | design D2（相册级参与幼儿；照片未逐张标记时的兜底） |
| `photos[].students`（请求名 `student_ids`） | design D2/D3（照片级幼儿关联；成长记录自动带出、显式覆盖） |
| `photos[].file_id` | design D1（media-assets 引用统一字段；可空，兼容历史 URL） |
| `source_media_id` | design D2/D3（从成长记录选图复用文件行，新建相册引用） |
| `unauthorized_photo_count` | design D4（发布预览提示未授权合影） |

## 本期不支持

家长端浏览（另有家长端班级相册契约）、管理端相册管理、视频、跨班级相册、打包导出。
