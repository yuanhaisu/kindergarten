# OS-ALB-4 · 对抗性实现审查

- verdict: PASS（round 3）
- score: 88/100
- 源码覆盖: `parent_album_service.go`（新）、`parent_class_album_test.go`（新）、`parent_class_album_error_test.go`（新）、`album_repo.go`、`parent_portal.go`、`router.go`、`13-class-albums.md`（新）、`TEST_CATALOG.md`
- 基线: `d8dc567`；HEAD `19c2115` 为并发 OS-ALB-6a 提交（不含本切片文件）；本切片无 commit
- 后端切片，不使用 DevTools；未重启 8080

## 证据（round 3）

- `go build ./...` → exit 0；`go vet ./internal/...` → exit 0
- `go test ./internal/handler/http/ -count=1` → exit 0（0.185s；404/403/500 映射 PASS）
- `go test ./internal/service/ -run 'TestParentClassAlbum' -count=1 -timeout 600s` → exit 0（98.874s；16/16 PASS；查询数 6）
- Secret scan（8 文件）→ 0 findings
- 变异 1（Go `-overlay`，忽略 opt-out）→ exit 1（预期）：TC-0005 / counterfactual / MultiGuardian 三用例 FAIL（`:684/:685/:686`）
- 变异 2（Go `-overlay`，any-off→all-off）→ exit 1（预期）：`:684/:686` FAIL，A 看到 `{A,E}`；`:685` 通过
- 两次变异只改 `%TEMP%` 副本，源码未编辑；正式复跑全绿
- Token/Console/截图：不适用（后端切片）

## Round 3 增量

- `testPCALPhotoMultiGuard = 997179`，album 997157 内关联 `{A,E}`；album `photo_count=3`，清理范围覆盖 997179。
- `TestParentClassAlbumMultiGuardianAnyOptOut`：A 只看得到 control `{A}`，看不到 pair `{A,C}` 与 multiGuard `{A,E}`；E1/E2 只看得到 multiGuard（自己的孩子豁免）。旧 photo-7 断言已删除，不再冒充 opt-out 证据。
- 两个独立变异均证明 `{A,E}` 断言依赖「任一监护人关闭」语义：忽略 opt-out 或 all-off 都会让 A 看到该照片。
- 原 P1 关闭；底层 any-off 另有基线 `TestTeacherAlbumUnauthorizedCountAnyGuardianOptOut` 覆盖。

## P1 · 阻塞问题 (0)

无。

## P2 · 建议 (3)

### 1. [style] `parent_class_album_test.go` 未 gofmt

- 证据: `gofmt -l` 列出该文件；`gofmt -d` 仅 const 块三行对齐多一个空格。
- 建议: 提交前 `gofmt -w`；不影响 build/vet/test，不在 brief 验收门内。

### 2. [test-evidence] 新路由无路由级 401/未绑定 403 用例

- 证据: `parent_portal.go:1024-1035` 的映射已测；`parentCtx` 403 与 AuthJWT 401 未触达（worker 已声明）。
- 建议: 补 httptest 路由级用例或在 OS-ALB-7 端到端覆盖。

### 3. [scope-record] 实际 8 文件 vs team-lead 消息 7

- 证据: `parent_class_album_error_test.go`；主会话编排、无 commit、无其它越界。
- 建议: checkpoint 按 8 文件登记；若坚持 brief 白名单需主会话 waiver。

## 残余风险

- 未做 HTTP 运行态；路由注册与 Casbin 通配静态核对通过。
- 90 天整端到端未测（纯函数固定 now 覆盖）。
- 列表保留 `visible_photo_count=0` 的已发布相册（契约已写明）。
