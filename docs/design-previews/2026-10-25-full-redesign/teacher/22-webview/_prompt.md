你是幼儿园管理平台「养成星球」的资深移动端设计师。为老师端微信小程序做一个页面的高保真 HTML 原型。

## 唯一任务
页面：内嵌网页（源码：teacher/src/pages/webview/index.vue，`pages.json` 标题：详情）
只做一个方向，不要 A/B 变体，不要多页面。写完 index.html / styles.css / app.js 三个文件后立刻停止：不要读回自己的产物，不要自查循环，不要提问，不要联网。

## 这个页面真实的产品事实（照此设计，不要发明字段）

这是一个**纯容器页**：老师从别处通过 `uni.navigateTo` 进入，页面只有一个 `<web-view :src="src">`，`src` 来自 `onLoad` 的 `query.url`（`decodeURIComponent`）。

- **没有接口调用**，没有接口契约，没有任何业务字段。
- 源码只有两个分支：
  1. 有 `url` → 渲染 `<web-view src>`（真实网页由外部站点渲染）。
  2. 没有 `url` → 整页空态文案「未提供页面地址」。
- **老师端是个人主体小程序，`<web-view>` 在真机会白屏**（不是设计问题，是平台能力限制）。原型**不要**因此画一个白屏，而是表达「一个容器应有的三态」：加载中 / 加载失败 / 正常占位。
- **绝对不要伪造第三方网页内容**（不要画新闻正文、商品、表单、假网站导航）。正常态只用一块**中性占位区** + 一句说明。
- 源码里不存在的功能一律不画：分享、复制链接、在浏览器打开、外链跳转、收藏、刷新按钮之外的任何操作。本页唯一的容器动作是「重新加载」，仅用于承载加载/失败态的演示。

## 页面结构与要求（从上到下）

1. **导航栏**：共享 shell 的 `mp-navbar`，**有返回箭头**（本页由 `navigateTo` 进入，是子页），标题「详情」，右侧胶囊。老师端**没有 tabBar，不要画**。
2. **网页地址条（正常 / 加载 / 失败三态可见）**：一张 `var(--surface)` 卡片，`var(--radius-card)`，`var(--shadow-sm)`，内边距 12–14px。
   - 左：40×40 圆角链接图标（`var(--accent-wash)` 底 + `var(--accent-deep)` 描边 SVG）。
   - 中：上行 11px `var(--muted)` 小标签「网页地址」；下行 13px `var(--fg-2)` 显示示例地址 `https://example.com/kindergarten/notice`，**必须能折行/截断，不得横向溢出**（`overflow-wrap: anywhere`，最多两行）。
   - 右：44×44 的图标按钮「重新加载」（refresh SVG，`var(--accent-deep)`，`:active` 时 `var(--accent-wash)` 底），`aria-label="重新加载网页"`。
   - **禁用态**：加载中时该按钮禁用（`--hint` 颜色、透明度 0.5、`pointer-events: none`、`aria-disabled="true"`）。
3. **网页容器（主体）**：一块撑满剩余高度的卡片，`var(--surface)` 底，`var(--radius-card)`，1px `var(--border)` 描边 + `var(--shadow-sm)`，`overflow: hidden`。
   - **正常态**：容器中部一块**中性占位区**——浅底 `var(--bg)`、1.5px 虚线 `var(--border)` 的圆角内框，居中放：
     - 56×56 圆角图标（`var(--surface)` 底 + `var(--muted)` 的窗口/地球 SVG）
     - 标题「内嵌网页」（16px，600，`var(--fg)`）
     - 一句说明「此处由外部网页渲染，原型以中性占位块代替」（13px，`var(--muted)`，最宽 260px）
   - 容器底部一条细脚注（12px `var(--hint)`）：「网页内容由第三方站点提供」，右侧一个中性小胶囊「原型占位」。脚注与占位区之间用 1px `var(--border)` 分隔。
   - 容器整体最小高度：`min-height: clamp(360px, calc(100dvh - 230px), 720px)`，让页面看起来就是一个满屏容器。
4. **页面内边距**：`var(--page-pad-x)`；底部 `calc(24px + env(safe-area-inset-bottom))`，不被演示条遮挡。

## 三种数据态（`data-state` 驱动，演示条用共享 `mp-demo`，四个芯片：正常 / 空态 / 加载 / 失败）

- **正常态（默认）**：地址条 + 中性占位容器 + 脚注。
- **空态**：对应源码「未提供页面地址」分支。**隐藏地址条与容器**，整页居中：
  - 64×64 圆角图标（`var(--surface)` 底 + `var(--muted)` 的断链/窗口 SVG）
  - 标题「未提供页面地址」（18px，600，`var(--fg)`）
  - 说明「请从包含网页地址的入口进入本页」（13px，`var(--muted)`）
  - 主按钮「返回上一页」（`var(--fg)` 深底 + `var(--surface)` 反白，`var(--radius-pill)`，高 44px）
- **加载态**：地址条保留（重新加载按钮进入禁用态）＋ 容器内居中：一个小圆环 loading（CSS 旋转，边框 `var(--border)` + 顶部 `var(--accent)`，不要进度条）＋「正在加载网页…」（13px `var(--muted)`）＋ 下方 3 行骨架条（宽 78% / 92% / 60%，8px 高，`var(--border)` 与 `var(--surface)` 混合色，轻微呼吸动画；`prefers-reduced-motion` 下静止）。
- **失败态**：地址条保留 ＋ 容器内居中：
  - 64×64 图标（`var(--danger-wash)` 底 + `var(--danger)` 的感叹号/断网 SVG）
  - 标题「网页无法打开」（18px，600）
  - 说明「可能是网络异常，或当前小程序未开通内嵌网页能力」（13px `var(--muted)`，最宽 270px）
  - 主按钮「重新加载」＋ 其下文字按钮「返回上一页」（`var(--muted)`，44px 触控区）
- **禁用态**：见地址条「重新加载」按钮的加载中表现；这是本页唯一的可禁用控件。

## 示例数据
- 示例地址：`https://example.com/kindergarten/notice`（刻意用 example.com，表示占位，不指向真实站点）
- 时间：无（本页没有列表与时间字段）
- 不要出现孩子姓名、班级、统计数据等本页不存在的业务内容

## 视觉与结构硬约束
- `<html lang="zh-CN" data-surface="teacher">`，390px 手机视口，单列纵向流，不做桌面布局。
- **外壳必须原样使用共享 shell**：照抄下面给出的 `mp-statusbar` / `mp-time` / `mp-status-icons` / `mp-navbar` / `mp-back` / `mp-title` / `mp-capsule` / `mp-main` / `mp-demo` 标记与 SVG，不许自造状态栏，不许复制或改写 shell.css。引入顺序：`tokens.css` → `../../_shared/shell.css` → `styles.css`；末尾先 `<script src="../../_shared/shell.js"></script>` 再 `<script src="app.js"></script>`。
- **styles.css 里绝对不许出现任何 `.mp-*` 选择器或覆盖**（唯一例外是 `.mp-app[data-state="…"] .xxx` 状态选择器）。
- 背景 `var(--bg)`，卡片 `var(--surface)`，层级靠字重 + 字号（标题 600 / 正文 400 / 标签 500）；颜色只写 `var(--*)`，禁止裸 hex / rgb / hsl；阴影只用 `var(--shadow-*)`；无直角（最小 `var(--radius-sm)`，胶囊用 `var(--radius-pill)`，卡片 `var(--radius-card)`）。
- 暖色克制：品牌橙只出现在链接图标、重新加载按钮、loading 圆环；大面积保持中性，暖色面积 ≤ 10%。
- 触控目标 ≥ 44×44px；正文 ≥ 13px；`:active` 必须有反馈（缩放或压暗）；不要 hover 才可见的信息。
- **禁止对可能超宽的文本用未标注的 `white-space: nowrap`**；地址这类可能超宽的文本必须 `overflow-wrap: anywhere`。所有容器 `max-width: 100%`，页面不得横向滚动。
- 多列排布（如果有）用 `grid-template-columns: repeat(N, minmax(0, 1fr))`。
- 不用 `<select>`、alert、prompt；无 console 报错。

## 交互（app.js 必须全部实现，无 JS 报错）
- 点击地址条的「重新加载」：toast「正在重新加载网页」，切到加载态；约 1.2 秒后回到正常态并 toast「网页已重新加载」。切换时同步演示条芯片的 `aria-pressed`（注意共用 `shell.js` 也会写 `data-state`，页面自己切态时要同步芯片状态）。
- 失败态「重新加载」执行同样流程；「返回上一页」toast「返回上一页」。
- 空态「返回上一页」toast「返回上一页」。
- 导航栏返回箭头：toast「返回上一页」。
- 加载态下重新加载按钮必须真的不可点（`disabled` / `pointer-events: none`）。
- toast 用页面内元素（底部居中小胶囊，深底反白），不要 alert/prompt。

## 共享外壳标记（原样照抄，只允许改标题文字与删/留返回箭头）
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
tokens.css 已由流程提供，不要重写，不要覆盖。每个文件 < 20KB。不要引用任何外部资源。不要写 notes.md，不要写 README。
