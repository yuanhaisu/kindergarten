# Tasks

## 1. 数据模型与迁移

- [x] 1.1 在 `server/internal/domain/` 新增 `DietGroupObject`、`DietGroupObjectMeal`、`DietGroupObjectClass` 三个模型；`MealPlan` 增加 `GroupObjectID`；`DietConfigSnapshot` 增加 `group_object_id` / `group_object_name` / `group_object_class_ids`；旧 `MealConfig*` 标注弃用但不删除。验证方式：`go build ./...` 通过
- [x] 1.2 在 `server/internal/app/provider.go` 的 AutoMigrate 列表注册 3 个新模型。验证方式：启动一次后端，`go run ./cmd/dbcheck diet_group_objects` / `diet_group_object_meals` / `diet_group_object_classes` / `diet_meal_plans` 确认列与索引存在
- [x] 1.3 在 `diet_seed_service.go` 实现 `EnsureDietGroupObjects`：按园所 × 学年幂等创建默认「全体幼儿」（全部班级、5 餐标准比例、市品重量、DRIs2000），并把该学年 `group_object_id=0` 的周食谱回填到默认对象；接入 `SeedDietForAll` 与 `EnsureKindergartenSeeded`。验证方式：新增 `diet_group_object_seed_test.go` 连续执行两次断言零重复，再启动后端用 API 确认列表只有一条默认对象

## 2. 后端仓储与服务

- [x] 2.1 新增 `repository/diet_group_object_repo.go`：按学年分页列表、按 ID 详情（含餐次/班级）、创建、整组更新关联、按学年取默认对象、回填计划对象。验证方式：`go test ./internal/repository -run DietGroupObject` 通过
- [x] 2.2 抽取 `diet_config_service.go` 中的餐次时间/占比/系数校验为无状态函数；新增 `DietGroupObjectService`（列表/详情/新增/修改、名称唯一、班级必选、学年归属）。验证方式：新增 `diet_group_object_service_test.go` 覆盖 TC-NEW-DIE-0065..0068、0074，`go test ./internal/service -run DietGroupObject` 通过
- [x] 2.3 改造 `DietPlanService`：创建必填 `group_object_id` 且校验学年归属；快照写入对象身份；列表/详情返回对象；`dietDriWeights` 只统计对象就餐班级。验证方式：更新 `diet_plan_test.go` 覆盖 TC-NEW-DIE-0069、0071..0073、0075，`go test ./internal/service -run DietPlan` 通过
- [x] 2.4 替换 handler 与路由：新增 4 条 `/admin/diet/group-objects` 路由，移除 `/admin/diet/config`；`middleware/admin_perm.go` 同步登记；`DietPlanService` 依赖改为 group object service 并 `make wire`。验证方式：`go build ./...`、`go test ./internal/app -run Router` 与 `./internal/middleware -run AdminPerm` 通过

## 3. 契约

- [x] 3.1 重写 `backend/api/api-admin/23-diet.md` 配置章节：删除 `/admin/diet/config`，新增 `/admin/diet/group-objects` 的列表/详情/新增/修改契约（含字段、校验、错误码）。验证方式：逐项对照 `design.md` D7 字段溯源表，无契约外字段
- [x] 3.2 更新周食谱章节：创建入参增加 `group_object_id`，列表/详情增加对象字段，补充校验规则。验证方式：用 curl 对真实后端逐端点核对请求/响应与文档一致

## 4. Admin 页面

- [x] 4.1 [P] 更新 `admin/api/diet.ts` 与 `admin/types/api.d.ts`：新增 group-object 类型与 API，删除 config 类型/函数；计划类型增加对象字段。验证方式：`npm run build` 通过
- [x] 4.2 重写 `admin/pages/diet/config.vue` 为列表 + 添加/修改/详情对话框（学年筛选、搜索、名称、托管形式、**实时拉取班级列表并按学年过滤、按年级分组全选**、餐次/时间/占比、标准/自定、只读市品重量与 DRIs2000、制定食谱跳转）。验证方式：真实点击新增/修改后刷新一致；新增班级后重新打开对话框能看到新班级；`node scripts/check-tokens.mjs admin` 无新增违规
- [x] 4.3 [P] 更新 `admin/pages/diet/plans/index.vue` 与 `[id].vue`：新建向导增加集体对象选择，读取 `create=1&group_object_id=` query 自动打开并预选；列表增加集体对象列；详情头显示对象。验证方式：`npm run build` 通过
- [x] 4.4 更新 `admin/constants/menus.ts` 基本参数菜单文案（保持不变亦可），确认页面 title 与目标产品一致为「基本参数设置」。验证方式：登录后菜单与页面标题可见且与参考产品一致

## 5. 测试与验收

- [x] 5.1 在 `docs/tests/TEST_CATALOG.md` 补录 `TC-NEW-DIE-0065..0077` 并让 specs 引用一一对应。验证方式：编号无重复/缺失，`openspec validate add-diet-group-objects --strict` 通过
- [x] 5.2 补 router 权限回归：4 条新路由的 `diet:view` / `diet:manage` 分支正确，旧 config 路由已移除。验证方式：`go test ./internal/app -run Router` 与 `./internal/middleware -run AdminPerm` 通过
- [x] 5.3 用 Playwright MCP 真实点击跑通：切换学年 → 添加「全体幼儿」→ 修改餐次 → 制定食谱 → 新建周食谱 → 保存 → 详情显示对象 → 营养分析 DRIs 只按对象班级加权。验证方式：截图与断言全部通过，0 未捕获异常、0 500
- [x] 5.4 最终验收：`go build ./...`、`npm run build`、`node scripts/check-tokens.mjs`、`openspec validate add-diet-group-objects --strict` 全部通过；逐条对照三份 delta specs 勾选

## 6. 切换补丁：diet 自读选项接口

- [x] 6.1 新增 `GET /api/v1/admin/diet/academic-years`（`diet:view`）与 `GET /api/v1/admin/diet/classes?academic_year=<label>`（`diet:view`），让只有 diet 权限的角色无需 `system:kindergarten` / `student:view` 即可加载学年与就餐班级。验证方式：`go test ./internal/app -run Router` 与 `./internal/middleware -run AdminPerm` 通过
- [x] 6.2 更新 `backend/api/api-admin/23-diet.md` 与 admin `api/diet.ts` / `config.vue` / `plans/index.vue` 改用新接口。验证方式：`npm run build` 通过
- [x] 6.3 用「保健医」账号真实点击：基本参数页学年正常、编辑对话框加载 6 个班级；「后勤主任」「行政园长」可查看但不能添加/修改。验证方式：Playwright 截图与 DOM 断言
