# Proposal

## Why

家长端与老师端目前共用同一个小程序 AppID（`wxdf989…`），导致同一个人在两端 `code2session` 拿到**同一个 openid**，`FindByOpenID` 用 `First()` 命中最小 id 的那条 User，另一端角色丢失、接口全 403（已实证，见 `openid-shared-first-match`）。现测试阶段只有一套微信凭据，无法验证真实双端身份隔离。

同时后端只持有一对微信凭据（`config.WeChat` 单 `AppID`/`Secret`），`WeChatLogin` 的 `role` 参数是形参、不参与凭据选择，这是串号的结构性根源。

现在拆开：两端各用各的 AppID，身份按端隔离，跨端同人用关联字段打通。

## What Changes

- **BREAKING** 老师端小程序 AppID 从 `wxdf989efff8796c53` 切换为 `wxbfb5f81a87f73a61`（新测试号，**个人主体**）。切换后老师端已有 User 的 `OpenID` 全部失效，须走邀请码重绑（见 tasks）。
- 后端微信凭据从单对改为**按端双对**（teacher / parent 各一套），微信侧调用按端选取对应 AppID/Secret：登录 `code2session` 按 `role` 选、家长绑定手机号用家长端凭据、教师邀请码二维码用老师端凭据；access_token 缓存按 appid 隔离。
- 微信登录身份模型从「一个 openid 命中一条 User」改为「**按端各落一条 User**」。同一人在两端 = 两条 `users` 记录。
- 新增跨端同人关联：**`unionid` 优先，`phone` 兜底**。测试阶段老师端是个人号、拿不到 unionid → 走 phone；生产两端绑同一开放平台后自动切 unionid。
- 家长端 AppID 保持 `wxdf989efff8796c53` 不变，家长 openid 与已有绑定关系不受影响。
- `UserRole` 的旧语义（「一个用户可有多个角色，如同时是老师和家长」）被推翻：跨端同人不再靠 UserRole 聚合，而是靠关联字段。同端内多角色（如老师兼管理员）语义不变。

## Capabilities

### New Capabilities

- `wechat-login`: 微信小程序登录的按端身份隔离——teacher / parent 各自以本端小程序身份登录、落本端独立 User；换端登录不得命中另一端的 User。
- `cross-end-identity`: 跨端同人关联——两条 User 通过 `unionid`（优先）或 `phone`（兜底）判定为同一人；关联是数据层约束，不改变登录接口对外形状。

### Modified Capabilities

（无。`openspec/specs/` 当前为空，无既有 requirement 需要 MODIFY。）

## Out of Scope

- **不改 `backend/api/` 接口形状**：`POST /auth/login/wechat` 的请求/响应契约已含 `role` 字段，本次不新增、不删除、不改名任何接口字段。跨端关联是数据层内部行为，不通过接口暴露。**本次不属于契约变更。**
- 老师端 `pages/webview/index`（`<web-view>`）：新测试号是个人主体，个人小程序不支持 `web-view` 组件，该页白屏。**明确划出验收范围**，不作为本次失败项。
- 微信开放平台账号的注册与小程序绑定操作（`open.weixin.qq.com`）——人工运维动作，不在代码范围内；本次只保证「有 unionid 就用」的代码路径正确。
- **AppSecret 轮换——2026-10-05 用户决定不做**，风险接受。仅保留「Secret 只落 gitignored 配置、不进 git」这条边界。
- admin 端登录（密码登录）不涉及。
- 跨端关联的**业务消费**（如「老师兼家长」的数据可见性、消息互通）——本次只建立关联，不改任何业务查询行为。
- `doctor` 端独立 AppID：`pages/doctor/checkin` 是老师端内页面，继续共用老师端 AppID，不单独拆。

## Impact

**后端（Go，`server/`）**

- `internal/config/config.go` — `WeChatConfig` 单对凭据 → 按端双对 + env override
- `internal/app/provider.go` — `NewWeChatClient` 单例 → 按 role 取客户端
- `internal/auth/wechat.go` — `Code2Session` 需按端拼 AppID/Secret
- `internal/service/auth_service.go` — `WeChatLogin` 的 `role` 从形参变为真正选凭据；`BindParentPhone` 走家长端凭据；`weChatLoginWithInvite` 重绑路径
- `internal/service/teacher_service.go` — 教师邀请码二维码走老师端凭据
- `internal/domain/user.go` — `UnionID` 语义启用；`UserRole` 注释语义更新
- `internal/repository/user_repo.go` — `FindByOpenID` 等查找语义按端核对
- Wire DI：`make wire`（凭据提供方式变化会牵动 provider set）

**端（小程序）**

- `teacher/` — `src/manifest.json`、`project.config.json` 的 AppID 切换；`src/webview` 页面本轮不验
- `parent/` — AppID 不动；登录代码无需改
- 两端各有独立的 `utils/login.js`（非共享包），本次不改其签名，仅后端语义变化

**不涉及**

- Casbin policy / `router.go` — 无新增路由，不改
- `backend/api/` — 不改
- design token — 不改
- admin — 不改

**数据**

- 老师端切换后 `users.open_id` 须重绑（邀请码路径），否则登录落新 User 无角色 → 403
- `users.union_id` / `users.phone` 承担跨端关联；两者已有列与索引，**不需改表结构**（`AutoMigrate` 只增不改，本次无新列）

**运维**

- 两个 AppID 都要在公众平台配置服务器域名白名单
- 两个小程序都要把开发者微信加为开发者
- 两个 AppSecret 只写 `server/configs/app.local.ini`（已 gitignore）或环境变量，绝不进 git

## 归档回溯：实际触点与设计阶段的差异（2026-10-11 补记）

**设计阶段的跨端触点清单全部命中**（config / auth / provider / service / domain 注释 / repo / manifest / CLAUDE.md），无遗漏层。设计阶段没有预料到的是：

1. **数据层的串号残留比 D2 假设的更深**。D2 断言「两端 openid 天然不同」——对**新**记录成立，但存量记录不是：普查发现 user 11（家长）与 user 12（老师）的 `open_id` 逐字节相同；另有 user 61（俊彦爸爸）、user 65（2026-10-05 开发遗留）两条串号账号。user 65 还把持着新老师 AppID 的 openid，令「再次登录命中本端」一度落到它身上（TC-0002 被它阻塞）。**处置**：经用户批准清空 user 65 的 `open_id`（`UPDATE users SET open_id=NULL WHERE id=65`），user 11/12 未动，老师账号走邀请码重绑。**教训**：拆端类变更必须普查存量 openid 重复度，「新值天然不同」不能外推到旧数据。
2. **`invite_code` 字段命名陷阱**（验收中实际踩到）。后端绑定字段是 snake_case `invite_code`（`handler/http/auth.go` 的 `wechatLoginReq`），手搓请求误写 `inviteCode` 会被**静默忽略并降级为无邀请码登录**——不报错、不返回 40020。端上 `api/auth.js` 做了映射所以端侧无恙；直接调接口的验证脚本必须核对 `json:` 标签。本项的实测结论已写进 `_verify/split-wechat-appid-identity-acceptance.md`。
3. **验证工具链被替换**。设计假定走 `wechat_automator`；本会话窗口未加载 `wechat_*` MCP，改用 CDP 等价物：新增 `scripts/evaluate.py`（在 appservice 逻辑层下发 `wx.login`/`wx.request`，与端上登录同业务路径）+ 既有 `click.py`/`shot.py`（视图层）。验证强度不降级，但**执行路径与设计文档不符**，记录备查。另记一坑：`location.reload()` 不真正重启小程序实例，视图层会留着旧渲染（逻辑层身份已换、页面文本还是旧的），不是产品缺陷。
4. **E2E 覆盖存在环境边界（唯一未闭环项）**。TC-0001（老师端首次建号）因模拟器微信号在老师端 openid 已固定而**不可复现**；TC-0009/0010/0011（unionid 三类）因老师端为个人主体、微信不下发 unionid 而**不可验**。四项 Unit 全过，E2E 留待换未绑定微信号 / 两端绑开放平台后补验。
5. **7.x 范围改判**。原计划含真机/体验版验证与服务器域名白名单；用户 2026-10-11 决定本阶段只做本地验证，白名单延后到上线前必须补做。7.3 以本地两端真实链路登录（老师端 user 12、家长端 user 61 各拿到令牌）作为等价验证。
