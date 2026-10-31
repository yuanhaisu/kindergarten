# W9-C · 对抗性实现审查

- verdict: **PASS**（W9-C 首审 + W9-C2 复审）
- score: 94/100（首审 90 → W9-C2 修复 3 项 P2 后 94）
- 源码覆盖: `server/internal/service/teacher_portal_service.go`、`server/internal/service/parent_portal_service.go`、`server/internal/handler/http/teacher_portal.go`、`server/internal/handler/http/parent_portal.go`、`server/internal/service/teacher_growth_comments_test.go`、`server/internal/service/parent_growth_comments_test.go`、`backend/api/api-teacher/06-growth.md`、`backend/api/api-parent/04-growth.md`
- 审查范围: diff vs base、untracked/脏树归因、router/domain/repo 零 diff、service/handler 调用链、测试反事实、gofmt/vet/secrets/tokens
- 基线: `2658de4` → W9-C 提交 `3f16d74`（8 manifest 文件 + 本审查文档）→ W9-C2 在工作区未提交（7 文件）
- DevTools: N/A（backend slice，team-lead 明确排除；未重启 8080）

## 结论

两端评论接口按顶层分页：保持全量查询 + `buildCommentTree`，仅切顶层列表，回复随所属顶层返回；
`root_total` / `has_more` 与 `total`（含回复）语义分离。W9-C2 修复首审 3 项 P2 后，
极端 page 不再溢出 panic，两端分页参数口径一致，契约示例自洽。无 P1、无遗留 P2。

## DevTools 证据（N/A · 后端切片）

- 构建: `cd server && go build ./...` → exit 0
- Vet: `cd server && go vet ./internal/handler/http/ ./internal/service/` → exit 0
- Test: `go test ./internal/service/ -run 'TestTeacherGrowthComments_|TestTeacherCreateGrowthComment_|TestTeacherGrowthCommentLikes_|TestParentGrowthComments_|TestCommentPageWindow_' -count=1 -v -timeout 600s` → exit 0，12/12 PASS，`ok 36.640s`
- Token: `node scripts/check-tokens.mjs` → 14 条全仓存量违规，exit 0；本片 0 样式文件
- Secret scan: `node scripts/check-secrets.mjs --files <8 files>` → 0 findings
- 正常态: teacher 990411 / parent 9903xx 各 25 顶层 + 2 回复；page_size=10 → 10/10/5，root_total=25，has_more true/true/false，total=27；page1 不含第 2 页顶层及其回复，page2 含
- 空态: page=99 与 page=math.MaxInt → list 为 `[]`（NotNil+Empty），root_total=25，has_more=false，total=27，不报错、不 panic
- 错误态: N/A（无新增错误路径；越界按契约降级为空页）
- 交互: N/A（纯后端；handler 仅透传 `ppPage`）
- Console: N/A
- 路由: N/A；`router.go` / domain / repository / app 内容 diff 为零
- 截图: 未尝试（backend slice，无前端改动）
- HTTP 真实请求: 未执行（brief 禁重启 8080；8080 仍为 pid=41736 started 03:49:46 的旧二进制），响应形状由 service 测试逐字段断言

## W9-C2 复审（2026-10-31）

- **delta 范围**: 7 文件，均在本片 manifest 内（parent handler 未改，原本已用 `ppPage`）；`server/internal` + `backend/api` 中晚于 `3f16d74` 的写盘恰为这 7 个；HEAD 仍 `3f16d74`，worker 未 commit。
- **溢出修复** (`parent_portal_service.go:1981-2005`): `commentPageWindow` 先算 `pages=ceil(total/pageSize)` 并判 `page-1>=pages`，再相乘；`start≤total`，返回值恒在 `[0,total]`。teacher `teacher_portal_service.go:2165` 与 parent `parent_portal_service.go:2056` 共用。
- **handler 对齐**: `teacher_portal.go:699` 改用 `ppPage(c)`（默认 1/20，page_size 1-50），与家长端一致。
- **契约示例**: `06-growth.md:295`、`04-growth.md:209` 的 `has_more` 改为 `false`，与 `page=1&page_size=20&root_total=9` 自洽。
- **新增测试**: `TestCommentPageWindow_ClampsAndDoesNotOverflow`（10 组边界：首页/中页/末页/max int page/max int page_size/零值/空列表/恰好末页）；两端 `PaginatesRoots` 各加 `math.MaxInt` 用例。
- **反事实**: 旧实现 `(page-1)*pageSize` 在 `page=math.MaxInt` 时溢出为负 → slice panic，新增 MaxInt 断言必失败；窗口单测无函数时不可编译。修复语义被真实覆盖。
- 首审 3 项 P2（示例 has_more 矛盾、溢出 panic、page_size 上限不一致）全部 resolved。

## P1 · 阻塞问题 (0)

无。

## P2 · 建议 (0)

无遗留。首审 3 项已在 W9-C2 全部修复（见上）。

## Scope 说明

- 本片提交 `3f16d74` 恰含 8 个 manifest 文件 + 本审查文档；W9-C2 delta 7 文件未提交。
- 工作树另有大量 `M` 文件，多为 stat-cache 脏标记（blob hash 与 HEAD 相同）及 13:11 前其它在途切片改动；同期 `teacher/src/api/teacher-growth.js`（13:23）属并发 W9-D 的 Allowed 文件，不计入本片。
- 无 `pages.json` / `shared-ui/` / tokens / router / domain / repository 改动；未重启 8080。
