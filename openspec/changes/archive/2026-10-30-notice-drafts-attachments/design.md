# Design

## Context

见 `proposal.md - Why`。关键现状：

- `domain.Notice` 已有 `status`（0/1/2/3）与 `ScheduledAt`；`NoticeAttachment` 表已存在并由 admin 写入。
- 老师端 `CreateNotice` 只 insert 通知与目标集，忽略 status/attachments；`UpdateNotice` 只改标题/正文/类型/确认开关与目标集。
- 前端定时时间拼 `YYYY-MM-DD HH:mm`，后端只认 RFC3339 → 定时发布必失败。
- 老师列表/详情不限制草稿可见性；家长确认接口不校验 status。
- 全仓无微信订阅消息推送，发布只改状态与时间。

## Goals / Non-Goals

**Goals:**

- 让契约声明的草稿/定时/附件在老师端真正可用且可验证。
- 收紧草稿可见性与家长确认状态。
- 时间格式统一为带时区 RFC3339。

**Non-Goals:**

- 不做定时到点执行（B03b）。
- 不做家长端附件展示、管理端改造、推送。
- 不新增表/列。

## Decisions

### D1 `status` 作为 POST/PUT 的可选字段

- POST 省略时默认 2（立即发布），保持现有调用兼容；0=草稿，1=定时（要求 `scheduled_at`），2=发布。
- PUT 省略时保持当前状态；显式传 2 表示「发布草稿/立即发布」。
- 备选「新增 /publish 端点」被否决：多一个端点、状态机分散；`status` 字段在契约列表中已存在，扩展入参最省。

### D2 草稿完整性

- 存草稿：标题或正文至少一项非空，`class_ids` 可为空。
- 定时/发布：标题、正文、`class_ids` 均非空；定时另需未来时间。
- 编辑草稿时允许先清空班级，保存仍为草稿。

### D3 可见性

- 列表：`status` 参数不传时，草稿仅返回 `creator_id = 当前老师` 的记录；传 0 时同样只返回自己的草稿。
- 详情：草稿只有创建者可读；其他老师得到「不存在/无权限」。
- 家长确认：仅 `status=2` 可确认，其余返回业务错误。

### D4 附件

- 仅 `file_type=image`；数量 ≤9；单张 ≤20MB（上传服务已限制）。
- 存储：`notice_attachments`（file_url/type/name/size/sort_num），创建与编辑时整体替换。
- 详情返回 `attachments`；列表返回 `attachments_count`（真实计数）。
- 家长端接口暂不返回附件，后续切片再做。

### D5 时间格式

- 请求统一 RFC3339 且必须带时区（如 `2026-11-01T09:00:00+08:00`）。
- 后端按 RFC3339 解析并转 `time.Local`；过去时间拒绝（允许 1 分钟时钟误差）。
- 前端把日期+时间拼成带 `+08:00` 的 RFC3339，不再提交裸 `YYYY-MM-DD HH:mm`。

### D6 删除/撤回

- 删除仅允许草稿；撤回仅允许已发布（置 3）。状态不符返回业务错误。
- 若路由或实现已存在，仅补状态校验，不新增端点。

## 跨端触点清单

| 层 | 是否改 | 说明 |
| --- | --- | --- |
| Go domain | 不改 | `Notice.status/scheduled_at` 与 `NoticeAttachment` 已存在 |
| repository | 会改 | `notice_repo.go`：创建/更新附件、列表真实计数、详情附件、草稿可见性过滤 |
| service | 会改 | `teacher_portal_service.go` 创建/编辑/删除/撤回状态规则与附件；`parent_portal_service.go` 确认校验 status |
| handler | 会改 | `teacher_portal.go` 绑定 `status/attachments`，RFC3339 解析与错误映射 |
| Casbin policy | 不改 | 现有 teacher 通知路由已覆盖 |
| router.go | 不改 | 端点已存在 |
| backend/api 契约 | 会改（契约变更） | `api-teacher/04-notices.md`：POST/PUT 增可选 `status`；草稿完整性；附件仅图片≤9；时间 RFC3339 |
| 端页面 | teacher 会改 | `notice/create.vue`（存草稿、附件上传、定时格式）、`notice/index.vue`（草稿 tab/继续编辑）、`api/teacher-notices.js`；parent/admin 不改 |
| design token | 不改 | 复用既有样式 |
| 迁移 | 不改 | 表与列已存在；`notice_attachments` 仅新增写入方 |

### 契约字段溯源

| 字段 | 出处 |
| --- | --- |
| POST/PUT `status` | 设计 `teacher/23-notice-index` 的草稿/定时 tab 与「继续编辑」；设计 03 无存草稿按钮，属**有意偏差**，在 review 记录 |
| `attachments` | 设计 `teacher/03-notice-create` 附件区「0/9」 |
| `scheduled_at` | 设计 03「定时发送」开关与时间滚轮 |
| `attachments_count` | 同页附件计数与列表附件数 |

## Risks / Trade-offs

- [草稿可见性收紧影响既有数据] → 存量 status=0 若为他人创建，将对非创建者隐藏；符合「草稿仅自己可见」，在发布说明标注。
- [PUT 传 2 发布与既有编辑语义混用] → 明确状态机并在测试覆盖；发布失败保持草稿。
- [附件整体替换可能误删] → 编辑时以提交列表为准，前端回填现有附件；测试覆盖「删掉一张后保存」。
- [时区] → 只接受带偏移的 RFC3339，避免服务器本地时区歧义。

## Migration Plan

无 schema 变更。上线后存量草稿仍可读；如回滚，仅回退代码与契约。
