# Design

## Context

动机见 proposal.md。影响设计的关键现状：

1. **前后端字段错位**：`GET /teacher/class/students` 返回 `student_id`（`teacher_portal_service.go:1321` 的响应映射），而 `create.vue` 两个选择器的 `:key` / `:class` 读 `s.id`（对象上不存在该键）→ 选中态永不呈现；`@click` 传的 `s.student_id` 反而是对的，数据一直在变、只是看不见。
2. **学生查询复用接送页方法**：`ClassStudents`（`:1881`）直接调 `PickupStudents(tc, "")`，后者用 `classStudentIDs` 取全班学生**不带 status 条件**（`:1178-1185`），再按 id 查姓名（`:1246-1250`）。接送页自己的 `PickupStudents` 也共用同一路径。
3. **`UpdateGrowthLog` 只更新主表**（`:996-1005`）：Updates 映射覆盖了正文/标签/评分/`target_type`/`sync_to_feed`，但**完全没有触碰 `growth_log_students`**——个体记录的「指定谁」在编辑时不可变。
4. **编辑态回填依赖详情响应**：`create.vue` 的 `loadExistingLog` 从 `related_students`（字段为 `id`/`name`/`avatar`）回填 `student_ids`。这与候选列表的 `student_id` 字段名不同，是错位 bug 的第二现场。
5. **数据规模**：班级学生典型 20-40 人（当前测试班 4-6 人）。这个量级本地过滤即可，无需后端搜索接口。

**关键事实核查**（本次改动范围判定的依据）：

| 事实 | 证据 |
|---|---|
| 老师端成长记录**无** 30 条上限、无分页 | `ListGrowthLogs` 只有 `Limit/Offset` 分页参数 |
| 选择器无设计稿来源（实现超设计） | design-data `teacher-04`/`teacher-08` 均无学生选择器 |
| `PickupStudents` 的调用方有两处 | 接送页（`TeacherPickupStudents` 也走 `PickupStudents`）与成长记录 `ClassStudents` |
| 学生表有 `status`（0待入学/1在读/2毕业/3退学） | `server/internal/domain/student.go:19` |

## Goals / Non-Goals

**Goals:**

- 选中态真实可见（含授徽章选择器），编辑回填正确呈现
- 30 人班可检索、已选数可见、空态语义清晰
- 候选只含本班在读学生
- 编辑个体记录的学生名单真实落库

**Non-Goals:**

- 接送页 UI 的任何改动（后端 query 口径收紧后，接送页表现一致或更好）
- 学生列表的分页 / 虚拟滚动（30 人量级不需要）
- 照片级学生关联（`growth_media_students`）——另案
- 任何路由 / 跳转变更（选择器是页内交互，不涉及 tabBar）

## Decisions

### D1：以「后端字段为准」修复错位，而非改后端迎合前端

**决定**：模板 `:key` 与选中判断统一改用 `student_id`；`loadExistingLog` 回填时把 `related_students[].id` 映射为 `student_ids`（这一步已经是 `map(s => s.id)`，保持），选择态判断统一走 `form.student_ids.includes(s.student_id)`。

**为何不改后端把字段名改成 `id`**：`student_id` 是接送页等既有消费方在用的字段（`create.vue:373` 的 `toggleStudent`、detail 页、家长端都用它）。改后端字段名会波及多个消费方，违反「最小改动」；前端读错字段才是缺陷本身。

**替代方案**（未采纳）：后端响应同时输出 `id` 与 `student_id`。（双字段冗余，且未来两处不一致时更难排查。）

### D2：搜索与计数在前端本地实现

**决定**：搜索框输入 → `computed` 过滤 `students`（按 `name.includes(query)`）；「已选 N 人」由 `form.student_ids.length` 派生。均不新增后端接口。

**为何不做后端搜索**：30 人量级本地过滤无感知延迟；后端搜索需要新接口/参数 + 防抖 + 契约变更，收益为零。当班级规模跨过 ~100 人（当前远未达到）再评估。

**空态语义**：`students.length === 0` → 「班级暂无学生」；`students.length > 0 && filtered.length === 0` → 「未找到匹配的学生」。

### D3：学生查询限定在读（status=1）

**决定**：在 `classStudentIDs`（`teacher_portal_service.go:1178`）的查询上补 `status = 1`，使其与成长记录创建页、接送页的语义一致。

```
修改前: WHERE kindergarten_id=? AND class_id=? AND deleted_at IS NULL
修改后: WHERE kindergarten_id=? AND class_id=? AND status=1 AND deleted_at IS NULL
```

**影响面**：`classStudentIDs` 被 `PickupStudents` 使用，接送页的候选也将只含在读学生——这是**一致性修复**而非单点回归：接送记录天然只对在读学生有意义。

**替代方案**（未采纳）：只在 `ClassStudents` 里过滤。（该校验散落到单个调用方，接送页仍会显示退学学生，口径继续分叉。）

### D4：`UpdateGrowthLog` 事务内重建学生关联

**决定**：更新主表成功后，在同一事务内：

```
target_type = 2 且 student_ids 非空:
    DELETE FROM growth_log_students WHERE log_id = ?
    INSERT (log_id, student_id) × N        ← 「先删后插」保证幂等

target_type = 1:
    DELETE FROM growth_log_students WHERE log_id = ?
```

**为何先删后插**：名单更新语义就是「替换」，先删后插天然幂等（重复提交同一名单不会叠加）；唯一索引 `idx_log_student` 也会兜住重复插入。

**为何放事务**：主表更新与关联重建必须同成败，否则会出现「记录改成个体但关联为空」或反之。

**边界**：`target_type=2` 但 `student_ids` 为空 → 拒绝（与创建路径的校验一致：至少选一位学生）。此校验归入 service 层，返回既有 `ErrTeacherInvalidParams`。

### D5：编辑态回填链路

```
detail 响应 related_students: [{id, name, avatar}]
        │  create.vue loadExistingLog: f.student_ids = related_students.map(s => s.id)
        v
form.student_ids = [1, 2]
        │  选择器渲染: :class="{ selected: form.student_ids.includes(s.student_id) }"
        v
候选列表（student_id）命中 → 高亮
```

两条字段链在此收敛：候选来自 `class/students`（`student_id`），回填来自 `related_students`（`id`），**选择器只消费 `form.student_ids`（纯数组）**——两侧都只写不读对方的字段名，避免再次错位。

## 跨端触点清单

| 层 | 会改 / 不改 | 说明 |
|---|---|---|
| Go domain | **不改** | `GrowthLogStudent` 表已存在，无结构变更 |
| Go repository | **不改** | 关联的删/插用现有 `db` 直写（与 `CreateGrowthLog` 写 `GrowthLogStudent` 的既有模式一致），不新增 repo 方法 |
| Go service | **会改** | `classStudentIDs` 加 `status=1`；`UpdateGrowthLog` 加事务化关联维护 + `target_type=2` 空名单校验 |
| Go handler | **不改** | `TeacherUpdateGrowthLog` 的请求结构体 `TeacherGrowthInput` 已含 `StudentIDs` |
| Casbin policy | **不改** | 无新增路由 |
| router.go | **不改** | 无新增路由 |
| Wire DI | **不改** | 无新增构造依赖 |
| `backend/api/` 契约 | **会改（契约变更）** | `api-teacher/06-growth.md`：PUT 请求体的 `student_ids` 处理语义（契约已列该字段，补记实现语义）；`GET /teacher/class/students` 的 `status` 查询参数（补充说明候选范围为在读）。**无响应字段新增/删除** |
| 老师端页面 | **会改** | `teacher/src/pages/growth/create.vue`：选择器模板字段名、搜索框、已选计数、空态；授徽章学生选择器的同类修复 |
| 家长端页面 | **不改** | 不涉及 |
| admin 页面 | **不改** | 不涉及 |
| design token | **不改** | 无新增样式；搜索框复用 `$component-input-*` 与既有 token |

**tabBar 核对**：本变更为页内交互（选择器在 `create.vue` 内），不新增任何页面跳转；老师端无 tabBar，不存在 `switchTab` 场景。编辑入口的既有 `uni.navigateTo` 不变。

**迁移核对**：无新表/新字段，不涉及 `AutoMigrate` 语义，无需手工迁移 SQL。`growth_log_students` 的既有数据（如历史个体记录）在下次编辑时按新逻辑重建——无需数据回填。

## Risks / Trade-offs

- **`status=1` 过滤影响接送页候选** → 缓解：这是语义一致性修复；若园所有「已毕业但仍在接送流程」的极端场景（不合理），需产品另行决策，本变更不为其保留旧行为。
- **编辑时先删后插的短暂空窗** → 缓解：整个操作在事务内，外部读不到中间态。
- **`target_type=2` 空名单拒绝，可能让「先存草稿再补学生」的用法受阻** → 缓解：创建路径本就有同样校验（「请选择至少一位学生」），行为一致；若未来要草稿语义，属独立变更。
- **列表刷新后选择态丢失（预期行为）** → 说明：`form.student_ids` 是页面内存态，重进页面从详情回填；这是既有语义，不是缺陷。

## Open Questions

（无。搜索/计数的交互密度、空态文案在实现层面可微调，不改变规格与任务拆分。）
