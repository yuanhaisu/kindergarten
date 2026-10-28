# 本项目 Validator 检查表

> 通用模板见 `.claude/skills/slice-delivery/references/validators.md`。
> 本文件把模板实例化为本仓库的页面、命令和证据。worker 复制一份到 `.claude/runs/<slice-id>/validators.md` 填写；supervisor 独立重跑，不信 worker 的勾。

## 通用门禁

未勾项 = `FAIL`，除非用户在 `PROGRESS.md` 明确豁免并写明理由和后续切片 ID。

每条证据只写「命令 + 退出码 + 关键输出摘要」或「file:line」，不贴全量日志。

---

## A. UI 页面

适用：P01–P12、T01–T10、X01–X02 中涉及页面渲染的切片。

- [ ] 路由存在于 `pages.json`；未新增页面。
- [ ] 目标端确认：老师端探 `pages/more/index`，家长端探 `pages/points/index`；显式传 `project_path`。
- [ ] 源码最后一次修改后重跑 `npm run dev:mp-weixin`；`dist/dev/mp-weixin` mtime 晚于源码。
- [ ] 构建退出码 0。
- [ ] `node scripts/check-tokens.mjs <end>` 无新增违规。
- [ ] WXML 无内联 SVG；图标走资产/iconfont。
- [ ] 无 `color-mix`、`dvh`、`aspect-ratio`、`sticky`、`backdrop-filter`、`scroll-snap` 等未替换用法。
- [ ] 正常态：真实数据渲染，结构/文案符合设计。
- [ ] 空态：显示规定文案，不是空白。
- [ ] 错误态：显示错误与重试；重试可恢复。
- [ ] 每个声明的交互都用 `wechat_automator` / `_verify/act.py` 真实执行。
- [ ] Console 0 error。
- [ ] 家长端 tab 页用 `switchTab`；老师端无 tabBar，不存在 `switchTab`。
- [ ] 与设计稿的差异清单（字号/间距/色值/圆角/图标）已记录；P1 差异已修。
- [ ] 本页设计审查的 P1 已解决，或用户批准延期。

### 证据格式

```markdown
- 构建: `cd parent && npm run dev:mp-weixin` → exit 0；dist mtime 2026-10-28T...
- Token: `node scripts/check-tokens.mjs parent` → 0 violations
- 正常态: 数据 <来源>；页面显示 ...
- 空态: 方法 <...>；显示 ...
- 错误态: 方法 <failure-build / stop-8080>；显示 ...；重试后恢复 ...
- 交互: <tap 入口> → <页面/状态>；<返回> → <列表刷新>
- Console: 0 error
- 路由: <switchTab / navigateTo>
- 截图: <成功 / 失败原因 + DOM/CSS 断言>
```

---

## B. 后端

适用：B01–B04。

- [ ] `cd server && go build ./...` 退出码 0。
- [ ] 触及包的定向 `go test` 退出码 0；没有测试的写明原因。
- [ ] 先改 `backend/api/` 契约，再改 handler/service。
- [ ] 路由或角色变化时同步 `server/internal/app/router.go` 与 Casbin policy。
- [ ] 新增字段/表已跑 `AutoMigrate`，或用 `docs/migrations/` SQL + `go run ./cmd/dbcheck <table>` 确认。
- [ ] 用真实请求验证响应形状。
- [ ] 受影响页面在 DevTools 中仍能渲染，新字段可见。
- [ ] OpenSpec：proposal 已批准、apply 完成；若切片即完整变更，已 archive。
- [ ] `node scripts/check-secrets.mjs --files <manifest>` 0 findings。

### 证据格式

```markdown
- Go build: `cd server && go build ./...` → exit 0
- Test: `go test ./internal/service/... -run TestX -count=1` → ok
- Contract: `backend/api/api-parent/...` 已更新，字段 ...
- Schema: `go run ./cmd/dbcheck growth_reports` → 列/索引 ...
- API: `curl ...` → code 0，data.<field> = ...
- 页面: 家长端 `pages/growth/report` 显示 ...
- OpenSpec: change `<id>` 已批准/归档
```

---

## C. 边界 / 资产

适用：X01–X02、图标资产化、F0 的图标部分。

- [ ] 图标是文件或 iconfont，WXML 无内联 SVG。
- [ ] 页面有明确入口，或已明确隐藏；没有死链。
- [ ] 平台限制已记录：老师端个人主体不能用 `<web-view>`；家长端需要业务域名。
- [ ] 空态/错误态有意义，不是白屏。
- [ ] 构建退出码 0；Console 0 error。
- [ ] 入口/隐藏决策已由用户批准。

### 证据格式

```markdown
- 入口: <从哪个页面进入 / 已隐藏的原因>
- 平台限制: <老师端个人主体 / 家长端域名>
- 空态: <显示文案>
- 构建: `<cmd>` → exit 0
- Console: 0 error
```

---

## D. F0 基线

适用：F0。

- [ ] `AppIcon` 覆盖约定图标集（工作台 8 入口 + 导航/状态）。
- [ ] `PageShell` / 安全区 / 底栏约定已文档化；teacher 工作台已采用；另外 4 个试点页记录根容器差异，不重构。
- [ ] `StateBlock` 覆盖 loading / empty / error / retry。
- [ ] `CSS-COMPAT` 指南覆盖已知不支持特性及替代写法。
- [ ] `.gitignore` 包含 `.claude/runs/`、`.claude/locks/`，并放行 `.claude/skills/slice-delivery/`。
- [ ] `scripts/check-secrets.mjs` 存在；干净树退出 0；临时伪造 secret 能非零退出且不打印密文。
- [ ] 5 个试点页（老师工作台、老师发通知、老师通知列表、家长成长报告、家长接送二维码）无回归。
- [ ] 两端 `npm run dev:mp-weixin` 退出 0；`check-tokens` 不高于基线（teacher 204 / parent 152）。
- [ ] 失败构建方案可行性结论（支持 / 不支持，及替代方案）。

### 证据格式

```markdown
- AppIcon: <路径>；覆盖 <N> 个图标
- PageShell: <路径>；teacher 工作台已引用；其他试点页根容器差异 <...>
- StateBlock: <四态验证结果>
- CSS-COMPAT: <文档路径>；覆盖 <...>
- .gitignore: <新增行>
- Secret scan: 干净树 exit 0；伪造 `sk-...` 文件 exit 1，输出 `file:line [openai-key]`
- 试点回归: 老师 01/03/23、家长 12/14，Console 0 error
- 两端构建: teacher exit 0；parent exit 0
- 失败构建: <可行 / 不可行 + 原因>
```
