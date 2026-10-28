# CSS 兼容手册（小程序 / WXSS）

> 适用：`teacher/`、`parent/` 的所有 `.vue` 样式。
> 设计稿（`docs/design-previews/...`）是浏览器原型，允许使用较新的 CSS；
> 落到小程序前必须按本表替换。**原型照抄 = 真机静默失效**（不报错、只是不生效）。

## 0. 先确认可用

以下写法在两端已在使用、可放心使用：

- Flexbox、基础 Grid（`grid-template-columns: repeat(...)`）
- CSS 自定义属性（`var(--color-*)`）与 `calc()`
- `rpx`、`100vh`、`env(safe-area-inset-bottom)`
- `transform` / `transition` / `@keyframes`
- `box-shadow`、`border-radius`、`-webkit-line-clamp`
- 伪元素 `::before` / `::after`（只放装饰，不放关键文案）

不在下表、但原型里出现的新特性，先查微信小程序 WXSS 文档；查不到就按「不支持」处理。

## 1. 替代写法速查

| 不支持的写法 | 小程序可用写法 | 注意点 |
| --- | --- | --- |
| `color-mix(in srgb, …)` | ① 预生成 token（首选）：`$color-brand-primary-bg`<br>② Sass 构建期混合：`rgba($color-brand-primary, 0.12)`<br>③ 半透明叠层：`background: rgba(255, 138, 61, 0.12)` | 不要用运行时 `color-mix`；半透明值若只此一处，注释色值来源。品牌色变化时 ②③ 会跟着变，① 需要重跑 token 构建。 |
| `100dvh` / `svh` / `lvh` | `100vh`；页面区域已经不含原生导航栏 | 键盘弹起时 `100vh` 不会缩小；输入页用 `adjust-position`、`cursor-spacing`，不要用 `dvh` 赌行为。需要真实窗口尺寸时用 `uni.getWindowInfo()`。 |
| `aspect-ratio: 16 / 9` | ① 固定 `width` + `height`（rpx）<br>② 百分比内边距：容器 `padding-top: 56.25%` + 子元素绝对定位<br>③ 图片用 `<image mode="widthFix">` + `width: 100%` | ② 的子元素必须 `position: absolute; top: 0; right: 0; bottom: 0; left: 0;`，不要用 `inset`。 |
| `position: sticky` | 普通流占位 + `position: fixed` 工具栏；或把工具栏放在 `scroll-view` 外 | 部分新基础库支持 sticky，但在 `scroll-view` 内、与原生组件同层时不可靠；项目统一按不支持处理。固定工具栏要让内容预留高度。 |
| `backdrop-filter: blur(…)` | 不透明底色 `$color-bg-card` + `$shadow-md`；遮罩用 `rgba(0,0,0,.45)` | `filter: blur()` 模糊的是元素自身，不是背后的内容，替代不了毛玻璃。弹层底部不要依赖半透明模糊来保证文字可读。 |
| `scroll-snap-type` / `scroll-snap-align` | ① 分页轮播用 `<swiper>`（`current`、`circular`、`previous-margin`）<br>② 横向列表用 `<scroll-view scroll-x>` + `scroll-into-view` + JS 索引 | `swiper` 有固定高度，内容高度不一时要测量；横向 `scroll-view` 需要 `white-space: nowrap` + 子项 `display: inline-block`（或 flex）。 |
| flex / grid 的 `gap` | 现代基础库支持；老 Android / iOS < 14.1 会**静默变成 0 间距**。关键间距用 `margin`（配合 `:last-child` / `+` 选择器）或 `justify-content: space-between` | 现有代码大量使用 `gap`，不要为它全局重写；新页面在「不能塌」的位置（表单行、按钮组、卡片网格）改用 margin 或显式 `padding`。 |
| 内层滚动：`overflow: auto/scroll` | `<scroll-view scroll-y>`，且必须有确定高度 | 见 §2。`view` 上的 `overflow: auto` 不可靠；嵌套滚动会与页面下拉刷新抢手势。 |
| 原生 `input` / `textarea` / `canvas` 上方盖层 | `cover-view` / `cover-image`；或弹层打开时 `v-if` 隐藏原生组件；或关闭键盘后再展示下拉层 | 见 §3。`z-index` 盖不住老基础库的原生组件。 |

## 2. 内层滚动（scroll-view）

模板：

```html
<view class="page">
  <view class="header" />
  <scroll-view class="body" scroll-y>
    <!-- 列表 -->
  </scroll-view>
  <view class="footer" />
</view>
```

```scss
.page {
  display: flex;
  flex-direction: column;
  height: 100vh;
  box-sizing: border-box;
}

.body {
  flex: 1;
  min-height: 0; // 关键：没有它，flex 子项不会收缩，scroll-view 会被内容撑开
}
```

注意点：

1. `scroll-view` 必须拿到确定高度（`height` 或 `flex: 1 + min-height: 0`），否则 `scroll-y` 不滚动。
2. 外层已经是页面滚动时，不要再套一层 `scroll-view`；页面级滚动优先。
3. 同时开 `enablePullDownRefresh` 与内层 `scroll-view` 会在顶部抢手势；二选一。用 `scroll-view` 时改用 `refresher-enabled`。
4. 顶部固定区放在 `scroll-view` **外面**，不要依赖 `sticky`。
5. `enhanced`、`bounces`、`paging-enabled` 只在较新基础库生效，不能作为唯一实现。

## 3. 原生组件层级（input / textarea / canvas）

原生组件由客户端渲染，不在 WXSS 的普通层级里；老基础库上 `z-index: 9999` 也盖不住。

替代做法（按优先级）：

1. **隐藏**：弹层/下拉/日期面板打开时 `v-if` 移除输入框或 canvas；关闭后再挂回。
2. **同层渲染**：新基础库对部分组件支持同层，但必须真机验证，不能只信 DevTools。
3. **cover-view**：需要覆盖在 `map` / `video` / `canvas` 上时用 `cover-view` / `cover-image`。它只支持有限样式（文本、背景、边框、圆角），复杂布局做不了。

输入相关：

- 键盘弹起遮挡底部按钮：用 `adjust-position`、`cursor-spacing`，不要靠 `dvh`。
- `textarea` 在弹层里时，弹层关闭前先 `uni.hideKeyboard()`。
- 自定义下拉框放在 `input` 下方时，先失焦再展开，或直接把输入框 `v-if` 掉。
- `canvas` 优先 `type="2d"`；截图/导出前不要把它藏在普通 `view` 下面。

## 4. 自查命令

改动前后各跑一次，命中项逐个确认是「已替换」还是「确实无影响」：

```bash
grep -rnE "color-mix|100dvh|[0-9]dvh|aspect-ratio|backdrop-filter|scroll-snap|position:[[:space:]]*sticky|inset:[[:space:]]*0" \
  teacher/src parent/src --include=*.vue

grep -rn "gap:" teacher/src parent/src --include=*.vue
```

第二条只用于人工判断：`gap` 本身可用，但出现在「塌了会错位」的布局里时改 margin。

## 5. 与设计稿的对应

原型里的写法 → 落地写法：

| 原型 | 落地 |
| --- | --- |
| `calc(100dvh - 64px)` | 原生导航栏已在页面区外，直接用 `100vh`；不要再减 64px |
| 半透明导航/毛玻璃 | 原生导航栏 + `navigationBarBackgroundColor`（见 `SHELL-DECISION.md`） |
| 原型里的 `position: sticky` 顶栏 | 页面级滚动 + `fixed` 顶栏；或 `scroll-view` 外置顶栏 |
| 原型里的分页横滑 | `swiper`，或 `scroll-view` + `scroll-into-view` |
