# Design

## Context

见 `proposal.md — Why`。当前实现要点（`add-quantified-recipe` 已归档）：

- `domain.MealConfig` + `MealConfigMeal` 是园所单例，`kindergarten_id` 上有唯一索引；`DietConfigService.GetConfig/UpdateConfig/BuildConfigSnapshot` 围绕单例工作。
- `MealPlan` 只有 `academic_year_id`，创建时 `buildDietConfigSnapshot` 读取单例配置，`buildDietDriSnapshot` 统计本园全部班级。
- 参考产品的「基本参数设置」是按学年的集体对象列表；对象 = 名称 + 托管形式 + 就餐班级 + 餐次/时间/占比 + 重量口径 + DRIs2000。

约束：仓库没有迁移工具，只有 GORM `AutoMigrate`（只增不改）；软删除行会占用唯一索引；测试直连开发库，模型加字段后需启动一次后端做 AutoMigrate。

## Goals / Non-Goals

**Goals**

- 用独立表表达集体对象，保留按对象生效的餐次/占比/重量口径校验。
- 周食谱绑定对象，历史快照可追溯；DRIs 加权只看对象内班级。
- admin 基本参数页复现参考产品的列表 + 表单 + 制定食谱入口。

**Non-Goals**

- 不做对象删除/合并/复制；不做对象级权限。
- 不改模板筛选、营养报表口径、食物/菜谱库。
- 不把旧单例的定制值迁移到新对象（发布前只有默认值；迁移只种默认对象并回填旧食谱）。

## Decisions

### D1. 三张新表，不复用单例表

新增：

- `diet_group_objects`：园所 + 学年 + 名称 + 托管形式 + 重量口径 + 比例模式 + DRIs 标准。
- `diet_group_object_meals`：对象 × 餐次（`meal_key` 唯一），保存时间、占比、启用、排序。
- `diet_group_object_classes`：对象 × 班级（`class_id` 唯一）。

**为什么**：旧表 `diet_meal_configs` 的 `kindergarten_id` 唯一索引无法靠 AutoMigrate 删除，复用会让「一园多条对象」永远撞键。新表避免破坏性迁移；旧表保留在库中但不再读写。

**唯一性**：对象名称唯一由服务层校验（同园同学年），不建唯一索引——软删除行占键是已踩过的坑；本变更不提供删除，风险可控。餐次/班级关联表整组保存时按业务键 upsert，删除的关联行用 `Unscoped()` 硬删。

### D2. 对象是配置的唯一边界

`DietGroupObject` 取代 `MealConfig` 成为餐次、占比、口径、系数的来源。`DietConfigSnapshot` 增加：

- `group_object_id`
- `group_object_name`
- `group_object_class_ids`（快照用班级 ID 列表，便于复核加权范围）

快照仍保存 `care_type`、`food_input_type`、`ratio_mode`、`coefficient`、`meals`、`dri_targets`、`dri_weights`。旧快照缺新字段时解析为空，不阻断报表。

### D3. 周食谱绑定对象，学年由对象决定

`MealPlan` 增加 `group_object_id`（索引，not null default 0 以兼容旧行）。创建入参增加 `group_object_id`：

- 必填；对象必须属于本园，且对象的 `academic_year_id` 必须等于请求的 `academic_year_id`（前端从对象带出学年，后端仍双向校验）。
- 创建时用对象配置生成快照，用对象的就餐班级生成 DRIs 权重。
- 列表/详情返回 `group_object_id` 与 `group_object_name`；列表可按对象筛选（可选参数）。
- 旧行 `group_object_id=0` 由迁移回填到同学年默认对象；详情仍能打开。

**为什么保留 `academic_year_id` 入参**：契约已有字段，前端按学年筛选列表；对象决定学年，但双写校验能挡住跨学年错配，且不破坏现有 API 形状。

### D4. DRIs 加权范围

`dietDriWeights` 从 `classes.List(全部班级)` 改为 `classes.ListByIDs(对象就餐班级)`；只统计能映射到 3/4/5/6 岁档的年级。对象内没有可用学生时沿用 26.7%/32.6%/40.8%/0。

**为什么**：参考产品的就餐对象就是供餐范围；「全体幼儿」与「日托班」应得到不同目标值。历史食谱用快照，不受对象后续改班级影响。

### D5. 基本参数页结构

`admin/pages/diet/config.vue` 改为：

- 顶部：学年下拉（默认当前学年）+ 搜索框 + 添加。
- 表格列：集体对象名称 / 托管形式 / 就餐对象 / 餐次 / 食物重量输入方式 / 膳食营养素参考摄入量标准 / 操作（修改、详情、制定食谱）。
- 添加/修改用同一个 `el-dialog`：名称、学年（添加时可选，修改时只读）、托管形式、就餐班级（按年级分组 + 全选）、餐次复选 + 时间 + 占比、比例模式、重量口径（只读「市品重量」）、DRIs2000（只读）。
- **就餐班级选项是实时拉取的班级列表**：对话框打开时调用 `GET /admin/classes?page_size=200`，按所选学年的 `academic_year` 字段过滤（该字段本就是学年标签，如 `2026-2027`），再按 `grade` 分组渲染复选与年级全选。选中结果只存 `class_id`；列表/详情返回时再与当前班级表 join 出名称。班级是动态数据，新增/改名后重新打开对话框即可见，不需要改班级接口；若所选学年没有班级，表单显示空态并禁止保存。
- 「制定食谱」跳转 `/diet/plans?create=1&group_object_id=<id>`；膳食计划页读取 query 后打开现有新建向导并预选对象。
- 无删除按钮。

### D6. 种子与迁移

`SeedDietGlobals` 之后新增 `EnsureDietGroupObjects`（幂等，按园所 × 学年）：

1. 若该学年已有任意对象，跳过。
2. 否则创建「全体幼儿」：全部本园班级、日托、默认 5 餐（标准比例，系数 0.95）、市品重量、DRIs2000。
3. 把该学年 `group_object_id=0` 的周食谱回填到默认对象。

新园所由 `SystemService.Initialize` 的 `EnsureKindergartenSeeded` 调用；已有园所在后端启动时由 `SeedDietForAll` 调用。**不在读取路径写库。**

旧 `diet_meal_configs` 不再读取；其定制值不迁移（当前只有默认值，发布前无生产数据）。

### D7. 契约端点

- `GET /admin/diet/group-objects?academic_year_id=&keyword=&page=&page_size=`（`diet:view`）
- `POST /admin/diet/group-objects`（`diet:manage`）
- `GET /admin/diet/group-objects/:id`（`diet:view`）
- `PUT /admin/diet/group-objects/:id`（`diet:manage`）
- 删除 `GET|PUT /admin/diet/config`（**BREAKING**，唯一消费方是 admin 基本参数页）
- `POST /admin/diet/plans` 增加必填 `group_object_id`；`GET /admin/diet/plans` 与 `GET /admin/diet/plans/:id` 增加对象字段。

列表只返回行渲染所需字段（名称、托管形式、班级、餐次名、口径、DRIs）；时间、占比、比例模式只在详情返回。输入用 `class_ids`，输出用 `classes[]`，不重复返回 `class_ids`。

### D8. 校验复用

把现有 `diet_config_service.go` 的餐次/时间/占比校验抽成无状态函数，新服务复用；错误码与提示文案保持不变，避免回归 TC-0003..0007。

## Cross-Layer Touchpoints

| 层 | 是否改 | 说明 |
| --- | --- | --- |
| Go domain | 会改 | 新增 3 个模型；`MealPlan` 加 `GroupObjectID`；`DietConfigSnapshot` 加对象字段；旧 `MealConfig*` 标记弃用 |
| Go repository | 会改 | 新增 `DietGroupObjectRepo`（列表/详情/创建/更新/按学年取默认/回填计划）；`DietPlanRepo` 增加对象筛选与回填方法；旧 config repo 停止使用 |
| Go service | 会改 | 新增 `DietGroupObjectService`；`DietPlanService` 创建/列表/详情/加权改造；`DietSeedService` 加迁移；删除或停用 `DietConfigService` |
| Go handler | 会改 | 替换 config handler 为 group-object CRUD；plan handler 入参/出参加对象 |
| Casbin policy | 不改 | 复用已存在的 `diet:view` / `diet:manage`，无新权限点 |
| `middleware/admin_perm.go` | 会改 | 登记 4 条 group-object 路由；移除 2 条 config 路由 |
| `app/router.go` | 会改 | 替换 `/diet/config` 为 `/diet/group-objects` |
| `app/provider.go` | 会改 | AutoMigrate 注册 3 个新模型 |
| Wire | 会改 | 新 repo/service provider；`DietPlanService` 依赖改为 group object service |
| `backend/api/api-admin/23-diet.md` | 会改（契约变更） | 配置章节重写；周食谱章节增加对象字段；README 索引不变 |
| admin `pages/diet/config.vue` | 会改 | 重写为列表 + 表单 |
| admin `pages/diet/plans/index.vue` | 会改 | 新建向导增加集体对象；列表增加对象列；读取 `create` query |
| admin `pages/diet/plans/[id].vue` | 会改 | 详情头显示集体对象 |
| admin `api/diet.ts` / `types/api.d.ts` | 会改 | 新端点与类型 |
| teacher / parent 小程序 | 不改 | 本变更只在 admin 与后端 |
| design token | 不改 | 页面复用既有 Element Plus + Tailwind token 类 |
| `GET /admin/classes` 班级列表接口 | 不改 | 复用现有接口拉取班级（返回 `academic_year`/`grade`），前端按学年过滤、按年级分组；不新增「按学年查班级」端点 |
| `admin/api/classes.ts` | 不改 | 现有 `getClasses` 已满足拉取与 `page_size` 需求 |
| `docs/tests/TEST_CATALOG.md` | 会改 | 补录 TC-NEW-DIE-0065..0077 |
| `cmd/dbcheck` | 不改 | 直接用于核实新表 |

**契约字段溯源**（新增/修改字段 → 依据）：

| 字段 | 依据 |
| --- | --- |
| `name` | 目标页表格「集体对象名称」列、表单「集体对象名称」 |
| `academic_year_id` | 目标页「学年」筛选与表单学年下拉 |
| `care_type` / `care_type_label` | 目标页表格「托管形式」列、表单 radio |
| `classes[]` | 目标页表格「就餐对象」列、表单班级复选（输出不复述 `class_ids`，由 `classes[].id` 可得） |
| `meals[].meal_key/name/time/energy_percent/enabled` | 目标页表格「餐次」列、表单餐次复选与时间/占比输入 |
| `ratio_mode` / `ratio_mode_label` | 目标表单「各餐热量及营养标准分配比」radio（标准/自定） |
| `food_input_type` / `food_input_type_label` | 目标页表格「食物重量输入方式」列（固定市品重量） |
| `policy_name` | 目标页表格「膳食营养素参考摄入量标准」列（固定 DRIs2000） |
| `coefficient` | 既有契约只读字段，报表折算用；不能由列表项派生（列表不返回占比，detail 才返回） |
| `group_object_id` / `group_object_name`（周食谱） | 目标页「计划食谱列表」的「集体对象」列、新建流程从对象行进入 |
| `config_snapshot.group_object_*` | 历史追溯业务场景：对象改名/改班级后旧报表仍需显示原对象 |

## Risks / Trade-offs

- [旧单例表残留] → 不读写、不删除；在迁移记录里注明，后续单独清理。
- [对象名称无唯一索引] → 管理端并发新增重名概率低；服务层校验 + 集成测试覆盖。
- [旧食谱 `group_object_id=0`] → 启动迁移回填；未回填的旧详情按快照或「—」显示，不报错。
- [删除班级后对象引用失效] → 列表/详情只返回仍存在的班级；保存时要求至少一个有效班级，不级联删除对象。
- [BREAKING 契约] → admin 同分支同步改；无第三方消费方。

## Migration Plan

1. 启动后端触发 AutoMigrate，新增 3 张表与 `diet_meal_plans.group_object_id` 列/索引。
2. `go run ./cmd/dbcheck diet_group_objects`、`diet_group_object_meals`、`diet_group_object_classes`、`diet_meal_plans` 核实列与索引。
3. 种子 `EnsureDietGroupObjects` 为每个园所 × 学年创建默认「全体幼儿」，并回填旧食谱。
4. 回滚：新表与列可保留不用；恢复旧 admin 页面与 `/diet/config` 路由即可回到单例（旧表未动）。

## Open Questions

无。
