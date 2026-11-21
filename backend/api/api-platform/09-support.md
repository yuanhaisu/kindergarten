# 09. 账号支持（解绑与手机号更正）

> 平台支持排障入口（#172）：查看目标用户的客户端身份与**全部园所**成员关系，
> 解绑绑错的微信、更正录入错误的手机号。
>
> 基础路径：`/api/v1/platform`。凭证必须带 `scope=platform`；
> 园所管理端 JWT 调用一律 `403`，平台凭证同样不能调用 `/admin/*`。
>
> 与园所管理端的同名能力（`GET|PUT /admin/parents/:id/*`）共用 service，
> 但**审计落不同表**：平台侧写 `platform_operation_logs`（`module=support`）。

---

## 通用约定

- 解绑只**物理删除**一条 `user_identities`（释放 `(app_id, open_id)` 唯一键），
  不删账号、角色、孩子关联、积分或档案。
- 更正手机号顺序硬性：先解绑账号上的错误身份，再改号，由正确家长重新绑定。
  账号仍有身份时直接拒绝；不静默删号、不合并账号。
- 新号不得被其他活跃账号占用（`users.phone_active` 唯一索引，软删账号不占号）。
- 解绑/改号与审计同事务；审计 detail 中的 openid、手机号由 `pkg/redact` 掩码。
- 解绑后该微信的**续期**立即失效；已签发的访问凭证在剩余 TTL（默认 ≤15 分钟）内仍可用。

---

## GET `/platform/users/:id/support`

> 用户支持视图：手机号、客户端身份（open_id 只回掩码）、全部园所成员关系。
> 目标用户不存在返回 `404`。

```json
// Response
{
  "user_id": 10,
  "nickname": "王小明爸爸",
  "phone": "13888888888",
  "identities": [
    {
      "id": 51,
      "app_id": "wxdf989efff8796c53",
      "open_id_masked": "oAbc****wxyz",
      "created_at": "2026-09-01T08:00:00Z"
    }
  ],
  "memberships": [
    {
      "kindergarten_id": 4,
      "kindergarten_name": "阳光幼儿园",
      "role": "parent",
      "class_id": 1,
      "class_name": "小一班",
      "title": "",
      "title_label": ""
    },
    {
      "kindergarten_id": 7,
      "kindergarten_name": "星光幼儿园",
      "role": "parent"
    }
  ]
}
```

---

## DELETE `/platform/users/:id/identities/:identity_id`

> 解绑客户端身份。身份不存在或不属于该用户时返回 `404`。
>
> 审计：`platform_operation_logs`，`operator=<平台账号>`、`module=support`、
> `action=unbind_identity`、`target_type=user_identity`。

```json
// Response
{
  "user_id": 10,
  "unbound_identity_id": 51,
  "identities": []
}
```

---

## PUT `/platform/users/:id/phone`

> 更正手机号。请求体 `phone` 必填，`^1[3-9]\d{9}$`。
>
> 审计：`platform_operation_logs`，`action=correct_phone`、`target_type=user`。

```json
// Request
{
  "phone": "13900001111"
}

// Response（与支持视图同构）
{
  "user_id": 10,
  "nickname": "王小明爸爸",
  "phone": "13900001111",
  "identities": [],
  "memberships": []
}
```

错误：

| HTTP | 场景 |
|---|---|
| 409 | 账号上仍有微信身份，需先解绑（`SUPPORT_IDENTITY_REMAINS`） |
| 409 | 新手机号已被其他活跃账号占用（`SUPPORT_PHONE_CONFLICT`） |
| 422 | 手机号格式不正确 |
| 404 | 用户不存在 |
