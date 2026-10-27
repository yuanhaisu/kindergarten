你是幼儿园管理平台「养成星球」的资深移动端设计师。为家长端微信小程序做**请假记录列表**的高保真 HTML 原型。

## 唯一任务
页面：请假记录（源码：parent/src/pages/leave/index.vue）
只做一个方向，不要 A/B 变体，不要多页面。写完 index.html / styles.css / app.js 三个文件后立刻停止：不要读回产物，不要自查循环，不要提问，不要联网。

## 这个页面真实的产品事实（照此设计，不要发明）

这是家长从首页「请假」入口 `uni.navigateTo` 进入的**子页**，自上而下 3 个区块：

1. **统计卡**（源码 `stats-card`）：三个数字 ——「全部请假」（当前列表总条数 `leaves.length`）、「待审批」（`status === 0` 的条数，源码用品牌色高亮）、「已同意」（`status === 1` 的条数）。源码第四格塞了一个「+ 请假」按钮；原型把新建入口提为**右下角悬浮胶囊按钮**（详见视觉约束），统计卡只留三个数字。
2. **状态分段**（源码 `tab-row`）：4 个胶囊 ——「全部 / 待审批 / 已同意 / 已拒绝」，默认「全部」。切换时前端按 `status` 过滤当前列表（pending=0、approved=1、rejected=2）。**注意：统计卡的三个数字按「全部记录」计算，不随分段筛选变化**（源码 `stats` 与 `filteredLeaves` 是两个独立的 computed）。
3. **请假列表**：每条是一张可点卡片。源码真实渲染的字段只有这些 ——
   - `student.name` 孩子姓名（`student.avatar` 列表接口并不返回，源码始终落到「?」占位圆）
   - `days` 天数（源码写「N天」）
   - `status_label` 状态文案，状态值：0 待审批 / 1 已同意 / 2 已驳回 / 3 已取消；源码给 0=琥珀底、1=绿底、2=红底、3=灰底四种徽章
   - `reason` 请假事由（源码单行省略）
   - `start_date` ~ `end_date` 起止日期（后端返回 `YYYY-MM-DD`）
   - `created_at` 提交时间、`reviewer.nickname` + `reviewed_at` 审批人/审批时间（列表接口有返回，源码未展示）
   - 点击整张卡 → `uni.navigateTo` 到 `/pages/leave/detail?id=<id>`

**本批次要求展示、但源码列表未渲染的两个字段（按提案设计，notes 里会标注「需产品确认」）：**
- **请假类型**：`parent/src/pages/leave/create.vue` 有「病假 / 事假 / 其他」三选项，但提交时并未带上（后端 `LeaveRequest` 也没有该字段）。原型在卡片上画一个小类型标签。
- **审批意见**：后端字段 `review_remark`，目前只有请假**详情**接口返回，列表接口未返回。原型在「已驳回」卡片上展示「审批意见：……」一行（已同意有意见时也可展示）。

**源码里没有、绝对不要画**：搜索框、日期区间筛选、多孩子切换器、批量操作、撤销/撤回按钮（只在详情页有）、附件缩略图、下拉刷新控件、「已读」标记、导出、底部 tabBar（子页）。

## 示例数据（中文，幼儿园语境；今天 = 2026年10月26日 星期一，本学期孩子是「乐乐 / 小一班」）

共 6 条，统计卡初始值：**全部请假 6 · 待审批 1 · 已同意 2**（1 待审批 + 2 已同意 + 2 已驳回 + 1 已取消 = 6）。

1. **待审批** · 病假 · 10月28日 ~ 10月29日（2天）· 提交于 10月26日 20:15 ——「发烧 38.5℃，医生建议在家休息两天，退了烧再回园。」
2. **已同意** · 事假 · 10月20日（1天）· 王老师 10月19日 ——「家里有事，需带乐乐回老家一趟。」审批意见：「已知悉，路上注意安全。」
3. **已驳回** · 病假 · 10月15日 ~ 10月18日（4天）· 王老师 10月14日 ——「感冒咳嗽，需要在家休养。」审批意见：「请假超过 3 天需补交医院证明，请补充后重新提交。」
4. **已同意** · 事假 · 9月30日（1天）· 李老师 9月29日 ——「参加亲属婚礼，当天不来园。」
5. **已驳回** · 其他 · 9月26日（1天）· 王老师 9月25日 ——「想带乐乐去参加社区亲子活动。」审批意见：「非必要事由，建议安排在周末，不占用在园时间。」
6. **已取消** · 病假 · 9月22日 ~ 9月23日（2天）· 家长 9月22日自行撤销 ——「肚子不舒服，先观察一天。」

日期写法统一「10月28日」；跨年不出现。

## 视觉与结构硬约束

- `<html lang="zh-CN" data-surface="parent">`，390px 手机视口，单列纵向流，不做桌面布局。
- **外壳必须原样使用共享 shell**：先读 `D:/yuanhaisu/personal_projects/kindergarten/docs/design-previews/2026-10-25-full-redesign/_shared/shell.md` 与同目录 `shell.css`，照抄 `mp-statusbar` / `mp-time` / `mp-status-icons` / `mp-navbar` / `mp-back` / `mp-title` / `mp-capsule` / `mp-main` / `mp-demo` 的标记与 SVG，不许自造状态栏。
- 引入顺序：`tokens.css` → `../../_shared/shell.css` → `styles.css`；末尾先 `<script src="../../_shared/shell.js"></script>` 再 `<script src="app.js"></script>`（app.js 监听 `document` 的 `mp:state` 事件）。
- **禁止把 shell.css 复制进项目**，也不要改写状态栏 SVG；只写 `<link href="../../_shared/shell.css">`，文件在当前项目里不存在也没关系（由流程拼装）。
- **styles.css 里绝对不许出现任何 `.mp-*` 选择器或覆盖**（唯一例外是 `[data-state]` 状态选择器，如 `.mp-app[data-state="empty"] .leave-list`）。
- **子页：必须画返回箭头**（`mp-back`）；导航标题是 `pages.json` 的 `navigationBarTitleText` —— **「请假记录」**。
- **不画 tabBar**（家长端只有首页/成长/接送/我的四个页面画，本页不是）。
- **演示条位置**：页面右下角有固定悬浮「＋ 请假」按钮，所以给 `mp-demo` 加唯一允许的行内覆盖 `style="bottom: calc(84px + env(safe-area-inset-bottom))"`，避免和按钮重叠。

### 设计方向（必须遵守）

**「一条请假 = 一张进度卡」**：整页是安静的中性底，只有状态徽章和新建按钮带颜色；家长扫一眼就知道哪条还在等老师。

- 整页底 `var(--bg)`；卡片 `var(--surface)` + `var(--radius-card)` + `var(--shadow-sm)`。暖色面积 ≤ 10%，暖色只出现在：待审批数字、选中的分段胶囊、待审批徽章/头像、右下新建按钮（渐变）。
- **统计卡**：一张白卡，`grid-template-columns: repeat(3, minmax(0, 1fr))`；数字 20px/600 `var(--fg)`（待审批用 `var(--accent)`），标签 11px `var(--muted)`；三段之间 1px `var(--border)` 竖分隔。不要渐变底、不要图标。
- **状态分段**：一行 4 个等宽按钮，`grid-template-columns: repeat(4, minmax(0, 1fr))`，高 44px，圆角 `var(--radius-pill)`，13px/500；未选中白底 + 1px `var(--border)` 描边 + `var(--fg-2)` 字；选中 `var(--accent)` 填充 + `var(--on-brand)` 字 + 600 字重。**不要数量角标**。
- **列表卡片**：`display: grid; grid-template-columns: 40px minmax(0, 1fr) 16px;`，列距 12px，内边距 14px，卡间距 10px。
  - 左：40px 圆形头像位，里面是孩子名字首字（「乐」）15px/600；**待审批**用 `var(--accent-wash)` 底 + `var(--accent-deep)` 字，其余状态用 `var(--bg)` 底 + `var(--muted)` 字。
  - 中：第一行「姓名 14px/600 `var(--fg)`」+ 右侧状态徽章（胶囊，11px/500，内边距 4px 10px；待审批 `var(--warning-wash)`/`var(--warning)`，已同意 `var(--success-wash)`/`var(--success)`，已驳回 `var(--danger-wash)`/`var(--danger)`，已取消 `var(--bg)`/`var(--muted)`）；第二行是小类型标签（「病假 / 事假 / 其他」，11px/500，`var(--bg)` 底 + `var(--fg-2)` 字，胶囊）+ 起止时间 12px `var(--muted)`「10月28日 ~ 10月29日 · 2天」；第三行事由 13px `var(--fg)`，最多两行截断。
  - 第四行（条件行）：已驳回/已同意且有事由意见时，展示一条**审批意见**块 —— 浅洗底（驳回 `var(--danger-wash)`，同意 `var(--bg)`）、圆角 `var(--radius-sm)`、内边距 8px 10px，12px `var(--fg-2)`，前缀「审批意见 · 王老师 10月14日」用 11px `var(--muted)`；待审批改展示「提交于 10月26日 20:15」11px `var(--muted)`，不加底。
  - 右：垂直居中 16px `var(--hint)` 的 `›`（提示可点）。
- **新建入口**：`position: fixed` 右下角胶囊按钮，文字「＋ 请假」（与源码按钮文案一致），高 48px、内边距 0 18px、`var(--accent-gradient)` 底、`var(--on-brand)` 字、14px/600、`var(--radius-pill)`、`var(--shadow-lg)`；`right: max(14px, calc(50vw - 195px + 14px))`、`bottom: calc(16px + env(safe-area-inset-bottom))`，保证在 390px 手机框内。
- 列表末尾一行居中 11px `var(--muted)`：「共 6 条请假记录」。
- 颜色只写 `var(--*)`，禁止裸 hex / rgb / hsl；无直角（最小 `var(--radius-sm)`）；阴影只用 `var(--shadow-*)`。
- 触控目标 ≥ 44×44px；正文 ≥ 13px（meta/徽章/脚注允许 11–12px）；`:active` 必须有反馈（`transform: scale(.99)` 或背景压暗）。
- **禁止 `white-space: nowrap`** 用于任何可能超宽的文本；多列统计用 `grid-template-columns: repeat(N, minmax(0, 1fr))`；所有容器 `max-width: 100%`，页面不得出现横向滚动；长事由自然折行（最多 2 行）。
- 不用 `<select>`、alert、prompt；无 console 报错；不引用任何外部资源、不内联 base64 图片；图标一律用内联 SVG `<symbol>` + `<use>`。

## 状态覆盖（`data-state` 驱动，演示条用 `mp-demo` 四芯片：正常 / 空态 / 加载 / 失败）

- **正常**：上面的 6 条数据完整渲染（默认）。
- **空态**：隐藏列表与「共 N 条」脚注，换成空态块 —— 44px 圆底（`var(--bg)`）+ `var(--hint)` 线性日历/文档图标 +「暂无请假记录」+ 一句说明「提交请假后，老师的审批结果会显示在这里」+ 主按钮「发起请假」。统计卡数字变 0 / 0 / 0。
- **加载**：统计卡三个数字换成灰条占位（`var(--bg)` 呼吸动画）；列表换成 3 张骨架卡（圆形头像占位 + 两行骨架条 + 一条短 meta 条）。
- **失败**：隐藏列表，换成错误块 —— `var(--danger-wash)` 圆底 + `var(--danger)` 圆形叹号图标 +「请假记录加载失败」+「网络不太稳定，请稍后再试」+「重试」按钮（品牌渐变，只保留一个主按钮）。统计卡保留但数字显示「—」。
- **禁用态（不占芯片）**：① 加载态下 4 个分段胶囊不可点：`opacity: .5` + `pointer-events: none`；② 失败态点「重试」后按钮立即 `disabled`、文案「重试中…」、降到 `var(--hint)` 视觉，约 1 秒后恢复并 toast「已重新加载」。
- 若某状态分段下无记录，显示块级空态「该状态下暂无请假记录」+ 按钮「查看全部」（不要另造第 5 个演示芯片）。

## 交互（app.js，全部真实可点）

- 点分段胶囊：切换选中态（`aria-pressed`），按 `data-status` 过滤卡片；**统计卡数字保持不变**；若该状态无记录，显示「该状态下暂无请假记录」+「查看全部」。
- 点任一请假卡：toast「已打开请假详情」（不要真的跳页）。
- 点右下「＋ 请假」/ 空态「发起请假」：toast「已打开发起请假」。
- 点失败态「重试」：按钮进入禁用忙碌态，1 秒后恢复并 toast「已重新加载」。
- 演示条芯片切换四态由 `_shared/shell.js` 写 `data-state` 并派发 `mp:state`；app.js 监听该事件做收尾（回到正常态时重置重试按钮、恢复统计数字）。
- toast 用页面自绘的轻提示（`var(--fg)` 底、`var(--on-brand)` 字、圆角胶囊、固定在中下部），2 秒自动消失；不用 alert。
- 所有按钮/卡片有 `:active` 反馈；无 console 报错。

## 输出文件（项目根目录，每个 < 20KB）
1. index.html —— 结构
2. styles.css —— 样式（引用 tokens.css）
3. app.js —— 交互
tokens.css 已由流程提供，**不要重写，不要覆盖**。不要生成 preview.png 或 notes.md。
