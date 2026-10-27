你是幼儿园管理平台「养成星球」的资深移动端设计师。为家长端微信小程序做一个页面的高保真 HTML 原型。

## 唯一任务
页面：成长报告（源码：parent/src/pages/growth/report.vue）
只做一个方向，不要 A/B 变体，不要多页面。写完 index.html / styles.css / app.js 三个文件后立刻停止：不要读回产物，不要自查循环，不要提问，不要联网。

## 这个页面真实的产品事实（照此设计，不要发明）

家长从「成长时光」页或「我的 → 成长报告」经 `uni.navigateTo` 进入本页（**子页，有返回箭头**）。`onLoad` 调 `GET /parent/growth/report?type=weekly`（默认 `weekly`），把响应 `data` 当作报告对象整体渲染。**页面自上而下只有这些区块，顺序不许改**：

1. **报告类型切换**：三个等宽分段 `周报 / 月报 / 学期`（`type` = `weekly` / `monthly` / `term`），默认「周报」。点击切换会重新请求同一接口（切换期间是加载态），**没有下拉刷新、没有导出、没有分享**。
2. **学生信息**：横向一行 —— 圆形头像（无头像时占位）+ 姓名 `student.name` + 周期 `period.start_date ~ period.end_date`。源码没有班级、没有学号、没有老师姓名。
3. **统计概览**：一行四格，依次为
   - `attendance.rate`（字符串，如 `80%`）→ 标签「出勤率」
   - `points_earned` → 标签「获得积分」
   - `tasks_completed` / `tasks_total`（显示成 `8/12`）→ 标签「完成任务」
   - `growth_media_count` → 标签「成长记录」
4. **能力评估**：`dimensions[]`，每项 `{ name, score, max_score }`。源码只显示「维度名 + 进度条 + 分数」，**不显示 `dimensions[].comment`**。本期 7 个维度：语言 / 认知 / 社交 / 运动 / 艺术 / 自理 / 情绪，满分均 100。
5. **成长亮点**：`highlights[]` 字符串数组，逐条圆点列表。
6. **需要关注**：`concerns[]` 字符串数组，同样逐条列表，语义上是提醒而非警告。
7. **老师评语**：`teacher_comment` 一段话。源码只有一个头像占位，**没有老师姓名、没有日期**。

接口还会返回 `meals` / `sleep` / `mood` / `top_performance_tags`，但**本页源码完全不渲染**——绝对不要画。也不要画：导出/分享/下载 PDF、成长照片墙、评论、点赞、雷达图、环比箭头、时间筛选器、班级名、孩子切换器、底部 tabBar（这是子页）。「能力评估」用 **纯 CSS 柱状/条形**示意，**不许引图表库、不许用 canvas/SVG 图表**。

## 示例数据（中文，幼儿园语境；今天 = 2026年10月26日 星期一；孩子 = 乐乐，小一班）

三段数据分别对应三个分段，切换时必须整体变化，且数字彼此自洽（出勤率与「x/y 天」要对得上）。

### 周报（默认，`weekly`）
- 周期：`2026-10-19 ~ 2026-10-25` → 显示「10月19日 – 10月25日」（上周一至周日，5 个上学日）
- 出勤率 `80%`（`present_days` 4 / `total_days` 5，`leave_days` 1）
- 获得积分 `45`
- 完成任务 `8` / `12`
- 成长记录 `15`
- 维度分数：语言 85、认知 78、社交 90、运动 72、艺术 95、自理 68、情绪 82
- 成长亮点：
  1. 美术课《我心中的春天》被贴到班级作品墙
  2. 周三主动把水杯递给朵朵，得了一次「乐于助人」
  3. 连续 5 天自己穿好外套和鞋子
- 需要关注：
  1. 午睡入睡偏慢，有两天到 13:10 才睡着
  2. 青菜剩得多，周三、周四都只吃了一半
- 老师评语：乐乐这周在美术课上的专注时间明显变长了，涂色时会自己换颜色，还会跟旁边的小朋友介绍自己的画。午睡还需要一点时间安静下来，我们会继续陪他躺一会儿。周末在家可以让他自己整理小书包，把这周养成的好习惯接着做下去。

### 月报（`monthly`）
- 周期：`2026-09-01 ~ 2026-09-30` → 显示「9月1日 – 9月30日」
- 出勤率 `91%`（20 / 22 天）
- 获得积分 `180`
- 完成任务 `31` / `40`
- 成长记录 `62`
- 维度分数：语言 84、认知 80、社交 89、运动 74、艺术 93、自理 70、情绪 83
- 成长亮点：
  1. 九月的 4 次户外活动全部参加，学会了双脚交替跳
  2. 第一次在集体面前讲完《小蝌蚪找妈妈》，声音很清楚
  3. 开始主动收拾自己的餐具，连续两周没让老师提醒
- 需要关注：
  1. 遇到不熟悉的小朋友会先站在旁边看，加入游戏比较慢
  2. 握笔姿势还需要纠正，画线时手腕容易压得太低
- 老师评语：这个月乐乐最大的变化是愿意开口了：从只跟同桌说话，到能在小组里讲完整的小故事。运动量也跟上来了，户外活动很少落在后面。接下来我们会多给他一些和陌生同伴合作的机会，也请家里继续提醒他握笔时手腕抬起来。

### 学期（`term`）
- 周期：`2026-09-01 ~ 2026-10-25` → 显示「9月1日 – 10月25日」
- 出勤率 `94%`（32 / 34 天）
- 获得积分 `610`
- 完成任务 `96` / `120`
- 成长记录 `186`
- 维度分数：语言 87、认知 82、社交 92、运动 76、艺术 96、自理 73、情绪 85
- 成长亮点：
  1. 入园两个月，从需要老师喂饭到自己独立吃完
  2. 交到了两个固定的好朋友，区域活动会主动约着一起玩
  3. 学会了写自己的名字，虽然「乐」字的最后一笔还要提醒
- 需要关注：
  1. 感冒请假 2 天，返园后情绪恢复得比上学期慢
  2. 自理里的「整理物品」最弱，玩具常常留在桌上
- 老师评语：这两个月乐乐像换了个人：吃饭、穿衣都能自己来，活动里也敢举手了。他最喜欢美工区，一做就是半小时，作品我们会继续帮他收进成长册。接下来重点练「收尾」——玩完把材料送回家，这件事需要家里和幼儿园一起盯一段时间。

## 视觉与结构硬约束

- `<html lang="zh-CN" data-surface="parent">`，390px 手机视口，单列纵向流，不做桌面布局。内容区左右各 14px（`--page-pad-x`），可用宽度 362px。
- **外壳必须原样使用共享 shell**：先读 `D:/yuanhaisu/personal_projects/kindergarten/docs/design-previews/2026-10-25-full-redesign/_shared/shell.md` 与同目录 `shell.css`，照抄 `mp-statusbar` / `mp-time` / `mp-status-icons` / `mp-navbar` / `mp-back` / `mp-title` / `mp-capsule` / `mp-main` / `mp-demo` 的标记与 SVG，不许自造状态栏。
- 引入顺序：`tokens.css` → `../../_shared/shell.css` → `styles.css`；末尾先 `<script src="../../_shared/shell.js"></script>` 再 `<script src="app.js"></script>`（app.js 监听 `document` 的 `mp:state` 事件）。
- **禁止把 shell.css 复制进项目**，也不要改写状态栏 SVG；只写 `<link href="../../_shared/shell.css">`，文件在当前项目里不存在也没关系（由流程拼装）。
- **styles.css 里绝对不许出现任何 `.mp-*` 选择器**（唯一例外是 `[data-state]` 状态选择器，如 `.mp-app[data-state="loading"] .sheet`）。
- **子页：必须画返回箭头**（`mp-back`）；导航标题是 `pages.json` 的 `navigationBarTitleText` —— **「成长报告」**。
- **不画 tabBar**。`.mp-main` 不要加 `mp-main--scroll`（让 window 滚动，外壳 sticky 才成立）；页面不得给 `body` 或 `.mp-app` 设 overflow。
- 颜色只写 `var(--*)`，禁止裸 hex / rgb / hsl（包括 SVG 的 `fill` / `stroke`，一律 `currentColor` / `none`）。
- **无直角**：最小圆角 `var(--radius-sm)`；按钮/标签/头像/进度条 `var(--radius-pill)`；纸面 `var(--radius-card)`。**禁止出现 `border-radius: 0`**。
- 阴影只用 `var(--shadow-*)`。
- 触控目标 ≥ 44×44px；正文 ≥ 13px；`:active` 必须有反馈（`transform: scale(.99)` 或背景压暗）。
- **`styles.css` 里禁止出现 `white-space: nowrap`**（审计直接判失败）；所有容器 `max-width: 100%`；多列统计用 `grid-template-columns: repeat(N, minmax(0, 1fr))`；页面不得横向滚动。
- 不用 `<select>`、alert、prompt；无 console 报错；不引用任何外部资源；不内联 base64 大图。

### 设计方向（必须遵守）

**「一张成长册纸面，不是五张卡片」**：整份报告是**一张**白纸（`var(--surface)` + `var(--radius-card)` + `var(--shadow-sm)`），从上到下用 1px `var(--border)` 细分隔线分段，读起来像老师发下来的一页成长册；品牌橙只出现在纸面顶部的暖光封面、七根能力柱和唯一的按钮上。

- **页面底** `var(--bg)`。纸面上方先放类型切换：一条 `var(--surface)` 胶囊轨道（高 44px、内边距 4px、1px `var(--border)` 描边），三个等宽分段，**选中段是 `var(--accent)` 实底 + `var(--on-brand)` 字 13px/600**，未选中段透明底 + `var(--muted)` 字；`transition` 只做背景与颜色。
- **纸面封面（head）**：`var(--accent-gradient-soft)` 铺满纸面顶部、只有上圆角；内边距 18px 16px；左侧 56px 圆形头像（`var(--surface)` 底 + `var(--accent-deep)` 的首字「乐」，20px/600，`var(--radius-pill)`），右侧姓名 20px/600 `var(--fg)`，姓名下 12px `var(--fg-2)` 的周期「10月19日 – 10月25日」（数字 `font-variant-numeric: tabular-nums`）；右上角允许一枚 12px `var(--accent-deep)` 的小标签「周报 / 月报 / 学期」，`var(--surface)` 半透明底、胶囊形，**不要**再放别的装饰字。封面右侧可以有一圈极淡的 `var(--accent-soft)` 描边圆环做背景（用伪元素 + `overflow: hidden`），不许越出纸面。
- **四格统计**：封面下一条通栏，四列等宽，格与格之间 1px `var(--border)` 竖线；每格内边距 14px 4px，数字 18px/600 `var(--fg)`（`font-variant-numeric: tabular-nums`），标签 11px `var(--muted)` 在下方 3px；**四格高度必须一致**，`8/12` 这种复合数字不许换行、不许溢出（可用 17px 或让分母字号 12px）。
- **分隔**：每个大区块之间一条 1px `var(--border)` 通栏线（左右各留 16px），不要用卡片阴影区分区块。
- **能力评估**：区块标题「能力评估」14px/600 `var(--fg)`，右侧同一行放 11px `var(--muted)` 的「满分 100」。下面是 **7 列竖向柱状图**：每列等宽（`grid-template-columns: repeat(7, minmax(0, 1fr))`），列内自下而上为 分数（11px/600 `var(--fg-2)`，在柱子顶端上方 4px）→ 柱槽（宽 16px、高 96px、`var(--bg)` 底、`var(--radius-pill)`、居中）→ 柱体（按 `score / max_score` 的高度百分比，`var(--accent-gradient)` 底、`var(--radius-pill)`）→ 维度名（12px `var(--muted)`，2 个字，居中）。柱槽与柱体都以底部对齐（柱槽用 flex `align-items:flex-end`，柱体 `height: X%`）。柱高比例直接用真实分数的百分比，**不要美化数字**。图表下方一行 11px `var(--muted)` 说明：「条形越高表示该维度表现越好」。分数最低的那根柱仍要可读，不要把它做成警示色（这不是错误）。
- **成长亮点 / 需要关注**：各自一个小节，标题 14px/600 `var(--fg)`；条目 13px/1.65 `var(--fg-2)`，左侧 6px 圆点（亮点 `var(--success)`、关注 `var(--warning)`）与首行文字对齐（`margin-top: 5px`），条目间距 8px；**不要**给「需要关注」加大面积黄底或警告图标，它只是提醒。两个小节之间用分隔线。
- **老师评语**：标题「老师评语」14px/600；正文区用 `var(--bg)` 底 + `var(--radius-xl)` + 内边距 14px，左侧一条 3px `var(--accent-soft)` 的竖线（`var(--radius-pill)`）标记「这是老师的话」；头像 32px 圆（`var(--surface)` 底 + `var(--accent-deep)` 的人形线性图标 16px）+ 正文 14px/1.75 `var(--fg-2)`，自然折行。**不要**加引号装饰、签名、日期。
- **暖色面积 ≤ 10%**：封面底、选中的分段、七根柱子、唯一按钮，仅此四处；纸面其余部分保持中性。
- 字号层级只用字重 + 字号；不要给区块加背景色块制造「仪表盘感」。

## 状态覆盖（`data-state` 驱动，演示条用共享 `mp-demo` 四芯片：正常 / 空态 / 加载 / 失败）

- **正常**（默认）：上面的完整纸面（周报），三段切换都可点。
- **空态（`data-state="empty"`，「本周期无数据」）**：纸面与骨架隐藏；类型切换**保留且可点**；下方居中一块空态 —— 96px `var(--accent-wash)` 圆底 + `var(--accent-deep)` 的线性「图表/纸页」图标、标题「这个周期还没有成长报告」、副文「老师通常周五整理周报，先看看月报或学期报告」、主按钮「查看月报」（`var(--accent-gradient)` 底、`var(--on-brand)` 字、高 44px、胶囊、`var(--shadow-md)`）。点它 → 切到月报 + 回到正常态 + toast「已切换到月报」。
- **加载（`data-state="loading"`）**：纸面隐藏，换成纸面骨架 —— 封面区（56px 圆 + 两条短条）、四格统计（每格两条短条）、七根等高柱槽、两组文字条；骨架用 `var(--border)` / `var(--bg)` 呼吸动画（`prefers-reduced-motion` 下关掉动画）；类型切换变灰且不可点（`aria-disabled="true"`，`pointer-events: none`）。
- **失败（`data-state="error"`）**：纸面与骨架隐藏；居中错误块 —— 48px `var(--danger-wash)` 圆底 + `var(--danger)` 线性图标、标题「成长报告加载失败」、副文「网络断开，或报告还在生成中」、主按钮「重试」；类型切换同样变灰不可点。点「重试」→ 回到正常态 + toast「已重新加载」。
- **禁用态（不占芯片）**：加载/失败时三个分段全部降级（`var(--border)` 底 + `var(--hint)` 字 + 无阴影），点击无效；正常态点当前分段时不做任何事。

## 交互（app.js，全部真实可点）

- 三个分段切换：切到对应周期的纸面（三段数据分别预渲染在三个 `<section class="report" data-period="...">` 里，只显示一个），同步 `aria-pressed` 与封面上的类型小标签、周期文字；切换后不弹 toast（静默，与源码一致）。
- 点返回箭头：toast「返回成长时光」（原型不真跳页）。
- 空态主按钮「查看月报」：切到月报 + 回正常态，并同步演示条芯片到「正常」。
- 失败态「重试」：回正常态，并同步演示条芯片到「正常」。
- 演示条芯片由 `_shared/shell.js` 写 `data-state` 并派发 `mp:state`；app.js 监听该事件，在进入正常态时把分段、卡片复位到默认（周报）。
- toast 用页面自绘的轻提示（`var(--fg)` 底、`var(--on-brand)` 字、`var(--radius-lg)`、固定在中下部、2 秒消失）；不用 alert。
- 所有可点元素有 `:active` 反馈；无 console 报错。

## 输出文件（项目根目录，每个 < 20KB）
1. index.html —— 结构（关键区块加 `data-od-id`，如 `report-tabs`、`report-sheet`、`report-stats`、`report-dims`、`report-comment`、`report-empty`、`report-error`）
2. styles.css —— 样式（引用 tokens.css）
3. app.js —— 交互
tokens.css 已由流程提供，**不要重写，不要覆盖**。不要生成 preview.png 或 notes.md。
