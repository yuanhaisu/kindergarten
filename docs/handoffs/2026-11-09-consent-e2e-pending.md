# 隐私同意门禁：实现与本地端到端验证完成

**日期**：2026-11-09（2026-11-10 更新）
**变更**：`openspec/changes/add-privacy-consent-and-log-redaction/`
**结论**：代码、构建与三端本地端到端验证已完成；仅剩 staging 集成（`tasks.md` 8.2）。**8.2 通过前不得 archive。**

## 已验证

| 范围 | 证据 |
|---|---|
| 数据库 | `00005_user_consents.sql` 已执行；`go run ./cmd/dbcheck user_consents` 确认唯一键 `(user_id, policy_version)` 与索引 |
| 服务端单测 | `go test ./internal/service ./internal/middleware ./internal/handler/... -run Consent -count=1` 通过 |
| 脱敏单测 | `go test ./pkg/redact ./internal/repository -run 'Redact|Consent' -count=1` 通过 |
| 回填命令 | `cmd/redactlogs` 单测覆盖 dry-run 不写库、执行后掩码、二次执行 0 行；开发库 dry-run 扫描 159 行（未写库） |
| 接口行为 | 用真实签发 JWT 验证：`GET /common/consent` 返回 `consented=false` → `POST` 后为 `true`，重复提交返回首次 `consented_at`（幂等） |
| 门禁行为 | 临时把 `[consent] enforce` 设为 `true`：业务接口返回 `403 {"code":40301}`；`/common/consent` 放行；同意后业务接口返回 `40300`（鉴权失败而非同意门禁），证明门禁已通过 |
| 构建 | `go build ./...`、`make wire`、家长端 `build:mp-weixin`、老师端 `build:mp-weixin`、管理端 `npm run generate` 均通过 |
| 老师端启动门禁 | 自动化 `page_stack` 显示启动后栈内只有 `pages/auth/consent`；`wechat_inspector` 采集 0 条 console 错误 |

## 2026-11-10 补充验证（已完成）

`wechat_automator(tap)` 仍会 120 秒超时，`page_data` 挂起；改用 `_verify/tap.py` 的 `Input.dispatchTouchEvent`（touchStart/touchEnd）在视图层真实点击，行为与 `bindtap` 一致。

| 端 | 结果 |
|---|---|
| 家长端 | 40301 → `pages/auth/consent`；触摸勾选后 `.agree-box.is-checked=true`；触摸「同意并继续」写入 `user_consents`（user 61 / v1），页面栈回 `pages/index/index`；业务接口恢复 200 |
| 老师端 | 40301 → consent；勾选 + 同意写入（user 11 / v1），页面栈回 `pages/index/index`（无 tabBar）；业务接口恢复 200 |
| 管理端 | Playwright 真实登录未同意账号 → `/consent`；「暂不同意」→ `/login` 且不写记录；重新登录勾选同意 → `/` dashboard，写入 `user_consents`（user 4 / admin / v1） |
| 历史日志 | `cmd/redactlogs -dry-run=false` 处理 159 行；二次执行为 0 行；`GET /admin/logs` 488 条，完整手机号/身份证号匹配数均为 0 |

未自动化项：家长端/老师端「暂不同意」后原生 `uni.showModal` 的「退出登录」确认按钮不在 pageframe DOM，CDP 点不中；已验证点击后不写记录且停留在阻断页，退出分支为代码走查（admin 端同类分支已完整实测）。

**来源端（已修复并复验）**：多角色用户原先按令牌角色顺序记 `source`，user 61 在家长端同意记为 `teacher`、user 11 在老师端同意记为 `parent`。现改为读取 `terminal_from` 请求头（parent/teacher/admin），服务端校验该端属于令牌角色后写入；缺省回退角色推导以兼容旧客户端。契约见 `backend/api/api-common.md` §5，单测见 `TestConsentHandler_HeaderOverridesRoleOrder`。2026-11-10 复验：user 61 + `terminal_from=parent` → `source=parent`；user 11 + `terminal_from=teacher` → `source=teacher`；`terminal_from=admin`（不在角色内）→ 400；缺省 header → 回退为 `teacher`。

## 可复现的手工验证步骤

### 家长端

前置：后端 `[consent] enforce=false`（当前默认）；开发库中该家长账号在 `user_consents` 无 `v1` 记录；`parent/dist/dev/mp-weixin` 为最新构建。

1. 微信开发者工具打开 `parent/dist/dev/mp-weixin`，清缓存后重新编译。
2. 启动小程序，观察是否直接落在「隐私政策与用户协议」页且底部「同意并继续」可点。
3. 不勾选直接点「同意并继续」→ 应提示「请先勾选同意」，不发生跳转。
4. 点击「我已阅读并同意…」→ 勾选框出现对勾。
5. 点「查看完整《隐私政策》」→ 进入 `pages/auth/policy`，web-view 打开官网 `/privacy`；返回。
6. 点「同意并继续」→ toast「已同意」→ 约 600ms 后回到首页 tab（`pages/index/index`，tabBar 可见）。
7. 杀掉小程序重进 → 不再出现阻断页（`GET /common/consent` 返回 `consented=true`）。
8. 直接访问业务接口（如首页数据）→ 无 `40301`。

### 老师端

前置同上，打开 `teacher/dist/dev/mp-weixin`。

1. 启动后应直接落在同意页（老师端无 tabBar，页面不应出现底部 tab）。
2. 未勾选点「同意并继续」→ 提示「请先勾选同意」。
3. 点「复制《隐私政策》链接」→ toast「链接已复制，请在浏览器打开」，剪贴板内容为 `${siteURL}/privacy`。
4. 勾选 → 点「同意并继续」→ toast「已同意」→ 回到工作台 `pages/index/index`（用的是 `reLaunch`，不是 `switchTab`）。
5. 重进小程序 → 不再出现阻断页。
6. 点「暂不同意」→ 弹窗确认「退出登录」→ 清空本地登录态并回到 `pages/auth/login`。

### 管理端

前置：`admin/.env` 的 `NUXT_PUBLIC_SITE_URL` 指向可访问的官网地址（本地可用 `http://127.0.0.1:3000`）；使用一个 `user_consents` 中无 `v1` 记录的管理员账号。

1. `npm run dev`，打开 `/login`，用该账号登录。
2. 登录成功后应落到 `/consent`，而不是 dashboard。
3. 不勾选时「同意并继续」为禁用态；勾选后可点。
4. 点「查看完整《隐私政策》」→ 新标签打开 `/privacy`。
5. 点「同意并继续」→ 提示「已同意」→ 进入 dashboard。
6. 刷新页面 → 不再回到 `/consent`。
7. 点「暂不同意」→ 清会话并回 `/login`。

## 剩余收尾

1. 在 staging 以 `enforce=true` 跑三端，并验证旧客户端在 `enforce=false` 下仍可用（任务 8.2）。
2. 8.2 通过后才可 archive；上线顺序见 `docs/runbook/deployment.md` §2.1。
