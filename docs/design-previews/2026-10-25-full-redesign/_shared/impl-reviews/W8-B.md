# W8-B · 对抗性实现审查

- verdict: **PASS**
- score: 94/100
- 源码覆盖: `server/internal/handler/http/parent_portal.go`、`server/internal/service/teacher_portal_service.go`、
  `server/internal/service/teacher_growth_like_test.go`（新）
- 审查范围: diff + untracked + handler/service 调用链 + 路由注册 + 唯一索引/软删语义 + 真实 MySQL 测试
- 基线: `9bd9d9d` → 工作区（HEAD 未变，worker 未 commit）
- 说明: 纯后端 P2 切片，按 brief Human decisions / team-lead 指示不做 HTTP 层与 WeChat DevTools

## 证据（后端等价）

- 构建: `cd server && go build ./...` → exit 0
- Vet: `go vet ./internal/handler/http/ ./internal/service/` → exit 0；`git diff --check` → 0
- Token: N/A（无前端/样式文件）
- Secret scan: `node scripts/check-secrets.mjs --files <3 文件>` → **0 findings**
- 测试: `go test ./internal/service/ -run 'TestTeacherGrowthLike' -v -count=1 -timeout 600s` → exit 0，
  4/4 PASS（顺序重复赞/8 并发双击/取消后再赞/清理残留）
- 复用覆盖: `TestParentSettings_RejectsInvalidTypesAndUnknownKeys`（6 子用例）+ 
  `TestUpdateAuthorizedPerson_RelationValidation` → exit 0；`go test ./internal/handler/http/` → exit 0
- 反事实: 阅读并发用例确认去掉 `OnConflict` 必触发唯一索引 `idx_log_user` 1062、`assert.NoError` 失败；
  worker 实测 5/8 报 1062，用例非空断言
- 清理: 逐用例日志 `growth_likes=1 logs=1 classes=1 users=1 kindergartens=1` 硬删（Unscoped）；
  残留用例断言五表全 0；下一用例 setup 复读均为 0
- 范围: `git diff --stat 9bd9d9d -- server` = 2 文件 +13/−1；untracked 仅 1 个测试文件；
  `backend/`、router、domain、repo、middleware、家长端 LikeGrowth 均无内容 diff
- 路由: 两条 PUT 已在 `router.go:514` / `:565` 注册（未改）；老师端无 tabBar 场景
- 截图: 不适用（纯后端）

## 修复核对

| 修复点 | 实现 | 依据 |
| --- | --- | --- |
| 更新授权人非法 relation | NotFound 之后新增 `ErrParentInvalidParams` → `ErrValidation`（422/42200） | `parent_portal.go:637-641`；service `parent_portal_service.go:797-800` |
| 更新设置非法分组/键/类型 | 新增同映射，其余 500 不变 | `parent_portal.go:1172-1177`；`apply` 用 `%w` 包装 sentinel（`:3693-3716`） |
| 老师 LikeGrowth 并发双击 | `Create` 加 `clause.OnConflict{DoNothing:true}`，与家长端同解 | `teacher_portal_service.go:2271-2272`；参照 `parent_portal_service.go:1822-1824` |
| 软删复活保留 | `DeletedAt.Valid` → `Unscoped().Update(deleted_at, nil)` 未改 | `teacher_portal_service.go:2277-2281` |
| 错误传播保留 | `Create` 失败仍 `return nil, e` | 同上 |

## P1 · 阻塞问题 (0)

无。

## P2 · 建议 (2)

1. **[evidence/已豁免]** 422 映射仅代码核对，未做 HTTP 实测（需登录 token，brief Human decisions 豁免）。
   service sentinel 已有测试覆盖；后续可补 httptest 断言 422/42200。
2. **[继承债务]** `teacher_portal_service.go` 基线即非 gofmt-clean（`:3405`、`:3988`），
   `gofmt -d` 与 base 输出一致；本片按外科手术式修改未顺手重排。

## 结论

PASS：3 个文件全在允许清单；三处 P2 修复与 brief 逐条一致；build/vet/4 新用例/2 组复用用例/handler 测试全绿；
并发用例经反事实判读有效；测试数据 Unscoped 硬删无残留；secret 0；未 commit。2 条 P2 不阻塞。
