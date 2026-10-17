# Proposal

## Why

通知的接收范围在数据模型上被钉死为「一条通知 = 一个班级」（`notices.class_id` 单值），与业务终版（2026-10-09 决策）冲突：老师要一次发多班、园所要按年级圈发多班，一条内容发 N 个班今天只能复制 N 条通知——多孩家长看到 N 条重复，已读率分母也随班级切分而失真。目标模型是后续发布流水线、门禁的地基，先改这里。

## What Changes

### 1. 目标模型：一条通知 → N 个班级目标（**BREAKING**）

- `Notice` 的单值班级归属改为班级目标集合；一次发布 = 一条通知 = N 个目标班级
- 发布入参由单 `class_id` 改为班级列表（字段形状以 `backend/api/` 契约为准，本次先改契约再改实现）
- 存量单班通知迁移到新模型后行为不变

### 2. 管理端支持按年级圈选（老师端不需要）

- 管理端发布通知时可按年级（小班/中班/大班，取值沿用 `class_service.validateGrade` 已有常量，不新增枚举）圈选，也可直接点选任意班级
- 年级圈选在**发布时展开为班级快照**：后续在该年级新开的班级不自动加入已发布通知（一条通知是一个历史事实）

### 3. 多孩家长合并展示

- 家长有多个孩子且分属同一通知的多个目标班级时，通知列表与首页动态**只出现一条**
- 读/确认按家长用户计一次，对全部孩子生效

### 4. 已读率分母 = 目标范围去重家长数

- `read_rate` / `total_count` 的分母改为**全部目标班级的家长去重数**（`student_parents.user_id` 去重），不再是单班学生数
- 修正现状两端口径不一：老师端按学生数、管理端按家长数

### 5. 可见范围语义

- 老师：能看到自己任教班级中**任一**为目标的通知（列表/详情/已读统计）
- 家长：任一孩子所在班级命中目标集即可见

## 明确不做什么（out of scope）

- **分类法收敛**（删除 `task` 分类、新增任务引用字段、家长端分类 Tab 整顿）——属 S4，另立变更
- **发布流水线**（定时发送调度、站内消息/微信推送触达、自动/手动提醒）——属 S2，另立变更。本次只保证「落库的目标模型正确」，不补触达
- **生命周期门禁**（已发布不可改删、审核可配、驳回回草稿）——属 S3，另立变更
- **模板园所共享**（现状按 creator 隔离）——属 S4
- 已读名单接口的空壳修复（`unread_students: []`）——属 S2 配套，本次只改统计口径
- 契约里的既有漂移顺手修（家长列表 `content` 全文 vs 契约 `content_preview`、附件计数硬编码 0 等）——非本次目标，避免夹带

## Capabilities

### New Capabilities

- `notice-targeting`: 通知投递目标——一条通知面向哪些班级、老师/家长的可见范围、多孩家长合并、已读率分母口径。可承载未来目标相关的规则演进（如按学生个体圈选），不吞并触达与生命周期语义

### Modified Capabilities

（无——`openspec/specs/` 现有 4 个能力规格均不涉通知）

## Impact

> ⚠️ **本次将修改 `backend/api/`，属于契约变更。**

**涉及端：teacher、parent、admin 三端全部。**（parent 的接口形状预计不变，但可见性与合并行为变）

| 层 | 触点 | 性质 |
|---|---|---|
| 契约 | `backend/api/api-teacher/04-notices.md`（发布入参、列表/详情响应的目标信息） | **契约变更** |
| 契约 | `backend/api/api-admin/13-notices.md`（发布入参加年级/多班级、响应目标信息） | **契约变更** |
| 契约 | `backend/api/api-parent/02-notice.md` | 预计形状不变，仅行为语义变（合并、可见性）；若核对中发现必须动字段，单独列出并溯源 |
| Go domain | `server/internal/domain/notice.go`：目标集合的新结构；`ClassID` 单值列的去留与迁移 | 会改 |
| Go repository | `server/internal/repository/notice_repo.go`：目标查询、分母统计重写 | 会改 |
| Go service | `teacher_portal_service.go` / `parent_portal_service.go` / `notice_service.go`：发布、列表、详情、统计 | 会改 |
| Go handler | `teacher_portal.go` / `parent_portal.go` / `notices.go` | 会改 |
| Casbin / router | 无新路由（沿用既有 `/notices` 系列） | 不改 |
| teacher 页面 | `pages/notice/create.vue` 接收范围改多选；`api/teacher-notices.js` | 会改 |
| admin 页面 | `pages/notices/index.vue` 班级多选 + 年级圈选；`api/notices.ts` | 会改 |
| parent 页面 | 预计不改（合并由数据模型天然产生）；`pages/notice/index.vue` 若出现重复卡片则修 | 视核对 |
| design token | 无新视觉元素，沿用现有选择器/标签样式 | 不改 |
| 数据迁移 | 存量 `notices.class_id` → 目标集合，迁移后行为等价；用 `go run ./cmd/dbcheck` 核实 | 会做 |

**契约字段溯源**（本次契约变更涉及的字段，逐个指认出处）：

| 字段 | 出处 |
|---|---|
| 发布入参的多班级选择 | `design-data/` 老师端发布通知页「接收范围」区域（现为单选下拉，业务决策 2026-10-09 升级为多选）；管理端 `admin-08-通知公告页.json` 「发布通知」对话框 |
| 管理端按年级圈选 | 业务场景：园所发多年级多班级（业务决策 2026-10-09）；年级取值已有权威来源 `class_service.validateGrade`（小班/中班/大班），不新增枚举 |
| 列表/详情响应的目标信息 | 现有响应已有 `class_id`/`class_name` 单值，改为集合属形状修正，与既有 UI「接收范围」「发布范围」列（`admin-08` 通知列表有「发布范围」列）对应 |
| `read_rate`/`total_count` 分母口径 | 业务决策 2026-10-09「已读率分母=目标范围去重家长数」；`admin-08` 有「平均已读率」统计卡，口径必须诚实 |

---

## 归档回填：实现阶段实际触点与设计稿的差异（2026-10-10）

- **`notices.class_id` 未停止写入**：设计 D1 原写「停止写入」，实现改为**冗余写入首个目标班**——回滚到旧代码时按 `class_id` 查仍能看到多班通知的首班（可接受降级），且使回填对账 SQL 的 `class_id > 0` 条件在迁移后持续成立。`docs/migrations/2026-10-09-notice-targets-backfill.sql` 与 design.md Migration Plan 已同步此口径。
- **契约变更实际落点**：管理端类型改动落在 `admin/types/api.d.ts`（`CreateNoticeBody`/`UpdateNoticeBody`/`NoticeItem`/`NoticeDetail`），`admin/api/notices.ts` 与老师端 `api/teacher-notices.js` 均为透传、未改；设计稿把「会改」标在 `api/` 文件上，实际是类型定义文件。
- **多补了两处可见性判定**：家长端 `GetNotice`（详情）与 `ConfirmNotice` 也加了 EXISTS 命中校验（设计只列了列表/首页动态/新通知计数）——否则非目标家长可凭 ID 直读详情与确认。
- **老师端 read-status 补 `confirmed_count`**（契约已有此字段）；通知详情补 `read_stats` 对象（沿用契约形状）。
- **附带触点**：`server/internal/app/provider.go`（AutoMigrate 注册 `NoticeTarget`）、`wire_gen.go`（`TeacherPortalService` 构造签名 +noticeRepo）、`docs/tests/TEST_CATALOG.md`（NTC 模块 13 条 TC）。
- **验证方式**：老师端用 `wechat_automator` 真实 tap（多选两班→发布，落库 #990124 targets={4,5}）；管理端用浏览器（Playwright）圈大班+勾班→发布（落库 #990125 `class_names=["大一班","大二班"]`）；家长端列表 4 条各出一行无重复。验证用临时任教行已清理。

