# 01. 平台登录

> 对应前端页面：`admin/pages/platform/login.vue`
>
> 基础路径：`/api/v1/platform`

> ⚠️ **行为变更**：平台访问凭证此前硬编码为 **24 小时**且没有续期。自本次起改为
> `jwt.access_ttl_min`（默认 **15 分钟**），客户端必须在访问凭证过期前后用
> `POST /platform/auth/refresh` 续期。续期凭证默认 7 天（`jwt.refresh_ttl_day`）。
>
> 平台续期与园所续期分键存储：平台为 `refresh:platform:{jti}`，园所为 `refresh:{jti}`。
> 平台续期凭证不能在园所 `/auth/refresh` 兑换，园所续期凭证也不能在平台续期端点兑换。

---

## POST `/platform/auth/login`

> 平台方账号登录（与园所后台的 `/auth/login/password` 是两套独立账号）

```json
// Request
{
  "username": "ops",
  "password": "********"
}

// Response
{
  "token": "eyJhbGciOi...",
  "refresh_token": "0f8c4d1e-...",
  "expires_in": 900,
  "admin": {
    "id": 1,
    "username": "ops",
    "name": "运营"
  }
}
```

- `expires_in` 是**访问凭证**的有效期（秒），取自 `jwt.access_ttl_min`，默认 `900`。
- `refresh_token` 是续期凭证（jti 字符串），用于 `POST /platform/auth/refresh`；
  有效期由 `jwt.refresh_ttl_day` 决定（默认 7 天）。
- 凭证 `scope=platform`，不含 `kindergarten_id`。
- 用户名或密码错误返回 `401`，不区分具体原因。
- 同一 IP 连续登录尝试限 **10 次/分钟**：超限返回 `429`；限流存储不可用时返回
  `503`（失败关闭，不会在 Redis 故障期间放开暴力破解）。

---

## POST `/platform/auth/refresh`

> 用续期凭证换取新的访问凭证与轮换后的续期凭证。此端点不需要访问凭证
> （访问凭证正是已经过期的那个）。

```json
// Request
{
  "refresh_token": "0f8c4d1e-..."
}

// Response（与登录响应同构）
{
  "token": "eyJhbGciOi...",
  "refresh_token": "7a2b9c3f-...",
  "expires_in": 900,
  "admin": {
    "id": 1,
    "username": "ops",
    "name": "运营"
  }
}
```

- 每次续期都轮换续期凭证：响应中的 `refresh_token` 与请求中的不同，
  请求里的旧 `refresh_token` 立即失效，不能重放。
- 以下情况返回 `401`，客户端应清态回到登录页：
  - 续期凭证不存在、已消费或已吊销（含登出后）；
  - 账号已被停用（`status=disabled`）；
  - 续期凭证的签发时间早于该账号最近一次改密/停用/重置密码的生效时间。
- 缺少 `refresh_token` 返回 `422`；会话存储不可用返回 `503`。

---

## POST `/platform/auth/logout`

> 注销平台会话。需要当前访问凭证（`Authorization: Bearer <token>`）。

```json
// Request（refresh_token 可选；带上则一并吊销续期凭证）
{
  "refresh_token": "0f8c4d1e-..."
}

// Response
{}
```

- 服务端吊销当前访问凭证。
- 请求带 `refresh_token` 时，删除 `refresh:platform:{jti}` 并把该 jti 写入吊销名单；
  此后用它续期会返回 `401`。不带时只吊销访问凭证。
- 吊销存储写失败返回 `503`，不会返回 200 让客户端误以为会话已结束。
