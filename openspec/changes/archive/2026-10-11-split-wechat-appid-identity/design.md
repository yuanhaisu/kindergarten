# Design

## Context

现状（动机见 proposal.md — Why）：

- `server/internal/config/config.go:47-50` `WeChatConfig` 只有一对 `AppID`/`Secret`；`overrideFromEnv` 只认 `WECHAT_APP_ID` / `WECHAT_SECRET`。
- `server/internal/app/provider.go:123-125` `ProvideWeChatClient` 造**一个** `*auth.WeChatClient`，经 `service_provider.go:20` 注入 `AuthService`。
- `server/internal/auth/wechat.go:62` `Code2Session` 用该单例的凭据拼 URL。
- `server/internal/service/auth_service.go:260` `WeChatLogin(ctx, wxCode, role, inviteCode)` —— **`role` 是形参，函数体内未被使用**，凭据选取与它无关。这是串号的结构性根源。
- `auth_service.go:270` 用 `FindByOpenID(ctx, sess.OpenID)` 命中 User，`user_repo.go` 用 `First()`。
- `domain/user.go:5-9` 已有 `OpenID` / `UnionID` / `Phone`（`Phone` 有索引），无需新列。
- `router.go:27` 路由 `POST /auth/login/wechat` 已注册；`handler/http/auth.go:76` 已绑定含 `role` 的请求体。

约束（来自项目 context）：登录两端各自独立实现（非共享包）；Wire 改 DI 必须 `make wire`；`AutoMigrate` 只增不改；MCP 导航不能替代真实交互验证。

## Goals / Non-Goals

**Goals:**

- 登录凭据按 `role` 选取，teacher / parent 各用各的 AppID/Secret。
- 同一自然人两端登录落两条 User，互不命中。
- 跨端同人可判定：unionid 优先、手机号兜底。
- 老师端切换 AppID 后有明确的重绑路径（复用现有邀请码流程）。
- 凭据缺失/换取会话失败必须显式报错，不静默降级。

**Non-Goals:**

- 不改 `backend/api/` 接口形状（见 proposal — Out of Scope）。
- 不改任何业务查询的可见性/权限行为（关联只做判定，不改业务）。
- 不做 `users` 表结构变更、不做 SQL 迁移。
- 不实现「老师兼家长」的跨端业务消费（消息互通、数据可见等）。
- 不做微信开放平台账号注册与小程序绑定（人工运维）。
- 老师端 `webview` 页面不验（个人主体不支持 `<web-view>`）。

## Decisions

### D1 — 凭据选取：单 client 持双凭据，全部四个方法按端取

`WeChatConfig` 扩为按端双对（`TeacherAppID`/`TeacherSecret`/`ParentAppID`/`ParentSecret`）。`WeChatClient` 内部持有两套凭据，**四个公开方法全部按端选取**，不止 `Code2Session`：

| 方法 | 调用方 | 凭据 |
|---|---|---|
| `Code2Session` | `auth_service.go:261` `WeChatLogin` | 按登录 `role` 选 |
| `GetPhoneNumber` | `auth_service.go:541` `BindParentPhone` | **家长端**（getPhoneNumber 仅家长端用） |
| `GetWXACodeUnlimit` | `teacher_service.go:519` 教师邀请码二维码 | **老师端**（码要指向老师端小程序） |
| `GetAccessToken` | 上两者的内部依赖 | 跟随调用方；**缓存键必须带 appid** |

`GetAccessToken` 的 Redis 缓存键从全局单键 `wechat:access_token`（`wechat.go:91`）改为 `wechat:access_token:<appid>`。两个 AppID 共用一个键会互相污染 access_token，后续 `getPhoneNumber` / 二维码全部报微信 `40001 invalid credential`，且症状指向「凭据错」而非「缓存串了」，极难定位。

**为什么**：`AuthService` 现在持单个 `*auth.WeChatClient`（`service_provider.go:20`）。改成「两个 client 实例 + 按 role 分派」要动 Wire 的 provider 形状（`ProvideWeChatClient` 返回值无法再是单值），牵动 `wire.go` / `wire_gen.go` / `service_provider.go` 三处。改成「一个 client，内部按端查凭据」只动构造签名与各方法的凭据选取，DI 形状不变。

**为什么不止 `Code2Session`**：`GetWXACodeUnlimit` 生成的是教师邀请码二维码，而 task 6.2 的重绑恰恰要扫这个码。若它继续用家长端凭据，生成的就是**家长端**小程序码，扫码进错端，重绑主路径断掉、TC-NEW-ATH-0007 无法验收。`GetPhoneNumber` 同理——家长端绑定手机号换的是家长端 code，必须用家长端凭据换取。

**备选**（放弃）：
- 每端一个 `WeChatClient`，`AuthService` 持 map —— Wire 返回类型变复杂，且 `teacher_service` 也要各拿一份，收益只是把 if 挪个位置。
- 只改 `Code2Session`，其余挂账 —— 教师邀请码二维码生成错端、access_token 缓存互相污染，重绑验收必失败。
- 把 `role` 直接当配置键传进 URL —— 更易把 role 拼错导致用错凭据，且无法在配置缺失时提前报错。

### D2 — 身份落库：按端各一条 User，不做合并

`WeChatLogin` 沿用「`FindByOpenID` 命中则用、未命中则建」，但因为两端 openid 天然不同（openid 是「用户×小程序」二元的），同一自然人必然落两条。

**为什么**：这正是本次要的效果。合并（一张 User 多个 openid）需要给 `users` 加列或加关联表，而 `AutoMigrate` 只增不改、且 `UserRole` 的旧注释语义（「一个用户多角色如同时是老师和家长」）本来就被本次推翻——与其在旧语义上打补丁，不如承认跨端是两条记录。

**备选**（放弃）：`users` 加 `teacher_open_id` / `parent_open_id` 两列 —— 可以用 AutoMigrate 加列，但会让「一条记录跨端」的旧假设继续存在，且 `FindByOpenID` 的语义更绕。

### D3 — 跨端关联：派生查找，不新增列、不做关联表

判定「同人」= 按 `unionid`（非空则比）→ 否则按 `phone`（非空则比）去查另一端 User。**不写 `linked_user_id` 之类的新列**。

**为什么**：
- `users` 已有 `union_id`（`user.go:6`，带索引）与 `phone`（`user.go:9`，带索引），查得到，无需迁移。
- 关联是**判定**而非**账户合并**，不承担一致性维护成本。
- 幼儿园规模（数百级用户）下按索引反查足够。

**备选**（放弃）：`user_identities` 关联表 / `users.linked_user_id` —— 需手工 SQL（`AutoMigrate` 只增不改），且业务尚未消费关联，提前物化是过度设计。将来若「取同人另一端账号」成为热点，再物化为列或表，判定逻辑不变。

### D4 — 老师端 AppID 切换后的重绑：复用现有邀请码路径，不写迁移代码

`auth_service.go:288 weChatLoginWithInvite` 已实现「把本次换取的 openid 写进邀请指向的预创建教师账号」。切换 AppID 后老师 `OpenID` 失效 → 登录落新 User 无角色 → 403。处置 = **重发邀请码**，走这条既有路径重绑。

**为什么**：这条路径已经存在且被 `role=teacher` 的登录流程覆盖，新增一次性迁移脚本（清空 `users.open_id` 之类）反而引入破坏性操作风险（参考 `cmd/initcheck` 的教训）。

**备选**（放弃）：`cmd/` 加一次性重绑脚本 —— 本仓库 `cmd/` 下一次性脚本已有多起破坏性事故记录，且重发邀请码是老师端本来就有的管理员操作。

### D5 — 失败语义：凭据缺失 / 换取会话失败一律显式报错

对应 `wechat-login` 规格「微信凭据选取失败必须显式报错」。配置缺失在**启动或登录入口**即报，不回退到另一端凭据；`code2session` 失败把微信侧 `errcode/errmsg` 带进错误链。

**为什么**：CLAUDE.md「Silent Failures」红线。若静默回退到另一端凭据，会重现串号——而且症状是「某端接口全 403」，极难定位（memory 里 `openid-shared-first-match` 就是这类）。

## 跨端触点清单

| 层 | 会改 / 不改 | 说明 |
|---|---|---|
| **Go `internal/config/config.go`** | **会改** | `WeChatConfig` 从单对 `AppID`/`Secret` 扩为按端双对；`overrideFromEnv` 补按端 env（如 `WECHAT_TEACHER_APP_ID` 等）。`configs/app.ini.example` 同步样例键名（**不含值**） |
| **Go `internal/auth/wechat.go`** | **会改** | `WeChatClient` 改持按端双凭据；**四个方法全部按端选取**（`Code2Session` 按 role / `GetPhoneNumber` 用家长端 / `GetWXACodeUnlimit` 用老师端 / `GetAccessToken` 跟随调用方）；`GetAccessToken` 的 Redis 缓存键加 appid 后缀；换取会话失败时透传微信 `errcode/errmsg` |
| **Go `internal/app/provider.go`** | **会改** | `ProvideWeChatClient` 构造时注入按端凭据（DI 返回类型保持 `*auth.WeChatClient` 不变，避免动 Wire 形状） |
| **Go `internal/app/wire.go` / `wire_gen.go` / `service_provider.go`** | **会改（仅 wire_gen 再生成）** | provider 签名若变需 `make wire`；`service_provider.go:20` 的 `wx *auth.WeChatClient` 参数类型不变则无需改 |
| **Go `internal/service/auth_service.go`** | **会改** | `WeChatLogin` 让 `role` 真正参与凭据选取；`BindParentPhone`（`auth_service.go:541`）的 `GetPhoneNumber` 走家长端凭据；`weChatLoginWithInvite` 保持，作为重绑路径 |
| **Go `internal/service/teacher_service.go`** | **会改** | `GetWXACodeUnlimit`（`teacher_service.go:519`，教师邀请码二维码）改走老师端凭据——否则生成的是家长端小程序码，重绑主路径断掉 |
| **Go `internal/domain/user.go`** | **会改（仅注释）** | `UserRole` 注释「一个用户可有多个角色，如同时是老师和家长」的跨端语义已被推翻，改注释说明：同端多角色仍成立，跨端同人是两条 User + 关联。**不改任何字段/索引** |
| **Go `internal/repository/user_repo.go`** | **会改** | 补「按 unionid / phone 反查」的查找方法（供关联判定）。`FindByOpenID` 保持——拆端后两端 openid 天然不同，`First()` 误命中问题自然消失 |
| **Go `handler/http/auth.go`** | **不改** | `wechatLoginReq` 已含 `Role`/`Code`/`InviteCode`，接口形状不动。仅当需要把「凭据缺失」映射为更明确的 HTTP 错误码时才碰，且不得改响应包络 |
| **Go Casbin policy（`rbac_policy.csv`）** | **不改** | 无新增/修改路由，`(role, path, method)` 三元组不变 |
| **Go `internal/app/router.go`** | **不改** | `POST /auth/login/wechat`（`router.go:27`）已注册，路径与 method 均不变 |
| **`backend/api/api-common.md`** | **不改** | `POST /auth/login/wechat` 契约已含 `role`，本次**不属于契约变更**。`api-teacher/` / `api-parent/` 同理不改 |
| **teacher `src/manifest.json` + `project.config.json`** | **会改** | AppID `wxdf98…` → `wxbfb5…`。注意 IDE 读 `dist/dev/mp-weixin/project.config.json`，改完必须重新 `npm run dev:mp-weixin` |
| **parent `src/manifest.json` + `project.config.json`** | **不改** | 已是 `wxdf98…`，保持 |
| **teacher / parent `src/api/auth.js`** | **不改** | `role` 已各自硬编码（`"teacher"` / `"parent"`），签名不变 |
| **teacher / parent `src/utils/login.js`** | **不改** | 两端各自独立实现，签名与调用方不变；本次只改后端对 `role` 的用法 |
| **teacher `pages/webview/index.vue`** | **不改** | 个人主体不支持 `<web-view>`，该页本轮白屏属**已知限制**，划出验收范围，不改代码 |
| **parent `pages/index/index.vue`** | **不改** | `open-type="getPhoneNumber"` 绑定卡片继续用家长端 AppID（非个人主体），能力不受影响 |
| **其余端页面（`pages/*.vue`）** | **不改** | 登录后业务页面不感知凭据选取与关联判定 |
| **跳转 / tabBar** | **不涉及** | 本次无任何新增/修改跳转。家长端 4 项 tabBar（`index`/`growth`/`pickup`/`profile`）与老师端无 tabBar 的差异不受影响，无 `switchTab` / `navigateTo` 需要核对 |
| **`CLAUDE.md`** | **会改** | 「架构铁律」的 SoT。本次推翻了「两端共用一个 AppID」这条既有认知，必须写清：两端各用哪个 AppID、老师端是个人号的两个后果（webview 白屏、无 unionid）、跨端同人是两条 User + 关联、四个微信方法各自选端。否则新会话会按旧结论行事 |
| **design token（`shared-design-tokens/`）** | **不改** | 无样式变更 |
| **admin/** | **不改** | 密码登录路径，不涉及微信凭据与 openid |

### 数据层：无迁移

`users` 的 `open_id` / `union_id` / `phone` 三列均已存在（`user.go:5-9`），**本次不新增列、不新增表**，因此不涉及 `AutoMigrate` 的「只增不改」局限，也不需要手工 SQL 与 `dbcheck` 核实。

若将来把关联物化为列（见 D3 备选），那时才需要手工迁移 + `go run ./cmd/dbcheck users`。

## 关键流程

### 微信登录（按端选凭据）

```
小程序端                     后端
   |                          |
   | POST /auth/login/wechat  |
   | {code, role, invite?}    |
   |------------------------->|
   |                          | 1. 校验 role ∈ {teacher, parent}
   |                          |    未知/为空 → 显式报错（不猜）
   |                          |
   |                          | 2. 按 role 取 (AppID, Secret)
   |                          |    缺配置 → 显式报错（不回退另一端）
   |                          |
   |                          | 3. code2session(AppID, Secret, code)
   |                          |    失败 → 透传 errcode/errmsg，不签发令牌
   |                          |
   |                          | 4a. inviteCode 非空
   |                          |     → 写 openid 进邀请指向的教师账号（重绑）
   |                          | 4b. inviteCode 为空
   |                          |     → FindByOpenID(本端 openid)
   |                          |        命中 → 复用该 User
   |                          |        未命中 → 新建本端 User
   |                          |
   |  {token, refresh, user}  |
   |<-------------------------|
```

### 跨端同人判定（派生，非物化）

```
判定两条 User A、B 是否同一自然人：

   A.union_id 非空 且 B.union_id 非空 ?
      ├─ 是 → 比 union_id：相等 → 同人；不等 → 不同人（结束）
      └─ 否 ↓
   A.phone 非空 且 B.phone 非空 ?
      ├─ 是 → 比 phone：相等 → 同人；不等 → 不同人
      └─ 否 → 无法判定 → 不同人（不猜测）
```

顺序约定：**只要两侧 unionid 都非空就以它为准**，不因「phone 也一致」而推翻；仅当至少一侧 unionid 为空时才降级到 phone。

有意选择「一侧无 unionid 时仍允许 phone 关联」——测试阶段老师端是个人主体、拿不到 unionid，必须靠 phone 才能验到关联路径。代价是 phone 冲突时可能误关联，已列入 Risks 并由「关联不修改账号归属」这条规格兜底。

## Risks / Trade-offs

- **[老师端切换 AppID 后已有老师登录必现 403]** → 一次性影响，根因是 `OpenID` 全部失效。处置：管理员重发邀请码，走 `weChatLoginWithInvite` 重绑（D4）。tasks 里排在切 AppID 之后、验证之前。家长端不受影响。
- **[phone 兜底可能误关联]** → 同一手机号被两个不同自然人使用（家长共用手机、老师换号未更新）会被判为同人。缓解：判定只用于「取另一端账号」这类只读场景，不合并账号、不动权限（见 `cross-end-identity` 规格「关联判定不修改账号归属」）；将来 unionid 可用后自动升级为首选，误关联面收窄。
- **[个人主体小程序无 unionid，关联只能靠 phone]** → 若老师账号 `phone` 为空（预创建时未填），关联判定返回「无法判定」而非报错。缓解：`weChatLoginWithInvite` 重绑路径本就指向管理员预创建的账号，建号时要求填手机号；tasks 里列为数据准备项。
- **[AppSecret 泄漏，风险已接受]** → 两个 Secret 已在对话中暴露。**2026-10-05 用户决定不轮换**，按已泄漏对待。仍有效的边界：Secret 只写 `server/configs/app.ini`（已 gitignore）或环境变量，`app.ini.example` 只放键名不放值，绝不进 git（CLAUDE.md Known Issues #1 的教训）。若日后要收口，唯一真正有效的补救仍是轮换——历史清理只是辅助。
- **[改 AppID 后忘记重新构建，IDE 跑旧产物]** → 症状是「登录仍用旧 AppID」或 MCP `cli auto` 报端口未监听。缓解：tasks 验证步骤强制 `npm run dev:mp-weixin`，并探该端独有页面确认驱动的是哪个项目。
- **[MCP 验证串行限制]** → 拆 AppID **不解决**微信开发者工具「单实例单项目窗口」的问题，双端验证仍须串行。缓解：tasks 的验证分组标注 `[P]` 的是可并行的构建/编译，验证一律串行。
- **[D3 派生查找 vs 物化]** → 派生查找每次多一两次索引查询。幼儿园规模可接受；若未来「取同人另一端账号」成为热点，再物化，判定逻辑不变。

## Migration Plan

1. **配置准备**：确认 `server/configs/app.ini` 已按端写入两对凭据（只落 gitignored 文件，不进 git）。`wxbfb5…` 主体类型为个人（已知）。~~重置 AppSecret~~ **2026-10-05 决定不做**，见 Risks。
2. **后端改造**：按 tasks 的后端分组实施 → `make check` → `make wire`（若 provider 签名变）。
3. **老师端 AppID 切换**：改 `teacher/src/manifest.json` + `teacher/project.config.json` → `npm run dev:mp-weixin` 重建产物。
4. **重绑老师账号**：管理员重发邀请码，老师扫码/输入走 `weChatLoginWithInvite` 重绑。
5. **平台配置（人工）**：两个 AppID 各自配置服务器域名白名单；把开发者微信加为两个小程序的开发者。
6. **验收**：按 specs 逐条对照（见 tasks 的验证任务）。老师端 `webview` 页面不在验收范围。

**回滚**：后端凭据改回单对（保留旧 `WECHAT_APP_ID`/`WECHAT_SECRET` 兼容读取，见 tasks）即可回到共用 AppID 状态；老师端 AppID 改回 `wxdf98…` 并重新构建。注意回滚后老师端 `OpenID` 又会失效一次，需再次重绑。

## Open Questions

（无。影响规格与任务拆分的决策均已在 D1–D5 定案。）
