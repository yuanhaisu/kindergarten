# Proposal

## Why

家长端「成长时光」已经实现「只看孩子 / 看全部」两模式，但列表接口不返回记录范围（个体记录 vs 全班记录）。客户端只能按照片点名过滤，导致个体记录（`target_type=2`）里未点名的照片在「只看孩子」被错误隐藏；用户已明确不接受这个兜底。

## What Changes

- 列表接口 `GET /api/v1/parent/growth/feed` 的每条记录新增 `scope` 字段，取值为 `individual`（个体记录）或 `class`（全班记录）。
- **本次将修改 `backend/api/`，属于契约变更（additive，非破坏性）。**
- 服务层 `assembleFeedItems` 从现有 `GrowthDailyLog.TargetType` 派生 `scope`；不新增数据库列、不改查询、不改可见性规则。
- 家长端 `parent/src/pages/growth/index.vue` 使用 `scope`：
  - 个体记录在「只看孩子」展示全部照片，不按照片点名过滤，标「专属记录」；
  - 全班记录继续按 `media[].student_ids` 过滤，保留「含孩子 / 集体 / 其他孩子 / 班级记录」标识与隐藏照片提示。
- 测试/文档：新增 TC-GRW-099~104，更新 `backend/api/api-parent/04-growth.md`、`docs/growth-feed-field-review.md` 和交接卡。

### Out of Scope

- 不新增/修改数据库表或字段；`scope` 是 `target_type` 的派生展示字段。
- 不改 `GET /parent/growth/feed/:id`（详情已有 `target_type`）。
- 不扩大列表可见性：其他孩子的个体记录仍然不可见；「看全部」只包含全班记录 + 当前孩子的个体记录。
- 不改教师端/管理端 UI，不改 Casbin/router/Wire，不改 design token。
- 不改年份筛选、分页、点赞/评论接口。

## Capabilities

### Modified Capabilities

- `growth-feed`: 列表条目提供记录范围（`individual` / `class`），并明确两模式对个体记录/全班记录的照片处理规则；调整「不返回冗余信息」Requirement，允许两模式必需的 `scope`。

## Contract Field Provenance

| 字段 | 来源 | 动作 | 为什么不能派生 |
| --- | --- | --- | --- |
| `scope` | `GrowthDailyLog.TargetType`（1=全班，2=个体，`parent_portal_service.go` 已有语义）；两模式产品模型见 `docs/face-tagging-delivery-notes.md` §4 和 `docs/design-previews/2026-10-20-parent-growth-two-modes/` | 列表新增 | 个体记录的照片可能未点名，无法从 `media[].student_ids` 推断；没有 `scope` 客户端无法区分个体记录与全班记录 |

`scope` 是展示层字符串，不把内部 `target_type` 直接暴露给客户端；取值为 `individual` / `class`，映射关系固定。

## Impact

| 层 | 位置 | 变更 |
| --- | --- | --- |
| Go service | `server/internal/service/parent_portal_service.go` | `assembleFeedItems` 增加 `scope`；新增纯函数 `feedScope` |
| Go domain/repo/handler | — | 不改 |
| Casbin/router/Wire | — | 不改 |
| API 契约 | `backend/api/api-parent/04-growth.md` | 列表新增 `scope`；字段说明 |
| 家长端 | `parent/src/pages/growth/index.vue` | 用 `scope` 区分个体/全班；个体记录不做照片点名过滤 |
| 测试目录 | `docs/tests/TEST_CATALOG.md` | 新增 TC-GRW-099~104 |
| design token / teacher / admin | — | 不改 |

**风险**：`scope` 是 additive 字段，旧客户端忽略即可；需要与后端、契约、家长端同 PR 上线。无数据库迁移。
