# 微信小程序可实现性审查

> 2026-10-27 · 范围：`docs/design-previews/2026-10-25-full-redesign/` 全部 51 页（老师端 23 / 家长端 28）
> 方法：逐页读 `index.html` / `styles.css` / `app.js` / `notes.md`，对照 `_shared/shell.css`、`_shared/shell.md`、两端 `pages.json`、源码与 uni-app 能力做静态判定。
> 未做 51 页真机逐页回归；CSS 兼容性结论按微信 WebView / 基础库能力判定，落地前仍需低端 Android、iOS 真机验证。

## 0. 结论

**按「功能与信息架构」口径：不是 51/51，是 49 可落地 + 1 条件落地 + 1 不可落地。**

| 结论 | 页数 | 页面 |
|---|---:|---|
| 完成通用适配后可落地 | 49 | 除老师端 22、家长端 28 之外的页面 |
| 条件落地 | 1 | 家长端 28 内嵌网页（非个人主体 + 业务域名） |
| 不可落地 | 1 | 老师端 22 内嵌网页（个人主体不支持 `<web-view>`） |

**按「像素级 1:1」口径：不能。** 主要不是页面内容画不出来，而是三类东西无法原样复刻：

1. **顶部外壳是原型模拟**：20px 状态栏、9:41、信号/Wi-Fi/电池、87×32 胶囊，都是 HTML/SVG 画的。小程序的状态栏和胶囊由系统/微信渲染，不能自绘；只能改用原生导航栏，或 `navigationStyle: custom` 自绘导航栏（胶囊仍然存在，不能画掉）。
2. **部分 CSS / 原生控件存在平台差异**：`color-mix()`、`aspect-ratio`、`dvh`、`scroll-snap`、`backdrop-filter`、`position: sticky`、内联 SVG、原生日期/搜索/多选框等，需要替换或降级。
3. **HTML/CSS/JS 不能直接搬进小程序**：51 页的 `app.js` 全部是 DOM 操作，必须重写成 Vue/WXML；内联 SVG 必须转成图片/iconfont/canvas。

一句话：**这套设计可以作为小程序的视觉与交互规范，但不能当作可直接移植的代码；49 页内容层可以落地，外壳层和少数控件必须按小程序重做。**

> 右下角「原型演示」状态条、四个演示芯片是原型工具，不属于产品 UI，不应实现；正常/空/加载/错误/禁用五态本身可以用小程序组件实现。

## 1. 判定口径

| 判定 | 含义 |
|---|---|
| 可落地 | 用小程序原生组件/属性可以做出等价功能与接近的视觉；仍需通用适配（外壳、SVG、px→rpx、CSS 回退） |
| 需改造 | 需要换组件、换 CSS 实现或自定义组件；视觉可能有差异 |
| 条件落地 | 依赖主体类型、业务域名、入口或后台配置 |
| 不可落地 | 当前主体/平台能力下无法实现设计中的正常态 |

## 2. P0：平台硬限制

### 2.1 外壳：51/51 页都不能照抄

`_shared/shell.md` 的「铁律 2：不许改外壳尺寸——状态栏 20px、导航栏 44px、胶囊 87×32」在真机上不成立：

- **状态栏**：系统渲染，不能画。iPhone 刘海/灵动岛机型高度约 44–59px，Android 20–30px，不是固定 20px。照抄 20px 会在真机上顶到/压住状态栏。
- **胶囊**：微信原生组件，始终在右上角，不能隐藏、不能替换图标。设计稿里的 `···` + `◎` 只是示意；如果自绘一个，会和真胶囊重叠。
- **导航栏**：用原生导航栏，标题/返回/胶囊都是系统样式，和设计的 15px 标题、44px 返回箭头不完全一致；用 `navigationStyle: custom` 可以自绘标题/返回，但胶囊仍在，必须用 `uni.getMenuButtonBoundingClientRect()` 避让。
- **渐变 hero**：设计稿的 `.mp-navbar` 背景本来就是 `var(--bg)`，51 页都不允许覆盖 `.mp-*`；渐变都在内容区，不在导航栏下面。原生导航栏设成同一个 `--bg` 即可无缝衔接，不需要为了渐变上 custom nav。
- **页面高度**：26 页写 `calc(100dvh - 64px)`，是在减原型自绘的 20+44 外壳。真机用原生导航栏时，页面区域本身已经不含导航栏，再减 64px 会多减；用 custom nav 时又要按 `statusBarHeight + navHeight` 重算。不能直接复制。

**已定方案：A. 原生导航栏 + 内容适配。** 详见 [SHELL-DECISION.md](SHELL-DECISION.md)。

- 两端 `pages.json` 保持原生导航栏，不设 `navigationStyle: custom`。
- 导航背景对齐 `--bg`：老师端 `#F7F3EA`，家长端 `#F8FAFC`；标题黑色。
- 删除原型里的假状态栏、假胶囊、假返回；页面从内容区开始。
- **不采用全量 custom nav**：51 页成本高，仍画不出系统状态栏/胶囊；设计稿本身也没有全幅渐变导航的需求。只有产品明确要求导航栏渐变/透明/自绘控件时才重新评估。

> 建议不要把「画一个假胶囊/假状态栏」列为验收项。它不是小程序能力，属于原型演示元素（和右下角「原型演示」芯片同类）。

### 2.2 老师端 22 内嵌网页：不可落地

- 老师端 AppID `wxbfb5f81a87f73a61` 是**个人主体**；微信 `<web-view>` 不支持个人主体小程序。
- 该页 `notes.md` 已自述「真实不可用；本原型只表达容器形态」，本审查确认该结论。
- 可落地的只有空态（未传 url）与错误态（未开通/加载失败）；「正常态显示外部网页」无法实现。
- 建议：**从本期验收范围移除**，或在入口处直接不出现该页；不要把它算进 51 页 KPI。

### 2.3 家长端 28 内嵌网页：条件落地

- 家长端 AppID `wxdf989efff8796c53` 是**非个人主体**，具备 `<web-view>` 主体条件。
- 仍需在微信公众平台配置**业务域名**（HTTPS、可校验），否则 `web-view` 打不开。
- `<web-view>` 会铺满导航栏以下整页并覆盖其他组件；设计稿已经正确规避：把来源信息放在「打开前确认」，网页挂载后只留裸 `web-view`。这个方案可以实现。
- 当前 `parent/src` 没有任何 `navigateTo` 指向 `pages/webview/index`（findings A38），页面做完也**没有入口**。需要产品先决定入口与参数。
- 结论：**页面本身可做，但需要域名 + 入口 + 产品确认，不能算无条件落地。**

### 2.4 tabBar：不是阻塞

- 家长端 `pages.json` 已经是 `"tabBar": { "custom": true }`，有 `parent/src/custom-tab-bar/index.vue`；设计的自绘 4 项底栏可以实现，图标换成图片/iconfont 即可。
- 老师端没有 `tabBar`，用 `teacher/src/components/BottomNav.vue` 固定底栏；设计稿的 4 项底栏也可以实现。当前源码用文字图标（首/班/消/我），要还原设计需要补图标资产。
- 唯一注意：老师端 4 个常驻页的底栏是 `position: fixed`，页面内容要留底部 padding，避免最后一条被遮住。

## 3. P1：通用改造清单

> 以下问题不阻塞「能不能做」，但决定「做完像不像设计稿」以及「老设备会不会坏」。

### 3.1 内联 SVG：51/51 页

- WXML **不支持** `<svg>` 标签；`<use href="#...">` 雪碧图也不能用（14 页使用）。
- 必须转成：PNG/WebP 图片、iconfont（项目已有 Remix Icon）、或 canvas（二维码等）。
- 设计里的星标、空态插画、装饰形状、状态图标、tabBar 图标都算在内。**这是最大的一项机械资产工作**，不是 CSS 小改。
- 复杂插画（登录页星球、空态轨道、照片占位抽象图）建议直接导出 PNG/WebP，不要用 CSS 拼。

### 3.2 CSS 兼容性：按页统计

| 特性 | 影响页数 | 风险 | 处理 |
|---|---:|---|---|
| `color-mix()` | 20 页 / 233 处 | **高**：iOS 16.2+ / Chromium 111+ 才支持；老设备整条声明失效，背景/边框/文字可能变透明 | 按 bright/deep/ink 三套主题预计算成静态 rgba/hex；或改用已有 token |
| `aspect-ratio` | 14 页 | 中：iOS 15+；老设备不生效，图片/方块高度塌陷 | 改用 `padding-top` 百分比占位，或固定 rpx 高度 |
| `dvh` | 26 页 | 中：iOS 15.4+；老设备不识别 | 用 `100vh` 兜底；或 `uni.getSystemInfoSync().windowHeight` |
| `flex gap` | 51 页均使用（flex/grid 混用） | 中：iOS <14.5 的 flex gap 不生效，间距全丢 | 关键布局改 margin；或设最低系统版本 |
| `position: sticky` | 5 页：teacher/07、09、14、18，parent/19 | 中：页面滚动大多可用，`scroll-view` 内不稳 | 优先 fixed + 占位；不要依赖 sticky 做核心交互 |
| `backdrop-filter` | 3 页：teacher/04、16，parent/25 | 低-中：iOS 可用，Android 兼容不一 | 降级为半透明纯色；不把可读性押在模糊上 |
| `filter: blur` | 3 页：teacher/04、16，parent/25 | 低：可用，但低端机性能 | 可用预模糊图片替代 |
| `scroll-snap` | 2 页：teacher/03、12 | 中：自绘滚轮依赖 scroll-snap，`scroll-view` 内不保证 | 用原生 `picker`，或自写 `scroll-view` + `scroll-into-view` 吸附 |
| `mix-blend-mode` | 1 页：teacher/03 | 低：装饰性色块 | 去掉或改普通透明度叠层 |
| `:is() / :where()` | 1 页 / 2 页 | 低：iOS 14+ | 拆成多条选择器 |
| `::placeholder` | 18 页 | 低-中：原生 input/textarea 上不可靠 | 用 `placeholder-class` / `placeholder-style` |
| `display: grid` | 51 页 | 低：现代 WebView 支持 | 低端 Android 回归；必要时降 flex |
| `max-width: 390px` | 18 页 | 低-中：430px 设备会两侧留白，360px 设备需确认 | 改为 100% / 430px 上限；按 360/390/414/430 回归 |
| `text-wrap: pretty` | 16 页 | 低：仅换行观感 | 忽略，用默认换行 |
| `overscroll-behavior` | 12 页 | 低：小程序支持不一 | 去掉，用 `catchtouchmove` 阻止穿透 |

**结论：** `color-mix`、`aspect-ratio`、`dvh`、flex gap 四项要作为「移植前必须替换」的硬清单；其余按页面降级即可。

### 3.3 滚动容器与固定底栏

- 21 页用 `.page { overflow-y: auto }` 做内层滚动；小程序里普通 `view` 的触摸滚动不如 `scroll-view` 稳，建议改页面滚动或 `scroll-view`。
- 48 页使用 `position: fixed` 的底栏/弹层/Toast。要补 `padding-bottom: calc(底栏高度 + env(safe-area-inset-bottom))`，并处理键盘顶起。
- 5 页有吸顶筛选/工具栏；改 `scroll-view` 后 sticky 可能失效，建议 fixed 头部 + 占位块。
- `teacher/07` 的底部操作栏是 `position: sticky; bottom: 0`，真机建议直接 fixed。

### 3.4 表单控件

| 设计稿 | 页数 | 小程序替换 |
|---|---:|---|
| `<input type="date">` | 1 页：teacher/09 | `<picker mode="date">`；原生 picker 外观与设计不同 |
| `<input type="search">` | 3 页：teacher/04、05、15 | `<input confirm-type="search">` |
| `<input type="tel">` | 2 页：parent/07、16 | `<input type="number">` + 正则校验；`inputmode` 不支持 |
| `<input type="checkbox">` | 3 页：teacher/06、09，parent/02 | 自定义 `view` 勾选框或 `<checkbox>` / `<switch>` |
| `<textarea>` | 12 页 | 原生 `textarea`；`auto-height`、`maxlength`、`placeholder-class` |
| 自定义开关 | 多页 | 用 `view` + `hover-class` 或原生 `switch`（原生样式不同） |
| 自定义滚轮 | teacher/03、12 | 原生 `picker`，或自写 `scroll-view` 滚轮 |

- `::placeholder` 在原生 input/textarea 上不要依赖，统一改 `placeholder-class`。
- 固定底栏 + 输入框的页面（teacher/03、05、09、12、21；parent/07、16、18、24）要处理键盘遮挡：`adjust-position`、`cursor-spacing`、`scroll-into-view`。

### 3.5 原生组件层级与弹层

- `input` / `textarea` / `canvas` / `video` 是原生组件；老基础库上可能盖住普通 `view` 的弹层/Toast。
- 设计里大量「底部弹层盖在表单上」：teacher/03、05、09、12、14、20、21；parent/07、16、24。落地时要：
  - 优先用新版基础库的同层渲染；
  - 或弹层打开时隐藏原生输入框；
  - 或使用 `cover-view` / `root-portal`（会牺牲样式自由度）。
- 二维码 `canvas` 上方不要放骨架/遮罩长时间覆盖；画完再显示。
- `web-view` 是最高层，任何自绘 chrome 都会被盖住；见 2.2/2.3。

### 3.6 单位与响应式

- 设计稿是 390px 固定视口，源码是 750rpx；所有 px 要转 rpx（1px ≈ 2rpx）或按 390 基准换算。
- `max-width: 390px` 的 shell/弹层在 430px 设备上会缩在中间；建议 shell 用 100%，弹层用 `max-width: 430px` 或 100%。
- 固定像素的二维码 200/240px、滚轮 220px、头像/图标等，按 rpx 或 canvas 实际像素分别处理。
- 要覆盖 360 / 390 / 414 / 430 宽度，以及 PC 微信窗口；不要只按 390 验收。

### 3.7 交互态与可访问性

- 设计稿大量 `:active` 按压反馈；WXML 的 `view` 更稳的是 `hover-class` / `hover-stay-time`，不要假设 `:active` 在所有组件上生效。
- `:hover` 只用于桌面预览；触屏没有 hover，不能把信息藏在 hover 里（brief 已要求）。
- 32 页有 `keydown` / focus 逻辑（Enter 提交、Esc 关闭、Tab 焦点环）；小程序触屏没有键盘，PC 微信也只有部分键盘能力。核心操作必须可点击完成，不能依赖键盘。
- `aria-*` / `role` 只有部分映射；视觉不受影响，但无障碍验收要按小程序能力重定。

### 3.8 原型 JS 不是可移植代码

51 页的 `app.js` 都是 `document.querySelector` + `addEventListener` + `classList`，必须用 Vue 响应式重写。具体替换：

| 原型用法 | 小程序替代 |
|---|---|
| `window.history` / `location.href` | `uni.navigateBack` / `uni.reLaunch` |
| `navigator.onLine` | `uni.getNetworkType` / `uni.onNetworkStatusChange` |
| `MutationObserver` | Vue `watch` / `nextTick` |
| `getBoundingClientRect` | `uni.createSelectorQuery().boundingClientRect` |
| `scrollTo` / `scrollIntoView` | `uni.pageScrollTo` / `scroll-view` 的 `scroll-top` |
| `requestAnimationFrame` | `nextTick` / `setTimeout` |
| `document` / `window` 事件 | 模板事件绑定 |
| `setInterval` / `setTimeout` | 可用，但页面卸载要清理 |

这不是「设计能不能实现」的问题，是「实现成本」的问题：**51 页相当于一次完整的前端重写，不是换皮。**

### 3.9 已有源码先例的能力

这些设计元素在现有源码里已经跑通过，不属于平台未知项：

| 能力 | 源码先例 |
|---|---|
| 扫码 | `teacher/src/pages/pickup/index.vue` 使用 `uni.scanCode` |
| 拍照/选图 | `teacher/src/pages/doctor/checkin.vue`、`teacher/src/pages/growth/create.vue`、`parent/src/pages/tasks/detail.vue` 使用 `uni.chooseImage` |
| 上传 | 源码使用 `uni.uploadFile`；设计里的百分比进度需在 wrapper 中暴露 `onProgressUpdate` |
| 二维码绘制 | `parent/src/pages/pickup/qrcode.vue` 用 `qrcode-generator` + `uni.createCanvasContext` 逐格绘制 |
| 图片预览 | 源码使用 `uni.previewImage` |
| 音频播放/录音 | 源码使用 `uni.createInnerAudioContext` / `uni.getRecorderManager` |
| 保存相册 | 源码使用 `uni.saveImageToPhotosAlbum` |
| 分享 | `teacher/src/pages/index/index.vue` 使用 `onShareAppMessage` + `open-type="share"` |
| 自定义 tabBar | `parent/src/custom-tab-bar/index.vue` + `pages.json` 的 `custom: true` |
| 老师端固定底栏 | `teacher/src/components/BottomNav.vue` |

因此二维码、扫码、上传、音频、分享、底栏这些不是「设计能不能实现」的问题；剩下的是外壳、CSS 兼容、原生表单/弹层和逐页工作量。

## 4. 逐页清单

> 所有页面都还要做通用适配（外壳、SVG、px→rpx、CSS 回退），下表只列**页面特有**的风险与工作。

### 老师端（23）

| 页面 | 结论 | 需重点处理 |
|---|---|---|
| 01 工作台 | 可 | 渐变 hero、分享 `open-type=share`、统计宫格、底栏 |
| 02 登录 | 可 | 自定义协议勾选、品牌 SVG 资产化、个人号静默登录 |
| 03 发布通知 | 需改造 | 自绘时间滚轮（scroll-snap）→ picker/自定义；附件上传；`mix-blend-mode` |
| 04 成长记录 | 需改造 | 搜索、筛选弹层、`backdrop-filter`、`color-mix`、分页 |
| 05 添加成长记录 | 需改造 | 九宫格上传/进度、幼儿搜索、`aspect-ratio`、视频是否支持待定 |
| 06 成长详情 | 需改造 | 图片预览左右滑、评论、勾选、底部弹层 |
| 07 接送管理 | 需改造 | `uni.scanCode`（源码已有）、底部 fixed 操作栏 |
| 08 接送记录 | 可 | 时间线、筛选、异常态 |
| 09 亲子任务 | 需改造 | 原生 date → picker、switch、搜索、表单 |
| 10 任务回应 | 需改造 | 单图上传、评论、提交态 |
| 11 班级日历 | 可 | 7 列月历网格、事件点 |
| 12 添加活动 | 需改造 | 自绘日期/时间滚轮（scroll-snap）→ picker/自定义 |
| 13 活动详情 | 可 | 详情、编辑/删除确认 |
| 14 请假审批 | 需改造 | 吸顶筛选、审批弹窗（可编辑原因）、列表 |
| 15 我的班级 | 需改造 | 学生搜索、宫格、统计 |
| 16 班级相册 | 需改造 | 相册宫格、上传入口、`backdrop-filter` |
| 17 更多功能 | 可 | 功能宫格、入口 |
| 18 消息 | 需改造 | 吸顶分类、未读角标、列表 |
| 19 我的 | 可 | 头像、统计、菜单、底栏 |
| 20 入园晨检 | 需改造 | 行内手风琴、三态 chips、拍照、textarea |
| 21 园级事件 | 需改造 | 月历 + 事件表单 |
| 22 内嵌网页 | **不可** | 个人主体不支持 `web-view`；只能空态/错误态 |
| 23 通知列表 | 可 | 列表/筛选/详情弹层；草稿附件与定时后端未落库 |

### 家长端（28）

| 页面 | 结论 | 需重点处理 |
|---|---|---|
| 01 首页 | 可 | 多孩切换、`getPhoneNumber` 绑定（非个人主体可用）、feed |
| 02 登录 | 可 | 协议勾选、品牌 SVG、静默登录 |
| 03 绑定孩子 | 需改造 | 邀请码表单；**入口缺失**（零入边） |
| 04 通知公告 | 可 | 列表、筛选、分页 |
| 05 通知详情 | 需改造 | 段落正文、附件、确认回执；后端字段缺（creator/category_label/attachments/read_at/confirmed_at） |
| 06 请假记录 | 可 | 统计、列表、状态 |
| 07 请假申请 | 需改造 | 日期/时间 picker、附件上传；`activeType` 不进 payload |
| 08 请假详情 | 可 | 时间线、撤销；`cancelled_at` 等字段缺 |
| 09 成长时光 | 需改造 | feed、照片、月份分组；后端聚合缺 |
| 10 成长详情 | 需改造 | 图片预览滑动、评论、点赞、fixed 输入栏；点赞不落库 |
| 11 成长相册 | 需改造 | 相册宫格；后端 `groups` 缺失会永远空态 |
| 12 成长报告 | 需改造 | 7 柱图表、周期切换；后端返回嵌套结构与页面扁平读取不一致 |
| 13 接送记录 | 可 | 接送状态卡、今日叙事 |
| 14 接送二维码 | 可 | `canvas` 二维码（源码已实现）、保存相册；`expire_at` 时区修正 |
| 15 授权接送人 | 需改造 | 列表/弹层；后端无停用/启用 API |
| 16 添加授权接送人 | 需改造 | 表单/手机号；关系枚举前后端不一致 |
| 17 亲子任务 | 可 | 列表、分段；`end_date`/`star_scope` 等字段未展示 |
| 18 任务详情 | 需改造 | 单图上传、语音播放；多图/截止日待产品确认 |
| 19 班级墙 | 需改造 | 吸顶日期条、卡片；无点赞/评论/任务筛选接口 |
| 20 积分与成就 | 需改造 | 进度条、榜单；history 字段与等级阈值缺 |
| 21 班级信息 | 可 | 班级信息；`title_label`/`phone` 后端未返回 |
| 22 班级日历 | 需改造 | 月历；请求未传 `start`/`end`，日历可能空 |
| 23 孩子信息 | 可 | 档案、晨检；部分字段源码未渲染 |
| 24 意见反馈 | 需改造 | 表单、图片上传；`category_label`/回复链路缺 |
| 25 设置 | 需改造 | 开关列表、`backdrop-filter`；隐私开关 API 缺 |
| 26 消息 | 可 | 消息列表；**入口缺失**、类型口径不全 |
| 27 我的 | 可 | 头像、统计、菜单、底栏 |
| 28 内嵌网页 | **条件** | 非个人主体可用；需业务域名、入口；`web-view` 全屏覆盖 |

## 5. 非平台阻塞（但会让「全部实现」落空）

### 5.1 四个页面零入边

即使平台能做，用户也到不了：

| 页面 | 状态 |
|---|---|
| 老师端 `/pages/webview/index` | 零入边；且平台不可用 |
| 家长端 `/pages/auth/bind` | 零入边 |
| 家长端 `/pages/messages/index` | 零入边 |
| 家长端 `/pages/webview/index` | 零入边 |

老师端 `/pages/notice/index` 已在 2026-10-27 补源码 + 注册，分享死链已修，不在上表。其余 4 页需要产品决定入口或从验收范围移除。

### 5.2 设计依赖、后端未返回

这与小程序无关，但会导致页面「做出来是空的」。详见 `_shared/findings.md` 的 A1–A42。影响最大的几条：

- **家长端 05 通知详情**：`creator`、`category_label`、`attachments`、`read_at`、`confirmed_at` 后端详情未返回，设计里 5 个区块会空。
- **家长端 11 成长相册**：前端读 `groups`，后端只回 `{list,total,year,month}`，真机永远空态。
- **家长端 12 成长报告**：后端 `{report, dimensions}` 嵌套，页面按扁平字段读，统计/维度拿不到。
- **家长端 20 积分**：`PointsHistory` 无 `type`/`amount`/`source_label`，等级阈值 1500 无接口来源。
- **家长端 21 班级信息**：契约有 `title_label`/`phone`，`ParentPortalService.ClassInfo` 未返回。
- **家长端 22 班级日历**：请求不传 `start`/`end`，月历只能空数据。
- **家长端 24 意见反馈**：`category_label` 未返回，园所回复无后台入口。
- **老师端 04 成长记录**：列表不带 `class`，选「全部班级」时分不清记录属于哪个班。
- **老师端 10 任务回应**：`/checkins` 只回 `student`，没有家长姓名/头像。
- **老师端 23 通知列表**：草稿编辑已接 `notice/create?id=`，但后端 `UpdateNotice` 不落 `attachments` / `scheduled_at`，编辑页只能隐藏这两块。

### 5.3 设计里新增、源码没有的能力

这些不是平台问题，但落地前要产品拍板，否则会出现「设计有、接口没有」：

- 家长端 07 请假申请：附件上传 UI（源码模板无入口，但 API 支持）。
- 家长端 15 授权接送人：停用/启用（源码/契约只有删除）。
- 家长端 19 班级墙：点赞/评论/按任务筛选（契约无）。
- 家长端 24 意见反馈：历史回声/回复（源码只有提交表单）。
- 老师端 05 成长记录：视频（源码 `chooseImage` 只选图）。
- 老师端 16 班级相册：上传入口（源码只有占位 `openUpload()`）。

## 6. 建议的落地方式

### 6.1 外壳策略（已定）

**采用原生导航栏 + 内容适配。** 不追求外壳像素级，接受原生标题/返回/胶囊；把设计稿的内容区、状态、交互做扎实。详见 [SHELL-DECISION.md](SHELL-DECISION.md)。

- 两端 `pages.json` 不设 `navigationStyle: custom`。
- 导航背景对齐 `--bg`：老师端 `#F7F3EA`，家长端 `#F8FAFC`（**2026-10-27 已改，两端 build 通过**）。
- 删除假状态栏/假胶囊/假返回；页面从内容区开始，不再减 64px。
- 不采用全量 custom nav；只有产品明确要求导航栏渐变/透明/自绘控件时才重新评估。

### 6.2 建一层「小程序移植层」

不要逐页从 HTML 翻译。先做共享组件与规范：

- `AppTabBar`（继续用现有 `BottomNav` / `custom-tab-bar`；导航栏用原生，不做 `AppNav`）
- `AppSheet` / `AppToast` / `AppDialog`
- `AppSkeleton` / `AppEmpty` / `AppError`
- `AppInput` / `AppTextarea` / `AppSwitch` / `AppPicker`
- `MediaUploader`（chooseImage/chooseMedia + 进度 + 失败重试 + 预览）
- `WheelPicker`（若坚持自绘滚轮）/ 直接 `picker`
- `MonthCalendar`（老师/家长日历共用）
- 图标资产层：SVG → PNG/WebP/iconfont；二维码走 canvas

### 6.3 先做 4–5 页试点

建议试点覆盖最高风险：

1. **老师端 01 工作台**：渐变 hero、分享、底栏、原生/custom nav 取舍。
2. **老师端 03 发布通知**：表单、附件上传、时间滚轮、textarea、底部固定栏。
3. **家长端 12 成长报告**：分栏/7 柱图、周期切换、数据形状。
4. **家长端 14 接送二维码**：canvas 二维码、保存相册、权限。
5. **家长端 28 内嵌网页**：业务域名、`web-view` 全屏、打开前确认。

试点通过后再批量推进，避免 51 页做完才发现外壳/表单/上传方案不对。

### 6.4 验收矩阵

- **设备**：iPhone 刘海/灵动岛、iPhone 非刘海、Android 360px 小屏、Android 430px 大屏、PC 微信。
- **系统**：至少覆盖一个 iOS 15 以下、一个低端 Android，专门验 `color-mix` / `aspect-ratio` / `dvh` / flex gap。
- **页面状态**：正常 / 空 / 加载 / 错误 / 禁用；长文案 Toast（不要用 `uni.showToast` 截断）。
- **表单**：键盘顶起、fixed 底栏、placeholder、日期/时间 picker、多行输入。
- **媒体**：相册/拍照、上传进度、失败重试、预览、二维码保存、附件打开。
- **导航**：老师端 `goTab` 不涨页面栈；家长端 `switchTab`；4 个零入边页是否保留。
- **外壳**：真机状态栏/胶囊不重叠；底部安全区；无假状态栏/假胶囊。

## 7. 最终判断

| 问题 | 答案 |
|---|---|
| 51 页内容都能在小程序里做出来吗？ | 49 页可以；家长端 28 条件可以；老师端 22 不可以。 |
| 能 1:1 还原设计稿吗？ | 不能。外壳（状态栏/胶囊）、原生 picker/表单、部分 CSS 必然有差异。 |
| 能直接把 HTML/CSS/JS 搬进小程序吗？ | 不能。必须重写 Vue/WXML，SVG 资产化，CSS 回退。 |
| 最大的风险是什么？ | 不是页面画不出来，而是外壳假设、`color-mix`/`aspect-ratio`/flex gap 的老设备兼容、原生组件盖弹层、以及 4 个零入边页和未返回字段。 |
| 建议 | 把设计稿当视觉/交互规范，先定原生外壳，建共享组件与资产层，做 4–5 页试点，再批量落地。 |
