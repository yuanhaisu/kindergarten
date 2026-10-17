# Proposal

## Why

老师端「成长记录 → 指定学生」模式的选择器有三个问题，在大班里会显著影响使用：

1. **选中态不亮**：点击学生后胶囊不高亮，老师无法确认选了什么——模板读 `s.id`，后端返回的是 `student_id`（数据其实进了 `form.student_ids`，纯显示错位）。授徽章面板的学生选择器有同样问题
2. **大班不可用**：一个班 30 人时，选择器把全部名字一次性铺成胶囊流（每行约 4 个 → 7-9 行），没有搜索、没有已选计数，找特定孩子要靠肉眼扫
3. **编辑时改学生会静默丢失**：`UpdateGrowthLog` 的更新映射不含 `growth_log_students`——编辑一条个体记录、改动学生名单，保存后名单不变（同 `UpdateGrowthLog` 不更新日期的同类问题）

另有一处口径不一致：选择器数据来自接送页复用的学生查询，**不过滤在读状态**，毕业/退学的孩子也会出现在选择列表。

## What Changes

**老师端（teacher/）**

- 选择器选中态修复：`:key` / `:class` 改读后端实际字段，点击即高亮（含授徽章面板的同一模式）
- 30 人班可用性：学生选择区加**搜索框**（按姓名本地过滤）+ **「已选 N 人」计数**；空态区分「可搜索」与「搜索无结果」/「班级暂无学生」
- 编辑态回显：进入编辑时已选名单从详情响应的 `related_students` 正确回填（选择态可见）

**后端（server/）**

- 学生列表查询限定 `status=1`（在读），与 `classStudentIDs` 的口径一致
- `UpdateGrowthLog` 同步维护 `growth_log_students`：`target_type=2` 时按请求的 `student_ids` 重建关联（先删后插）；`target_type=1` 时清空关联

**明确不做什么（Out of scope）**

- **不做**接送页的改动（虽然复用同一服务方法，本次只收紧其查询口径，不改响应形状、不改接送页 UI）
- **不做**成长记录创建页其他区块（模板、照片、标签、日期）的任何改动
- **不做**家长端、admin 端任何改动
- **不做**学生列表分页/虚拟滚动（30 人量级本地过滤足够；超过 ~100 人才需要，属未来问题）
- **不做**照片级学生关联（`growth_media_students`）与 AI 识别——仍属另案

**契约变更**

本次将修改 `backend/api/api-teacher/06-growth.md`，属于契约变更：`PUT /teacher/growth/logs/:id` 的请求体补写 `student_ids` 的处理语义（此前契约已列该字段，但实现未处理），`GET /teacher/class/students` 的查询参数补记 `status`。无响应字段新增。

## Capabilities

### New Capabilities

- `growth-target-students`: 老师端成长记录「指定学生」的目标学生选择——候选范围（本班在读）、选中交互的可反馈性、大班场景下的检索与已选可见性，以及编辑时目标学生关联的持久化

### Modified Capabilities

（无。`teacher-growth-authoring` 的既有需求不因本变更而变化，个体记录的学生关联写入属本变更首次定义的语义。）

## Impact

**后端（Go）**

- `server/internal/service/teacher_portal_service.go` — 学生列表查询加在读过滤；`UpdateGrowthLog` 增加 `growth_log_students` 的同步维护
- `server/internal/repository/` — 如需按 log 清理/重建关联的辅助方法

**契约**

- `backend/api/api-teacher/06-growth.md` — PUT 请求体的 `student_ids` 语义、学生列表 `status` 参数

**老师端小程序**

- `teacher/src/pages/growth/create.vue` — 选择器（模板 + 本地过滤 + 计数 + 空态）、授徽章选择器同一修复

**测试**

- `docs/tests/TEST_CATALOG.md` — 补录 TC-NEW-TGP-*（候选范围 / 选中态 / 搜索 / 计数 / 编辑持久化）

**不改动**

- `server/internal/domain/` — 无表结构变更（`growth_log_students` 已存在）
- `server/internal/app/router.go`、`rbac_policy.csv`、`wire.go` — 无新增路由/依赖
- `teacher/src/pages/growth/index.vue`、`detail.vue`、`teacher/src/api/teacher-growth.js` — 本变更不涉及
- `parent/`、`admin/` — 不涉及
