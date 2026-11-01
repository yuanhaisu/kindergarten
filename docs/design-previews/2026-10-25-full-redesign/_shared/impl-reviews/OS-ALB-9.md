# OS-ALB-9 · 对抗性实现审查

- verdict: PASS
- score: 90/100
- 源码覆盖: `server/cmd/albumretention/main.go`（新）、`server/internal/service/album_retention.go`（新）、`album_retention_test.go`（新）、`student_status_time_test.go`（新）、`student_service.go`、`teacher_album_service.go`、`parent_album_service.go`、`parent_portal_service.go`、`docs/tests/TEST_CATALOG.md`
- 审查范围: 工作树 diff + 4 个新文件 + 清理/埋点调用链 + 契约（`api-admin/03-students.md`）+ 指标调用点正反路径
- 基线: `ebc2100`；审查时 HEAD 已推进至 `35326bb`（OS-ALB-6b，只改 teacher UI/docs，不触及本切片）；9 个文件仍全部未提交，worker 未 commit

## 证据

- 构建: `cd server && go build ./...` → exit 0；`go vet ./internal/... ./cmd/albumretention/` → exit 0
- Lint: `golangci-lint run ./internal/service/... ./cmd/albumretention/...` → 0 issues
- 单测: `go test ./internal/service/ -run 'TestAlbumRetention|TestAlbumMetric|TestStudentStatusChangedAt|Test.*EmitsMetricEvent' -count=1 -timeout 600s` → 10 个 Test 全 PASS，51.237s；读侧留存 `TestParentClassAlbumGraduationWindow|TestParentClassAlbumRetentionDenied` → PASS
- Secret scan: `node scripts/check-secrets.mjs --files <9 files>` → 0 findings；gofmt 无输出
- 清理服务: 181 天删 / 179 天留 / 未删留；照片与两级关联全删；`operation_logs` 保留；二次执行 0；dry-run 不删；batch=2 删完 5 本（真实 MySQL + Unscoped 残留计数）
- 命令实测（自建 9969xx 夹具，执行前 albums 表为空）: `-dry-run` 输出 1/2/2/1 且未删；`-batch-size 1` 清理 1 本；复查 albums/photos/album_students/photo_students=0、audit_logs=1；再跑清理 0 本
- 错误态: 临时 `CONFIG_PATH` + 不可达假 DSN（含假口令）→ exit 1，输出无 DSN/口令；`CONFIG_PATH` 覆盖与 `-batch-size` 均生效
- 学生状态: 1→2 写 now、2→2 不刷新、2→1 清空、1→1 保持空、1→3 写、历史 2 空值不回填；编辑路径不覆盖已有时间戳
- 指标: 三事件精确键集合、无 phone/openid；真实 service 调用点——发布发/建草稿与下架不发、详情成功发/404 不发、真实开→关发/重复关闭与重新开启不发
- 残留: 9962xx/9963xx/9969xx/9971xx 夹具全 0，`albums` 表总数 0
- DevTools / 截图 / 交互: N/A（纯后端切片，禁止重启 8080）

## P1 · 阻塞问题 (0)

无。

## P2 · 建议 (4)

### 1. [contract] `UpdateStudent` 无法触发状态迁移
- 证据: `server/internal/service/student_service.go:620`；`backend/api/api-admin/03-students.md` 的 `PUT /admin/students/:id` 无 status 字段
- 建议: 若产品要求编辑页改状态，先扩契约与请求结构；当前「编辑不覆盖时间戳」是契约内唯一正确行为，已登记 known gap

### 2. [edge] 整批 `RowsAffected==0` 时清理提前结束
- 证据: `server/internal/service/album_retention.go:104`
- 建议: 若未来加入相册恢复端点，改用游标 `id > lastID` 继续，避免整批恢复后其余到期相册延后到下次调度

### 3. [test evidence] 失败传播/并发恢复仅代码审阅
- 证据: `album_retention.go:99-143`；命令错误路径已实测 exit 1 且无 DSN 泄漏
- 建议: DB 层具备错误注入后补一条 mid-transaction 失败用例

### 4. [docs] secret scan 文件数表述不一致
- 证据: `self-review.md:70` 写 6 个文件，`validators.md`/`worker-result.json` 写 9 个；重跑 9 个文件 0 findings
- 建议: 更新工件时改掉 6 文件表述

## 备注

- 3 个指标调用点文件在 brief Allowed list 之外，已由主会话跟进消息明确批准并登记，non-P1。
- `album_photo_save` / `album_complaint` 为客户端 `uni.reportAnalytics`，本切片不新增后端端点（brief 明确）。
- 未做 HTTP E2E：8080 未重启（按要求）；证据为 service 层真实 DB 测试 + 真实命令二进制。
