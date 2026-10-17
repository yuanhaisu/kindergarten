# Design

## Context

动机见 proposal.md。影响设计的关键现状约束：

1. **家长端可见性有两套实现**：`verifyGrowthAccess`（`server/internal/service/parent_portal_service.go:1028-1047`，正确）只服务于详情/点赞/评论入口；feed、首页动态、相册、报告四处各写各的查询，其中三处缺过滤。
2. **老师端响应映射是单一函数**：`teacherPortalGrowthMap`（`teacher_portal_service.go:783-785`）被列表（`:765-781`）与详情（`:810-819`）共用，只回 14 个字段；老师端前端按富数据渲染，字段全部落空。
3. **契约已定义目标响应形状**：`backend/api/api-teacher/06-growth.md` 的列表/详情响应含 `creator`、`media_preview`、`likes_count`、`comments_count`、`related_students_count`、`related_students`、`media`——实现未对齐，本次是「追平契约」而非「扩张契约」。
4. **家长端已有正确的批量取数模式**：`ListGrowthFeed` 用「先取 logIDs，再批量查 media/likes/comments 数」避免 N+1（`parent_portal_service.go:160-204`），老师侧补数应复用该模式。
5. **详情接口完全不读媒体**：`GetGrowthLog` 只调映射函数，没有任何 media 查询——这是老师端照片区全空的直接原因。

## Goals / Non-Goals

**Goals:**

- 家长端四个入口采用同一可见性规则，且规则只有一处实现
- 报告接口具备租户与孩子归属校验，只返回已生成完成的报告
- 老师端列表/详情响应与既有契约对齐，老师能看到自己发布记录的完整成果
- 编辑入口语义正确：载入既有内容、提交更新原记录

**Non-Goals:**

- 照片上传落库与挂载（`CreateGrowthMedia` 无 `log_id` 通道）——另案
- 家长端相册/报告页面的视图契约对齐——另案
- 审核门禁与 `sync_to_feed` 查询过滤——另案
- 老师端评论区的楼中楼/点赞数渲染（契约定义了 `replies`/`likes_count`，但本次只对齐「老师能看到的统计」这一层，评论区渲染深度保持现状）
- 任何 `server/internal/domain/` 表结构变更

## Decisions

### D1：可见性规则收敛为单一构造器（家长端）

**决定**：在 `parent_portal_service.go` 内新增一个私有辅助函数，输入「家长上下文 + 可选学生」，输出可直接挂到 `*gorm.DB` 的可见性条件（`kg 过滤 + 班级 IN + target_type 规则`）。四个入口全部改用它：

```
   修改前（四处各写）                    修改后（一处构造，四处挂载）
   ────────────────────                 ────────────────────────────
   ListGrowthFeed      ✅ 正确     ──┐
   ListHomeFeeds       ❌ 无过滤   ──┤    ┌──────────────────────────┐
   ListGrowthMedia     ❌ 缺班级   ──┼──> │ growthVisibilityScope()  │
   GetGrowthReport     ❌ 无租户   ──┤    │  kg + class + target_type │
   new_growth_count    ❌ 无过滤   ──┘    └──────────────────────────┘
```

**为何不用「让各入口调用 verifyGrowthAccess」**：它按单条记录校验（详情语义），而四个入口都是集合查询，逐条校验会退化为 N 次查询。构造器把同一规则表达为查询条件，语义等价且一次查询完成。

**替代方案**（未采纳）：在 repository 层加一个可见性查询方法。家长侧目前完全不使用 `GrowthRepo`（inline 查询），为保持本变更最小，不引入跨层重构。

### D2：报告校验归位到服务层

**决定**：`GetGrowthReport` 在查询前先做归属校验：`studentID` 必须属于 `pc.Students` 且属于 `pc.KindergartenID`；为空时回退 `pc.CurrentStudent`（含 nil 保护）；查询条件加 `kindergarten_id` 与 `status = 1`（已生成）。

**为何用白名单校验而非「查出来再判断」**：报告行本身没有 `kindergarten_id` 归属校验时不可信——先查后判会把他人数据读出内存再丢弃。白名单校验在查询前拒绝，越权请求不触碰数据。

### D3：老师端列表/详情响应按契约补齐，字段级清单如下

**列表**（`GET /teacher/growth/logs`）——列表卡片渲染所需（`media_preview` 只给缩略图以控制响应体积）：

| 字段 | 设计稿溯源 | 说明 |
|---|---|---|
| `creator` | `teacher-08` 记录头部「李老师」；列表卡片作者行 | 契约已有，实现缺失 |
| `media_preview` | `teacher-04` 列表卡片封面图 | 契约已有，实现缺失；只含 `thumb_url`/`type` |
| `likes_count` / `comments_count` | `teacher-08` 互动统计栏「28 个点赞 / 12 条评论」 | 契约已有，实现缺失 |
| `related_students_count` | 列表卡片头像组（`teacher-04`） | 契约已有，实现缺失；只给计数不给数组 |

**详情**（`GET /teacher/growth/logs/:id`）——详情页渲染所需：

| 字段 | 设计稿溯源 | 说明 |
|---|---|---|
| `media` | `teacher-08` 照片区 6 张 | 契约已有，实现**完全不查** |
| `creator` | `teacher-08` 头部「李老师 / 发布于…」 | 契约已有，实现缺失 |
| `related_students` | `teacher-08`「相关孩子（28人）」区块 | 契约已有；`GrowthRelatedStudents` 服务已实现可复用 |
| `related_students_count` | 同上区块的「+N人」总数 | 契约已有，实现缺失 |
| `likes_count` / `comments_count` | `teacher-08` 互动统计栏 | 契约已有，实现缺失；需新增计数查询 |

**从契约删除**：

| 字段 | 删除理由 |
|---|---|
| `shares_count` | 设计稿无任何分享计数渲染；无业务场景 |
| `media_count` | 可由 `media_preview.length` 派生（冗余派生值） |

**不新增任何契约未定义的字段**。契约中定义但本次不实现的字段（`replies`、`likes` 列表的完整 payload），在契约文档中保留定义不动，任务中不涉及。

### D4：老师端列表按真实数据来源裁剪渲染

**决定**：老师端 `index.vue` / `detail.vue` 只渲染后端实际返回且有设计稿来源的内容：

- 去除：`views_count`（无数据源、契约无定义）、`title` 固定文案、`category` 分类标签、`student_count` 的独立来源（统一由 `related_students_count` 承载）、分类筛选 pill（`学习/体能/艺术/其他/今日` 无分类法来源——分类法本身是 `docs/openspec-usage.md` §10 记录的待产品决策阻塞项）、「今日待记录」卡片（依赖后端不存在的 pending/draft 状态）
- 保留并接线：同步状态改用 `sync_to_feed` 字段驱动、封面图用 `media_preview`、作者用 `creator`、统计用 `likes_count`/`comments_count`

**替代方案**（未采纳）：从后端补 `views_count` 等字段让现有 UI 成立。违反契约字段溯源——设计稿的「26 次浏览」与契约无对应定义，属展示性样例而非功能需求。

### D5：编辑态复用发布页表单

**决定**：`create.vue` 的 `onLoad` 接收 `options.id` 区分模式：有 `id` 时加载既有记录（`fetchTeacherGrowthLog`）填入表单，提交调 `updateTeacherGrowthLog`（前端已定义未调用）；无 `id` 保持新建。编辑态下**日期不可改**（后端 `UpdateGrowthLog` 的 Updates 映射本就无 `date` 键，契约 PUT 也未定义日期更新）——表单将日期选择器置为只读展示，避免「改了不生效」的假象。

**为何不扩展后端支持改日期**：契约 PUT 的字段列表不含 `date`；记录日期是「这一天发生了什么」的锚点，改动它会让家长端时间线错位。若未来产品需要，属于独立变更。

### D6：编辑/创建流程

```
   老师点「编辑」                  create.vue onLoad
   ─────────────                  ─────────────────
   /pages/growth/create?id=123 ──> options.id 存在？
                                     │           │
                                   是│           │否
                                     v           v
                              fetchTeacherGrowthLog  空白表单
                              填充表单（编辑态）      （新建态）
                                     │
                                     v
                              提交 ──> updateTeacherGrowthLog(id)
                                        （原记录更新，不新建）
```

## 跨端触点清单

| 层 | 会改 / 不改 | 说明 |
|---|---|---|
| Go domain | **不改** | 无表结构变更；`GrowthReport.Status` 等字段已存在 |
| Go repository | **会改（小）** | `growth_repo.go` 可能需要新增「按 logIDs 批量取相关孩子计数/列表」查询；其余复用现有方法 |
| Go service | **会改** | `parent_portal_service.go`：新增可见性构造器 + 改 4 个入口（feed/首页/相册/报告/未读计数）+ 报告校验；`teacher_portal_service.go`：列表/详情响应补齐、编辑支持、相关孩子与计数查询 |
| Go handler | **会改（小）** | `parent_portal.go:253-261` 报告 handler 的参数透传保持不变（校验在服务层）；`teacher_portal.go` 无签名变化 |
| Casbin policy | **不改** | 无新增路由 |
| router.go | **不改** | 无新增路由 |
| Wire DI | **不改** | 无新增构造依赖（新辅助函数为 service 私有方法） |
| `backend/api/` 契约 | **会改（契约变更）** | `api-teacher/06-growth.md`：列表/详情响应字段对齐（D3 清单）+ 删除 `shares_count`/`media_count` |
| 老师端页面 | **会改** | `teacher/src/pages/growth/index.vue`（卡片渲染裁剪+接线）、`detail.vue`（消费 media/creator/统计/相关孩子；编辑入口）、`create.vue`（编辑态）、`teacher/src/api/teacher-growth.js`（接上 `updateTeacherGrowthLog`） |
| 家长端页面 | **不改** | 本次只改数据层（后端），家长端小程序视图无需变更 |
| admin 页面 | **不改** | 不涉及 |
| design token | **不改** | 无新增样式；仅改渲染逻辑与数据源 |

**tabBar 核对**：本次改动的主体是接口与页面内数据渲染。老师端**无 tabBar**（`teacher/src/pages.json` 无 `tabBar` 字段），编辑入口为 `create.vue` 的**页内导航**（从 detail 跳 create），使用 `uni.navigateTo` 即可——老师端不存在 `switchTab` 场景。家长端无页面跳转变更。

**迁移核对**：本变更**不涉及新表/新字段**，无 `AutoMigrate` 语义问题，无需手工迁移 SQL，无需 `go run ./cmd/dbcheck`。

## Risks / Trade-offs

- **可见性收紧后首页动态可能「变少」** → 这是预期行为（修正过度展示）。风险在于家长感知为「内容消失」：缓解——个体记录仍会出现在目标家长首页（D1 的 OR 条件覆盖），正常使用下家长可见范围不变。
- **老师端列表新增批量查询可能引入 N+1** → 缓解：复用家长侧已验证的「先取 ID 再批量查」模式，加一条计数聚合查询；实现后对照 `ListGrowthLogs` 的查询次数做一次确认。
- **`related_students_count` 与详情 `related_students` 的口径可能不一致**（计数用 count 查询、列表用数组长度） → 缓解：两处都基于 `growth_log_students` 的同一关联，任务中加一条一致性校验用例。
- **编辑态与新建态共用表单可能引入状态残留** → 缓解：`onLoad` 中按模式显式重置/填充；编辑态日期只读以对齐后端行为。
- **契约删除 `shares_count`/`media_count` 是对既有文档的破坏性编辑** → 缓解：已在前端确认无消费（列表/详情渲染都不读这两个字段）；proposal 中已显式声明为契约变更。

## Open Questions

- 老师端列表卡片是否需要在后续变更中呈现「相关孩子头像组」（当前只给计数、不渲染头像）——依赖照片关联层的建成（另案），本变更不阻塞。
