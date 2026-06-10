# Workbook · 05 parents

> Status: `Done`

---

## 1. 元信息

- **模块编号**: `05`
- **模块名**: `parents`（中文：家长管理）
- **Sprint**: `S2`（P1）
- **接口数**: `5`
- **规格文档**: [`../05-parents.md`](../05-parents.md)
- **前端 API**: `admin/api/parents.ts`
- **前端页面**: `admin/pages/parents/index.vue`
- **前端类型**: `admin/types/api.d.ts`（`ParentItem`）
- **Status**: `Done`
- **Started**: `2026-04-21`
- **Finished**: `2026-04-21`

---

## 2. Domain 核对

| 涉及实体 | 文件 | 当前字段够吗 | 需要新增的字段 |
|---|---|---|---|
| User + UserRole | `user.go` | ✅ | — |
| StudentParent | `student.go` | ✅ | — |
| ParentProfile | `parent.go`（新建） | ✅ | `real_name`, `status`, `last_login_at`（可选） |

**是否需要新建表**：

- [x] 需要，新表：`parent_profiles`（已加入 `provider.go` AutoMigrate）

---

## 3. 前后端字段对照速查

列表 `children[]`：`student_id`, `student_name`, `class_name`, `relation`, `relation_label`  
详情额外：`student_avatar`, `class_id`, `is_primary`, `last_login_at`（无登录记录时为空字符串）

---

## 4. Interface Progress

| # | 方法 | 路径 | 前端函数 | Recipe 步骤 | 完成 | 完成日期 | 备注 |
|---|---|---|---|---|:---:|---|---|
| 1 | GET | `/admin/parents` | `getParents` | 1→9 | [x] | 2026-04-21 | |
| 2 | GET | `/admin/parents/:id` | `getParent` | 1→9 | [x] | 2026-04-21 | |
| 3 | POST | `/admin/parents` | `createParent` | 1→9 | [x] | 2026-04-21 | 新用户默认密码 `KindyParent#1` |
| 4 | PUT | `/admin/parents/:id` | `updateParent` | 1→9 | [x] | 2026-04-21 | 全量替换本园 `student_parents` |
| 5 | PUT | `/admin/parents/:id/status` | `updateParentStatus` | 1→9 | [x] | 2026-04-21 | 1=正常 2=禁用 |

---

## 5. 预计改动文件清单

### 新增

- [x] `server/internal/domain/parent.go`
- [x] `server/internal/repository/parent_repo.go`
- [x] `server/internal/service/parent_service.go`
- [x] `server/internal/handler/http/parent.go`
- [x] `backend/api/api-admin/samples/05-parents/parents.http`

### 修改

- [x] `server/internal/repository/user_role_repo.go`（`DeleteParentRole`）
- [x] `server/internal/repository/wire.go`
- [x] `server/internal/service/wire.go`
- [x] `server/internal/handler/http/handler.go`
- [x] `server/internal/app/router.go`
- [x] `server/internal/app/provider.go`
- [x] `server/internal/app/wire_gen.go`（手改；`go generate` 因网络 502 未跑通）

---

## 6. Decisions & Notes

- `[2026-04-21]` 家长主键为 **`users.id`**，与老师模块一致。
- `[2026-04-21]` 新增 **`parent_profiles`** 存 `real_name`、家长维度 `status`（1/2）；`last_login_at` 预留，当前多为空字符串。
- `[2026-04-21]` 列表 INNER JOIN `parent_profiles`：仅含档案的家长会出现（本模块创建路径会同时写 role+profile）。

---

## 7. Questions

无。

---

## 9. DoD Checklist

### 接口级
- [x] Response JSON 字段与 MD 规格一致
- [x] 空列表返回 `[]`
- [x] Service 层 `kindergarten_id` 过滤
- [x] 写操作 `operation_logs`
- [x] `go build ./...` 通过

### 模块级
- [ ] `admin/pages/parents/index.vue` 全页 CRUD（需用户本地起 server+admin 确认）
- [x] `PROGRESS.md` 模块 ✅
- [x] `README.md` 模块索引 ✅
- [x] `SESSION_HANDOFF.md` 已更新

---

## 11. Final Summary

- **Git commits**: 建议 `feat(admin): complete parents module (29/87)`
