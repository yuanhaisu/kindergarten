# Design

## Context

动机见 `proposal.md - Why`。行为契约见 `specs/pickup-custody/spec.md`。此处只写实现所需的现状约束。

现状关键事实（均已实测核过）：

- 接送逻辑分散在 3 个 service：`pickup_service.go`（管理端）、`teacher_portal_service.go`（老师端）、`parent_portal_service.go`（家长端），**无统一 pickup domain service**
- `PickupRecord.PickupPersonID` 与 `isAuthorized` 在 `teacher_portal_service.go:978-980` 声明后**从未赋值**
- 扫码链路三段对不上：前端 `teacher/src/pages/pickup/index.vue:63` 只传 `qr_code`，handler `teacher_portal.go:609` 直接 `ShouldBindJSON` 绑定，service `:913` 第一步 `if req.StudentID == 0` 就拦死
- QR token 形如 `PKUP_{studentID}_{yyyyMMdd}_{hex}`，**内嵌 studentID 可枚举**
- 防重规则 `WHERE date = ? AND type = ?`（`:935`）阻断同日多次往返
- `PickupAuthorization` / `PickupQRCode` **无 `kindergarten_id` 列**（E1，本轮不修，见 Non-Goals）
- 通知管道现成：`Message` 域已有 `MessageTypePickup`，`WechatPushLog` 也在；但接送事件**一处都没有写入 Message**
- 项目**无任何定时/调度基础设施**（`cmd/api/main.go` 是纯 gin HTTP server）
- 服务端用 Casbin RBAC 但 `rbac_policy.csv` **无 pickup 专用策略**，靠角色通配；行级隔离在 service/repo 的 `kgID/classID/studentID` 条件里

## Goals / Non-Goals

**Goals:**

- 扫码接走「一条请求只带凭证内容」即可完成，后端从凭证自解释学生与接走人
- 接走登记在两条路径（扫码 / 手动）上都能归因到具体的人，例外可统计
- 同日多次往返可登记，且重复接走 / 提前接走强制留理由
- 滞留可见 + 可一键通知家长 + 计入统计
- 契约与实现重新对齐

**Non-Goals（设计层面，补充 proposal 的 out-of-scope）：**

- 不做统一 pickup domain service 的重构——本次仍在现有三个 service 内落改动，避免借机改架构
- 不引入调度器、消息队列、领域事件总线；通知用同步 insert `Message` 完成
- 不改 `PickupAuthorization` / `PickupQRCode` 的租户列（E1）；新增表则**自带** `kindergarten_id`
- 不做二维码签名、并发锁、幂等键

## Decisions

### D1. 凭证改为「随机串 + 表内绑定」，后端自解释

**决定**：`pickup_qr_codes` 的 `code` 改为纯随机串（如 32 位十六进制），新增 `authorization_id` 列；扫码请求**只带 `qr_code`**，后端按码查出 `student_id` + `authorization_id`，完成资格校验后落 `PickupPersonID`。

**为什么**：一次解决三件事——(a) token 不再泄露可枚举 studentID（E4）；(b) 扫码断链消失（前端不必再传 student_id）；(c) 接走人绑定落到可外键关联的列上（E2）。

**否决项**：token 内嵌 `authorization_id`（`PKUP_{sid}_{aid}_...`）——仍泄露可枚举 ID，且无法做到「改指定后旧码立即失效」的强一致。

### D2. 「今日接走人」建独立指定记录，不与码混为一谈

**决定**：新增 `pickup_daily_designations` 表（`kindergarten_id` + `student_id` + `authorization_id` + `date`，`UNIQUE(student_id, date)`）。家长指定即写/覆盖该行；生成凭证时从该行取 `authorization_id` 落到码上。重新指定 → 覆盖该行 **且** 失效该生当日全部旧码。

**为什么**：码是一次性易失的（用完即焚），若「今日指定」只活在码上，焚掉后「原计划谁来接」无处可查，而「计划接走人 vs 实际接走人」正是审计要回答的问题（I3）。指定记录让计划与实际可比对。

**否决项**：不建独立表，「指定 = 生成码」——省一张表，但码焚掉后指定状态丢失，家长要反复重选，且计划侧不可回查。新增表对 AutoMigrate 友好（建表属「只增」，不受「只增不改」局限）。

### D3. 学生当日状态改为「看最后一次转移」

**决定**：`PickupStudents` / `PickupOverview` 的状态派生由「当日是否存在某类记录」改为「当日最后一次转移的方向」。防重由 `WHERE date=? AND type=?` 改为「不允许连续同向」：插入前查当日最后一记录，同向则拒。

**为什么**：支持同日多次往返（就医返园），且状态语义与物理事实一致（孩子在不在园）。

**否决项**：加 `status` 列冗余维护当前状态——引入状态一致性问题，且推导即可得，无必要。

### D4. 班级放学时刻存 `Class` 新列，应用层兜底 16:00

**决定**：`Class` 增 `pickup_deadline`（time 类型，可空）。判定提前接走时，空/零值回退到 16:00。滞留判定共用同一时刻。

**为什么**：产品决策是班级可配（2026-10-07）。应用层兜底保证既有班级无需回填即得合理行为，也让默认值只写在一处。

**迁移**：加列属 AutoMigrate 支持范围（只增不改）。完成后 `go run ./cmd/dbcheck classes` 核实列存在。

**否决项**：存 `kindergarten_app_setting` —— 粒度是园所，与「班级可配」决策不符；存 dict 字典 —— 无班级维度。

### D5. 理由校验在 service 层集中判定，不在 handler 分散

**决定**：在接走登记路径上做一次统一的「是否需要理由」判定，返回是否必填及其触发原因（重复 / 提前）。handler 只透传校验结果。

**为什么**：两个触发条件（重复、提前）会各自增长，散在 handler 里必然漏改。集中判定也便于单测覆盖。

### D6. 通知用同步 `Message` 插入，不引入事件总线

**决定**：接走成功、一键通知两处，直接 `INSERT messages`（`type = 'pickup'`）。`NotifyOnPickup` 由死配置转为该插入的开关。

**为什么**：通知管道已现成，缺的只是触发。同步插入足以满足「谁接走了孩子」的即时告知；引入事件总线是本轮不需要的架构负担。

**风险**：通知失败是否回滚接走登记？见 Risks。

### D7. 滞留「放学时段」由后端给截止时刻与超时标志，前端只负责高亮

**决定**：接送列表响应对每人给出「是否已超时未接」；老师端/管理端据此高亮。一键通知为独立端点。

**为什么**：老师端与管理端共用一套判定口径，避免两端各自算一遍漂移。前端不做时间判断，也便于日后改口径只动后端。

### D8. 手动登记的例外走显式标记，不走「空 authorization_id」

**决定**：手动登记请求带「是否授权例外」与「来接者姓名」。列表选人 → 落 `authorization_id` 且例外标志为否；标例外 → `authorization_id` 空 **且** 例外标志为是 + 姓名。

**为什么**：仅靠 `authorization_id IS NULL` 无法区分「老师漏选」与「有意例外」。显式标志让例外可观测，也满足「紧急不阻断」。

## 跨端触点清单（强制）

| 层 | 会改 / 不改 | 改什么 / 为何不改 |
|---|---|---|
| **Go `domain`** | **会改** | `pickup.go`：`PickupQRCode` 加 `AuthorizationID`；`PickupRecord` 语义注释更新（`PickupPersonID` 真正落值、`OperatorID` 语义明确为核验人）。**新增** `pickup_daily_designations` struct（含 `kindergarten_id`）。`class.go`：`Class` 加 `PickupDeadline` |
| **Go `repository`** | **会改** | `pickup_repo.go`：指定记录 CRUD（含按 `(student_id, date)` upsert）；按 `code` 联查 `authorization`；防重查询改为「查当日最后一条」；状态派生改为「最后一条方向」；滞留/超时计数查询；失效该生当日全部码。`class` 侧补按 ID 取放学时刻 |
| **Go `service`** | **会改** | `teacher_portal_service.go`：`CreatePickupRecord` 重写校验链（从码解学生与接走人、授权二次校验、连续同向校验、理由必填判定、落 `PickupPersonID`/例外标志、写 Message）；`PickupStudents`/`PickupOverview` 状态派生改口径 + 超时标志。`parent_portal_service.go`：指定今日接走人、生成码改绑人、`GetQRCode` 响应补接走人信息、接走通知按 `NotifyOnPickup` 写 Message、`ListPickupRecords` 响应补 `is_abnormal`（修既有漂移，见契约字段清单）。`pickup_service.go`（管理端）：滞留名单、超时未接计数 |
| **Go `handler`** | **会改** | `teacher_portal.go`：扫码请求只收凭证内容；手动登记请求加例外标记与理由约束；一键通知端点；`records/:id` 响应包装与契约对齐。家长侧：指定端点、二维码响应补接走人。管理侧：滞留名单与计数端点 |
| **Casbin `rbac_policy.csv`** | **会改** | 新增路由（家长指定、老师一键通知、管理滞留名单/计数）需登记。现有 pickup 无专用策略、走角色通配 `/api/v1/teacher/*` 等，故多数新路由被通配覆盖；**仍需核对**新路由是否落在通配前缀内，落不进则补策略 |
| **`router.go`** | **会改** | 注册家长「指定今日接走人」、老师「一键通知家长」、管理「滞留名单 / 超时未接计数」；接走登记路由语义不变但入参变 |
| **`backend/api/` 契约** | **会改（契约变更）** | 见下方字段级清单 |
| **老师端 `teacher/src/pages/pickup/index.vue`** | **会改** | 扫码只传凭证；手动登记加「选授权人 / 标例外」二选一与理由输入；滞留名单高亮；一键通知按钮 |
| **老师端 `teacher/src/pages/pickup/records.vue`** | **会改** | 显示理由与例外标记；重复/提前接走的失败提示 |
| **老师端 `teacher/src/api/teacher-pickup.js`** | **会改** | 扫码入参收敛为凭证；新增一键通知函数；手动登记入参加例外与理由 |
| **家长端 `parent/src/pages/pickup/authorized.vue`** | **会改** | 增「设为今日接走人」操作与当前指定展示 |
| **家长端 `parent/src/pages/pickup/qrcode.vue`** | **会改** | 呈现孩子与接走人姓名/照片；真实二维码图（现状是文本占位框） |
| **家长端 `parent/src/pages/pickup/index.vue`** | **不改** | 仅列表，本次行为不变 |
| **家长端 `parent/src/api/parent-pickup.js`** | **会改** | 新增指定端点；二维码响应字段按契约对齐（注意 `student_id` 必须放 query 的既有约定） |
| **管理端 `admin/pages/pickup/index.vue`** | **会改** | 滞留名单与超时未接计数展示；核验人列名对齐 |
| **管理端 `admin/api/pickup.ts`** | **会改** | 新增滞留/计数接口；补 `getAuthorizedPersons` 等既有函数的契约对齐 |
| **design token** | **不改** | 滞留高亮、例外标记复用既有语义色（`$color-semantic-warning` / `danger` 系）。若实现中发现需新视觉语义，按 token 流程新增 `tokens.json` 并重建，**不得**在页面硬编码 |
| **`docs/tests/TEST_CATALOG.md`** | **会改** | 补录 `TC-NEW-PKU-*`；修订与新行为冲突的 `TC-TCH-PKUT-014` / `019` / `020` |
| **跳转（tabBar）** | **不改** | 本次**不新增任何页面跳转**，故不涉及 `uni.navigateTo` / `uni.switchTab` 选型。家长端 pickup 是 tabBar 页（`pages/pickup/index`），子页用 `navigateTo` 是既有做法，维持 |

### 契约字段级清单（`backend/api/`）

**溯源说明**：`design-data/` **无**家长端接送/二维码/授权人设计稿，故下列字段按「明确业务场景」溯源（规则允许的第二来源）。管理端字段可溯源到 `design-data/admin-05-接送记录页.json`。

| 端点 | 变更 | 字段 | 溯源 |
|---|---|---|---|
| `POST /parent/pickup/designate`（新增） | 新增 | 请求：`student_id`(query，沿用既有约定)、`authorization_id`。响应：`student_id`、`authorization_id`、`person:{name,relation,avatar}`、`designated_at` | 业务场景：今日接走人指定（产品决策 2026-10-07）。响应回显 `person`/`designated_at` 是为家长页在指定成功后直接展示「今日接走人是谁」，避免再拉一次列表 |
| `GET /parent/pickup/qrcode` | 改（契约变更） | 响应 `qr_code` / `expire_at`（已有）；**新增** `person:{name, photo}`；`student` 补 `photo` | 业务场景：A2「码上带名字」人证对照（产品决策 2026-10-07） |
| `GET /parent/pickup/records` | 改（契约对齐） | 响应补 `is_abnormal`（`PickupRecord.IsAbnormal` 存在于 domain，契约已列但实现未返回） | 业务场景：家长应能看到某次接送被园所标记异常。设计稿 `admin-05` 明细有「状态」列（正常/异常），家长侧同语义。**修复方向：改实现补返回，不是删契约** |
| `POST /teacher/pickup/scan` | 改（契约变更） | 请求：**仅** `qr_code`（`student_id` 从契约移除，改由后端自解释）；可选 `remark` | 业务场景：D1 后端自解释，修复扫码断链 |
| `POST /teacher/pickup/scan` | 改 | 响应：**新增** `person:{name, photo}`；`student` 补 `photo`；`is_authorized_override` | 业务场景：老师侧人证对照。`is_authorized_override` 扫码路径恒为 `false`（扫码必经授权校验），补齐是为与手动登记响应形状一致，前端不必分支判空 |
| `POST /teacher/pickup/manual` | 改（契约变更） | 请求：**新增** `authorization_id`（可空）、`is_authorized_override`(bool)、`override_name`；`remark` 约束收紧 | 业务场景：例外通道（产品决策 2026-10-07）+ 重复/提前理由必填 |
| `POST /teacher/pickup/notify-overdue`（新增） | 新增 | 请求：`student_id`。响应：`student_id`、`notified`、`message_id`、`notified_at` | 业务场景：滞留一键通知（档 B，产品决策 2026-10-07）。`message_id`/`notified_at` 供前端提示与留痕 |
| `GET /teacher/pickup/students` | 改（契约变更） | 响应每人：**新增** `is_overdue`、`pickup_deadline`、`last_remark`、`is_authorized_override` | 业务场景：滞留高亮 + 理由/例外可见。接口粒度：仅列表项渲染所需，**不**塞详情字段 |
| `GET /teacher/pickup/overview` | 改 | 响应**新增** `overdue_not_picked`（超时未接计数，按学生计） | 设计稿 `admin-05`「本月超时未接」统计卡；业务场景：超时计数 |
| `GET /admin/pickup/overdue`（新增） | 新增 | 响应：`overdue_total`、`by_class[]`（含 `pickup_deadline`/`overdue_count`/`students[]`/`overdue_minutes`） | 设计稿 `admin-05`「本月超时未接」；业务场景：按班看滞留 |
| `GET /admin/pickup/authorized-persons` | 改（补文档） | 既有端点补进 `api-admin/08-pickup.md`；响应列名与 `admin-05` 的「接送人」对齐 | 业务场景：契约与实现一致（缺陷修复） |
| `PUT /teacher/pickup/authorized-persons/:id/approve\|reject` | 改（补文档） | 既有端点补进 `api-teacher/07-pickup.md`；**同时决定是否让老师端露出**（现前端未调用） | 业务场景：契约与实现一致 |
| `GET /parent/pickup/authorized-persons` | 改（字段对齐） | 文档 `qr_image_url`/`id_card_tail`/`status_label` → 实现 `qr_code`/脱敏 phone/`review_status_label`，**以实现为准改文档**（实现更保守，不暴露证件号） | 业务场景：契约与实现一致 |
| `GET /teacher/pickup/records/:id` | 改（包装修复） | 文档扁平对象 → 实现 `{"record": {...}}`，**以实现为准改文档** | 业务场景：契约与实现一致 |

**接口粒度约束**：`/students` 列表只放列表渲染所需；授权人完整证件、授权历史、Message 明细一律不进列表响应，详情走 `records/:id` 或专门端点。

## ASCII 时序图：扫码接走（新链路）

```
家长端                    后端                         老师端
------                    -----                        ------
authorized.vue
  [设为今日接走人]
      │
      ├─► POST /parent/pickup/designate
      │        │  校验 authorization 在有效集合内
      │        │  upsert pickup_daily_designations
      │        │  失效该生当日全部旧码
      │        └─► ok
      │
  qrcode.vue
      ├─► GET /parent/pickup/qrcode
      │        │  读指定记录 → authorization_id
      │        │  再校验授权仍有效
      │        │  建码: code(随机) + student_id + authorization_id
      │        └─► { qr_code, student:{name,photo}, person:{name,photo} }
      │
      ▼
  [码图：小明 · 奶奶张三]                uni.scanCode
                                          │
                            POST /teacher/pickup/scan {qr_code}
                                          │
                                          ▼
                                         按 code 查 → student + authorization
                                          │ 校验: 有效/未过期/未用
                                          │ 校验: auth.status=1 AND review_status=1
                                          │ 校验: 不连续同向
                                          │ 校验: 理由必填规则(重复/提前)
                                          │
                                          ▼
                                         落 PickupPersonID = auth.id
                                         码置 is_active=false
                                         若 NotifyOnPickup → INSERT messages
                                          │
                                          └─► {student:{name,photo}, person:{name,photo}}
                                                │
                                                ▼
                                          [小明照][张三照]
                                           小明 · 奶奶张三
                                           [确认登记]  ← 肉眼比对
```

## ASCII 状态图：当日托管序列（B1）

```
  ┌──────────────┐   送入    ┌──────────────┐   接走    ┌──────────────┐
  │  with_family │ ────────► │  in_school   │ ────────► │  with_family │
  └──────────────┘           └──────────────┘           └──────────────┘
         ▲                                                    │
         │                        送入                        │
         │              （返园，允许）                         │
         │                        ▼                           │
         │                ┌──────────────┐   接走            │
         └────────────────│  in_school   │◄──────────────────┘
                          └──────────────┘
                                    │
                                    │  超过班级放学时刻仍未接走
                                    ▼
                          ┌──────────────────┐
                          │  overdue_visible  │  ← 名单高亮 + 可一键通知
                          │  （仍属 in_school）│
                          └──────────────────┘

  非法转移（拒绝登记）：
    in_school  ──送入──►  ✗  连续同向
    with_family ──接走──► ✗  连续同向
```

## Risks / Trade-offs

- **[通知失败影响主流程]** → 接走登记是安全关键动作，**不得**因消息写入失败而回滚。通知插入失败仅记日志，登记照常成功。
- **[重新指定后旧码失效可能让家长意外]** → 前端在「换人」时明确提示「将使今日已生成的凭证失效」。
- **[理由必填在紧急场景碍事]** → 提前接走的理由是自由文本，不设枚举、不阻断；真正紧急且来接者不在名单时走例外通道（D8），两条路都不死锁。
- **[班级放学时刻未配置导致判定不一致]** → 应用层统一兜底 16:00（D4），默认值只在一处；`dbcheck classes` 核实列落地。
- **[状态派生改口径影响既有统计]** → `PickupOverview` 的 arrived/picked 口径随 B1 变化，需在契约与前端同步说明；管理端既有「接送记录」列表口径不变。
- **[例外通道被滥用]** → `is_authorized_override` 可统计，但本轮**不**做告警阈值。留待「非授权接送尝试」下轮一并做。
- **[E1 租户列缺失被本次新表对照放大]** → 新表自带 `kindergarten_id`，不复制旧缺陷；旧表补列留待下轮，proposal 已声明。
- **[老师审核端点是否露出]** → 路由已注册但前端未调用。本轮只补契约文档，**不**改前端入口，避免与管理端审核双入口职责不清。

## Migration Plan

1. 后端：`domain` 加列/加表 → `go build ./...` → `go run ./cmd/dbcheck classes` 与 `go run ./cmd/dbcheck pickup_qr_codes` 核实列落地；`go run ./cmd/dbcheck pickup_daily_designations` 核实新表
2. 契约：先改 `backend/api/` 三端文档，再改实现（契约先行，符合本仓库「改接口先改契约」）
3. 端页面：家长端 → 老师端 → 管理端，各自 `npm run dev:mp-weixin` / `npm run build`
4. 回滚：新列/新表均为增量，回滚仅需回退代码；不需数据回迁。若已产生指定记录与新码，回退后旧逻辑忽略新列即可（旧行为对 `authorization_id` 为空的码会拒绝扫码，属预期）

## Open Questions

（无。凡会改变 specs、实现路径或任务拆分的问题已在探索阶段解决；剩余不确定性均为可逆的实现细节。）
