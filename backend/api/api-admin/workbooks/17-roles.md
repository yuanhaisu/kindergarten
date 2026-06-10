# Workbook · 17 roles

> Status: **Done**

## 1. 元信息

- **模块编号**: `17` · **Sprint**: `S2`
- **规格**: [`../17-roles.md`](../17-roles.md)
- **前端**: `admin/api/roles.ts` · `admin/pages/roles/index.vue`
- **Finished**: `2026-04-21`

## 2. Domain

- `admin_roles`（园所维度）+ `admin_permissions`（全局目录）+ `admin_role_permissions`
- 权限目录与 `17-roles.md` 中 GET `/admin/permissions` 静态树一致；首次访问会种子写入 `admin_permissions`（含 `*` 全部权限）
- 「超级管理员」无关联权限行时列表返回 `permissions: ["*"]`

## 3. Interface Progress

| # | 接口 | 完成 |
|---|------|:---:|
| 1 | GET `/admin/roles` | [x] |
| 2 | POST `/admin/roles` | [x] |
| 3 | PUT `/admin/roles/:id` | [x] |
| 4 | DELETE `/admin/roles/:id` | [x] |
| 5 | GET `/admin/permissions` | [x] |

## 4. Decisions

- 系统内置角色（`is_system`）不可删除、不可改名；可改描述与权限关联
- 替换权限关联时对 `admin_role_permissions` **物理删除**旧行（避免软删唯一约束冲突）
- 写操作写入 `operation_logs`

## 5. Samples

- [x] `samples/17-roles/roles.http`
