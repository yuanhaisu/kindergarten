# W8-C · 对抗性实现审查

- verdict: PASS
- score: 90/100
- 源码覆盖: `parent/src/pages/pickup/qrcode.vue`、`teacher/src/pages/album/index.vue`
- 审查范围: diff + untracked + build config + 调用链 + 路由 + 设计审查 P1/P2
- 基线: 6076d56 → 6076d56（HEAD 未变，无 commit）

## DevTools 证据

- 构建: `cd parent|teacher && npm run build:mp-weixin` → exit 0；`npm run dev:mp-weixin` 均 Build complete，
  `dist/dev/mp-weixin/app.js` mtime 10:43:52 / 10:44:06 > 源码 10:39:52；watcher 已清理
- Token: `node scripts/check-tokens.mjs parent|teacher` → 3/3（既有 CalendarEventCard 基线，无新增 violation）
- Secret scan: `node scripts/check-secrets.mjs --files <2 files>` → 0 findings
- 正常态（家长，appid wxdf989efff8796c53）: 临时指定 authorization 613 后，请求顺序
  `GET /parent/pickup/designation → GET /parent/pickup/qrcode`，QR 渲染「有效 / 小俊彦的接送二维码 / 333 爸爸 / 有效期至 10月31日 23:59 / 绘图区域」
- 空态（student 27 无今日指定）: 仅发 `designation`，**未发 `qrcode`**（无 422/红色请求）；
  DOM「还没有接送凭证 / 请先指定今日接走人，指定后即可获取二维码 / 去指定今日接走人」；
  真实点击 CTA → 页面栈 `pickup/qrcode → pickup/authorized`
- 错误态: CDP mock 两接口失败 → designation 失败后回退 qrcode（两 URL 均发出），
  DOM「凭证加载失败 / 网络异常 / 重试 / 更换今日接走人 / 去指定今日接走人」
- 交互: 空态 CTA 真实点击 ✓；二维码页「刷新二维码」真实点击 ✓（经 auth/refresh 续期后重放，QR 保留）；
  错误态重试合成点击未派发（DevTools 限制，源码未改），直接调用 handler `e0_51` 可正常回退到空态
- Console: 0 error（家长 qrcode 空态、老师 album 空态）
- 路由: 两页均非 tabBar；家长二维码页 `navigateTo` 进入，无 switchTab 需求
- 截图: 未采集（`wechat_automator start` 两次失败：9420 未监听），以 DOM/CSS 断言替代并留真机备注
- 数据清理: 临时指定与测试 qrcode 已从 DB 删除，既有 qrcode is_active 已恢复；API `designated:false`

## P1 · 阻塞问题（0）

无。

## P2 · 建议（4）

### 1. [accuracy] self-review 称「两文件未出现在 token 报告里」，实际 album/index.vue:715 为既有 ℹ️ 信息行
- 证据: `sup-tokens-teacher.log` 含该行；violation 仍为基线 3
- 建议: 自述改为「无新增 violation」

### 2. [verification-gap] 错误态「重试」未能通过合成点击验证
- 证据: 错误态 `.primary-btn` mouse/emulated touch 均无请求；直接调用 `e0_51` 后 loadData 正常执行
- 建议: 真机或 automator 可用时补一次回归

### 3. [ux] 指定查询硬失败可能先弹一次共享错误 toast 再回退
- 证据: `parent/src/api/parent-pickup.js:69` 无 silent；worker 已披露
- 建议: 按 brief 保持现状

### 4. [artifacts] 设计工件仍保留旧文案
- 证据: `teacher/16-album-index/notes.md:19`、`index.html:214`
- 建议: 主会话决定是否同步；源码已正确
