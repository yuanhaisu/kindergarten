# W9-B · 对抗性实现审查

- verdict: **PASS**
- score: 94/100
- 源码覆盖: `server/internal/service/parent_portal_service.go`、`server/internal/handler/http/parent_portal.go`、`server/internal/service/parent_growth_report_test.go`（M）、`server/internal/service/parent_daily_designation_test.go`（新）
- 审查范围: diff vs base、untracked、全仓 scope、service/handler 调用链、测试反事实复跑、gofmt/vet/secrets
- 基线: `e9dc87b` → 工作区（3 M + 1 ??，未提交；HEAD 仍为 `e9dc87b`）
- DevTools: N/A（backend slice，任务明确排除；未重启 8080）

## 结论

两处技术债均已收口：报告正文查询增加 `period_start >= Start AND period_start < EndExcl`，
保留 student/kg/status/report_type 与 `created_at DESC`，无正文周期仍返回四格 + 空 dimensions；
`ParentPickupDesignation` 把 `ErrParentInvalidParams` 映射为 422（`ErrValidation`），其它错误仍 500。
新用例经反事实验证：在 `e9dc87b` 旧代码上会取到上一周正文而失败，说明断言语义真实。无 P1。

## 证据

- Build/Vet/gofmt: `cd server && go build ./...` / `go vet ./internal/handler/http/ ./internal/service/` → exit 0；4 文件 `gofmt -l` 无输出
- Test: `go test ./internal/service/ -run 'TestGetDailyDesignation|TestParentGrowthReport' -count=1 -timeout 600s -v` → exit 0，14 顶层 + 4 子用例 PASS（`ok 83.994s`）
- Handler: `go test ./internal/handler/http/ -count=1` → exit 0（`ok 0.377s`）
- 反事实: `git worktree add --detach e9dc87b` + 新测试 → `BodyMatchesRequestedPeriod` FAIL（expected 2026-03-16/语言，actual 2026-03-09/运动）；worktree 已删除，项目树未动
- Secret: `node scripts/check-secrets.mjs --files <4 files>` → 0 findings；文件内无 DSN/口令（`PasswordHash:"x"` 为合成夹具）
- Token: 全仓 14 条存量违规（exit 0）；本切片 0 样式文件，不相关
- 清理: 夹具全部 `Unscoped()` 硬删；`CleanupLeavesNoResidue` 覆盖 14 表断言 0 行，且在反事实跑后重跑通过
- Scope: `backend/`、`domain/`、`repository/`、`app/` 零 diff；工作树其余脏项为 stat-cache 或 2026-10-27~29 前序 UI 改动，非本切片

## P1 · 阻塞问题（0）

无。

## P2 · 建议（3，均不阻塞）

1. handler 422 映射无 handler 级自动化用例（测试文件不在 Allowed；验收允许代码路径核对）——后续可补 httptest。
2. 周期过滤依赖未来写入方将 `PeriodStart` 对齐到周期起点（周一起始）；当前无写入方，新增时需固化该口径。
3. self-review 引用 handler 行号 1222-1225，实际为 1223-1225（文档细节，不影响结论）。
