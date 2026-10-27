# parent/19-tasks-wall · 对抗性单页审查

- verdict：**WARN**（score 70/100）
- 源码覆盖：源码覆盖基本完整：规则提示、加载/错误/空/正常/已有内容刷新失败五态、按 date 分段（今天/昨天/M月D日）、照片卡与一句话卡、可选正文、头像/孩子名/时间/任务名、图片预览与 noMore 墙脚都在；未演示 loadingMore 文案与真实头像 image 分支（前者已由 A26 记录，后者 notes 自认待素材），空态 CTA 与固定错误文案属已记录的 A26 提案。

## P1 · 明显缺陷（1）

### 1. [design-core] 日期条吸顶完全失效：页面滚到 600px 时「今天」日期条已滚出视口，与 notes 宣称的「日期当钉子、吸顶在导航栏下方」相反。

- 证据：styles.css:168-170 声明 position:sticky; top:64px；但 shell.css:133 的 .mp-main{overflow-x:hidden} 使其成为滚动容器，sticky 相对不滚动的 .mp-main 定位。Chrome 实测：window.scrollTo(0,600) 后 .wall-datebar.getBoundingClientRect().top = -472（预期 64）；把 .mp-main 的 overflow-x 临时改为 clip 后同一测量为 64。notes.md:13、notes.md:46 与 _prompt.md:36 都把吸顶当作已实现。
- 建议：把 shell.css 的 .mp-main 改为 overflow-x: clip（clip 不建立滚动容器，实测 sticky 恢复为 top 64），或让 .mp-main 成为真正的滚动容器并约束高度；改后必须在 400px+ 滚动位置真实测量日期条 top 恒为 64。

## P2 · 打磨项（4）

### 1. [logic] 照片预览层打开时不锁背景滚动，也未把背景设为 inert；遮罩上的滑动会带着背后的列表一起滚，关闭后页面位置已经变了。

- 证据：app.js:43-63 的 openPreview/closePreview 只切换 hidden 与焦点；实测预览打开时 document.body.style.overflow === '' 且 window 仍可滚动。同批次 parent/08-leave-detail/app.js:71、81 与 parent/13-pickup-index/app.js:156、167 都在弹层打开/关闭时成对设置/恢复 body overflow。
- 建议：打开预览时给 body 加 overflow:hidden（或给 .mp-app 加滚动锁）并给 .mp-main/.mp-navbar 加 inert，关闭时成对恢复；焦点回到触发卡已实现，保留。

### 2. [design-core] 多处小字号文字对比度低于 WCAG AA：主按钮白字压品牌橙渐变约 2.26–2.81:1，任务胶囊/分享计数文字 #EA580C 压浅底约 3.35–3.56:1。

- 证据：styles.css:709-725（.state-action 用 var(--accent-gradient) + var(--on-brand)）、styles.css:276-289（.task-chip）、styles.css:95-106（.wall-count）；tokens.css:20-24、97-101。实测计算：白字 vs #F97316=2.81:1、vs #FB923C=2.26:1；#EA580C vs #FFF7ED=3.35:1、vs #FFFFFF=3.56:1；11px/500 与 13px/600 均属正常字号，需 4.5:1。--hint 的 11px 时间/计数（实测约 2.45:1）属已记录的 token 缺口 3，不另计。
- 建议：主按钮改用更深的橙底（或深色字）使对比度 ≥4.5:1；胶囊/计数改用 --fg-2 或加深底；这属于批次级 token 决策，应在 tokens.json 调整后重跑三端产物，而不是逐页打补丁。

### 3. [design-core] 5 张照片占位全部铺品牌暖色渐变，整页暖色像素约 40.8%，与 brief 的「暖色面积 ≤10%」及本页「安静的墙面、白+冷雾底」方向明显不符。

- 证据：styles.css:415-571 五个 .wall-art[data-photo] 背景均含 --surface-warm/--accent-wash/--warning-wash/--accent-gradient-soft；5 张照片卡各为 4:3（实测 362×272），是 7 条内容的主体面积。对 preview.png（780×4936）按 r-b>14 且 r>200 统计，暖色像素占 40.8%。brief.md:60 要求暖色 ≤10%，_prompt.md:41 要求暖色只出现在规则胶囊底、日期竖条、一句话卡底、任务胶囊文字。
- 建议：照片占位改用中性灰或多色低饱和渐变，或把「照片内容不占暖色配额」的例外写进 notes 并让整页回到白+冷雾底；若保留暖色，至少把照片占位的暖色面积压到页面的 10% 以内。

### 4. [notes-accuracy] 两个「小雨」条目的头像字用了「雨」，与源码「名字首字」兜底（应为「小」）不符，notes 却声称与源码同逻辑。

- 证据：index.html:101、index.html:138 的头像 span 文本为「雨」，同卡 .child-name 为「小雨」；wall.vue:60-63 的 initial() 取 name.slice(0,1)；notes.md:58 写「示例数据里用名字首字兜底（与源码 avatar-fallback 同逻辑）」。
- 建议：把两处头像字改为「小」，或把 notes 改成「取名字末字」并说明这是设计改动；同时补一条带真实头像的示例，覆盖源码 wall.vue:153-156 的 avatar image 分支。

## 2026-10-27 P1 修复记录

### 1. [design-core] 日期条吸顶完全失效 — 已修

- 改法（页面内自洽，不动共享文件）：`index.html` 的 `<main class="mp-main">` 加本页自有类变成 `mp-main wall-main`；`styles.css` 新增 `.wall-main { overflow-x: clip }`。根因是 shell 的 `.mp-main{overflow-x:hidden}` 让 `overflow-y` 计算为 `auto`，形成「不滚动的滚动容器」，页内 sticky 全部相对它定位；`clip` 仍裁横向溢出但不建立滚动容器。这样也避开了 shell.md 第 8 条「页面 CSS 不得出现 `.mp-*` 选择器」。
- 实测（headless Chrome，390×900，文档高 2468）：`window.scrollTo(0,400 / 600 / 900)` 后 `.wall-datebar` 的 `top` 恒为 `64`，`.mp-navbar` 恒为 `20`；滚到 1400 时「今天」被自身段落推出（-364），「昨天」接棒在 `64` —— 每个日期条只在本段内吸顶，符合预期。
- 门禁：`od-check` → `problems: []`；`od-audit` → 本页 `issues: []`；`preview.png` 已按 390 / dsf 2 重出。
- 批次级附注：根因在 `_shared/shell.css` 的 `.mp-main`，本次仅在本页覆盖。其余含页内 sticky 子元素的页面需在 shell 层统一改为 `overflow-x: clip` 才能一并根治（属共享文件变更，本任务范围外）。

> P2 四项按本次任务要求保持原样，未改动。
