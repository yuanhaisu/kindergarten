# 0022 — 平台访问凭证 15 分钟 + 7 天续期，账号级失效

- Status: Accepted
- Date: 2026-11-19

## Context

平台访问凭证在 `ProvidePlatformService` 中硬编码为 **24 小时**，注入的 `atTTL` 被 `_ = atTTL` 丢弃，契约 `01-auth.md` 也写着 `expires_in: 86400`；没有续期端点。后果是：

- 跨天工作时凭证会在操作中途过期；
- 停用账号、修改密码、重置密码都无法让已签发的凭证立即失效——只能等 24 小时自然过期；
- 如果直接加续期而不校验账号状态，停用/改密后的旧续期凭证仍能换出新凭证，失效语义落空。

园所侧的续期键是 `refresh:{jti}`。平台若复用同一前缀，平台续期凭证可以在园所 `/auth/refresh` 换出园所访问凭证，两个信任域互相污染（ADR-0014 要求隔离）。

## Decision

### 1. 短访问凭证 + 独立续期

- 平台访问凭证 TTL 改为读取 `jwt.access_ttl_min`（默认 **15 分钟**），删除硬编码 24 小时。**这是显式行为变更**：平台会话从 24 小时缩短到 15 分钟，必须靠续期维持。
- 登录与续期都签发续期凭证，存 Redis `refresh:platform:{jti}`，值为 `adminID:issuedAtUnix`，TTL 取 `jwt.refresh_ttl_day`（默认 7 天）。园所保持 `refresh:{jti}`，两类键不重叠。
- `POST /platform/auth/refresh` 只消费 `refresh:platform:*`，园所 `/auth/refresh` 只消费 `refresh:*`；各自不能消费对方的凭证。
- 每次续期轮换续期凭证：旧 jti 消费后立即失效，不能重放。
- 平台登出接收续期凭证，删除平台键并写入吊销名单；不接收续期凭证的旧客户端仍可吊销访问凭证。

### 2. 续期必须重新校验账号状态

- 账号 `status != enabled` 时拒绝续期；
- 续期凭证的签发时间早于账号级失效时间（epoch）时拒绝续期。

改密、停用、重置他人密码写 Redis epoch `platform:auth_epoch:{adminID}`；`PlatformAuth` 中间件与续期端点都检查 epoch。**epoch 写入失败时操作本身必须失败（503），不得返回成功**；Redis 不可用时 fail closed，与现有 token revoker 行为一致。

### 3. 登录限流

平台登录端点挂 10 次/分钟/IP 的 Redis 限流，Redis 故障时 fail closed。限流与审计 IP 都依赖 `[server] trusted_proxies` 配置：容器 + 宿主机 Nginx 的部署必须把网桥网关地址加入，否则 `ClientIP()` 得到网关 IP，限流退化为全站共享配额。

## Consequences

- 前端 `usePlatformHttp` 必须实现单飞（single-flight）自动续期：401 时复用同一个续期 Promise，成功后更新 access/refresh Cookie 并重放原请求。平台端是第一次落地这个客户端流程，没有现成先例。
- 续期成为强依赖：续期接口故障会导致频繁重新登录，上线前必须在 staging 实测整条链路。
- 旧续期凭证、停用账号的旧凭证、登出后的旧凭证都必须被拒绝，需要服务、中间件与 handler 三层测试覆盖。
- `[jwt] access_ttl_min` 现在是平台与园所共用的会话参数；部署说明必须写明 15 分钟是平台的行为变更，不是「维持现状」。
- 平台续期凭证存在非 HttpOnly Cookie（沿用现有架构），15 分钟 access TTL 限制泄露窗口。
- Redis 淘汰/清空会连带抹掉续期凭证与 `platform:auth_epoch:` / `tenant:auth_epoch:` 失效记录。若只淘汰了 epoch 而续期键仍在，已吊销的 access（剩余 TTL 内）与 refresh 会重新可用（refresh 仍受口令指纹、角色复核约束）；`FLUSH` 或换用空实例等同于全部账号的 epoch 归零。会话 Redis 必须按完整会话生命周期（refresh + access + 1 天）预留内存，禁用 `allkeys-lru` 一类逐出策略。

## Alternatives considered

- **保留 24 小时 access、只加续期**：安全收益低，且与「短 access + refresh」的会话模型不一致，被否。
- **只吊销当前 jti**：不满足「此前签发的凭证全部失效」，被否。
- **给 `platform_admins` 加 `token_version` 并在每请求查库**：每次请求多一次数据库查询；Redis epoch 更轻。
- **平台复用园所 `refresh:` 前缀**：两个信任域可互相兑换凭证，违反 ADR-0014，被否。
- **限流键改成 `username + IP`**：仍依赖正确 IP，不能替代可信代理配置，只能作为补充。
