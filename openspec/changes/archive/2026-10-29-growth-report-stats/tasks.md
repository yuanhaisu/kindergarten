# Tasks

## 1. 契约（先改）

- [x] 1.1 `backend/api/api-parent/04-growth.md`：四格字段（`attendance`/`points_earned`/`tasks_completed`/`tasks_total`/成长记录）由「已声明未实现」改为真实返回；`growth_media_count` 更名 `growth_record_count`；补 `date` 生效规则与 `type` 允许值（weekly/monthly/term）；不新增 `stats` 等重复字段
- [x] 1.2 对照 design「契约字段溯源」逐项核对，确认无多余字段

## 2. 后端

- [x] 2.1 `parent_portal_service.go`：报告读取增加四格聚合（出勤率分母=周期内班级有考勤记录天数、迟到算出席；积分正向合计；任务条数口径；成长记录复用现有可见性谓词）
- [x] 2.2 `parent_portal.go` handler：读取 `date`，校验 `type`/`date`，非法返回参数错误
- [x] 2.3 保持 `{report, dimensions}` 兼容；无报告行时四格仍返回
- [x] 2.4 [P] 确认未改其它端点、无表结构变化、Casbin/router 不变

## 3. 家长端页面

- [x] 3.1 `parent/src/pages/growth/report.vue`：四格读既有字段（`attendance.rate`、`points_earned`、`tasks_completed/total`、`growth_record_count`）；空数据展示零值而非空态错误
- [x] 3.2 周期切换（周/月/学期）传参正确；`date` 按当前所选周期传递
- [x] 3.3 不改样式 token、不改路由；家长端 tabBar 跳转如需返回成长时光用 `switchTab`

## 4. 测试

- [x] 4.1 service 测试：无报告行返回四格、有报告行兼容、四格口径（迟到、负积分、任务条数、可见性）、孩子隔离、零值
- [x] 4.2 handler 测试：非法 type/date 返回参数错误
- [x] 4.3 在 `docs/tests/TEST_CATALOG.md` 登记 `TC-NEW-GRP-0001`–`0009`（若已有近似用例则改为引用）

## 5. 验证

- [x] 5.1 `cd server && go build ./...` exit 0；定向 `go test ./internal/service/ ./internal/handler/http/ -run 'Report'` exit 0
- [x] 5.2 真实 HTTP：无报告行孩子返回四格；有报告行孩子返回 report+四格；非法参数 400；两个孩子互不混入
- [x] 5.3 家长端 DevTools：真实点击切换周期，四格数字与接口一致；空数据零值；Console 0 error
- [x] 5.4 `node scripts/check-tokens.mjs parent` 不高于基线；`node scripts/check-secrets.mjs --files <manifest>` 0 findings
- [x] 5.5 逐条对照 `specs/growth-report/spec.md` 验收，未验证不得宣布完成
