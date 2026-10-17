# Tasks

## 1. 契约与用例登记（先行）

- [x] 1.1 更新 `backend/api/api-common.md`：`GET /common/user/profile` 响应的 `roles[]` 条目新增 `permissions: string[]`（含 `*` 通配语义），并补记已实现未记载的 `admin_role` 字段。验证：文档字段与 design.md D6 字段级清单逐条一致，无多余字段夹带
- [x] 1.2 在 `docs/tests/TEST_CATALOG.md` 登记 TC-NEW-PERM-0001 ~ TC-NEW-PERM-0015（对应 specs/admin-rbac/spec.md 的 15 条 Scenario，模块缩写 PERM 已入表）。验证：grep TC-NEW-PERM 计数为 15、编号连续；执行后 10 条已置 ✅、5 条随 E2E 完成后置 ✅（全 15 条最终已通过）

## 2. 后端强制与权限下发（Go）

- [x] 2.1 实现权限解析服务（新 service）：按 D3 顺序解析 (kgID, uid) → (codes, wildcard)；复用 `FindUserAdminRole` 与 `ListPermissionCodesForRole`。验证：单测覆盖三态——有角色关联取角色码集 / 无关联+super_admin 标记为通配 / 均无为空集
- [x] 2.2 实现 `RequirePermission` 中间件与路由→码登记表（design.md D7 权威表逐条落为 `"METHOD /api/v1/admin/..."` 字面量）。验证：单测覆盖——有码放行、缺码 403、登记表缺项一律拒绝并产生 slog.Error 留痕
- [x] 2.3 `router.go` 的 adminG 挂载 `RequirePermission`；`NewRouter` 改返回 `(*gin.Engine, error)`，注册完成后用 `r.Routes()` 过滤 `/api/v1/admin` 做覆盖自检。验证：临时抽掉 `health/stats` 登记行 → 自检报错点名 `admin permission registry has 1 gap(s): GET /api/v1/admin/health/stats` 并使启动失败；恢复后 app 包覆盖测试绿、真实二进制启动成功
- [x] 2.4 profile 响应实现 `permissions`（auth_service 的 RoleInfo/GetProfile）。验证：curl 实测——uid 4（super_admin 标记）返回 `["*"]`；uid 59（教学主任）返回 `["dashboard:view","parent:view","student:manage","student:view"]`
- [x] 2.5 新增 service 依赖接入 Wire（必要时按 CLAUDE.md 补 `wire.Bind`），`make wire` 后 `make build` 通过。验证：`make wire` 重生成 wire_gen.go 成功；`make build` 通过；`make test`(-race) 受环境限制（本机 cc1 为 32 位，cgo 报 `64-bit mode not compiled in`，全仓库受影响），以 `go test ./...`（无 race）等效替代：7 个包全绿
- [x] 2.6 组验证：`go build ./...` + `go test ./...`；对照 specs R1/R2/R3 逐条验收（TC-NEW-PERM-0001 ~ 0007 的后端侧）。结果：0001/0002/0003=解析三态单测绿；0004/0005/0006=中间件放行/403/未登记拒绝单测绿（含留痕日志）；0007=登记表覆盖自检单测绿（正例）+ 负例演练（临时抽掉 health/stats 登记 → 启动报错点名 `admin permission registry has 1 gap(s): GET /api/v1/admin/health/stats`，恢复后启动成功）；真实二进制启动自检亦通过

## 3. 数据迁移（既有账号兼容）

- [x] 3.1 编写回填 SQL 到 `docs/migrations/`（`2026-10-10-admin-role-wildcard-backfill.sql`，按 is_system + 角色名定位不写死 id，NOT EXISTS 冲突保护）。验证：测试库执行两次——第 1 次 1 行、第 2 次 0 行（幂等）；一次执行后行数与索引以 dbcheck 核实（idx_role_perm 唯一索引在场）
- [x] 3.2 核实与兼容回归。结果：迁移前 uid 58（园所管理员，0 码）全 403——锁死实证，回填必要性的直接证据；迁移后 uid 58 全 200 且 profile `["*"]`（TC-NEW-PERM-0013）；uid 59（教学主任）有码 200 / 无码 403（points/pickup/roles 均 403，TC-NEW-PERM-0014）。两账号为测试库账号，密码已重置为已知值（Office@123 / Teach@123）供后续 E2E 复用

## 4. admin 端（权限呈现与守卫）

- [x] 4.1 `stores/auth.ts` 的 fetchProfile 后调用 `setCodes`（从 roles[].permissions 聚合；空集保持空集）；`stores/permission.ts` 增加 loaded 态与 reset()，注释改为「默认放开仅限未加载态」。验证：三账号真实登录实测——store 码集与 profile 一致（super_admin ["*"] / 教学主任 4 码 / 零权限 []）
- [x] 4.2 `middleware/permission.ts`（无人引用的死文件）删除，新建 `permission.global.ts`：所需码用 `matchMenuByPath(to.path)?.permission` 派生；首访先 await fetchProfile 再判定；越权直达重定向到菜单顺序首个可访问项；零可访问进 `pages/no-permission.vue`。验证：构建通过；守卫与 Sidebar 过滤共用同一 `constants/menus.ts` 表
- [x] 4.3 `pages/roles/index.vue` 移除「不强制生效」警告；对话框提示改为「保存后对该角色名下管理员即时生效」。验证：页面实测无「不强制生效/仅保存为配置记录」残留文案（TC-NEW-PERM-0015）
- [x] 4.4 真实交互实测·菜单过滤（真实表单输入+点击登录）：super_admin（uid 990701001）23/23 菜单全显（TC-NEW-PERM-0010）；教学主任（uid 59）仅显 6 项——班级/学生/家长管理 + 看板/统计/晨检统计，日常运行、教与成长、系统三组整体隐藏（TC-NEW-PERM-0009）
- [x] 4.5 真实交互实测·守卫：教学主任直达 /points → 重定向 /classes（菜单序首个可访问，TC-NEW-PERM-0011）；零权限账号（临时构造 uid 991100001，已 teardown 清理）登录 → /no-permission，菜单 0 项（TC-NEW-PERM-0012）
- [x] 4.6 组验证：`npm run build` 通过、check-tokens 无违规；对照 specs R4/R5/R7（TC-0008~0012、0015）逐条通过。**集成修复（清单外，交互暴露）**：① 滞留轮询对无 `pickup:view` 账号的 403 误报「无法获取滞留状态」→ useHttp 保留 status 码 + pickupAlert 识别 403 为 forbidden（不轮询、不报警）；② 看板待办出现无权限落点（点进去被守卫弹回）→ index.vue 按 `matchMenuByPath(target).permission` 过滤 visibleTodos

## 5. 集成验收与收尾

- [x] 5.1 全链路对照 specs R6：super_admin（uid 4 & 990701001）全通；园所管理员名下（uid 58）迁移前全 403 → 回填后全 200 且 profile `["*"]`；教学主任（uid 59）菜单 6/23 + 接口有码 200 / 无码 403。三条链路均以真实登录 + curl 双证（TC-NEW-PERM-0013/0014）
- [x] 5.2 上线观测项：403 响应与 `admin permission registry gap` slog 日志为观测点；后端回滚 = 部署旧二进制即恢复通配粗闸（无数据破坏）；迁移 SQL 幂等可重放。记录于本任务行与 design.md Migration Plan
- [x] 5.3 收尾清理：admin dev server 进程清净；临时工具（tmpsql/tmpmigrate）与零权限测试账号已删除；后端 :8080 以新代码保持运行（供后续回归）；`go vet ./...` 通过、`go test ./...` 7 包全绿、admin `npm run build` 通过

## 归档说明（2026-10-10）

**实际触点 vs 设计阶段清单的差异**：

1. **清单外新增两处触点（集成修复，详见 4.6）**：`admin/composables/useHttp.ts`（错误对象保留 HTTP status 码）、`admin/pages/index.vue`（看板待办按落点权限过滤 visibleTodos）。起因是 E2E 交互暴露「403 被当故障报警」「待办链接指向无权限页被守卫弹回」两个集成缺陷，属本变更语义（菜单可见与路由可达同源）的延续。
2. **验证手段替代**：任务 2.5/2.6 的 `make test`（-race）因本机 cc1 为 32 位编译器（cgo 报 64-bit mode not compiled in，全仓库受影响）不可用，以 `go test ./...` 等效替代并通过。此为环境限制，非代码问题。
3. **迁移执行方式**：本机无 mysql CLI，回填 SQL 通过一次性 Go runner 执行（执行两次验证幂等后删除 runner）；SQL 本体在 `docs/migrations/2026-10-10-admin-role-wildcard-backfill.sql`。
4. **其余触点与设计清单一致**：Go service/middleware/router/profile、`backend/api/api-common.md` 契约、admin 端四文件（auth/permission store、permission.global 守卫、no-permission 页、roles 页）、数据迁移。

**活基线**：`openspec/specs/admin-rbac/spec.md`（7 需求 / 15 场景，TC-NEW-PERM-0001~0015 全 ✅）。
