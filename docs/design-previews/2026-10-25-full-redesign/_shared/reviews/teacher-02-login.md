# teacher/02-login · 对抗性单页审查

- verdict：**WARN**（score 76/100）
- 源码覆盖：源码可见结构（品牌区、卡片内错误条、主按钮三态文案、协议勾选行、卡外脚注）与加载/失败分支都已覆盖；缺的是登录成功后的 reLaunch 去向、错误态的多分支文案，以及首屏应呈现的未勾选初值（真实初值只藏在「空态」芯片里）。

## P1 · 明显缺陷（1）

### 1. [logic] 首屏 normal 态协议默认已勾选，与源码 agreed=ref(false) 的「主动勾选」相反；点「微信授权登录」会直接进入登录中并成功，源码里必须通过的同意门禁在默认路径上完全失效。

- 证据：teacher/src/pages/auth/login.vue:11 `const agreed = ref(false)`、login.vue:62 注释「主动勾选」；02-login/index.html:12 `data-state="normal"`、index.html:78 `aria-checked="true"`；02-login/app.js:22 `CHECKED` 含 normal、app.js:83-89 仅当不在 CHECKED 时才弹同意提示；preview.png 中勾选框为已勾选。notes.md:25 只写「勾选已选」，未登记该差异（对照 parent/02-login/notes.md:62 已明确登记并给出对调方案）。
- 建议：把 normal 初值改为未勾选（或对调 normal/empty 两枚芯片的语义），仅在勾选后才允许进入 pending；同时在本页 notes「与现状的差异」补一条，与 parent 登录页同口径说明，避免首屏截图被当成真实初值。

## P2 · 打磨项（5）

### 1. [notes-accuracy] notes 声称「保留同一颗星球…与家长端 BrandBlock 同源」，但源码 BrandBlock 渲染的是 `/static/logo.png`（蓝色星球+嫩芽+星星），原型画的是琥珀色带轨道 SVG；parent 登录 notes 明说「不是最终 logo」，本页没有这条声明。

- 证据：02-login/notes.md:64、notes.md:11；teacher/src/components/BrandBlock.vue:3 `<image class="brand-logo" src="/static/logo.png" mode="aspectFit" />`；teacher/src/static/logo.png（蓝底星球+嫩芽）；02-login/index.html:37-54（planet-gradient 琥珀星球与轨道）；parent/02-login/notes.md:66 已声明「轨道 + 星核是标识的抽象转写，不是最终 logo」。
- 建议：在 notes「与现状的差异」补一条同样的声明（内联 SVG 是抽象转写、不是最终 logo），或直接嵌入 logo.png 并保留 BrandBlock 的圆角+阴影规格；不要把抽象转写说成「同一颗星球」。

### 2. [navigation] 登录成功路径没有表现源码的 uni.reLaunch('/pages/index/index')：pending 结束回到 normal（按钮又变回「微信授权登录」），页面仍停在登录页只弹 2 秒 toast，状态机把成功当成了「回到初始态」。

- 证据：teacher/src/pages/auth/login.vue:28-29（toast 后 reLaunch 工作台）；02-login/app.js:89-94（setState('normal') + showToast('登录成功')）；02-login/notes.md:29（「1.4s 后回 normal 并 toast『登录成功』」）；目的地 teacher/01-workbench 在 rubric 原型清单中存在。
- 建议：增加一个成功终态（如整页「已登录，正在进入工作台」或跳转占位），或至少把 toast 改为「登录成功，正在进入工作台」并在 notes 标注原型不真实跳转。

### 3. [completeness] 错误态只有固定一条「微信登录失败 / 请稍后重试」，而源码 toLoginError 实际有 5 个可达分支（40020 邀请码无效、40021 邀请码关联账号不存在、40002 微信登录失败、网络异常、兜底 message），这些差异文案在原型里一个都没有。

- 证据：teacher/src/utils/login.js:55-73（五个 return 分支）；02-login/index.html:61-64（固定两条文案）；02-login/app.js 全文没有任何错误文案注入逻辑；02-login/notes.md:30 的失败态也只描述这一条。
- 建议：至少补一个差异化错误（如「邀请码无效或已过期 / 请向园方索取新的邀请码」或「网络异常 / 请检查网络后重试」），或在 notes 明确说明该芯片只代表 40002 分支、其余分支未覆盖。

### 4. [logic] 切换演示芯片/状态时只清登录定时器，不清 toast，上一状态的提示会滞留在新状态：例如「登录成功」toast 还在时切到「失败」，错误条与成功提示同屏；未勾选提示还在时切到「加载」，页面已禁用但 toast 还在催同意。

- 证据：02-login/app.js:74-77 的 mp:state 处理器只 clearLoginTimer + syncA11y；app.js:63-71 showToast 持有的 toastTimer 未被清理；_shared/shell.js:17-25 芯片点击直接改 data-state 并派发 mp:state。
- 建议：在 mp:state 处理器里同时移除 toast 的 is-visible 并 clearTimeout(toastTimer)，或把清 toast 收进 setState 统一处理。

### 5. [design-core] 主按钮白字压在 teacher 品牌橙渐变上，实测约 1.9–2.35:1，远低于正文 4.5:1；这是全批次 token 级问题（parent 登录 notes 已登记），但本页 notes 的对比度遗留只写了 --muted，没有提这个更严重的主 CTA。

- 证据：02-login/styles.css:167-169（background: var(--accent-gradient); color: var(--on-brand)）；02-login/tokens.css:89（teacher accent-gradient #ffa866→#ff8a3d）、tokens.css:18（--on-brand #FFFFFF）；parent/02-login/notes.md:72 已登记「白字压橙 2.80:1…teacher 端同样」；本页 notes.md:71 只讨论 --muted 的 4.30:1。
- 建议：回 shared-design-tokens/tokens.json 给品牌橙配深色 on-brand 或把渐变压深（token 层一次修复）；过渡期至少在本页 notes 登记继承该问题，避免被当成已达标。

## 2026-10-27 P1 修复记录

- **P1-1 [logic] 首屏协议默认已勾选、同意门禁在默认路径上失效 → 已修**：首屏 `data-state` 由 `normal` 改为 `empty`（未勾选，对齐源码 `agreed = ref(false)`），空态芯片改为默认选中、`aria-checked="false"`；「正常」芯片仍代表「已勾选、可登录」。未勾选时点主按钮只进 `unagreed` + toast，不进入 `pending`；勾选（→ `normal`）后才允许登录。notes.md 的覆盖表与「与现状的差异」已按 parent/02-login 同口径登记。
- 其余 P2 五项（logo 声明、成功态导航、错误分支、切态残留 toast、主按钮对比度）按要求未动。
- **批次级暂缓**：`parent/02-login` 首屏仍停在「正常」（已勾选），与本页首屏「空态」（未勾选）呈现相反；同套芯片语义（正常＝已勾选、空态＝未勾选）已一致，只是默认芯片不同。是否全批次统一首屏芯片需批次 owner 决定，本页无法单方面改 parent。

验证：`od-check --width 390` → `problems: []`；`od-shot --width 390 --dsf 2` 重出预览（首屏勾选框为未勾选）；`od-audit` 本页无 ✗；CDP 真点 22 条断言全过（首屏未勾选、未勾选点登录不进入 pending、勾选后可登录、四芯片状态互斥、console 0 error）。
