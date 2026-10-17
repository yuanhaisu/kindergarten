# Design

## Context

现状：`notices.class_id` 单值（`domain/notice.go`），三端发布入参、列表查询、已读统计全部按单班写死。已读率两端口径不一（老师端分母=学生数 `classStudentCount`，管理端分母=去重家长 `COUNT(DISTINCT sp.user_id)`）。可见性查询是 `class_id IN (孩子班级集)`。动机见 proposal.md。

数据模型事实（已核实）：`Class.Grade` 字段已存在，取值经 `class_service.validateGrade` 校验（小班/中班/大班）；无需新枚举、无需新实体。全库无迁移工具，只有 AutoMigrate（只增不改）。

## Goals / Non-Goals

**Goals:**

- 一条通知 N 个班级目标的模型与查询语义正确
- 分母口径统一为「目标范围去重家长数」
- 存量数据迁移后行为等价

**Non-Goals:**

- 不补发布触达、定时调度、提醒（S2）；不加生命周期门禁（S3）；不动分类法与模板归属（S4）
- 不引入「按学生个体圈选」——模型上留出可能（目标是班级集合），但本次不实现
- 不顺手修契约既有漂移（`content_preview`、附件计数等）

## Decisions

### D1 目标模型：关联表 `notice_targets`，不复用 `class_id`、不用 JSON 列

新建 `notice_targets(notice_id, class_id)` 表达 N 对多；`notices.class_id` 停止写入，列保留一个版本周期后手工删除。

- **备选 A：JSON 数组列存 class_ids**——查询要 `JSON_CONTAINS`，无法 JOIN 班级/家长做统计，分母查询会退化成循环；否决
- **备选 B：继续一班一条通知 + 发布时批量创建**——多孩家长合并展示无从谈起（N 条记录），已读率也不再是一条通知一个数；否决
- 关联表方案下，多孩合并是天然的（一条记录），分母是 `JOIN` 一次 `DISTINCT user_id`

```
notices (1) ────< notice_targets >──── classes (N)
                     |
                     └──< 学生 ──< student_parents >── 用户（家长）
```

### D2 年级圈选在发布时展开为班级快照

管理端入参可带年级标记；service 层在发布时刻解析为当时该年级的班级集合，与手工勾选的班级**并集去重**后落 `notice_targets`。不在模型里存「年级目标」。

- 理由：一条通知是历史事实；后续新开班级不应被卷入旧通知的已读率分母
- 备选（存 grade 目标、查询时动态展开）否决：已读率会随班级增减漂移，违反「可核实」

### D3 分母口径：目标范围 `COUNT(DISTINCT sp.user_id)`

统一到一处统计函数，老师端/管理端共用。已读、已确认同为按 `user_id` 去重（现状已如此，保持）。老师端 `classStudentCount` 不再用作分母。

### D4 查询语义：命中即可见（EXISTS / JOIN 目标表）

- 家长可见性：`EXISTS (notice_targets WHERE notice_id=n.id AND class_id IN (孩子班级集))`——一条通知只出一行，多孩合并免费获得
- 老师可见性：`EXISTS (... class_id IN (任教班级集))`
- 管理端列表：JOIN 目标表聚合成目标信息（列表只给摘要，见字段清单）

### D5 迁移策略：新表 AutoMigrate + 手工回填 SQL + 双读过渡不必要

存量库的 `notices` 量级（演示/单园）远小于需要双写的规模，采用「停机窗口内一次性回填」即可；回填幂等（`INSERT ... SELECT ... WHERE NOT EXISTS`）。

## 跨端触点清单

| 层 | 会改？ | 改什么 / 为何不改 |
|---|---|---|
| Go domain `domain/notice.go` | **会改** | 新增 `NoticeTarget` 结构（`NoticeID`+`ClassID`，联合唯一索引）；`Notice.ClassID` 字段标注弃用注释，暂留列 |
| Go repository `notice_repo.go` | **会改** | 列表/详情/统计查询改走目标表；新增分母统计（目标范围去重家长）；回填 SQL 交付 |
| Go service `teacher_portal_service.go` | **会改** | 发布接收多班级（校验均为任教班）；可见性改 EXISTS；列表/详情统计换口径 |
| Go service `parent_portal_service.go` | **会改** | 列表/首页动态/新通知计数改命中判定（一条一行，合并） |
| Go service `notice_service.go`（管理端） | **会改** | 发布接收多班级+年级圈选（展开去重）；列表/详情响应目标信息与分母 |
| Go handler `teacher_portal.go` / `parent_portal.go` / `notices.go` | **会改** | 入参与响应形状随契约调整 |
| Casbin policy | 不改 | 无新路由，沿用 `/notices` 系列既有策略 |
| router.go | 不改 | 无新端点，仅既有端点的请求/响应形状变 |
| backend/api/ `api-teacher/04-notices.md` | **会改（契约变更）** | 字段级清单见下 |
| backend/api/ `api-admin/13-notices.md` | **会改（契约变更）** | 字段级清单见下 |
| backend/api/ `api-parent/02-notice.md` | 不改 | 形状不变；合并与可见性是行为语义，不出新字段（`is_read`/`is_confirmed` 已按用户） |
| teacher 页面 `pages/notice/create.vue` | **会改** | 「接收范围」单选 → 多选（任教班级多选）；跳转逻辑不变 |
| teacher 页面 `pages/notice/index.vue` 等 | 不改 | 列表项结构不变（目标信息单班时展示同一字段） |
| admin 页面 `pages/notices/index.vue` | **会改** | 发布对话框：班级多选 + 年级快捷圈选；列表「发布范围」列改多目标展示 |
| admin 页面 `api/notices.ts` | **会改** | 发布入参与列表响应类型随契约 |
| parent 页面 `pages/notice/index.vue` / `detail.vue` / `index/index.vue` | 不改 | 合并由查询天然产生；详情已是单条。验收时若发现重复卡片才修（行为回归） |
| design token | 不改 | 多选控件沿用现有 picker/checkbox 样式，无新视觉元素 |

**跳转/tabBar 核对**：本变更不新增页面、不改任何跳转。家长端 `pages/notice/index` 非 tabBar 页（tabBar 为 index/growth/pickup/profile），入口跳转维持 `navigateTo`；老师端无 tabBar。**不涉及 switchTab 场景。**

## 契约字段级清单（变更字段逐个溯源）

### `api-teacher/04-notices.md`

| 字段 | 变更 | 溯源 |
|---|---|---|
| 发布/编辑入参 `class_ids` | **新增**（替换单 `class_id`） | 老师端发布通知页「接收范围」区域（`design-data` 老师端发布通知页稿；业务决策 2026-10-09 老师可发多班）。单 `class_id` **删除**——多目标下保留会歧义 |
| 列表项/详情 `class_names` | **替换** `class_id`/`class_name` | UI 对应「接收范围/全体家长」展示文案；集合形态直接来自目标集，无法由单值派生 |

### `api-admin/13-notices.md`

| 字段 | 变更 | 溯源 |
|---|---|---|
| 发布/编辑入参 `class_ids` | **新增**（替换单 `class_id`） | `admin-08-通知公告页.json` 发布通知对话框「发布范围」；业务决策 2026-10-09 |
| 发布入参 `grades` | **新增**（可选，年级圈选，取值 小班/中班/大班） | 业务决策 2026-10-09「园所可发多年级多班」；取值溯源 `class_service.validateGrade` 已有常量，不新增枚举。服务端展开为班级快照，故**不是**列表响应字段 |
| 列表项 `class_names` | **替换** `class_id`/`class_name` | `admin-08` 通知列表「发布范围」列——该列在设计稿里就叫发布范围，多目标后语义就是班级名列表 |
| 详情 `class_names` | **替换** `class_id`/`class_name` | 同上，详情页「发布范围」信息 |

**不进契约的字段**（防夹带）：不加 `target_count`（`class_names.length` 直接派生，反例同 `media_count`）；不加年级回显字段（班级名已含信息）；列表不塞家长名单（详情接口 `read-status` 的事，接口粒度）。

### 响应分母字段的口径修正（形状不变，语义变）

`read_stats.total_count` / `read_rate` / `unread_count`（详情）与列表 `read_rate`：分母语义改为「目标范围去重家长数」。字段名不变故不算形状变更，但属**行为变更**，proposal 已声明；两端数值必须一致。

## Migration Plan

```
1. 部署新代码（AutoMigrate 自动建 notice_targets 表）
2. 执行回填 SQL（幂等，落 docs/migrations/2026-10-09-notice-targets-backfill.sql）：
   INSERT INTO notice_targets (notice_id, class_id, kindergarten_id, created_at, updated_at)
   SELECT n.id, n.class_id, n.kindergarten_id, NOW(), NOW()
   FROM notices n
   WHERE n.deleted_at IS NULL
     AND n.class_id > 0
     AND NOT EXISTS (SELECT 1 FROM notice_targets t
                     WHERE t.notice_id = n.id AND t.class_id = n.class_id);
3. 核实：
   go run ./cmd/dbcheck notice_targets
   对账：SELECT COUNT(*) FROM notices WHERE deleted_at IS NULL AND class_id > 0
         应等于 SELECT COUNT(DISTINCT notice_id) FROM notice_targets
4. 观察一个版本周期后，手工 ALTER TABLE notices DROP COLUMN class_id
   （AutoMigrate 不删列；删除前先跑步骤 2 的对账）
```

**回滚**：回填幂等且 `class_id` 列保留，回滚只需回退代码版本；旧代码读 `class_id` 对单班存量数据仍正确（多班数据回滚后退化为按创建时班级，可接受的降级）。

**AutoMigrate 局限声明**：新表 `notice_targets` 属「增」，AutoMigrate 可建；`notices.class_id` 的删除属「改」，必须手工 SQL + `dbcheck` 核实，已列入步骤 4。

## Risks / Trade-offs

- [多班发布后，老师端列表若仍按「我的当前班」过滤，多班通知的可见性会碎] → 可见性判定改为「命中任教班级集」（D4），不再按单一 `tc.ClassID` 硬过滤
- [分母去重后，已读率可能比现状「学生分母」更低，园所观感波动] → 口径诚实优先（业务决策）；在已读率旁的展示沿用现有百分比格式，不做视觉补偿
- [回填遗漏 → 存量通知消失] → 幂等 SQL + 步骤 3 的行数对账；对账不等则不进步骤 4
- [年级圈选展开时机：若班级在编辑时被改年级，历史通知是否重算] → 不重算（D2 快照语义）；编辑通知的目标集合是显式重选，不做增量
- [老师端「当前班」语义（`tc.ClassID`）与多任教班并存] → 本变更只保证通知按命中判定可见；工作台待办等仍按当前班，不在本次范围
