# parent/28-webview · 对抗性单页审查

- verdict：**WARN**（score 72/100）
- 源码覆盖：源码两个分支（有 url 渲染 `<web-view>`、无 url 显示「缺少页面地址」）及返回箭头/标题「详情」均已覆盖；加载、失败、重试、来源条、脚注属 notes 已声明的设计提案，web-view 的 binderror/bindmessage、域名白名单与加载进度未体现（notes 已列为遗留）。

## P1 · 明显缺陷（1）

### 1. [logic] 来源条、画布与脚注被排成同一页面的上下层，但微信 `<web-view>` 会自动铺满整个页面并覆盖其他组件，真实落地时来源条与脚注必然被网页盖住，页面骨架不可实现；notes 把来源条当成可常驻方案，只讨论了与网页标题重复，没有记录这条平台约束。

- 证据：index.html:37-58 的 source-bar 与 index.html:60-112 的 web-canvas 为同层兄弟节点；源码 parent/src/pages/webview/index.vue:16 只有一个裸 `<web-view>`；notes.md:36-38 只写「重建 web-view（改 key 或重新赋值 src）」，notes.md:46 只讨论来源条与网页标题重复。微信文档：web-view「会自动铺满整个页面，并覆盖其他组件」。
- 建议：把来源信息前置到入口页或打开外链前的确认弹层，让目标页保持裸 web-view；若坚持同屏，只能改 cover-view 叠加并接受遮挡网页内容，并在 notes「遗留」写明该约束。

## P2 · 打磨项（4）

### 1. [redundancy] 同一「这是外部网页」的信息重复四层，其中域名 example.com 在一屏内出现三次（域名行、URL 内、脚注）；脚注没有提供来源条之外的新信息。

- 证据：index.html:47「外部网页」、index.html:73「外部网页内容」、index.html:74「页面内容由第三方站点提供…」、index.html:82「来源 example.com」；example.com 另见 index.html:48 与 index.html:49；preview.png 中三处同屏可见。
- 建议：只保留来源条一处来源说明，占位区收敛成一行中性文案，删掉画布脚注；若要保留脚注，换成不同信息（如「外部内容，请勿在此输入账号密码」）。

### 2. [design-core] 画布脚注 12px 文本用 `--hint`(#94A3B8) 压在白色画布上，实测对比度 2.56:1，低于 WCAG AA 4.5:1，也低于批次「正文 ≥13px」的下限，次要信息实际不可读。

- 证据：styles.css:274 `color: var(--hint)`、styles.css:275 `font-size: var(--text-sm)`；tokens.css:17 `--hint:#94A3B8`、tokens.css:40 `--text-sm:12px`、tokens.css:11 画布底 `--surface:#FFFFFF`；计算对比度 2.56:1。
- 建议：脚注改用 `var(--muted)`（约 4.76:1）并提到 13px，`--hint` 只留给禁用态。

### 3. [redundancy] 失败态同一「重新加载」动作有两个并列入口（来源条刷新图标 + 画布主按钮），触发后又同时出现两处加载反馈（toast「正在重新加载网页」与画布「正在加载网页…」），完成后再次 toast「网页已重新加载」，同一动作被表达三次。

- 证据：index.html:52-57 与 index.html:108 都绑定 app.js:105-106 的 startReload；app.js:82 与 index.html:89 的文案同屏出现；app.js:88 完成后再次 toast。
- 建议：失败态只保留画布内「重新加载」主按钮（顶部刷新在失败态隐藏），进行中的 toast 与画布 loading 二选一，仅保留完成提示。

### 4. [design-core] 脚注用「锁」图标注「来源」，锁的通行语义是 HTTPS/安全认证，会让家长误以为该外链已被平台核验；notes 自己把图标用途写作「标明内容归属」，图标与语义不匹配。

- 证据：index.html:77-83 为锁形 SVG（锁体 rect + 锁梁 path）；notes.md:12「锁图标 + 「来源 example.com」，标明内容归属」。
- 建议：换成链接/地球/info 图标，与「外部网页」来源条保持同一语义；若要表达安全，需用文案说明这是微信/平台的校验结果，而非本产品的背书。

## 2026-10-27 P1 修复记录

### P1-1 [logic] 来源条 / 画布 / 脚注同页堆叠，与 `<web-view>` 铺满覆盖冲突 —— 已修

走审查建议的第一条分支（来源信息前置，而不是 cover-view 叠加遮挡网页）：目标页保持裸 `web-view`，来源信息只出现在网页挂载之前。

- `index.html`：删掉页顶常驻 `source-bar`（含刷新按钮）与画布脚注 `canvas-footnote`；新增 `confirm-state`（`data-state="confirm"`）＝地球图标 + 「即将打开外部网页」+ 一句「该页面由第三方站点提供，请注意保护个人信息」+ 域名/完整地址卡 + 「继续打开」/「返回」。`web-canvas` 成为导航栏以下唯一内容；演示芯片新增「打开前」，共 5 个。
- `styles.css`：`.page` 去掉上下内边距，`.web-canvas` 改为填满 `100dvh - 64px`（= 状态栏 20 + 导航栏 44），对应真机 web-view 的占满行为；新增 `.confirm-state` / `.source-card` / `.state-icon--accent`，删除 `.source-bar` / `.source-icon` / `.source-copy` / `.source-label` / `.reload-button` / `.canvas-footnote` 及其 hover/禁用/动效规则。占位说明与确认页说明的 `max-width` 由 250/260 调到 300，避免新宽度下出现单字换行。
- `app.js`：`allowedStates` 增加 `confirm`；`startReload` → `openExternal`（确认/重试 → loading → 1.2s → normal，不再连发「正在重新加载 / 已重新加载」两条 toast）；`openBtn` / `confirmBackBtn` 接线，移除 `reloadBtn` 引用；加载中再点「继续打开」直接返回，切换芯片会 `cancelOpen()` 清掉计时器。
- `notes.md`：方向、信息层级、覆盖状态、与现状差异、遗留全部同步；遗留 1 明确写入平台约束——`<web-view>` 自动铺满整页并覆盖其他组件，除原生导航栏外不能常驻自有 UI，坚持同屏只能用 `cover-view` 叠加并接受遮挡。

**验证**

- `node scripts/od-check.mjs …/parent/28-webview/index.html --width 390` → `problems: []`、`overflowing: []`、`scrollWidth = clientWidth = 390`
- `node scripts/od-shot.mjs … --width 390 --dsf 2` 重出 `preview.png`（390×1000 CSS）
- `node scripts/od-audit.mjs docs/design-previews/2026-10-25-full-redesign` → 50 页全部通过，本页无 ✗
- headless Chrome（390×844）24 项断言全过：默认 `normal` 且画布顶边 64 / 高 780（正好填满导航栏以下）；确认态独占页面（画布 `display:none`）且各态均无横向溢出；点「继续打开」→ `loading` + `aria-busy=true` → 1.2s 后回 `normal`；确认态「返回」弹 toast「返回上一页」；失败态「重新加载」→ loading → normal；空态隐藏画布；切芯片会取消进行中的计时器；console 0 error。状态截图存 `_verify/28-webview/`。
- 视觉复核（390 视口截图）：确认页标题、说明各占一行不折行，地址卡左对齐，主/次按钮层级清楚；正常态画布从导航栏下沿铺到屏幕底部。

### P2 说明（未动）

四条 P2 全部依附于本次被移除的元素，随 P1 一并消失：P2-1 的第四层重复、P2-2 的脚注对比度、P2-4 的锁图标语义都针对该脚注，P2-3 的「两个重新加载入口」也随来源条一起不存在了。本轮没有做任何以 P2 为目标的独立改动——没有另调颜色、图标或新增文案来「顺便」修它们。

