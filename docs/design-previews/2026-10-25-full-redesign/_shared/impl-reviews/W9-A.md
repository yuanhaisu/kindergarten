# W9-A · 对抗性实现审查

- verdict: **PASS**
- score: 96/100
- 源码覆盖: `parent/src/api/http.js`、`parent/src/components/BottomNav.vue`（删除）、`teacher/16-album-index/{index.html,notes.md}`
- 审查范围: diff + 未跟踪文件 + manifest 对照 + 调用链（rawRequest / 401 / 登录页）+ 两端组件引用 + 设计工件文案
- 基线: `e9dc87b` → 工作区（未提交）。审查期间并发 W9-B 提交 `2658de4` 使 HEAD 前进，但不含本切片任何路径

## DevTools 证据

- 构建: `cd parent && npm run dev:mp-weixin` → `DONE Build complete`（ready 12553ms），`dist/dev/mp-weixin` mtime `2026-10-31 13:07:49` 晚于源码 `13:03:10`；watch 清理后包装退出 127 属预期。`cd parent && npm run build:mp-weixin` → exit 0
- Token: `node scripts/check-tokens.mjs parent` → 3 violations，与基线逐条一致（0 新增）
- Secret scan: `node scripts/check-secrets.mjs --files <manifest>` → 0 findings
- 正常态: 恢复真实会话后 `pages/index/index` 渲染真实数据（小俊彦 / 中二班 / 爸爸 / 2026-10-31 在校情况）
- 空态: `pages/points/index`（家长端独有探针）→「还没有积分记录 / 完成任务、坚持出勤都会获得积分 / 查看亲子任务」
- 错误态: mock `wx.login` 成功 + `wx.request` 401，直调页面编译产物 `handleLogin`（`e0_07`）→ 登录页 `pending_route` 保持为空；对照：401 延迟期间切到 `pages/index/index` → `pending_route` 写入该路径；调用计数 `{login:1, request:1}`
- 交互: 上述两次 401 分支均为 CDP appservice 直调页面事件处理器（DevTools 鼠标事件不触发 WXML invoker），非 `wechat_automator` 点击；本切片无可点击 UI 变更
- Console: 父端 target（host 56779，appservice + pageframe）ERROR_COUNT 0；`wechat_inspector` 聚合中的 1 条 401 来自老师端窗口 `/teacher/home/overview`
- 路由: `parent/src/pages.json` 无 diff；无新增 `switchTab/navigateTo`；删除组件无调用方
- 截图: `scripts/shot.py --title 家长端` 因 IDE 前台无法确认失败；以 DOM 文本断言替代（非视觉切片）
- 组件同步: 两端 `sync-ui.js` 均 `synced 9 components`，`shared-ui/` 基线起无 BottomNav；`parent/dist/{dev,build}` 均无 BottomNav 产物；教师端 `BottomNav.vue` 及 4 处 import 未动
- 会话恢复: token/refresh/user 写回、`wx.login/wx.request` 还原、home 真实数据复验通过；未打印 token

## P1 · 阻塞问题 (0)

无。

## P2 · 建议 (2)

### 1. [docs] `_prompt.md` 仍要求旧空态文案
- 证据: `docs/design-previews/2026-10-25-full-redesign/teacher/16-album-index/_prompt.md:74`
- 建议: 下次重生成该页前同步更新；历史 review/T06b 属存档，无需改。

### 2. [docs] 删除后仍有过期注释指向 BottomNav
- 证据: `parent/src/custom-tab-bar/index.js:3`、`index.wxss:10-11`
- 建议: 后续改为「与老师端 BottomNav 一致」或删除该句；基线时 shared-ui 即无该组件，属既有漂移，不在本切片 manifest。
