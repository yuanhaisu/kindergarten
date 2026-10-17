# Design: wire-admin-permissions

动机见 proposal.md「Why」；行为需求与验收见 specs/admin-rbac/spec.md。本文只写「怎么做」与为什么这么选。

## Context

当前强制链路的真实状态（已核实）：

- Casbin 管理端策略 `p, admin, /api/v1/*, *`——所有 admin 角色账号粗放行；`admin` 是 JWT 里的粗角色，与 `AdminRole` 无关。
- 账号 → 管理员角色有两条并存表示：`UserAdminRole`（user_id → admin_role_id，**权威关联表**，admin_service/system_service 写入）与 `UserRole.AdminRole` 字符串标记（`super_admin`，cmd/setadmin 与 system_service 写入，用于列表展示）。
- 权限码存 `admin_permissions` 表（目录 16 模块 33 码，由 role_catalog.go 播种，含 `*` 行）；角色-码关系存 `admin_role_permissions`。
- 现网数据（kg 4）：uid 4 与 990701001 **无 UserAdminRole 行**、标记 `super_admin`；uid 58 关联「园所管理员」（is_system，**权限为空**）；uid 59 关联「教学主任」（4 码）。
- admin 端 `permission.ts` 是命名中间件但**无任何页面引用**（零 `definePageMeta`），`stores/permission.ts` 的 `setCodes` 零调用，常量 `['*']`。
- profile 响应现含 `roles[].admin_role`（实现已有，契约未记载），无 permissions。

## Goals / Non-Goals

**Goals**

- 服务端按「路由 → 权限码」登记表强制管理端接口，登记缺口在启动时暴露、运行期一律拒绝。
- 权限解析即时生效（角色改动无需重新登录）。
- 客户端菜单/守卫与服务端使用同一套权限码语义。
- 迁移后既有管理员（super_admin、园所管理员名下）访问能力不变。

**Non-Goals**

- 不动 teacher/、parent/ 端与 Casbin 策略文件（粗闸保留）。
- 不新增权限码（health/class/leave 见 proposal 阻塞项）。
- 不做按钮级细粒度、不做权限缓存层。

## 跨端触点清单

| 层 | 会改 / 不改 | 说明 |
| --- | --- | --- |
| Go domain | **不改** | `UserAdminRole`、`AdminRolePermission`、`AdminPermission` 现有结构够用，无新表新字段 |
| Go repository | **不改** | 复用 `FindUserAdminRole`、`ListPermissionCodesForRole`（admin_repo.go 已有） |
| Go service | **会改** | 新增权限解析服务：给定 (kgID, uid) 返回 (codes, wildcard, err)，含 super_admin 兜底与空集语义 |
| Go handler | **会改** | profile 响应 `roles[]` 增加 `permissions`（auth_service.GetProfile + RoleInfo） |
| Go middleware | **会改** | 新增 `RequirePermission` 中间件 + 路由→码登记表（新文件） |
| Casbin policy | **不改** | `admin` 通配保留为粗闸；细闸在 Go 中间件。`rbac_policy.csv` 无新增路由，无需同步 |
| router.go | **会改** | adminG 挂 `RequirePermission`；注册完成后做登记表覆盖自检（缺口即启动失败） |
| backend/api 契约 | **会改（契约变更）** | `api-common.md`：profile 响应的 roles 条目新增 `permissions`；补记 `admin_role`。字段级溯源见下 |
| 端页面（admin） | **会改** | `stores/auth.ts`（fetchProfile 后 setCodes）、`stores/permission.ts`（去掉占位默认语义）、`middleware/permission.ts` → `permission.global.ts`（从菜单表派生，改名全局）、新增 `pages/no-permission.vue`、`pages/roles/index.vue`（移除未生效警告） |
| 端页面（teacher/parent） | **不改** | 本次不涉及两端；两端无 AdminRole 概念，接口与页面均不动 |
| design token | **不改** | 无新增视觉样式；新增页面/提示全部复用既有 token 类 |
| 数据迁移 | **会改** | `admin_role_permissions` 回填「园所管理员」的 `*` 行；`docs/migrations/` 新增 SQL，`go run ./cmd/dbcheck admin_role_permissions` 核实 |

tabBar 核对：本变更仅涉及 admin（Nuxt web），无小程序页面、无 `navigateTo`/`switchTab` 场景，不适用 tabBar 差异规则。

AutoMigrate 局限：本次**无新表/新字段**，不依赖 AutoMigrate；唯一数据变更是关系表回填，必须走手工 SQL（见迁移方案）。

## Decisions

### D1 强制机制：静态登记表 + 单中间件 + 启动自检

`RequirePermission` 挂在 `adminG`（在 AuthJWT/Casbin 之后），对每个请求以 `"<METHOD> <FullPath>"`（如 `GET /api/v1/admin/students/:id`）查登记表，命中则比对解析出的权限码（或在表缺项时拒绝并 slog.Error）。

- 备选 A：把权限码写进 Casbin 策略——否决：Casbin 模型是 (sub, obj, act) 路径导向，码→路径的映射表仍要另建，且角色改动需重载策略，复杂度更高。
- 备选 B：~90 处路由行内挂中间件——否决：散落难审、无法机械检测缺口。
- 备选 C：handler 内逐个人工校验——否决：100+ handler 侵入且漏一步就静默开口。

**登记表是数据、可审、可测**：启动自检用 `engine.Routes()` 过滤 `/api/v1/admin` 前缀逐条比对登记表，缺口 → `NewRouter` 返回错误，启动失败。这样 fail-closed 不会在运行期制造惊喜。

### D2 解析时机：逐请求解析，无缓存

每次请求经解析服务查库（`UserAdminRole` → `ListPermissionCodesForRole`，均为索引查询）。否决 JWT 内嵌（角色编辑需重登才生效，违背「可核实」即时性，且 token 膨胀）与 Redis 缓存（失效面复杂，园所规模不值当）。规模评估：单园所管理员数 ≤ 数十，QPS 极低，两条索引查询可忽略。

### D3 超管判定与兜底顺序

解析顺序固定：① `UserAdminRole` 有行 → 取该角色码集（集内含 `*` 亦为通配）；② 无行且 `UserRole.AdminRole == "super_admin"` → 通配；③ 否则空集。该顺序同时覆盖现网 uid 4 / 990701001（无行、标记超管）与 uid 58（有行、系统角色，见 D4）。

### D4 迁移：系统角色回填 `*` 而非逐码枚举

「园所管理员」是 is_system 的「系统自动创建」角色（描述原文），其 0 码是播种缺口而非产品意图。回填目录中已有的 `*` 行（而非枚举 33 码）：复现强制前等价访问、diff 最小、园长可在角色页一眼改回显式码集。已核实目录中存在 `*`（role_service 的 `permissionIDsForPayload` 已专门处理 `*`）。

### D5 客户端守卫与菜单同源

`permission.ts` 升级为 `permission.global.ts`（当前是无人引用的命名中间件，等于死代码），判定所需权限码改为 `matchMenuByPath(to.path)?.permission`——与侧栏过滤共用 `constants/menus.ts` 同一张表，从此菜单可见与路由可达不可能分叉。否决为 23 个页面补 `definePageMeta`（重复第二来源，制造漂移面）。

启动顺序：`auth.global.ts`（有 cookie）→ `permission.global.ts`（若 codes 未加载，先 `await auth.fetchProfile()` 一并 setCodes）→ 判定。越权直达 → 重定向到按菜单表顺序找到的第一个可访问项；一个都没有 → `/no-permission`（新增极简页，复用既有 token 类）。

### D6 契约字段与溯源

`GET /common/user/profile` 的 `roles[]` 条目新增 `permissions: string[]`：

- **溯源**：design-data/admin-09-角色权限页.json 配置的权限，其消费端就是管理端的菜单可见性（23 项侧栏）与页面可达性；「登录后知道自己能看什么」是该设计的前置数据，无此字段则配置无消费方。
- **不可派生**：`roles[].admin_role` 是角色显示名，权限集合存于服务端角色-码多对多关系，客户端无法推导。
- 通配账号返回 `["*"]`，与既有客户端 `hasPermission` 的 `*` 语义一致。
- 同时补记实现已存在、契约漏记的 `admin_role` 字段（纯文档对齐，非新字段）。

### D7 路由 → 权限码登记表（登记表的权威定义）

约定：查看类用查看码；写操作用对应 manage/audit/export 码；无独立码的资源组沿用其查看码（在表内标注）。

| 资源组 | 路由（METHOD 后缀） | 权限码 |
| --- | --- | --- |
| dashboard | GET `/dashboard/*`（overview、attendance-trend、class-ranking、growth-stats 共 4 条） | dashboard:view |
| classes | GET `/classes`、`/classes/:id` | student:view |
| classes | POST `/classes`、POST `/classes/:id/adjust`、PUT `/classes/:id`、DELETE `/classes/:id` | student:manage |
| students | GET `/students`、`/students/:id` | student:view |
| students | POST `/students`、PUT `/students/:id`、PUT `/students/:id/status`、DELETE `/students/:id` | student:manage |
| students | POST `/students/import` | student:import |
| students | GET `/students/export` | student:export |
| teachers | GET `/teachers`、`/teachers/:id`、`/teachers/:id/invitation`、`/teachers/:id/invitation/qrcode` | teacher:view |
| teachers | POST `/teachers`、POST `/teachers/:id/invitation`、PUT `/teachers/:id`、PUT `/teachers/:id/status`、DELETE `/teachers/:id` | teacher:manage |
| parents | GET `/parents`、`/parents/:id` | parent:view |
| parents | POST `/parents`、PUT `/parents/:id`、PUT `/parents/:id/status` | parent:manage |
| settings | GET/PUT `/settings`、GET/POST `/settings/academic-years` | system:kindergarten |
| kindergarten | GET/PUT `/kindergarten/info` | system:kindergarten |
| admins | GET `/admins` | admin:view |
| admins | POST `/admins`、PUT `/admins/:id`、PUT `/admins/:id/reset-password`、DELETE `/admins/:id` | admin:manage |
| roles | GET/POST `/roles`、PUT/DELETE `/roles/:id` | admin:role |
| permissions | GET `/permissions` | admin:role |
| attendance | GET `/attendance/overview`、`/attendance/records` | attendance:view |
| attendance | GET `/attendance/report` | attendance:export |
| leaves | GET `/leaves`、`/leaves/stats`、PUT `/leaves/:id/audit` | attendance:view（请假无独立码，与出勤共码——已知耦合，proposal 阻塞项） |
| pickup | GET `/pickup/overview`、`/records`、`/abnormal`、`/authorized-persons`、`/overdue`、`/overdue/count` | pickup:view |
| pickup | PUT `/pickup/authorized-persons/:id/audit`、POST `/pickup/overdue/notify` | pickup:manage |
| growth | GET `/growth/stats`、`/growth/logs`、`/growth/logs/:id` | growth:view |
| growth | DELETE `/growth/logs/:id` | growth:manage |
| growth | PUT `/growth/logs/:id/audit` | growth:audit |
| tasks | GET `/tasks`、`/tasks/stats`、`/tasks/:id` | task:view |
| tasks | POST `/tasks`、PUT `/tasks/:id`、DELETE `/tasks/:id` | task:manage |
| points | GET `/points/overview`、`/records`、`/rules` | points:view |
| points | POST `/points/adjust`、PUT `/points/rules` | points:manage |
| achievements | GET `/achievements` | achievement:view |
| achievements | POST `/achievements`、PUT `/achievements/:id`、DELETE `/achievements/:id` | achievement:manage |
| notices | GET `/notices`、`/notices/:id` | notice:view |
| notices | POST `/notices`、PUT `/notices/:id`、DELETE `/notices/:id` | notice:manage |
| notices | PUT `/notices/:id/audit` | notice:audit |
| announcements | GET `/announcements`、`/announcements/:id` | announcement:view |
| announcements | POST `/announcements`、PUT `/announcements/:id`、PUT `/announcements/:id/top`、DELETE `/announcements/:id` | announcement:manage |
| audit | GET `/audit/list`、`/audit/stats` | audit:view |
| audit | PUT `/audit/:id` | audit:manage |
| export | POST `/export/attendance` | attendance:export |
| export | POST `/export/students` | student:export |
| export | POST `/export/growth` | growth:view（目录无 growth:export；以「可查看即有资格导出」为门槛，标注待产品决策） |
| export | GET `/export/:taskId`、GET `/export/download/:taskId` | attendance:export（共享端点按导出功能总码，近似处理） |
| logs | GET `/logs` | system:log |
| health | GET `/health/stats` | dashboard:view（无 health 码；proposal 阻塞项） |
| dicts | GET/POST `/dicts/types`、PUT/DELETE `/dicts/types/:id`、GET/POST `/dicts/types/:id/items`、PUT/DELETE `/dicts/items/:id`、PUT `/dicts/items/sort`（9 条） | system:kindergarten |

（表内为权威定义；实现时以一条一条 `"METHOD /api/v1/admin/..."` 字面量落表，启动自检保证与路由一一对应。）

## 时序（强制链路）

```
请求 ──> AuthJWT ──> CasbinRBAC(粗闸 admin 通配) ──> RequirePermission(细闸)
                                                     │
             ┌───────────────────────────────────────┘
             ▼
  查登记表 "METHOD FullPath"
     ├─ 缺项 ──────────────> 403 + slog.Error（运行期缺口）
     └─ 命中 code
          ▼
  解析权限码：UserAdminRole ──有行──> 角色码集(含* 即通配)
                │无行
                ▼
        AdminRole=="super_admin" ──是──> 通配放行
                │否
                ▼
        空集 ──> 403
```

## Migration Plan

1. 上线前：备份 `admin_role_permissions`（导出 SQL 快照）。
2. 执行回填 SQL（`docs/migrations/`）：为系统角色「园所管理员」（按 `kindergarten_id` + `is_system` + 角色名定位，不写死 id）插入其对应 `admin_permissions.code='*'` 的关联行；`INSERT ... SELECT` 带唯一键冲突保护（`idx_role_perm`）。
3. `go run ./cmd/dbcheck admin_role_permissions` 核实行数与索引。
4. 发布顺序：先发后端（强制生效）→ 再发 admin 前端（菜单/守卫）。中间窗口内前端仍全量显示菜单，但接口层 403 生效——可接受（窗口期分钟级）；回滚策略：前端回滚无副作用；后端回滚即恢复通配粗闸，权限即时回到「宽松」，无数据破坏。
5. 观测：上线后 24h 关注 `403` 环比与 `slog.Error` 中的登记缺口（应为零）。

## Risks / Trade-offs

- [教学主任等非系统角色访问被收紧 → 运营感觉「坏了」] → 预期行为（假锁变真锁）；pre-launch 在回归清单与验收脚本中显式覆盖该角色，出问题时的第一处置是去角色页补码而非回滚强制。
- [登记表漏登记新路由 → fail-closed 直接 403] → 启动自检 + 独立 TC（TC-NEW-PERM-0007）把缺口提前到启动期；运行期缺项亦有 slog.Error 留痕。
- [逐请求解析增加两次索引查询] → 规模评估可忽略；若未来管理员数量级增长，可在解析服务内加缓存（接口已按可替换设计）。
- [`/export` 共享端点按总码近似] → 已标注；若产品要求细分，等权限码补全时一并处理（阻塞项的一部分）。
- [uid 4 类账号依赖字符串标记判定] → 该标记由 setadmin/system_service 写入且被列表页消费，短期内是唯一线索；在解析服务注释中记录，未来应把超管也落 `UserAdminRole`。

## Open Questions

- 无。health/class/leave 码属产品决策，已按规则写入 proposal 阻塞清单，不在本设计内解决。
