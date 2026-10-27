# teacher/23-notice-index · 补设计审查

> 2026-10-27 后补页；不属于原 50 页 subagent 审查批次。

- verdict：**PASS**（补设计，非原批次评分）
- 源码覆盖：源码原没有 `teacher/src/pages/notice/index.vue`，2026-10-27 已补；本页按 `backend/api/api-teacher/04-notices.md` 的列表契约设计，覆盖 `status` 0–3、`category`、`read_count` / `total_count` / `read_rate`、`attachments_count`、`class_names`、`need_confirm`。
- 页面覆盖：概要、状态/类型筛选、5 条示例（已发布 2 / 定时 1 / 草稿 1 / 已撤回 1）、筛选空态、整页空态、加载骨架、失败重试、详情弹层（已读/未读/已确认、未读名单、再次提醒、草稿/定时编辑入口）。
- 验证：`od-check --width 390` → `problems: []`；`od-shot` → 780×3160；DOM 实测筛选、详情、未读名单、toast、四态切换均正常，无 console error。
- 遗留：源码已补 `/pages/notice/index` 并注册；独立通知详情/完整已读名单页未画；源码列表已接 `status` / `category` 服务端参数，但分页仍按 20 条加载更多；草稿编辑（`notice/create` 预填 + `updateTeacherNotice`）未实现。
