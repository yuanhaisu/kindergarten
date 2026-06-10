# 17. 角色权限

> 对应前端页面：`admin/pages/roles/index.vue`
>
> 基础路径：`/api/v1/admin`，需 `Authorization: Bearer <token>`，角色 `admin`。
>
> 状态：✅ 后端已实现（2026-04-21）。

> 💡 管理员账号管理见 [`16-admins.md`](./16-admins.md)。

---

## GET `/admin/roles`

> 角色列表（对应前端 `getRoles`）

```json
// Response
{
  "list": [
    {
      "id": 1,
      "name": "系统管理员",
      "description": "拥有所有权限",
      "is_system": true,
      "admin_count": 1,
      "permissions": ["*"],
      "created_at": "2020-01-01T00:00:00Z"
    },
    {
      "id": 2,
      "name": "园所管理员",
      "description": "日常管理权限，不含系统设置",
      "is_system": false,
      "admin_count": 3,
      "permissions": [
        "dashboard:view",
        "student:manage",
        "teacher:manage",
        "parent:manage",
        "notice:manage",
        "growth:manage",
        "attendance:view",
        "pickup:view",
        "task:manage",
        "points:view",
        "announcement:manage"
      ],
      "created_at": "2020-01-01T00:00:00Z"
    }
  ]
}
```

---

## POST `/admin/roles`

> 创建角色（对应前端 `createRole`）

```json
// Request
{
  "name": "财务管理员",
  "description": "查看数据报表",
  "permissions": [
    "dashboard:view",
    "attendance:view",
    "points:view"
  ]
}

// Response
{
  "id": 3,
  "created_at": "2026-03-19T10:00:00Z"
}
```

---

## PUT `/admin/roles/:id`

> 编辑角色（对应前端 `updateRole`）

```json
// Request
{
  "name": "财务管理员",
  "description": "查看数据报表和积分管理",
  "permissions": [
    "dashboard:view",
    "attendance:view",
    "points:view",
    "points:manage"
  ]
}

// Response
{
  "id": 3,
  "updated_at": "2026-03-19T10:30:00Z"
}
```

---

## DELETE `/admin/roles/:id`

> 删除角色（仅未关联管理员的角色可删除；对应前端 `deleteRole`）

```json
// Response
{}
```

---

## GET `/admin/permissions`

> 权限树（用于角色配置；对应前端 `getPermissionTree`）

```json
// Response
{
  "permissions": [
    {
      "module": "dashboard",
      "module_label": "数据看板",
      "items": [
        { "key": "dashboard:view", "label": "查看看板" }
      ]
    },
    {
      "module": "student",
      "module_label": "学生管理",
      "items": [
        { "key": "student:view", "label": "查看学生" },
        { "key": "student:manage", "label": "管理学生" },
        { "key": "student:import", "label": "批量导入" },
        { "key": "student:export", "label": "导出名册" }
      ]
    },
    {
      "module": "teacher",
      "module_label": "老师管理",
      "items": [
        { "key": "teacher:view", "label": "查看老师" },
        { "key": "teacher:manage", "label": "管理老师" }
      ]
    },
    {
      "module": "parent",
      "module_label": "家长管理",
      "items": [
        { "key": "parent:view", "label": "查看家长" },
        { "key": "parent:manage", "label": "管理家长" }
      ]
    },
    {
      "module": "notice",
      "module_label": "通知管理",
      "items": [
        { "key": "notice:view", "label": "查看通知" },
        { "key": "notice:manage", "label": "管理通知" },
        { "key": "notice:audit", "label": "审核通知" }
      ]
    },
    {
      "module": "growth",
      "module_label": "成长记录",
      "items": [
        { "key": "growth:view", "label": "查看记录" },
        { "key": "growth:manage", "label": "管理记录" },
        { "key": "growth:audit", "label": "审核记录" }
      ]
    },
    {
      "module": "attendance",
      "module_label": "出勤管理",
      "items": [
        { "key": "attendance:view", "label": "查看出勤" },
        { "key": "attendance:export", "label": "导出报表" }
      ]
    },
    {
      "module": "pickup",
      "module_label": "接送管理",
      "items": [
        { "key": "pickup:view", "label": "查看接送" },
        { "key": "pickup:manage", "label": "管理接送" }
      ]
    },
    {
      "module": "task",
      "module_label": "亲子任务",
      "items": [
        { "key": "task:view", "label": "查看任务" },
        { "key": "task:manage", "label": "管理任务" }
      ]
    },
    {
      "module": "points",
      "module_label": "积分管理",
      "items": [
        { "key": "points:view", "label": "查看积分" },
        { "key": "points:manage", "label": "管理积分" }
      ]
    },
    {
      "module": "achievement",
      "module_label": "成就管理",
      "items": [
        { "key": "achievement:view", "label": "查看成就" },
        { "key": "achievement:manage", "label": "管理成就" }
      ]
    },
    {
      "module": "announcement",
      "module_label": "公告管理",
      "items": [
        { "key": "announcement:view", "label": "查看公告" },
        { "key": "announcement:manage", "label": "管理公告" }
      ]
    },
    {
      "module": "audit",
      "module_label": "内容审核",
      "items": [
        { "key": "audit:view", "label": "查看审核" },
        { "key": "audit:manage", "label": "执行审核" }
      ]
    },
    {
      "module": "admin",
      "module_label": "管理员",
      "items": [
        { "key": "admin:view", "label": "查看管理员" },
        { "key": "admin:manage", "label": "管理管理员" },
        { "key": "admin:role", "label": "角色权限配置" }
      ]
    },
    {
      "module": "system",
      "module_label": "系统设置",
      "items": [
        { "key": "system:kindergarten", "label": "园所设置" },
        { "key": "system:log", "label": "操作日志" }
      ]
    }
  ]
}
```
