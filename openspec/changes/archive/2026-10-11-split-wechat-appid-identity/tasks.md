# Tasks

> 分组按触点切。每组自带该组产出的测试/文档与验证门，不堆到最后。
> 标 `[P]` 的任务可与同组其它 `[P]` 任务并行；**MCP 验证一律串行**（微信开发者工具单实例单项目窗口，拆 AppID 不解决这一点）。

## 1. 凭据与配置准备（配置层）

- [x] 1.1 ~~在微信公众平台重置两个小程序的 AppSecret~~ **用户决定不做（2026-10-05）**。轮换这一步放弃，两个 Secret 仍为对话中暴露过的旧值，按已泄漏对待。剩余部分已达标：Secret 只落在 `server/configs/app.ini`（已 gitignore），`git status --porcelain` / `git diff` 中不出现任何 Secret 字样，`app.ini.example` 只有键名占位。**风险已知悉并接受**，不再是待办
- [x] 1.2 `server/internal/config/config.go`：`WeChatConfig` 从单对 `AppID`/`Secret` 扩为按端双对（teacher / parent），`overrideFromEnv` 补按端环境变量。验证：`go build ./...` 通过；未配置某端时该端字段为空串而非回退到另一端
- [x] 1.3 `server/configs/app.ini.example` 补按端键名样例（**只写键名与占位注释，不写真实值**）。验证：example 文件可被 `ini.Load` 解析，且 `grep` 不到任何真实 AppID/Secret
- [x] 1.4 【本组验证】`cd server && go build ./...`；`grep -r "e89fa413\|a1c2e233\|wxdf989e\|wxbfb5f8" --include="*.go" --include="*.md" .` 确认 Secret 未进源码（AppID 进 `project.config.json` / `manifest.json` 是允许的）

## 2. 后端微信凭据选取（Go auth / service / provider）

> 本组覆盖 `WeChatClient` 的**全部四个方法**，不止 `Code2Session`——见 design D1。
> 漏掉 `GetPhoneNumber` / `GetWXACodeUnlimit` 会让教师邀请码二维码生成错端、重绑主路径断掉。

- [x] 2.1 `server/internal/auth/wechat.go`：`WeChatClient` 改持按端双凭据，`Code2Session` 按 `role` 选取 (AppID, Secret)；`role` 未知或为空时**显式报错**，不猜测、不回退另一端。验证：单测覆盖「role 未知 → 报错」
- [x] 2.2 同文件：换取会话失败时把微信侧 `errcode`/`errmsg` 带进错误链，不得静默返回空会话。验证：单测断言错误串含 errcode
- [x] 2.3 同文件：`GetAccessToken` 的 Redis 缓存键从全局 `wechat:access_token` 改为带 appid 后缀，两个 AppID 不共用缓存。验证：单测断言两个 appid 的缓存键不同
- [x] 2.4 同文件：`GetPhoneNumber` 改走**家长端**凭据（`BindParentPhone` 只在家长端调用）。验证：单测断言用的是 parent 凭据
- [x] 2.5 同文件：`GetWXACodeUnlimit` 改走**老师端**凭据（`teacher_service.go:519` 教师邀请码二维码必须指向老师端小程序）。验证：单测断言用的是 teacher 凭据
- [x] 2.6 `server/internal/service/auth_service.go` `WeChatLogin`：让 `role` 真正参与凭据选取；该端凭据未配置时返回明确错误（对应 spec「微信凭据选取失败必须显式报错」TC-NEW-ATH-0005）。验证：单测覆盖「凭据缺失 → 明确错误，不签发令牌」。另抽取 `resolveWeChatUser` 以便对身份落库做单测（TC-NEW-ATH-0001~0004）
- [x] 2.7 `server/internal/app/provider.go` `ProvideWeChatClient`：注入按端双凭据，**DI 返回类型保持 `*auth.WeChatClient` 不变**（避免动 Wire 形状）。验证：签名未变，无需 `make wire`；`go build ./...` 通过
- [x] 2.8 补录 TC-NEW-ATH-0001 ~ 0008 到 `docs/tests/TEST_CATALOG.md` 的「认证 (ATH)」表（名称与 `specs/wechat-login/spec.md` 的 8 个场景一一对应）。验证：catalog 中 TC ID 与 spec 场景数一致、无重复
- [x] 2.9 【本组验证】`go vet ./...` 与 `go test ./...` 全部通过。`golangci-lint` 已补装（v2.14.0，须用 `GOTOOLCHAIN=go1.26.0 go install`——v2.14 构建需 go≥1.26，本模块是 1.25）并跑通：本次改动**未引入新 lint 问题**，并修掉了 `internal/auth/wechat.go` 里 4 处 `defer resp.Body.Close()` 的 errcheck。剩余 7 条均为**本次未触碰的既有文件**的债务，详见 3.5 附注。对照 `specs/wechat-login/spec.md`：TC-NEW-ATH-0001~0004/0007/0008 的单测已落地（0007 覆盖重绑写回部分，签发令牌属 E2E）；0005/0006 单测通过。**0001~0004/0007/0008 的 E2E 部分待 6.x 真实登录链路验证**

## 3. 跨端同人关联（Go repository / domain）

- [x] 3.1 `server/internal/repository/user_repo.go`：补按 `unionid` 反查、按手机号反查的方法（供关联判定使用）。⚠️ 实现中发现 `FindByPhone`/`FindByUnionID` 会命中**自己**那条（`First()` 按 id 最小），已改为 `FindOtherByUnionID` / `FindOtherByPhone` 排除自己。验证：单测覆盖命中 / 未命中 / 自己不算关联
- [x] 3.2 实现关联判定逻辑（`internal/service/cross_end_identity.go`）：**unionid 优先**（两侧都非空才比），否则 **phone 兜底**（两侧都非空才比），否则判定「无法判定」不猜测。验证：单测覆盖 `specs/cross-end-identity/spec.md` 的 6 个场景（TC-NEW-ATH-0009~0014）
- [x] 3.3 `server/internal/domain/user.go` `UserRole` 注释更新：同端多角色仍成立，**跨端同人是两条 User + 关联**，不再靠 UserRole 聚合「同时是老师和家长」。验证：注释与 `specs/cross-end-identity` 语义一致，无字段/索引改动
- [x] 3.4 补录 TC-NEW-ATH-0009 ~ 0014 到 `docs/tests/TEST_CATALOG.md`（已在 2.8 一并登记）。验证：catalog 中 TC ID 与 spec 场景数一致、无重复
- [x] 3.5 【本组验证】`go vet ./...` 与 `go test ./...` 全部通过；`golangci-lint run ./...` 已执行。对照 `specs/cross-end-identity/spec.md` 逐条核对 6 个场景的单测均已落地（TC-NEW-ATH-0009~0014）；`git diff internal/domain/user.go` 只有注释变更（+6/-1），`users` 表**无新增列**。⚠️ lint 剩余 7 条均在**本次未触碰**的文件（`handler/http/student.go`、`handler/http/teacher.go`、`service/file_service.go`、`storage/oss/oss.go`、`service/teacher_portal_service.go`、`service/auth_service_test.go`，以及工作区里他人未提交的 `service/parent_portal_service.go`），属既有债务，不在本次范围

## 4. 端页面：老师端 AppID 切换（teacher/）

- [x] 4.1 [P] `teacher/src/manifest.json:10`：AppID `wxdf989efff8796c53` → `wxbfb5f81a87f73a61`。⚠️ `teacher/project.config.json` 是极简结构（只有 `setting.minifyWXML`），**没有 appid 字段**——uni-app 从 `src/manifest.json` 注入到 dist，与 parent 的结构不同。验证：manifest 已为新值，重建后 `teacher/dist/dev/mp-weixin/project.config.json:16` 亦为新值
- [x] 4.2 [P] 确认 `parent/src/manifest.json` 与 `parent/project.config.json` 保持 `wxdf989efff8796c53` **不变**。验证：`git diff parent/` 无 AppID 相关改动；`parent/dist/dev/mp-weixin/project.config.json` 亦保持旧值
- [x] 4.3 重建产物（IDE 读 `dist/dev/mp-weixin/project.config.json`，不读 `src/manifest.json`）。验证：`teacher/dist/dev/mp-weixin/project.config.json` 中 appid 已是新值 `wxbfb5f81a87f73a61`；家长端 appid 未变、无需重建
- [x] 4.4 【本组验证】老师端已验：`wechat_ide(action='status')` → `node_available: true`、`project_path=teacher/dist/dev/mp-weixin`、`appid=wxbfb5f81a87f73a61`、`project_name=幼儿园老师端`；`wechat_automator(action='start')` → `tcp_ready` 与 `ws_ready` **均为 true**；探 `pages/more/index`（老师端独有）渲染成功、0 错误。⚠️ **家长端自动化探测未能完成**：`wechat_ide(action='open')` 指向 parent dist 返回 success，但随后 `status` 的 `project_path` 仍是老师端——CLAUDE.md 记录的单实例锁竞态，`success:true` 不可信。家长端探测延后到 6.3 一起做

## 5. 契约与路由：确认不改（集成检查）

- [x] 5.1 [P] 确认 `backend/api/api-common.md` 的 `POST /auth/login/wechat` 请求/响应契约未被本次改动（**本次不属于契约变更**）。验证：`git diff backend/api/api-common.md` 为空 ✓。⚠️ `backend/api/api-parent/04-growth.md` 有 diff，但那是**既有未提交的**成长 feed 字段梳理改动（配套 `server/internal/service/parent_portal_service.go` 批量查询优化），与本次无关，未触碰登录契约
- [x] 5.2 [P] 确认 `server/internal/app/router.go` 与 `configs/rbac_policy.csv` 未改。验证：`git diff` 两者均为空 ✓
- [x] 5.3 【本组验证】`go build ./...` 通过 ✓；`LoginResult` / `UserInfo` 的 json 字段与 `backend/api/api-common.md` 的响应结构逐字段一致，且本次 diff 中未出现任何 `json:` 标签改动（对应 TC-NEW-ATH-0013）
- [x] 5.4 同步「架构铁律」SoT：`CLAUDE.md` 写清两端各用哪个 AppID、老师端是个人号的两个后果（webview 白屏 / 无 unionid）、跨端同人是两条 User + 关联、四个微信方法各自选端；env 变量名改为按端。同步 auto-memory：`mp-appid-single` → `mp-appid-per-end`（内容重写），`openid-shared-first-match` 补注「跨端串号成因已消除但 First() 陷阱仍在」，更新 `MEMORY.md` 索引与入链。⚠️ **已开着的 Claude Code 窗口读的是启动时的快照，须重启才生效**

## 6. 数据准备与老师账号重绑

- [x] 6.1 确认测试用老师账号在预创建时**已填手机号**（关联判定兜底依赖它；空手机号会判定为「无法判定」）。验证：`users` 中目标老师记录 `phone` 非空。✅ 2026-10-10 实测：user 12 `phone=13900010002` 非空（`cmd/linkwx` + 只读 SQL 双向确认）。⚠️ 同时发现遗留数据问题：user 11（家长）与 user 12（老师）的 `open_id` 逐字节相同（共用 AppID 时期残留），user 65（2026-10-05 开发遗留账号）占用新 AppID 的 openid。经用户确认，已清空 user 65 的 `open_id`（`UPDATE users SET open_id=NULL WHERE id=65`），user 11 保持原样不动
- [x] 6.2 管理员重发老师邀请码，老师端走 `weChatLoginWithInvite` 重绑本端新 openid（**复用现有路径，不写一次性迁移脚本**）。验证：老师登录命中**原有**账号，角色与班级信息不变，业务接口不再 403（对应 TC-NEW-ATH-0007）。✅ 2026-10-10 实测：`POST /admin/teachers/12/invitation` 重发邀请码 → 老师端（IDE B，CDP 9223）以新 openid + 邀请码调 `POST /auth/login/wechat` → 命中 **user 12**（非新建），邀请码转 `status=2`；库里 `user 12.open_id` 变为新值。邀请码 `bbbc7575b44c9cfc` 见 `_verify/kv-course.log` 摘要。**「角色与班级信息不变」实测**：`class_teachers(user=12, class=1, 小一班, assistant, status=1)` 与 `teacher_profiles(id=2, 王敏, status=1)` 在重绑前后均未变（未合并、未迁移）。**「不再 403」实测**：以登录写入的真实 token 调 `GET /api/v1/teacher/home/overview` → 200，`class_info={id:1, 小一班, assistant}`、`student_count=3`。截图 `_verify/teacher-6.2-workbench.png`
- [x] 6.3 验证家长端不受影响：家长登录仍命中原有账号，孩子绑定关系不变（对应 TC-NEW-ATH-0003 / 0004）。✅ 2026-10-10 实测：家长端（IDE A，CDP 9222）清 storage → 重载 → 自动登录命中 **user 61（俊彦爸爸，151****0727）**——**未**串到 user 11（家长）或任何老师账号。`GET /api/v1/parent/home/overview` → 200，`current_student=小俊彦(27, 中二班)`，`students=[小俊彦(27), 王小花(32)]`，两个孩子绑定完整。截图 `_verify/parent-6.3-home.png`
- [x] 6.4 【本组验证】对照 `specs/wechat-login/spec.md` 的 TC-NEW-ATH-0001/0002/0007/0008 逐条在真实登录链路上核对；**UI 交互必须用 `wechat_automator` 的 tap/input 或真机，不能用 MCP 导航替代**。✅ 2026-10-10 实测（本会话无 wechat MCP，改用同等路径的 CDP 工具：`scripts/evaluate.py` 在 appservice 上下文下发 `wx.login()`/`wx.request()`，与端上 `utils/login.js` 同一业务路径；`scripts/click.py` 派发真实鼠标事件序列）：
  - **0001（老师端首次建号）**：⚠️ **单元通过，E2E 本环境不可复现**——本模拟器微信账号在老师端的 openid 已固定绑到 user 12，`wx.login` 换出的恒为同一 openid，无法再造「该端从未登录过」前提。需换一个未绑定过的微信号才能验，见 `_verify/` 验收记录
  - **0002（再次登录命中本端）**：✅ 两次连续登录均命中 user 12，不新建、不命中另一端
  - **0007（邀请码绑定预创建教师）**：✅ 见 6.2，命中原有账号、角色/班级不变、不新建 User
  - **0008（邀请码无效/已使用）**：✅ 已使用 → `400/40020`；无效码 → `400/40020`；两次均未签发令牌、未新建 User。附带实证 TC-0006：同一 code 二用被微信拒 → `400/40002`，未签发令牌
  - 全量逐条结论见 `_verify/split-wechat-appid-identity-acceptance.md`

## 7. 平台配置（人工运维）

- [x] 7.1 两个 AppID 各自在「开发管理 → 服务器域名」配置 request 合法域名白名单。验证：真机上不勾「不校验合法域名」也能发起登录请求。⏭️ **本阶段不做（用户决定 2026-10-11）**：当前只在本地（IDE 模拟器）验证，模拟器勾选「不校验合法域名」即可，白名单非必须。**上线/真机前必须补做**
- [x] 7.2 把开发者微信加为两个小程序的开发者。验证：真机预览/体验版可进入两端。✅ 用户确认已完成（2026-10-11）
- [x] 7.3 【本组验证】两端真机（或体验版）均可完成微信登录拿到令牌。⏭️ **范围改判为本地（用户决定 2026-10-11）**：本地等价验证已完成——老师端经 6.2 真实链路登录拿令牌（`teacher_user.id=12`、`teacher_token` 已落盘）；家长端经 6.3 真实链路登录拿令牌（`parent_user.id=61`、`parent_token` 已落盘）。**真机/体验版的三件事（域名白名单生效、真机登录、体验版进入）留待上线前补验**

## 8. 验收与收工

- [x] 8.1 对照 `specs/wechat-login/spec.md` **8 个场景逐条**验收（TC-NEW-ATH-0001~0008），每条记录通过/失败。未验证不得宣布完成、不得 archive。✅ 2026-10-10 完成，记录见 `_verify/split-wechat-appid-identity-acceptance.md`。结论：0002/0003/0004/0006/0007/0008 **实测通过**；0005 Unit 通过；0001 **Unit 通过、E2E 受环境限制不可复现**（本模拟器微信号在老师端 openid 已固定，无法再造首登前提，需换未绑定微信号）
- [x] 8.2 对照 `specs/cross-end-identity/spec.md` **6 个场景逐条**验收（TC-NEW-ATH-0009~0014），每条记录通过/失败。✅ 2026-10-10 完成，记录同上。结论：0012/0013/0014 **通过**；0009/0010/0011 **Unit 通过、E2E 不可验**（老师端个人主体无 unionid；无同手机号的两端账号对）
- [x] 8.3 明确记录：老师端 `pages/webview/index` **不在验收范围**（个人主体不支持 `<web-view>`），白屏属已知限制而非本次失败项。✅ 已记入 `_verify/split-wechat-appid-identity-acceptance.md`「已知限制」第 5 条与 CLAUDE.md；本次未将其计入任何 TC 的通过/失败
- [x] 8.4 收工清理：按 PID 杀掉残留 uni watcher（`dev:mp-weixin` 是 watch 进程，`TaskStop` 只终止 shell 包装）
  ```powershell
  Get-CimInstance Win32_Process -Filter "Name='node.exe'" |
    Where-Object { $_.CommandLine -match "dev:mp-weixin|kindergarten\\(teacher|parent)\\node_modules" } |
    ForEach-Object { Stop-Process -Id $_.ProcessId -Force }
  ```
  验证：`Get-CimInstance` 复查无匹配进程，工作区无 watch 进程弄脏的产物

## 回滚预案（不作为任务，仅供应急）

1. 后端：保留旧 `WECHAT_APP_ID`/`WECHAT_SECRET` 的兼容读取即可退回共用 AppID
2. 老师端：AppID 改回 `wxdf98…` 并重新 `npm run dev:mp-weixin`
3. 注意：回滚后老师端 `OpenID` 又失效一次，需再次走 6.2 重绑
