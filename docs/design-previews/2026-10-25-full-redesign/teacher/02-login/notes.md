# 老师登录（老师端）

- **源码**：`teacher/src/pages/auth/login.vue`（另读 `teacher/src/components/BrandBlock.vue`、`AppCard.vue`、`AppButton.vue`、`teacher/src/utils/login.js`、`teacher/src/pages.json`）
- **设计系统**：`user:nurture-planet-local`，`surface=teacher`
- **外壳**：`_shared/shell.css` + `_shared/shell.js`（canonical shell），本页不定义任何 `.mp-*` 类
- **方向**：「推门」——一次呼吸的登录页。只剩三层：我是谁（紧凑品牌 hero）→ 我怎么进去（唯一主按钮）→ 我同意了什么（一行协议）。不做账号密码表单，因为源码里根本没有。

## 信息层级（从上到下）

1. **系统外壳**（共享 shell）：20px 状态栏（9:41 + 信号/wifi/电池）+ 44px 导航栏（「老师登录」居中，`pages.json` 的真实标题）+ 右上 87×32 胶囊。入口根页面，**无 `mp-back`**。
2. **品牌区**：内联 SVG 星球标记（暖橙渐变球 + 一道倾斜轨道 + 轨道小点 + 球面柔光）76px；「养成星球」20px/700；「幼儿园数字化管理平台」14px/400。
3. **登录卡**：白底 `--surface`、`--radius-card`、`--shadow-md`。
   - 错误条（仅失败时）：`--danger-wash` 浅红底，标题 14px/600 + 说明 12px，**文字用 `--fg` 而非红色**（源码注释：`#EF4444` 压 `#FEE2E2` 只有 3.1:1，不达正文 4.5:1，错误语义只由浅红底承担）。
   - 主按钮：`--accent-gradient` 胶囊，50px 高，左侧白色微信气泡图标，文案随状态变化。
   - 协议行：20px 方形勾选框 + 「我已阅读并同意《用户协议》和《隐私政策》」，整行可点，命中区 ≥44px。
4. **脚注（卡片外）**：「仅对幼儿园老师开放，需园方统一开通账号」，12px，像脚注不像警告。
5. **底部安全区**：`calc(28px + env(safe-area-inset-bottom))`，**没有 tabBar**（老师端无 tabBar，本页也不画 `mp-tabbar`）。

## 覆盖状态

登录页没有列表数据，四态按共享外壳的四个芯片接线，另有两个**由交互触发**的非标准态。视觉全部由 `.mp-app[data-state]` 属性选择器驱动，`styles.css` 里没有一行 JS 改样式的代码。

| 入口 | `data-state` | 表现 |
|---|---|---|
| 芯片「正常」 | `normal` | **已勾选、可登录**（由勾选动作进入，不是首屏）；无错误条，按钮「微信授权登录」 |
| 芯片「空态」 | `empty` | **首屏初值**：协议未勾选（本页唯一的"缺内容"形态）；按钮保持可点，但点了只提示先同意 |
| 交互：未勾选时点登录 | `unagreed` | 与 `empty` 同一套视觉；额外弹 toast「请先阅读并同意用户协议和隐私政策」，且不进入登录中 |
| 芯片「加载」 | `loading` | 按钮「登录中...」+ `opacity:.55` + `pointer-events:none`；勾选行同步降级。这是本页的**禁用态** |
| 交互：点登录 | `pending` | 与 `loading` 同一套视觉；1.4s 后回 `normal` 并 toast「登录成功」 |
| 芯片「失败」 | `error` | 卡片内常驻错误条「微信登录失败 / 请稍后重试」+ 按钮变「重新登录」；点它重走 `pending → normal` |

`normal / loading / pending / error` 视为已勾选；`empty / unagreed` 视为未勾选。`aria-checked`、`aria-disabled`、`tabindex`、按钮 `disabled` 由 `app.js` 的 `syncA11y()` 同步（视觉不靠它）。

## 外壳回归（2026-10-26）

批次把小程序外壳收敛成 `_shared/shell.css` + `_shared/shell.js`，本页做了纯外壳替换，**没有重做设计**。

- **容器**：`.app` → `.mp-app`（带 `data-state`）；`.screen` 保留为内容区自己的类，与 `.mp-main` 并存（`<main class="mp-main screen">`）。
- **状态栏 / 导航栏 / 胶囊**：删除页面自绘的 `.status-bar`、`.nav-bar`、`.nav-capsule`（含自绘 SVG），改用 `mp-statusbar`（shell.md 里的固定三枚 SVG）、`mp-navbar` + `mp-title`、`mp-capsule`。登录页是入口根页面，**没有 `mp-back`**。
- **演示条**：自绘 `.demo` / `.demo-panel` / `.demo-chip` 全部删除，改用 `mp-demo` + `mp-demo-chips` + `data-mp-state`。折叠与芯片选中交给 `shell.js`，`app.js` 不再绑定演示条，只监听 `mp:state` 同步 a11y 并清掉上一轮登录定时器。
- **四态改造**：原来用 JS 切 `state.view` 再改 class / `hidden` / `textContent`，现在全部改成属性选择器。按钮文案改成三个 `<span>`（idle / pending / retry），由 `[data-state]` 决定显示哪一个——这样芯片切换时文案也能纯 CSS 变化，不依赖 JS 跑过。
- **引入顺序**：`tokens.css` → `../../_shared/shell.css` → `styles.css`；`shell.js` 在 `app.js` 之前。
- **样式归属**：`styles.css` 里已无 `.mp-*` 定义（`od-audit` 的 shell 重定义检查通过），只剩 `.mp-app[data-state=…]` 状态选择器。
- **验证**：`od-check --width 390` → `problems: []`；CDP 真点 35 条断言全过（外壳存在、无 `mp-back`、无 `mp-tabbar`、六个状态、芯片 `aria-pressed`、按钮 `disabled` 与透明度、toast 文案），console 0 error；`od-audit` 本页 0 issue。

## token 映射

| 元素 | token |
|---|---|
| 页面底 | `--bg`（shell）+ 内容区 `--bg-gradient`（对齐源码 `$gradient-page`，渐变起点正是 `--bg`，与导航栏无接缝） |
| 卡片 | `--surface` + `--radius-card` + `--shadow-md` |
| 星球标记 | `--accent-soft` → `--accent` 线性渐变；轨道 `--accent-soft`；轨道点 `--accent-deep`；球面高光 `--accent-wash`（径向到透明）；背后柔光 `--accent-wash` |
| 主按钮 | `--accent-gradient`、`--on-brand`、`--radius-pill`、`--shadow-md`，按下 `scale(.98)` |
| 文字层级 | `--fg`（品牌名/错误标题/按钮标签）/ `--muted`（tagline、协议、脚注） |
| 错误 | `--danger-wash` 底 + `--fg` 文字 |
| 勾选框 | 未选 `--border` + `--surface`；已选 `--accent` 底 + `--on-brand` 对勾 |
| 圆角 | `--radius-pill`（按钮/胶囊/柔光）、`--radius-card`（卡片）、`--radius-lg`（错误条）、`--radius-sm`（勾选框，全页最小） |
| 字号 | `--text-3xl`（品牌名）/ `--text-lg`（按钮）/ `--text-md`（tagline、错误标题）/ `--text-sm`（协议、脚注） |
| 动效 | `--motion-fast` + `--ease` |

## 与现状的差异

- **首屏就是源码的「未勾选」初值**：源码 `const agreed = ref(false)`，必须主动勾选才能登录（`login.vue:11`、`:62`）。「空态」芯片画的就是这个首屏初值，「正常」芯片是勾选后的可登录态——与 `parent/02-login` 同一套芯片语义（正常＝已勾选、空态＝未勾选），区别只是本页首屏停在「空态」，parent 首屏停在「正常」。未勾选时点主按钮只 toast「请先阅读并同意用户协议和隐私政策」，不进入登录中，同意门禁与源码一致；首屏截图因此不会被当成「已同意」。（2026-10-27 评审 P1 修复，此前首屏误画成已勾选）
- **纯平暖白底 → 渐变底**：源码 `$gradient-page` 本身是渐变，原型第一版写成纯 `--bg`，整屏发死。改回 `--bg-gradient`，靠渐变起点与 `--bg` 同色避免接缝。
- **品牌区压到 76px 紧凑 hero**：登录页每天开十几次，不需要占半屏的标识。保留同一颗星球、同一字阶，与家长端 `BrandBlock` 同源。
- **脚注从 `--hint` 提到 `--muted`**：`--hint`(#94A3B8) 压在暖奶白(#f7f3ea) 上只有 **2.28:1**，12px 读不清；`--muted` 是 4.30:1。
- **错误态用卡片内常驻错误条**：这是源码已经做过的决定（1.5 秒消失的 toast 读不到失败原因），原型沿用，并把按钮文案切到「重新登录」。
- **去掉「已登录直接跳转」的隐藏分支**：源码 `onLoad` 里有 token 就 `reLaunch` 到工作台，不可见，原型不表现。

## 遗留 / 需要批次层面决策

1. **`--muted` 在 teacher `--bg` 上只有 4.30:1，略低于 AA 4.5**（`--fg` 9.4:1、`--fg-2` 8.9:1 都够）。这是 token 层面的问题，本批次所有把 `--muted` 压在页面底色上的说明文字都受影响。建议回 `shared-design-tokens/tokens.json` 把 teacher 的 `--bg` 提亮或 `--muted` 加深，而不是逐页打补丁。
2. **协议名仍是纯文本**：源码刻意不做死链（协议页未落地），原型同样不可点。协议页落地后本页要补两个跳转。
3. **`empty` 芯片在本页的含义需要产品确认**：登录页没有列表可"空"，我把它映射成「协议未勾选」，2026-10-27 P1 修复后它同时是本页首屏初值——它是本页唯一真实的"缺一项"形态，视觉上与交互触发的 `unagreed` 完全一致。如果批次希望登录页的「空态」另有所指，这里要重新定义。
4. **预览截图**：批次已把 `renderPreview()` 换成 `scripts/od-shot.mjs`（CDP `setDeviceMetricsOverride` + `captureScreenshot`），本页已用它重出，视口确认 `cssViewport=390 / cssWidth=390`，不再有 125% 缩放下的右侧裁切。
5. **批次 `index.html` 需重跑** `node scripts/od-gallery.mjs docs/design-previews/2026-10-25-full-redesign` 才会登记本页（我没动，避免与其他页并发写冲突）。
