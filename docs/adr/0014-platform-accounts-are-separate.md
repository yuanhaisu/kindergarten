# 0014 — 平台方使用独立账号体系

- Status: Accepted
- Date: 2026-11-06

## Context

现有 JWT 必带 `kindergarten_id`，`middleware/context.go`、园所 scoped repository 与 Casbin policy 都以它为前提；后台账号本身就是园所自己的管理员。平台方需要跨园查看与操作，属于另一个信任域。

## Decision

平台方使用独立的 `platform_admins` 表与独立登录入口，JWT 携带 `scope=platform` 且不含 `kindergarten_id`；接口走 `/api/v1/platform/*`，前端与园所后台共用同一个 Nuxt 应用，但使用独立路由树与独立登录态。

## Consequences

- 存在两个登录入口，token 不能互换。
- 平台接口不能复用按 `kindergarten_id` 自动过滤的仓储，跨租户查询必须是显式行为。
- 首个平台账号通过一次性命令或环境变量引导，不走 `/system/init`。

## Alternatives considered

- **users 表加 `role='platform_admin'` + 空 `kindergarten_id`**：每个下游中间件都要长出 null 分支，园所超管与平台超管的边界只靠 if 维持，是最容易出越权的写法。
- **给平台方一个假园所**：会把真实园所的数据与平台运营数据混在一起，且任何按园所聚合的统计都会失真。
