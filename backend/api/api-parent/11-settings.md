# 家长端 API — 设置

> 基础路径：`/api/v1/parent` | 鉴权：`Authorization: Bearer <token>`（角色 `parent`）
>
> Response 均为 `data` 部分，省略外层 `{ "code": 0, "message": "success" }` 包装。

---

### GET `/parent/settings`

> 获取设置项

```json
// Response
{
  "notification": {
    "notice_push": true,
    "leave_push": true,
    "pickup_push": true,
    "growth_push": true,
    "task_push": true,
    "announcement_push": true
  },
  "privacy": {
    "show_phone_to_teacher": true,
    "allow_growth_share": true
  }
}
```

---

### PUT `/parent/settings`

> 更新设置

```json
// Request
{
  "notification": {
    "notice_push": true,
    "leave_push": true,
    "pickup_push": true,
    "growth_push": false,
    "task_push": true,
    "announcement_push": true
  },
  "privacy": {
    "show_phone_to_teacher": false,
    "allow_growth_share": true
  }
}

// Response
{
  "updated_at": "2026-03-19T10:00:00Z"
}
```
