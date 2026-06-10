# Workbook · 16 admins

> Status: **Done**

## 1. 元信息

- **模块编号**: `16` · **Sprint**: `S2`
- **规格**: [`../16-admins.md`](../16-admins.md)
- **前端**: `admin/api/admins.ts` · `admin/pages/admins/index.vue`
- **Finished**: `2026-04-21`

## 2. Domain

- `users` + `user_roles(role=admin, kindergarten_id)` + `user_admin_roles` + `admin_roles`
- 系统初始化（`system_service`）现创建首条 `admin_roles` + `user_admin_roles`，与首管理员绑定

## 3. Interface Progress

| # | 接口 | 完成 |
|---|------|:---:|
| 1 | GET `/admin/admins` | [x] |
| 2 | POST `/admin/admins` | [x] |
| 3 | PUT `/admin/admins/:id` | [x] |
| 4 | DELETE `/admin/admins/:id` | [x] |
| 5 | PUT `/admin/admins/:id/reset-password` | [x] |

## 4. Decisions

- 列表主键为 **`users.id`**；分页与 `PageResult` 对齐（`OKList`）
- 手机列表脱敏 `138****0000`；`last_login_at` 暂无数据源，省略字段
- 删除：软删 `user_roles` + `user_admin_roles`；禁止删自己、至少保留一名管理员
- 园所**尚无** `admin_roles` 时：`role_id` 无效会触发自动创建「园所管理员」；已有角色后无效 `role_id` 返回 `ROLE_NOT_FOUND`
- 写操作 `operation_logs`

## 5. Samples

- [x] `samples/16-admins/admins.http`
