# Proposal

## Why

`add-quantified-recipe` 为了控制 v1 范围，把参考产品的「集体对象（groupObj）」折叠成了园所单例供餐配置：只有一份托管形式、餐次、占比与重量口径，没有名称、学年与就餐班级。参考产品的「基本参数设置」实际是按学年管理的集体对象列表——一个园所可以有多条对象（如「全体幼儿」「日托班」），每条对象有自己的就餐班级、餐次与占比；周食谱创建时挂到某条对象上，营养分析的 DRIs 加权也只看该对象的就餐班级。

用户要求对比参考产品后复现这一套配置。当前单例模型无法表达「同一年度多套就餐范围」，也无法复现目标页面上的 添加 / 修改 / 制定食谱 / 详情 流程，需要把配置提升为按学年管理的集体对象。

## What Changes

- **BREAKING** 供餐配置从「园所单例」改为「按学年管理的集体对象列表」：
  - 字段：集体对象名称（≤20 字）、学年、托管形式、就餐班级（多选，至少一个）、餐次与就餐时间、供能占比（标准/自定）、食物重量输入口径（固定市品重量）、DRIs 标准（固定 DRIs2000）。
  - 就餐班级选项 MUST 从本园班级列表接口实时拉取（按所选学年过滤、按年级分组、支持年级全选），不得硬编码班级名，也不得只展示种子时的班级快照。
  - 同一园所同一学年内名称唯一；允许一个学年存在多条对象。
  - 保留既有校验：时间递增、标准/自定比例、相邻餐次上限、至少一餐启用、折算系数只读。
- 新增集体对象管理接口：列表（按学年 + 关键词）、详情、新增、修改；不提供删除（对齐参考产品，误建通过改名/停用处理）。
- 周食谱创建 MUST 指定集体对象；食谱列表与详情展示所属对象；创建时的 `config_snapshot` 增加对象身份（名称、ID、就餐班级）以便历史追溯。
- DRIs 目标值加权范围从「本园全部班级」收窄为「该集体对象的就餐班级」；没有可用学生数据时沿用固定回退权重。
- admin「基本参数」页从单例表单改造成参考产品同构的列表页：学年筛选、搜索、添加、修改、制定食谱、详情；「制定食谱」携带对象与学年跳转到膳食计划新建向导。
- 迁移：为每个园所每个学年幂等种一条「全体幼儿」默认对象（就餐班级 = 全部班级，餐次沿用旧的园所单例配置），并把已有周食谱回填到对应学年的默认对象。
- **本次将修改 `backend/api/api-admin/23-diet.md`，属于契约变更**：配置端点由 `/admin/diet/config` 改为 `/admin/diet/group-objects`，周食谱创建/列表增加集体对象字段。

## Capabilities

### New Capabilities

无。复用既有四个膳食能力，不新增能力边界。

### Modified Capabilities

- `diet-meal-config`: 供餐配置由园所单例改为按学年管理的集体对象；新增对象标识、就餐班级、列表/详情/新增/修改行为；既有餐次、占比、重量口径与快照规则改为按对象生效。
- `diet-meal-plan`: 周食谱创建 MUST 选择集体对象；列表与详情 MUST 返回所属对象；创建快照 MUST 固化对象身份与就餐班级。
- `diet-nutrition-analysis`: DRIs 目标值加权 MUST 只统计集体对象的就餐班级，而不是本园全部班级。

## Impact

- **Go 后端**：`server/internal/domain`（新增集体对象与关联模型）、`repository`、`service`（配置服务改造、周食谱创建与 DRIs 加权、种子/迁移）、`handler/http`、`app/router.go`、`app/provider.go`（AutoMigrate）、`middleware/admin_perm.go`、`service/role_catalog.go`、Wire provider/bind。
- **契约**：`backend/api/api-admin/23-diet.md`（配置章节、周食谱章节），`backend/api/api-admin/README.md` 索引不改。
- **admin**：`pages/diet/config.vue` 重写为列表 + 表单；`pages/diet/plans/index.vue` 新建向导增加集体对象；`api/diet.ts`、`types/api.d.ts` 同步。
- **测试**：`docs/tests/TEST_CATALOG.md` 增补集体对象相关 TC；service/repository/router 权限测试；admin Playwright 真实点击验证。
- **不涉及**：teacher / parent 小程序、食物库、菜谱库、营养报表口径与 Excel 结构、模板周、token 体系。

## Out of Scope

- 采购清单、餐后调查、膳食会议、月伙食计划、就餐人数管理。
- 集体对象删除与合并；对象级权限（沿用 `diet:view` / `diet:manage`）。
- 参考产品的「省级模板按对象/天数动态筛选」：模板过滤规则不变，不随对象变化。
- 6 岁档 DRIs 的班级映射（现有班级年级只有小/中/大，维持 6 岁权重为 0）。
- 数据许可门（仍为发布阻断项，与本变更无关）。
