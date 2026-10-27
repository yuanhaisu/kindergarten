# 老师工作台（老师端）

- **源码**：`teacher/src/pages/index/index.vue`（另读 `teacher/src/api/teacher-home.js`、`teacher/src/utils/nav.js`、`teacher/src/pages/more/index.vue`、`backend/api/api-teacher/03-home.md`）
- **设计系统**：`user:nurture-planet-local`，`surface=teacher`
- **方向**：「今天」优先——把「班上什么情况」和「我现在要处理什么」提到第一层，8 个功能入口退到第二层。用一条数字带替掉四色彩虹，用一张分格卡替掉三张各自投影的待办卡。

## 信息层级（从上到下）

1. **系统外壳**：20px 状态栏（9:41 + 信号/wifi/电池）+ 44px 导航栏（「老师工作台」居中）+ 右上胶囊。根页面，无返回箭头。
2. **问候区**：头像（姓名首字）+ 「王老师」+ 「小一班 · 主班老师」+ 日期「2026年10月24日 星期五」；右侧铃铛，未读时带红点。**头像即「我的」入口、班级行即「班级」入口**（行尾 chevron 提示可点），铃铛进消息。底部使用共享 `mp-tabbar`（首页 / 班级 / 消息 / 我的，本页「首页」高亮）；问候区入口保留为快捷方式。
3. **今日概况数字带**：已出勤 26 / 请假 2 / 待审批 3 / 未读通知 5。26 + 2 = 全班 28 人，与页面语境自洽。
4. **功能入口 8 格（4×2）**：发布通知 / 成长记录 / 接送管理 / 亲子任务 / 班级日历 / 请假审批 / 班级相册 / 更多。顺序与源码一致。
5. **待办事项**：3 条（源码 `visibleTodos = todos.slice(0, 3)`），每条带真实 action 与文案：同意 / 拒绝 / 转发 / 去记录。
6. **底部安全区**：内容区在共享 `mp-tabbar` 之上结束；演示条固定在底栏上方。

## 覆盖状态

| 状态 | 表现 |
|---|---|
| 正常 | 默认。完整问候 + 4 格数字 + 8 入口 + 3 条待办 |
| 空 | 待办区换成图标位 + 源码原文案「今天没有新的待办，保持轻松节奏。」+ 主按钮「写今日记录」；数字带保留 |
| 加载 | 数字变骨架块、待办区 2 条骨架行；不用全屏转圈 |
| 错误 | 待办区「班级数据加载失败」+「重试」按钮 |
| 禁用 | 点过「同意」的待办行按钮变「已同意」灰态（`--hint` 文字 / `--surface-warm` 底 / `aria-disabled`），再点无效 |

状态切换用右下角「原型演示」胶囊：默认收起，点击展开四个 chip（正常 / 空态 / 加载 / 失败），状态写到 `.mp-app[data-state]`，由 `_shared/shell.js` 统一管理。

## token 映射

| 元素 | token |
|---|---|
| 页面底 | `--bg` / `--bg-gradient` |
| 卡片底 | `--surface`；暖调区块 `--surface-warm` |
| 文字层级 | `--fg`（标题）/ `--fg-2`（正文）/ `--muted`（说明）/ `--hint`（禁用、占位） |
| 强调 | `--accent`、`--accent-deep`（文字）、`--accent-wash`（浅底）、`--accent-gradient`（仅主按钮） |
| 分割线 | `--border` |
| 阴影 | `--shadow-sm` / `--shadow-md` / `--shadow-lg` |
| 圆角 | `--radius-card`（卡片）、`--radius-pill`（按钮/标签/头像）、`--radius-sm`（最小） |
| 字号 / 字重 | `--text-xs…--text-3xl`、`--weight-normal/medium/semibold`（层级只靠字重+字号） |
| 间距 | `--space-xs…--space-xl`、`--page-pad-x` |

## 与现状的差异

- **去掉整屏品牌渐变 header**：现状 `$gradient-brand` 铺满首屏 + 白卡浮起，暖色过半屏，违反 Warm-Dominance（≤10%）。改为中性底色直接排问候区，accent 只留在「待审批」与主操作上。
- **4 个数字去彩虹**：现状绿/黄/红/蓝四色同权重。改为统一 `--fg`，只有「待审批」用 `--accent` + `--accent-wash` 胶囊底——它是唯一需要动作的数字。
- **入口图标位图 → 内联 SVG 单色线性图标**：现状是 8 张位图（`/static/icons/*.png`），统一暖橙但 8 个同重方块无主次。原型的 SVG 用 `currentColor`，语义靠图形区分，并把「更多」降到 `--muted` 音量。
- **3 张待办卡 → 1 张分格卡**：现状每条独立 `--shadow-sm` 卡片 + 6 种按钮配色（绿/红/蓝/紫 ×2）。改为一张卡 + `--border` 分隔线，操作分主次（主操作 `--accent-wash` / `--accent`，次操作透明底 `--muted`）。
- **操作按钮移到第二行右对齐**：现状与标题同行，390px 下触控目标被压到约 32px 高。改为独立一行，保证 ≥44px。
- **补骨架屏**：现状只有 `loading` 布尔值用于抑制空态，没有加载视觉。按共享约束补骨架。

## 外壳回归（2026-10-26）

统一到 `_shared/shell.md` 的 canonical shell，不再自画外壳。只动外壳，内容区设计未改。

### 换掉的类

| 旧 | 新 |
|---|---|
| `.screen`（根容器，`id="screen"`） | `.mp-app` |
| `.page`（内容区） | `.mp-main.page` |
| `.statusbar` / `.statusbar-time` / `.statusbar-icons` | `.mp-statusbar` / `.mp-time` / `.mp-status-icons` |
| `.navbar` / `.nav-title` | `.mp-navbar` / `.mp-title` |
| `.capsule` / `-dots` / `-line` / `-ring` | `.mp-capsule` / `-dots` / `-divider` / `-ring` |
| `.proto` 整套（panel/chip/bar/open/close） | `.mp-demo` + `.mp-demo-chip[data-mp-state]` + `.mp-demo-toggle` |

- 状态栏图标换成 shell.md 固定三份 SVG（17×11 / 15×11 / 24×11），不再自绘。
- **不画 `mp-back`**：工作台是入口根页面。
- **画 `mp-tabbar`**：老师端 4 个常驻页统一底栏；本页「首页」高亮。
- 引入顺序改为 `tokens.css` → `../../_shared/shell.css` → `styles.css`，`</body>` 前先加载 `shell.js` 再加载 `app.js`。

### 删掉的样式

- `.screen` 容器块（宽/居中/底色改由 `.mp-app` 提供）。
- 整段 `.wx-chrome` / `.statusbar*` / `.navbar` / `.nav-title` / `.capsule*`，含 `.wx-chrome.is-scrolled` 的滚动分割线。
- 整段 `.proto*`（面板、chip、胶囊条、关闭按钮）。
- `styles.css` 中已无任何 `.mp-*` 类定义；保留的只有 `.mp-app[data-state="…"]` 属性选择器（shell.md 明确允许）。

### 四态

`.screen[data-state=…]` → `.mp-app[data-state=…]`，共 8 处选择器（待办三态 + 数字带骨架）。状态唯一写入方是 `shell.js`，页面 CSS 只做属性选择器响应。

### JS 删改

- 删：`onScroll` 滚动分割线、原型面板开合、chip 切换、`P` 键恢复面板、`Escape` 关面板那半段。（`Escape` 关闭遮罩/对话框保留。）
- 留：入口宫格 toast、铃铛、查看全部、同意 / 拒绝 / 转发 / 去记录、拒绝原因对话框、分享 sheet、空态「写今日记录」。
- 「重试」原本调 `setState('normal')`，现改为点击 `normal` chip —— 保证 chip 选中态与 `data-state` 由同一个出口同步；无 chip 时回退直接写属性。
- 「已同意」灰态的回滚改挂在 `mp:state` 事件上（`shell.js` 切状态时派发），行为与旧的 `setState → resetApprovals` 一致。

### 验证

- `od-check --width 390`：`problems: []`（无横向溢出、无越界元素、无 console 报错）。
- DOM 实测：状态栏 20px、导航栏 44px、胶囊 87×32、无 `.mp-back`、有 `.mp-tabbar`（4 项，首页高亮）；四个 chip → 四态切换、chip `aria-pressed` 同步、重试回正常、「同意」灰态与回滚均正常。
- `od-audit`：本页 `issues: []`。

### 视觉变形（外壳统一优先，已接受）

1. **导航栏不再吸顶**：旧 `.wx-chrome` 是 `position: sticky` + 滚动加分割线；共享外壳的 `.mp-navbar` 是普通流内元素，页面整体滚动时状态栏与导航栏一起滚走。长内容页会明显。要恢复得改 `_shared/shell.css`（批次级决策），页面侧不允许覆盖 `.mp-*`。
2. **状态栏文字色加深**：旧用 `--fg-2`，shell 用 `--fg`，略深一档。
3. **演示条横向位置**：旧 `right: max(10px, calc(50vw - 185px))` 贴着 390 列；shell 用视口 `right: 12px`，在宽于 390 的预览窗口里会漂到手机框外。390 视口下与旧版视觉一致，属 `shell.css` 行为，批次级。
4. **状态栏图标细节**：尺寸与图标种类不变，但 SVG 路径换成 shell.md 的版本，电池/信号笔画略有差别。

## 遗留 / 需要批次层面决策

1. **底栏已补**：`teacher/src/pages.json` 没有 `tabBar` 字段，但源码在首页 / 班级 / 消息 / 我的 4 个常驻页自绘了固定底栏，并用 `utils/nav.js` 的 `goTab` 规避 `navigateTo` 页面栈溢出。2026-10-27 按产品决定给这 4 页补共享 `mp-tabbar`。落地前仍需把源码的自绘底栏正式化（抽组件或落 `pages.json` 的自定义 tabBar），否则设计稿与源码仍是两套实现。
2. **待办 action 契约漂移**：`backend/api/api-teacher/03-home.md` 里「通知未读提醒」的 actions 是 `["remind"]`，但前端 `todoActionLabel` 只映射 `approve / reject / share / go_record`，`remind` 会落到兜底文案「处理」；且只有 `share` 走 `open-type="share"`。原型按**前端实际能力**画「转发」，未画「一键提醒家长」。这条要么改契约、要么改前端映射。
3. **下拉刷新未体现**：`enablePullDownRefresh: true` 是真实能力，静态原型不表现，未做下拉手势模拟。
4. **老师端个人号限制**：与本页无关，但同批次其他老师端页面涉及 `web-view` 时需注意白屏（个人号不支持）。
