# W8-A · 对抗性实现审查

- verdict: **PASS**
- score: 94/100
- 源码覆盖: `teacher/src/api/http.js`、`parent/src/api/http.js`（逐行阅读）；参考 `utils/auth.js`、`utils/pendingRoute.js`、两端 `pages/auth/login.vue`、`server/internal/handler/http/auth.go`、`server/internal/service/auth_service.go`、`server/internal/auth/refresh.go`
- 审查范围: diff + untracked + 构建配置（未改）+ HTTP 调用链 + 登录页回跳 + 后端 refresh 契约 + 真实 DevTools 运行时
- 基线: `9bd9d9d` → `38f8169`（W8-B 后端提交，仅 server/）；W8-A diff 仅两个 http.js，+244/−11
- 说明: 纯前端 HTTP 层切片，无 UI/样式/路由变更；brief 无设计页与空态文案

## DevTools 证据

- 构建: `cd teacher && npm run dev:mp-weixin` → `DONE Build complete`，0 error，dist 10:07:08 > src 09:56:19；`cd parent && npm run dev:mp-weixin` 同（dist 10:07:08 > src 09:56:30）；watcher 已按 CLAUDE.md 杀净。补充一次性产物构建：两端 `npm run build:mp-weixin` → exit 0。
- Token: `node scripts/check-tokens.mjs teacher|parent` → 各 3 violations（基线 3，均为既有 `CalendarEventCard.vue`），无新增。
- Secret scan: `node scripts/check-secrets.mjs --files teacher/src/api/http.js parent/src/api/http.js` → **0 findings**，exit 0。
- 正常态: 老师端工作台/成长页真实数据（overview/todos、growth logs/class 均 200）；家长端首页 overview/feeds/tasks 均 200，无刷新、无跳登录。
- 空态: N/A——brief 未规定空态文案，本片无 UI 改动。
- 错误态: 两端同时污染 access+refresh → 并发 401 共享 1 次 refresh（401）→ 会话键全清 → `pages/auth/login`；家长端 `pending_route` 正确保留深链。
- 交互（真实点击，非 MCP 导航）: `_verify/act.py tap .agree-row` → `is-checked`；`tap .login-btn` → `POST /auth/login/wechat` 200 → 定时器 `resumePendingRoute` 消费 `pending_route` → `reLaunch` 回 `pages/notice/detail?id=<真实 id>`，详情与 `/read` 均 200 并渲染正文。
- 单飞/轮换: 老师端 2 并发 401 → **恰好 1 次** `/auth/refresh` 200 → 2 次重放 200，access/refresh 哈希均变化；家长端 3 并发 401 → 1 次刷新 → 3 次重放 200，同样轮换。再污染新 access → 各再刷新 1 次成功（证明使用的是轮换后的 refresh）。
- 服务端一次性: 同一 refresh token 直接 POST 两次 → 第一次 `(200, code 0)`，第二次 `(401, code 40100)`。
- Console: 恢复会话后两端均 **0 error**（测试期间的 401/refresh 红条为故意注入的测试痕迹）。
- 路由: 失败跳转用 `uni.reLaunch` 到非 tab 的 `/pages/auth/login`（正确）；本片未新增 tabBar 跳转；老师端无 tabBar。
- 截图: 未尝试——team lead 明确要求避免 `wechat_screenshot`（易挂死），改用 CDP DOM/文本断言 + 真实触摸点击。

## P1 · 阻塞问题 (0)

无。

## P2 · 建议 (2)

### 1. [并发边界] 家长端迟到的 401 可能覆盖深链回跳
- 证据: `parent/src/api/http.js:70-75`，`failToLogin` 无条件 `rememberPendingRoute(currentRoutePath())`，只有 `reLaunch` 被 `redirecting` 去重。
- 影响: 极端时序下（已跳到登录页后仍有失败请求落地）`pending_route` 可能被写成 `/pages/auth/login`，覆盖原深链。运行时未复现：3 并发失败记录的是 `/pages/index/index`，深链场景记录的是详情路由；该调用本身是 brief 明确要求的。
- 建议: 若后续复现，用同一 `redirecting` 标志把路由记录也限制为每批一次。

### 2. [覆盖缺口] 家长端直连 `uni.uploadFile` 未走静默续期
- 证据: `parent/src/pages/leave/create.vue:248`、`parent/src/pages/tasks/detail.vue:320`；家长端 `http.js` 无 `uploadFile()` 辅助函数，且这两个文件在 Allowed files 之外。
- 建议: 后续切片抽一个与 teacher 同契约的上传辅助函数；本片按 brief 范围豁免。

## 结论

PASS：改动严格限定两个 http.js；单飞、轮换落盘、重放一次、重放 401 不再刷新、非 auth 不续期、失败清理并跳登录、家长深链回跳均在真实 DevTools 中观测到；后端一次性消费已实测；build/token/secret/console 全绿。2 条 P2 不阻塞交付。
