你是幼儿园管理平台「养成星球」的资深移动端设计师。为家长端微信小程序做**首页**的高保真 HTML 原型。

## 唯一任务
页面：家长首页（源码：parent/src/pages/index/index.vue）
只做一个方向，不要 A/B 变体，不要多页面。写完 index.html / styles.css / app.js 三个文件后立刻停止：不要读回自己的产物，不要自查循环，不要提问，不要联网。

## 这个页面真实的产品事实（照此设计，不要发明字段）

这是家长端 **tabBar 首页**（4 个 tab 中的第 1 个，选中态是「首页」）。家长每天第一眼看的页面。自上而下 6 个区块：

1. **孩子卡（hero）**——源码是一整块橙色渐变 header，内含：
   - 孩子头像（圆形；无头像时用占位圆）
   - 孩子姓名（如「乐乐」）
   - 班级标签（`current_student.class_name`，如「小一班」）
   - 问候语「{{relation}} 您好~」，relation 来自 `students[].relation`：father=爸爸 / mother=妈妈 / grandfather=爷爷 / grandmother=奶奶，匹配不到显示「家长」
   - 右侧**设置按钮** → `/pages/settings/index`
2. **今日在校情况卡**：
   - 标题「今日在校情况」+ 日期（`overview.date` 格式化为「2026年10月26日」；`overview.weekday` 契约里有「星期一」，可并列显示）
   - 4 个状态列（固定顺序）：**出勤**（`attendance_label` 如「已入园」，副行 `arrive_time`「07:52」）、**进餐**（`meal_label`「很好」）、**午睡**（`sleep_label`「很好」，副行 `nap_duration`「120分钟」）、**情绪**（`mood_label`「很好」）
   - 有 `performance_tag` 时卡片底部一条表现条：「今日获得「小小艺术家」奖章」
3. **多孩切换**（`students.length > 1` 才出现）：一排孩子名字胶囊，当前孩子高亮，点了切换（源码 `handleSwitchStudent`）
4. **功能入口 4 格（一行）**：**通知公告**（未读数 `new_notice_count` 显示角标）→ 通知列表；**接送管理** → 接送页；**亲子任务**（`pending_tasks` 角标）→ 任务列表；**我要请假** → 请假申请
5. **今晚一件事**（有未提交任务时出现，`todayTask`）：任务标题 + 描述（`description`，没有则 `learning_link`）+「去完成 ›」，整卡可点 → 任务详情
6. **最新动态**：通知 + 成长记录混合流（接口最多取 3 通知 + 3 成长，按时间倒序，page_size 5）。两种卡片：
   - 通知：小标签「通知」+ 标题 + **未读红点**（`notice.is_read === false`）+ 两行内容预览（`content_preview`）
   - 成长：小标签「成长」+ 内容（`growth.content`）
   - 右下角时间：1 小时内「N分钟前」、24 小时内「N小时前」、更早「2026-10-25」

**真实但不画的边界态**：未绑定手机号时，整页换成「绑定手机号以关联孩子信息」引导卡（`needBindPhone` 分支）。本页只做下面四态，这个分支不要画，留给产品单独定页面。

**绝对不要画**：消息中心入口（源码首页没有）、成长报告入口、积分入口、「全部动态 / 查看更多」链接（源码无对应页面）、下拉刷新控件、视频、置顶、天气、幼儿园公告轮播（`fetchAnnouncements` 首页模板并未渲染）。

## 示例数据（中文，幼儿园语境；今天 = 2026年10月26日 星期一）

- 当前孩子：**乐乐**，小一班，关系「妈妈」→ 问候「妈妈 您好~」
- 多孩：乐乐（选中）/ 小雨
- 今日：出勤 已入园 · 07:52 ｜ 进餐 很好 ｜ 午睡 很好 · 120分钟 ｜ 情绪 很好
- 表现条：「今日获得「小小艺术家」奖章」
- 角标：通知公告 1，亲子任务 3
- 今晚一件事：「亲子阅读 15 分钟」——「和孩子一起读《好饿的毛毛虫》，说说毛毛虫都吃了什么」，按钮「去完成 ›」
- 最新动态 3 条：
  1. 通知（未读）「秋季亲子运动会报名通知」——「本周五上午 9:00 在大操场举行秋季亲子运动会，请填写回执交回班级。」——2小时前
  2. 成长「今天的美术课主题是《我心中的春天》，乐乐用棉签点画了一棵开满花的树。」——5小时前
  3. 成长「午睡后孩子们一起整理了自然角，小雨给多肉浇了水。」——2026-10-25

## 视觉与结构硬约束

- `<html lang="zh-CN" data-surface="parent">`，390px 手机视口，单列纵向流，不做桌面布局。
- **外壳必须原样使用共享 shell**：先读 `D:/yuanhaisu/personal_projects/kindergarten/docs/design-previews/2026-10-25-full-redesign/_shared/shell.md` 与同目录 `shell.css`，照抄其中 `mp-statusbar` / `mp-time` / `mp-status-icons` / `mp-navbar` / `mp-title` / `mp-capsule` / `mp-main` / `mp-tabbar` / `mp-tab` / `mp-demo` 的标记与 SVG，不许自造状态栏或 tabBar。
- 引入顺序：`tokens.css` → `../../_shared/shell.css` → `styles.css`；末尾先 `<script src="../../_shared/shell.js"></script>` 再 `<script src="app.js"></script>`（app.js 监听 `document` 的 `mp:state` 事件）。
- **禁止把 shell.css 复制进项目**，也不要改写状态栏 SVG；只写 `<link href="../../_shared/shell.css">`，文件在当前项目里不存在也没关系（最终由流程拼装）。
- **styles.css 里绝对不许出现任何 `.mp-*` 选择器或覆盖**（唯一例外是 `[data-state]` 状态选择器，可写成 `.mp-app[data-state="empty"] .xxx`）。
- **tabBar 页：不画返回箭头**；导航标题就是 `pages.json` 的 `navigationBarTitleText`——**「养成星球」**（不是「家长首页」）。
- **必须画 4 项 tabBar**：首页 / 成长 / 接送 / 我的，每项 22×22 线性图标 + 10px 文字；「首页」用 `aria-current="page"`（品牌色），其余 `--muted`。因为 tabBar 在底部常驻，演示条 `mp-demo` 用行内 `style="bottom: calc(62px + env(safe-area-inset-bottom))"` 避让（这是唯一允许的行内外壳覆盖）。

### 设计方向（必须遵守，否则重做）

**「清晨第一眼」：孩子 → 今天 → 该做什么。**
- **不许整屏铺品牌渐变**（现状就是这个毛病，暖色过半屏）。整页底是冷雾白 `var(--bg)`；唯一的暖色块只允许出现在**孩子卡这一张紧凑 hero**（`var(--accent-gradient-soft)`）与各处的浅色 wash 圆点/胶囊上，**暖色面积 ≤ 10%**。
- **孩子卡是 hero，不是工具栏**：头像 52px 圆（外圈 2px `--surface` 环），姓名 18px/600 `--fg`，班级标签是 `--surface` 底 + `--accent-deep` 字的胶囊，问候语 12px `--muted`。设置按钮 44×44、`--surface` 圆底、齿轮线性图标。
- **今日卡是视觉锚点**：白卡 `var(--radius-card)` + `var(--shadow-md)`；4 个状态列用 `grid-template-columns: repeat(4, minmax(0, 1fr))`；每列一个 38px 圆（底色 wash：出勤 `--success-wash`、进餐 `--info-wash`、午睡 `--warning-wash`、情绪 `--accent-wash`；图标 `currentColor` 取对应实色），下面 12px/500 标签 + 11px `--muted` 副行。
- **首屏密度硬要求**：孩子卡 + 今日卡 + 快捷入口必须在**第一屏（约 700px 内容高度）内全部可见**，不要在这三块之间塞大标题、大留白或大插图。
- **入口不画位图**：现状是 4 张 PNG 插画，原型一律用 22px `currentColor` 线性 SVG（喇叭/接送箭头/任务小人/请假日历），放在 46px `--accent-wash` 圆角方（`--radius-xl`）里；角标是 `--danger` 底白字小胶囊（数字来自真实字段）。
- **最新动态是卡片流**，不是表格：类型标签 11px 胶囊（通知 `--info-wash`/`--info`，成长 `--success-wash`/`--success`），标题 14px/600，预览 12px `--muted` 两行截断，时间 11px `--muted` 右对齐（不要用 `--hint`，对比度不够）。
- 区块标题在卡片外：12px/500 `--muted`（「今晚一件事」「最新动态」）。
- 颜色只写 `var(--*)`，禁止裸 hex / rgb / hsl；无直角（最小 `var(--radius-sm)`）；阴影只用 `var(--shadow-*)`。
- 触控目标 ≥ 44×44px；正文 ≥ 13px；`:active` 必须有反馈（缩放或压暗）。
- 禁止对可能超宽的文本用 `white-space: nowrap`；多列统计用 `grid-template-columns: repeat(N, minmax(0, 1fr))`；所有容器 `max-width: 100%`，页面不得横向滚动；长标题/内容自然折行。
- 不用 `<select>`、alert、prompt；无 console 报错。

## 状态覆盖（`data-state` 驱动，演示条用 `mp-demo`，四个芯片：正常 / 空态 / 加载 / 失败）

- **正常**：上面的示例数据完整渲染（默认）。
- **空态**：隐藏「今晚一件事」和动态列表，动态区换成空态块——线性图标位 + 一句「这周还没有新的通知和成长记录」+ 主按钮「看看成长时光」；孩子卡、今日卡、快捷入口照常显示。
- **加载**：孩子卡保留姓名/班级文字（缓存态），今日卡的 4 个状态替换成灰色骨架圆 + 骨架短条，动态区换成 2 条骨架卡（`--border` 底的圆角块，带呼吸动画 `prefers-reduced-motion` 关掉）；「今晚一件事」隐藏。
- **失败**：今日卡与动态区整体换成一张错误卡：`--danger-wash` 图标位 + 一句「首页数据加载失败」+「重试」按钮；孩子卡与快捷入口仍显示。
- **禁用态（不占芯片）**：① 多孩胶囊里当前孩子不可再点（`aria-current="true"`，无按压反馈）；② 错误态点「重试」后按钮立刻 `disabled`、文案「重试中…」、降到 `--hint`/`--border` 视觉，约 1 秒后恢复并 toast「已重新加载」。

## 交互（app.js，全部真实可点）

- 点孩子胶囊：切换当前孩子（乐乐 ↔ 小雨），同时更新孩子卡姓名/班级/问候语与今日卡 4 个状态值（小雨=大二班、奶奶、已入园 08:05、进餐 一般、午睡 120分钟、情绪 很好），并 toast「已切换到 小雨」。
- 点设置按钮 → toast「已打开设置」。
- 点 4 个功能入口 → 各自 toast「已打开「通知公告」」等。
- 点「今晚一件事」整卡或「去完成 ›」→ toast「已打开任务详情」。
- 点动态卡 → 通知 toast「已打开通知详情」，成长 toast「已打开成长详情」。
- 点空态「看看成长时光」→ toast「已切换到成长时光」。
- 演示条芯片切换四态（由 `_shared/shell.js` 写 `data-state`）；页面自己监听 `document` 的 `mp:state` 事件来做需要 JS 配合的收尾（如错误态里重置重试按钮）。
- toast 用页面自绘的轻提示（`--fg` 底、`--on-brand` 字、圆角胶囊、固定在 tabBar 上方），2 秒自动消失；不用 alert。
- 所有按钮/卡片有 `:active` 反馈；无 console 报错。

## 输出文件（项目根目录，每个 < 20KB）
1. index.html —— 结构
2. styles.css —— 样式（引用 tokens.css）
3. app.js —— 交互
tokens.css 已由流程提供，**不要重写，不要覆盖**。
每个文件 < 20KB。不要引用任何外部资源。不要生成 preview.png 或 notes.md。
