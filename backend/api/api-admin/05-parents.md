# 05. 家长管理

> 对应前端页面：`admin/pages/parents/index.vue`
>
> 基础路径：`/api/v1/admin`，需 `Authorization: Bearer <token>`，角色 `admin`。
>
> 状态：✅ 5 个接口已实现（2026-04-21）；#172 增补身份支持 3 个接口（2026-11-21）。

---

## GET `/admin/parents`

> 家长列表（对应前端 `getParents`）

```json
// Query: ?keyword=王&class_id=1&page=1&page_size=20

// Response
{
  "list": [
    {
      "id": 10,
      "nickname": "王小明爸爸",
      "avatar": "https://cdn.example.com/avatar/10.jpg",
      "phone": "138****8888",
      "gender": 1,
      "children": [
        {
          "student_id": 1,
          "student_name": "王小明",
          "class_name": "小一班",
          "relation": "father",
          "relation_label": "爸爸"
        }
      ],
      "status": 1,
      "status_label": "正常",
      "created_at": "2025-09-01T08:00:00Z"
    }
  ],
  "total": 520,
  "page": 1,
  "page_size": 20
}
```

---

## GET `/admin/parents/:id`

> 家长详情（对应前端 `getParent`）

```json
// Response
{
  "id": 10,
  "nickname": "王小明爸爸",
  "real_name": "王明",
  "avatar": "https://cdn.example.com/avatar/10.jpg",
  "phone": "13888888888",
  "gender": 1,
  "children": [
    {
      "student_id": 1,
      "student_name": "王小明",
      "student_avatar": "...",
      "class_id": 1,
      "class_name": "小一班",
      "relation": "father",
      "relation_label": "爸爸",
      "is_primary": true
    }
  ],
  "status": 1,
  "status_label": "正常",
  "last_login_at": "2026-03-19T07:30:00Z",
  "created_at": "2025-09-01T08:00:00Z"
}
```

---

## POST `/admin/parents`

> 添加家长（并绑定孩子；对应前端 `createParent`）

```json
// Request
{
  "real_name": "陈梅",
  "nickname": "赵小刚妈妈",
  "phone": "13700003333",
  "gender": 2,
  "student_bindings": [
    { "student_id": 301, "relation": "mother", "is_primary": true }
  ]
}

// Response
{
  "id": 521,
  "created_at": "2026-03-19T10:00:00Z"
}
```

---

## PUT `/admin/parents/:id`

> 编辑家长（对应前端 `updateParent`）

```json
// Request
{
  "nickname": "赵小刚妈妈",
  "student_bindings": [
    { "student_id": 301, "relation": "mother", "is_primary": true }
  ]
}

// Response
{
  "id": 521,
  "updated_at": "2026-03-19T10:30:00Z"
}
```

---

## PUT `/admin/parents/:id/status`

> 修改家长状态（禁用 / 恢复；对应前端 `updateParentStatus`）

```json
// Request
{
  "status": 2,                  // 1=正常 2=禁用
  "reason": "信息虚假"
}

// Response
{
  "id": 10,
  "status": 2,
  "status_label": "禁用",
  "updated_at": "2026-03-19T10:00:00Z"
}
```

> ⚠️ **边界**：这里的 `status` 是园所内的家长档案状态，**不参与登录认证**——改状态
> 不会吊销已签发的 access / refresh，也不会阻止该微信继续登录家长端。账号级会话失效
> 只由自助改密、管理员重置密码、管理员账号被复用、删除管理员等账号级动作触发
> （见 [`16-admins.md`](./16-admins.md)），与家长档案状态无关。

---

## GET `/admin/parents/:id/support`

> 家长账号支持视图（#172）。用于「家长绑错了微信 / 手机号录错了」的排障入口：
> 展示手机号、客户端身份（open_id 只回掩码）与本园成员关系。
>
> 只能看到**本园**的成员关系；目标账号不属于本园时与不存在同样返回 `404`。
> 权限码：`parent:manage`。

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
    }
  ]
}
```

---

## DELETE `/admin/parents/:id/identities/:identity_id`

> 解绑客户端身份（#172）。只**物理删除**这一条 `user_identities`（释放
> `(app_id, open_id)` 唯一键），账号、角色、孩子关联、积分与档案全部保留。
> 解绑后该微信的**续期**立即失效；已签发的访问凭证在剩余 TTL（默认 ≤15 分钟）内仍可用。
>
> 审计与解绑同事务：`operation_logs`，`module=parents`、`action=unbind_identity`、
> `target_type=user_identity`，detail 中的 openid 只留掩码。
> 权限码：`parent:manage`。

```json
// Response
{
  "user_id": 10,
  "unbound_identity_id": 51,
  "identities": []
}
```

错误：

| HTTP | 场景 |
|---|---|
| 404 | 账号不存在、不属于本园，或身份不存在 / 不属于该账号 |
| 422 | 路径参数非法 |

---

## PUT `/admin/parents/:id/phone`

> 更正预创建账号的手机号（#172）。顺序是硬性的：
>
> 1. 先解绑错误身份（`DELETE /admin/parents/:id/identities/:identity_id`）
> 2. 再改手机号
> 3. 由正确家长用新手机号重新绑定
>
> 账号上仍有客户端身份时拒绝（`409`）——否则录错号对应的微信会随着新号继续拥有访问权。
> 新号不得被其他活跃账号占用（`users.phone_active` 唯一索引，软删账号不占号）；
> 并发冲突由唯一索引最终裁决，同样返回 `409`。**不会静默删号或合并账号。**
> 手机号更新与审计同事务：`operation_logs`，`module=parents`、`action=correct_phone`、
> `target_type=user`。权限码：`parent:manage`。

```json
// Request
{
  "phone": "13900001111"
}

// Response（与支持视图同构，phone 已更新）
{
  "user_id": 10,
  "nickname": "王小明爸爸",
  "phone": "13900001111",
  "identities": [],
  "memberships": [
    {
      "kindergarten_id": 4,
      "kindergarten_name": "阳光幼儿园",
      "role": "parent",
      "class_id": 1,
      "class_name": "小一班"
    }
  ]
}
```

错误：

| HTTP | 场景 |
|---|---|
| 409 | 账号上仍有微信身份，需先解绑（`SUPPORT_IDENTITY_REMAINS`） |
| 409 | 新手机号已被其他活跃账号占用（`SUPPORT_PHONE_CONFLICT`） |
| 422 | 手机号格式不正确 |
| 404 | 账号不存在或不属于本园 |
