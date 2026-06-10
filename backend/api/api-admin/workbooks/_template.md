# Workbook · <XX> <Module>

> **使用方法**：开工时复制本模板到 `workbooks/<XX>-<module>.md`，替换 `<...>` 占位符。
>
> Status: `Not Started` / `In Progress` / `Paused` / `Done`

---

## 1. 元信息

- **模块编号**: `<XX>`
- **模块名**: `<module>`（中文：`<中文名>`）
- **Sprint**: `S<N>`（P0 / P1 / P2 / P3）
- **接口数**: `<N>`
- **规格文档**: [`../<XX>-<module>.md`](../<XX>-<module>.md)
- **前端 API**: `admin/api/<module>.ts`
- **前端页面**: `admin/pages/<module>/index.vue`
- **前端类型**: `admin/types/api.ts`（相关段落）
- **Status**: `Not Started`
- **Started**: `YYYY-MM-DD`
- **Finished**: （未完成）

---

## 2. Domain 核对

> 检查 `server/internal/domain/` 里的实体字段是否足够支撑本模块的 Response。

| 涉及实体 | 文件 | 当前字段够吗 | 需要新增的字段 |
|---|---|---|---|
| `<Entity1>` | `server/internal/domain/<file>.go` | ✅ / ⚠️ / ❌ | `<field1>, <field2>` |
| `<Entity2>` | ... | | |

**是否需要新建表**：
- [ ] 不需要
- [ ] 需要，新表：`<table_name>`（写在 AutoMigrate 注册表里）

---

## 3. 前后端字段对照速查

> 开工前把 MD 的 Response 示例、前端 `*.ts` 的类型、domain 的 struct 三者摆一起对照。

列出本模块所有接口的关键 Response 字段。重点关注 **蛇形 vs 驼峰**、**类型（number/string）**、**嵌套层级**。

```
GET /admin/<xxx>
- list[].id            uint64      ← domain.Xxx.ID
- list[].name          string      ← domain.Xxx.Name
- list[].status        int8        ← domain.Xxx.Status  ⚠️ 0/1 还是枚举字符串？
- total                int64
- page / page_size     int

POST /admin/<xxx>
Request:
- name                 string      (必填，长度 ≤ 50)
- xxx_ids              []uint64    (至少 1 个)
Response:
- id                   uint64
- created_at           string(RFC3339)
```

---

## 4. Interface Progress（本模块的主表）

> 每完成一个接口，勾 `[x]` + 填完成时间。**同步更新 PROGRESS.md**。

| # | 方法 | 路径 | 前端函数 | Recipe 步骤 | 完成 | 完成日期 | 备注 |
|---|---|---|---|---|:---:|---|---|
| 1 | GET | `/admin/<xxx>` | `getXxxList` | 1→9 | [ ] | | |
| 2 | GET | `/admin/<xxx>/:id` | `getXxx` | 1→9 | [ ] | | |
| 3 | POST | `/admin/<xxx>` | `createXxx` | 1→9 | [ ] | | |
| 4 | PUT | `/admin/<xxx>/:id` | `updateXxx` | 1→9 | [ ] | | |
| 5 | DELETE | `/admin/<xxx>/:id` | `deleteXxx` | 1→9 | [ ] | | |

Recipe 步骤（来自 `EXECUTION_PLAN.md` §5）：
1. 读文档 → 2. 读前端 → 3. 核对 domain → 4. Repo → 5. Service → 6. DTO → 7. Handler → 8. 路由 → 9. Wire

---

## 5. 预计改动文件清单

开工时预测，过程中补充实际改动：

### 新增

- [ ] `server/internal/repository/<module>_repo.go`
- [ ] `server/internal/service/<module>_service.go`
- [ ] `server/internal/handler/http/<module>.go`

### 修改

- [ ] `server/internal/domain/<relevant>.go`（加字段）
- [ ] `server/internal/repository/wire.go`（加 Provider）
- [ ] `server/internal/service/wire.go`（加 Provider）
- [ ] `server/internal/handler/http/handler.go`（加字段 & 构造参数）
- [ ] `server/internal/app/router.go`（加路由组）
- [ ] `server/internal/app/wire_gen.go`（`go generate` 自动重生）

### 前端待补（如果有）

- [ ] `admin/api/<module>.ts`（补函数 `<fnName>`）
- [ ] `admin/pages/<module>/index.vue`（调用补的函数）

---

## 6. Decisions & Notes

> 记录实现过程中的关键决策和注意事项。

- `[YYYY-MM-DD]` <决策/注意事项>
- `[YYYY-MM-DD]` ...

---

## 7. Questions（待用户决策）

> 规格不清晰或可选方案的问题，记在这里，收尾时向用户汇报。

- [ ] `[YYYY-MM-DD]` <问题描述>
  - 选项 A：...
  - 选项 B：...
  - 暂时采用：...（可在用户确认后更新）

---

## 8. Test Samples

> REST Client 样例放在 `samples/<XX>-<module>/`（可选但推荐）。

- [ ] `samples/<XX>-<module>/list.http`
- [ ] `samples/<XX>-<module>/create.http`
- [ ] `samples/<XX>-<module>/update.http`
- [ ] `samples/<XX>-<module>/delete.http`

---

## 9. DoD Checklist（收尾时填）

### 接口级（每个接口都要过）
- [ ] Response JSON 字段与 MD 规格一致
- [ ] 空列表返回 `[]` 而不是 `null`
- [ ] Service 层有 `kindergarten_id` 过滤
- [ ] 写操作写入 `operation_logs`（或明确豁免）
- [ ] `go build ./...` 通过

### 模块级
- [ ] 所有接口 Interface Progress 全勾
- [ ] `admin/pages/<module>/index.vue` 跑过 CRUD（用户确认）
- [ ] `PROGRESS.md` 该模块状态改为 ✅
- [ ] `README.md` 模块索引状态改为 ✅
- [ ] `SESSION_HANDOFF.md` 已更新
- [ ] Git commit 已建议用户执行

---

## 10. Session Log（按时间倒序）

### Session N · YYYY-MM-DD

- **Duration**: `~Xh`
- **Focus**: `<接口范围>`
- **Completed**: `<完成了什么>`
- **Blocked By**: `<遇到什么问题>`
- **Next**: `<下次该做什么>`

---

## 11. Final Summary（完成后填）

- **总会话数**: `N`
- **总工时（估）**: `Xh`
- **Git commits**: `<hash1>, <hash2>`
- **踩坑 & 经验**：（简要列 3–5 条，给后续模块参考）
  - ...
