# Tasks

> 分组对应 design.md 的「跨端触点清单」。每组末尾有验证任务；未验证不得进入下一组、不得 archive。
> `[P]` = 与其他 `[P]` 任务可并行。

## 1. 后端 — service 派生 scope

- [x] 1.1 `server/internal/service/parent_portal_service.go`：新增纯函数 `feedScope(targetType int8) string`（`2 -> "individual"`，其他 -> `"class"`），并在 `assembleFeedItems` 的条目 map 中输出 `scope`；不新增 DB 列、不改查询；验证：`go build ./...` 通过，`go test ./internal/service/ -run 'TestFeedScope|TestAssembleFeedItems'` 通过
- [x] 1.2 [P] 在 `server/internal/service/parent_portal_feed_test.go` 补单测：`feedScope(1)=class`、`feedScope(2)=individual`、未知值兜底 `class`；`assembleFeedItems` 对 `TargetType=1/2` 的条目分别输出 `scope=class/individual`，且不回归既有字段断言（TC-GRW-099/100/101）；验证：上述单测全绿
- [x] 1.3 验证（后端组）：`go vet ./...`、`go test ./internal/service/...` 通过；对照 spec 的「列表条目提供记录范围」3 个 Scenario 逐条确认

## 2. 契约 — 列表新增 scope

- [x] 2.1 [P] `backend/api/api-parent/04-growth.md` 列表段：在响应示例与字段说明中加入 `scope`，注明 `individual` / `class` 及「由 target_type 派生、additive、非破坏性」；验证：文档字段与 `assembleFeedItems` 实际输出逐字对齐
- [x] 2.2 [P] 在契约顶部补一条变更说明：本次为 additive 契约变更，旧客户端可忽略 `scope`；验证：`openspec validate --all` 通过，且契约中无 `target_type`/`related_students` 回填到列表
- [x] 2.3 验证（契约组）：对照 design.md 的「契约字段清单」，确认 `scope` 是唯一新增字段、来源可追溯、无冗余字段；无 `scope` 的设计稿/业务场景缺口

## 3. 家长端 — 使用 scope 修正两模式

- [x] 3.1 `parent/src/pages/growth/index.vue`：`mediaFor(item, mode)` 在 `scope === "individual"` 时返回全部照片（不做 `student_ids` 过滤）；`scope === "class"` 时保持现有点名过滤；`scope` 缺失时退回现有点名过滤（兼容旧后端/回滚）；验证：构建通过，代码路径可被 5.3 的 UI 用例覆盖
- [x] 3.2 同文件：`isVisibleInChildView` 对 `scope === "individual"` 返回 true；`ownerKind/ownerLabel` 对个体记录输出「专属记录」，对全班记录保留「含孩子 / 集体 · 含孩子 / 其他孩子 / 班级记录」；验证：构建通过，UI 用例确认标签与可见性
- [x] 3.3 同文件：更新顶部注释，删除「列表不返回 target_type」的旧兜底说明，改为「scope 由列表提供；缺失时保守退回」；验证：注释与 design.md D1/D3 一致
- [x] 3.4 验证（家长端组）：`npm run build:mp-weixin` 通过（`DONE Build complete.`）；`node scripts/check-tokens.mjs parent` 对 `growth/index.vue` 0 新增违规；`openspec validate --all` 通过

## 4. 测试目录与文档

- [x] 4.1 [P] `docs/tests/TEST_CATALOG.md`：新增 TC-GRW-099~104，名称/优先级/类型/状态按 spec 场景一一对应；验证：6 个 TC ID 全部存在且与 spec 引用一致
- [x] 4.2 [P] `docs/growth-feed-field-review.md` §6.1 追加「scope 回写」：说明原 `target_type` 被删后，两模式催生了展示层 `scope`；并同步 `docs/handoffs/2026-10-19-parent-growth.md` 的 7.1 节，移除「已知限制」中的兜底描述；验证：两处文档都不再把列表描述为「无范围字段」
- [x] 4.3 验证（文档组）：`openspec validate --all` 通过；TC ID 与 spec 场景的引用无缺失

## 5. 集成与真实交互验证

> 2026-10-20：测试环境已启动（本地 MySQL 3316 / Redis 6379 / Go 8080），用真实 token 调 API；微信开发者工具自动化可用 `switchTab`/`evaluate`，但该 IDE 实例的 `tap`/`element_info`/`screenshot` 超时（已知自动化限制），模式切换改用编译后的真实 click handler 并核对 `page.data`。混合点名 fixture 当前测试库没有，TC-GRW-103 留待补数据/真机点击。

- [x] 5.1 后端集成验证：启动真实后端（本地 MySQL/Redis），请求 `GET /api/v1/parent/growth/feed`；实测 7 条记录：id 21/22/27/28/29 为 `scope=class`，id 23/24 为 `scope=individual`；id 23 `media=1` 且 `student_ids=[]`，id 21 `media=2` 且 `student_ids=[]`；TC-GRW-099/100/101 通过
- [x] 5.2a 家长端运行时验证：只看孩子 6 条 / 1 张；个体 id 23（未点名照片）可见并标「专属记录」；全班 id 21（两张未点名照片）隐藏；看全部（编译后的真实 click handler `e1_09`）7 条 / 3 张，id 21 可见；TC-GRW-102/104 通过
- [x] 5.2b 补「部分照片点名当前孩子、部分未点名」的班级记录 fixture（log `991000522`，media `991000522`/`991000523`，前者确认到学生 27），在新 IDE 实例做真实鼠标点击（屏幕坐标 `501,391`）切到只看孩子；截图 `runtime-child.png` 显示仅 1 张照片 + 「还有 1 张班级照片，切到「看全部」查看」；`runtime-all.png` 显示 2 张照片；TC-GRW-103 通过
- [x] 5.3 收尾：`go build ./...`、`go test ./internal/service/...`、`npm run build:mp-weixin`、`check-tokens`（growth 文件 0 新增违规）、`openspec validate --all`、`openspec doctor` 全部通过；确认没有残留 `dev:mp-weixin` watcher；逐条对照 specs 完成
