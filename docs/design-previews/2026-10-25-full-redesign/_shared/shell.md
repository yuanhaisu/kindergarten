# 小程序外壳规范（canonical shell）

> **本文件只描述原型预览壳，不是真机落地方案。**
> 真机外壳已定：两端全部使用微信原生导航栏，不实现这里的 20px 状态栏、87×32 胶囊和自绘返回。
> 落地规则见 [SHELL-DECISION.md](SHELL-DECISION.md)；本文件继续约束 51 页 HTML 原型的外观一致性。

> 补充 `brief.md` 第 2 节没定死的部分：外壳的**具体结构、尺寸、类名、复用方式**。
> 所有页面必须直接用这里的标记与类名，**不要在页面自己的 styles.css 里重写外壳**。

## 引入顺序

```html
<link rel="stylesheet" href="tokens.css">
<link rel="stylesheet" href="../../_shared/shell.css">
<link rel="stylesheet" href="styles.css">
```

`styles.css` 最后，页面只在里面写内容区样式；外壳类（`mp-*`）一律不加覆盖。

## 完整骨架

```html
<body>
  <div class="mp-app" data-state="normal">

    <!-- 状态栏 20px -->
    <div class="mp-statusbar" aria-hidden="true">
      <span class="mp-time">9:41</span>
      <span class="mp-status-icons">
        <svg width="17" height="11" viewBox="0 0 17 11" fill="currentColor"><rect x="0" y="7" width="3" height="4" rx="1"/><rect x="4.5" y="5" width="3" height="6" rx="1"/><rect x="9" y="2.5" width="3" height="8.5" rx="1"/><rect x="13.5" y="0" width="3" height="11" rx="1"/></svg>
        <svg width="15" height="11" viewBox="0 0 15 11" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round"><path d="M1 3.6C2.8 2 5.3 1 7.5 1s4.7 1 6.5 2.6"/><path d="M3.5 6.1c1.1-1 2.4-1.5 4-1.5s2.9.5 4 1.5"/><path d="M6 8.6c.5-.4 1-.6 1.5-.6s1 .2 1.5.6"/><circle cx="7.5" cy="10" r=".7" fill="currentColor"/></svg>
        <svg width="24" height="11" viewBox="0 0 24 11" fill="none"><rect x=".5" y=".5" width="20" height="10" rx="3" stroke="currentColor" stroke-opacity=".5"/><rect x="2" y="2" width="15" height="7" rx="1.8" fill="currentColor"/><path d="M22 3.5v4c.9-.3 1.4-1 1.4-2s-.5-1.7-1.4-2Z" fill="currentColor" fill-opacity=".5"/></svg>
      </span>
    </div>

    <!-- 导航栏 44px；子页保留 .mp-back，入口页/登录页删掉 -->
    <header class="mp-navbar">
      <button class="mp-back" type="button" aria-label="返回">
        <svg width="20" height="20" viewBox="0 0 20 20" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M12.5 4.5 7 10l5.5 5.5"/></svg>
      </button>
      <h1 class="mp-title">页面标题</h1>
      <div class="mp-capsule" aria-hidden="true">
        <span class="mp-capsule-dots">···</span>
        <span class="mp-capsule-divider"></span>
        <span class="mp-capsule-ring">◎</span>
      </div>
    </header>

    <main class="mp-main">…页面内容…</main>

    <!-- 家长端 4 个 tab 页；老师端 4 个常驻页（01 工作台 / 15 班级 / 18 消息 / 19 我的） -->
    <nav class="mp-tabbar" aria-label="主导航">
      <button class="mp-tab" aria-current="page">…图标 + 首页…</button>
      <button class="mp-tab">…成长…</button>
      <button class="mp-tab">…接送…</button>
      <button class="mp-tab">…我的…</button>
    </nav>
    <!-- 老师端消息项可加角标：<span class="mp-tab-icon">…svg…<span class="mp-tab-badge">3</span></span> -->
    <!-- 老师端底栏链接到 ../01-workbench/、../15-class-index/、../18-messages-index/、../19-profile-index/ -->

    <!-- 原型演示状态条，仅原型可见 -->
    <div class="mp-demo" data-open="false">
      <div class="mp-demo-chips" role="group" aria-label="原型状态">
        <button class="mp-demo-chip" data-mp-state="normal" aria-pressed="true">正常</button>
        <button class="mp-demo-chip" data-mp-state="empty" aria-pressed="false">空态</button>
        <button class="mp-demo-chip" data-mp-state="loading" aria-pressed="false">加载</button>
        <button class="mp-demo-chip" data-mp-state="error" aria-pressed="false">失败</button>
      </div>
      <button class="mp-demo-toggle" type="button" aria-expanded="false">原型演示</button>
    </div>

  </div>
  <script src="../../_shared/shell.js"></script>
</body>
```

## 状态约定

`shell.js` 只做两件事：折叠/展开演示条；点击 chip 时把 `data-state` 写到 `.mp-app` 上。

页面用属性选择器接管自己的四态，不需要写 JS：

```css
.mp-app[data-state="empty"] .feed { display: none; }
.mp-app[data-state="empty"] .empty { display: flex; }
.mp-app[data-state="loading"] .feed { display: none; }
.mp-app[data-state="loading"] .skeleton { display: block; }
.mp-app[data-state="error"] .feed,
.mp-app[data-state="error"] .skeleton { display: none; }
.mp-app[data-state="error"] .error { display: flex; }
```

加载/空/失败三块默认 `display:none`，只在对应 state 下出现。

## 铁律

1. **不许自造状态栏**：图标 SVG 用上面这份，尺寸 17×11 / 15×11 / 24×11。
   **不许把 shell.css 复制进页面目录、也不许改编它**：只写 `<link href="../../_shared/shell.css">`，共享文件是唯一真相源。
2. **不许改外壳尺寸**：状态栏 20px、导航栏 44px、胶囊 87×32、返回按钮 44×44。
3. **外壳吸顶**：状态栏 `top:0`、导航栏 `top:20px`，两者都是 `position: sticky` + 页面底色。页面不得给 `.mp-app` 加 `overflow`（会破坏 sticky），也不得给 `body` 设 `overflow:auto`。
4. **导航标题**单行省略，不换行；标题就是 `pages.json` 的 `navigationBarTitleText`。
5. **返回箭头**：入口页（老师工作台、家长首页、登录页）不画；由 `navigateTo` 进入的子页画。
6. **tabBar**：家长端只有首页/成长/接送/我的四个页面画；老师端只有 01-workbench / 15-class-index / 18-messages-index / 19-profile-index 四个常驻页画，其余页面不画。
7. **演示条**固定在手机框右下（`right: max(12px, calc(50vw - 183px))`），默认收起；有固定底栏时在 `mp-demo` 上写行内 `style="bottom: calc(底栏高度 + 12px + env(safe-area-inset-bottom))"`，这是唯一允许的行内外壳覆盖。
8. 页面 CSS 里出现 `.mp-*` 选择器即视为违规，唯一例外是 `[data-state]` 状态选择器。
9. `od-check` 会在可滚动页面上滚 400px 验证吸顶（statusbar=0、navbar=20），失败即 `problems` 非空。

## 已出稿页面的回归清单（1–5 页）

- [ ] 换成 `mp-statusbar / mp-navbar / mp-title / mp-capsule / mp-back`
- [ ] 页面自己的状态栏/导航样式删除
- [ ] 演示条换成 `mp-demo`，chip 用 `data-mp-state`
- [ ] `styles.css` 里不再有 `.mp-*` 定义
- [ ] `od-audit` 的 shell 检查通过
