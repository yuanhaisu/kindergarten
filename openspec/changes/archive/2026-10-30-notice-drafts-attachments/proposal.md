# Proposal

## Why

契约 `api-teacher/04-notices.md` 已声明通知的 `status`（0 草稿/1 定时/2 已发布/3 已撤回）、`scheduled_at`、`attachments` 与「仅草稿/定时可编辑、仅草稿可删除」，但实现只落标题/正文/目标班级：附件从不入库，创建时丢弃 status，编辑不落定时与附件，列表附件数恒 0。更直接的两个缺陷：前端提交 `YYYY-MM-DD HH:mm` 而后端只认 RFC3339，定时发布实际必然失败；草稿对全体老师可见，家长确认接口也不校验状态。

## What Changes

- `POST /teacher/notices` 增加可选 `status`：0 草稿 / 1 定时 / 2 立即发布（默认 2，保持兼容）；`scheduled_at` 必须为带时区的 RFC3339 且晚于当前时间。
- 草稿允许标题/正文/班级不完整暂存；发布（status=2）要求标题、正文、目标班级完整。
- `PUT` 支持草稿/定时编辑与「发布草稿」：仅草稿/定时状态可编辑，发布时写 `published_at`。
- 附件落库：仅图片、最多 9 张、单张 ≤20MB；创建/编辑时替换；详情返回 `attachments`，列表返回真实 `attachments_count`。
- 可见性：草稿仅创建者可见（列表与详情）；家长端只能确认 `status=2` 的通知。
- 前端：创建页增加「存草稿」次按钮，提交带 `+08:00` 的 RFC3339；列表草稿 tab 与「继续编辑」可用；编辑草稿可增删附件与定时。
- **本次将修改 `backend/api/api-teacher/04-notices.md`，属于契约变更。**
- 明确不做（out of scope）：
  - 不做定时到点执行（B03b worker）。
  - 家长端不展示附件；管理端不改；不新增微信推送（全仓无推送能力）。
  - 不新增表、不改列结构。

## Capabilities

### New Capabilities

- `notice-authoring`: 老师端通知草稿、定时提交、附件与可见性、发布规则的端到端行为。

### Modified Capabilities

（无）

## Impact

- 契约：`backend/api/api-teacher/04-notices.md`（新增可选 `status`、草稿完整性规则、附件限制、列表/详情字段落实）。
- 后端：`server/internal/service/teacher_portal_service.go`、`repository/notice_repo.go`、`handler/http/teacher_portal.go`；
  `server/internal/service/parent_portal_service.go`（仅确认接口状态校验）。
- 老师端：`teacher/src/pages/notice/create.vue`、`notice/index.vue`、`api/teacher-notices.js`。
- 家长端：无页面改动（仅服务端校验）。
- 数据：复用现有 `notices.status/scheduled_at` 与 `notice_attachments`，无迁移。
