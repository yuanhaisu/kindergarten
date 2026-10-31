# W9-C · 对抗性实现审查

- verdict: **PASS**
- score: 90/100
- 源码覆盖: `server/internal/service/teacher_portal_service.go`、`server/internal/service/parent_portal_service.go`、`server/internal/handler/http/teacher_portal.go`、`server/internal/handler/http/parent_portal.go`、`server/internal/service/teacher_growth_comments_test.go`、`server/internal/service/parent_growth_comments_test.go`、`backend/api/api-teacher/06-growth.md`、`backend/api/api-parent/04-growth.md`
- 审查范围: diff vs base、untracked/脏树归因、router/domain/repo 零 diff、service/handler 调用链、测试反事实、gofmt/vet/secrets/tokens
- 基线: `2658de4` → 工作区（8 文件，未提交；HEAD 仍为 `2658de4`）
- DevTools: N/A（backend slice，team-lead 明确排除；未重启 8080）

## 结论

两端评论接口已按顶层分页：保持全量查询 + `buildCommentTree`，仅对顶层列表切片，
回复随所属顶层返回；`root_total` / `has_more` 与 `total`（含回复）语义分离且契约同片更新。
独立重跑 11/11 用例通过，page1/2/3、回复归属、越界页均有断言；无 P1。

## DevTools 证据（N/A · 后端切片）

- 构建: `cd server && go build ./...` → exit 0
- Vet: `cd server && go vet ./internal/handler/http/ ./internal/service/` → exit 0
- Test: `go test ./internal/service/ -run 'TestTeacherGrowthComments_|TestTeacherCreateGrowthComment_|TestTeacherGrowthCommentLikes_|TestParentGrowthComments_' -count=1 -v -timeout 600s` → exit 0，11/11 PASS，`ok 35.631s`
- Token: `node scripts/check-tokens.mjs` → 14 条全仓存量违规，exit 0；本片 0 样式文件
- Secret scan: `node scripts/check-secrets.mjs --files <8 files>` → 0 findings
- 正常态: teacher 990411 记录 25 顶层 + 2 回复；page_size=10 → 10/10/5，root_total=25，has_more true/true/false，total=27；page1 不含第 2 页顶层及其回复，page2 含
- 空态: page=99 → list 为 `[]`（非 null，测试 `NotNil`+`Empty`），root_total=25，has_more=false，total=27，不报错
- 错误态: N/A（无新增错误路径；越界按契约降级为空页）
- 交互: N/A（纯后端；handler 仅透传 `teacherPortalPage` / `ppPage`）
- Console: N/A
- 路由: N/A；`router.go` / domain / repository / app 内容 diff 为零
- 截图: 未尝试（backend slice，无前端改动）
- HTTP 真实请求: 未执行（brief 禁重启 8080；实测 8080 进程 started 03:49:46，早于本片 13:12 改码，运行中为旧二进制），响应形状由 service 测试逐字段断言

## P1 · 阻塞问题 (0)

无。

## P2 · 建议 (3)

### 1. [文档] 两端契约示例 `has_more` 与 `root_total/page_size` 矛盾

- 证据: `backend/api/api-teacher/06-growth.md:241,294-295`、`backend/api/api-parent/04-growth.md:179,208-209`；Query 为 `page=1&page_size=20`、`root_total=9`，却写 `has_more=true`。按本片语义 `end=min(20,9)=9`，应为 false；示例 list 也少于 root_total 且无省略号。
- 建议: 示例改为 `has_more=false`，或调整 page_size/root_total 使三者自洽。

### 2. [健壮性] 超大 page 触发 int 溢出并 panic（500）

- 证据: `server/internal/service/teacher_portal_service.go:2167-2173`、`parent_portal_service.go:2030-2036`。独立复现 `page=922337203685477581&page_size=20`：`(page-1)*pageSize` 溢出为 -16，`all[-16:4]` panic；gin.Recovery 兜成 500，违反「越界页返回空 list、不报错」。现有测试仅覆盖 page=99。
- 建议: 乘法前限制 page 上界（或按 `rootTotal/pageSize` 判定直接空页），确保 start 永不为负。

### 3. [一致性] 两端 page_size 实际上限不一致

- 证据: `teacher_portal.go:40-44` 原样透传，service `pagination.Normalize` 上限 100；`parent_portal.go:69-79` 的 `ppPage` 接受 1-50（>50 回落 20）。team-lead 期望「上限 50」仅对家长端成立；契约未写上限。属既有助手行为，worker 已披露。
- 建议: 契约写明上限，或统一两个页参数助手后再复用；不阻塞本片。

## Scope 说明

- 工作树另有大量 `M` 文件，多为 stat-cache 脏标记（blob hash 与 HEAD 相同）及 13:11 前其它在途切片的前端/tokens/logo 改动；本片可归因写入仅 manifest 8 文件（mtime 13:12:46–13:13:28）。
- 无 commit；无 `pages.json` / `shared-ui/` / tokens / router / domain / repository 改动；未重启 8080。
