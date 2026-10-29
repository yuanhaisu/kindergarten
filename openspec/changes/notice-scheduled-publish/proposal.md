# Proposal

## Why

通知契约与界面都支持「定时发送」（status=1 + `scheduled_at`），但全仓没有任何执行者：到点后通知永远停在 status=1，家长永远收不到。B03a 让定时通知可被正确保存后，这个缺口会立刻变成可见缺陷。

## What Changes

- 新增后台定时发布 worker：周期性扫描 `status=1` 且 `scheduled_at <= now` 的通知，将其发布（`status=2`、写 `published_at`）。
- 幂等与并发安全：用条件更新（`WHERE status=1 AND scheduled_at<=now`）保证一条通知只被发布一次，多实例/重复扫描不会重复处理。
- 失败隔离与重试：单条失败只记录日志，不影响其它通知；下一轮继续尝试。
- 每轮限制批量条数，避免长事务。
- 明确不做（out of scope）：
  - 不做微信订阅消息/模板消息推送（全仓无此能力，发布只改状态与时间）。
  - 不改通知接口与契约，不改前端页面，不改管理端。
  - 不引入新的调度中间件；沿用现有后台 ticker 的启动方式。

## Capabilities

### New Capabilities

- `notice-scheduling`: 定时通知到点自动发布的行为与幂等、失败语义。

### Modified Capabilities

（无）

## Impact

- 后端：新增定时发布 service/worker，并在应用启动处挂载 ticker（参照现有后台队列的启动方式）。
- 契约/前端/管理端：不改。
- 数据：只读写既有 `notices` 表，无新表/新列/迁移。
