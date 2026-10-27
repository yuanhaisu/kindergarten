# teacher/11-calendar-index · 对抗性单页审查

- verdict：**WARN**（score 80/100）
- 源码覆盖：源码的月份条/月历卡/当日列表三区块、完整字段、草稿与全园分支、编辑权限、空/加载/失败/禁用四态与跳转入口均已覆盖，但缺少下拉刷新（enablePullDownRefresh/onPullDownRefresh）的对应处理，类型元数据也没有按字典动态化。

## P1 · 明显缺陷（1）

### 1. [logic] 失败态仍显示月份条的派生统计「本月 11 个活动」，与错误块「未获取到本月活动」正面矛盾；加载态也在数据未到达前先报出具体数量，且错误态下月份箭头仍可点击、把月份文字和统计一起改掉而错误块不变。

- 证据：index.html:45 静态「本月 11 个活动」；app.js:81-84 renderMonth 只对 empty 特判，error/loading 仍输出真实 count；styles.css:561-566 error 只隐藏 cal-card/day-skeleton/day-section，.month-bar 保留；index.html:121「网络异常，未获取到本月活动。」；app.js:257-265 的 shiftMonth 在 error 态同样重渲染 monthLabel/monthSub（如 2026年11月 / 本月 2 个活动）。
- 建议：error 态把副标题改为「—」或「活动加载失败」，或整条 month-bar 只保留月份与箭头；loading 期间同样不输出具体计数；error 态禁用月份箭头，直到重试成功。

## P2 · 打磨项（6）

### 1. [logic] 空态/错误态并不重建月历网格，日期格的 aria-label 仍按真实数据播报「N 个活动」，屏幕阅读器听到的内容与页面宣称的「本月暂无活动」冲突。

- 证据：app.js:117 渲染时把 `${dayEvents.length} 个活动` 写进 aria-label；app.js:274-279 applyState 只切 grid class + renderMonth/renderDay，不调用 renderGrid；styles.css:566 空态仅对圆点做 visibility:hidden。
- 建议：applyState 中补一次 renderGrid，或空态渲染时从 aria-label 中去掉活动数（只留日期）。

### 2. [redundancy] 无活动的当天，日期与「没有活动」这一结论被重复表述三次：日期行、计数胶囊、空态标题；空态标题又把日期说了一遍。

- 证据：app.js:236-240 dayDate 写「10月24日 周六」、dayCount 写「暂无活动」、emptyTitle 写「10月24日暂无活动安排」；index.html:84/86/93 三行在同一卡片内相邻出现。
- 建议：空态时收起 day-head 的日期与计数，或把空态标题改为不带日期的「暂无活动安排」，让三处各自只承担一种职责。

### 3. [notes-accuracy] 空态块多出一句未在 notes/源码登记的承诺文案「可以添加一场活动，家长端会同步看到」，而从空态去添加时保存的是草稿，家长端并不会看到。

- 证据：index.html:94；notes.md:16 只列「日历图标 + 10月26日暂无活动安排 + 添加活动」，未记录该句；源码 index.vue:138 仅一行提示；家长端查询硬过滤 status=published（server/internal/repository/calendar_repo.go:42），创建页「存为草稿」走 draft（teacher/src/pages/calendar/create.vue:134）。
- 建议：删除该句，或改为「保存并发布后家长端可见」；同时在 notes.md 的「与现状的差异」中登记这条新增文案。

### 4. [completeness] 类型 label/颜色被硬编码成 5 类，图例与图标不随字典走；字典扩展出的自定义类型会回退成 muted，与「家长会」同色，视觉上无法区分（源码有 is-unknown 中性降级并优先读字典 label）。

- 证据：app.js:22-28 固定 TYPES；app.js:105-141 图例只遍历 TYPES；app.js:145 未命中回退 tone:'muted'；app.js:154 图标首字取硬编码 meta.label；styles.css:72 与 app.js:26（meeting 同为 muted）；源码 index.vue:68-70/143 用字典 label，124/130/195/200/210 有 is-unknown；字典明确内置 5 类可改名/停用、园所可扩展（server/internal/app/seeder.go:89）。
- 建议：图例按当月实际出现的类型动态生成；图标首字与类型名优先用 event_type_label；未命中类型改用不与内置类重合的中性 tone（或照源码 is-unknown 的灰），避免与家长会同色。

### 5. [logic] 加载骨架固定 35 格 5 行，而月历任何月份都输出 42 格 6 行，加载完成时月历与当日列表都会明显跳高。

- 证据：app.js:311-316 固定 35 个 sk-cell；app.js:68-78 buildCells 恒为 42 格（firstWeekday+天数 28–37，remaining 补到 42，CAP 14 永不触发截断）；styles.css:450-455 骨架格 40px/gap 6 vs styles.css:106 真实格 min-height 47px/row-gap 2；index.html:99-114 当日骨架仅 12px 标题条 + 两张矮卡。粗略差值：月历约 68px、当日区约 100px+。
- 建议：骨架改成 42 格 6 行且格高与真实格一致；当日骨架补足标题行与卡片底栏（44px）高度，减少切换跳动。

### 6. [completeness] 源码启用了下拉刷新并绑定了刷新处理，原型没有对应的手势/提示，也没有在 notes 里标注为未覆盖项。

- 证据：teacher/src/pages.json:72（pages/calendar/index 的 enablePullDownRefresh:true）；teacher/src/pages/calendar/index.vue:3,104（onPullDownRefresh(loadEvents)）；原型 index.html/app.js 无任何下拉相关处理，notes.md 未提及。
- 建议：在 notes.md 的「遗留/与现状的差异」注明原型未做下拉手势；若要覆盖，可增加一个「下拉刷新」演示态或在空/失败态提供刷新入口。

## 2026-10-27 P1 修复记录

- 已修 · P1-1 [logic] 失败/加载态月份条统计与错误块矛盾：`app.js` 的 `renderMonth()` 在加载态把副标题改为「正在加载…」、失败态改为「活动加载失败」，两态均不再输出具体数量；新增 `syncMonthNav()` 在失败态禁用左右月份箭头（`styles.css` 补 `.month-nav:disabled { opacity: .4 }`），点「重新加载」回到正常态后恢复。`notes.md` 的覆盖状态与「与现状的差异」已同步。
- 本轮只处理 P1，6 条 P2 未动。
