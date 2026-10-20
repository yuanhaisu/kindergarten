# Design

## Context

- `ListGrowthFeed`（`server/internal/service/parent_portal_service.go`）的查询已按 `growthLogVisibilityWhere` 返回「孩子所在班级的全班记录 + 明确发给当前孩子的个体记录」；`target_type=1` 为全班，`target_type=2` 为个体。
- `assembleFeedItems` 目前输出 `id` / `date` / `content` / `created_at` / `media[]` / `is_liked` / `comments_count` / `creator` / `badge`，刻意不输出 `target_type`。
- 家长端两模式 UI 已实现，但只能按 `media[].student_ids` 过滤照片。个体记录的照片可能没有点名，导致「只看孩子」错误隐藏记录；用户已明确不接受该兜底。
- `growth-visibility` 已定义可见性边界：其他孩子的个体记录本来就不可见；本次不改。

## Goals / Non-Goals

**Goals:**

- 让家长端能区分个体记录与全班记录，且不暴露内部数字枚举。
- 个体记录在「只看孩子」展示全部照片并标「专属记录」；全班记录继续按照片点名过滤。
- 契约保持最小：只加一个两模式必需字段，不把详情字段塞进列表。

**Non-Goals:**

- 不新增/修改数据库表或字段。
- 不改变 `growth-visibility` 的可见性规则，不把其他孩子的个体记录放进「看全部」。
- 不改详情接口、教师端/管理端、Casbin/router/Wire、design token。
- 不重构列表查询或抽 repository。

## 跨端触点清单

| # | 层 | 位置 | 判定 | 说明 |
| --- | --- | --- | --- | --- |
| 1 | Go domain | `server/internal/domain/growth*.go` | **不改** | `GrowthDailyLog.TargetType` 已存在且语义明确（1=全班，2=个体） |
| 2 | Go repository | — | **不改** | 沿用 `ParentPortalService` 现有查询与 `growthLogVisibilityWhere` |
| 3 | Go service | `server/internal/service/parent_portal_service.go` | **会改** | `assembleFeedItems` 增加 `scope`；新增纯函数 `feedScope(targetType)` |
| 4 | Go handler | `server/internal/handler/http/parent_portal.go` | **不改** | handler 只透传 service 的 map，不感知字段 |
| 5 | Casbin policy | `server/configs/rbac_policy.csv` | **不改** | 无新路由 |
| 6 | router.go | `server/internal/app/router.go` | **不改** | 无新路由 |
| 7 | Wire DI | `server/internal/app/wire.go` | **不改** | 无新 repo/service/handler 类型 |
| 8 | API 契约 | `backend/api/api-parent/04-growth.md` | **会改（契约变更）** | 列表响应新增 `scope`，additive；字段级清单见下 |
| 9 | 家长端页面 | `parent/src/pages/growth/index.vue` | **会改** | `mediaFor` / `isVisibleInChildView` / `ownerKind` / `ownerLabel` 使用 `scope`；个体记录绕过照片点名过滤并标「专属记录」 |
| 10 | 教师端 | `teacher/src/` | **不改** | 本次不涉及教师发布流程与 UI |
| 11 | 管理端 | `admin/` | **不改** | 本次不涉及后台 |
| 12 | design token | `shared-design-tokens/` | **不改** | 只复用现有语义色/间距/圆角 |
| 13 | 测试用例目录 | `docs/tests/TEST_CATALOG.md` | **会改** | 登记 TC-GRW-099~104 |

### tabBar / 跳转核对

**本次不新增任何页面跳转。**

- 「成长时光」是家长端 tabBar 第 2 项；页面内切换两模式不涉及路由。
- 卡片 → 详情仍是 `uni.navigateTo('/pages/growth/detail?id=...')`；详情页不是 tabBar 页，`navigateTo` 正确。
- 老师端没有 tabBar，本次不碰老师端。

## 契约字段清单（`rules.design`【契约字段溯源】）

列表 `GET /parent/growth/feed` 响应条目新增：

| 字段 | 动作 | 来源 | 为什么不能由已有字段派生 |
| --- | --- | --- | --- |
| `scope` | **新增** | `GrowthDailyLog.TargetType`（1=全班，2=个体，服务层已有语义）；两模式产品模型见 `docs/face-tagging-delivery-notes.md` §4 与 `docs/design-previews/2026-10-20-parent-growth-two-modes/` | 个体记录的照片可能未点名，`media[].student_ids` 为空；仅凭照片无法判断个体/全班。必须由记录本身的发送范围派生 |

取值固定为 `individual` / `class`：

- `target_type=2` → `individual`
- `target_type=1` → `class`
- 防御性兜底：其他值按 `class` 处理（当前可见性查询只会返回 1/2，正常不会出现）

不直接返回 `target_type` 的原因：它是服务端内部数字枚举，前端不该耦合；`scope` 是展示层语义，客户端只关心「个体 / 全班」。

## Decisions

### D1：用字符串 `scope`，不恢复 `target_type`

**选择**：列表新增 `scope: "individual" | "class"`。
**理由**：前端需要的是行为语义，不是内部枚举；字符串可读、可扩展，也避免把曾经因「设计不渲染」而删除的 `target_type` 原样塞回。
**备选与放弃**：恢复 `target_type` —— 会把内部枚举暴露给客户端，且重新引入字段审查中已判定为冗余的形状；从 `media[].student_ids` 推导 —— 个体记录照片可能无点名，推导不可靠。

### D2：只改列表，不改详情

**选择**：`scope` 只出现在列表条目；详情继续使用已有 `target_type`。
**理由**：本次需求只服务列表两模式；详情没有新的消费场景，避免无依据的契约扩展。

### D3：前端按 `scope` 决定是否做照片点名过滤

**选择**：
- `scope=individual`：`mediaFor` 返回全部照片；「只看孩子」也展示全部，标「专属记录」。
- `scope=class`：继续按 `media[].student_ids.includes(currentStudentId)` 过滤；保留「含孩子 / 集体 · 含孩子 / 其他孩子 / 班级记录」与隐藏照片提示。
- `scope` 缺失（旧后端/回滚场景）：退回现有按照片点名的保守逻辑，不崩溃。

**理由**：个体记录的所有照片都属于该孩子，不应该因为老师没逐张点名而隐藏；全班记录才需要按点名保护其他孩子隐私。

### D4：服务层派生，不碰数据库

**选择**：`feedScope(targetType)` 纯函数 + `assembleFeedItems` 写 `scope`；不新增列、不改 SQL。
**理由**：`target_type` 已在 `growth_daily_logs` 中；派生成本为零，避免迁移与 AutoMigrate 限制。
**备选与放弃**：新增 `scope` 列 —— 纯冗余，且需要手工迁移与回填。

### D5：测试分两层

**选择**：
- Go 纯函数/组装测试覆盖 `feedScope` 与 `assembleFeedItems` 的 `scope` 输出；
- 家长端用真实数据做 UI 交互验证（个体记录未点名照片、全班混合点名、全班无孩子照片），并跑构建 + token 检查。

**理由**：`scope` 的正确性在服务层可单测；两模式过滤是客户端行为，必须真实点击验证，不能用 MCP 导航代替。

## 状态 / 数据流（ASCII）

```
GET /parent/growth/feed
  -> ListGrowthFeed
       growthLogVisibilityWhere: 全班记录 + 当前孩子的个体记录
  -> assembleFeedItems
       scope = target_type == 2 ? "individual" : "class"
  -> parent/src/pages/growth/index.vue
       viewMode = child:
         scope=individual -> 全部照片 + 「专属记录」
         scope=class      -> media.filter(student_ids includes child)
                             + 「含孩子 / 集体 / 其他孩子 / 班级记录」
       viewMode = all:
         全部记录 + 全部照片
```

`growth-visibility` 不变：其他孩子的个体记录不会因为「看全部」而出现。

## Risks / Trade-offs

- **[契约新增字段]** → additive，旧客户端忽略即可；后端、契约、家长端同一 PR 上线，回滚客户端即可。
- **[未知 target_type]** → `feedScope` 兜底为 `class`；当前查询只返回 1/2，测试覆盖 1/2。
- **[个体记录照片未点名]** → 现在会全部展示，这是个体记录的产品语义；需在 UI 验证中确认不会把其他孩子照片带进来（个体记录本来就只发给当前孩子）。
- **[旧后端/回滚]** → 前端 `scope` 缺失时退回照片点名过滤，保持可用但恢复旧限制。
- **[UI 验证成本]** → 需要真实数据与真实点击；MCP 导航只验证页面能加载，不能证明筛选逻辑正确。

## Migration Plan

无数据库迁移。

1. 服务层：`feedScope` + `assembleFeedItems`，补单测。
2. 契约：`backend/api/api-parent/04-growth.md` 列表段新增 `scope` 与字段说明。
3. 家长端：`parent/src/pages/growth/index.vue` 使用 `scope`，更新注释。
4. 测试目录：登记 TC-GRW-099~104。
5. 验证：`go test ./internal/service/...`、`npm run build:mp-weixin`、token 检查、真实数据 UI 点击。
6. 回滚：客户端回滚到照片点名兜底即可；后端新增字段对旧客户端无影响。

## Open Questions

（无。其他孩子的个体记录是否可见由 `growth-visibility` 决定，本次不扩大可见性。）
