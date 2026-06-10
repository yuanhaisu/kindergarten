# 05. 家长管理

> 对应前端页面：`admin/pages/parents/index.vue`
>
> 基础路径：`/api/v1/admin`，需 `Authorization: Bearer <token>`，角色 `admin`。
>
> 状态：✅ 5 个接口已实现（2026-04-21）。

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
