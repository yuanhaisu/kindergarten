# 家长登录（家长端）

- **源码**：`parent/src/pages/auth/login.vue`（另读 `parent/src/components/BrandBlock.vue`、`AppCard.vue`、`AppButton.vue`、`parent/src/utils/login.js`、`parent/src/pages.json`）
- **设计系统**：`user:nurture-planet-local`，`surface=parent`
- **外壳**：`_shared/shell.css` + `_shared/shell.js`（canonical shell）；`styles.css` 里没有任何 `.mp-*` 定义，只有 `.mp-app[data-state=…]` 状态选择器
- **方向**：「冷雾白上的一点暖橙」—— 家长端底色是冷白 `--bg`(#F8FAFC)（DESIGN.md「冷家长」），全页橙色只出现在星核、主按钮、勾选框三处。页面只回答一个问题：怎么进去。

## 信息层级（从上到下）

1. **系统外壳**（共享 shell）：20px 状态栏（9:41 + 信号/wifi/电池）+ 44px 导航栏（「家长登录」居中，`pages.json` 的真实标题）+ 右上 87×32 胶囊。**入口根页面，无 `mp-back`**——三个调用点（`parent/src/api/http.js:62`、`pages/messages/index.vue:49`、`pages/settings/index.vue:38`）全部是 `uni.reLaunch`；**无 `mp-tabbar`**，登录页不在 tabBar 四项里。
2. **品牌区**：内联 SVG 倾斜虚线轨道 + 轨道卫星点 + 38px 渐变星核（径向高光 + `--accent-gradient` + `--shadow-md`）；「养成星球」20px/700；「幼儿园数字化管理平台」14px/`--muted`。字阶与字重和 `teacher/02-login`、`BrandBlock.vue` 对齐（2026-10-08 评审 P1：两端并排不能像两个 App）。
3. **登录卡**：`--surface` 底、`--radius-card`(12px)、`--shadow-md`。
   - 错误条（仅失败时）：`--danger-wash` 浅红底 + 图标；标题 14px/600、说明 12px。文字用 `color-mix(--danger 72%, --fg)` ≈ #B43C41 而不是纯 `--danger`（3.08:1 不达正文 4.5:1），实测 4.69:1。语义由浅红底承担。
   - 主按钮：`--accent-gradient` 胶囊、48px 高、左侧白色微信气泡图标；文案随状态切换「微信一键登录 / 登录中... / 重新登录」。
   - 协议行：20px 圆角勾选框 + 「我已阅读并同意《用户协议》和《隐私政策》」，`<label>` 整行可点，`min-height: 48px`（≥44pt）。
4. **脚注（卡外）**：「登录后需绑定园所已开通的家长账号」，12px/`--muted`（#64748B 压 #F8FAFC = 4.55:1，达 AA）。
5. **底部安全区**：`calc(32px + env(safe-area-inset-bottom))`，**没有 tabBar**。

## 覆盖状态

登录页没有列表数据，四态按共享外壳的四个芯片接线，另有交互触发的同构状态。视觉全部由 `.mp-app[data-state]` 驱动，`app.js` 只改属性与文案，不改样式。

| 入口 | `data-state` | 表现 |
|---|---|---|
| 芯片「正常」 | `normal` | 已勾选、无错误条、按钮「微信一键登录」（可登录态） |
| 芯片「空态」 | `empty` | **协议未勾选**（本页唯一的"缺一项"形态），按钮保持可点；切到这颗芯片会顺带提示一次 |
| 交互：未勾选点登录 | `empty` | 与空态同视觉；勾选行抖动 + toast「请先阅读并同意用户协议和隐私政策」，**不进入**登录中 |
| 交互：取消 / 勾选协议 | `empty ↔ normal` | `change` 事件以 checkbox 为唯一真相源，状态与芯片 `aria-pressed` 同步；勾回时收起上一轮的 warning toast |
| 芯片「加载」 | `loading` | 按钮 `disabled` + `opacity:.55` + 转圈 + 「登录中...」；微信图标让位；勾选框 `opacity:.55` 且行内 `pointer-events:none`。这是本页的**禁用态** |
| 交互：点登录 | `loading → normal` | 1.4s 后回正常并 toast「登录成功」；若 `navigator.onLine === false` → `error` + 「网络异常 / 请检查网络后重试」 |
| 芯片「失败」 | `error` | 卡片内常驻错误条「微信登录失败 / 请稍后重试」+ 按钮「重新登录」；点按钮重走 `loading → normal` |

`normal / loading / error` 视为已勾选，`empty` 视为未勾选。`data-state` 由 `setState()` 单点写入，同时同步 `aria-busy`、按钮 `disabled`、checkbox `disabled` 与芯片 `aria-pressed`。

## 交付链验证

- `od-check --width 390` → `problems: []`（scrollWidth 390 = clientWidth 390，bodyHeight 900）
- CDP 交互探针 **11 条断言全过**：四个芯片、未勾选拦截（不进 loading）、重试→成功、勾选往返、`aria-pressed`、`disabled`、按钮/勾选框透明度、toast 文案与语气、标题居中、console 0 error
- `od-shot --width 390 --dsf 2` → `preview.png`（CSS 390×1000，图像 780×2000）
- `od-audit` → 本页 0 issue
- ⚠️ 本页**不可滚动**，`od-check` 的吸顶断言返回 `sticky: null`，这是预期结果，不是漏检；吸顶只有列表页才验证得到

生成过程：OpenDesign 第一次只回 `index.html` + `tokens.css`（缺 `styles.css` / `app.js`，页面裸奔），补写 `_prompt-short.md` 明确「必须交付三个文件」后重跑成功（约 467s）。随后人工修了 7 处：缺全局 reset（导致 3 个元素右边界 422/407/398）、导航栏里多余的 `nav-slot` 让标题右偏 22px、芯片文案改回 canonical「空态/加载」、正常态初始未勾选、取消勾选不更新状态、主按钮缺微信图标、品牌与协议字号对齐源码与 teacher 页。

## token 映射

| 元素 | token |
|---|---|
| 页面底 | `--bg`(#F8FAFC) + 共享 shell |
| 卡片 | `--surface` + `--radius-card` + `--shadow-md` |
| 星核 | `--accent-gradient` + 径向 `color-mix(--on-brand 55% → transparent)` + `--shadow-md`；轨道 `color-mix(--accent-soft 58%, --border)`；卫星 `--accent` |
| 主按钮 | `--accent-gradient`、`--on-brand`、`--radius-pill`、`--shadow-md`；按下 `translateY(1px)` + `brightness(.94)` |
| 文字层级 | `--fg`（品牌名、错误标题）/ `--fg-2`（协议、错误说明）/ `--muted`（副标题、脚注） |
| 错误 | `--danger-wash` 底 + `color-mix(--danger 72%, --fg)` 文字与图标 |
| 勾选框 | 未选 `--muted` 边 + `--surface`；已选 `--accent-deep`(#EA580C) 底 + `--on-brand` 对勾；focus 2px `--accent-deep` |
| 圆角 | `--radius-pill`（按钮/胶囊/演示条）、`--radius-card`（卡片）、`--radius-xl`（错误条/协议行/轻提示）、`--radius-md`（勾选框） |
| 字号 | `--text-3xl`(品牌名 20) / `--text-md`(副标题 14) / `--text-lg`(按钮 15) / `--text-md`(错误标题 14) / `--text-sm`(说明、协议、脚注 12) |
| 动效 | `--motion-fast` / `--motion-base` + `--ease` |

## 与现状的差异

- **登录页默认就是未勾选**（源码 `agreed = ref(false)`）。原型的「正常」芯片画的是勾选后的可登录态，用户第一次进入的真实样子在「空态」芯片里——与 `teacher/02-login` 同一约定。若批次希望首屏截图还原源码初值，把两颗芯片的含义对调即可，两套视觉都已存在。
- **错误条改横排左对齐 + 图标**：源码是居中两行文字；原型加图标、左对齐、收紧宽度，扫读更快。文字色沿用源码的决定（不用纯红），只是把红压深到 4.69:1。
- **toast 代替 `uni.showToast`**：未勾选提示与登录成功提示改页面内轻提示（可控语气色、不遮底部），文案完全沿用源码。
- **失败原因只留在卡片内**：与源码一致（1.5 秒消失的 toast 读不到失败原因、也没有第二次机会）。失败路径只有常驻错误条，不再叠加同文案 danger toast（2026-10-27 评审 P1：同屏重复且 `role=alert` 与 `role=status` 双重播报）。
- **logo 用内联 SVG 代替 `/static/logo.png`**：原型目录不放二进制资源；轨道 + 星核是标识的抽象转写，**不是最终 logo**。字阶/字重/tagline 已与 `BrandBlock.vue` 对齐。
- **省略「已登录直接跳转」隐藏分支**：源码 `onLoad` 有 token 就 `reLaunch`，不可见，原型不表现（与 teacher 原型一致）。

## 遗留 / 需要批次层面决策

1. **返回箭头与任务元数据冲突**：元数据写「本页由 navigateTo 进入的子页，画返回箭头」，每页需求写「入口页不画返回箭头」。源码三个调用点全是 `uni.reLaunch`、`pages.json` 无 tabBar，「入口页」成立，故**未画 `mp-back`**，与 `teacher/02-login` 一致。**需批次 owner 修正元数据**，否则后续页面会继续照错的那条画。
2. **主按钮白字压橙只有 2.80:1**（`--on-brand` #FFFFFF on `--accent` #F97316；渐变浅端 #FB923C 仅 2.26:1），15px/600 属正文级，不达 AA 4.5。这是 token 层问题（teacher 端同样），建议给品牌橙配深色 `on-brand` 或把渐变压深，不要逐页打补丁。
3. **`--muted` 压页面底 4.55:1**（parent 冷白）刚好过 AA；teacher 端同一 token 只有 4.30。跨端一致建议一并在 token 层解决。
4. **协议名仍是纯文本**：源码刻意不做死链（协议页未落地），原型同样不可点；协议页落地后本页要补两个跳转。
5. **「空态」含义需产品确认**：登录页没有列表可"空"，映射为「协议未勾选」（本页唯一真实的缺一项形态），与 teacher 原型同问。
6. **批次 `index.html` 需重跑** `node scripts/od-gallery.mjs docs/design-previews/2026-10-25-full-redesign` 才会登记本页（未动，避免与其他页并发写冲突）。
7. **预览截图**：用 `scripts/od-shot.mjs`（CDP `setDeviceMetricsOverride` + `captureScreenshot`），已确认 `cssViewport=390 / cssWidth=390`，无缩放裁切。
