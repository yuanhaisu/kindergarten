# Design

## Context

动机见 `proposal.md` 与 `specs/`。设计所依据的现状事实（本次已逐一核实）：

- `calendar_events` 现为班级级单表（`server/internal/domain/calendar.go`），5 类事件类型为域常量，`is_published`（注释「是否发布并通知家长」）与 `notice_id` 字段存在但**无任何写入/消费链路**
- 家长查询（`calendar_repo.go:FindEvents` → `parent_portal_service.go:ListClassCalendar`）不过滤发布状态、直接返回裸 model rows
- 老师端页面自行按事件计算月历标记点（硬编码类型色映射），`/dots` 接口与契约格式不一致且无页面使用
- 契约 `api-teacher/08-calendar.md` 用 `year/month` 查询参数，实现为 `start/end`（老师端页面按 `start/end` 调用）
- 字典基建就绪：`SysDictType.IsSystem` + `SysDictItem.Extra`（JSON，注释明示「预留存储颜色、图标等 UI 元数据」）
- 园所配置：`kindergarten_app_settings.ExtraConfig`（JSON，与管理端 SystemSettings 块对应）
- 管理端权限机制：RBAC 角色 → 权限码，且**接口→权限码登记表漏登记会使服务启动失败**（`openspec/specs/admin-rbac`）
- 仓库无迁移工具（AutoMigrate 只增不改）；软删除行占唯一索引是已知陷阱

## Goals / Non-Goals

**Goals:**

- 让已实现的日历「形」获得正确的「魂」：可见性、推送、类型、受众四类语义一次收敛
- 课表作为**新的内容供给机制**接入日历，且不拖累老师录入
- 全部新行为可由 `specs/class-calendar` 与 `specs/class-timetable` 的场景直接验收

**Non-Goals（design 层边界）:**

- 不做通用重复规则引擎、不做课表审批流（proposal 已列）
- 不为「变更提示」引入 per-parent 已读状态表（见 D4）
- 不改通知模块的表结构与需求（复用其定向与已读口径）

## 跨端触点清单

| 层 | 改/不改 | 内容 |
| --- | --- | --- |
| Go domain | **会改** | `calendar.go`：+scope +status +source +source_slot +overridden +revised_at +change_summary；新增 `timetable.go`（模板/微调两表）；字典 seed（`calendar_event_type`） |
| Go repository | **会改** | `calendar_repo.go`：按 scope 查询、按 status 过滤、Unscoped 查已覆盖槽位；新增 `timetable_repo.go` |
| Go service | **会改** | `teacher_portal_service.go`（日历语义、草稿、覆盖标记、字典渲染字段）；`parent_portal_service.go`（可见性、变更提示数据）；新增管理端日历/课表 service 与 doctor 日历 service；课表同步 service（物化/滚动） |
| Go handler | **会改** | `teacher_portal.go`、`parent_portal.go`、admin 对应 handler、`doctor` handler 新增 |
| Casbin policy | **不改** | 新路由均落在既有通配（`teacher/*`、`parent/*`、`doctor/*`、admin 通配）之下；园级权限走管理端 RBAC 权限码而非 Casbin CSV |
| `router.go` | **会改** | admin 新增日历/课表组；doctor 新增 calendar 组；teacher 新增课表只读（如需）；不得触碰既有路由形状 |
| 管理端权限登记表 | **会改** | 全部新增管理端接口必须登记权限码（漏登记 = 启动失败）；新增权限码如 `calendar:garden:manage` |
| `backend/api/` 契约（**BREAKING**） | **会改** | `api-teacher/08-calendar.md`、`api-parent/08-class.md`、`api-admin` 新增段、`api-teacher`/`api-parent` 的 dicts 段补 `calendar_event_type`。字段级溯源见下节 |
| teacher 页面 | **会改** | `pages/calendar/index.vue`（字典驱动着色、草稿标记、园级只读）；`calendar/create.vue`（状态+通知开关；移除无设计出处的封面/重要性控件）；`calendar/detail.vue`（保存确认弹窗）；新增 `pages/doctor/calendar.vue`（保健医发布园级事件） |
| parent 页面 | **会改** | `pages/calendar/index.vue`（顶部变更提示、类型标签字典驱动） |
| admin 页面 | **会改** | 新增「园级日历管理」与「课表管理」（年级模板+班级微调+开关+手动触发同步）；`constants/menus.ts` 加菜单与权限码；角色页/seed 建立 4 个园长角色 |
| design token | **不改** | 日历类型色是**数据驱动语义色**（存字典 `Extra`），不进 `tokens.json`；页面布局/间距仍走既有 token，无新增 token |
| 数据库迁移 | **会改** | 见「Migration Plan」；`AutoMigrate` 只能加列不能改约束，全部关键 DDL 走手工 SQL + `cmd/dbcheck` 核实 |
| 设计稿 | **需补** | 家长端变更 banner、老师端保存确认弹窗、admin 两个新页、医生发布页 —— 均无 `design-data/` 出处，走 impeccable 补设计 |

### 契约字段级溯源（BREAKING 部分）

| 字段 | 动作 | 出处（设计稿 UI 元素 或 业务场景） | 不可派生的理由 |
| --- | --- | --- | --- |
| `status`（草稿/发布） | 老师侧新增 | 业务场景：老师保存草稿、老师端列表「草稿」标记（设计稿无此态，需补设计） | 无法由其他字段派生；可见性由它决定 |
| `scope`（班级/园级） | 老师侧新增 | 业务场景：园级事件在老师端需标识且只读（否则编辑按钮点击即失败） | 无法派生 |
| 类型 `label`/`color` | 新增 | `design-data/teacher-06` 图例（5 类型色）与事件卡片 | 字典驱动后不再由客户端硬编码 |
| dots 响应（日期→类型数组） | 修正为原契约格式 | `design-data/teacher-06` 月历标记点按类型着色 | 计数无法着色 |
| `revised`（变更要旨） | 家长侧新增 | 业务场景：家长端变更提示（需补设计） | 无法派生（详情页无旧值） |
| `kindergarten_id`、`creator_id`、`is_published`、`notice_id` | **删（不再外泄）** | 无任何 UI 出处 | — |
| `importance`、`cover` | 不暴露 | 无设计出处且产品未裁决（见 Open Questions） | — |
| `21-calendar.md` 全部字段（园级事件与课表） | 新增 | 业务场景：spec `class-calendar` R4/R6（发布/通知/权限）与 `class-timetable` R2（星期 × 节次 → 课程，增/改/删微调）；管理端两页无设计稿，按 D11 走 impeccable 补设计 | 课表槽位（weekday/period）是能力本身的数据形状，无既有字段可派生 |

**接口粒度**：家长/老师列表接口只含列表卡片渲染所需字段（标题、时间、地点、类型、状态/变更标记）；事件描述全文、创建者等仅详情接口返回。`/dots` 只返回日期与类型集合，不含事件内容。

### tabBar 核对

两小程序端**本次不新增任何跨页跳转**：家长端日历页非 tabBar 页（tabBar 4 项为 index/growth/pickup/profile），既有入口均为 `navigateTo`，保持不变；老师端无 tabBar（不存在 `switchTab` 场景）。医生发布页与老师端日历同为普通页（`navigateTo`）。**结论：不涉及 tabBar 差异风险**。

## Decisions

### D1 受众范围：新增 `scope` 字段（值：`kindergarten` / `class`）

- **决定**：`calendar_events` 增加 `scope`；园级事件 `class_id` 为 **NULL**（列改可空）。查询口径：班级日历 = `scope='kindergarten'` ∪ `(scope='class' AND class_id=本班)`
- **理由**：显式 scope 让意图可读；存量行全部 `default 'class'` 兼容
- **实施修正（2026-10-17）**：原计划 `class_id = 0` 哨兵，实施中发现表上有**真实外键** `fk_calendar_events_class`（→ classes.id, ON DELETE RESTRICT），哨兵插入撞 FK 1452 → 改为 NULL（MySQL 外键不检查 NULL，且保留对真实班级行的 RESTRICT 保护）。见迁移文件第 3b 步
- **备选**：独立园级事件表（查询需 union、维护面翻倍）

### D2 发布状态与推送彻底分离

- **决定**：新增 `status`（`draft`/`published`，默认 `published` 兼容存量）；`is_published` 语义收窄为「是否已在推送中通知」的记录位（不再决定可见性）；家长查询硬过滤 `status='published'`
- **理由**：两件事正交——可见性（持久态）与推送（一次性动作）；当前实现两者纠缠且都没生效
- **备选**：复用 `is_published` 兼任两义（正是现状的病根）

### D3 推送 = 生成一条通知（复用通知模块）

- **决定**：确认推送时调用通知 service 创建一条通知，目标 = 事件受众（班级事件→该班；园级事件→全园班级快照，沿用 notice-targeting 的年级/多班口径）；事件上 `notice_id` 记录**最近一次**推送的通知（溯源用）
- **理由**：已读追踪是产品核心差异点，且通知模块的定向/口径现成；不为「一对多次推送」建关联表（YAGNI）
- **备选**：站内 Message（无已读口径，弱）；自建日历推送表（重复造轮子）

### D4 变更记录：最小实现 + banner 时间窗口（存储粒度 [待定]，先落最小）

- **决定**：写路径计算字段级变更摘要存入 `change_summary`（如「时间 10:00 → 10:30」）+ `revised_at`；家长端日历响应携带「近 N 天（design 默认 7）内被修订且尚未发生」的事件变更要旨，页面顶部渲染 banner；事件发生后或超出窗口自然消失
- **理由**：无需 per-parent 已读状态表即可「显式提示」；过期自然消失避免噪音
- **备选**：per-parent 已读状态（新表，重）；永久显示（噪音）
- **[待定]**：存储粒度升级为完整 revision 表由用户后续决定；届时只换存储，不改行为与验收

### D5 课表两层模型 + 物化 + 覆盖 + 滚动

```
                 ┌──────────────────────────────────────┐
                 │ 年级模板 timetable_templates          │
                 │ (grade × weekday × period × 课程/地点) │
                 └──────────────┬───────────────────────┘
                                │ 叠加
                 ┌──────────────v───────────────────────┐
                 │ 班级微调 timetable_overrides          │
                 │ (class × weekday × period × 增/改/删) │
                 └──────────────┬───────────────────────┘
                                │ 物化（滚动窗口，默认 4 周）
                 ┌──────────────v───────────────────────┐
                 │ calendar_events(source='timetable')   │
                 │ 唯一槽位 = (class_id, event_date,      │
                 │            source_slot)               │
                 └──────────────┬───────────────────────┘
                                │ 老师调整 / 取消
                 ┌──────────────v───────────────────────┐
                 │ overridden=true → 同步永不触碰         │
                 └──────────────────────────────────────┘
```

- **决定**：微调作用于**周模式**（weekday×period，动作 = 替换/删除/新增）；物化实例带 `source='timetable'` + `source_slot`（如 `3-2`）；唯一索引 `(class_id, event_date, source_slot)`——手工事件 `source_slot` 为 NULL，MySQL 唯一索引不约束 NULL，天然共存
- **覆盖**：老师调整实例 → `overridden=true`；取消 → 软删 + `overridden=true`，重生成时以 **Unscoped** 检查槽位存在性跳过（软删除行占位是既有陷阱，明确绕开）
- **模板/微调变更**：只重算「窗口内、未来、未被覆盖」的实例；过去日期与已覆盖实例不动
- **备选**：读取时合并（例外逻辑散在展示层，dots/详情/变更通知全要特判）；按日期的微调模型（与「班级微调周课表」语义不符）

### D6 同步触发：保存即同步 + 每日滚动补齐；**不在读取路径写数据**

- **决定**：触发点为 (a) 开关开启或课表保存后立即重算窗口；(b) 服务内每日定时任务补齐窗口尾部（幂等，按槽位 upsert）；(c) 管理端提供手动「立即同步」按钮排障
- **理由**：项目已有明确教训——**读取路径挂写操作会污染生产数据**（`ensureBootstrap*` 的由来）；且定时补齐保证长期不断档
- **备选**：惰性读取时生成（违反上述教训，否决）

### D7 字典 seed

- **决定**：`sys_dict_types` 建 `calendar_event_type`（`IsSystem=true`），5 内置项 `Extra` 存 `{color}`（色值取 `design-data/teacher-06` 图例**实际值**：通知 #3B82F6 / 活动 #22C55E / 假期 #F97316 / 家长会 #A855F7 / 体检 #EF4444——旧契约示例里的 #10B981/#8B5CF6 不是设计稿值，已弃）；图标键由前端按 item_key 映射、不入字典（避免凭空拟值）；园所可增项、内置项可改名/停用不可删（`IsSystem` 兜底）
- **存量与新建园所**：正常路径是 `seeder.go` 启动播种（`SeedDictsForAll` 全园扫描 + kgID=0 模板；本次已把判重修为**按类型独立**——原先只查 `performance_tag` 会整体早退，新增类型会被已播种园所全部漏掉）；迁移 SQL 第 4 步为等价手工路径（幂等共存）。项目**无**「创建园所时播种」路径，新园所由启动扫描覆盖
- **前端**：家长端无 dicts 路由（实现核实）；标签与颜色由**服务端按字典解析**后随事件下发，老师端类型选择器经 `GET /teacher/dicts/calendar_event_type` 读取；两端渲染一律以字典值为源，删除 `eventDotColors` 硬编码

### D8 权限落点

- **决定**：
  - 4 个园长角色（行政/教学/后勤/招生）= **管理端 RBAC 角色**，携带 `calendar:garden:manage` 等权限码；由 seed/迁移创建，园所可再调整
  - 保健医 = 既有系统角色 `doctor`，发布入口走 **`/doctor/calendar/*` 路由**（老师端小程序内新增医生页），成为园级事件发布者之一
  - 老师对园级事件只读在 **service 层**校验（路由存在但操作被拒），错误语义明确
  - 全部新增管理端接口登记进权限登记表（否则启动失败——这是防护，不是风险）
- **理由**：园长是办公桌场景（管理端 web 顺手）；保健医本就在小程序 doctor 场景内工作
- **备选**：doctor 走管理端账号（角色体系交叉，账号模型别扭）

### D9 契约收敛（契约是 SoT，先改契约）

- `api-teacher/08-calendar.md`：查询参数统一为 `start`/`end`；dots 响应收敛为「日期 → 类型数组」（与设计稿着色需求一致，退役 count 形状）；补 `status`/`scope` 相关行为描述
- `api-parent/08-class.md`：仅返回已发布且命中受众的事件；删内部字段；补 `revised` 要旨
- `api-admin`：新增园级日历事件、课表模板/微调、开关与同步触发的端点段
- dicts 段：补 `calendar_event_type` 可读

### D10 迁移方案（详见 Migration Plan）

### D11 新界面无设计稿，走 impeccable 补设计

admin 两个新页、家长端 banner、老师端确认弹窗、医生发布页均无 `design-data/` 出处：实施时按 dev workflow 走 impeccable（`scripts/impeccable.cmd context` → 对应 playbook），色/间距/圆角一律 token。

## Risks / Trade-offs

- [同步任务与老师编辑并发，覆盖标记竞态] → 覆盖标记在同一事务内先行写入；同步重算时对已覆盖槽位跳过；幂等可重跑
- [每日滚动任务失败导致日历断档] → 幂等设计 + 手动「立即同步」兜底 + 失败显式告警日志（不静默）
- [存量数据迁移（scope/status 默认值、class_id=0 语义）] → 全部 default 兼容存量；`dbcheck` 逐表核实后再宣布完成
- [字段级 `change_summary` 在复杂修改下可能不精确] → 可接受的降级：退化为「内容已更新」；升级 revision 表留待 [待定] 项
- [字典缓存导致两端显示滞后] → 采用现有 dicts 拉取时机，不新增缓存层

## Migration Plan

1. **前置核实**：`go run ./cmd/dbcheck calendar_events` 记录现状索引与行数
2. **DDL（手工 SQL，`docs/migrations/`）**：
   - `calendar_events`：加 `scope`（default `'class'`）、`status`（default `'published'`）、`source`（default `'manual'`）、`source_slot`（nullable）、`overridden`（default false）、`revised_at`（nullable）、`change_summary`（nullable）；加唯一索引 `(class_id, event_date, source_slot)`
   - 新建 `timetable_templates`、`timetable_overrides`
   - 字典 seed：为现存园所幂等插入 `calendar_event_type` 及 5 项
   - 园长角色 seed 与权限码登记（随代码发布，非 DDL）
3. **核实**：`go run ./cmd/dbcheck calendar_events` / `timetable_templates` / `timetable_overrides` 确认索引与列
4. **回滚策略**：本次为纯增量（加列/加表/seed），回滚 = 代码回滚 + 保留冗余列；不删列不做破坏性变更

## Open Questions

- **变更记录存储粒度**（[待定]，用户已声明稍后定）：当前最小实现不阻塞验收；升级为 revision 表时仅换存储
- **importance 紧迫性表达与「近期重要活动」聚合**（产品待决策）：本次不暴露 `importance`；裁决后另行提案
- **课表节次是否绑定时段**（如第 2 节 = 09:30-10:30）：当前模板条目可带可选起止时间；若不带，物化事件为「全天/无时间」，产品后续可定
