你是幼儿园管理平台「养成星球」的资深移动端设计师。为家长端微信小程序做一个页面的高保真 HTML 原型。

## 唯一任务
页面：通知详情（源码：parent/src/pages/notice/detail.vue）
只做一个方向，不要 A/B 变体，不要多页面。写完 index.html / styles.css / app.js 三个文件后立刻停止：不要读回产物，不要自查循环，不要提问，不要联网。

## 这个页面真实的产品事实（照此设计，不要发明）

家长从「通知公告」列表点一条通知，`uni.navigateTo` 进入本页（子页，有返回箭头）。`onLoad` 拿 `id` 调 `GET /parent/notices/:id`，成功后**静默**调 `PUT /parent/notices/:id/read` 置为已读。页面自上而下只有：

1. **通知卡**
   - 顶行：`category_label` 类型标签（左）+ `published_at` 发布时间（右，源码截取到分钟）
   - 标题 `title`
   - 发布人行：圆形头像位 + `creator.nickname`（没有班级、没有职位字段）
   - 正文 `content`（源码 `white-space: pre-wrap`，是有段落的长文本）
   - 附件区（`attachments.length > 0` 才有）：小节标签「附件」+ 一行方形图片缩略（源码 200rpx ≈ 100px，`aspectFill`），点击 `uni.previewImage` 看原图
   - 已读信息块（`is_read` 为真才有）：一行小字「已读 2026-10-26 08:36」，已确认时追加「 · 已确认 2026-10-26 09:12」
2. **底部固定确认栏**：仅当 `need_confirm === true && is_confirmed !== true` 时出现，整宽主按钮「确认收到」；点击调 `PUT /parent/notices/:id/confirm`，成功后 `is_confirmed = true`、`confirmed_at = 现在`，toast「已确认」，**确认栏随即消失**，已读信息块补上「已确认」时间。

**源码里没有、绝对不要画**：置顶/标星、收藏、分享、评论、点赞、搜索、相关通知推荐、上一篇/下一篇、语音朗读、阅读进度、附件下载按钮、班级名称、阅读人数统计（家长端没有）、「已读回执」列表。**不画底部 tabBar**（这是子页，家长端只有首页/成长/接送/我的四个页面画）。

## 示例数据（中文，幼儿园语境；今天 = 2026年10月26日 星期一）

- `category_label`：活动
- `published_at`：2026-10-26 08:30 → 显示「10月26日 08:30」
- `title`：秋季亲子运动会报名通知
- `creator.nickname`：李老师
- `content`（按段落拆分，原文即多段）：
  1. 各位家长：
  2. 10月30日（周五）上午 9:00，我们将在幼儿园大操场举行秋季亲子运动会，小一班全体家庭参加，请家长提前 10 分钟带孩子到班级集合。
  3. 当天安排：9:00 开幕式与热身操，9:30 亲子接力、袋鼠跳、拔河三项比赛，11:00 颁奖与合影。孩子请穿园服和运动鞋，家长穿方便活动的衣服。
  4. 每名幼儿需由一位家长陪同，如需两位家长到场请提前在班级群说明。当天不设午餐，活动 11:30 结束，请家长准时接回孩子。
  5. 如遇雨天，活动顺延至下周三同一时间，另行通知。
- 附件：2 张图片占位（图片不联网加载，用占位块）
- `is_read`：true，`read_at` → 显示「已读 10月26日 08:36」
- `need_confirm`：true，`is_confirmed`：false（正常态）
- 已回执态：`is_confirmed`：true，`confirmed_at` → 「已确认 10月26日 09:12」

## 视觉与结构硬约束

- `<html lang="zh-CN" data-surface="parent">`，390px 手机视口，单列纵向流，不做桌面布局。
- **外壳必须原样使用共享 shell**：先读 `D:/yuanhaisu/personal_projects/kindergarten/docs/design-previews/2026-10-25-full-redesign/_shared/shell.md` 与同目录 `shell.css`，照抄 `mp-statusbar` / `mp-time` / `mp-status-icons` / `mp-navbar` / `mp-back` / `mp-title` / `mp-capsule` / `mp-main` / `mp-demo` 的标记与 SVG，不许自造状态栏。
- 引入顺序：`tokens.css` → `../../_shared/shell.css` → `styles.css`；末尾先 `<script src="../../_shared/shell.js"></script>` 再 `<script src="app.js"></script>`（app.js 监听 `document` 的 `mp:state` 事件）。
- **禁止把 shell.css 复制进项目**，也不要改写状态栏 SVG；只写 `<link href="../../_shared/shell.css">`，文件在当前项目里不存在也没关系（由流程拼装）。
- **styles.css 里绝对不许出现任何 `.mp-*` 选择器或覆盖**（唯一例外是 `[data-state]` 状态选择器，如 `.mp-app[data-state="loading"] .sheet`）。
- **子页：必须画返回箭头**（`mp-back`）；导航标题是 `pages.json` 的 `navigationBarTitleText` —— **「通知详情」**。
- **不画 tabBar**。`.mp-main` 不要加 `mp-main--scroll`（让 window 滚动，外壳 sticky 才成立）；页面不得给 `body` 或 `.mp-app` 设 overflow。
- 底部确认栏是固定在视口底部的整宽条：`position: fixed; left: 50%; transform: translateX(-50%); width: 100%; max-width: 390px; bottom: 0;`，内边距含 `env(safe-area-inset-bottom)`；正文区 `padding-bottom: calc(96px + env(safe-area-inset-bottom))` 避免被遮。
- 演示条 `mp-demo` 需要抬到确认栏之上，用唯一允许的行内覆盖：`style="bottom: calc(84px + env(safe-area-inset-bottom))"`。
- 颜色只写 `var(--*)`，禁止裸 hex / rgb / hsl；无直角（最小 `var(--radius-sm)`）；阴影只用 `var(--shadow-*)`。
- 触控目标 ≥ 44×44px；正文 ≥ 13px；`:active` 必须有反馈（`transform: scale(.99)` 或背景压暗）。
- 禁止 `white-space: nowrap` 用于可能超宽的文本（导航标题由 shell 处理，不用管）；所有容器 `max-width: 100%`，页面不得横向滚动；标题与正文自然折行。
- 不用 `<select>`、alert、prompt；无 console 报错；不引用任何外部资源；不内联 base64 大图。

### 设计方向（必须遵守）

**「一页正式通知，不是一张信息卡」**：像贴在班级门口的那张通知单——白纸、留白、字距舒展，读起来是一封信；品牌橙只出现在类型标签、头像底色和底部那一个主按钮上。

- 整页底 `var(--bg)`；正文纸面是一张 `var(--surface)` 卡，圆角 `var(--radius-card)`，`var(--shadow-sm)`，内边距 20px 18px。
- **顶行**：类型标签是一枚小胶囊（`var(--accent-wash)` 底 + `var(--accent-deep)` 字，11px/500，高 24px）；发布时间 12px `var(--muted)`，右对齐，数字用 `font-variant-numeric: tabular-nums`。
- **标题**：20px/600，`line-height: 1.45`，`var(--fg)`，`max-width: 22em`；下方 14px 间距。
- **发布人行**：32px 圆头像（`var(--accent-wash)` 底 + `var(--accent-deep)` 的首字「李」，13px/600）+ 姓名 13px/500 `var(--fg-2)`；不要「作者 / 发布」这类标签词。
- 标题与正文之间一条 1px `var(--border)` 细分隔线（不是卡片内再套卡片）。
- **正文排版是主角**：15px，`line-height: 1.85`，`color: var(--fg-2)`，`max-width: 34em`，段落之间 14px 间距；称呼段「各位家长：」与其他段同样式（不要放大加粗成标题）；最后一段是落款语气，可以略短。正文必须分成 5 个 `<p>`，**不允许挤成一段**；长句自然折行，`overflow-wrap: anywhere` 只作为兜底。
- **附件区**：小节标签「附件」12px/500 `var(--muted)`（右侧跟一个「2 张」的等宽小字，不要胶囊）；一行 2 个方形占位块，边长 96px、圆角 `var(--radius-xl)`、`var(--bg)` 底 + 1px `var(--border)` 描边，里面居中一个 22px `var(--hint)` 的图片线性图标，块下方 10px 处一行 11px `var(--hint)` 的「图片 1 / 图片 2」。**不要用渐变色占位、不要画假照片**。
- **已读信息块**：`var(--bg)` 底、圆角 `var(--radius-xl)`、内边距 12px 14px，一行 12px `var(--muted)` 文字，前面一个 14px 的对勾图标（用 `currentColor`）。已确认时行末追加「· 已确认 10月26日 09:12」，并且整块文字换成 `var(--success)`、图标同步换色（用 `currentColor` 一处改）。
- **底部确认栏**：`var(--surface)` 底 + 顶部 1px `var(--border)` + `var(--shadow-lg)` 向上投影的轻浮层；内边距 12px 14px，主按钮高 48px、圆角 `var(--radius-pill)`、`var(--accent-gradient)` 底、`var(--on-brand)` 字、15px/600，文案「确认收到」，左侧一个 16px 的对勾图标。**整条只有一个按钮，不要旁边加「稍后」「忽略」等源码没有的次按钮。**
- 暖色面积 ≤ 10%：全页只有类型胶囊、头像底、底部主按钮三处是暖色，纸面与背景保持中性。
- 字号层级只用字重 + 字号，不要给正文加背景色块、不要加装饰引号、不要加水印印章。

## 状态覆盖（`data-state` 驱动，演示条用 `mp-demo` 四芯片：正常 / 已回执 / 加载 / 失败）

芯片文案用中文，`data-mp-state` 分别写 `normal`、`empty`（映射「已回执」，本页没有列表空态，不要造第五个状态）、`loading`、`error`。

- **正常**（默认）：上面的完整通知（need_confirm 且未确认）→ 底部确认栏可见；已读信息块只有「已读 10月26日 08:36」。
- **已回执**（`data-state="empty"`）：纸面顶部多一条 `var(--success-wash)` 底的确认横幅（对勾图标 + 「已确认收到」+ 「10月26日 09:12」，圆角 `var(--radius-xl)`，14px/500 `var(--success)`）；底部确认栏隐藏；已读信息块变成绿色并带「· 已确认 10月26日 09:12」；正文 padding-bottom 收到 48px。
- **加载**：纸面与确认栏隐藏，换成纸面骨架：标签条 + 标题两行 + 发布人行（圆 + 短条）+ 4 行正文 + 2 个 96px 附件方块；骨架用 `var(--border)`/`var(--bg)` 呼吸动画，`prefers-reduced-motion` 下关掉动画。
- **失败**：纸面与骨架隐藏，换成错误块 —— `var(--danger-wash)` 圆底 + `var(--danger)` 线性图标（圆形叹号 / 断网）、标题「通知加载失败」、副文「网络异常或这条通知已被撤回」、白底描边的「重试」按钮；底部确认栏隐藏。
- **禁用态（不占芯片）**：① 点「确认收到」后按钮立即 `disabled`：文案「确认中…」、`opacity: .6`、`pointer-events: none`，约 0.8 秒后切到「已回执」并 toast「已确认收到」；② 加载/失败态下附件方块不可点（整块不渲染即可）。

## 交互（app.js，全部真实可点）

- 点返回箭头：toast「返回通知列表」（原型不真跳页）。
- 点附件方块：toast「打开图片预览 · 原型不联网加载图片」，并让被点的方块出现一圈 `var(--accent-soft)` 的 `outline` 短暂反馈。
- 点「确认收到」：先进禁用文案态，0.8 秒后把页面切到已回执态（`data-state="empty"` 等价表现，直接改 DOM 类即可，不必改 mp-app 的 data-state，以免演示芯片状态错乱），toast「已确认收到」。
- 点「重试」：toast「已重新加载」，回到正常态。
- 演示条芯片切换由 `_shared/shell.js` 写 `data-state` 并派发 `mp:state`；app.js 监听该事件，进入正常态时把确认按钮与回执横幅重置回初始。
- toast 用页面自绘的轻提示（`var(--fg)` 底、`var(--on-brand)` 字、圆角胶囊、固定在中下部、2 秒消失）；不用 alert。
- 所有可点元素有 `:active` 反馈；无 console 报错。

## 输出文件（项目根目录，每个 < 20KB）
1. index.html —— 结构
2. styles.css —— 样式（引用 tokens.css）
3. app.js —— 交互
tokens.css 已由流程提供，**不要重写，不要覆盖**。不要生成 preview.png 或 notes.md。
