# Proposal

## Why

成长记录的产品定位是**家长信任核心**（产品方案 3.2），它必须能回答「我的孩子在这里被怎么对待」。但当前实现里，这个信任建立在两处裂缝上：

1. **可见性规则没有单一实现点，三个入口已实际穿透**：家长端的详情接口有正确的 `verifyGrowthAccess` 校验，但首页动态、成长相册、成长报告三个入口各写各的查询——首页动态不带 `target_type` 过滤（发给别人孩子的记录会出现在全班家长首页）、相册子查询缺班级限制（大班家长能看到全园所有班级的照片）、报告接口无租户与归属校验（改一个 `student_id` 即可读任意园任意孩子的报告）。这直接违反产品方案第六节「儿童照片仅对授权家长可见，不对外公开」。
2. **老师端是「富数据 UI + 贫数据接口」的错位**：老师端列表/详情按设计稿渲染封面、发布者、互动统计、相关孩子，而后端只返回 14 个基础字段；详情接口不读媒体，导致老师看不到自己发布的照片。发布页「编辑」按钮跳到创建页而创建页忽略 `id` 参数——编辑实际会新建一条记录。

后果是一致的：家长端隐私边界失守，老师端看不到自己的发布成果，而「老师不废弃才算成功」是产品的首要成功标准。

## What Changes

**P0 — 可见性统一（家长端）**

- 首页动态、相册、成长报告、未读计数四个入口，统一采用与详情接口一致的可见性规则：家长只能看到「自己孩子所在班级的全班记录」与「指向自己孩子的个体记录」
- 报告接口增加幼儿园与孩子归属校验，并排除生成中/失败的报告
- 可见性规则收敛为单一实现点，四个入口共用

**P1 — 老师端发布与预览闭环（老师端）**

- 老师列表/详情接口补齐设计稿要求的响应字段（发布者、媒体、互动统计、相关孩子），与既有契约 `backend/api/api-teacher/06-growth.md` 对齐
- 发布页的编辑入口真正进入编辑态（加载既有记录、提交走更新），而非再创建一条
- 老师列表卡片只渲染有真实数据来源的内容：去掉无来源的浏览数、分类筛选与统计数字，同步状态改用 `sync_to_feed` 字段驱动

**明确不做什么（Out of scope）**

- **不做**老师端照片上传落库与 `log_id` 挂载（照片管道属后续变更）
- **不做**家长端相册/报告页面的数据契约对齐（页面渲染仍为空壳，另案处理）
- **不做**审核流定位变更（审核保持事后台账）与 `sync_to_feed` 的查询过滤接线（另案）
- **不做**照片级学生关联（`growth_media_students` 写入路径）与 AI 识别
- **不做**家长端成长报告页、相册页的视图重排
- **不做**admin 管理端的任何变更
- **不做**老师端列表的分类法定义（`docs/openspec-usage.md` §10 已记录：分类法取值是待产品决策的阻塞项）

**契约变更声明**

本次将修改 `backend/api/api-teacher/06-growth.md`，属于契约变更。变更性质是**对齐而非扩张**：列表/详情响应所需的字段已在既有契约中定义（`creator`、`media_preview`、`likes_count`、`comments_count`、`related_students_count`、`related_students`、`media`），实现未落地。同时删除契约中已被设计证明为冗余的字段（`shares_count`：设计稿无分享计数；`media_count`：可由 `media_preview` 派生）。字段级溯源见 design.md。

## Capabilities

### New Capabilities

- `growth-visibility`: 家长端成长记录的可见性规则——由「记录 → 孩子 → 家长」的关联链决定，并在家长端所有入口（列表流、首页动态、相册、报告、未读计数）保持一致
- `teacher-growth-authoring`: 老师端成长记录的发布、编辑与预览——发布者与媒体可见、编辑入口语义正确、列表仅承载有数据来源的内容

### Modified Capabilities

（无。现有 `growth-feed` 规格描述的是家长侧卡片的呈现与契约字段，本次不改其行为需求。）

## Impact

**后端（Go）**

- `server/internal/service/parent_portal_service.go` — 首页动态、相册、报告、未读计数四处查询；抽取统一可见性规则
- `server/internal/service/teacher_portal_service.go` — 列表/详情响应映射补齐；补互动统计与相关孩子查询
- `server/internal/repository/growth_repo.go` — 如有批量查询需要（复用 parent 侧已有的批量取数模式，避免 N+1）

**契约**

- `backend/api/api-teacher/06-growth.md` — 列表/详情响应字段对齐（见上方声明）

**老师端小程序**

- `teacher/src/pages/growth/index.vue` — 列表卡片按真实字段渲染，去掉无来源内容
- `teacher/src/pages/growth/detail.vue` — 消费详情响应中的媒体/发布者/统计/相关孩子
- `teacher/src/pages/growth/create.vue` — 支持编辑态（读取 `options.id`，提交走更新）
- `teacher/src/api/teacher-growth.js` — 接上已定义未调用的接口

**测试**

- `docs/tests/TEST_CATALOG.md` — 补录新 TC（TC-NEW-VIS-01~04、TC-NEW-TGR-01~02），TC-GRW-005 状态回填

**不改动**

- `server/internal/domain/` — 无表结构变更，无需 `AutoMigrate` 或手工迁移
- `server/internal/app/router.go`、`rbac_policy.csv`、`wire.go` — 无新增路由/依赖，无需 `make wire`
- `parent/`（小程序视图层）、`admin/` — 无改动
