# Proposal

## Why

设计稿在成长记录卡片上画了徽章（「小小艺术家」「全勤小明星」），孩子卡片把「18 获得奖章」列为与出勤率并列的**一级指标**，成长时光还有「获奖记录」筛选维度。产品意图明确：**老师表彰学生，家长在记录上见证这份荣誉**。

后端 `Achievement` / `StudentAchievement` 体系已建成 80%（定义层 + admin CRUD + 家长查询全有），**唯独缺授予链路**——`StudentAchievement` 全库零写入路径。结果是：家长侧永远看到 `earned_count=0`，设计稿的一级指标形同虚设。

设计稿里有一条记录的正文就是「恭喜小明在上个月获得全勤小明星称号，整个2月份没有请假……」，卡片上挂同名徽章——**这条记录本身就是获奖见证**。本变更即实现这个场景。

完整分析见 `docs/achievement-badge-review.md`。

## What Changes

**本次将修改 `backend/api/`，属于契约变更。**

字段级溯源（`rules.design`【契约字段溯源】要求）：

| 字段 | 设计稿出处 | 动作 |
| --- | --- | --- |
| `student_achievements.log_id` | 「恭喜小明获得全勤小明星」那条记录正文即获奖贺词，卡片挂徽章 | **新增**（见证记录关联，可空——非见证式授予留 NULL） |
| 老师端授予请求体（徽章 ID + 学生 ID） | 设计稿卡片徽章「小小艺术家」「全勤小明星」（取值来自 `achievements` 表，非本次拟） | **新增** |
| 家长端卡片 `badge` | 卡片右上「奖章」彩底+图标+文字 | **新增**（形状 `{name, icon, color_kind}`） |

### 做四件事

1. **数据结构**：`StudentAchievement` 加 `LogID *uint64`（获奖见证记录，可空）——纯增列，`AutoMigrate` 可覆盖
2. **后端授予接口**（老师端）+ 家长查询带出 `log_id`；新增路由**必须同步 `rbac_policy.csv`**
3. **老师端授予 UI**：成长记录发布 / 详情里选徽章、选学生、授予
4. **家长端**：成长记录卡片渲染徽章（彩底 + 图标 + 文字，用 design token 语义色）

### 明确不做什么（Out of Scope）

- **不做 `Condition` / `TargetValue` 规则引擎与进度制自动授予** —— 那是 P2，独立切片（契约里的 `progress: {current, target}` 属该阶段）
- **不做成就定义的 admin 配置改动** —— `Achievement` CRUD 已实现
- **不改 `performance_tags`** —— 与徽章是两套正交语义（表现标签=记录级高频打标，徽章=学生级稀疏荣誉）
- **不做「精彩照片」「出勤率」两个孩子卡片统计的接线** —— 无可靠来源，另案
- **不涉及 admin 端 UI**
- **不做徽章撤销 / 历史追溯** —— 只做正向授予

## Capabilities

### New Capabilities

- `achievement-award`: 学生获得成就徽章的授予行为——谁有权授予、授予记录什么、与成长记录的见证关系。随着系统演进，撤销、进度制自动授予、授予历史都归入本能力。

### Modified Capabilities

- `growth-feed`: 成长记录卡片**呈现徽章**（此前卡片只呈现发布者/时间/正文/照片/互动态）。这是既有卡片呈现能力的增量，不是改写既有需求。

## Impact

**涉及端**：`teacher`（授予入口）、`parent`（徽章呈现）、Go 后端、`backend/api/` 契约。**不涉及 `admin`。**

| 层 | 位置 | 变更性质 |
| --- | --- | --- |
| 数据结构 | `server/internal/domain/points.go` | **会改**：`StudentAchievement` 加 `LogID *uint64`（纯增列） |
| 数据访问 | `server/internal/repository/`（若拆）/ 直查 | **会改**：授予写入 + 查询带 `log_id` |
| 业务逻辑 | `server/internal/service/` | **会改**：授予校验（徽章启用、学生属于本园、老师有权限） |
| HTTP | `server/internal/handler/http/` | **会改**：老师端授予端点 |
| Casbin | `server/configs/rbac_policy.csv` | **会改**：新增授予路由必须同步策略 |
| router.go | `server/internal/app/router.go` | **会改**：新路由（老师端 `/teacher/achievements/...`） |
| Wire | `server/internal/app/wire.go` | **不改**（不新增 service 类型，沿用 `AchievementService` 或在 parent_portal_service 内实现） |
| 接口契约 | `backend/api/api-teacher/`、`backend/api/api-parent/` | **契约变更**（见上） |
| 老师端 | `teacher/src/pages/growth/create.vue`、`detail.vue` | **会改**：授予 UI |
| 家长端 | `parent/src/pages/growth/index.vue`、`detail.vue` | **会改**：徽章渲染 |
| 样式 | `shared-design-tokens/` | **不改**（语义色 `$color-semantic-warning/bg`、`$color-semantic-info/bg` 已够用） |

**风险**：

1. **重复授予**——同一学生对同一徽章 `StudentAchievement` 有 `uniqueIndex:idx_student_achievement`，DB 层已挡；服务层需返回明确错误而非 500
2. **跨园所授予**——徽章定义按 `kindergarten_id` 隔离，授予时必须校验徽章与学生同属老师所在园所（近期刚修过 `ListAchievements` 的同类泄漏）
3. **`LogID` 语义**——见证记录必须属于同园所、且该学生可见（`target_type=2` 指向他，或 `target_type=1` 全班）；否则「见证」是假关联
4. **契约增量性**——新增字段为增量（家长端卡片加 `badge`），不删不改名，旧客户端不破坏
