# Proposal

## Why

家长端「成长时光」列表卡片**看起来在渲染，其实字段全是空的**：模板按 mock 形状写（`item.teacher` / `item.time` / `item.photos`），后端返回的是另一套（`creator.nickname` / `created_at` / `media[]`），两边对不上。同时契约里躺着 5 个设计稿从不渲染的字段，而设计稿要的照片、点赞态、评论数，`ListGrowthFeed` **一个都没返回**。

这是四层漂移：设计稿**超前**、契约**膨胀**、handler **缩水**、模板**错位**。完整逐字段证据见 `docs/growth-feed-field-review.md`。

## What Changes

**本次将修改 `backend/api/`，属于契约变更。**

字段级溯源（`rules.design`【契约字段溯源】要求）：

| 字段 | 设计稿出处 | 动作 |
| --- | --- | --- |
| `id` | 卡片可点进详情 | 保留 |
| `date` | 「时间节点-3月」月份分组节点 | 保留 |
| `created_at` | 「3月19日 11:30」时间戳 | 保留 |
| `content` | 卡片正文 | 保留 |
| `media[]` | 「美术作品」「上课照片」真图网格 | **补**（handler 未返回） |
| `creator.{id,avatar,nickname}` | 老师头像 + 「李老师」 | **补**完整对象（handler 返回但模板没映射） |
| `is_liked` | 「已点赞 / 点赞」状态 | **补**（handler 未返回） |
| `comments_count` | 「2条评论」 | **补**（handler 未返回） |
| `likes_count` | **设计稿不渲染**（互动栏只有状态无数字） | **砍** |
| `media_count` | **设计稿不渲染**（用真图非数字）；`media.length` 可派生 | **砍** |
| `target_type` | **设计稿不渲染**（服务端可见性内部态） | **砍** |
| `class_name` | **设计稿不渲染**（班级只在孩子信息卡） | **砍** |
| `related_students*` | 「相关孩子（28人）」在 `teacher-08-成长记录详情页` 的详情页语义 | **从列表挪到详情** |
| `performance_tags` | 设计稿卡片不渲染它（卡片上是「奖章」）；详情页按数组渲染 | **列表砍、详情修类型**（字符串→数组）。存储与语义不动（那属阻塞项 B1） |

### 本次做四件事

1. **补** `ListGrowthFeed` 返回设计要的 `media[]` / `is_liked` / `comments_count` / 完整 `creator` 对象
2. **砍** 契约里设计不渲染的 `likes_count` / `media_count` / `target_type` / `class_name`
3. **挪** `related_students*` 从列表接口到详情接口 `GET /parent/growth/feed/:id`（`GrowthFeedDetail` 目前**没有**返回它）
4. **修** `parent/src/pages/growth/index.vue` 的 5 处字段映射错名：
   `teacher→creator.nickname`、`time→created_at`、`photos→media[]`（数组非数字）、`month→由 date 派生`、`teacherAvatarBg/Color→creator.avatar`

### 明确不做什么（Out of Scope）

- **`badge`（小小艺术家 / 全勤小明星）** —— 设计稿画了，但 `Achievement`/`StudentAchievement`（`server/internal/domain/points.go`）与 `GrowthDailyLog` **无任何关联**，老师端创建表单也没有徽章入口。**无数据源，阻塞于产品决策**，本次不实现、不造字段
- **`category`（「3月19日 11:30 · 美术课」里的「美术课」）** —— 领域模型无活动上下文字段，老师端创建表单也没有。**无数据源，阻塞于产品决策**，本次不实现、不造字段
- **不改 `performance_tags`** —— 它与设计稿「奖章」语义错位（9 个表现标签 ≠ 小小艺术家），但改它要先定「奖章」是什么，与上一条同源
- **不实现点赞/评论的前端接线** —— `toggleLike` 目前纯本地改状态不调接口，是另一个 bug，本次不修
- **不改 admin、不改老师端小程序 UI**
- **不动 `likes_count` 在 `POST/DELETE .../like` 响应里的出现** —— 那是动作返回值，与 feed 卡片渲染无关，本次不扩面

### 阻塞项（需产品决策，不在本次范围）

| # | 问题 | 选项 |
| --- | --- | --- |
| B1 | 奖章（小小艺术家）怎么落地？ | (a) 把 `Achievement` 关联到 `GrowthDailyLog`，老师端加徽章入口；(b) 从设计稿去掉「奖章」 |
| B2 | 「· 美术课」活动上下文要不要？ | (a) 老师端创建表单加「活动/课程」字段 + 契约补返回；(b) 从设计稿去掉「· 美术课」 |

## Capabilities

### New Capabilities

- `growth-feed`: 家长端成长时光的呈现行为——时间线按月分组呈现成长记录，每条卡片给出发布者、时间戳、正文、照片、点赞态与评论数；点进详情可看到完整记录含相关孩子。后续列表排序、可见性、分页语义的演进都归入本能力。

### Modified Capabilities

（无——`openspec list --specs` 当前为空，本仓库尚未建立任何 capability 基线。本次是首个 capability。）

## Impact

**涉及端**：`parent`（列表/详情呈现）+ Go 后端。**不涉及 `admin`，不涉及 `teacher` 小程序 UI。**

| 层 | 位置 | 变更性质 |
| --- | --- | --- |
| 数据结构 | `server/internal/domain/` | **不改**（已有 `GrowthMedia`/`GrowthLike`/`GrowthComment`/`GrowthLogStudent`，纯查询组合） |
| 数据访问 | — | **不改**（复用 `s.db` 查询；如需拆 repository 层见 design D3） |
| 业务逻辑 | `server/internal/service/parent_portal_service.go` | **会改**：`ListGrowthFeed` 补 4 项返回、去掉 5 项；`GrowthFeedDetail` 补 `related_students` |
| HTTP | `server/internal/handler/http/parent_portal.go` | **不改**（透传 service 结果） |
| Casbin / router / Wire | `rbac_policy.csv` / `router.go` / `wire.go` | **不改**（无新路由、无新类型） |
| 接口契约 | `backend/api/api-parent/04-growth.md` | **契约变更**：列表砍 4 字段 + 挪 1 字段；详情补 `related_students` |
| 家长端 | `parent/src/pages/growth/index.vue` | **会改**：5 处字段映射 |
| design token | `shared-design-tokens/` | **不改** |

**风险**：

1. **契约删字段对已接入方不兼容** —— 但设计稿从不渲染这 4 个字段，前端也从未读取（模板用的是 `item.likes`/`item.comments` 等错名），实际风险低。仍属**破坏性契约变更**，需在契约文件标注
2. **`related_students` 只在 `target_type=2` 时有值** —— 全班记录为空，前端需按可选处理
3. **`GrowthFeedDetail` 补 `related_students` 需查 `growth_log_students`** —— 仅详情路径单次查询，无 N+1
