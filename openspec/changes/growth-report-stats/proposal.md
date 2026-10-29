# Proposal

## Why

家长端「成长报告」页（设计 `parent/12-growth-report`）的四格统计——出勤率、获得积分、完成任务、成长记录——在界面上已按设计落地，但后端 `growth_reports` 表**没有任何写入方**，四格字段也不存在：报告页目前永远落在空态。家长看不到任何周期统计，设计稿的四格与能力评估全部落空。

## What Changes

- `GET /parent/growth/report` 在**没有报告行**时也返回契约中已声明的四项统计（`attendance`、`points_earned`、`tasks_completed`/`tasks_total`、成长记录数），按 `student_id` + `type` 周期**实时聚合**；有报告行时正文/维度仍读报告行。
- 成长记录字段由契约中的 `growth_media_count` 更名为 `growth_record_count`（该字段从未被实现，且语义为「可见记录条数」而非媒体数），家长端同步读取新字段。
- 四格口径（用户已确认）：
  - 出勤率：分母 = 周期内班级实际产生考勤记录的天数；迟到算出席；请假未记考勤不进分母。
  - 获得积分：周期内正向积分流水合计。
  - 完成任务：分子 = 已打卡，分母 = 周期内发布的任务条数。
  - 成长记录：该孩子可见的 growth log 条数（字段命名 `growth_record_count`）。
- 实现契约中已声明但未实现的 `date` 参数；`term` 增加取值校验。
- **本次将修改 `backend/api/api-parent/04-growth.md`，属于契约变更。**
- 明确不做（out of scope）：
  - 不新增报告生成链路（老师/管理端生成、AI 生成、定时任务），不写入 `growth_reports`。
  - 不改能力评估 7 维、亮点、关注、老师评语的数据来源（继续读报告行，无行则为空）。
  - 不改老师端/管理端。
  - 不改表结构，无迁移。

## Capabilities

### New Capabilities

- `growth-report`: 家长成长报告的周期统计与读取行为（四格口径、空报告回退、参数校验）。

### Modified Capabilities

（无）

## Impact

- 契约：`backend/api/api-parent/04-growth.md`（四格字段从「已声明未实现」变为真实返回；`growth_media_count` 更名 `growth_record_count`；`date` 生效、`term` 校验）。
- 后端：`server/internal/service/parent_portal_service.go`（report 聚合）、`server/internal/handler/http/parent_portal.go`（参数）、定向测试。
- 家长端：`parent/src/pages/growth/report.vue` 四格改读既有字段（含 `growth_record_count`），无路由/样式改动。
- 其它端：teacher / admin 不受影响。
- 数据：只读现有 `attendances`、`point_transactions`、`parent_tasks`/`task_check_ins`、`growth_daily_logs`/`growth_media`，无新表。
