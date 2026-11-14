# Design

## Context

见 `proposal.md` — Why。当前状态与约束：

- 登录入口有三个：家长端与老师端的微信登录、管理端的密码登录。两端小程序启动时静默登录（`App.vue onLaunch → ensureLogin()`），登录页的勾选只在客户端，不落库。
- 老师在微信侧是个人主体 AppID，**不支持 `<web-view>`**；家长端是非个人主体，支持 `<web-view>`。官网 `www.nurtureplanet.cn` 尚未备案上线。
- 服务端已用 goose 迁移，应用启动不跑 AutoMigrate；新增表必须写 `server/migrations/00005_*.sql` 并用 `cmd/dbcheck` 核对。
- `operation_logs.detail` 是自由文本；已知明文写入点在 `parent_service.go:395`、`teacher_service.go:406`、`admin_service.go:279`、`student_service.go:565-566 / 625-626 / 656-657`。
- ADR-0009 要求接口对线上旧版小程序保持向后兼容；本次的 403 门禁对旧客户端是破坏性的，需要分阶段启用。

## Goals / Non-Goals

**Goals:**
- 服务端可证明「某用户在某时间同意了某版本政策」，并支持版本升级后的重新同意。
- 三端在缺同意时无法进入业务页；服务端有兜底，不只依赖 UI。
- 操作日志不再新增明文手机号、身份证号、学号、姓名；历史可识别 PII 可批量清理。
- 旧版小程序在 consent 新功能上线初期仍可正常使用，直到明确启用门禁。

**Non-Goals:**
- 不改隐私政策正文与官网页面结构。
- 不做账号注销、OSS 对象删除、人脸授权、短信接入。
- 历史自由文本中的姓名不做自动识别（无法可靠 regex），只做新写入与结构化写入点。

## Decisions

### D1. 同意记录用独立表，不用 `users` 字段

新建 `user_consents`，一行 = 一个用户对一个政策版本的一次同意；`UNIQUE(user_id, policy_version)`。字段：`user_id / policy_version / source / ip / user_agent / consented_at / created_at`。

- **为什么不用 `users.consent_version`**：用户升级政策版本后需要保留历史同意；单字段会覆盖历史。
- **为什么不加 `deleted_at`**：同意是审计记录，只追加不删除；软删行会占唯一索引，导致同一用户同一版本无法重新写入（见项目既往坑）。
- **`source` 优先取 `terminal_from` 请求头**（parent / teacher / admin）：服务端校验该端出现在令牌角色中，避免多角色用户记错端，也避免伪造来源；旧客户端未带该头时回退为按令牌角色推导（兼容期行为）。
- **幂等**：`INSERT ... ON DUPLICATE KEY UPDATE id=id`，随后读取首次 `consented_at`；重复提交不改时间、不新增行。

### D2. 政策版本是服务端常量，官网同步显示

`CurrentPrivacyPolicyVersion = "v1"` 放在服务端；官网 `website/content/site.ts` 增加同值 `policyVersion`，法务页显示「版本：v1」。备案通过、文本定稿后两处一起改为生效日期（如 `2026-12-01`）。

- 备选：把版本放进共享 JSON 供 Go 与 Nuxt 读取。收益是单一来源，但会引入跨语言构建耦合；当前只有两处，用 checklist 控制足够。
- 风险：两处漂移。缓解：发布前任务里逐条核对，并在 `backend/api/api-common.md` 记录当前值。

### D3. 登录响应增量返回同意状态；服务端门禁用配置开关分阶段启用

登录响应（三端）新增 `consent_required` 与 `current_policy_version`；这是增量字段，旧客户端会忽略。

服务端新增 consent 中间件：已登录但未同意当前版本时，业务接口返回 `403` + 可识别错误码；`/common/consent`、`/auth/logout`、`/health`、`/ready` 放行。门禁由 `app.ini` 的 `[consent] enforce` 控制，**默认 false**；两端小程序都发布带同意页的版本后，再置 true。

- **为什么不直接强制**：ADR-0009 要求旧版小程序仍可用；直接 403 会让未升级的线上用户全部不可用。
- **为什么仍然要服务端门禁**：只靠客户端跳转可被绕过；门禁是兜底，不是 UI 的替代。
- 管理端与服务器同版本发布，可在管理端登录后立即走客户端阻断页；服务端 403 同样等 `enforce=true` 后生效。

### D4. 三端阻断页与政策入口

- 家长端：`pages/auth/consent`，`uni.reLaunch` 进入（不是 tabBar 页）；同意后 `uni.switchTab('/pages/index/index')`（家长端有 4 个 tab）。
- 老师端：`pages/auth/consent`，`uni.reLaunch` 进入；同意后 `uni.reLaunch('/pages/index/index')`（老师端无 tabBar，不存在 switchTab）。
- 管理端：`pages/consent`，登录后由 `middleware/auth.global.ts` / auth store 检查并 `navigateTo('/consent')`；同意后回 `/`。
- 阻断页内容：政策版本、摘要（收集什么/谁可见/权利/联系方式）、完整政策入口、勾选、「同意并继续」、「暂不同意」。
- **完整政策入口**：家长端可用 `<web-view>` 打开官网 `/privacy`；老师端个人号不支持 `<web-view>`，提供「复制完整政策链接」+ 端内摘要。两端都不把全文硬编码进小程序，避免三处文本漂移。
- 「暂不同意」：不写记录，登出并回登录页；不得进入业务页。

### D5. 脱敏在唯一写入点 + 结构化写入点双层处理

新增 `server/pkg/redact`：

- `Phone`：保留前 3 后 4；`IDCard`：保留前 6 后 4；`StudentNo`：保留前 2 后 2；`Name`：保留首字，其余 `*`（两字姓名 → `张*`，三字及以上 → `张**`）。
- `Text`：对自由文本做手机号（`1[3-9]\d{9}`）与身份证号（`\d{17}[\dXx]`）正则替换；**不做学号与姓名的自由文本识别**（误伤率高）。
- 在操作日志服务的唯一写入函数中统一调用 `redact.Text`；已知写入点改为先 `redact.Phone/Name/StudentNo` 再拼接。
- `Text` 必须幂等：已含 `*` 的片段不再二次替换；正则只匹配完整模式。

### D6. 历史回填命令

`server/cmd/redactlogs`：

```
-dry-run  默认 true；只统计将修改行数
-batch    默认 500
-limit    0 = 全表
```

- 用 `detail REGEXP` 先筛出含手机号/身份证号的行，再在 Go 中 `redact.Text`，仅更新有变化的行。
- 不处理姓名；命令结束时打印「姓名需人工复核」的提示，并在任务里登记为已知缺口。
- 可重复执行；第二次运行修改行数应为 0。

### D7. 契约字段与来源

| 端点 | 字段 | 来源 / 场景 | 为什么不能派生 |
|---|---|---|---|
| `POST /auth/login/*` 响应 | `consent_required` | 静默登录后决定是否进入阻断页 | 取决于该用户 × 当前版本的记录，无法由角色或令牌派生 |
| 同上 | `current_policy_version` | 阻断页展示、POST 校验 | 服务端常量；客户端时间不可信 |
| `GET /common/consent` | `policy_version` / `consented` / `consented_at` | 管理端启动、阻断页回显 | 来自 `user_consents`；`consented` 不能由登录时间推导 |
| `POST /common/consent` 请求 | `policy_version` | 客户端确认自己看到的是当前版本 | 防止旧页面同意旧版本；服务端必须比对 |
| 业务接口错误 | `403 CONSENT_REQUIRED` | 客户端拦截并跳转阻断页 | 不能复用 401（未登录）或普通 403（权限不足），否则客户端无法区分 |

### D8. 迁移与核对

`00005_user_consents.sql`（goose Up/Down）：

```sql
CREATE TABLE user_consents (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  user_id BIGINT UNSIGNED NOT NULL,
  policy_version VARCHAR(32) NOT NULL,
  source VARCHAR(16) NOT NULL,
  ip VARCHAR(64) NOT NULL DEFAULT '',
  user_agent VARCHAR(512) NOT NULL DEFAULT '',
  consented_at DATETIME NOT NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  UNIQUE KEY uk_user_consent_version (user_id, policy_version),
  KEY idx_user_consents_user (user_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

- 应用不跑 AutoMigrate；必须 `make migrate`，再用 `go run ./cmd/dbcheck user_consents` 核对唯一索引与列。
- 回滚：Down 删除表；consent 功能随服务回滚，旧客户端不受影响（`enforce` 默认 false）。

## 跨端触点清单

| 层 | 会改 / 不改 | 说明 |
|---|---|---|
| Go domain | 会改 | 新增 `UserConsent`（无软删），`server/internal/domain/consent.go` |
| Go repository | 会改 | 新增 consent repo：按 user+version 查询、幂等写入、列表 |
| Go service | 会改 | 新增 consent service（当前版本、校验、记录）；操作日志服务接入 `redact` |
| Go handler | 会改 | 新增 `GET/POST /common/consent`；三端登录 handler 返回同意状态 |
| Go middleware | 会改 | 新增 consent 门禁；按 `enforce` 配置决定是否 403 |
| Casbin policy | 会改 | `/common/consent` 对 parent/teacher/admin 三角色放行；无需新角色 |
| router.go | 会改 | 注册 common consent 路由；业务分组挂 consent 中间件 |
| wire.go / wire_gen.go | 会改 | 新增 repo/service/handler provider；接口依赖补 `wire.Bind`；`make wire` |
| `backend/api/` | 会改（契约变更） | `api-common.md` 增加 consent 端点与错误码；三端登录响应增加两个字段 |
| 迁移 | 会改 | `server/migrations/00005_user_consents.sql` |
| 家长端页面 | 会改 | 新增 `pages/auth/consent.vue`；`pages.json`；`utils/login.js` 检查；`utils/http.js` 拦截 403 |
| 老师端页面 | 会改 | 同上；老师端无 tabBar，同意后用 `uni.reLaunch` 回工作台 |
| 管理端页面 | 会改 | 新增 `pages/consent.vue`；auth store + global middleware 检查；登录页展示同意状态 |
| design token | 不改 | 阻断页复用现有 token（Ink/Amber/Cream、胶囊按钮、卡片圆角），无新增色值 |
| `cmd/` | 会改 | 新增 `cmd/redactlogs` |
| 测试 | 会改 | redact 表驱动单测；consent service 幂等/版本升级；三端阻断页真机/automator 点击；回填 dry-run/执行 |
| `docs/tests/TEST_CATALOG.md` | 会改 | 登记 `TC-NEW-CNS-0001..0014`、`TC-NEW-LGR-0001..0011` |
| 官网 | 会改 | `content/site.ts` 增加 `policyVersion`；法务页显示版本号 |

## 流程

```
客户端（parent / teacher / admin）                 服务端
  |-- 登录 ---------------------------------------->|
  |<-- token + consent_required + version ----------|
  |                                                 |
  |  (consent_required=true)                        |
  |-- GET /common/consent ------------------------->|
  |<-- { policy_version, consented:false } ---------|
  |  展示阻断页（摘要 + 完整政策入口 + 勾选）         |
  |-- POST /common/consent { policy_version } ----->|
  |<-- { consented_at } ----------------------------|
  |-- 进入首页/后台 -------------------------------->|
  |                                                 |
  |-- 业务接口（未同意，enforce=true） ------------->|
  |<-- 403 CONSENT_REQUIRED ------------------------|
  |  拦截 -> 回阻断页                                |
```

```
未登录 --登录--> 已登录(无当前版本同意) --同意--> 已同意 --业务接口--> 正常
                        |                              ^
                        +--暂不同意--> 登出 ------------+
```

## Risks / Trade-offs

- **旧客户端在 enforce=true 后收到 403** → 分阶段：先发客户端，再开开关；发布说明写清「两端小程序都上线后再开启」。
- **政策版本两处漂移** → `backend/api` 记录当前值；发版 checklist 同时改 Go 常量与 `site.ts`。
- **老师端无法打开完整政策** → 端内摘要 + 复制链接；在阻断页明确提示「完整文本见官网」。这是个人号限制下的取舍。
- **回填误伤非 PII 数字** → 只处理手机号与身份证号正则；学号只在结构化写入点处理，不做自由文本匹配。
- **历史姓名无法自动清理** → 已知缺口，命令提示人工复核；新写入已覆盖。
- **consent 表随版本增长** → 每用户每版本一行，规模可忽略；唯一索引保证不重复。
- **管理端刷新后丢失登录响应** → auth store 启动时调用 `GET /common/consent`，不依赖内存中的登录响应。

## Migration Plan

1. 先跑 `00005_user_consents.sql`（staging → prod），`dbcheck` 核对。
2. 部署服务端（`enforce=false`）；三端登录响应开始带 consent 字段。
3. 发布家长端、老师端、管理端带阻断页的版本。
4. 观察一周，确认无旧版小程序仍在请求（或确认可强制升级），再把 `[consent] enforce` 置 true。
5. 运行 `cmd/redactlogs -dry-run` 核对行数，再正式执行；`dbcheck` 抽查 `operation_logs`。
6. 回滚：配置置 false → 回滚服务端镜像；表可保留（不删数据）。旧客户端立即恢复。

## Open Questions

（无。政策版本值取 `v1`；老师端政策入口取「摘要 + 复制链接」；门禁分阶段启用。均在本文决定。）
