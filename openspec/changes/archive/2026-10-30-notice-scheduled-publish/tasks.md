# Tasks

## 1. 后端

- [x] 1.1 `notice_repo.go`：新增「查询到期通知（status=1、scheduled_at<=now、未软删、limit 100）」与「条件发布（UPDATE ... WHERE id=? AND status=1 AND scheduled_at<=now）」
- [x] 1.2 新增 `notice_scheduler.go`：单轮处理函数（逐条条件更新、失败隔离、结构化日志）+ `Start/Stop`（ticker 默认 60s）
- [x] 1.3 应用启动处挂载 worker（参照 `RecognitionQueue.Start` 的启动方式）；Wire 如需要则同步
- [x] 1.4 [P] 确认无新路由/契约/表结构变化，Casbin 不变

## 2. 测试

- [x] 2.1 单轮处理：到点发布、未到点不动、已撤回/草稿/已发布跳过
- [x] 2.2 幂等：连续两轮只发布一次，`published_at` 不变
- [x] 2.3 并发：两个 goroutine 同时处理同一条，只成功一次（用条件更新断言）
- [x] 2.4 失败隔离：注入一条失败，其余仍成功
- [x] 2.5 在 `docs/tests/TEST_CATALOG.md` 登记 `TC-NEW-SCH-0001`–`0006`

## 3. 验证

- [x] 3.1 `cd server && go build ./...` exit 0；定向 `go test ./internal/service/ -run 'NoticeSched'` exit 0
- [x] 3.2 真实短间隔演练：构造 `scheduled_at=now+5s` 的通知，确认到点后状态变 2、`published_at` 写入；重复扫描不再变更
- [x] 3.3 老师端通知列表真实刷新可见状态变化；家长端只看到已发布
- [x] 3.4 secret 0；无表结构变化；测试数据 0 残留
- [x] 3.5 逐条对照 `specs/notice-scheduling/spec.md` 验收
