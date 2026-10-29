# Design

## Context

见 `proposal.md - Why`。现状约束：

- `GET /parent/growth/report` 已存在，返回 `{report, dimensions}`；`report` 取自 `growth_reports` 最新一条 `status=1` 行。
- `growth_reports` **没有任何写入方**，四格字段也不在 domain 中，因此该接口当前实际返回空 map。
- 四格所需数据分散在 `attendances`、`point_transactions`、`parent_tasks`/`task_check_ins`、`growth_daily_logs`/`growth_media`。
- 可见性已有唯一实现（growth log 可见性谓词），必须复用，不能另写一套。

## Goals / Non-Goals

**Goals:**

- 无报告行时也能给出四格统计，口径可解释、可测试。
- 契约先行，字段可溯源到设计稿四格。
- 家长端报告页最小改动即可展示。

**Non-Goals:**

- 不生成/不写入报告正文，不改能力评估 7 维与文案。
- 不引入缓存、不建汇总表、不加定时任务。
- 不改老师端/管理端。

## Decisions

### D1 实时聚合，而非补生成链路

四格在请求时按周期聚合，报告行存在与否都计算。理由：当前无写入方，补生成链路会引入触发时机、快照一致性、补数据等新问题；设计页只需要四格 + 已有正文。备选「快照进 `growth_reports`」被否决——没有生产者时快照只会过期，且需要迁移。

### D2 四格口径

| 指标 | 分子 | 分母 | 理由 |
| --- | --- | --- | --- |
| 出勤率 | 该孩子出勤天数（含迟到） | 周期内班级实际产生考勤记录的天数 | 数据可得；请假未记考勤不进分母，避免把请假算成缺勤 |
| 获得积分 | 正向积分流水合计 | — | 与「获得」语义一致，扣减不计 |
| 完成任务 | 已打卡任务条数 | 周期内发布任务条数 | 与 ParentTask 模型一致；按频率展开留待后续 |
| 成长记录 | 可见 growth log 条数 | — | 与页面标签一致；字段名 `growth_record_count` 表明是记录数而非媒体数 |

备选：出勤率以校历工作日为分母（日历数据不完整，会算不准）；积分净额（与「获得」不符）；任务按频率展开（复杂且需产品再确认）。均未采用，作为后续可演进点。

### D3 响应形状：实现既有字段，不新增 `stats`

契约 `04-growth.md` 已声明 `attendance{present_days,leave_days,total_days,rate}`、`points_earned`、
`tasks_completed`/`tasks_total`、`growth_media_count`，缺的是实现。因此：

- 直接填充这些既有字段，**不新增 `stats` 对象**（避免同一语义两份字段）。
- `growth_media_count` 更名 `growth_record_count`：该字段从未实现且语义为「可见记录条数」，
  名称与页面标签「成长记录」一致；家长端同步改读新字段。
- `date` 参数按契约生效：给定日期落在哪个周期就取哪个周期；`type` 只接受 weekly/monthly/term，
  非法返回参数错误。`term` 周期取 `academic_years` 中覆盖该日期的学期区间；
  未配置学期时返回参数错误（不臆造日期范围）。

### D4 聚合查询与可见性

- 出勤：按学生 + 周期 + 班级日期集合聚合；分母用 `COUNT(DISTINCT date)`。
- 积分：`SUM(points) WHERE points > 0 AND 周期`。
- 任务：分母为周期内任务条数，分子为 `task_check_ins` 命中条数。
- 成长记录：复用现有可见性谓词统计 log 数。
- 单次请求最多 4 组聚合查询；不加缓存，先保证正确性。

## 跨端触点清单

| 层 | 是否改 | 说明 |
| --- | --- | --- |
| Go domain | 不改 | 四格为计算值，无新表/字段；`GrowthReport` 结构不变 |
| repository | 不改 | 聚合走 service 现有 `s.db` 模式，与周边 report 读取一致 |
| service | 会改 | `parent_portal_service.go` 报告读取增加四格聚合与可见性复用 |
| handler | 会改 | `parent_portal.go` report handler 读取 `date`、校验 `type`/`date` |
| Casbin policy | 不改 | `/parent/growth/report` 已在现有策略内 |
| router.go | 不改 | 路由已存在 |
| backend/api 契约 | 会改（契约变更） | `api-parent/04-growth.md`：四格字段变为真实返回；`growth_media_count` → `growth_record_count`；`date` 生效；`term` 校验 |
| 端页面 | parent 会改 | `parent/src/pages/growth/report.vue`：读 `growth_record_count`，四格直接取既有字段；teacher/admin 不改 |
| design token | 不改 | 无新增样式 |
| 迁移 | 不改 | 无表结构变化 |

### 契约字段溯源

| 字段 | 出处 |
| --- | --- |
| `attendance.rate`（含 `present_days`/`leave_days`/`total_days`） | 设计 `parent/12-growth-report` 四格「出勤率」；契约已声明，本次实现 |
| `points_earned` | 同页「获得积分」；契约已声明 |
| `tasks_completed` / `tasks_total` | 同页「完成任务 8/12」；契约已声明 |
| `growth_record_count` | 同页「成长记录 15」；由未实现的 `growth_media_count` 更名，语义为可见记录条数 |
| `date` | 契约已声明但未实现，本次补齐 |

## Risks / Trade-offs

- [聚合口径与未来报告快照不一致] → 统计以实时数据为准；未来生成链路落地时，若要用快照需另开变更并明确覆盖关系。
- [聚合查询性能] → 单学生 + 单周期，数据量有限；若慢查询出现，再评估索引/缓存，不在本片预先优化。
- [出勤率分母受班级考勤覆盖率影响] → 明确写入口径并在页面/文档说明，避免与「应到天数」混淆。
- [任务分母为任务条数，daily 任务不展开] → 已知差异，后续按频率展开需产品确认。

## Migration Plan

无 schema 变更，无迁移。发布后无需回滚数据；如需回滚，仅回退代码与契约。
