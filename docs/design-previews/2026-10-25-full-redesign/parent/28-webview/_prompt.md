你是幼儿园管理平台「养成星球」的资深移动端设计师。为家长端微信小程序做一个页面的高保真 HTML 原型。

## 唯一任务
页面：内嵌网页（源码：parent/src/pages/webview/index.vue，`pages.json` 标题：详情）
只做一个方向，不要 A/B 变体，不要多页面。写完 index.html / styles.css / app.js 即停止：不读回产物，不自查循环，不提问，不联网。

## 真实产品事实（照此设计，不要发明字段）

纯容器页：家长经 `uni.navigateTo` 进入，页面只有一个 `<web-view :src="webviewUrl">`，`webviewUrl` 来自 `onLoad` 的 `options.url`（`decodeURIComponent`）。

- 没有接口调用、没有业务字段、没有列表/时间/孩子/班级信息。
- 源码仅两个分支：有 `url` → 渲染 `<web-view>`；无 `url` → 整页空态「**缺少页面地址**」（家长端原文，不要写成「未提供页面地址」）。
- 家长端是**非个人主体**小程序，`<web-view>` 可用。**不要**写「未开通内嵌网页能力」这类老师端限制。
- **绝对不要伪造第三方网页内容**（不画新闻正文、商品、表单、假导航）。正常态只用**中性占位区** + 一句说明。
- 源码不存在的功能一律不画：分享、复制链接、浏览器打开、收藏、外链跳转。唯一容器动作是「重新加载」，只为承载加载/失败态演示。

## 页面结构（从上到下）

1. **导航栏**：共享 `mp-navbar`，**必须有返回箭头**（子页），标题「详情」，右侧胶囊。**不画 tabBar**（本页不是家长端 4 个 tabBar 页之一）。
2. **来源条（正常/加载/失败可见）**：`var(--surface)` 卡片，`var(--radius-card)`，`var(--shadow-sm)`，左右 12px 内边距，外边距 `var(--page-pad-x)`。
   - 左：40×40 圆角方块（`var(--accent-wash)` 底 + `var(--accent-deep)` 地球/链接描边 SVG）。
   - 中（`flex:1; min-width:0`）：11px `var(--muted)`「外部网页」；13px 600 `var(--fg)` 域名 `example.com`；12px `var(--muted)` 完整地址 `https://example.com/kindergarten/notice`，最多两行，`overflow-wrap:anywhere`，不得横向溢出。
   - 右：44×44「重新加载」图标按钮（refresh SVG，`var(--accent-deep)`，`:active` 用 `var(--accent-wash)` 底），`aria-label="重新加载网页"`。
   - **禁用态**：加载中禁用（`--hint`、透明度 .5、`pointer-events:none`、`aria-disabled="true"`）。
3. **网页画布（主体，通栏）**：`var(--surface)` 底，占满剩余高度，左右不留边（真实 `<web-view>` 就是整块内容区），顶部 1px `var(--border)` 分隔线，`overflow:hidden`。
   - 内左右/上下各 16px 放**中性占位区**：`var(--bg)` 底、1.5px 虚线 `var(--border)`、`var(--radius-xl)`，居中：
     - 56×56 圆角图标（`var(--surface)` 底 + `1px var(--border)` + `var(--muted)` 窗口/地球 SVG）
     - 标题「外部网页内容」（16px，600，`var(--fg)`）
     - 说明「页面内容由第三方站点提供，此处以中性占位块代替」（13px `var(--muted)`，最宽 250px，居中）
   - 画布底部一行居中脚注：小锁/信息图标 + 12px `var(--hint)`「来源 example.com」，与占位区隔 12px。
   - 画布 `min-height: clamp(340px, calc(100dvh - 240px), 680px)`。
4. 内容区底部 `calc(24px + env(safe-area-inset-bottom))`，不被演示条遮挡。

## 四种状态（`data-state` 驱动，演示条用共享 `mp-demo`：正常/空态/加载/失败）

- **正常（默认）**：来源条 + 通栏画布 + 占位区 + 脚注。
- **空态**：对应「缺少页面地址」分支。**隐藏来源条与画布**，整页居中：64×64 图标（`var(--surface)` + `1px var(--border)` + `var(--muted)` 断链/空窗口 SVG）、标题「缺少页面地址」（18px 600）、说明「请从带有网页链接的入口进入本页」（13px `var(--muted)`）、主按钮「返回上一页」（`var(--fg)` 深底 + `var(--on-brand)` 反白，`var(--radius-pill)`，44px 高）。
- **加载**：来源条保留（刷新禁用）＋ 画布内居中：28px CSS 旋转圆环（2px `var(--border)`，顶边 `var(--accent)`，不用进度条）、「正在加载网页…」（13px `var(--muted)`）、3 行骨架条（78%/92%/60% 宽，8px 高，`var(--radius-pill)`，`var(--border)` 与 `var(--surface)` 的 `color-mix`，轻呼吸；`prefers-reduced-motion` 下静止）；容器 `aria-busy="true"`。
- **失败**：来源条保留 ＋ 画布内居中：64×64 图标（`var(--danger-wash)` 底 + `var(--danger)` 叹号/断网 SVG，无边框）、标题「网页无法访问」（18px 600）、说明「可能是网络不稳定，或链接已失效，请稍后重试」（13px `var(--muted)`，最宽 260px）、主按钮「重新加载」＋ 文字按钮「返回上一页」（`var(--muted)`，44px 触控区）。
- **禁用**：加载中刷新按钮不可点，这是本页唯一可禁用控件。

## 示例数据
域名 `example.com`；地址 `https://example.com/kindergarten/notice`（刻意占位，不指向真实站点）。本页没有孩子/班级/老师/时间字段，**不要出现**。

## 视觉与结构硬约束
- `<html lang="zh-CN" data-surface="parent">`，390px 视口，单列纵向流，不做桌面布局。
- **外壳原样用共享 shell**：照抄下方 `mp-*` 标记与 SVG，不自造状态栏，不复制/改编 shell.css。引入顺序 `tokens.css` → `../../_shared/shell.css` → `styles.css`；末尾先 `<script src="../../_shared/shell.js"></script>` 再 `<script src="app.js"></script>`。
- **styles.css 不许出现任何 `.mp-*` 选择器**（唯一例外 `.mp-app[data-state="…"] .xxx` 状态选择器）。
- 背景 `var(--bg)`，卡片/画布 `var(--surface)`；层级靠字重+字号（标题 600 / 正文 400 / 标签 500）。
- 颜色只写 `var(--*)`，**禁止裸 hex / rgb / rgba / hsl**；阴影只用 `var(--shadow-*)`；无直角（最小 `var(--radius-sm)`，胶囊 `var(--radius-pill)`，卡片 `var(--radius-card)`）。
- 暖色克制：橙只出现在图标底、刷新按钮、loading 圆环；暖色面积 ≤ 10%。
- 触控目标 ≥ 44×44px；正文 ≥ 13px；`:active` 必有反馈；信息不能只在 hover 可见。
- **禁止未标注的 `white-space: nowrap`**；地址必须 `overflow-wrap:anywhere`。所有容器 `max-width:100%`，不得横向滚动。
- 多列排布用 `grid-template-columns: repeat(N, minmax(0, 1fr))`。
- 不用 `<select>`、alert、prompt；无 console 报错；不引用任何外部资源。

## 交互（app.js 全实现，无 JS 报错）
- 点「重新加载」：toast「正在重新加载网页」→ 加载态；约 1.2 秒后回正常态并 toast「网页已重新加载」。切态时**同步演示条芯片 `aria-pressed`**（`shell.js` 也写 `data-state`）。
- 失败态「重新加载」同样流程；各「返回上一页」与导航栏返回箭头均 toast「返回上一页」。
- 加载中刷新必须真不可点，并清理定时器，避免连点叠加。
- toast 用页面内元素（底部居中小胶囊，深底反白，`var(--shadow-lg)`），不用 alert/prompt。

## 共享外壳标记（原样照抄，只可改标题文字、删/留返回箭头）
```html
<div class="mp-statusbar" aria-hidden="true">
  <span class="mp-time">9:41</span>
  <span class="mp-status-icons">
    <svg width="17" height="11" viewBox="0 0 17 11" fill="currentColor"><rect x="0" y="7" width="3" height="4" rx="1"/><rect x="4.5" y="5" width="3" height="6" rx="1"/><rect x="9" y="2.5" width="3" height="8.5" rx="1"/><rect x="13.5" y="0" width="3" height="11" rx="1"/></svg>
    <svg width="15" height="11" viewBox="0 0 15 11" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round"><path d="M1 3.6C2.8 2 5.3 1 7.5 1s4.7 1 6.5 2.6"/><path d="M3.5 6.1c1.1-1 2.4-1.5 4-1.5s2.9.5 4 1.5"/><path d="M6 8.6c.5-.4 1-.6 1.5-.6s1 .2 1.5.6"/><circle cx="7.5" cy="10" r=".7" fill="currentColor"/></svg>
    <svg width="24" height="11" viewBox="0 0 24 11" fill="none"><rect x=".5" y=".5" width="20" height="10" rx="3" stroke="currentColor" stroke-opacity=".5"/><rect x="2" y="2" width="15" height="7" rx="1.8" fill="currentColor"/><path d="M22 3.5v4c.9-.3 1.4-1 1.4-2s-.5-1.7-1.4-2Z" fill="currentColor" fill-opacity=".5"/></svg>
  </span>
</div>

<header class="mp-navbar">
  <button class="mp-back" id="backBtn" type="button" aria-label="返回">
    <svg width="20" height="20" viewBox="0 0 20 20" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M12.5 4.5 7 10l5.5 5.5"/></svg>
  </button>
  <h1 class="mp-title">详情</h1>
  <div class="mp-capsule" aria-hidden="true">
    <span class="mp-capsule-dots">···</span>
    <span class="mp-capsule-divider"></span>
    <span class="mp-capsule-ring">◎</span>
  </div>
</header>

<main class="mp-main"><!-- 页面内容 --></main>

<div class="mp-demo" id="demoPanel" data-open="false" style="bottom: calc(20px + env(safe-area-inset-bottom))">
  <div class="mp-demo-chips" role="group" aria-label="原型状态">
    <button class="mp-demo-chip" type="button" data-mp-state="normal" aria-pressed="true">正常</button>
    <button class="mp-demo-chip" type="button" data-mp-state="empty" aria-pressed="false">空态</button>
    <button class="mp-demo-chip" type="button" data-mp-state="loading" aria-pressed="false">加载</button>
    <button class="mp-demo-chip" type="button" data-mp-state="error" aria-pressed="false">失败</button>
  </div>
  <button class="mp-demo-toggle" type="button" aria-expanded="false">原型演示</button>
</div>
```

## 输出文件（项目根目录）
1. index.html —— 结构
2. styles.css —— 样式（引用 tokens.css）
3. app.js —— 交互
tokens.css 已由流程提供，不要重写/覆盖。每个文件 < 20KB。不引用外部资源。不写 notes.md、README。
