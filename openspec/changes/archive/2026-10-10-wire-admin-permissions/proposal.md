# Proposal: wire-admin-permissions 管理端权限端到端强制

## Why

管理端「角色权限」今天是三层脚手架、零承重：Casbin 管理端策略是通配（`p, admin, /api/v1/*, *`）、JWT 不带权限、客户端 `setCodes` 全库零调用（常量 `['*']`）、23 个菜单项对所有人放行。园长在角色页为「保健医」配出「只能看晨检」只是一句 UI 承诺——对第一原则是「可核实」、第四原则是「儿童隐私是底线」的产品，**假锁比无锁更糟**。导航评审（2026-10-09）将权限假锁列为 P0；当时的处理是诚实告知（角色页常驻警告），本次把强制做成真的。

## What Changes

- **服务端强制**：admin 路由组挂权限中间件，按静态「路由 → 权限码」登记表（约 90 条，覆盖 dashboard/classes/students/teachers/parents/settings/kindergarten/admins/roles/permissions/attendance/leaves/pickup/growth/tasks/points/achievements/notices/announcements/audit/export/logs/health/dicts）逐条强制查看码/管理码/导出码；查询不到的已注册路由一律拒绝。启动时自检登记表覆盖全部 admin 路由，缺失即启动失败。
- **权限解析**：按请求解析账号权限码（不做缓存，天然即时生效）——`UserAdminRole` 关联优先；无关联且 `UserRole.AdminRole == "super_admin"` 视为通配 `*`；两者皆无视为空集。**BREAKING**：既有非系统角色将按其已配权限收紧访问（如「教学主任」仅剩其 4 个码覆盖的范围）——这正是「假锁变真锁」的预期行为。
- **迁移兼容**：系统角色「园所管理员」（is_system，当前权限为空）回填 `*` 通配，其名下管理员保持与强制前等价的访问；园长可随后在角色页收紧。`super_admin` 标记账号（uid 4 等）保持全量访问。
- **契约变更（显式声明）**：**本次将修改 `backend/api/`，属于契约变更。** `GET /common/user/profile` 的 `roles[]` 条目新增 `permissions: string[]`（含 `*` 通配语义）；同时补记已实现但契约未记载的 `admin_role` 字段。
- **admin 端**：登录/取资料后调用 `setCodes`；侧栏按权限过滤（过滤架构已就位，只差数据）；路由守卫从「与菜单同源」的菜单表派生所需权限码（不再依赖无人设置的 `definePageMeta`），越权直达重定向到首个可访问页；全部不可访问时进入新增的「无权限」页。
- **角色页去谎**：强制生效后移除「权限码仅保存为配置记录，当前版本不强制生效」警告——该提示届时会变成反向谎言。

**明确不做的（out of scope）**：

- **不改 teacher/ 与 parent/ 端**：本次仅涉及 admin 端；两端权限模型与接口均不变。
- **不新增任何权限码**。`health:*`（保健医晨检）、`class:*`（班级独立于学生）、`leave:*`（请假独立于出勤）三组码在 design-data/（仅有「保健医生」**角色名**，无晨检权限模块设计）、dict 字典、代码常量集合三处均无权威来源 → **阻塞项，待产品决策**。保健医在管理端暂以现有码近似表达（晨检统计沿用 `dashboard:view`）。
- 不做按钮级细粒度（管理码只到路由级；manage/audit/export 等码绑定在对应端点上）。
- 不改 Casbin 策略文件（`admin` 通配保留为粗闸，本次在 Go 中间件加细闸；新增路由仍需同步 `rbac_policy.csv` 的铁律不变）。
- 不做权限变更推送/缓存失效（逐请求解析已保证即时性）。

## Capabilities

### New Capabilities

- `admin-rbac`: 管理端角色权限的解析、下发与强制——权限码解析规则（角色关联优先 / super_admin 通配兜底 / 空集拒绝）、服务端路由级强制与登记表启动自检、客户端权限下发与菜单过滤与路由守卫、既有账号迁移兼容、角色页陈述与强制行为一致。

### Modified Capabilities

（无。现有 specs：achievement-award / growth-feed / notice-targeting / pickup-custody / teacher-pickup-registration 均不定义管理端权限行为。）

## Impact

- **Go**：`internal/middleware/`（新增 RequirePermission 中间件 + 路由→码登记表）、`internal/app/router.go`（adminG 挂中间件 + 启动自检）、`internal/service/`（新增权限解析服务，复用现有 `FindUserAdminRole` / `ListPermissionCodesForRole`）、`internal/handler/http/`（profile 响应增加 permissions）。新服务需要 `make wire` + 必要时 `wire.Bind`。
- **契约**：`backend/api/api-common.md`（profile 响应新增 `permissions`、补记 `admin_role`）。
- **admin/**：`stores/auth.ts`（fetchProfile 后 setCodes）、`stores/permission.ts`（接线）、`middleware/permission.ts`（改全局并从菜单表派生）、新增 `pages/no-permission.vue`、`pages/roles/index.vue`（移除警告）。
- **数据**：一次性迁移——`admin_role_permissions` 回填系统角色「园所管理员」的 `*` 权限（SQL 入库 `docs/migrations/`，用 `go run ./cmd/dbcheck admin_role_permissions` 核实行数）。
- **不影响**：teacher/、parent/ 全部代码；`rbac_policy.csv`；`backend/api/` 其余文档。
