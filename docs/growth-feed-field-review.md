# 家长端成长时光 Feed —— 契约字段必要性审查

> **日期**：2026-10-04
> **对象**：`GET /api/v1/parent/growth/feed`
> **方法**：product-discovery 假设映射（desirability 逐字段打分），证据优先级 设计稿 > 领域模型 > 业务叙事 > 契约文本
> **状态**：🔍 审查完成，**未实施**——实施触点 ≥3 层，按路由表应走 `/opsx:propose`

---

## 0. 结论速览

| 判定 | 数量 | 字段 |
| --- | --- | --- |
| ✅ 必需 | 9 | `id` `date` `created_at` `content` `media[]` `creator{id,avatar,nickname}` `is_liked` `comments_count` |
| ❌ 契约多余（砍） | 5 | `likes_count` `media_count` `target_type` `class_name` `related_students*` |
| 🔄 语义错位 | 1 | `performance_tags`（设计要的是「奖章」，不是表现标签） |
| 👻 设计超前（需产品决策） | 2 | `badge`（小小艺术家）、`category`（美术课） |
| 🐛 模板字段名全错 | 5 | `teacher` `time` `photos` `month` `teacherAvatarBg/Color` |

**四层全在漂，方向各不相同**：设计稿**超前**（画了模型不支持的东西）；契约**膨胀**（塞了设计不渲染的字段）；handler **缩水**（契约写了它没返回）；模板**错位**（字段名对不上还造新字段）。

---

## 1. 目标产出（Outcome）

`growth-records` skill 的模块叙事：

> Teachers log children's daily moments... and share these moments with parents. **Parents engage via likes and comments.**

→ Feed 的价值是**让家长看见孩子的在园瞬间，并产生情感互动**。任何字段必须服务这条，否则是噪音。

## 2. 证据源与权重

| 源 | 性质 | 权重 |
| --- | --- | --- |
| `design-data/02-家长端-成长时光页.json` | 产品意图（设计师画了什么） | **最高** |
| `server/internal/domain/` | 数据来源（什么字段真实存在） | 硬约束 |
| `.claude/skills/growth-records/SKILL.md` | 业务目的 | 校验用 |
| `backend/api/api-parent/04-growth.md` | 契约作者的假设 | **最低**（无署名、无来源） |

> 判定原则：**设计稿是产品意图的唯一硬依据**。契约与实现都要往它靠，而不是反过来拿契约去要求设计。

## 3. 设计稿单条记录卡片的完整元素（硬证据）

从 `design-data/02` 的 `记录1` 子树逐节点提取：

```
记录1
  老师头像                     → creator.avatar
  李老师                       → creator.nickname
  3月19日 11:30 · 美术课        → created_at + ???(美术课)
  奖章「小小艺术家」             → ???(badge)
  [正文内容]                   → content
  美术作品 / 上课照片（真图×2）   → media[]
  已点赞                       → is_liked    ← 只有状态，无数字
  2条评论                      → comments_count
  分享                         → 无字段（纯 UI 动作）
```

> **关键观察**：互动栏里评论带数字（`2条评论`），赞**不带数字**（`点赞`/`已点赞`）。三张卡片一致。这不是漏画。

---

## 4. 逐字段假设映射

**判定标记**：✅ 必需 · ⚠️ 可疑多余 · ❌ 多余 · 🔄 语义错位 · 👻 设计超前

### 4.1 契约里已有的字段

| 契约字段 | 设计稿画了？ | 数据模型有？ | 业务叙事要？ | 判定 |
| --- | --- | --- | --- | --- |
| `id` | ✅ 卡片可点进详情 | ✅ | — | **✅ 必需** |
| `date` | ✅「时间节点-3月」月份分组 | ✅ | — | **✅ 必需**（分组依据） |
| `created_at` | ✅「3月19日 11:30」 | ✅ | — | **✅ 必需** |
| `content` | ✅ 正文 | ✅ | ✅ | **✅ 必需** |
| `media[]` | ✅ 真图网格 | ✅ `GrowthMedia` | ✅「1286 精彩照片」 | **✅ 必需** |
| `creator.{id,avatar,nickname}` | ✅ 头像+「李老师」 | ✅ | — | **✅ 必需** |
| `is_liked` | ✅「已点赞/点赞」 | ✅ `GrowthLike` | ✅ parents engage via likes | **✅ 必需** |
| `comments_count` | ✅「2条评论」 | ✅ `GrowthComment` | ✅ | **✅ 必需** |
| `likes_count` | ❌ **设计只显示状态不显示数字** | ✅ | ⚠️ 叙事说 engage，不是看热度 | **⚠️ 可疑多余** |
| `media_count` | ❌ 用的是真图不是数字 | 可由 `media.length` 派生 | — | **❌ 多余**（派生值） |
| `target_type` | ❌ | ✅ | 服务端可见性过滤用 | **❌ 对前端多余**（服务端内部态） |
| `class_name` | ❌ 卡片不显示班级 | ✅ | 孩子卡片才有班级 | **❌ 多余** |
| `performance_tags` | ❌ 卡片上是「奖章」不是表现标签 | ✅ | ✅ 老师端有 9 个表现标签 | **🔄 语义错位** |
| `related_students*` | ❌ 列表卡片不显示 | ✅ | 详情页才有 | **❌ 列表多余**（该归详情接口） |

### 4.2 模板要的、契约/模型都不给的

| 模板字段 | 设计稿 | 真实来源 | 判定 |
| --- | --- | --- | --- |
| `item.teacher` | ✅「李老师」 | `creator.nickname` | **🔄 字段名错** |
| `item.time` | ✅「3月19日 11:30」 | `created_at` | **🔄 字段名错** |
| `item.photos`（数字） | ✅ 真图 | `media[]` 数组 | **🔄 用错**（设计是图不是数字） |
| `item.month` | ✅「时间节点-3月」 | 可由 `date` 派生 | **🔄 应派生不该是字段** |
| `item.teacherAvatarBg/Color` | ✅ **图片头像** | `creator.avatar` | **🔄 用错**（设计是图，模板做成色块） |
| `item.badge` | ✅「小小艺术家」 | ⚠️ `Achievement`/`StudentAchievement` 在 `points.go`，**与 `GrowthDailyLog` 无关联** | **👻 设计超前** |
| `item.category` | ✅「美术课」 | ❌ 领域模型无活动字段；老师端创建表单也没有 | **👻 设计超前** |

---

## 5. 对抗性拷问

### Q1：`likes_count` 真需要吗？

**设计稿给了反证**——三张记录卡片的互动栏全是 `已点赞`/`点赞`，**没有任何一处显示数字**。评论有数字，赞没有。

→ **契约多余**。要砍。若产品想显示「28 人赞」，那是**改设计**，不是先塞进接口。

### Q2：`performance_tags` 是设计要的「奖章」吗？

**不是，差得很远**：

| | 取值 | 来源 |
| --- | --- | --- |
| `performance_tags` | 乐于助人 / 积极发言 / 遵守纪律 / 动手能力强 / 专注力强 / 有礼貌 / 爱干净 | 老师端创建表单「表现」区，9 个预设 |
| 设计稿「奖章」 | 小小艺术家 · 全勤小明星 | **无来源** |

「小小艺术家」不在表现标签里。**契约返回了设计不渲染的东西，设计渲染的东西契约不给**——双向错位，不是单向漏字段。

### Q3：「美术课」需要新字段吗？

设计画在时间戳后面，但：

- `GrowthDailyLog` 无活动字段
- 老师端创建表单（`design-data/teacher-04-成长记录页.json`）只有 饮食/睡眠/情绪/表现/照片/内容——**没有活动/课程选择**
- 契约里「美术课」只出现在 `content` 的示例正文里

→ **设计超前**。要么老师端补活动字段 + 契约补返回，要么设计去掉「· 美术课」。**不该由前端凭空造一个 `category` 字段去凑**。

### Q4：`target_type` / `class_name` / `related_students` 该在列表里吗？

设计稿三张卡片都不显示这三样。`related_students` 是**详情页**语义（「相关孩子（28人）」在 `teacher-08` 详情设计里）。塞进列表是**接口粒度画错**。

### Q5：`media_count` 有用吗？

设计用真图网格（美术作品 + 上课照片），`media.length` 即得。单独返回一个数字是**给前端省一个 `.length`**，代价是多一处可能与 `media` 不一致的冗余。

---

## 6. 实施清单（已执行，见 §6.1 回写）

按上面判定，正确的落地动作分四步：

| # | 动作 | 触点 | 说明 |
| --- | --- | --- | --- |
| 1 | **补** handler 返回 | service + handler | 补上设计要的 `media[]` / `is_liked` / `comments_count` / 完整 `creator` 对象——目前 `ListGrowthFeed` 一个都没返回 |
| 2 | **砍** 契约多余字段 | `backend/api/api-parent/04-growth.md` | 删 `likes_count` `media_count` `target_type` `class_name` |
| 3 | **挪** `related_students*` | 契约 + detail 接口 | 从列表移到 `GET /parent/growth/feed/:id` |
| 4 | **修** 模板字段映射 | `parent/src/pages/growth/index.vue` | 5 处错名：`teacher→creator.nickname`、`time→created_at`、`photos→media[]`、`month→由 date 派生`、`teacherAvatarBg/Color→creator.avatar` |
| 5 | **挂起** `badge` / `category` | — | 等产品决策，见下 |

**产品决策待定项（阻塞）**：

- `badge`（小小艺术家/全勤小明星）：是把 `Achievement` 体系关联到成长记录？还是从设计稿去掉？
- `category`（美术课）：老师端要不要补「活动/课程」字段？还是设计去掉「· 美术课」？

> ⚠️ **步骤 1–4 合起来触点 ≥3 层**（service + handler + 契约 + 前端），按 `CLAUDE.md` 路由表**应走 `/opsx:propose`**，不要直接改。本文档是 propose 的输入材料。

---

## 6.1 实施回写（2026-10-04）

已通过 OpenSpec 变更 `align-parent-growth-feed` 落地。与本审查的差异记录如下：

| 审查判定 | 实施结果 | 差异原因 |
| --- | --- | --- |
| 补 handler 返回 `media[]`/`is_liked`/`comments_count`/`creator` | ✅ 已做 | 抽成纯函数 `assembleFeedItems` 以便单测 |
| 砍列表 `likes_count`/`media_count`/`target_type`/`class_name` | ✅ 已做 | — |
| 挪 `related_students` 到详情 | ✅ 已做 | 实现是「列表删 + 详情补」；详情前端**本就已渲染**该字段，此前只是后端没返回 |
| 修模板 5 处映射 | ✅ 已做 | 时间格式按项目既有惯例 `date · created_at.slice(11,16)`（对齐 `detail.vue`/`pickup`），不是设计稿的「3月19日」 |
| `performance_tags` 语义错位 | ⚠️ **部分处理** | 列表已砍（设计不渲染）；详情修了类型（字符串→数组，此前前端按字符迭代）。**语义本身仍未解**，与 B1 同源 |

**实施中发现、审查未覆盖的 3 处**：

1. **契约详情段漂移**：`target_type`/`sync_to_feed` 实现返回但契约没记；`media_count` 契约记了但实现从未返回。已按「契约=实际返回 + 不留派生值」修正。
2. **`GrowthFeedDetail` 的 `performance_tags` 返回逗号串**，而契约与 `detail.vue:92` 都按数组用 → 详情页会把字符串按**字符**迭代。已修。
3. **`LikeGrowth` 幂等分支返回 `likes_count: 0` 硬编码**（`parent_portal_service.go`），属点赞接线，**本次未修**。

**TC 编号**：审查里的 `TC-NEW-GRW-030~042` 已登记为 `TC-GRW-080~092`（030–074 区间被评论/点赞/审核/统计/边界用例占用）。

**验证完成（2026-10-05 补记）**：TC-GRW-080~092 **全部通过**，三重验证——单元测试 + 真实 HTTP + 小程序页面实例数据（`wechat_automator` auto_port=9420）。

期间发现并修掉一个数据准备陷阱：小程序微信登录的用户（uid 65，openid `oCJZt3fwWq…`）与种子目标不一致，且孩子在 class 4 而种子在 class 6 → `ListGrowthFeed` 按孩子班级过滤，页面空列表。已改为按实际登录账号/班级播种，并把 user 65 接到 class 4 的学生上。**教训：验证前必须确认「小程序登录的是谁、他的孩子在哪个班」，否则空列表会被误判成代码 bug。**

⚠️ **视觉层未验**：渲染层（`page_data`/`screenshot`）在该 IDE 实例上持续挂死，本次验证读的是**页面实例数据**（逻辑层），字段映射/月分组/条件渲染逻辑均正确；缩略图是否真显示需另开实例截图核对。

## 6.2 scope 回写（2026-10-20）

家长端两模式 UI 落地后发现：列表只能按 `media[].student_ids` 过滤照片，个体记录（`target_type=2`）的照片若未点名，会在「只看孩子」被错误隐藏。经 OpenSpec 变更 `add-parent-growth-feed-scope` 修正：

- 列表条目新增 `scope`（`individual` / `class`），由 `GrowthDailyLog.TargetType` 派生；**不新增数据库列、不改查询、不改可见性规则**。
- `scope=individual` 的记录在「只看孩子」展示全部照片并标「专属记录」；`scope=class` 的记录继续按已确认点名过滤。
- 本审查原先「列表不得出现发送范围标识」的判定被修正为：`scope` 是两模式筛选与归属标识的必需字段，不属于冗余；`target_type` / `related_students` 仍不得回填列表。
- 其他孩子的个体记录仍不可见（由 `growth-visibility` 约束），「看全部」只增加全班记录与当前孩子的个体记录。
- 新增 TC-GRW-099~104。

---

## 7. 为什么会出现四层漂移

| 层 | 漂移性质 | 根因 |
| --- | --- | --- |
| 设计稿 | **超前** | 画了数据模型不支持的奖章、活动上下文 |
| 契约 | **膨胀** | 按「社交 feed 接口惯例」加字段，没对设计 |
| handler | **缩水** | 契约写了没实现完就上线 |
| 模板 | **错位** | 按 mock 形状写，接口变了没回头改 |

**共同根因**：没有任何一环要求「字段必须指到设计稿元素或明确业务场景」。

→ 已补进 `openspec/config.yaml` 的**契约字段溯源**规则（`rules.proposal` + `rules.design`），详见 `docs/openspec-usage.md` 第 11 节。

---

## 8. 归档触点差异（2026-10-05）

变更 `align-parent-growth-feed` 已归档至
`openspec/changes/archive/2026-10-05-align-parent-growth-feed/`，
delta 合并进活基线 `openspec/specs/growth-feed/spec.md`（6 Requirement / 13 Scenario）。

**实际触点 vs design.md 清单的差异**：

| design.md 判定 | 实际 | 差异原因 |
|---|---|---|
| Go domain / repository / Casbin / router / Wire：**不改** | ✅ 确实未改 | 与设计一致 |
| service `parent_portal_service.go`：**会改** | ✅ 改了，且**比设计多**：额外抽 `feedMediaItem`/`toFeedMedia`/`assembleFeedItems`/`splitPerformanceTags` 四个纯函数 | 设计未预见「附件字段泄漏」与「可测性」需求 |
| handler / token：**不改** | ✅ 未改 | 与设计一致 |
| 契约 `04-growth.md`：**会改** | ✅ 改了，且**比设计多**：详情段还核出 3 处漂移（补记 `target_type`/`sync_to_feed`、删从未实现的 `media_count`） | 设计阶段未做字段级实测 |
| 家长端 `growth/index.vue`：**会改** | ✅ 改了，**比设计多**：移除 `category`/`badge` 假渲染与其死 CSS | 设计未列入移除项 |
| 家长端 `growth/detail.vue`：**会改（仅相关孩子）** | ✅ 改了，**比设计多**：文案「关联学生」→「相关孩子（N人）」对齐设计稿 | 设计只说「新增区块」，实测发现已有区块 |
| 测试用例目录：**会改** | ✅ 新增 TC-GRW-080~092 | 与设计一致 |

**设计未预见、实施中新增的工作**：
1. `media` 条目泄漏 14 个 GORM 字段 → 抽 `feedMediaItem` 收敛到契约的 5 个
2. `performance_tags` 在详情是逗号串而契约/前端都按数组 → split 修复
3. 契约详情段 3 处漂移 → 按「契约=实际返回 + 不留派生值」修正
4. 种子工具 `cmd/seedgrowthfeed`（幂等 + `-reset`）——设计完全没提，是验证前置所需

**遗留（非本次范围）**：
- `performance_tags` 与设计「奖章」语义错位（与 B1 同源，待产品决策）
- `LikeGrowth` 幂等分支 `likes_count: 0` 硬编码
- 视觉层截图未验（渲染层实例卡死，验证读的是页面实例数据）

---


---

## 9. 后续分支：奖章 / 徽章（已独立成档）

本审查的阻塞项 B1（奖章）经重审后判定为**真实需求**（不是「设计超前」——奖章是设计+模型+契约
三方一致，只缺授予机制），并已完成 **P0**（孩子卡片「获得奖章」接真实 `earned_count`）。

完整分析、产品决策（Q1/Q2/Q3）、P1/P2 路线见 **`docs/achievement-badge-review.md`**。
另：B1/B2 混为一谈是本审查的一处误判，已在该文档 §0 修正；「performance_tags 与奖章语义错位」的结论作废。
