# Tasks

## 1. 契约（先改）

- [x] 1.1 `api-teacher/04-notices.md`：POST/PUT 增可选 `status`（0/1/2，POST 默认 2、PUT 默认保持），写明草稿/定时/发布完整性与状态机
- [x] 1.2 附件：仅图片、≤9、单张 ≤20MB；详情 `attachments`、列表 `attachments_count` 落实
- [x] 1.3 `scheduled_at` 明确 RFC3339 带时区且必须未来；删除仅草稿、撤回仅已发布
- [x] 1.4 对照 design 字段溯源核对，无多余字段

## 2. 后端

- [x] 2.1 `teacher_portal_service.go`：创建支持 status（0/1/2）、完整性校验、定时未来校验；附件写入
- [x] 2.2 编辑：仅草稿/定时；支持发布草稿（status=2）并写 `published_at`；附件整体替换
- [x] 2.3 列表/详情：草稿仅创建者可见；详情返回 `attachments`；列表 `attachments_count` 真实计数
- [x] 2.4 删除/撤回状态校验（仅草稿可删、仅已发布可撤回）
- [x] 2.5 `parent_portal_service.go`：确认通知仅 `status=2`
- [x] 2.6 [P] 确认无表结构变化、无新路由、Casbin 不变

## 3. 老师端页面

- [x] 3.1 `notice/create.vue`：底部增加「存草稿」次按钮；提交 `status`（草稿 0/发布 2）
- [x] 3.2 定时时间改为带 `+08:00` 的 RFC3339；未选时间拦截；过去时间提示
- [x] 3.3 附件：`chooseImage` 后真实上传（`biz_type=notice`），最多 9 张、单张 20MB；编辑回填与删除
- [x] 3.4 `notice/index.vue`：草稿 tab 与「继续编辑」可用；草稿列表显示附件数
- [x] 3.5 `api/teacher-notices.js`：创建/更新透传 status/attachments/scheduled_at

## 4. 测试

- [x] 4.1 service 测试：存草稿（仅标题/空）、草稿仅创建者可见、定时过去时间拒绝、编辑已发布拒绝、发布草稿完整性、附件 9 张上限与落库/回填、删除非草稿拒绝、家长确认定时拒绝
- [x] 4.2 在 `docs/tests/TEST_CATALOG.md` 登记 `TC-NEW-NOT-0001`–`0012`

## 5. 验证

- [x] 5.1 `cd server && go build ./...` exit 0；定向 `go test ./internal/service/ -run 'Notice'` exit 0
- [x] 5.2 真实 HTTP：存草稿→列表可见→编辑→发布；定时过去 400；附件 3 张详情/列表一致；家长确认定时 400
- [x] 5.3 老师端 DevTools：真实点击存草稿、继续编辑、上传图片、定时开关；Console 0 error
- [x] 5.4 家长端 DevTools：草稿/定时通知不出现在列表（仅已发布）
- [x] 5.5 token ≤164、secret 0、无 uni watcher 残留
- [x] 5.6 逐条对照 `specs/notice-authoring/spec.md` 验收
