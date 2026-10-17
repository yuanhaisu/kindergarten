# Design

## Context

- `ListGrowthFeed`（`server/internal/service/parent_portal_service.go:110`）返回条目：`id` `date` `target_type` `content` `performance_tags` `created_at` `creator{id,nickname,avatar}` `class_name`（+`related_students*` 当 `target_type=2`）。**缺** `media` `is_liked` `comments_count`。
- `GrowthFeedDetail`（同文件 `:802`）已返回 `media` `is_liked` `comments_count` `likes_count` `creator`，但**没有** `related_students`。
- `parent/src/pages/growth/index.vue` 的模板字段（`item.teacher` / `item.time` / `item.category` / `item.badge` / `item.photos` / `item.isLiked` / `item.likes` / `item.comments` / `item.month` / `item.teacherAvatarBg|Color`）与上述两处返回**均不匹配**。
- 完整证据与逐字段判定见 `docs/growth-feed-field-review.md`。

## Goals / Non-Goals

**Goals:**

- 列表卡片渲染出设计稿画的每一样：发布者、时间戳、正文、照片、点赞态、评论数
- 契约与实现、模板三者对齐到同一套字段
- 列表与详情的**接口粒度**分清：详情才有的内容归详情

**Non-Goals:**

- 不新增任何数据字段/表（`GrowthDailyLog` 不动）
- 不实现徽章、活动上下文（阻塞于产品决策）
- 不接线点赞/评论的前端交互（`toggleLike` 仍是纯本地状态，另案处理）
- 不改 `performance_tags` 的存储与语义

## 跨端触点清单

| # | 层 | 位置 | 判定 | 说明 |
| --- | --- | --- | --- | --- |
| 1 | Go domain | `server/internal/domain/growth*.go` | **不改** | 已有 `GrowthMedia` / `GrowthLike` / `GrowthComment` / `GrowthLogStudent`，本次纯查询组合，不加字段不建表 |
| 2 | Go repository | — | **不改** | `ParentPortalService` 现状直接用 `s.db` 查询。本次沿用，不抽 repository（抽离属重构，另案） |
| 3 | Go service | `server/internal/service/parent_portal_service.go` | **会改** | `ListGrowthFeed`：补 `media` / `is_liked` / `comments_count` / 完整 `creator`，去 `target_type` / `class_name` / `related_students*`；`GrowthFeedDetail`：补 `related_students` |
| 4 | Go handler | `server/internal/handler/http/parent_portal.go` | **不改** | `ParentGrowthFeed` / `ParentGrowthDetail` 透传 service 返回，无需改动 |
| 5 | Casbin policy | `server/configs/rbac_policy.csv` | **不改** | 无新路由，复用现有 `/parent/growth/feed` 与 `/parent/growth/feed/:id` |
| 6 | router.go | `server/internal/app/router.go` | **不改** | 无新路由、无新分组 |
| 7 | Wire DI | `server/internal/app/wire.go` | **不改** | 不新增 repo/service/handler 类型，无需补 `wire.Bind` |
| 8 | API 契约 | `backend/api/api-parent/04-growth.md` | **会改（契约变更）** | 列表：砍 4 字段、挪 1 字段、补 4 字段；详情：补 `related_students`。字段级清单见下 |
| 9 | 老师端小程序 | `teacher/src/` | **不改** | 本次不涉及老师端 UI；老师端发布记录的字段也未改 |
| 10 | 管理端 | `admin/` | **不改** | 不涉及 |
| 11 | 家长端页面 | `parent/src/pages/growth/index.vue` | **会改** | 5 处字段映射错名；月份分组由 `date` 派生；照片区改用 `media[]` 数组 |
| 12 | 家长端页面 | `parent/src/pages/growth/detail.vue` | **会改（仅当需要展示相关孩子）** | 详情补「相关孩子」呈现；若该页已按现有字段渲染则只补这一处 |
| 13 | design token | `shared-design-tokens/` | **不改** | 照片缩略图、点赞态用现有语义色与圆角 token 即可 |
| 14 | 测试用例目录 | `docs/tests/TEST_CATALOG.md` | **会改** | 登记 TC-NEW-GRW-030 ~ 042（见 tasks.md） |

### tabBar / 跳转核对

**本次不涉及任何页面跳转。**

- 列表 → 详情的 `uni.navigateTo`（`growth/index.vue:goDetail`）已经存在且正确：`pages/growth/detail` **不是** tabBar 页，用 `navigateTo` 正确
- 家长端「成长时光」本身是 tabBar 页（`pages.json` → `tabBar.list` 第 2 项），本次改的是页内渲染与详情页数据，不新增跳转
- 老师端无 tabBar，本次不碰老师端
- 结论：不存在 tabBar 场景误用 `navigateTo` 的风险

## 契约字段级清单（`rules.design`【契约字段溯源】）

### 列表 `GET /parent/growth/feed` 响应条目

| 字段 | 动作 | 设计稿出处 | 为什么不能派生 |
| --- | --- | --- | --- |
| `id` | 保留 | 卡片 `@click="goDetail(item.id)"` | — |
| `date` | 保留 | 「时间节点-3月」月份分组 | — |
| `created_at` | 保留 | 「3月19日 11:30」 | — |
| `content` | 保留 | 卡片正文 | — |
| `media[]` | **补** | 「美术作品」「上课照片」矩形图层（`fills: image`） | 不可派生，需查 `growth_media` |
| `creator.{id,avatar,nickname}` | **补** | 「老师头像」+「李老师」 | 不可派生 |
| `is_liked` | **补** | 「已点赞 / 点赞」 | 不可派生，需查 `growth_likes` 按当前用户 |
| `comments_count` | **补** | 「2条评论」 | 不可派生，需查 `growth_comments` 计数 |
| `likes_count` | **砍** | **设计稿不渲染**（互动栏只有状态文本） | 若未来要显示数字，是**改设计**，不是先塞接口 |
| `media_count` | **砍** | **设计稿不渲染**（用真图非数字） | **可由 `media.length` 派生** → 规则明确要求删 |
| `target_type` | **砍** | **设计稿不渲染** | 服务端可见性过滤的内部态 |
| `class_name` | **砍** | **设计稿不渲染** | 班级只在孩子信息卡，卡片不显示 |
| `related_students*` | **挪走** | 「相关孩子（28人）」在 `teacher-08-成长记录详情页` 的**详情页**语义 | 列表不渲染 → 按【接口粒度】移到详情 |
| `performance_tags`（列表） | **砍** | **设计稿卡片不渲染它**（卡片上是「奖章」不是表现标签） | 列表卡片从未展示，按【契约字段溯源】属多余 |
| `performance_tags`（详情） | **修类型** | `parent/src/pages/growth/detail.vue:92` 的「表现标签」区块按数组渲染；契约详情段也写数组 | 现返回逗号拼接字符串 → 前端会按**字符**迭代。改为 split 成数组。**不动存储与语义**（那属阻塞项 B1） |

### 详情 `GET /parent/growth/feed/:id` 响应

| 字段 | 动作 | 出处 | 备注 |
| --- | --- | --- | --- |
| `related_students` | **补** | `teacher-08-成长记录详情页` 的「相关孩子（28人）」 | 契约已有、实现**缺失**；仅 `target_type=2` 有值 |
| `performance_tags` | **修类型** | `parent/src/pages/growth/detail.vue:92` 按数组渲染；契约写数组 | 现为逗号串 → split 成数组；空串返回空数组而非 `[""]` |
| 其余 | 不动 | — | 无家长端详情设计稿，见下「证据缺口」 |

## 证据缺口（必须挑明）

`design-data/` 下**没有家长端成长详情页的设计稿**（只有 `01-首页` `02-成长时光` `04-亲子任务` `06-成长报告` `08-班级日历` `09-我要请假` `10-联系老师`）。

因此：

- 详情接口的字段**无法逐项对设计稿溯源**，本次只补 `related_students` 一项（有 `teacher-08` 详情设计作代理证据 + 契约已声明）
- 详情里现存的 `breakfast_rating` 等日志评分字段、`sync_to_feed`、`likes_count`，**本次一律不动**——没有设计稿就不能判定它们该不该留，判定需要产品补家长端详情设计
- 这不是疏漏，是**证据不足时的正确停手**。补设计稿后应另起变更做详情契约审查

## Decisions

### D1：在 `ParentPortalService` 内补查询，不抽 repository

**选择**：沿用现有 `s.db` 直查风格，`ListGrowthFeed` 内补齐 `media` / `likes` / `comments` 三个查询。

**理由**：`GrowthFeedDetail` 已经是这个写法（`s.db.WithContext(ctx).Model(&domain.GrowthLike{})...`），保持一致；本次是字段补齐不是架构改造，抽 repository 会把 diff 从「对齐契约」膨胀成「重构」。

**备选与放弃原因**：*抽 `GrowthRepository`* —— 方向对，但触点会多出 `wire.Bind`、provider set、接口定义，与本次「对齐三方」的目标正交。**留给后续重构变更**。

### D2：`media` 一次查全，不逐条 N+1

**选择**：列表页一次 `WHERE log_id IN (?)` 查出当页所有 `growth_media`，按 `log_id` 分组挂载。

**理由**：`ListGrowthFeed` 已有 `Preload("Creator")` 的防 N+1 意识。若每条记录单独查 media/likes/comments，一页 20 条会变成 60+ 次查询。

**备选与放弃原因**：*逐条查询（照抄 `GrowthFeedDetail`）* —— 代码更短，但分页页大小是 20，性能不可接受。

### D3：`is_liked` 与 `comments_count` 同样批量查

**选择**：`is_liked` 一次 `WHERE log_id IN (?) AND user_id = ?` 拿到本人点赞过的 log_id 集合；`comments_count` 一次 `GROUP BY log_id` 计数。

**理由**：同 D2，避免 N+1。

### D4：模板月份分组由 `date` 派生，不新增接口字段

**选择**：前端按 `item.date` 的 `YYYY-MM` 切分组，渲染「YYYY年M月」标记。

**理由**：设计稿的「时间节点-3月」是**呈现层分组**，不是数据属性。契约加 `month` 字段是把派生值塞进接口，违反 `rules.design`【契约字段溯源】里「不能由已有字段派生」那条。

**备选与放弃原因**：*接口返回 `month`* —— 就是被砍掉的 `media_count` 同类问题（冗余派生值）。

### D5：`related_students` 的形状沿用契约

**选择**：`[{id, name, avatar}]`，与 `backend/api/api-parent/04-growth.md` 详情段已声明的一致。

**理由**：契约已经写了这个形状，实现补上即可；`teacher-08` 详情设计只画了「相关孩子（28人）」+ 名字列表，头像是否展示由家长端详情页决定。

## 筛选/渲染链路（ASCII）

```
家长进入「成长时光」（tabBar 页，onShow 触发）
    │
    ▼
parent/src/pages/growth/index.vue  loadData()
    │  fetchGrowthFeed({ page, page_size, student_id })
    ▼
GET /api/v1/parent/growth/feed
    │
    ▼
ListGrowthFeed
    │  1. 查当页 GrowthMedia  WHERE log_id IN (当页 ids)   → 按 log_id 分组
    │  2. 查本人点赞集合       WHERE log_id IN (?) AND user_id = ?
    │  3. 查评论计数           GROUP BY log_id
    │  4. 组装条目：
    │       id, date, created_at, content,
    │       media[], creator{id,avatar,nickname}, is_liked, comments_count
    │     （不再输出 target_type / class_name / related_students / likes_count / media_count）
    ▼
前端渲染
    │  按 item.date 的 YYYY-MM 分组 → 月份标记
    │  item.creator.nickname / created_at / content
    │  v-if item.media.length > 0 → 照片网格
    │  item.is_liked → 点赞态
    │  item.comments_count → 「N条评论」/「评论」
    ▼
点卡片 → uni.navigateTo(/pages/growth/detail?id=)   ← 已有，非 tabBar 页，用 navigateTo 正确
    │
    ▼
GrowthFeedDetail
    │  现有字段 + 补 related_students（仅 target_type=2 时非空）
    ▼
详情页渲染（含「相关孩子」）
```

## AutoMigrate 与迁移核实

**本次不涉及新表/新字段**，纯查询组合与响应形状调整：

- `GrowthDailyLog` 不加列，`AutoMigrate` 无需动作
- 不需要 `dbcheck`，不需要手工 SQL
- 无数据回填

## Risks / Trade-offs

- **[契约删字段对已接入方不兼容]** → 设计稿从不渲染这 4 个字段，且当前前端模板用的是另一套错名（`item.likes` 等），**实际无消费方**。仍按破坏性变更标注在契约文件里，并在契约顶部注明「2026-10 移除」
- **[批量查询当页 id 集合为空]** → `IN (?)` 空切片在 GORM 会生成 `IN (NULL)`。需显式处理「当页无记录则跳过三个查询」
- **[`related_students` 仅 `target_type=2` 有值]** → 前端按可选渲染，全班记录不显示该区块
- **[详情页无设计稿]** → 见「证据缺口」。本次只补一项，不做详情契约全面审查
- **[`performance_tags` 语义错位未解]** → 本次不动，但它和 `badge` 同源（阻塞项 B1），遗留问题在 proposal 里登记

## Migration Plan

1. 后端：`ListGrowthFeed` 补字段 + 批量查询；`GrowthFeedDetail` 补 `related_students`（先合，向后兼容——只增不删）
2. 契约：同步 `backend/api/api-parent/04-growth.md`（与后端同 PR，避免再次漂移），标注移除字段
3. 家长端：修模板 5 处映射 + 月份分组 + 照片数组
4. 详情页：补「相关孩子」
5. 回滚：客户端回滚即可；后端新增字段对旧客户端无害，删掉的字段旧客户端本来也没读

## Open Questions

（无——会影响本次范围的问题都已划出 Out of Scope 或列为阻塞项。详情契约的全面审查等家长端详情设计稿补齐后再做。）
