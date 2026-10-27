# teacher/10-tasks-detail · 对抗性单页审查

- verdict：**WARN**（score 70/100）
- 源码覆盖：对照 detail.vue，任务信息、四列统计、待回应/已回应、审核四态、批量回应、单条贴纸/加星/录音、语音播放、编辑与生成小程序码、空/加载/失败四态均已覆盖；缺任务详情加载失败分支、结构未锁定时的可编辑分支，以及一诺/可心两张已回应卡的补充回应入口。

## P1 · 明显缺陷（5）

### 1. [logic] 朵朵卡片提交时间 10月24日 18:40 处于任务周期 10月20日~10月26日 之内，却被标记「旧周期 · 请逐条回应」，与源码 isCurrentPeriod 的判定相反，周期提示与日期自相矛盾。

- 证据：index.html:51 周期「10月20日 ~ 10月26日」；index.html:196 朵朵时间「10月24日 18:40」；index.html:199「旧周期 · 请逐条回应」；源码 detail.vue:72-75 isCurrentPeriod 仅在 row.date 落在 period_start/period_end 之外时为旧周期。
- 建议：把朵朵的日期改到周期外（如 10月18日）以演示旧周期提示，或删除该卡的旧周期标记，确保样例数据与周期判定一致。

### 2. [logic] 编辑弹层里确认「关闭公开」后，页面立即写入 page.dataset.public="off"，随后点「取消」或关闭弹层不会回滚；破坏性改动脱离了表单的保存/取消语义，与源码「先暂存、保存时才提交」不一致。

- 证据：app.js:214-223 关闭公开的 dialogAction 直接执行 page.dataset.public = "off"；app.js:272-279 与 index.html:539 的「取消」只调用 closeSheet，不恢复状态；源码 detail.vue:478-501 仅在确认后改 edit.allow_public，真正提交发生在 saveEdit。DOM 实测：确认关闭→取消后 page.dataset.public 仍为 off，重开弹层开关 aria-checked=false。
- 建议：将 allow_public 改为弹层内暂存值，仅在点「保存」时应用到页面状态；「取消」或关闭弹层时恢复原公开状态。

### 3. [logic] 保存编辑时只校验任务标题，「学习关联」清空后仍能保存成功，导致 hero 上「学习关联」标签保留而内容为空；源码要求标题与学习关联同时必填。

- 证据：app.js:316-323 只判断 title 非空；源码 detail.vue:512-515 同时校验 edit.title 与 edit.learning_link；index.html:55 的 .hero-learning-text 会被 app.js:326 写成空串。DOM 实测：清空 #taskLinkInput 后点保存，toast 显示「已保存」，.hero-learning-text 变为空。
- 建议：保存前增加 learning_link 非空校验并就地提示；若不希望显示空字段，至少在 hero 中隐藏空的「学习关联」行。

### 4. [completeness] 「补充回应」入口只在果果卡出现；一诺（语音回应）与可心（班级星星回应）同样有主回应，却完全没有补充回应入口，而源码对所有 responded 且 row.response 存在的行都渲染补充区。

- 证据：index.html:296-315 果果卡有 .supplement-toggle；index.html:359-366（一诺）与 index.html:378-382（可心）在 .response-block 后直接结束，无 data-supplement-toggle；源码 detail.vue:1065-1079 条件为 status === 'responded' && row.response。DOM 实测 responded 列表 3 卡中仅 1 卡存在补充入口。
- 建议：为所有存在主回应的已回应卡补上「补充回应（个别夸一句）」入口，保持源码的全量可用性。

### 5. [redundancy] 四列统计里的「3 待回应 / 3 已回应」与紧邻分段的「待回应 3 / 已回应 3」是同一组数字重复；统计、分段、审核筛选、批量条四层连续堆叠，390 宽下第一条家长提交在约 633px 处才开始，首屏看不到任何提交内容，与「一页看清谁交了」的目标相悖。

- 证据：index.html:63-80 四列统计，index.html:82-89 紧接着的分段再次显示 3/3；index.html:91-104 还有审核 chip 与批量条。DOM 实测：.mp-app 宽度 390px，.list[data-list="pending"] top=633px（844px 视口），第一张 .checkin top=633px。
- 建议：合并统计与分段（计数只保留一处），把审核筛选收进折叠筛选或与分段合并，保证首屏露出至少一条提交卡片。

## P2 · 打磨项（3）

### 1. [logic] 录音进行中不阻断其它回应动作：点乐乐的「录音」后点小雨的 ❤️，仍弹出「已送爱心贴纸 · 小雨」的成功提示；源码在录音期间用 recordingBlocked() 拦下所有贴纸/加星/批量动作。

- 证据：app.js:387-419 录音状态只用于判断其它录音按钮；app.js:459-461 通用 data-toast 处理器没有检查 recording；源码 detail.vue:622-631 respond()/respondStar()/batchRespond() 均先调用 recordingBlocked()。DOM 实测：录音中点击小雨的 data-toast，toast 文本为「已送爱心贴纸 · 小雨」。
- 建议：录制期间禁用或拦截数据贴纸、加星与批量按钮，并提示「请先停止当前录音」，与源码状态机对齐。

### 2. [redundancy] 空态下「生成小程序码」重复出现两次：空态主按钮 #emptyShareBtn 与底部任务设置卡里的 #shareTaskBtn 同屏可见；空态样式只隐藏列表/汇总条/审核 chip，没有隐藏任务设置卡。

- 证据：index.html:113 空态「生成小程序码」；index.html:398-402 任务设置卡「生成小程序码」；styles.css:971-974 空态仅隐藏 .list/.batch-strip/.review-filter/.filter-empty。DOM 实测 empty 态两者均可见（top 664 / 890）。
- 建议：空态隐藏任务设置卡，或移除空态里重复的「生成小程序码」按钮，只保留一个入口。

### 3. [notes-accuracy] notes.md 多处声明与实现不符：(a) 声称「批准/拒绝/撤回/关闭公开都走微信居中对话框二次确认」，实际「批准公开」没有确认框；(b) 声称失败态重试后 toast「已重新加载」，实际 retry 只切换状态不弹提示；(c) token 映射把任务标题写成 --text-3xl、统计数字写成 --text-2xl，实际正好相反。

- 证据：notes.md:43 与 app.js:154-166（data-approve 点击后直接 lockCard 并 setReview，无 openConfirm）；notes.md:30 与 app.js:491-492（retryBtn 仅 setState("normal")）；notes.md:52 与 styles.css:89（.hero-title 用 --text-2xl）及 styles.css:208（.stat-num 用 --text-3xl）。
- 建议：修正 notes.md 的交互与 token 声明，或补齐「批准公开」二次确认与重试 toast，保证 notes 与页面实际行为一致。

## 2026-10-27 P1 修复记录

- **P1-1 [logic] 已修**：朵朵卡片提交时间由 `10月24日 18:40` 改为 `10月18日 18:40`（落在周期 10月20日~10月26日 之外），卡片 `.when`、`data-time` 与媒体弹层时间同步，样例数据与「旧周期 · 请逐条回应」判定重新一致。
- **P1-2 [logic] 已修**：编辑弹层里的公开开关改为弹层内暂存（`pendingPublic`）；确认关闭只提示「已关闭公开，保存后生效」，不再立刻写 `page.dataset.public`；点「保存」才应用，「取消」/关闭弹层不改动已公开状态（重开弹层 `aria-checked` 回到真实值）。
- **P1-3 [logic] 已修**：保存前同时校验任务标题与学习关联（对齐 `saveEdit`），学习关联为空时 toast「学习关联不能为空」并聚焦输入框，hero 的「学习关联」不再被写成空串。
- **P1-4 [completeness] 已修**：一诺（语音主回应）与可心（班级星星主回应）补上「补充回应（个别夸一句）」入口与同一套补充面板（4 贴纸 + 录语音 + 提示 + 收起），三张已回应卡入口一致，对应源码 `status === 'responded' && row.response`。
- **P1-5 [redundancy] 已修**：四列统计、两段分段、审核 chip 合并为一张「统计卡」——摘要行只保留 `28 应交 / 6 已提交`（待回应/已回应计数只出现在分段），审核 chip 收进同卡；首卡堆叠由「统计 + 分段 + chip + 批量条」四层降为「统计卡 + 批量条」两块。390×844 下 `.list` 首卡 top 由 633px 提前到 603px，首条提交的姓名、正文与照片完整落在首屏内（照片底 831px < 844px）。

P2（3 条）按要求未处理：录音期间不拦截其它回应、空态「生成小程序码」重复入口、notes 的批准确认/token 映射表述校正。notes.md 已按 P1 行为变化同步（信息层级、空态口径、公开开关暂存语义、学习关联必填、补充回应入口）。
