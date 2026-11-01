# 管理端 API — 班级相册（只读 + 下架）

> 基础路径：`/api/v1/admin` | 鉴权：`Authorization: Bearer <token>`（管理端账号）
>
> Response 均为 `data` 部分，省略外层 `{ "code": 0, "message": "success" }` 包装。
>
> 对应 OpenSpec `class-album-parent-sharing` 的任务 6.1–6.3（spec「园长只读与下架」）。
> 字段口径与 design D3（状态机）/ D9（管理端只读，无编辑表单）一致。
>
> 编号说明：`21` 已被 `21-calendar.md` 占用，故本文件为 `22-albums.md`
>（管理端页面切片 OS-ALB-8 引用的也是此路径）。

---

## 权限与范围

| 维度 | 规则 |
| --- | --- |
| 园所收敛 | 只返回**当前管理员 JWT 所属园所**的相册；`kindergarten_id` 由服务端从鉴权上下文取，不接受客户端传参 |
| 未绑定园所 | 返回 403 `未绑定园所，请重新登录` |
| 班级 | 列表可按 `class_id` 筛选；详情不要求传班级，园所内任一班均可读 |
| 状态 | 草稿 / 已发布 / 已下架都可查看（园长只读），列表可用 `status` 筛选 |
| 写操作 | **只有下架**（`POST /:id/unpublish`）。不提供创建、编辑、删除端点，也不存在绕过本组修改相册的其它 admin 路由 |
| 软删 | 已软删相册在列表与详情中均不可见（与不存在同样 404） |

> 权限码：列表/详情走 `growth:view`，下架走 `growth:manage`（复用成长记录权限码，
> 不新增 `album:*`；与 `leaves → attendance:view` 的既有近似处理一致）。
> 登记见 `server/internal/middleware/admin_perm.go`。

---

## 接口清单

- `GET /admin/albums` — 本园班级相册列表（分页 + 筛选）
- `GET /admin/albums/:id` — 相册详情（照片 + 参与者）
- `POST /admin/albums/:id/unpublish` — 下架已发布相册（幂等）

---

## GET `/admin/albums`

> 本园全部班级相册，按 `album_date DESC, id DESC` 排序（同日按 id 倒序，顺序稳定）。

```json
// Query: ?page=1&page_size=20&class_id=3&status=published&category=activity

// Response
{
  "list": [
    {
      "id": 12,
      "class_id": 3,
      "class_name": "小一班",
      "title": "秋日郊游",
      "category": "activity",
      "category_label": "活动",
      "album_date": "2026-10-25",
      "photo_count": 12,
      "status": "published",
      "published_at": "2026-10-26T02:30:00Z",
      "creator": { "id": 7, "nickname": "李老师" }
    }
  ],
  "total": 1,
  "page": 1,
  "page_size": 20
}
```

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `id` | uint64 | 相册 ID |
| `class_id` / `class_name` | uint64 / string | 所属班级 |
| `title` | string | 相册标题 |
| `category` | string | `activity` / `daily` / `work` / `festival` |
| `category_label` | string | 分类中文标签，由服务端映射 |
| `album_date` | string | 相册日期 `YYYY-MM-DD`（上海时区） |
| `photo_count` | int | 相册照片数（写入时冗余值） |
| `status` | string | `draft` / `published` / `unpublished` |
| `published_at` | string \| null | 发布时间，RFC3339（UTC）；草稿与已下架为 `null` |
| `creator` | object | 创建相册的老师 `{id, nickname}`；用户行缺失时 `nickname` 为空串 |

筛选参数：

| 参数 | 类型 | 默认 | 说明 |
| --- | --- | --- | --- |
| `page` | int | 1 | 页码 |
| `page_size` | int | 20 | 每页条数，上限 100 |
| `class_id` | uint64 | 不限 | 0/省略表示全部班级 |
| `status` | string | 不限 | 仅接受 `draft` / `published` / `unpublished`；其它值 422 |
| `category` | string | 不限 | 仅接受四个分类值；其它值 422（不静默退化成「全部」） |

> 列表不返回照片与参与者，只给元信息与 `photo_count`，避免管理端列表 N+1。

---

## GET `/admin/albums/:id`

> 相册详情：元信息 + 全部照片 + 相册级参与幼儿。草稿与已下架相册同样可读。

```json
// Response
{
  "id": 12,
  "class_id": 3,
  "class_name": "小一班",
  "title": "秋日郊游",
  "category": "activity",
  "category_label": "活动",
  "album_date": "2026-10-25",
  "photo_count": 12,
  "status": "published",
  "published_at": "2026-10-26T02:30:00Z",
  "creator": { "id": 7, "nickname": "李老师" },
  "photos": [
    {
      "id": 101,
      "file_id": 5001,
      "file_url": "https://cdn.example.com/album/1.jpg",
      "thumb_url": "https://cdn.example.com/album/1_thumb.jpg",
      "width": 1200,
      "height": 900,
      "sort_num": 0,
      "students": [{ "id": 27, "name": "小明", "avatar": "" }]
    }
  ],
  "participants": [{ "id": 27, "name": "小明", "avatar": "" }]
}
```

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `photos` | array | 按 `sort_num, id` 升序；元素字段 `id/file_id/file_url/thumb_url/width/height/sort_num/students` |
| `photos[].file_id` | uint64 \| null | 文件行 ID；历史/兼容路径写入的照片可能为 `null` |
| `photos[].students` | array | 照片级关联幼儿 `{id, name, avatar}`，可能为 `[]`（未逐张标记） |
| `participants` | array | 相册级参与幼儿，照片未逐张标记时的兜底口径 |

> 管理端是园所内部视图，可以展示照片关联幼儿；家长端契约不返回这些字段。

---

## POST `/admin/albums/:id/unpublish`

> 下架已发布相册。成功后 `status=unpublished`、`published_at=null`、`published_by=0`。
> 家长端只查 `status=published`，因此下一次读取立即不可见（design D3/D4）。

```json
// Request: 无 body

// Response（真实状态迁移）
{
  "id": 12,
  "status": "unpublished",
  "changed": true
}

// Response（重复下架，幂等）
{
  "id": 12,
  "status": "unpublished",
  "changed": false
}
```

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `id` | uint64 | 相册 ID |
| `status` | string | 恒为 `unpublished` |
| `changed` | bool | `true`=本次真正下架；`false`=调用前已是 `unpublished`（幂等成功） |

状态与错误口径：

| 当前状态 / 场景 | 结果 |
| --- | --- |
| `published` | 200，`changed=true`，写一条审计 |
| `unpublished` | 200，`changed=false`，**不重复写审计**（状态未变化） |
| `draft` | 422 `只有已发布相册可以下架`，状态不变 |
| 不存在 / 跨园 / 已软删 | 404 `相册不存在`，不区分「无权」与「不存在」 |
| 并发下架 | 原子状态迁移失败后重读：已是 `unpublished` 按幂等成功返回 |

审计（`operation_logs`，best-effort，失败不阻断主流程但服务端记 error 日志）：

| 字段 | 值 |
| --- | --- |
| `module` | `albums` |
| `action` | `unpublish` |
| `target_type` / `target_id` | `album` / 相册 ID |
| `detail` | JSON，含 `title`、`class_id`、`class_name`、`status`（`published->unpublished`） |
| `user_id` / `created_at` | 操作人 / 操作时间 |
| `ip` / `user_agent` | 请求 IP 与 UA（UA 截断 300 字符） |

---

## 禁止的端点

以下能力**不存在**，不要调用（见 `server/internal/app/router.go` 的 albums 组）：

- `POST /admin/albums`（创建）
- `PUT /admin/albums/:id` / `PATCH /admin/albums/:id`（编辑）
- `DELETE /admin/albums/:id`（删除）
- 任何修改照片、参与幼儿的 admin 端点

`TestAdminAlbumRoutesHaveNoEditEndpoint` 对真实路由表做断言；新增写端点必须先改本契约。

---

## 错误

| 场景 | HTTP | `code` | message |
| --- | --- | --- | --- |
| 未登录 / token 失效 | 401 | 40100 | 统一鉴权中间件 |
| 未绑定园所 | 403 | 40300 | `未绑定园所，请重新登录` |
| 无 `growth:view` / `growth:manage` 权限 | 403 | 40300 | 统一权限中间件 |
| `status` / `category` 非法 | 422 | 42200 | `筛选条件无效` |
| 相册 ID 非法（非数字 / 0） | 422 | 42200 | `无效的相册 ID` |
| 相册不存在 / 跨园 / 已软删 | 404 | 40400 | `相册不存在` |
| 草稿下架 | 422 | 42200 | `只有已发布相册可以下架` |
| 其它内部错误 | 500 | 50000 | 各自接口的通用文案 |

---

## 实现与测试

- 服务层：`server/internal/service/admin_album_service.go`（挂在 `AdminService` 上）。
- 仓储：`server/internal/repository/album_repo.go` 的 `AdminList` / `AdminFindByID`。
- Handler：`server/internal/handler/http/admin_album.go`；路由：`server/internal/app/router.go`。
- 测试：`server/internal/service/admin_album_test.go`，覆盖 `TC-NEW-ALB-0018`（查看/下架/审计/
  幂等/跨园隔离）与 `TC-NEW-ALB-0019`（无编辑入口）。
