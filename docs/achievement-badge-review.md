# 成就徽章（奖章）—— 产品需求与设计审查

> **日期**：2026-10-05
> **对象**：成长记录卡片上的「奖章」徽章 + 孩子卡片「获得奖章」统计
> **方法**：product-discovery 假设映射；证据优先级 设计稿 > 领域模型 > API 契约 > 业务叙事
> **状态**：🔍 审查完成 · ✅ P0 已实施 · 📋 P1 待 propose

---

## 0. 修正一处先前误判

早先把 **B1（奖章）与 B2（美术课）并列成「设计超前、无数据源」是错的**。两者性质完全不同：

| | B2 美术课 | B1 奖章 |
| --- | --- | --- |
| 设计稿画了 | ✅ | ✅ |
| 领域模型有 | ❌ 无活动字段 | ✅ `Achievement` + `StudentAchievement` |
| API 契约有 | ❌ | ✅ `api-admin/12-achievements.md`（4 接口已实现）+ `GET /parent/achievements` |
| admin 配置入口 | ❌ | ✅ |
| 授予入口 | — | ❌ **缺失** |
| 性质 | **设计超前**（只有设计画了） | **实现缺一角**（设计+模型+契约三方一致） |

**同时作废**「performance_tags 与奖章语义错位」的结论——见 §2，两者正交。

---

## 1. 硬证据

### 1.1 设计稿（产品意图）

**记录卡片上的徽章** = 彩底 + 图标 + 文字，两例：

| 徽章 | 底色 | 图标色 | 对应 token |
| --- | --- | --- | --- |
| 小小艺术家 | `rgba(254,249,195)` | `rgba(202,138,4)` | `$parent-yellow-bg` / `$parent-yellow` |
| 全勤小明星 | `rgba(219,234,254)` | `rgba(37,99,235)` | `$parent-blue-bg` / `$parent-blue` |

→ **用 design token 语义色认真画的**，不是随手色块。

**孩子卡片四个一级指标**（`design-data/02-家长端-成长时光页.json`）：

```
236  成长记录  │  1286  精彩照片  │  18  获得奖章  │  98%  出勤率
```

→ **奖章与出勤率并列**，一级数据，不是装饰。

**其他设计证据**：
- 「**获奖记录**」是成长时光的筛选 tab 之一
- 有一条记录正文即「恭喜小明在上个月获得全勤小明星称号，整个2月份没有请假，每天都准时到园…」，配「获奖照片」
  → **这条记录本身就是获奖见证**，是 Q2 决策的直接依据

### 1.2 后端实现现状

| 层 | 状态 | 证据 |
| --- | --- | --- |
| `Achievement` 定义 | ✅ 完整 | `Name/Icon/Category/Condition/TargetValue/Points/SortNum/Status`，按 `kindergarten_id` 隔离 |
| admin CRUD | ✅ 已实现 | 契约 `12-achievements.md` 状态「✅ 4 个接口已实现」 |
| `GET /parent/achievements` | ✅ 已实现 | 返回 `earned` / `not_earned` / `earned_count` / `total_count` |
| **`StudentAchievement` 授予** | 🔴 **零写入路径** | 全库仅 `provider.go:79` 注册模型 + `parent_portal_service.go:1174` 一处 `Pluck` 读 |
| 自动授予引擎（`Condition`/`TargetValue`） | ❌ 不存在 | 积分/任务服务零引用 |
| 老师手动授予入口 | ❌ 不存在 | — |
| 成长记录 ↔ 徽章关联 | ❌ 不存在 | — |

### 1.3 契约意图（`backend/api/api-parent/07-points.md`）

```json
{"id":1,"name":"全勤之星","category":"attendance","earned_at":"2026-03-01T08:00:00Z"}
{"id":3,"name":"艺术小天才","category":"growth","progress":{"current":6,"target":10}}
```

→ 明确是**进度制成就**（累积达标获得），`category` 有 `attendance`/`growth` 等维度。

---

## 2. 与 performance_tags 是否重叠？—— **不重叠**

| | `performance_tags`（表现标签） | 奖章（成就） |
| --- | --- | --- |
| 取值 | 9 个固定（乐于助人/积极发言/…） | 按园所配置（全勤之星/艺术小天才/…） |
| 触发 | 老师每条记录可选打标 | 达标获得或老师授予 |
| 粒度 | **记录级**，高频，可多选 | **学生级**，低频，累积 |
| 语义 | 「今天表现怎样」 | 「获得荣誉」 |
| 归属 | `growth_daily_logs.performance_tags` | `student_achievements` |
| 消费方 | 详情页标签行 / 成长报告 top3 | 成就页 / 孩子卡片计数 / 记录卡片徽章 |

**设计稿上两者共存**：老师端创建表单有「表现」区（9 个预设），家长端卡片有「奖章」。是两套正交语义。

---

## 3. 是不是强需求

### 支持
1. **一级指标地位**——与出勤率并列
2. **体系已建成 80%**——定义层、admin CRUD、家长查询全有，只缺授予
3. **幼儿园真实实践**——小红花/全勤宝宝/每月之星是刚需正向激励，情感价值高
4. **稀疏性使其可行**——不像分类法要求每条记录都打标；10 条里 1 条有徽章就有价值
5. **有独立叙事载体**——设计稿那条「获奖贺词」记录证明「获奖见证」场景已被想清楚

### 反对
1. **授予机制零实现**——纯做展示，数据永远是 0
2. **依赖行为改变**——老师/系统必须主动授予
3. **`Condition`/`TargetValue` 规则引擎是另一大块**——契约画了 progress（6/10），需累积计数器

### 判定

> **真实需求，中高优先级，但不是「现在就做完整闭环」的强需求。**
> 它**必须做**（否则设计稿一级指标永远是 0 或假数据，重现 `c596073` 修掉的问题），但要分阶段。

---

## 4. 数据模型怎么关联

| 方案 | 做法 | 代价 | 评价 |
| --- | --- | --- | --- |
| A. 日期关联 | 徽章按 `achieved_at` 落在哪天就在哪天的记录卡片显示 | 最小 | 一条记录可能挂多个无关徽章 |
| **B. 记录显式关联** ⭐ | `StudentAchievement.LogID *uint64`（获奖见证记录） | 中 | 语义清晰，与设计稿两例完全吻合 |
| C. 不挂记录 | 只在成就页 + 孩子卡片显示 | 最小 | 偏离设计稿，但可作 P0/P1 之间的过渡 |

**设计证据倾向 B**：「全勤小明星」挂在正文即获奖贺词的记录上；「小小艺术家」挂在美术作品记录上。两者都是**教师主动把某条记录标为获奖见证**。

---

## 5. 产品决策（已定，2026-10-05）

| # | 问题 | 决策 |
| --- | --- | --- |
| **Q1** | 徽章授予者是谁？ | **老师手动授予**（起步） |
| **Q2** | 徽章与记录怎么关联？ | **`StudentAchievement.LogID` 显式见证记录** |
| **Q3** | 「获得奖章」口径？ | **`StudentAchievement` 累计数** |

---

## 6. 分阶段路线

| 阶段 | 做什么 | 解决什么 | 触点 | 状态 |
| --- | --- | --- | --- | --- |
| **P0** | 孩子卡片「获得奖章」接 `earned_count` 真实数据 | 消除 fabricated data | 1–2 层 | ✅ **已完成** |
| **P1** | 老师端授予入口 + `StudentAchievement` 写入 + `LogID` 关联 + 家长卡片显示徽章 | 跑通「表彰」闭环 | ≥4 层 | 📋 待 `/opsx:propose` |
| **P2** | `Condition`/`TargetValue` 规则引擎（进度制自动授予） | 自动化全勤类徽章 | 独立切片 | ⏸ 等 P1 用起来 |

### P1 的跨端触点预估（propose 时 design 要逐项写）

| 层 | 会改？ | 说明 |
| --- | --- | --- |
| `server/internal/domain/points.go` | **会改** | `StudentAchievement` 加 `LogID *uint64`（迁移只增不改，AutoMigrate 可覆盖） |
| repository / service / handler | **会改** | 授予接口 + 查询带 LogID |
| Casbin / router | **会改** | 新增授予路由需同步 `rbac_policy.csv` |
| Wire | **不改** | 若不新增 service 类型 |
| `backend/api/api-teacher/` | **会改（契约变更）** | 授予请求体 |
| `backend/api/api-parent/` | **会改（契约变更）** | 徽章带 `log_id` |
| `teacher/src/pages/growth/create.vue` / `detail.vue` | **会改** | 授予 UI |
| `parent/src/pages/growth/index.vue` | **会改** | 卡片渲染徽章 |
| design token | **不改** | 语义色已够用 |

---

## 7. P0 实施记录（已完成）

**改动**：`parent/src/pages/growth/index.vue`

| 指标 | 之前 | 现在 | 来源 |
| --- | --- | --- | --- |
| 获得奖章 | 硬编码 `0` | `earned_count` | `GET /parent/achievements` |
| 成长记录 | 硬编码 `0` | `feed.total` | 已在拉的 feed 响应（零额外成本） |
| 精彩照片 | 硬编码 `0` | `0` | 无可靠来源，暂留真 0 |
| 出勤率 | 硬编码 `""` | `""` | 无可靠来源，暂留空 |

**健壮性**：成就接口失败单独吞错（`.catch(() => null)`），不连累 feed 加载——次要指标不该让主内容白屏。

**端到端验证**：灌 2 条成就定义 + 授 1 条 → API 返回 `earned_count=1 / total_count=2`
（earned=小小艺术家，not_earned=全勤小明星）。`build:mp-weixin` ✅ · `check-tokens.mjs` growth/index 0 违规。

---

## 8. 审查中发现的遗留 bug

🔴 **`ParentPortalService.ListAchievements` 跨园所泄漏**

```go
var all []domain.Achievement
s.db.WithContext(ctx).Find(&all)   // ← 缺 kindergarten_id 过滤
```

CLAUDE.md 明写「All business tables include `kindergarten_id` for multi-tenancy」。
当前库里恰好都是 kg4 才没暴露，一旦第二个园所配成就就会串。
触点 1 行（service），属单点修复，按路由表不走 OpenSpec。

**✅ 已修（2026-10-05）**：补 `Where("kindergarten_id = ?", pc.KindergartenID)`。
用跨园所诱饵实测——kg7 埋「别园徽章-不应出现」后，kg4 家长查询 `total_count` 仍为 2、诱饵未出现，隔离生效。
同文件已无同型裸 `Find(&list)` 无 `Where` 的查询。

---

## 9. 输入材料说明

本文档是 **P1 `/opsx:propose` 的输入材料**。propose 时：
- 「为什么做」→ §1 设计证据 + §3 强需求判定
- 「明确不做什么」→ §6 的 P2 划出范围
- design 的跨端触点清单 → §6 的触点预估表
- 决策依据 → §5 的 Q1/Q2/Q3

---

## 10. P1 实施记录（2026-10-06）

已通过 OpenSpec 变更 `add-achievement-grant` 落地，跑通「老师表彰 → 家长见证」闭环。

### 实际触点 vs §6 预估表的差异

| 预估 | 实际 | 差异 |
| --- | --- | --- |
| domain `points.go` 加 `LogID` | ✅ 一致 | 纯增列，`dbcheck` 确认 `idx_student_achievements_log_id` |
| repository **不改** | ✅ 一致 | 但 design 原写「AchievementService 均直查」**不准**——它走 `AchievementRepo`；改为「repo 不动，service 加 `db` 做跨实体校验」 |
| service/handler 会改 | ✅ 一致 | 多抽了 `ListEnabled`（徽章选择器） |
| Casbin **会改** | ❌ **无需改** | `p, teacher, /api/v1/teacher/*, *` 通配已覆盖新路由；tasks 假设不成立 |
| router 会改 | ✅ 一致 | 新增 `GET /teacher/achievements` + `POST .../:id/grant` |
| Wire **不改** | ⚠️ **需重跑 `make wire`** | `AchievementService` 构造函数加了 `db` 参数，`wire_gen.go` 必须重生成 |
| 契约 2 个文件 | ✅ 一致 | 另**新建** `api-teacher/11-achievements.md`（此前老师端无成就契约） |
| 老师端 create/detail | ✅ 一致 | create 发布后授予；detail 可补授 |
| 家长端 index/detail | ✅ 一致 | — |
| design token **不改** | ✅ 一致 | 但**修了一处硬编码**：`.badge-pick` 的 `12rpx` → `$spacing-sm` |

### 语义决策（Q1/Q2/Q3）落地情况

- **Q1 老师手动授予** ✅ —— 独立端点 `POST /teacher/achievements/:id/grant`
- **Q2 `LogID` 显式见证** ✅ —— 实测见证式 `log_id=25` 落库、非见证式 `log_id` 无键
- **Q3 累计数** ✅ —— `earned_count` 随授予 0→1→2

### 端到端验证（真实 HTTP + 真实 MySQL）

设计稿「恭喜小明获得全勤小明星」场景完整复现：

```
授予徽章1 + 绑定见证(log25) → 201
授予徽章2 不绑见证        → 201
家长 feed:  id=25 badge={name:全勤小明星,icon:…,color_kind:info}  ✅
            id=26 badge=—（不渲染）                                  ✅
累计数: earned_count=2
```

| TC | 结果 | TC | 结果 |
| --- | --- | --- | --- |
| ACH-001 授予成功 | ✅ 201 | ACH-006 不绑见证 | ✅ `log_id=null` |
| ACH-002 徽章不存在/未启用 | ✅ **400** | ACH-007 见证学生不可见 | ✅ **400** |
| ACH-003 跨园所/跨班 | ✅ **403**（两变体） | ACH-008 累计数 | ✅ 0→1→2 |
| ACH-004 重复授予 | ✅ **409 非 500** | GRW-096/097/098 徽章渲染 | ✅ |

**验证层级说明**：tasks 原写「单元测试覆盖 TC-ACH-001~009」，但 `GrantToStudent` 是 DB-bound，仓库无 sqlite 测试设施。改用**真实 HTTP + 真实 MySQL** 验证——比 mock 单测更强。纯函数部分（`toBadge`/`badgeColorKind`/徽章条件渲染）有 8 个单测。

### 新发现的既有漂移（非本次引入，已记契约注记）

`GET /parent/achievements`：实现返回 `icon`，契约写 `icon_url`；实现返回 `points` 契约未记；契约有 `description` 实现未返回。需另行对齐。

### 未完成

- TC-ACH-009（未授予时为零）未单独实测
- 老师端/家长端**真实点击验证**未做（需微信开发者工具 + wechat_automator）

---

## 11. 真实点击验证与根因（2026-10-06）

**结论：TC-ACH-001~008 + TC-GRW-096/097/098 全部通过**（真实 HTTP + 真实 UI 点击 + DB 三方确认）。

### 端到端链路（真实 tap/input）

```
点徽章 → grant.achievement_id=1 + 学生选择器展开 + 见证勾选框出现
点学生 → 选中
填内容 → 发布
发布成功 → 自动授予
DB: 记录 id=28 · 授予 {student_id:9, achievement_id:1, log_id:28}   ← 见证绑定正确
```

家长端：见证记录（id=21）渲染 `badge` 且 class 为 `rc-badge--warning`；非见证记录无徽章键。

### 三个根因（systematic-debugging 定位）

| # | 症状 | 根因 | 处置 |
| --- | --- | --- | --- |
| ① | `automator(start)` 报 `ws_ready=false` | **MCP 自检假阴性**——裸 WS 握手返回 101、`evaluate` 完全可用 | 忽略该报错，直接用工具 |
| ② | `module 'api/teacher-achievements.js' is not defined` | **`src/api/` 新增文件不进 IDE 模块注册表**，热构建不够 | 重载 IDE 项目（`dual-ide.ps1 -Kill → -Auto`） |
| ③ | 授予请求 `student_ids=[0]` → 400 | **`fetchTeacherClassStudents` 返回 `student_id`，模板写 `s.id`** → undefined → Go 绑定为 0 | 改为 `s.student_id`，真实点击复验通过 |

### 🔴 真实点击发现的真 bug（单测与 API 验证都碰不到）

`teacher/src/pages/growth/create.vue` 有**两处**同款错误假设：

| 行 | 用途 | 性质 |
| --- | --- | --- |
| 333 | `toggleStudent(s.id)` 记录目标学生选择 | **既有 bug** —— 意味着个体记录（target_type=2）选学生发布一直是坏的 |
| 410 | `toggleGrantStudent(s.id)` 授予学生选择 | 本次引入（继承同一错误假设） |

两处均已改为 `s.student_id`。

> **这是「交互验证必须用 tap」的实证**：单测测纯函数、API 测后端，都碰不到前端数据绑定字段名。只有真实点击才会把 `student_id=0` 顶到后端日志里。

### 关于 `teacher_profiles`（2026-10-06 修正）

先前记的「uid 65 无 `teacher_profiles` 记录」**是误报，不是缺口**：
`ResolveContext` 里该查找是**可选**的（`if err == nil && RealName != ""` 才覆盖显示名），
缺记录只让 `TeacherName` 回退到 `user.Nickname`，**不影响任何功能**。
`TeacherProfile` 本就只由 admin 创建老师 / 跨端身份迁移写入，微信登录自动注册的用户本就不该有。

唯一实际影响：测试账号 uid 65 的昵称「王小明妈妈」会被当老师名显示（P0 时设的家长名）。
已补一行 `teacher_profiles`（`RealName=李老师`）让显示合理——**这是测试数据整理，不是修 bug**。

### 遗留

- TC-ACH-009（未授予为零）由「授予前 earned_count=0」间接覆盖，未单独实测
- 详情页补授（`detail.vue`）的 UI 未做真实点击（create.vue 全流程已覆盖同一逻辑）

---

## 12. 归档触点差异（2026-10-06）

变更 `add-achievement-grant` 已归档至
`openspec/changes/archive/2026-10-06-add-achievement-grant/`，
delta 合并进活基线：

- **新增 capability** `openspec/specs/achievement-award/spec.md`（4 Requirement / 9 Scenario）
- **追加** `openspec/specs/growth-feed/spec.md`（6 → 7 Requirement，新增「卡片呈现获奖徽章」）

**实际触点 vs design.md 清单的差异**：

| design 判定 | 实际 | 差异原因 |
| --- | --- | --- |
| domain 加 `LogID` | ✅ 一致 | — |
| repository **不改** | ✅ 一致 | 但 design 原写「AchievementService 均直查」**不准**（它走 repo）→ 已改为「repo 不动，service 加 `db`」 |
| service/handler 会改 | ✅ 一致 | **多抽了 `ListEnabled`**（徽章选择器，design ASCII 里提到但触点表未列） |
| Casbin **会改** | ❌ **无需改** | `p, teacher, /api/v1/teacher/*, *` 通配已覆盖；tasks 2.2 假设不成立 |
| router 会改 | ✅ 一致 | 新增 2 条路由（design 只提了授予，未提选择器） |
| Wire **不改** | ⚠️ **需 `make wire`** | `AchievementService` 构造函数加 `db`，`wire_gen.go` 必须重生成 |
| 契约 2 文件 | ✅ + **1 新建** | 另建 `api-teacher/11-achievements.md` |
| 老师端 create/detail | ✅ 一致 | **额外修了 2 处 `s.id`→`s.student_id`**（1 处既有 bug） |
| 家长端 index/detail | ✅ 一致 | — |
| design token **不改** | ⚠️ 修了 1 处硬编码 | `.badge-pick` 的 `12rpx` → `$spacing-sm` |

**设计未预见、实施中新增的工作**：
1. `ListEnabled` 徽章选择器接口
2. **真实点击挖出的 `s.id` vs `s.student_id` 字段错配**（既有 bug + 新引入，共 2 处）
3. MCP `start` 的 `ws_ready` 假阴性、`src/api/` 新文件需重载 IDE —— 两个环境坑
4. `teacher_profiles` 缺失（**误报**——该查找本就可选；已补测试档案行）

**遗留**：
- ~~`teacher_profiles` 缺口~~ **误报，已澄清**（可选查找，非缺陷）
- `LikeGrowth` 幂等分支 `likes_count: 0` 硬编码
- `GET /parent/achievements` 的 3 处契约漂移（`icon`/`icon_url`、`points`、`description`）
- 详情页补授 UI 未做真实点击（create.vue 全流程已覆盖同一逻辑）

---

## 13. 真实交互验证挖出的 4 个 bug（2026-10-06）

**四个都不是单测或 API 验证能发现的**——只有 tap 真实点击 + 查库/截图才会暴露。

| # | Bug | 怎么发现 | 处置 |
| --- | --- | --- | --- |
| 1 | **`s.id` vs `s.student_id`**（`create.vue` 两处，含 1 处既有） | tap 后查库发现 `student_ids=[0]` → 400 | 改字段名；既有那处意味着**个体记录选学生发布一直是坏的** |
| 2 | **`loadStudents` 登录时序竞态**（`detail.vue`） | 学生列表永远空 | `loadStudents` 先 `await ensureLogin()` |
| 3 | **`.grant-submit` 仅 21px 高** | `element_info` 量出 `height:21`，低于 44px 触控标准 | 补 `padding: $spacing-lg $spacing-xl`，tap 才落得准 |
| 4 | **徽章名编码损坏** | 截图肉眼见乱码 | 用 UTF-8 hex 直写修数据（`_utf8mb4 0xE585A8...`） |

另有两处 MCP 假阴性/局部故障（均为环境，非功能）：
- `wechat_automator(start)` 的 `ws_ready` 假阴性
- 家长端实例 `captureScreenshot` 持续超时（老师端同机制正常）

### 演示产物

| 文件 | 内容 |
| --- | --- |
| `_verify/demo3-teacher-full-panel.png` | 老师端完整授予面板（修复后的按钮尺寸 + 徽章名） |
| 家长端 | 数据层铁证：仅见证记录 R1 挂徽章（`rc-badge--warning`），其余 5 条无 |

### 家长端仍未闭环的验收

- **视觉截图**：该实例 `captureScreenshot` 故障，未取到像素级证明（数据层已验）
- **点击交互**：点赞/进详情的真实点击未在家长端跑（`toggleLike` 仍是纯本地状态，属另案）

---

## 14. 家长端截图故障的根因与修复（2026-10-06）

### 症状
家长端实例 `wechat_screenshot` / `captureScreenshot` 持续 90s 超时；老师端同一机制正常。

### 排查路径（systematic-debugging）

| 测试 | 结果 | 推论 |
| --- | --- | --- |
| 裸 WS 握手 9420/9421 | 都返回 101 | 不是 WS 层 |
| `evaluate` on 9420 | 正常 | 逻辑层通 |
| `page_data` on 9420 | 120s 超时 | **视图层卡** |
| 换页面再截 | 同样失败 | **实例级，非页面级** |
| 强制 topmost + 前台 | 仍失败 | 不只是遮挡 |
| 老师端此刻再截 | **成功** | **家长端实例特有** |
| `shot.py`（激活+截图同进程） | **成功 0.22s** | 根因是**焦点被抢** |

### 根因（两层）

1. **「先激活窗口、再调 MCP 截图」分两步走不通** —— 中间的终端操作（每次 Bash 调用都 spawn PowerShell）会把焦点抢回 VS Code，窗口被覆盖 → `Page.captureScreenshot` 挂死。`scripts/shot.py` 的注释早已写明，其解法是**激活与截图在同一进程内一次完成**。
2. **`focus-ide.ps1` 是单实例专用** —— `Select-Object -First 1` 从所有 `wechatdevtools` 进程取第一个，**双 IDE 时会聚焦错窗口**，目标仍被遮挡。

### 修复

| 文件 | 改动 |
| --- | --- |
| `scripts/focus-ide.ps1` | 新增 `-Title` 参数（子串匹配 `MainWindowTitle`，匹配在 PowerShell 进程内进行、不受终端编码影响） |
| `scripts/shot.py` | 新增 `--title`；`--title` 命中 `家长/老师` 关键词时**自动切换 CDP 端口**（9222/9223），避免「聚焦 A 窗口、截 B 画面」 |

**用法**：

```bash
python scripts/shot.py --title 家长端 out.png   # CDP 9222
python scripts/shot.py --title 老师端 out.png   # CDP 9223
```

**稳定性实测**：连续 3 次 `--title 家长端` 全部成功（52024 字节 / 0.08s）。

### 演示产物

| 文件 | 内容 |
| --- | --- |
| `_verify/demo7-fixed-shot.png` | **家长端**：R1 卡片带「小小艺术家」徽章 + 两张照片缩略图 + 已点赞 + 2条评论 + 「2026年8月」月分组 |
| `_verify/demo3-teacher-full-panel.png` | **老师端**：完整授予面板（选择徽章 / 授予给 6 学生 / 见证勾选 / 全宽「确认授予」按钮） |

### ✅ 后续核查：所谓的「端口串了」是误判

曾怀疑 CDP 9223 渲染的是家长端内容（疑为 `.ide` 端口混淆遗留）。**哈希核对推翻了这一怀疑**：

```
fec927…  demo7-fixed-shot.png      ← shot.py --title 家长端
fec927…  recheck-parent.png        ← 复现一致 ✅
57d017…  recheck-teacher.png       ← shot.py --port 9223
8c3da3…  demo9-teacher-mcp.png     ← MCP 老师端截图（授予面板）
```

- `shot.py --port 9222` **可复现**家长端截图（哈希一致）
- 端口↔进程映射正确：9222→PID 34008（家长端）/ 9223→PID 32624（老师端）
- 两端 appservice 版本不同（3.17.2 / 3.15.2）佐证确是两个实例

**真正的陷阱是看图工具**：同一个哈希（`57d017`）在两次查看中被呈现成不同画面。**教训：判定两张截图是否同一张，用 `md5sum`，不要靠肉眼比对缩略图。**

⚠️ 另注：MCP 截图（`8c3da3`）与 CDP `captureScreenshot`（`57d017`）对同一页面产出**不同图像**——两条通道的裁剪/时机不同。需要精确对照时以其中一条为准，不要混用。

---

## 15. 家长端验收闭环（2026-10-06）

三项遗留验收全部通过。

| 项 | 结果 | 方法 |
| --- | --- | --- |
| **点卡片进详情** | ✅ 点击 6 张卡片中第 1 张 → `pages/growth/detail`，栈深 2 | CDP 真实点击 |
| **多孩切换** | ✅ 点第 2 个孩子 → `parent_current_student_id` 27→32；feed 6→5 条（R3 缺失，因其只关联 27） | CDP 真实点击 |
| **tabBar 往返 / onShow 重载** | ✅ 切走期间入库 R5 → 切回 `hasR5: true`，feed 5→6 | `wx.switchTab` |

> ⚠️ 第 3 项的诚实边界：`custom-tab-bar` 是**原生组件、不在 DOM 里**，`click.py` 与 `wechat_automator` 的 tap 都够不到。用的是 `wx.switchTab`——**tab 点击最终调的同一个 API**（`uni.switchTab` 同路径）。测的是「往返后是否重载」，未测「tab 按钮是否正确调 switchTab」（一行连线）。

### 顺带产出的工具：`scripts/click.py`

家长端 `wechat_automator` 的 **tap / element_info / page_data 通道持续挂死**（120s 超时；`evaluate` 与 CDP 截图正常——daemon 视图层通道卡死）。仿 `shot.py` 写了 CDP 真实点击工具：

```bash
python scripts/click.py --title 家长端 ".record-card"
python scripts/click.py --title 家长端 ".switch-chip" --nth 1
```

- 激活窗口 + 查元素 + `Input.dispatchMouseEvent`，**单进程完成**（与 `shot.py` 同理，避免焦点被抢）
- `--title` 命中 `家长/老师` 自动切 CDP 端口（9222/9223）
- `--nth N` 点第 N 个匹配

**与 `wechat_navigate` 有本质区别**：后者直接调 IDE 路由 API（reLaunch/switchTab），绕过页面业务代码；`click.py` 派发真实鼠标事件，**走完整 DOM 事件管线、触发 Vue 的 `@click`**——与真机点击同路径。

### 三件工具的分工

| 工具 | 用途 | 何时用 |
| --- | --- | --- |
| `shot.py` | 截图（激活+CDP captureScreenshot） | MCP `wechat_screenshot` 挂死时 |
| `click.py` | 真实点击（激活+CDP Input.dispatchMouseEvent） | MCP `wechat_automator` tap 挂死时 |
| `wechat_automator evaluate` | 读页面数据 / 调 wx API | 一直稳定，首选 |
