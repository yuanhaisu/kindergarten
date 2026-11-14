# AQR-S9a · 对抗性实现审查

- verdict: **PASS**
- score: 95/100
- 源码覆盖: `server/internal/service/settings_service.go`、`server/internal/service/settings_service_test.go`、`backend/api/api-admin/18-settings.md`、`admin/api/settings.ts`
- 审查范围: 4 文件语义 diff + `gofmt(base)` 对照 + handler/route/RBAC/domain/迁移面 + `diet_plan_service` 下游调用链 + admin 类型/调用方
- 基线: `f20560a` → 工作区未提交（后端 + admin 类型 slice，无 DevTools）

## 证据（无 DevTools）

- 构建: `cd server && go build ./...` → **exit 0**；`go vet ./...` → **exit 0**
- 测试: `go test ./internal/service -run 'Settings|AcademicYear' -count=1 -timeout 600s -v` → **exit 0**，`ok 48.950s`；`TestListAcademicYears_IncludesID` PASS
- gofmt: `gofmt -l` 两个 Go 文件 → **无输出**
- Secret scan: 4 个 manifest 文件 → **0 findings**
- Admin build: `cd admin && npm run build` → **exit 0**，`Build complete!`，1.98 MB
- 范围: worker 时段内被改文件恰为 manifest 4 个；`git status --porcelain admin/` 仅 `admin/api/settings.ts`；无 router/Casbin/domain/migrations 改动
- Token/DevTools/截图: N/A（无样式或无页面变更）；admin build prebuild 刷新 token 生成时间戳行属既有行为

## 端到端链路

- Service: `settings_service.go:360-365` 新增 `ID uint64 \`json:"id"\``，`:381` 由 `r.ID` 填充；`BaseModel.ID` 为主键，`Find` 回填无零值风险。
- JSON: `settings_service_test.go:120-146` 断言 `ID == 991203` 且 marshal 含 `"id":991203`。
- Handler: `handler/http/settings.go:63-79` 无独立 DTO，`response.OK` 直传；信封 `data.list[].id` 生效。
- 契约: `18-settings.md:111-119` 两行示例均含 `id`，该段无字段表需同步。
- Admin: `admin/api/settings.ts:23` 类型含 `id: number`；`types/api.d.ts` 无学年列表类型；唯一调用方设置页结构子类型兼容，构建通过。

## S9 下游

`POST /admin/diet/plans` 要求数值 `academic_year_id > 0` 且按主键匹配（`diet_plan_service.go:264-273, 879-891`）。空库路径：设置页新增学年（`settings/index.vue:243-255`）→ 首条自动 `is_current=true`（`settings_service.go:439`）→ GET 返回其 `id`；初始 `super_admin` 通配权限可访问。S9 仍需自行处理空学年列表引导与 `diet:*` 角色的 403 风险（权限本 slice 明确不改）。

## P1 · 阻塞问题

无。

## P2 · 建议

1. **证据表述** `validators.md:11` 的「git status 仅 4 文件」与全仓 151 条 porcelain 不符；主管已用 mtime 窗口 + `gofmt(base)` 语义 diff 证明实际改动仅 manifest 4 文件。
2. **文档漂移** `openspec/.../design.md:186` 仍称学年接口「不改」；建议编排器用 OpenSpec update 工作流补注「S9a 追加数值 id」。
3. **类型漂移** `admin/pages/settings/index.vue:141` 本地 `yearList` 未含 `id`（不在 Allowed）；当前无功能影响。
4. **证据缺口** 无真实 HTTP 探测；handler 直传 + service JSON 断言已覆盖序列化面，留待 S10 E2E 间接验证。

## diff 卫生

`settings_service.go` 的 `gofmt -w` 产生约 35 行空白对齐噪声；对 base 单独 gofmt 后与当前文件 diff，结果仅 `ID` 字段、注释与 `r.ID` 填充三处。该文件在 base 即非 gofmt-clean，只加字段无法通过「`gofmt -l` 无输出」验收，故属必要规范化。`shared-design-tokens/build/scss/*.scss` 的时间戳变化来自 admin build prebuild（AQR-S8 已记录），不在 manifest。

## 结论

数值 `id` 从 service DTO 经 JSON、契约到 admin 类型全链路一致；构建、vet、定向测试、gofmt、secret 门全部通过；manifest 与 worker 实际改动一致，无路由/权限/表结构改动。S9 的下游阻塞已解除。判 **PASS**。
