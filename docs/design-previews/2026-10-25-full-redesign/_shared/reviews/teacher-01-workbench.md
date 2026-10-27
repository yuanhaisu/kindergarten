# teacher/01-workbench · 对抗性单页审查

- verdict：**WARN**（score 72/100）
- 源码覆盖：源码的问候区、4 项今日概况、8 个功能入口、3 条待办及同意/拒绝/转发/去记录动作、空/加载/错误/禁用四态均已覆盖；缺源码底栏的「班级」「我的」两个主入口（原型全站无替代落点），下拉刷新未做手势表现。

## P1 · 明显缺陷（1）

### 1. [navigation] 源码底栏的「班级」「我的」两个主入口在原型全站都没有落点，入口链断裂；notes 声称的兜底（其余在「更多」里）与源码不符。

- 证据：teacher/src/pages/index/index.vue:272-289 的自绘底栏含 /pages/class/index（:277）与 /pages/profile/index（:285）；teacher/src/utils/nav.js:13-18 的 TEACHER_TABS 同 4 项。原型 index.html:78-160 只有 8 个功能入口，铃铛只去消息，无班级/我的入口；teacher/src/pages/more/index.vue:18-29 的 10 个菜单项也不含这两页，故 notes.md:104「其余在「更多」里」不成立。
- 建议：在问候区头像或「更多」页补「班级」「我的」入口（不必画 tabBar，但不能没有可达路径），并修正 notes 的兜底说明。

## P2 · 打磨项（7）

### 1. [logic] 待办动作后的状态不闭环：同意只改被点按钮且不更新「待审批」计数，同一行的「拒绝」仍可点；拒绝成功后整行与计数完全不变。

- 证据：app.js:96-103 只把被点的 approve 按钮改成「已同意」，index.html:181 同行的「拒绝」未禁用，点击仍会打开拒绝弹窗；app.js:120-129 拒绝确认后只 toast，行内容与 index.html:66-69 的「待审批 3」都不变。源码 index.vue:125-131 在任何 action 成功后都 await loadHome() 重拉，行会消失、计数会刷新。
- 建议：动作成功后把整行置为结果态（或移除该行），同步递减「待审批」；已同意/已拒绝后整行按钮都应失效。

### 2. [logic] 空态与数字带自相矛盾：切到空态后仍显示「待审批 3」，同屏文案却是「今天没有新的待办」。

- 证据：styles.css:367-372 的空态只隐藏 .todo-list，未动数字带；index.html:66-69 的待审批 3 与 index.html:213-223 的空态文案同时可见。源码 index.vue:188 在 pending_approvals 缺失时回退到 todos.length，空列表时应为 0。
- 建议：空态下把待审批归零或隐藏/替换数字带（如「今日暂无待处理」），四态共用同一份演示数据。

### 3. [logic] 日期与星期对不上：页面写「2026年10月24日 星期五」，2026-10-24 实际是星期六（星期五对应的是 2025-10-24）。

- 证据：index.html:55；notes.md:10 照抄同一串。brief.md:74 的示例「10月24日 周五」本身不带年份，页面补上 2026 后就错了。
- 建议：改为「2026年10月24日 星期六」，或去掉年份沿用「10月24日 周五」。

### 4. [notes-accuracy] notes.md 有四处与实现不符，其中「导航栏不再吸顶」把不存在的回归写成了已接受的偏差。

- 证据：(1) notes.md:97 称 .mp-navbar 是普通流内元素、会随页面滚走，但 _shared/shell.css:50-52 是 position: sticky; top: 20px（状态栏 shell.css:24-25 也是 sticky）；(2) notes.md:22 写「待办区 2 条骨架行」，index.html:226-238 实为 3 条 .sk-row；(3) notes.md:24 写禁用态用 --hint 文字，styles.css:329 实为 var(--muted)；(4) notes.md:10 把日期放在「问候区」，实际在 index.html:53-56 的「今日班级概况」标题行。
- 建议：按实现重写这四条；吸顶那条直接删除，避免把 shell 的正确行为当成已知退化。

### 5. [design-core] 「已同意」禁用态的背景与卡片背景同色，降级几乎只靠 1px 边框，达不到「明显降级」。

- 证据：tokens.css:84 给 teacher 的 --surface 是 #fffaf0，而 tokens.css:12 的 --surface-warm 也是 #fffaf0 且未被 teacher 覆盖；styles.css:326-333 用 background: var(--surface-warm) 作禁用底，落在同样 #fffaf0 的 .todo-card 上，填充色差为 0。
- 建议：禁用底改用 var(--bg)（#f7f3ea）或为 teacher surface 定义更深的 --surface-warm；属 token 层改动，改 tokens.json 后重建。

### 6. [design-core] 关键标签与说明文字低于 brief 的「正文 ≥13px」下限：8 个入口主标签、统计标签、日期都只有 11px，拒绝原因弹窗的说明句 12px。

- 证据：brief.md:45「正文 ≥ 13px」；styles.css:231-237 .entry-label 用 --text-xs=11px（tokens.css:39），styles.css:195-200 .stat-k 11px，styles.css:119-125 .section-meta 11px，styles.css:452-459 .dialog-sub 12px（整句「填写后会同步给家长，请说明具体原因。」）。
- 建议：入口/统计/日期标签至少用到 --text-sm(12px)，弹窗说明升到 --text-base(13px)；若全局偏小，回 tokens.json 调 text-xs/sm 后重建。

### 7. [logic] 两个弹层都没有滚动锁定，也没有焦点陷阱：打开「拒绝原因」或「转发」后背景仍能滚动，Tab 可穿到遮罩后的按钮。

- 证据：app.js:20-33 的 openOverlay/closeOverlay 只切 is-open 与焦点，无 overflow 锁定或 inert/焦点圈；styles.css:406-431 的 .overlay 为 fixed 但未锁背景；页面内容高于一屏（preview.png 390×1000 @2x），滚动可见。关闭时焦点恢复已实现（app.js:28-33），打开时没有。
- 建议：open 时给滚动容器加 overflow:hidden 并在弹层内做 Tab 循环，close 时解除；背景加 aria-hidden/inert。

## 2026-10-27 P1 修复记录

- **P1-1 已修**：问候区补「班级」「我的」两个可达入口，不画 tabBar（不变更批次硬约束）。头像由装饰性 `<span aria-hidden>` 改为按钮，`data-open="我的"`（`aria-label="我的，王老师"`）；「小一班 · 主班老师」由段落改为按钮，`data-open="班级"`，行尾加 10px chevron 提示可点，并用 `::after { inset: -10px -4px }` 把 ~20px 高的文字行扩到 ~40px 触控区（不改变行距）。两处都走本页既有的 `[data-open]` toast 通道，与另外 8 个入口同一条交互路径。经实测问候区几何未变（名字底 → 班级行顶仍为 2px，问候区内容高度 46.7px），4 个源码 tab 全部可达：首页=本页、消息=铃铛、我的=头像、班级=班级行。
- **P1-1（notes 部分）已修**：`notes.md` 信息层级第 2 条补入两个入口；遗留第 1 条删除不成立的「其余在『更多』里」（`teacher/src/pages/more/index.vue` 的 10 项与原型 `17-more-index` 均不含这两页），改为记录问候区兜底路径；「要不要做真正的自定义底栏组件」仍作为产品决策保留。

**验证**：`od-check --width 390` → `problems: []`（`scrollWidth = clientWidth = 390`，`overflowing: []`）；`od-shot --width 390 --dsf 2` 重出 `preview.png`（CSS 390×1000）；`od-audit` 批次 50 页全部通过、本页无 ✗。CDP 探针（headless Chrome，390×900）12 项断言全过：头像为 `BUTTON` / `data-open=我的` / `aria-label=我的，王老师` / 40×40 / 无 `aria-hidden`；班级行为 `BUTTON` / `data-open=班级` / chevron 可见；点头像 toast「已打开 我的」、点班级行 toast「已打开 班级」、铃铛仍为「已打开 消息中心」；班级行下方 +8px、上方 −8px、左侧 −3px 的 `elementFromPoint` 均命中该按钮（触控区已扩大）；console 0 error。P2 七项按要求未动。
