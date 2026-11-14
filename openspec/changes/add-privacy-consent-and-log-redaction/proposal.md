# Proposal

## Why

官网法务文本自查（`docs/handoffs/2026-11-09-legal-selfcheck.md`）发现两个合规缺口：

1. **没有服务端同意记录**。两端的「我已阅读并同意」只是登录页的客户端 `ref`，不落库；静默登录的用户从未经过勾选，平台无法证明取得过同意，也无法在隐私政策改版后要求重新同意。
2. **操作日志含明文个人信息**。`operation_logs.detail` 在账号创建、学生创建/修改等路径写入明文手机号、昵称、幼儿姓名与学号（`parent_service.go:395`、`teacher_service.go:406`、`admin_service.go:279`、`student_service.go:565-566 / 625-626 / 656-657`），与隐私政策「按最小必要原则处理并记录关键操作」的承诺不符。

现在做，是因为官网已按实际数据流补齐了隐私政策，文本先于系统承诺了同意与最小化；需要让系统行为与文本一致，避免上线后出现「政策写了、系统没做」。

**本次将修改 `backend/api/`，属于契约变更。**

## What Changes

### 同意记录（privacy-consent）

- 新增按「用户 × 政策版本」的记录表，保存同意时间、政策版本、来源端（teacher / parent / admin）、IP 与 User-Agent；同一用户同一版本只有一条，版本升级后新增一条。记录不可软删。
- 新增 `GET /api/v1/common/consent`（当前政策版本与本人同意状态）与 `POST /api/v1/common/consent`（记录同意）。
- 登录响应（家长/老师微信登录、管理端密码登录）增加 `consent_required` 与 `current_policy_version`；字段为增量，不改变既有令牌与身份逻辑。
- 服务端兜底：已登录但未同意当前版本的请求，访问业务接口时返回 `403 CONSENT_REQUIRED`；客户端统一拦截并跳转同意页。
- 家长端、老师端：静默登录后若 `consent_required=true`，进入阻断页；勾选并同意后写库，再进入首页。
- 管理端：密码登录后若 `consent_required=true`，进入 `/consent` 阻断页；同意后写库并进入后台。
- 政策版本值由服务端常量提供，官网 `website/content/site.ts` 同步显示同一版本号。

**BREAKING**：未同意当前政策版本的已登录用户，下一次访问业务接口会收到 403，必须先完成同意。这是本次变更的目的，不是副作用。

### 操作日志脱敏（operation-log-redaction）

- 新增统一的脱敏工具：手机号（保留前 3 后 4）、身份证号（保留前 6 后 4）、学号（保留前 2 后 2）、姓名（保留首字，其余替换为 `*`）。
- 操作日志服务在写入前对 `detail` 应用文本级脱敏；已知写入点改为先脱敏再拼接，不再写入明文手机号、昵称、幼儿姓名/学号。
- 新增一次性回填命令 `cmd/redactlogs`：按批扫描 `operation_logs`，对手机号/身份证/学号模式做脱敏，默认 dry-run，输出将修改的行数；执行后可用 `cmd/dbcheck` 核对。
- 端页面不需要改动；后台操作日志列表继续展示脱敏后的 `detail`。

## Capabilities

### New Capabilities

- `privacy-consent`: 政策版本化的同意记录、登录与业务接口的同意门禁、政策改版后的重新同意。
- `operation-log-redaction`: 操作日志写入前脱敏与历史明文回填。

### Modified Capabilities

（无。`wechat-login` 的身份隔离需求不变；登录响应新增字段由 `privacy-consent` 拥有。）

## Impact

**后端（Go）**：新增 `server/internal/domain` 同意模型、repository、service、handler、consent 中间件；`router.go` 注册 `/common/consent` 与业务分组门禁；`wire.go` / `wire_gen.go` 补 provider 与 `wire.Bind`；新增 `server/migrations/00005_*.sql`。

**契约（`backend/api/`）**：`api-common.md` 新增同意端点与 `CONSENT_REQUIRED` 错误码；家长/老师/管理端登录响应增加 `consent_required`、`current_policy_version`。字段来源是本次合规需求，不是设计稿新增 UI。

**家长端（`parent/`）**：`src/utils/login.js` 检查同意状态；新增阻断页（`pages/auth/consent`）；`pages.json` 注册；登录页的协议名改为可点击链接（家长端非个人号，可用 `<web-view>` 打开官网政策页）。

**老师端（`teacher/`）**：同上，但**个人号不支持 `<web-view>`**；政策全文的端内呈现方式在 design 决定（摘要 + 复制链接，或端内全文页）。

**管理端（`admin/`）**：登录后同意门禁；新增 `/consent` 页；`middleware/auth.global.ts` 或布局层接入检查。

**运维/工具**：`cmd/redactlogs`；`docs/tests/TEST_CATALOG.md` 登记新 TC；`website/content/site.ts` 增加政策版本号。

**测试**：同意记录的幂等与版本升级、403 兜底、两端阻断页真实点击、脱敏函数的表驱动单测、回填命令的 dry-run/执行。

## Out of Scope

- 法务复核与隐私政策正文的进一步修改（已完成一轮代码事实自查）。
- OSS 对象删除、人脸库移除、人脸授权流程、短信服务商接入、账号注销、家长「禁用」登录校验。
- 历史日志中**姓名**的自动识别：自由文本里的姓名无法用正则可靠识别；回填只处理手机号/身份证/学号模式，姓名脱敏从新写入开始。已知写入点的历史明文姓名需人工复核或后续专项处理。
- 管理端以外的平台运营账号（`platform_admins`）的同意记录。

## 待定 / 批准前需确认

1. **政策版本值**：建议 `v1`（备案通过、文本定稿后改为生效日期，如 `2026-12-01`）。若希望直接用日期，请在批准时说明。
2. **老师端政策全文呈现**：个人号无 `<web-view>`；design 会给出两个方案并选一个，不阻塞 proposal。
3. **是否对园长/行政也做 403 强阻断**：本提案按你的要求包含 admin；若只想先覆盖家长/老师，请在批准前说明。
