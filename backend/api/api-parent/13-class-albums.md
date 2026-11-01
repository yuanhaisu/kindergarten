# 家长端 API — 班级相册

> 基础路径：`/api/v1/parent` | 鉴权：`Authorization: Bearer <token>`（角色 `parent`）
>
> Response 均为 `data` 部分，省略外层 `{ "code": 0, "message": "success" }` 包装。
>
> 对应 OpenSpec `class-album-parent-sharing`（Phase 1）与 `class-album-classwide`
> （Phase 2-A，`scope=mine|all`）。编号说明：任务允许「项目现有编号的下一个」，
> `12` 已被 `12-health.md` 占用，故本文件为 `13`。

---

## 可见性口径

| 维度 | 规则 | 依据 |
| --- | --- | --- |
| 相册状态 | 只返回 `status=published` 且未软删的相册；`draft` / `unpublished` / 已软删一律不可见 | design D3 / spec「只展示已发布相册」 |
| 班级与园所 | 只返回**当前孩子**所在班级、同园所的相册；跨班/跨园与不存在同样返回 404，不泄露存在性 | design D4 / spec「班级与园所隔离」 |
| scope | `mine`（缺省）按「我的孩子」过滤；`all` 不做 my-child 过滤，但要求园所开关 `classwide_enabled=true`，否则 403；其它值 422 | classwide design D2 |
| 我的孩子 | `scope=mine` 时，照片有 `album_photo_students` 就要求当前孩子在其中；照片没有逐张关联时，当前孩子在该相册 `album_students`（相册级参与者）中才可见 | design D4 第 2–3 步 |
| 班内共享撤回 | 照片关联到任何 `allow_class_share=false` 的幼儿、且该幼儿不是当前家长的孩子时，**整张照片**隐藏；同相册其他照片不受影响 | design D4 第 4 步、D5 |
| 自己的孩子 | 当前家长自己的孩子始终可见，即使其监护人关闭了班内共享 | design D4 第 4 步 |
| 多监护人 | 同一幼儿多监护人时，任一监护人关闭即视为关闭 | design D5 |
| 多孩 | 所有过滤以**当前孩子**为准（`student_id` 选择，未传时取第一个孩子） | design D4 / ResolveContext |
| 留存 | 毕业（`status=2`）后保留 90 天只读；超过 90 天或 `status_changed_at` 为空视为已过期；退学/转园（`status=3`）立即失去访问 | design D8 |

> 家长端**不返回**照片关联的其他幼儿（姓名/头像/id），避免把班里其他孩子的身份
> 发给每一位家长。需要参与者信息的是老师端契约 `api-teacher/10-album.md`。
>
> Phase 1 的「导出」定义为逐张保存原图（小程序预览/保存能力），本契约不禁用图片 URL，
> 但**不做打包导出**。不提供家长上传、相册内点赞评论等写操作。

---

## 接口清单

- `GET /parent/class-albums` — 已发布相册列表（分页）
- `GET /parent/class-albums/:id` — 相册详情（可见照片）

---

## GET `/parent/class-albums`

> 当前孩子所在班级的已发布相册，按 `album_date DESC, id DESC` 排序。
> 只返回相册元信息与可见张数，**不返回全量照片**。

```json
// Query: ?page=1&page_size=20&student_id=27&scope=mine
// student_id / scope 均可省略：student_id 省略时取该家长的第一个孩子，
// scope 省略时为 mine（Phase 1 行为）

// Response
{
  "list": [
    {
      "id": 12,
      "title": "秋日郊游",
      "category": "activity",
      "category_label": "活动",
      "album_date": "2026-10-25",
      "photo_count": 12,
      "visible_photo_count": 5,
      "cover_urls": [
        "https://cdn.example.com/album/1.jpg",
        "https://cdn.example.com/album/3.jpg",
        "https://cdn.example.com/album/6.jpg"
      ]
    }
  ],
  "total": 1,
  "page": 1,
  "page_size": 20,
  "classwide_enabled": false
}
```

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `id` | uint64 | 相册 ID（详情接口用） |
| `title` | string | 相册标题 |
| `category` | string | `activity` / `daily` / `work` / `festival` |
| `category_label` | string | 分类中文标签，由服务端映射，端上不自行转换 |
| `album_date` | string | 相册日期 `YYYY-MM-DD`（上海时区） |
| `photo_count` | int | 相册总照片数（老师端写入时的冗余值） |
| `visible_photo_count` | int | 当前 `scope` 下可见的照片数（由详情同一套过滤聚合） |
| `cover_urls` | string[] | 封面 URL，最多 3 个，**只取当前 `scope` 可见照片的前 3 张**；无可见照片时为 `[]` |
| `total` | int64 | 符合条件的相册总数 |
| `page` / `page_size` | int | 分页回显；`page_size` 上限 50 |
| `classwide_enabled` | bool | 园所「全班视图」开关；家长端据此决定是否显示「我的孩子 / 全班」切换。无设置行时为 `false` |

> `cover_urls` 不从相册原始照片取：老师端语义是「相册前 3 张」，家长端若照搬，
> 会把别人孩子（或已关闭班内共享的孩子）的脸当封面返回，等同绕过照片级过滤。

---

## GET `/parent/class-albums/:id`

> 相册详情。照片按老师创建时的 `sort_num, id` 顺序返回，过滤只删减、不改变相对顺序。

```json
// Query: ?student_id=27&scope=mine
// scope 缺省 mine；all 仅在 classwide_enabled=true 时可用

// Response
{
  "id": 12,
  "title": "秋日郊游",
  "category": "activity",
  "category_label": "活动",
  "album_date": "2026-10-25",
  "photo_count": 12,
  "visible_photo_count": 5,
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
| `photos[].file_url` | string | 原图 URL（逐张保存用） |
| `photos[].thumb_url` | string | 缩略图 URL，可能为空串 |
| `photos[].width` / `height` | int | 原始像素尺寸，0 表示未知 |

> 详情不返回 `file_id`、`students` 等老师端字段。

---

## 错误

| 场景 | HTTP | 说明 |
| --- | --- | --- |
| 未登录 / token 失效 | 401 | 统一鉴权中间件 |
| 家长未绑定任何孩子 | 403 | `未绑定孩子信息,请联系管理员`（既有家长上下文口径） |
| `scope=all` 但园所开关关闭 | 403 | 不返回全班照片；家长端应退回 `mine` 并隐藏切换 |
| `scope` 不是 `mine` / `all` | 422 | 不静默降级为 mine，避免端上误以为已切到全班 |
| 相册不存在、草稿、已下架、已软删、跨班、跨园 | 404 | 统一 `相册不存在`，不区分「无权」与「不存在」 |
| 留窗外（毕业 >90 天 / 无 `status_changed_at` / 退学/转园） | 列表返回空列表，详情 404 | 不暴露历史相册 |

---

## 留存窗口（design D8）

| 孩子状态 | `status_changed_at` | 访问 |
| --- | --- | --- |
| 0 待入学 / 1 在读 | 不适用 | 正常 |
| 2 毕业 | 距今 ≤ 90 天 | 只读可访问（列表 + 详情 + 逐张保存） |
| 2 毕业 | 距今 > 90 天 | 列表空、详情 404 |
| 2 毕业 | 空（迁移前历史数据） | 列表空、详情 404，不用 `updated_at` 猜 |
| 3 退学/转园 | 任意 | 立即 404 |

---

## 实现与性能约束

- 列表最多 7 条查询（`Count` / 相册 / 照片 / 照片级关联 / 相册级参与者 / opt-out /
  园所开关；无照片时跳过关联查询），与相册数量和照片数量无关；不得逐相册查详情聚合
  `visible_photo_count`。
- 详情最多 6 条查询（相册 / 照片 / 照片级关联 / 相册级参与者 / opt-out / 园所开关；
  `scope=mine` 不读开关时仍为 5 条）。
- 可见性判断逻辑在 `server/internal/service/parent_album_service.go`；
  测试见 `server/internal/service/parent_class_album_test.go`
  （TC-NEW-PCA-0001–0010、TC-NEW-CAW-0003–0007/0011）。
