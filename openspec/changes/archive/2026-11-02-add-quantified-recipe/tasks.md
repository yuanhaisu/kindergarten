# Tasks

## 1. 契约与数据模型

- [x] 1.1 编写 `backend/api/api-admin/23-diet.md` 契约（配置/分类/食物/菜谱/食谱/模板/分析/导出），并在 `backend/api/api-admin/README.md` 索引表登记；验证方式：逐项对照 `proposal.md`、`design.md` D8 与调研文档 §2/§3，确认无契约外字段
- [x] 1.2 在 `server/internal/domain/` 新增 11 个模型（MealConfig、MealConfigMeal、FoodCategory、Food、Recipe、RecipeItem、MealPlan、MealPlanItem、TemplatePlan、TemplatePlanItem、DriAgeTarget）与枚举常量；`Food.kg=0` 表示平台预置哨兵；验证方式：`go build ./...` 通过
- [x] 1.3 在 `server/internal/app/provider.go` 的 AutoMigrate 列表加入 11 个模型；验证方式：启动一次后端，再跑 `go run ./cmd/dbcheck diet_foods`、`diet_meal_plans`、`diet_meal_plan_items`，确认列与索引存在
- [x] 1.4 新增 `server/internal/repository/` 下的 config/food/recipe/plan/template/dri 仓储；验证方式：`go test ./internal/repository -run 'Diet|Food|Recipe|MealPlan'` 通过
- [x] 1.5 在 `repository/wire.go`、`service/wire.go`、`app/wire.go` 注册新 provider；若 service 依赖接口，补 `wire.Bind`；wire 必须用 Go 1.25 构建；验证方式：`make wire` 成功且 `go build ./...` 通过
- [x] 1.6 更新 `server/internal/app/router.go` 的 `/admin/diet/**` 路由组与 `handler.go` 的 Handler 聚合；验证方式：`go build ./...` 与 `go test ./internal/app -run Router` 通过

## 2. 供餐配置后端

- [x] 2.1 实现供餐配置 service：单例读写、默认值、餐次时间递增、标准/自定比例模式（standard 跳过相邻上限、custom 校验上限）、市品重量固定、配置快照；验证方式：新增 `diet_meal_config_test.go`，`go test ./internal/service -run DietMealConfig` 通过
- [x] 2.2 实现配置 handler 与 `GET/PUT /admin/diet/config`；验证方式：`go test ./internal/handler/...` 或路由级测试通过
- [x] 2.3 在 `middleware/admin_perm.go` 登记配置路由（`diet:view`/`diet:manage`），在 `service/role_catalog.go` 增加 `diet` 模块（view/manage/export）；验证方式：`go test ./internal/middleware -run AdminPerm` 与 `./internal/app -run RouterPerm` 通过
- [x] 2.4 验证方式：`go build ./...`，并逐条对照 `specs/diet-meal-config/spec.md` 的 TC-NEW-DIE-0001..0008 跑 service 测试

## 3. 食物库与菜谱库后端

- [x] 3.1 实现食物 service：列表/详情/新增/修改/删除/批量删除、分类筛选、NULL 语义、引用删除规则、`is_customized` 升级跳过；验证方式：`go test ./internal/service -run DietFood` 通过
- [x] 3.2 实现菜谱 service：园内名称唯一、保存重名「替换/仅本次」、同菜同食物合并、联想查询、改名/删除/合并重复、历史食谱不受影响；验证方式：`go test ./internal/service -run DietRecipe` 通过
- [x] 3.3 实现 `server/cmd/dietfood` 导入命令（CSV → `kg=0` 平台预置，按 `preset_key` 幂等）与全局食物分类 + `DriAgeTarget`（3/4/5/6 岁 × 11 项）seeder；验证方式：用样例 CSV 导入，确认新增/更新/跳过/错误四类计数，并跑 `go run ./cmd/dbcheck diet_foods`
- [x] 3.4 实现 `SeedDietAfterMigration`（全局分类/DriAgeTarget → 按园复制 `kg=0` 食物 → 种模板；无食物跳过并告警）与 `SystemService.Initialize` 的 `EnsureKindergartenSeeded` hook；验证方式：新园与已有园重复启动幂等，`diet_foods` 不产生重复行
- [x] 3.5 实现食物/菜谱 handler 与路由，登记 `admin_perm.go` 权限；验证方式：路由权限测试通过
- [x] 3.6 验证方式：`go build ./...`，逐条对照 `specs/diet-food-library/spec.md` 的 TC-NEW-DIE-0009..0027

## 4. 周食谱后端

- [x] 4.1 实现周食谱 service：按学年管理、三种创建方式、模板种子复制、复制历史按星期映射并重算快照、名称唯一；验证方式：`go test ./internal/service -run DietPlan` 通过
- [x] 4.2 实现整单保存事务：校验 → 同菜同食物合并 → 生成营养快照 → 替换 items；验证方式：并发/重复保存测试通过，`diet_meal_plan_items` 无重复行
- [x] 4.3 实现编辑器辅助接口：菜谱联想、同类替换候选、位置调整、清空、删除；验证方式：service 测试通过
- [x] 4.4 实现周食谱 handler 与路由，登记 `admin_perm.go` 权限；验证方式：路由权限测试通过
- [x] 4.5 验证方式：`go build ./...`，逐条对照 `specs/diet-meal-plan/spec.md` 的 TC-NEW-DIE-0028..0043

## 5. 营养分析后端

- [x] 5.1 [P] 页面列渲染与导出收口：营养表 14 列（视黄醇列复用 vitaminA、胡萝卜素「-」）、膳食构成 10 列、脂肪 2 列；导出 5 sheet 同样不含未展示列；验证方式：`notes.md` §3.13 与 `api-samples/nutrition-analysis-export.xls` 已就位
- [x] 5.2 实现营养聚合 service：明细公式、NULL 跳过、DRIs 按班级人数加权、0.95 双向折算（显示值 ÷0.95、比较基准 ×0.95）、±20% 评价；验证方式：固定样本单测（含 79.98/80/120/120.07 边界与 1341.75/1484.26=90.4%）通过
- [x] 5.3 实现 7 组表（4 页签）与自动结论结构化输出，报表使用 `MealPlan.config_snapshot` 中的 DRIs 目标值快照；验证方式：用 `api-samples/report-nutrition-analysis.json` 做黄金样本回归，页面实际展示的 12 个数值全部精确匹配（retinol/carotin 不展示）
- [x] 5.4 实现打印视图数据与 Excel 导出（`excelize/v2` 同步生成 .xlsx；参考导出为 .xls，内容对齐即可）；验证方式：导出 5 个 sheet、膳食构成 10 列、无 retinol/carotin，数值与页面一致
- [x] 5.5 实现分析 handler 与路由，登记 `admin_perm.go`（查看 `diet:view`、导出 `diet:export`）；验证方式：路由权限测试通过
- [x] 5.6 验证方式：`go build ./...`，逐条对照 `specs/diet-nutrition-analysis/spec.md` 的 TC-NEW-DIE-0044..0064

## 6. Admin 前端

- [x] 6.1 [P] 新增 `admin/api/diet.ts` 与 `admin/types/api.d.ts` 类型；验证方式：`npm run build` 通过
- [x] 6.2 [P] 在 `admin/constants/menus.ts` 新增「膳食管理」菜单组（基本参数/膳食计划/菜谱库/资源库），挂 `diet:view`；页面按钮统一用 `usePermission().can('diet:manage')` 条件渲染；验证方式：登录后菜单可见，无权限角色不可见
- [x] 6.3 [P] 实现 `/diet/config` 基本参数页；验证方式：保存 5 餐配置后刷新一致，超限/时间倒序有错误提示
- [x] 6.4 [P] 实现 `/diet/foods` 资源库页（列表/详情/增改删/批量删除/分类筛选）；验证方式：新增食物后可在食物列表与周食谱编辑器的食物搜索中找到
- [x] 6.5 [P] 实现 `/diet/recipes` 菜谱库页（列表/详情/改名/删除/合并）；验证方式：合并重复后配方克数正确
- [x] 6.6 [P] 实现 `/diet/plans` 列表与新建向导（自定义/模板/复制历史）；验证方式：三种方式均能生成周食谱
- [x] 6.7 实现 `/diet/plans/:id` 网格编辑器（7 天 × 餐次、菜→食物→克数、增删、同类替换、位置调整、保存菜谱、清空、实时小计）；验证方式：真实点击完成一次编辑并保存，刷新后内容一致
- [x] 6.8 [P] 实现 `/diet/plans/:id/analysis` 营养分析页（4 页签 / 7 组表 + 结论 + 打印 + Excel；按页面复刻 10 列构成、2 列脂肪、14 列营养表）；Excel 下载用 `useHttp().getBlob`（带 Bearer），不用 `window.open`；验证方式：页面数值与后端一致，打印视图可打印
- [x] 6.9 运行 `node scripts/check-tokens.mjs`；验证方式：本模块无新增 token 硬编码
- [x] 6.10 验证方式：`npm run build` 通过，并用真实后端跑通「配置 → 食物 → 菜谱 → 周食谱 → 分析 → 导出」全流程

## 7. 测试与验收

- [x] 7.1 在 `docs/tests/TEST_CATALOG.md` 增加 `DIE` 模块缩写与 64 条 `TC-NEW-DIE-0001..0064`；验证方式：编号与 specs 引用一一对应，无重复/缺失
- [x] 7.2 补 router 权限回归测试：未登记路由启动失败、`diet:view`/`diet:manage`/`diet:export` 分支正确；验证方式：`go test ./internal/app -run Router` 与 `./internal/middleware -run AdminPerm` 通过
- [x] 7.3 admin 端到端验证：用 Playwright MCP 真实点击（不依赖空的 `admin/tests/e2e` harness）完成一次完整带量食谱流程，覆盖空状态、校验错误、删除二次确认；验证方式：截图与断言全部通过
- [ ] 7.4 最终验收：`make test`、`npm run build`、`node scripts/check-tokens.mjs`、`openspec validate add-quantified-recipe --strict` 全部通过；逐条对照 4 份 specs 勾选
  > 2026-11-03 状态：`npm run build` 与 `openspec validate --strict` 通过；`make test` 与 `check-tokens` 未全绿，失败均为**既有非 diet** 项（notice 测试写死 `2026-11-01` 已过期；service 套件 MySQL COMMIT 抖动；14 条存量 token 违规）。diet：103 测试 PASS、0 新增 token 违规。豁免 W1/W2，详见 PROGRESS。
- [ ] 7.5 数据许可门：与中国疾控中心营养与健康所签《中国国家食物成分数据授权使用协议》（覆盖入库/终端展示/SaaS 分发/衍生作品/续期五项），取得授权 CSV 与 2002 版字段清单；验证方式：合同/授权文件归档，未授权数据不在仓库与发行物中
  > 发布阻断：需用户/法务签署授权协议并导入授权 CSV；Sanotsu 数据仅本地，未提交。
