# teacher/14-leave-index · 对抗性单页审查

> 2026-10-27 已修复本页 P0（审批时间 NaN），修复与验证见 `_shared/REVIEW.md` 2.1。

- verdict：**FAIL**（score 55/100）
- 源码覆盖：源码的统计卡、待审批/最近已审批两段、卡片字段（孩子/家长/状态/时间/事由）、同意/拒绝动作与两处空态均已覆盖；缺口是头像图片分支被简化为文字首字、审批时源码会写入的「同意」意见未体现、下拉刷新只有文案没有手势原型。

## P0 · 用户可见硬伤（1）

### 1. [logic] 点击「同意」或「确定驳回」后，新审批记录显示「审批 10月NaN日」，展开审批留痕显示「10月NaN日 00:00」——核心动作产生可见数据错误。

- 证据：app.js:34 `NOW_STAMP = "2026-10-26 12:00"` 用空格分隔，而 parseDate/parseTime 只按 "T" 切分（app.js:61-69）；approve/reject 把 reviewedAt 写成 NOW_STAMP（app.js:360、375），formatDay/formatStamp（app.js:87-95、75-85）得到 d=NaN。实测：同意 id=1 后 .card-time 文本为「审批 10月NaN日」，.trace 为「10月NaN日 00:00」；驳回 id=3 同样；notes.md:20 声称「已用 CDP 逐条跑通」。
- 建议：把 NOW_STAMP 改为 "2026-10-26T12:00"，或让 parseDate/parseTime 兼容空格分隔；补交互回归断言（审批后卡片时间不得含 NaN）。

## P1 · 明显缺陷（3）

### 1. [logic] 状态分段条的「滚动吸顶」不生效：sticky 声明存在，但滚动后过滤器随页面上移消失。

- 证据：styles.css:106-113 声明 position: sticky; top: 64px；实测 scrollTo(0,250) 后 .filter-sticky 的 rect.top=-72px，scrollTo(0,600) 后 -422px（应停在 64px），同期 .mp-navbar 稳定在 20px。根因是 .mp-main 计算 overflow-y:auto（shell.css:130-134）成为最近滚动容器却不滚动（clientHeight=scrollHeight=1628）。notes.md:10 宣称「滚动吸顶」。
- 建议：让 sticky 相对文档滚动（把过滤器移出 .mp-main，或让 .mp-main 真正滚动并把 top 改为 0）；在 od-check 中增加 390 视口滚动后 filter 顶部位置断言。

### 2. [logic] 统计卡口径标注错误：整卡标为「本月请假统计 / 统计为本月全量」，但后端只有「本月总计」按月，待审批/已同意/已拒绝是班级全量；分段计数又只统计当前 20 条，三处数字在真实数据下会互相矛盾。

- 证据：index.html:40（sr-only 标题「本月请假统计」）、index.html:59（「统计为本月全量 · 列表显示最近 20 条」）；server/internal/service/teacher_portal_service.go:732-742 仅 monthly_total 用 monthStart 过滤；app.js:229-239 的分段计数来自当前 records（8 条）；契约 backend/api/api-teacher/05-leaves.md 的 stats 响应里也只有 monthly_total 带月份语义。
- 建议：卡片标题改为「请假统计」，只给第四项保留「本月总计」；分段计数改读服务端 ?status= 的 total，或明确标注「最近 20 条内」；删除误导性脚注。

### 3. [logic] 合并成单列后只按提交时间倒序，待审批不置顶；旧的待审批申请会被近期已审批挤出第一屏，与设计目标相反，也弱于源码的独立待审批区块。

- 证据：app.js:243-247、274 只按 createdAt 排序，未按 status 优先；源码 teacher/src/pages/leave/index.vue:112-159 用独立「待审批申请」区块保证可见；notes.md:28 宣称「把「现在要处理什么」提到第一屏」。示例数据里 3 条待审批恰好是最新 3 条（app.js:40-42），掩盖了该问题。
- 建议：全部视图按「待审批优先、再按提交时间倒序」排序，或默认筛选落在「待审批」；列表头写明排序规则。

## P2 · 打磨项（4）

### 1. [redundancy] 列表头「最近 20 条」与「共 8 条」并列矛盾；8 在统计卡、分段、列表头重复三次；已审批卡只显示审批时间，却写「按提交时间倒序」，排序口径无法核对。

- 证据：index.html:80-81（listTitle「最近 20 条」/ listSub「共 8 条 · 按提交时间倒序」）；index.html:55、65、81 三处出现 8，index.html:59 再出现 20；app.js:148-150 历史卡 timeText 取 reviewedAt，排序却用 createdAt（app.js:274）。
- 建议：列表标题改为语义名（全部申请/待审批），页大小从可见文案移除；已审批卡同时显示提交与审批时间，或把排序说明改为按审批时间。

### 2. [completeness] 示例数据日期与事由自相矛盾：10/13 的请假写「周一早高峰堵车」，但 2026-10-13 是周二；且「晚到半天」按整天记录（共 1 天）。

- 证据：app.js:46（reason「周一早高峰堵车，晚到半天」，startDate/endDate 均为 2026-10-13）；实测 2026-10-13 为周二、2026-10-12 才是周一；formatRange 显示「共 1 天」（app.js:97-106）。
- 建议：把该条日期改为 2026-10-12，或把事由改成周二；「半天」改成整天事由，或在契约/模型补半天字段（当前没有）。

### 3. [logic] 打开拒绝原因对话框时没有锁定背景滚动，遮罩下页面仍可滚动。

- 证据：app.js:328-339 openRejectDialog 只设置 hidden 与 focus，没有滚动锁；实测 dialog 打开时 window.scrollTo(0,300) 生效（scrollY 0→300）。关闭时焦点恢复已实现（app.js:341-350），但滚动位置未处理。
- 建议：打开时给 body/滚动容器加 overflow:hidden 并记录 scrollTop，关闭时还原；或用 inert 背景。

### 4. [notes-accuracy] notes 声称覆盖「按状态筛选为空」的三种文案，但「已同意 / 已拒绝」两个空态在原型里不可达，对应分支是死代码。

- 证据：notes.md:15 宣称「刷新后按状态筛选时若该状态为空：显示对应文案…」；app.js:258-265 为 approved/rejected 写了空态分支，但记录只能 pending→approved/rejected（app.js:359、374），已同意 4 条、已拒绝 1 条只会增加，永远无法筛空；演示条也没有对应状态（index.html:130-133 只有正常/空/加载/失败）。
- 建议：要么删掉不可达分支和 notes 声明，要么给演示条补一个「筛选空」状态或提供可清空历史记录的重置入口。

## 2026-10-27 P1 修复记录

- P1-1 [logic] 状态分段条「滚动吸顶」不生效 —— **已修**：shell 的 `.mp-main { overflow-x: hidden }` 会把 `overflow-y` 计算成 `auto`，生成不滚动的滚动容器。本页给 `<main>` 追加 `.leave-main`，以 `overflow-x: clip` 覆盖（保留横向裁切，不建立滚动容器），筛选条改为相对文档吸顶；实测滚动 250 / 600 后 `.filter-sticky` 的 top 恒为 64（`.mp-navbar` 仍为 20）。
- P1-2 [logic] 统计卡口径标注错误 —— **已修**：`sr-only` 标题改为「请假统计」；删除「统计为本月全量」页脚，改为「待审批/已同意/已拒绝为班级累计 · 分段与列表取最近 20 条」；第四项保留「本月总计」；`notes.md` 同步口径说明。
- P1-3 [logic] 待审批未置顶 —— **已修**：列表统一按「待审批优先，再按提交时间倒序」排序（`sortForDisplay`），列表头副文案显式写明该规则；示例数据首屏现为 1/3/2 三条待审批，历史记录随后。

P2 项未动。

验证：`od-check --width 390` → `problems: []`；`od-audit` 批次 50/50 通过、本页无 ✗；CDP 实测吸顶 top=64、首屏卡片顺序 1/3/2、同意/驳回/筛选/空/加载/失败全链路无 console error、无 NaN。
