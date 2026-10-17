# Design

## Context

- `StudentAchievement`（`server/internal/domain/points.go:48`）已有 `StudentID`/`AchievementID`（`uniqueIndex:idx_student_achievement`）/`AchievedAt`，**但全库零写入路径**——只有 `provider.go:79` 的 AutoMigrate 注册 + `parent_portal_service.go` 一处 `Pluck` 读。
- `Achievement` 定义层完整（`Name/Icon/Category/Condition/TargetValue/Points/Status`，按 `kindergarten_id` 隔离），admin CRUD 4 接口已实现，家长 `GET /parent/achievements` 已实现。
- 老师端 `growth/create.vue` 已有 `target_type`（1 全班 / 2 指定学生）与 `student_ids` 选择，授予入口可搭在这里。
- 徽章配色已在 token 中：`$color-semantic-warning` `#CA8A04` / `$color-semantic-warning-bg` `#FEF9C3`，`$color-semantic-info` `#2563EB` / `$color-semantic-info-bg` `#DBEAFE`——与设计稿「小小艺术家」（黄）/「全勤小明星」（蓝）逐值吻合。
- 产品决策 Q1/Q2/Q3 见 `docs/achievement-badge-review.md` §5，本设计不得偏离。

## Goals / Non-Goals

**Goals:**

- 老师能在成长记录发布/详情里把徽章授予指定学生，并可绑定该记录为获奖见证
- 家长在见证记录的卡片与详情看到这枚徽章
- 学生已获徽章数实时反映授予结果
- 跨园所、重复授予、无效见证都被明确拒绝

**Non-Goals:**

- 不做 `Condition`/`TargetValue` 规则引擎与进度制自动授予（P2）
- 不做徽章撤销、授予历史、授予通知
- 不改成就定义的 admin 配置
- 不改 `performance_tags` 语义

## 跨端触点清单

| # | 层 | 位置 | 判定 | 说明 |
| --- | --- | --- | --- | --- |
| 1 | Go domain | `server/internal/domain/points.go` | **会改** | `StudentAchievement` 加 `LogID *uint64`（见证记录，可空）。**纯增列**，AutoMigrate 可覆盖 |
| 2 | Go repository | `server/internal/repository/` | **不改** | `AchievementRepo` 只管 `Achievement` CRUD；授予涉及跨实体（徽章+学生+记录+去重），在 service 层用 `db` 直查（`ParentPortalService` 同款风格），不在本次抽新 repository |
| 3 | Go service | `server/internal/service/achievement_service.go` | **会改** | 加 `db *gorm.DB` 字段（跨实体校验用）；新增 `GrantToStudent`：徽章启用 → 跨园所（徽章/学生）→ 见证记录学生可见 → 去重 |
| 4 | Go service | `server/internal/service/parent_portal_service.go` | **会改** | `ListAchievements` / 记录查询带出 `log_id` 与徽章信息 |
| 5 | Go handler | `server/internal/handler/http/` | **会改** | 老师端授予端点 |
| 6 | Casbin policy | `server/configs/rbac_policy.csv` | **会改** | **新增路由必须同步策略**，否则 403 |
| 7 | router.go | `server/internal/app/router.go` | **会改** | 新增老师端授予路由（挂在 `teacher` 分组） |
| 8 | Wire DI | `server/internal/app/wire.go` | **不改** | 不新增 service 类型，`AchievementService` 已注入 |
| 9 | API 契约 | `backend/api/api-teacher/` | **会改（契约变更）** | 授予请求体与响应 |
| 10 | API 契约 | `backend/api/api-parent/07-points.md` | **会改（契约变更）** | 徽章带 `log_id`；记录卡片/详情带 `badge` |
| 11 | 老师端页面 | `teacher/src/pages/growth/create.vue` | **会改** | 授予 UI（选徽章 + 选学生 + 可选见证） |
| 12 | 老师端页面 | `teacher/src/pages/growth/detail.vue` | **会改** | 已发布记录可补授徽章 |
| 13 | 家长端页面 | `parent/src/pages/growth/index.vue` | **会改** | 卡片渲染徽章 |
| 14 | 家长端页面 | `parent/src/pages/growth/detail.vue` | **会改** | 详情渲染徽章 |
| 15 | design token | `shared-design-tokens/` | **不改** | 语义色已够用（warning/info 各一对） |
| 16 | 测试用例目录 | `docs/tests/TEST_CATALOG.md` | **会改** | 登记 TC-NEW-ACH-001~009、TC-NEW-GRW-010~012 |

### tabBar / 跳转核对

**本次不涉及新页面跳转。**

- 老师端：授予 UI 出现在 `growth/create.vue`（`navigateTo` 栈内页）与 `growth/detail.vue`，均为页内交互，不新增跳转。老师端**无 tabBar**，不存在 switchTab 场景
- 家长端：徽章在 `growth/index.vue`（tabBar 页）卡片内与 `growth/detail.vue` 渲染，纯呈现，不触发跳转
- 结论：无 tabBar 场景误用 `navigateTo` 的风险

## 契约字段级清单（`rules.design`【契约字段溯源】）

| 字段 | 动作 | 设计稿出处 | 为什么不能派生 |
| --- | --- | --- | --- |
| `student_achievements.log_id` | **新增** | 「恭喜小明获得全勤小明星」那条记录正文即获奖贺词，卡片挂同名徽章 | 记录与徽章是**多对多语义**（一条记录可见证多次授予、一枚徽章可多次授予），不可由日期推断 |
| 老师端授予请求 `achievement_id` / `student_ids` / `log_id` | **新增** | 卡片徽章「小小艺术家」「全勤小明星」（**取值来自 `achievements` 表，非本次拟**） | 授予意图是人工动作，不可派生 |
| 家长端卡片 `badge` | **新增** | 卡片右上「奖章」彩底+图标+文字 | 不可派生——徽章是独立授予事件，不是记录属性 |
| `Condition`/`TargetValue`/`progress` | **不改** | — | 属 P2 规则引擎，本次划出范围 |

**接口粒度**：徽章信息挂在**记录条目**上（列表卡片与详情都渲染）；`StudentAchievement` 的完整授予历史**不进**成长 feed 契约——它属于成就页（`GET /parent/achievements`），本次只在其上补 `log_id`。

## 授予链路（ASCII）

```
老师在 growth/create.vue 或 detail.vue
    │  选徽章（来自 GET /teacher/achievements，只列本园已启用）
    │  选学生（来自记录的 target_type=2 学生，或全班时点选）
    │  可选：把「这条记录」标为见证
    ▼
POST /api/v1/teacher/achievements/{id}/grant
    body: { student_ids: [...], log_id?: 123 }
    │
    ▼
AchievementService.GrantToStudent
    ├─ 徽章存在且 status=1？        否 → 400 徽章未启用
    ├─ 徽章.kindergarten_id == 老师 kg？ 否 → 403 跨园所
    ├─ 学生.kindergarten_id == 老师 kg？ 否 → 403 跨园所
    ├─ log_id 给出时：
    │    记录属于同园？ 学生对该记录可见？  否 → 400 见证无效
    └─ 已存在 (student_id, achievement_id)？ 是 → 409 重复授予
    │
    ▼
INSERT student_achievements (student_id, achievement_id, log_id, achieved_at)
    │
    ▼
家长端 growth/index.vue 卡片
    记录有 badge → 彩底+图标+文字渲染
    无 badge    → 不渲染徽章区（布局不变）
```

## Decisions

### D1：`LogID` 设为可空，见证是**可选**语义

**选择**：`LogID *uint64`，允许 NULL。

**理由**：设计稿那条「全勤小明星」是见证式，但老师也可能单纯表彰（「这学期表现很棒」）而不挂具体记录。强制见证会让授予门槛过高。

**备选与放弃原因**：*强制 `LogID` 非空*——把「表彰」窄化成「发奖状」，丢掉一般性表扬场景。

### D2：授予走独立端点，不塞进成长记录创建

**选择**：`POST /teacher/achievements/:id/grant`，与记录创建解耦。

**理由**：授予是**追加动作**——设计稿里那条记录先存在、徽章后挂上；`growth/detail.vue` 也要能补授。塞进创建会让「已有记录补授」无法实现。

**备选与放弃原因**：*创建记录时一并授予*——把两个正交动作耦合，且无法补授。

### D3：徽章类别 → 配色用映射表，不落库

**选择**：`category`（`attendance`/`growth`/…）在前端映射到 token 语义色（warning 系 / info 系），`Achievement` 不存色值。

**理由**：色值属样式语义，SoT 是 `tokens.json`。落库会让每个园所自己配色，破坏设计一致性。设计稿两例（黄=艺术类、蓝=全勤类）正好是 category→色的映射实例。

**备选与放弃原因**：*徽章表存 `color` 字段*——把样式塞进数据模型，与 token 体系冲突。

### D4：`StudentAchievement` 唯一约束已足够，服务层补语义化错误

**选择**：依赖既有 `uniqueIndex:idx_student_achievement` 挡重复，服务层捕获后返回 409 而非 500。

**理由**：DB 约束是最后防线，服务层预检 + 错误翻译让客户端拿到可理解的错误。

## AutoMigrate 与迁移核实

- 本次是**纯增列**（`student_achievements.log_id`），`AutoMigrate` 可覆盖——它只增不改，本次不涉及改列类型/删列
- 上线后必须核实线上库真的变了：

  ```bash
  go run ./cmd/dbcheck student_achievements
  ```

  期望看到 `log_id` 列存在且允许 NULL。若缺失，补手工 SQL：

  ```sql
  ALTER TABLE student_achievements
    ADD COLUMN log_id BIGINT UNSIGNED NULL COMMENT '获奖见证记录 ID';
  ```

  跑完再 `dbcheck` 复核。
- 既有授予记录（若有）的 `log_id` 自然为 NULL——语义正确（非见证式授予）

## Risks / Trade-offs

- **[重复授予]** → DB `uniqueIndex` 兜底 + 服务层 409；测试覆盖 TC-NEW-ACH-004
- **[跨园所授予]** → 服务层显式校验徽章与学生的 `kindergarten_id`；刚修过 `ListAchievements` 的同类泄漏，本次一并覆盖 TC-NEW-ACH-003
- **[见证记录学生不可见]** → 校验记录对该学生可见（`target_type=1` 全班，或 `target_type=2` 含该学生）；否则「见证」是假关联，TC-NEW-ACH-007
- **[老师端是个人号]** → 不影响本功能（无 web-view、无 unionid 依赖），但授予入口在老师端小程序内，需注意个人号包体限制
- **[契约增量性]** → 家长端卡片加 `badge` 是纯增量，旧客户端忽略即可；`GET /parent/achievements` 加 `log_id` 同理

## Migration Plan

1. 后端：增列 + 授予接口 + 查询带 `log_id`（先合，向后兼容）
2. 契约：同步 `api-teacher/` 与 `api-parent/07-points.md`（与后端同 PR）
3. 老师端：授予 UI 上线（此后开始产生真实授予数据）
4. 家长端：卡片渲染徽章
5. 回滚：客户端回滚即看不到徽章；后端 `log_id` 保留不影响旧客户端

## Open Questions

（无——Q1/Q2/Q3 已定，P2 范围已划出。若实现中发现 `category` 取值不足以致设计稿两例配色，再补映射。）
