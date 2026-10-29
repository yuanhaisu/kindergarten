# Design

## Context

见 `proposal.md - Why`。约束：

- `notices` 已有 `status`（1=定时待发）与 `scheduled_at`，B03a 负责把它们写对。
- 现有后台 worker 先例是 `RecognitionQueue`：构造时 `Start(n)`，内部 `time.Ticker` 循环；可参照其启动方式。
- 全仓无微信推送能力，发布动作只改状态与时间；目标集（notice_targets）在创建时已写入。

## Goals / Non-Goals

**Goals:**

- 到点通知自动发布，幂等、并发安全、失败隔离。
- 不引入新依赖、不新增表。

**Non-Goals:**

- 不做推送、不做失败重试队列、不做管理端可视化。
- 不调整扫描间隔的自适应策略。

## Decisions

### D1 轮询扫描，而非延迟队列

用固定间隔（默认 60 秒）扫描 `status=1 AND scheduled_at <= now()`，每轮最多 100 条。理由：数据量小、无需新中间件；延迟队列会引入 Redis 依赖与消息丢失处理。备选「按最近 scheduled_at 精确 sleep」被否决：多实例与增删改会让 sleep 失效。

### D2 条件更新保证幂等与多实例安全

发布 SQL：`UPDATE notices SET status=2, published_at=now WHERE id=? AND status=1 AND scheduled_at<=now AND deleted_at IS NULL`。
`RowsAffected==1` 才视为本次发布成功；0 表示已被其它 worker 处理或状态已变，直接跳过。这样重复扫描与并发扫描都只发布一次。

### D3 失败隔离

逐条处理，单条错误只记录结构化日志并继续；失败的条目保持 status=1，下一轮自然重试。不做单独重试计数（避免复杂性），但日志带 notice_id 与错误。

### D4 时间与边界

`now` 用 `time.Now()`（进程时区已对齐 Asia/Shanghai）；边界含等号（`scheduled_at <= now`）。软删通知不处理。

### D5 启动方式

参照 `RecognitionQueue.Start`：在 service 构造或应用初始化时启动一个 goroutine；提供 `Start/Stop`，测试可直接调用单轮处理函数，不依赖真实 ticker。

## 跨端触点清单

| 层 | 是否改 | 说明 |
| --- | --- | --- |
| Go domain | 不改 | `Notice.status/scheduled_at` 已存在 |
| repository | 会改 | 新增「查询到期通知」与「条件发布」两条语句（可放 notice_repo） |
| service | 会改 | 新增 `NoticeScheduler`（单轮处理 + 启动/停止） |
| handler | 不改 | 无新端点 |
| Casbin policy | 不改 | 后台任务不走 HTTP |
| router.go | 不改 | 无新路由 |
| backend/api 契约 | 不改 | 内部行为，无接口变化 |
| 端页面 | 不改 | 老师/家长/管理端均无改动 |
| design token | 不改 | 无 UI |
| 迁移 | 不改 | 无新表/列 |

### 契约字段溯源

本变更不改契约，无新增字段。

## Risks / Trade-offs

- [扫描延迟最多一个间隔] → 默认 60 秒，可接受；如需更准可调间隔。
- [多实例重复处理] → 条件更新兜底，只有一行能成功。
- [批量过大导致长事务] → 每轮上限 100 条，单条独立更新。
- [无推送] → 发布只改状态；家长刷新可见，符合当前系统能力。

## Migration Plan

无 schema 变更。上线即生效；回滚只需停止 worker（回退代码）。
