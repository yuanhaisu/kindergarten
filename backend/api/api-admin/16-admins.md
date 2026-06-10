# 16. 管理员

> 对应前端页面：`admin/pages/admins/index.vue`
>
> 基础路径：`/api/v1/admin`，需 `Authorization: Bearer <token>`，角色 `admin`。
>
> 状态：✅ 5 个接口已实现（后端）。

> 💡 角色与权限配置见 [`17-roles.md`](./17-roles.md)。

---

## GET `/admin/admins`

> 管理员列表（对应前端 `getAdmins`）

```json
// Query: ?keyword=&page=1&page_size=20

// Response
{
  "list": [
    {
      "id": 100,
      "nickname": "张园长",
      "avatar": "https://cdn.example.com/avatar/100.jpg",
      "phone": "138****0000",
      "admin_role": {
        "id": 1,
        "name": "系统管理员",
        "permissions_count": 45
      },
      "status": 1,
      "status_label": "正常",
      "last_login_at": "2026-03-19T08:00:00Z",
      "created_at": "2020-01-01T00:00:00Z"
    }
  ],
  "total": 5
}
```

---

## POST `/admin/admins`

> 添加管理员（对应前端 `createAdmin`）

```json
// Request
{
  "phone": "13700004444",
  "nickname": "副园长",
  "role_id": 2,
  "password": "Abc123456"       // 初始密码
}

// Response
{
  "id": 105,
  "created_at": "2026-03-19T10:00:00Z"
}
```

---

## PUT `/admin/admins/:id`

> 编辑管理员（对应前端 `updateAdmin`）

```json
// Request
{
  "nickname": "副园长",
  "role_id": 2
}

// Response
{
  "id": 105,
  "updated_at": "2026-03-19T10:30:00Z"
}
```

---

## DELETE `/admin/admins/:id`

> 删除管理员（对应前端 `deleteAdmin`）

```json
// Response
{}
```

---

## PUT `/admin/admins/:id/reset-password`

> 重置管理员密码（对应前端 `resetAdminPassword`）

```json
// Request
{
  "new_password": "NewPass123"
}

// Response
{
  "message": "密码已重置"
}
```
