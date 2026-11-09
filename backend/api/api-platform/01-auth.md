# 01. 平台登录

> 对应前端页面：`admin/pages/platform/login.vue`
>
> 基础路径：`/api/v1/platform`

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
  "expires_in": 86400,
  "admin": {
    "id": 1,
    "username": "ops",
    "name": "运营"
  }
}
```

> 凭证 `scope=platform`，不含 `kindergarten_id`。
> 用户名或密码错误返回 `401`，不区分具体原因。

---

## POST `/platform/auth/logout`

> 注销平台会话

```json
// Request
{}

// Response
{}
```
