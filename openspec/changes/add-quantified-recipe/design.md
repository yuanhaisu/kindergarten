# Design

## Context

- 当前仓库没有膳食模块；`academic_years` 已存在（学年 + 上下学期 + 当前学年），直接复用。
- admin 是 Nuxt 3 SPA；Go 后端按 `domain → repository → service → handler → router → provider(AutoMigrate) → wire` 分层。
- admin 路由权限是 fail-closed：`middleware/admin_perm.go` 未登记的路由启动期即失败；新权限码还要加进 `service/role_catalog.go`。
- 既有出勤/学生导出用 `excelize/v2` 同步生成并通过 `c.Data` 直接下载；`/admin/export/*` 是 `export_tasks` 异步任务 + `c.File`。营养报表数据量小，采用同步生成；前端用 `useHttp().getBlob` 下载（带 Bearer），不用 `window.open`。
- 参考产品与营养数据的实证见：
  - `docs/research/2026-11-01-oyujian-diet/notes.md`
  - `docs/research/2026-11-01-nutrition-data/notes.md`
- 决策基线：`docs/adr/0004-diet-module-parity.md`、`0005-food-library-per-tenant.md`、`0006-nutrition-snapshot.md`。
- 数据许可约束：Sanotsu 数据仅本地开发验证，不提交、不分发；ncmi.cn 只提供元数据、实体数据「暂不共享」；上线前必须取得中国疾控中心营养与健康所《中国国家食物成分数据授权使用协议》，合同覆盖入库、终端展示、SaaS 分发、衍生作品、续期五项。

## Goals / Non-Goals

**Goals:**

- 在 admin 端到端交付 4 个能力：供餐配置、食物/菜谱库、周食谱编排、营养分析。
- 行为、字段、按钮文案、报表 7 组表与结论话术对齐参考产品；视觉使用本项目 token。
- 食物库按园隔离；历史食谱使用录入时快照，食物库或配置修改不回溯。
- 所有 admin 路由纳入 fail-closed 权限登记；新模块可通过角色管理授权。

**Non-Goals:**

- 不实现餐后调查、膳食会议、月伙食计划、采购清单、就餐人数、分析图、草稿箱。
- 不修改 teacher / parent 小程序，不新增家长端/老师端食谱展示。
- 不实现智能配餐、营养推荐算法、菜品图片、扫码/称重设备对接。
- 不在仓库中提交或分发未授权的食物成分数据。

## Decisions

### D1. 模块边界与能力拆分

拆成 4 个 capability，与 admin 的 4 个菜单 + 1 个报表页对应：

| capability | 拥有的行为 |
|---|---|
| `diet-meal-config` | 园所单例供餐配置、餐次/时间/占比、折算系数、配置快照语义 |
| `diet-food-library` | 食物库按园复制、营养/食部语义、食物维护、菜谱沉淀与管理 |
| `diet-meal-plan` | 周食谱创建三方式、网格编辑、模板与历史复制、保存规则 |
| `diet-nutrition-analysis` | 营养聚合、DRIs 折算、四张报表、自动结论、打印/Excel |

不采用单一大 capability（需求会失控），也不按页面拆（页面会变，行为边界更稳定）。

### D2. 数据模型

新增 11 张表，全部带 `kindergarten_id`（平台级数据除外）：

```
AcademicYear 1──n MealPlan 1──n MealPlanItem
MealConfig 1──n MealConfigMeal
FoodCategory 1──n Food 1──n RecipeItem n──1 Recipe
Food 1──n MealPlanItem (food_id 可空；行内保留 food_name + 营养快照)
Recipe 1──n RecipeItem
TemplatePlan 1──n TemplatePlanItem (每园种子副本，只读)
DriAgeTarget (全局，只读)
```

关键字段与约束：

- `MealConfig`：园所唯一；`care_type`（全托/日托）、`food_input_type` 固定为市品重量。
- `MealConfigMeal`：`meal_key` 用显式字符串（`breakfast` / `morning_snack` / `lunch` / `afternoon_snack` / `dinner` / `evening_snack`），**不复刻参考产品的位掩码**；每餐含名称、时间、供能占比、排序、启用标记；`(config_id, meal_key)` 唯一。
- `FoodCategory`：全局只读两级分类（`parent_id`、`level`、`name`），无 `kindergarten_id`，启动时只种一次；不塞进 `sys_dict`，因为需要稳定的层级 ID 与食物外键。
- `Food`：`kindergarten_id`（**0 = 平台预置哨兵行，>0 = 租户副本**）、名称、拼音码、叶子分类、食部比例、27 个 `*float64` 营养列（每 100g 可食部）、来源（system/custom）、`is_customized`、`preset_key`、软删除。所有租户查询必须带 `kindergarten_id = ?`，仓储层默认强制，禁止回传 kg=0 行。
- `Recipe`：`kindergarten_id`、名称、拼音码、软删除；名称唯一由服务层保证（不建唯一索引，避免软删除行占键）。
- `RecipeItem`：`recipe_id`、`food_id`、`grams`、`sort`；不存营养快照。
- `MealPlan`：`kindergarten_id`、`academic_year_id`、名称、起止日期、`day_num`、周次、`created_method`（custom/template/copy）、`config_snapshot` JSON（餐次/占比/口径 + 计算后的 11 项 DRIs 目标值与权重）、来源模板/来源食谱 ID、软删除。
- `MealPlanItem`：`plan_id`、日期、星期、`meal_key`、`recipe_name`、`recipe_id` 可空、`food_id` 可空、`food_name` 快照、`grams`、`nutrition_snapshot` JSON、`sort`。
- `TemplatePlan` / `TemplatePlanItem`：每园在食物库就绪后种子化；只读；条目引用本园 `food_id`，因此不跨租户引用。
- `DriAgeTarget`：全局只读（无 `kindergarten_id`），启动时只种一次；3/4/5/6 岁 × 11 项 = 44 行，字段 `age_band`、`nutrient_key`、`label`、`unit`、`value`、`sort`；能量用男值列，脂肪/碳水/千焦由能量与蛋白在 seed 时派生。视黄醇/胡萝卜素不进入目标表（页面不展示）。

**营养列 vs JSON**：`Food` 用 27 个显式列（导入映射、NULL 语义、列表列都直接）；`MealPlanItem` 用 JSON 快照（按行整体读写，不按营养素查询）。这是刻意的不对称。

**分组字段**：`Food` 额外存 `group12`、`group16`、`source_flags`（动物性/植物性/乳类/肉类）。`group12` 已由 22,809 行明细确定性重建（37=1谷、38=10豆、39=6+7肉、40=8蛋、41=12乳、42=3+4蔬菌、43=13+14小吃速食、44=5果、45=19调味品、46=9水产、47=18植物油、48=11坚果；薯类/饮料/糖蜜饯无槽位）；`group16` 由 20 个一级类合并（13→速食小吃等 4 次合并）。字段由导入映射填充、可在食物编辑页覆盖，不硬编码在报表里。

### D3. 供餐配置与营养折算

- 供餐配置是园所单例，默认日托、5 餐：早餐 25%、早点 5%、午餐 35%、午点 5%、晚餐 25%；`ratio_mode` 默认 `standard`（对齐参考产品 `precentType=1`）。
- 校验：时间递增（两种模式）；标准模式固定标准占比并跳过相邻上限；自定模式校验比例 >0 与相邻上限（早+早点≤30，日托 29；午+午点≤39；晚+晚点≤30）；至少一餐启用。
- 折算系数 = 已启用餐次占比之和（默认 0.95）。0.95 双向使用：显示的「平均每人每日」= 原始人均 ÷ 0.95；比较% 的分母 = 全日 DRIs × 0.95；显示的 DRIs 行 = 全日原值。
- DRIs 目标值 = 3/4/5/6 岁四档按本园班级人数加权（小→3、中→4、大→5、6 岁默认 0；能量男值）；无学生回退 26.7%/32.6%/40.8%/0。计算后写入 `MealPlan.config_snapshot`。
- `MealPlan.config_snapshot` 在创建时固化餐次结构、占比、口径与按班级人数加权后的 11 项 DRIs 目标值；报表只读快照。
- 不复刻位掩码：显式 `meal_key` 更可读、可外键、可排序；参考产品的位掩码只是其历史实现。

### D4. 食物库与菜谱库

- 新园初始化复制平台预置食物；每园独立副本。平台升级命令只更新 `is_customized=false` 的行，冲突行写入升级报告。
- 食物删除：被菜谱引用 → 拒绝；只被历史食谱引用 → 允许（历史行保留 `food_name` 与营养快照）。
- 菜谱 = 食物 + 一人均量克数的命名快照；名称在园内唯一（服务层保证）。
- 保存菜谱：新名新增；重名返回「替换 / 仅本次」两个分支；替换覆盖配方，仅本次不写库。
- 同菜内同一种食物在保存时合并克数。
- 菜谱管理页支持改名、删除、合并重复；合并把源菜谱同种食物克数累加到目标后删除源。
- 菜谱变更不影响历史周食谱（历史行存菜名与快照）。

不采用全局食物库 + 覆盖层（ADR-0005 已记录理由）。

### D5. 周食谱

- 三种创建方式：自定义 / 选用模板 / 复制历史。
- 模板每园种子化、只读；选用后复制成独立的 `MealPlan`，后续修改不影响模板。
- 复制历史按星期几映射到新周期，并**用当前食物库重新固化营养快照**（复制出的是一份新计划，不是历史归档）。
- 编辑器保存采用整单替换：前端提交完整 `items` 数组，后端在一个事务内校验、合并同食物、生成快照、落库。
- 保存即生效，无草稿/发布状态；删除与清空需二次确认。
- 编辑器保留参考产品的核心便利：菜名联想、同类替换、位置调整、实时营养小计、保存菜谱、清空。

### D6. 营养分析与报表

**通用公式**（已用参考产品 187 行明细逐项反算）：

- 可食部克数 = `grams × edible_percent / 100`。
- 人均日供给 = `Σ(可食部克数 × 每100g营养素 / 100) ÷ dayNum ÷ 0.95`；NULL 不参与求和。
- 比较% = `原始人均 ÷ (fullRecom × 0.95)`，等价于 `显示值 ÷ fullRecom`；显示的 DRIs 行是 `fullRecom` 原值。
- 评价：`<80%` 低、`80–120%` 合格、`>120%` 高（闭区间）。

**DRIs 目标值**：DRIs2000 的 3/4/5/6 岁四档 × 11 项，按本园班级人数加权（小→3、中→4、大→5、6 岁默认 0；能量男值）；无学生回退 26.7%/32.6%/40.8%/0。脂肪 = 能量×30%÷9，碳水 = (能量−蛋白×4−脂肪×9)÷4，千焦 = 千卡×4.186。实测校准见 `notes.md` §5.4；视黄醇/胡萝卜素不进入目标表——页面不展示它们。

**报表 7 组数据 / 4 个页签**（页签归属已由 JSP 源码确认，证据：`docs/research/2026-11-01-oyujian-diet/report-page.jsp` 与 `print-report.jsp`）：

| 页签 | 组 |
|---|---|
| 计划食谱营养分析表 | 营养分析表、三大营养素供能、蛋白质来源、脂肪来源、定性分析结论 |
| 计划食谱三餐供能比 | 三餐供能比 |
| 计划食谱膳食构成表 | 膳食构成 |
| 计划食谱钙铁来源 | 钙铁来源 |

| # | 组 | 口径（页面实际渲染） |
|---|---|---|
| 1 | 营养分析表 `nutrition` | 14 列 × 3 行；视黄醇当量列复用维生素A值，胡萝卜素列写死「-」；retinol/carotin 不展示 |
| 2 | 三大营养素供能 `energy` | 4 行；占总热量% 分母 = 蛋白×4 + 脂肪×9 + 碳水×4（本周 1380.7），不是 1341.75 |
| 3 | 三餐供能比 `scgnb` | 6 餐次 × 2 行；分子未除 0.95、分母用 fullRecom，合计 85.88% |
| 4 | 膳食构成 `ssgcb` | 接口 16 键；页面与导出都只渲染 10 类（谷类/薯类/蔬菜/水果/畜禽肉类/蛋类/水产类/大豆/乳制品/油脂类）；公式 = Σ可食部 ÷ dayNum ÷ 0.95 |
| 5 | 钙铁来源 `tgly` | 5 项 × 2 行；钙动物性含乳类且乳类重复计入（合计 126%）；铁肉类仅畜肉+禽肉（合计 100%） |
| 6 | 蛋白质来源 `dbz` | 3 项；动物 {6,7,8,9,12}、豆类 {10}、要求 = 总蛋白×50%；三项合计 62.49%，不是穷尽分解 |
| 7 | 脂肪来源 `fats` | 接口 3 项；页面只渲染动物性/植物性 2 列。其他脂 = 总脂肪 − 动物性 − 植物性（残差，含 13/14/15/16/18/19 等未归类项）；仅用于分析图（首版不做图） |

**刻意保留的非标准口径**（ADR-0004：优先对齐旧产品，不在本变更中“修 bug”）：

- 同一响应存在三个能量总量：1341.75（营养表）、1274.65（三餐比）、1380.7（占总热量% 分母）。
- 营养表的视黄醇当量列实际显示维生素A值，胡萝卜素列显示「-」；retinol/carotin 在 API 有值但页面不用。
- 膳食构成 API 返回 16 键、页面只渲染 10 列；脂肪来源 API 3 项、页面 2 列。
- 钙三来源不互斥；蛋白质来源只统计动物+豆类。
- 三餐供能比合计 85.88%，不是 100%。

**契约裁剪**：按「契约字段必须有页面出处」的规则，retinol/carotin 不进入契约；页面实际展示的 12 个数值（energy/heat/protein/fats/carb/vitaminA/B1/B2/C/Ca/Zn/Fe）全部可复现。

**已解出但页面不用的公式**（供未来 API 对齐）：视黄醇当量 = 维生素A + 胡萝卜素报表值 × 0.167；胡萝卜素报表值 = 基础公式值 × 2（24 份周食谱拟合，R²≈1，逐条误差 < 0.01）。本变更不展示，不进入契约。

**导出取样**（`api-samples/nutrition-analysis-export.xls`，82,944 字节 BIFF8）：5 个 sheet（带量食谱营养分析表 / 营养分析图 / 三餐功能比 / 膳食构成表 / 钙铁来源）；膳食构成只有 10 列；retinol/carotin 不导出；`otherfats` 只出现在「营养分析图」数据 sheet。参考文件是 `.xls`，我们用 excelize 生成 `.xlsx`，内容对齐、扩展名不同；sheet 名「三餐功能比」与页面「供能比」不一致，实现时按页面用词修正并在验收记录中标注。

- 自动结论按参考产品结构拼装；建议话术按营养素固定，不随缺口大小变化。
- 报表只读 `MealPlanItem.nutrition_snapshot` 与 `MealPlan.config_snapshot`。
- Excel 用 `excelize/v2` 同步生成并通过 `c.Data` 下载；前端用 `useHttp().getBlob`（带 Bearer），不用 `window.open`。打印用独立打印视图（print CSS），不引入 PDF 服务。
- 前端渲染结构化 JSON；结论文案按固定模板渲染，保证可测试。

### D7. 权限

新增权限模块 `diet`：

| 权限码 | 用途 |
|---|---|
| `diet:view` | 查看配置、食谱、食物、菜谱、报表 |
| `diet:manage` | 维护配置/食物/菜谱/周食谱 |
| `diet:export` | 导出营养分析 Excel |

- 所有 `/api/v1/admin/diet/**` 路由逐条登记到 `middleware/admin_perm.go`（fail-closed）。
- `service/role_catalog.go` 增加 `diet` 模块，角色管理页可勾选。
- Casbin `rbac_policy.csv` 预计不改（admin 组为通配）；实现时先核对，若策略不是通配则补 `p, admin, /api/v1/admin/diet/*, (GET|POST|PUT|DELETE)`。
- 菜单项挂 `diet:view`。

### D8. 契约与接口粒度

新增契约文件 `backend/api/api-admin/23-diet.md`，并在 `backend/api/api-admin/README.md` 索引表登记。本次属于契约变更。

端点分组（具体字段以契约为准）：

| 分组 | 端点 |
|---|---|
| 配置 | `GET/PUT /admin/diet/config` |
| 分类 | `GET /admin/diet/food-categories` |
| 食物 | `GET/POST /admin/diet/foods`、`GET/PUT/DELETE /admin/diet/foods/:id`、`POST /admin/diet/foods/batch-delete`、`GET /admin/diet/foods/:id/replace-candidates` |
| 菜谱 | `GET/POST /admin/diet/recipes`、`GET/PUT/DELETE /admin/diet/recipes/:id`、`POST /admin/diet/recipes/merge`、`GET /admin/diet/recipes/suggest` |
| 食谱 | `GET/POST /admin/diet/plans`、`GET/PUT/DELETE /admin/diet/plans/:id`、`POST /admin/diet/plans/:id/clear` |
| 模板 | `GET /admin/diet/templates`、`GET /admin/diet/templates/:id` |
| 报表 | `GET /admin/diet/plans/:id/analysis`、`GET /admin/diet/plans/:id/analysis/export` |
| 学年 | 复用 `GET /admin/settings/academic-years`（不改） |

接口粒度：

- 食物列表只返回参考产品列表页展示的列（名称/来源/食部/核心营养）；27 项全量只在食物详情。
- 菜谱列表只返回名称、拼音码、配料数；配料明细只在详情。
- 周食谱列表只返回名称、周期、天数、创建方式；完整网格只在详情。
- 营养分析只返回 7 组表聚合结果与结论，不返回明细行。

### D9. 前端页面

新增菜单组「膳食管理」（新组，≤7 项）：

| 页面 | 路由 | 权限 |
|---|---|---|
| 基本参数 | `/diet/config` | `diet:view` |
| 膳食计划 | `/diet/plans` | `diet:view` |
| 周食谱编辑器 | `/diet/plans/:id` | `diet:view` |
| 菜谱库 | `/diet/recipes` | `diet:view` |
| 资源库 | `/diet/foods` | `diet:view` |
| 营养分析 | `/diet/plans/:id/analysis`（从列表行操作进入） | `diet:view` |

- 编辑器：7 列（周一至周日）× 餐次行；单元格内嵌套「菜 → 食物 → 克数」明细；使用 Element Plus 表格/表单组件，横向滚动。
- 所有颜色/圆角/阴影/间距/字号走 design token；本模块不新增 token（若出现新视觉模式再走 token 流程）。
- 页面按钮用 `usePermission().can('diet:manage')` 条件渲染；菜单挂 `diet:view`。
- 动态路由 `pages/diet/plans/[id].vue` 与 `pages/diet/plans/[id]/analysis.vue`；仓库暂无动态路由先例，实现前先验证 Nuxt 文件路由。
- 不使用图表库（分析图首版不做）。
- 周食谱编辑器保存采用整单提交，前端维护临时 item 数组与实时营养小计。

### D10. 数据导入与种子

- 新增离线命令 `server/cmd/dietfood`：读取授权 CSV，导入到 `Food` 的 `kindergarten_id=0` 平台预置行（按 `preset_key` 幂等 upsert）；可选 `--kindergarten` 直接分发到某园；输出新增/更新/跳过/错误四类计数；不提交、不分发未授权数据。
- 新增 `SeedDietAfterMigration(db)`（独立于 `SeedDictsAfterMigration`，显式日志，失败不阻断启动）：
  1. `SeedDietGlobals`：食物分类（20 个一级，并为每个一级幂等补一个二级占位「其他」）、`DriAgeTarget`（3/4/5/6 岁 × 11 项），全局只种一次；
  2. `SeedDietForAll`：遍历园所，幂等复制 `kg=0` 食物到该园（按 `preset_key` 判重），再种 2~4 套模板；
  3. 若园所没有食物（未导入授权 CSV），跳过模板并记录警告。
- 新园初始化：`SystemService.Initialize` 注入 `DietSeedService`，在创建园所后调用 `EnsureKindergartenSeeded`；启动 seeder 作为兜底。若园所通过脚本创建，重启后由 `SeedDietForAll` 补齐。
- 导入顺序：先跑 `dietfood` 写入 `kg=0` 预置，再启动 seeder/新园初始化；模板条目引用本园 `food_id`，找不到时跳过并在导入报告列出。
- 模板内容由我们提供初稿，上线前由园所营养师审核。
- 营养数据缺失项写 NULL；导入器不得把 `—` / 空字符串转成 0。
- 二级占位分类「其他」是**授权 CSV 未导入时的占位叶子分类**：不承载食物，导入真实二级分类后与之共存（按 `parent_id + name` 幂等，不重复、不覆盖）。它不是正式营养分类，正式分类以授权数据为准。

## Cross-layer Touchpoint List（强制）

| 层 | 会改 / 不改 | 说明 |
|---|---|---|
| Go `domain/` | **会改** | 新增 11 个 struct + 枚举常量（meal_key、created_method、source） |
| Go `repository/` | **会改** | 新增 config/food/recipe/plan/template/DRI 仓储 |
| Go `service/` | **会改** | 新增配置、食物、菜谱、周食谱、营养分析、导入服务；`Handler` 聚合 |
| `service/system_service.go` | **会改** | `Initialize` 注入 `DietSeedService`，新园创建后调用 `EnsureKindergartenSeeded` |
| Go `handler/http/` | **会改** | 新增 admin diet handlers；`Handler` 结构体与构造函数加字段 |
| Casbin `rbac_policy.csv` | **预计不改** | admin 组为通配；实现前核对，不是通配则补 policy |
| `app/router.go` | **会改** | 新增 `/admin/diet/**` 路由组 |
| `app/provider.go` | **会改** | AutoMigrate 增加 11 张表 |
| `app/seeder.go` | **会改** | 分类、DRIs、模板种子 |
| Wire | **会改** | `repository/wire.go`、`service/wire.go`、`app/wire.go`；若 service 依赖接口补 `wire.Bind` |
| `middleware/admin_perm.go` | **会改** | 所有 diet 路由逐条登记权限码（fail-closed） |
| `service/role_catalog.go` | **会改** | 新增 `diet` 权限模块（view/manage/export） |
| `server/cmd/dietfood` | **会改** | 新增食物导入命令（CSV → `kg=0` 预置，可分发到指定园） |
| `backend/api/api-admin/` | **会改（契约变更）** | 新增 `23-diet.md` + README 索引 |
| `admin/api/` | **会改** | 新增 `diet.ts` |
| `admin/types/api.d.ts` | **会改** | 新增膳食类型 |
| `admin/pages/diet/*` | **会改** | 新增页面；含动态路由 `plans/[id].vue` 与 `plans/[id]/analysis.vue` |
| `admin/constants/menus.ts` | **会改** | 新增「膳食管理」菜单组 |
| `admin` 权限过滤 | **会改（仅菜单表 + 页面按钮）** | 菜单挂 `diet:view`；按钮用 `usePermission().can('diet:manage')`；`permission.global.ts` / `Sidebar.vue` 从菜单派生，不改 |
| `teacher/` | **不改** | 不消费任何 diet 接口 |
| `parent/` | **不改** | 同上；无 tabBar 跳转变更 |
| `shared-design-tokens/tokens.json` | **不改** | 复用现有 token；无新视觉模式 |
| 测试 | **会改** | service/repository 单测、router 权限回归；admin 用 Playwright MCP 真实点击，不新建空 E2E harness；补录 TEST_CATALOG |
| `docs/tests/TEST_CATALOG.md` | **会改** | 新增 `DIE` 模块缩写与 57 个 `TC-NEW-DIE-*` |

## Contract Field Provenance（契约字段溯源）

本次没有 `design-data/` 设计稿；字段基线是参考产品的真实页面/接口（ADR-0004 明确授权），来源见调研文档：

| 字段组 | 来源 |
|---|---|
| 供餐配置：托管形式、餐次、时间、占比、重量口径 | `notes.md` §2.3（groupObj 表单与接口） |
| 食物：名称、拼音码、食部、分类、27 项营养 | `notes.md` §2.5 / §2.6（dietInfo、orgFood、新增食物弹窗） |
| 菜谱：名称、拼音码、配料（食物 + 克数） | `notes.md` §3.6（getOrgRecipeInfo） |
| 周食谱：名称、周期、天数、周次、创建方式 | `notes.md` §2.4 / §3.7（queryDiet、createMethod） |
| 食谱明细：日期、餐次、菜名、食物、克数、营养快照 | `notes.md` §2.5（queryDiet infos，187 行样本） |
| 营养分析 7 组表与结论 | `notes.md` §3.5 / §3.9 / §3.11 / §3.12、`report-page.jsp` / `print-report.jsp`、`api-samples/report-nutrition-analysis.json` 的 `_derived`；retinol/carotin 不进入契约（页面不展示） |
| 模板：名称、天数、餐次集合、条目 | `notes.md` §3.8（getAvilableDietary / queryDietary） |

以下字段是架构/运维需要，不属于参考产品契约，不进入对外响应或仅内部使用：`kindergarten_id`、`created_by`、`updated_by`、`is_customized`、`preset_key`、`config_snapshot`（快照在详情中只读返回，不作为写入字段）。

## Interface Granularity

- 列表接口不返回详情字段：食物列表不含 27 项全量、菜谱列表不含配料、食谱列表不含明细。
- 详情接口返回完整内容：食物详情含 27 项；菜谱详情含配料；食谱详情含配置快照与全部 items。
- 分析接口只返回聚合结果；不提供「原始明细 + 聚合」混合响应。

## ASCII Flows

**新建周食谱**

```
管理员点击「制定食谱」
   │
   ├─ 自定义 ──► 选学年/周期 ──► 创建空 MealPlan ──► 编辑器
   ├─ 选用模板 ─► 选模板 ──► 复制 TemplatePlanItem ──► 生成 MealPlan ──► 编辑器
   └─ 复制历史 ─► 选历史 MealPlan ──► 按星期映射 ──► 重新生成营养快照 ──► 新 MealPlan
```

**保存周食谱**

```
前端提交 items[]
   │
   ▼
校验（菜名/克数/餐次） ──失败──► 返回具体单元格
   │
   ▼
同菜同食物合并 ──► 读取 Food + 食部 ──► 生成 nutrition_snapshot
   │
   ▼
事务内整单替换 MealPlanItem ──► 返回保存结果
```

**营养分析**

```
加载 MealPlan + config_snapshot
   │
   ▼
按天/餐次/营养素聚合 nutrition_snapshot
   │
   ▼
平均每人每日（分母=有食谱天数）
   │
   ▼
DRIs = full_recom × Σ启用餐次占比
   │
   ▼
比较% ±20% 评价 ──► 7 组表 + 自动结论 ──► 打印/Excel
```

## Risks / Trade-offs

| 风险 | 缓解 |
|---|---|
| 食物数据许可 | Sanotsu 仅本地；ncmi 实体数据暂不共享；上线前与营养所签授权（五项范围），导入授权 CSV |
| 微量营养素缺失 30–50% | NULL 语义 + 「—」；报表标注缺失项；后续补录 6 版分类数据 |
| 12 组口径无官方字典 | 已用 22,809 行明细确定性重建：37=1谷、38=10豆、39=6+7肉、40=8蛋、41=12乳、42=3+4蔬菌、43=13+14小吃速食、44=5果、45=19调味品、46=9水产、47=18植物油、48=11坚果；薯类/饮料/糖蜜饯无槽位（NULL），采购口径会漏这三类 |
| DRIs 年龄加权 | 已确认按 3/4/5/6 岁加权（3 岁≈26.7%、4 岁≈32.6%、5 岁≈40.8%、6 岁≈0）；本实现按班级人数计算，试点园小/中/大→3/4/5；无学生回退固定权重。参考产品各班真实人数不可得，若与本园名册不一致，目标值会有差异 |
| 参考报表存在非标准口径 | 刻意保留（三个能量总量、视黄醇列复用维生素A、胡萝卜素列「-」、构成 10/16、脂肪 2/3、钙重复计入、蛋白质非穷尽、三餐比 85.88%）；不在本变更修 bug |
| 页面列数与 API 键数不一致 | 按页面复刻：构成 10 列、脂肪 2 列、营养表 14 列；契约只保留页面实际展示的 12 个数值 |
| Excel 导出内容 | 已取样：5 sheet、构成 10 列、无 retinol/carotin；`.xls` → `.xlsx` 与 sheet 名用词按已记录偏差处理 |
| 27 个营养列使 struct 冗长 | 显式列换取导入/NULL/查询简单；快照用 JSON 平衡 |
| 软删除行占唯一索引 | 菜谱/食谱名称唯一由服务层保证，不建唯一索引 |
| 模板内容质量 | 先出 2~4 套初稿，园所营养师审核后再上线 |
| 平台预置用 `Food.kg=0` 哨兵 | 仓储默认强制 kg 过滤；新增「租户查询不回传 kg=0」测试 |
| 种子顺序与幂等 | 先导入 kg=0 预置，再复制/种模板；按 `preset_key`/`template_key` 判重；无食物跳过模板并告警 |
| 新园初始化 hook | `SystemService.Initialize` 注入 `DietSeedService`；启动 seeder 兜底 |
| admin 无 E2E harness | 不新建空目录 harness；用 Playwright MCP 真实点击验收 |
| Excel/打印版式与旧产品差异 | 用参考产品报表结构做验收清单；不追求像素级，只保证 7 组表与数值一致 |

## Migration Plan

1. 新增表全部是新表，`AutoMigrate` 只增不改，可直接加进 `provider.go`；启动后跑：
   - `go run ./cmd/dbcheck diet_foods`
   - `go run ./cmd/dbcheck diet_meal_plans`
   - `go run ./cmd/dbcheck diet_meal_plan_items`
   核实索引与列。
2. Wire：用 Go 1.25 构建的 wire（`GOTOOLCHAIN=go1.25.0 go install github.com/google/wire/cmd/wire@latest`），否则报 `package requires newer Go version go1.25`。
3. 种子顺序：`SeedDietAfterMigration` 先种全局分类/DRIs，再复制 `kg=0` 食物到各园，最后种模板；无食物的园所跳过模板并告警。`SystemService.Initialize` 对新园调用 `EnsureKindergartenSeeded`。
4. 食物导入：先本地跑 Sanotsu CSV 验证，再 `server/cmd/dietfood --csv <授权文件>` 写入 `kg=0` 预置，再启动服务完成分发；导入器输出四类计数。
5. 上线门：营养所书面授权到位后才允许把食物数据随发行物部署。
6. 回滚：模块路由与菜单可下线；新表不影响既有功能，不需要回滚既有表。

## Open Questions

- 营养所书面报价/条款与到位时间；1,744 vs 1,506 的 238 条差额来源。
- 参考产品各班真实人数不可得；本实现按本园名册加权，若人数不同，`fullRecom` 会有差异。
- Excel 扩展名（参考 `.xls` vs 我们 `.xlsx`）与 sheet 名用词是否需要完全一致。
