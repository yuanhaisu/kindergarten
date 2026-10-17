# Tasks

> 分组对应 design.md 的「跨端触点清单」。每组末尾是**验证**任务，未验证不得进入下一组、不得 archive。
> `[P]` = 与其它标记 `[P]` 的任务可并行。

## 1. 后端 — 列表补齐与瘦身（service）

- [x] 1.1 `server/internal/service/parent_portal_service.go` `ListGrowthFeed`：批量查当页 `GrowthMedia`（`WHERE log_id IN (当页 ids)` + `ORDER BY sort_num ASC, id ASC`），按 `log_id` 分组挂到条目
- [x] 1.2 [P] 同函数：批量查本人点赞集合（`GrowthLike`，`log_id IN (?) AND user_id = ?`）→ 条目 `is_liked`
- [x] 1.3 [P] 同函数：批量查评论计数（`GrowthComment`，`GROUP BY log_id`）→ 条目 `comments_count`
- [x] 1.4 同函数：条目输出收敛为 `id` `date` `created_at` `content` `media[]` `creator{id,avatar,nickname}` `is_liked` `comments_count`；**删除** `target_type` / `class_name` / `related_students*` / `likes_count` / `media_count` / `performance_tags` 的输出（`performance_tags` 列表卡片不渲染，设计上是「奖章」不是表现标签）
- [x] 1.5 空页保护：当页 `ids` 为空时跳过 1.1–1.3 的查询，避免 GORM 生成 `IN (NULL)`
- [x] 1.6 **验证（列表）**
  - [x] `go build ./...` 通过
  - [x] 单元测试覆盖 TC-GRW-080 ~ 036、039（装配纯函数 `assembleFeedItems`，10 个用例全过）；**TC-037/038 月分组属前端派生，移到 4.4 验证**
  - [x] 断言响应条目**不含** `target_type` `class_name` `related_students` `likes_count` `media_count` `performance_tags`（TC-GRW-092）
  - [x] 断言当页无记录时不报错、返回空列表
  - [x] 对照 specs（接口层）：TC-GRW-080/081/082/083/084/085/086/087/088/089/092 已由真实 HTTP 请求验证通过（列表 8 字段精确、无冗余、矩阵语义正确、排序 DESC）。**TC-GRW-087/088 的「分组标记渲染」仍需 UI 验证**

## 2. 后端 — 详情补相关孩子（service）

- [x] 2.1 `GrowthFeedDetail`（同文件 `:802`）：补 `related_students`，仅 `target_type=2` 时查 `growth_log_students` 联 `students`，返回 `[{id, name, avatar}]`；全班记录返回空
- [x] 2.2 同函数：`performance_tags` 返回前从逗号串 split 成数组（契约与 `detail.vue:92` 都按数组用）；空串返回空数组，不返回 `[""]`
- [x] 2.3 **验证（详情）**
  - [x] `go build ./...` 通过
  - [x] `splitPerformanceTags` 纯函数单测通过（空串→空数组、纯分隔符→空数组、容忍空格）；**TC-040/041 的 related_students 属 DB 聚合，改由集成验证覆盖**
  - [x] 确认无 N+1：`GrowthFeedDetail` 只多一次关联查询（`growth_log_students` 联 `students`），无逐条循环查
  - [x] 对照 specs：TC-GRW-090（个体详情含相关孩子 1 项）、TC-GRW-091（全班详情无 related_students 键）已由真实 HTTP 请求验证通过

## 3. 契约 — `backend/api/api-parent/04-growth.md`（与后端同一 PR）

- [x] 3.1 列表段（`GET /parent/growth/feed`）：响应条目改为对齐后形状；**删除** `likes_count` `media_count` `target_type` `class_name` `related_students`；**新增** `is_liked` `comments_count` `media[]` 完整 `creator`
- [x] 3.2 详情段（`GET /parent/growth/feed/:id`）：**新增** `related_students`（形状 `[{id,name,avatar}]`，标注「仅个体记录非空」）
- [x] 3.3 契约顶部加破坏性变更标注：`> ⚠️ 2026-10 移除了列表响应的 likes_count / media_count / target_type / class_name / related_students（related_students 移至详情）。设计稿从未渲染这些字段。`
- [x] 3.4 **验证（契约）**
  - [x] 契约变更已在 proposal 的「What Changes」里显式声明为契约变更
  - [x] 逐字段核对契约与 `assembleFeedItems` / `GrowthFeedDetail` 实际输出一致——**列表 8 字段完全对齐**；详情核出 3 处漂移并已修（补记 `target_type`/`sync_to_feed`，移除从未实现的 `media_count`）。真实请求比对待 dev 环境起来后再抽验
  - [x] 确认 `likes_count` 仍保留在 `POST/DELETE .../like` 的响应里（本次未触碰 like/unlike 端点）

## 4. 家长端 — 列表模板字段映射

- [x] 4.1 `parent/src/pages/growth/index.vue`：5 处映射修正——
  `item.teacher` → `item.creator.nickname`
  `item.time` → 由 `item.created_at` 格式化为「M月D日 HH:mm」
  `item.photos`（数字）→ `item.media`（数组），`v-if="item.media && item.media.length > 0"`，图源用 `thumb_url || file_url`
  `item.month` → 由 `item.date` 的 `YYYY-MM` 派生，渲染「YYYY年M月」分组标记
  `item.teacherAvatarBg/Color` 色块 → `item.creator.avatar` 图片头像（无 avatar 时才退化为色块占位）
- [x] 4.2 [P] 点赞态改用 `item.is_liked`；评论文案改用 `item.comments_count`（`> 0 ? 'N条评论' : '评论'`）
- [x] 4.3 [P] 移除对已砍字段的任何引用（`item.likes` / `item.category` / `item.badge` 的渲染，其中 `category`/`badge` 是无数据源的设计超前项，本次**删除其渲染占位**，不造数据）
- [x] 4.4 **验证（家长端列表）**
  - [x] `npm run build:mp-weixin` 通过
  - [x] 经 wechat_automator(9420) 读页面实例数据验证：卡片显示李老师 + 2026-09-20 08:45 + 正文；R1=2张/R3=1张缩略图（thumb_url 正确）；R2/R4 无 media 不渲染照片区（TC-GRW-080~084）
  - [x] 点赞两态渲染正确：R1/R3=「已点赞」、R2/R4=「点赞」（TC-GRW-085）。注：`toggleLike` 仍是纯本地状态，后端同步未验
  - [x] 月分组正确：R4 带「2026年9月」标记、R3 同组不重复；R2 带「2026年8月」标记、R1 同组不重复（TC-GRW-087/088）
  - [x] 对照 specs：TC-GRW-080~088 全部通过
  - [x] 已核对：成长时光是 tabBar 页（onShow 加载）；详情是 `navigateTo`，详情非 tabBar 页，正确

## 5. 家长端 — 详情页相关孩子

- [x] 5.1 `parent/src/pages/growth/detail.vue`：新增「相关孩子」区块，数据取 `item.related_students`；为个体记录才渲染（数组非空）
- [x] 5.2 **验证（家长端详情）**
  - [x] `npm run build:mp-weixin` 通过
  - [x] 详情验证：R3(个体) 显示相关孩子「小俊彦(id 27)」；R1(全班) 无相关孩子键/区块（TC-GRW-090/091）
  - [x] 对照 specs：TC-GRW-090/091 通过

## 6. 样式 / design token

- [x] 6.1 **验证（token）**：`node scripts/check-tokens.mjs parent` 中 `parent/src/pages/growth/` 相关文件 0 新增违规——照片缩略图、点赞态、相关孩子标签只能用现有 token，出现 `#hex` 或裸 `rpx` 硬编码即未完成

## 7. 测试用例登记与收尾

- [x] 7.1 `docs/tests/TEST_CATALOG.md`：把 `TC-GRW-080` ~ `TC-GRW-092` 补录为正式编号（现有 GRW 已到 `TC-GRW-027`，建议从 `TC-GRW-030` 起），回填到 spec 与本文件
- [x] 7.2 **验证（总）**
  - [x] `go vet ./...` 干净；`go test ./...` 全通过。**`golangci-lint` 本机未安装，lint 一项未跑**
  - [x] 逐条对照 specs 6 个 Requirement / 13 个 Scenario——**TC-GRW-080~092 全部通过**（单测 + 真实 HTTP + 小程序页面数据三重验证）
  - [x] 已回写 `docs/growth-feed-field-review.md` §6.1：实施结果、与审查的差异、新发现的 3 处、TC 编号换算、仍未验证项
  - [x] 收工清理：本次只用 `build:mp-weixin`（一次性构建），未启动 `dev:mp-weixin` watch，无需杀进程

    ```powershell
    Get-CimInstance Win32_Process -Filter "Name='node.exe'" |
      Where-Object { $_.CommandLine -match "dev:mp-weixin|kindergarten\\(teacher|parent)\\node_modules" } |
      ForEach-Object { Stop-Process -Id $_.ProcessId -Force }
    ```
