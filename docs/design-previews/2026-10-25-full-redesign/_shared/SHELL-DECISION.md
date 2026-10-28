# 外壳方案决策

> 2026-10-27 · 适用于老师端 + 家长端全部页面
> 结论：**采用微信原生导航栏，不做全量 custom nav，不实现原型里的假状态栏/假胶囊。**
> 状态：`pages.json` 导航背景已改（老师端 `#F7F3EA` / 家长端 `#F8FAFC`），两端 `build:mp-weixin` 已通过。

## 1. 决策

| 项 | 决定 |
|---|---|
| 导航栏 | `pages.json` 原生导航栏，保持 `navigationBarTitleText` |
| `navigationStyle` | 不设 `custom`（两端 51 页均不设） |
| 状态栏 | 系统渲染；不画 9:41 / 信号 / Wi-Fi / 电池 |
| 胶囊 | 微信原生；不画 `···` / `◎`，不尝试覆盖 |
| 返回 | 原生返回；根页面/登录页无返回，子页由系统显示 |
| 导航背景 | 老师端 `#F7F3EA`（`--bg`），家长端 `#F8FAFC`（`--bg`）；标题黑色 |
| 页面高度 | 原生导航栏已在页面区域之外；不再减 64px |
| 底部 | 家长端用原生 `custom-tab-bar`（见 §7）；老师端沿用 `BottomNav`；安全区照旧 |
| 下拉刷新 | 继续用 `enablePullDownRefresh`；原生导航栏不冲突 |

## 2. 为什么选原生导航栏

1. **设计稿的导航栏本来就是页面底色，不是渐变/透明。**
   `_shared/shell.css` 的 `.mp-navbar` 背景是 `var(--bg)`；51 页都不允许覆盖 `.mp-*`，所以没有一页依赖「导航栏渐变」「内容穿透状态栏」。
   - 老师端 `--bg` = `#F7F3EA`
   - 家长端 `--bg` = `#F8FAFC`
   - 把 `navigationBarBackgroundColor` 设为同一个值，导航栏和页面内容无接缝。

2. **渐变 hero 都在内容区，不在导航栏下面。**
   - `teacher/02-login` 明确注释「渐变起点正是 `--bg`，与上方导航栏无色差接缝」。
   - `teacher/01-workbench` 的问候区、`parent/01-home` 的孩子卡、`parent/27-profile` 的身份卡都在内容区。
   - 原生导航栏不会切掉这些渐变；不需要为了全幅渐变上 custom nav。

3. **custom nav 仍然画不出状态栏和胶囊。**
   它只能自绘标题/返回，胶囊仍是原生且必须避让；收益只有「标题 15px、返回箭头 44px」这类细节，代价是 51 页都要处理 `statusBarHeight`、`menuButtonBoundingClientRect`、页面高度、下拉刷新、PC 微信、页面栈返回。不划算。

4. **原生导航栏自带平台能力。**
   状态栏前景色、胶囊、返回、iOS 侧滑返回、下拉刷新、PC 微信窗口适配都由微信处理；custom nav 需要逐项补，且容易在刘海屏/灵动岛/Android 上出错。

5. **原型外壳是审阅用 mock，不是交付物。**
   `_shared/shell.md` 的「状态栏 20px、胶囊 87×32、不许改尺寸」只服务于 390px 预览；真机上这些尺寸本来就不成立。落地以外壳决策为准。

## 3. 接受的差异

| 设计稿 | 真机 |
|---|---|
| 20px 状态栏 + 44px 导航栏 + 87×32 胶囊 | 系统状态栏 + 原生导航栏 + 原生胶囊；高度随设备变 |
| 15px 居中标题 | 原生导航标题（约 17px），单行省略 |
| 44×44 自绘返回箭头 | 原生返回箭头 |
| 自绘胶囊 `···` + `◎` | 原生胶囊；不重叠、不伪造 |
| `calc(100dvh - 64px)` | 页面区域已不含导航栏；用 `100vh` / 页面滚动 |
| 导航栏与内容同色 | 通过 `navigationBarBackgroundColor` 对齐 `--bg`，基本同色；iOS 可能有原生细线 |

这些差异不影响信息架构、操作路径和内容设计；验收时不以「和原型顶部像素一致」为标准。

## 4. 实施规则

### 4.1 `pages.json`

- 两端均不新增 `navigationStyle: custom`。
- 老师端 `globalStyle`：
  - `navigationBarBackgroundColor`: `#F7F3EA`
  - `navigationBarTextStyle`: `black`
  - `backgroundColor`: `#F7F3EA`
- 家长端 `globalStyle`：
  - `navigationBarBackgroundColor`: `#F8FAFC`
  - `navigationBarTextStyle`: `black`
  - `backgroundColor`: `#F8FAFC`
- 保留每页 `navigationBarTitleText`；标题文案以 `pages.json` 为准，不在页面内再画 `mp-title`。
- 如果某页顶部内容区不是 `--bg`，只改该页 `navigationBarBackgroundColor` 为该页顶部颜色；不要因此引入 custom nav。

### 4.2 页面结构

- 删除/忽略原型中的 `.mp-statusbar`、`.mp-navbar`、`.mp-title`、`.mp-back`、`.mp-capsule`。
- 页面从内容区开始写，不再包一层 64px 假外壳。
- 页面滚动用原生页面滚动；需要内层滚动时用 `scroll-view`，不要再写 `calc(100dvh - 64px)`。
- 固定底栏/弹层保留，补 `padding-bottom: calc(高度 + env(safe-area-inset-bottom))`。
- 原生导航栏已处理顶部安全区；不要再加 `statusBarHeight` 的 20px 占位。

### 4.3 返回与分享

- 根页面（老师工作台、家长首页、登录页）不画返回；原生导航栏在页面栈为 1 时不会显示返回。
- 子页面由原生返回；不要自绘 `mp-back`，也不要拦截原生返回。
- 分享继续用 `onShareAppMessage` + `button open-type="share"`；不要依赖自绘胶囊菜单。

### 4.4 验收

- iPhone 刘海/灵动岛、iPhone 非刘海、Android 360/430、PC 微信：导航标题不换行、不与胶囊重叠、内容从导航栏下方开始。
- 页面顶部无 64px 空白/多减；底部无安全区遮挡。
- WXML 中不出现假状态栏、假胶囊、假返回。
- 下拉刷新、返回、分享、页面栈正常。

## 5. 不采用的方案

| 方案 | 不采用原因 |
|---|---|
| 全量 custom nav | 51 页成本高；仍画不出状态栏/胶囊；收益只有标题字号/返回箭头；刘海屏/PC 适配风险大 |
| 仅 hero 页 custom nav（hybrid） | 设计稿导航栏本来就是 `--bg`，没有全幅渐变需求；hybrid 会造成页面间导航栏高度/样式跳变，不如统一原生 |
| 保留假状态栏/假胶囊 | 真机上会与系统状态栏/微信胶囊重叠；不是可实现的 UI |

## 6. 后续如果反悔

只有出现以下情况才重新评估 custom nav：

- 产品明确要求导航栏渐变/透明/内容穿透状态栏，且接受 51 页改造与真机适配成本；
- 需要在导航栏内放自绘控件（原生导航栏放不下）；
- 微信原生导航栏在目标机型上出现无法接受的样式问题。

否则保持原生导航栏，不在单页偷偷加 `navigationStyle: custom`。

## 7. 2026-10-28 修订：家长端改用原生 custom-tab-bar

**决定**：家长端底栏从「每页挂 `<BottomNav>`」改为微信原生自定义 tabBar（`parent/src/custom-tab-bar/index.{js,json,wxml,wxss}`）；`pages.json` 保持 `tabBar.custom: true` 与 4 项 list 不变；4 个 tab 页（首页 / 成长 / 接送 / 我的）移除页内 `BottomNav`。老师端没有 tabBar，继续用 `BottomNav`。

**原因**：`custom: true` 但缺原生组件时，微信不渲染任何底栏；原先的 `custom-tab-bar/index.vue` 会被编译器**原样复制**进 dist（不编译），所以从未生效。页内 BottomNav 只能覆盖「碰巧写了它」的页面，每页各写一份也容易漏。

| 项 | 取值 |
|---|---|
| 组件位置 | `parent/src/custom-tab-bar/index.{js,json,wxml,wxss}`；编译器整目录原样复制到 dist，不走 uni/Vue 编译 |
| 选中态 | 各 tab 页 `onShow` 调 `utils/tabbar.js` 的 `syncTabBar(1/2/3)` 回写；首页默认 0，不额外同步。组件内不再用 `getCurrentPages()` 自算——切 tab 时栈顶可能还是旧页 |
| 切换 | 同 tab 直接 return；否则先对所有已创建的 tabBar 实例 `setData({ selected })` 再 `wx.switchTab({ url: "/" + pagePath })`（原生组件内没有 `uni`）。微信为每个 tab 页各建一个组件实例，只改当前页那份会让隐藏页保留旧高亮 |
| 高度 | 内容 78rpx（图标 48 + gap 4 + 文字 26）+ 上下 padding 48rpx ≈ **126rpx**，不含安全区 |
| 安全区 | `padding-bottom: calc(24rpx + env(safe-area-inset-bottom))` |
| 页面留白 | 统一 `calc(140rpx + env(safe-area-inset-bottom))`（126rpx 底栏 + 约 14rpx 余量） |
| 配色 | 白底 `#FFFFFF`，上边框 `rgba(106,77,42,0.08)`；未选中 `#94A3B8`，选中家长端品牌橙 `#F97316` |
| 图标 | remixicon 字体（码位同 BottomNav）；依赖 `App.vue` → `app.wxss` 的 `@font-face` + 组件 `styleIsolation: apply-shared`。**不要**在组件 WXSS 里 `@import` remixicon.css：WXSS 把 `.css` 的 `@import` 编译失败时会静默让所有 tab 页白屏 |

**遗留（P2）**：P01 首页 `PageShell bottom-space="180rpx"` 比统一值 140rpx 多 40rpx，是否收窄交给 P01 复审。
