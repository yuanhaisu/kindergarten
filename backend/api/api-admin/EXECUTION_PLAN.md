# Admin 端接口实现 · 执行计划（总纲）

> 本文件为 **长期稳定文档**，描述方法论、约定和 DoD，一般不会频繁改动。
>
> 配套文件：
> - [`KICKOFF_PROMPT.md`](./KICKOFF_PROMPT.md) — 新会话开场提示词模板（**一模块一会话的入口**）
> - [`PROGRESS.md`](./PROGRESS.md) — 87 个接口逐条进度（每次会话更新）
> - [`SESSION_HANDOFF.md`](./SESSION_HANDOFF.md) — 全局会话接力卡片（每次结束会话前更新）
> - [`workbooks/`](./workbooks/) — 每模块一份工作簿（模块内部状态快照）
>
> **下次继续时**：用户只需复制 `KICKOFF_PROMPT.md` 顶部的一句话发到新会话，AI 会自动完成"读文档 → 确认状态 → 开工 → 结账"全流程。

---

## 0. 一模块一会话 · 工作流（强制）

> 本计划规定：**一次会话只做一个模块（或 S0 脚手架），做完即结账**。
>
> 这样做的原因：
> - 每次会话上下文容量有限，模块切换容易丢信息
> - 模块完成后有天然的 commit & 验证断点
> - 新会话零成本恢复：读 3 份文档即可续接
> - 你/AI 都能清晰地看到进度节奏

### 0.1 会话开场 · 用户操作

1. 打开新的 Cursor Chat
2. 复制 `KICKOFF_PROMPT.md` 顶部 3 种提示词之一（自动续接 / 指定模块 / S0 脚手架）
3. 粘贴发送，等 AI 汇报目标 + 简短确认即可

### 0.2 会话开场 · AI 操作（强制顺序）

1. **读 3 份核心文档**：`SESSION_HANDOFF.md` → `PROGRESS.md` → 本文件 §5/§6/§7/§9
2. **定目标**：按用户提示词 + `Next Action` 确定本次模块
3. **读规格**：`<XX>-<module>.md` + `admin/api/<module>.ts` + `admin/pages/<module>/index.vue` + `admin/types/api.ts`
4. **读 domain**：`server/internal/domain/<relevant>.go`
5. **建/读工作簿**：`workbooks/<XX>-<module>.md`（若不存在则从 `_template.md` 复制创建）
6. **汇报给用户**：目标、预期产出、预计改动文件，等确认

### 0.3 会话进行中 · AI 操作

- 按 §5 Recipe 9 步串行实现每个接口
- 每完成 1 个接口：勾 `PROGRESS.md` + 更工作簿 + `go build ./...` 通过
- 遇到规格不清晰 → 写入工作簿 `Questions` 并立即@用户确认（不要擅自决策）
- 遇到需要改 `EXECUTION_PLAN.md` 方法论的情况 → **先停下**，向用户请示

### 0.4 会话结账 · AI 操作（不做不能结束）

1. 跑 `go generate ./internal/app` 重生 Wire
2. 跑 `go build ./...` 全量通过
3. `PROGRESS.md` 模块状态 🚫 → ✅（若模块全部完成）
4. `README.md` 模块索引状态同步 ✅
5. 工作簿填 DoD checklist + Final Summary
6. `SESSION_HANDOFF.md` 追加 Session Log + 更新 `Next Action`
7. 建议用户 commit，给出 commit message 模板
8. 输出"交接语"告诉用户下次启动该说哪句话

### 0.5 硬性约束

| # | 约束 |
|---|---|
| 1 | ❌ 不得跳过会话开场的 3 份文档阅读 |
| 2 | ❌ 不得跳过会话结账的 `SESSION_HANDOFF.md` 更新 |
| 3 | ❌ 不得在一次会话里同时推进 2+ 模块 |
| 4 | ❌ 不得擅自改 MD 规格里的 JSON 字段名 |
| 5 | ❌ S0 脚手架未完成前不得开始业务模块 |
| 6 | ✅ 必须在 Service 层做 `kindergarten_id` 过滤 |
| 7 | ✅ 必须让 `go build ./...` 通过才能勾 `[x]` |
| 8 | ✅ 遇到规格歧义必须写入工作簿 `Questions` |

---

## 1. 范围 & 目标

- **范围**：`backend/api/api-admin/` 下 20 个 markdown 所列的 **87 个 admin 业务接口**。
- **目标**：每个接口达到 §6 的 Definition of Done，前端 `admin/pages/*` 能无 404 跑通 CRUD。
- **非目标**：不在本计划内处理 teacher / parent 端接口（在 `api-teacher.md` / `api-parent.md`，另起计划）。

---

## 2. 现状盘点（基线，2026-04-20）

| 维度 | 现状 |
|---|---|
| 后端已实现接口 | 18 个（auth / common / system / messages），见 `server/internal/app/router.go` |
| domain 模型 | 几乎全部实体已建模，见 `server/internal/domain/` |
| repo / service / handler | 仅 `user / userRole / kindergarten / file / message` 有三层 |
| 前端 API 封装 | `admin/api/*.ts` 23 个文件 **全部就绪** |
| 前端页面 | 22 个页面（22 = 登录 + 看板 + 20 业务页），所有业务表格空态 |
| RBAC 策略 | `server/configs/rbac_policy.csv` 里已有 `p, admin, /api/v1/*, *`，**admin 角色已全放行**，无需每模块改策略 |
| Wire | `go generate ./internal/app` 生成 `wire_gen.go`；新模块需在 `repository/wire.go`、`service/wire.go`、`handler/http/wire.go` 追加 |
| 统一响应 | `server/pkg/response/response.go` 已提供 `OK / OKList / Created / ErrXxx` |

---

## 3. 核心策略

1. **文档驱动**：以 `XX-*.md` 的 JSON 示例为唯一真源，字段名/层级必须完全一致。
2. **纵向切片**：一次做完一个接口的 `domain → repo → service → handler → router` 五层，再做下一个。
3. **按优先级串行**：P0 → P1 → P2 → P3，严格串行，不并行（单人开发）。
4. **模块内按 CRUD 顺序**：`GET list → GET detail → POST → PUT → DELETE → 业务动作`。

---

## 4. Sprint 划分

| Sprint | 内容 | 接口数 | 预计工时 | 状态 |
|---|---|---:|---|---|
| **S0** | 脚手架（分页工具、AdminGroup、Wire 模板、错误码、开发约定） | — | 1–2 天 | ⏳ 待开始 |
| **S1** | P0：dashboard(4) + classes(6) + students(8) | 18 | 5–7 天 | ⏳ |
| **S2** | P1：teachers(6) + parents(5) + settings(6) + admins(5) + roles(5) | 27 | 6–8 天 | ⏳ |
| **S3** | P2a：attendance(3) + leaves(2) + pickup(4) | 9 | 3 天 | ⏳ |
| **S4** | P2b：growth(5) + tasks(4) + points(5) + achievements(4) | 18 | 5–6 天 | ⏳ |
| **S5** | P3：notices(4) + announcements(6) + audit(3) + logs(1) + export(4) | 18 | 4–5 天 | ⏳ |
| **S6** | 缓冲 + 全量 e2e（跑 `admin/tests/playwright-*`） | — | 2–3 天 | ⏳ |

合计 **4–5 周**。

---

## 5. 单接口实现配方（Recipe，9 步）

以 `GET /admin/classes` 为例，每个接口统一走这 9 步：

1. **读文档**：`backend/api/api-admin/02-classes.md` 的 Query / Response 示例。
2. **读前端**：`admin/api/classes.ts` 的函数签名 + `admin/types/api.ts` 类型 + `admin/pages/classes/index.vue` 里怎么用。
3. **核对 domain**：`server/internal/domain/kindergarten.go`（Class 在此文件）字段是否够用，不够就加字段 + 让 AutoMigrate 处理。
4. **写 Repo**：`server/internal/repository/class_repo.go`，纯 SQL，返回 domain 实体。
5. **写 Service**：`server/internal/service/class_service.go`，聚合/组装 VO；多租户过滤（`kindergarten_id`）必须在此层兜底。
6. **写 DTO**：优先放在 `handler/http/xxx.go` 同文件 `type XxxReq / XxxResp struct`，JSON tag 用蛇形，**严格对照 MD 示例**。
7. **写 Handler**：`server/internal/handler/http/class.go`，`ShouldBindJSON` + `ShouldBindQuery` + 调 service + `response.OK/OKList`。
8. **注册路由**：`server/internal/app/router.go` 里的 `adminG`（见 S0 会新建）里加一行 `adminG.GET("/classes", h.inner.ListClasses)`。
9. **注入 Wire**：`repository/wire.go` 追加 `NewClassRepo`、`service/wire.go` 追加 `NewClassService`、`handler/http/handler.go` 的 `Handler` 结构体追加字段；`go generate ./internal/app` 重生。

---

## 6. Definition of Done（单接口）

每个接口必须满足以下 6 条才算完成，在 `PROGRESS.md` 勾上 `[x]`：

1. ✅ Response JSON 字段名、层级、类型 **与 `XX-*.md` 示例完全一致**。
2. ✅ `admin/pages/<module>/index.vue` 打开后该接口相关操作无 404、无 500、表格/弹窗正常渲染。
3. ✅ 列表为空时返回 `"list": []` 而非 `null`。
4. ✅ 多租户过滤（`kindergarten_id`）已在 Service 层兜底（若该接口涉及园所数据）。
5. ✅ 写操作接口（`POST/PUT/DELETE`）已写入 `operation_logs`（或明确豁免并在 PROGRESS 中注明）。
6. ✅ 用 `curl` 或 REST Client 至少跑过一次，请求样例留在 `backend/api/api-admin/samples/`（可选，推荐）。

---

## 7. Definition of Done（单模块）

除了每个接口的 DoD，模块级还需：

1. ✅ 所有接口勾完。
2. ✅ `admin/pages/<module>/index.vue` 整页跑一遍 CRUD（新增/编辑/删除/搜索/翻页），无报错。
3. ✅ `PROGRESS.md` 该模块状态从 🚫 改为 ✅。
4. ✅ `SESSION_HANDOFF.md` 更新。
5. ✅ Git commit：`feat(admin): complete <module> module (x/87)`。

---

## 8. S0 脚手架详细任务清单

> S0 是后续 5 个 Sprint 的前置，**必须先做完**。

S0-1. **公共分页工具**
- 新增 `server/pkg/pagination/pagination.go`：`Query{Page, PageSize, Keyword}`、`Normalize()`（page 默认 1、page_size 默认 20 上限 100）、`Offset()`。
- Response 复用已有 `pkg/response.OKList`。

S0-2. **Admin 路由组抽取**
- `server/internal/app/router.go` 中抽 `registerAdminRoutes(adminG *gin.RouterGroup, h *handler.Handler)`，后续每个模块在此文件里 **只加一个分组块**。

S0-3. **Claims / Context helper**
- `server/internal/middleware/context.go`（或 `claims.go`）新增 `GetUserID(c) uint64`、`GetKindergartenID(c) uint64`、`GetRoles(c) []string`。
- 业务 handler 统一用它取，不再直接访问 `c.MustGet("claims")`。

S0-4. **Wire Set 骨架**
- 确认 `repository/wire.go`、`service/wire.go`、`handler/http/wire.go` 存在且导出 `Set`。
- **已确认存在**，S0 不用改，只是告知约定：每个新模块要在对应 `Set` 里追加一行。

S0-5. **Handler 聚合器说明**
- `server/internal/handler/http/handler.go` 中的 `Handler` 结构体是所有模块 handler 方法的宿主；
- 每加一个模块要往 `Handler` 里加字段（如 `ClassSvc *service.ClassService`）+ 构造函数参数。

S0-6. **AutoMigrate 注册表**
- 确认 `server/internal/service/system_service.go` 里 AutoMigrate 的实体列表，后续新增 domain 字段不需要改这里（GORM 自动加列），但若新建表（如 `class_teachers` 关联表）需要在这里追加。

S0-7. **DB 种子（可选）**
- 若本地库为空，写一个 `cmd/seed/main.go`（或加到 `system_service` 初始化）种 1 个园所 + 3 个班 + 10 个学生，方便前端联调。

S0-8. **REST Client 脚本模板**
- `backend/api/api-admin/samples/_login.http`：一键获取 admin token，其他 `.http` 文件复用 `@token` 变量。
- 后续每个接口对一份 `.http` 样例，commit 进仓库（这是"离线文档"）。

---

## 9. 关键约定

### 9.1 命名
- **Go 包 & 文件**：小写下划线：`class_repo.go`、`class_service.go`、`class.go`（handler）。
- **Go 类型**：驼峰：`ClassRepo`、`ClassService`、`CreateClassReq`、`ClassListResp`。
- **JSON 字段**：蛇形，**严格对照 MD 示例**，不要自作主张换名。

### 9.2 分页参数
- Query：`?page=1&page_size=20&keyword=xxx`
- Response：`{ "list": [...], "total": 123, "page": 1, "page_size": 20 }`（用 `pkg/response.OKList`）

### 9.3 时间/ID
- 时间用 RFC3339（GORM 默认 + `time.Time`）。
- ID 用 `uint64`（JSON 数字 ≤ 2^53 安全范围）。

### 9.4 多租户
- 所有涉及业务数据的 Service 方法 **第一行** 从 context 拿 `kindergarten_id` 并加到 where 条件。
- 超级管理员（`super_admin` 角色）是否跨租户？当前假设 **不跨租户**（每个 admin 只能管自己园所），若有例外在对应接口备注。

### 9.5 软删除
- domain 里用 `SoftDeleteModel`（已存在）的接口默认软删，不要物理删。
- DELETE 接口返回 `{}`（空对象），HTTP 200。

### 9.6 导出 / Blob 接口
- `GET /admin/xxx/export`、`POST /admin/export/xxx` 不走统一 Envelope，直接流式返回 `application/octet-stream` + `Content-Disposition: attachment; filename=...`。
- 异步导出任务走单独的 `export_tasks` 表，状态机：`pending → running → success | failed`。

---

## 10. 会话接力协议

### 10.1 结束会话前必做 3 件事

1. 把当前会话完成的接口在 `PROGRESS.md` 打勾。
2. 更新 `SESSION_HANDOFF.md`：
   - `Last Session Date`
   - `Completed This Session`（列表）
   - `Next Action`（明确到"做哪个接口的第几步"）
   - `Blockers`（如果有）
   - `Last Modified Files`（文件路径列表）
3. 若代码有阶段性稳定，git commit。

### 10.2 开始新会话时必做 3 件事

1. 读 `SESSION_HANDOFF.md` 确认 `Next Action`。
2. 读 `PROGRESS.md` 确认当前 Sprint 和剩余项。
3. 若 `Blockers` 不为空，先处理 blocker，否则直接按 Recipe 9 步开工。

---

## 11. 风险 & 注意

1. **`teachers` / `parents` 是 user 的角色切面**，底层同一张 `users` 表 + `user_roles` + `user_profile`，VO 里要挑合适字段返回。
2. **前端类型严格**：`admin/types/api.ts` 里的类型若字段对不上，前端 TS 会报错。改动前查类型文件。
3. **Casbin policy 已全放行 admin**，**不需要**为每个 admin 接口加策略；但要做好 JWT claims 里 roles 包含 `admin`（或 `super_admin` 映射到 admin）。
4. **前端 `api/leaves.ts`、`api/tasks.ts`、`api/notices.ts` 有 5 个待补函数**（见 `README.md` §前端待补），后端做对应接口时要同步补前端封装 + 页面调用。
5. **Wire 重新生成**：改了 `*Set` 或新增 Provider 后，必须跑 `go generate ./internal/app`，否则 `wire_gen.go` 不更新会编译失败。

---

## 12. 参考链接

- 接口规格目录：[`README.md`](./README.md)
- 缺失清单原始报告：[`../../../admin/tests/playwright-reports/20260420/BACKEND_MISSING_APIS.md`](../../../admin/tests/playwright-reports/20260420/BACKEND_MISSING_APIS.md)
- 后端路由：`server/internal/app/router.go`
- 响应封装：`server/pkg/response/response.go`
- Casbin 策略：`server/configs/rbac_policy.csv`
- 前端 API 目录：`admin/api/`
- 前端页面目录：`admin/pages/`
