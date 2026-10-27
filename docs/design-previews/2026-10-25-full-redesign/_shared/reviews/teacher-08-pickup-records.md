# teacher/08-pickup-records · 对抗性单页审查

- verdict：**WARN**（score 75/100）
- 源码覆盖：源码的日期栏、三段统计、记录卡（学生/时间/类型/接送人/核验方式/例外/理由/异常）以及空、加载分支均已覆盖；仅按 notes 移除了只弹「开发中」的筛选按钮，源码的「已接走/已入园」状态徽章并入类型标签，无实质缺口。

## P1 · 明显缺陷（1）

### 1. [logic] 失败态与统计条同时成立：页面提示「没取到 10月26日 的接送记录」，但日期卡内统计仍显示 13/4/9 条，等于宣称加载失败却展示完整数据。

- 证据：app.js:230-235 只在 phase==='loading' 时跳过 updateSummary，error 仍会执行 app.js:249-258；styles.css:584-588 只隐藏 .normal-only，index.html:50-69 的 .date-card/.summary 不在其中；实测 data-state="error" 时 #sumTotal 文本为 "13"，#errorDesc 为 "网络异常，没取到 10月26日 的接送记录，请重试"。
- 建议：error 态把统计数字隐藏或改为「—」，或把文案改成「显示的是上次成功结果，点重试更新」；不要让「没取到」和 13 条同时出现。

## P2 · 打磨项（7）

### 1. [logic] 日期弹层固定为 10/20–10/26 七天，但左箭头可无限回退；回退超过 7 天后打开弹层没有任何选中项，副标题仍写「最近 7 天」。

- 证据：app.js:10-18 DATE_WINDOW 写死 7 天；app.js:503-505 prevDay 无下限；实测连点 7 次后日期为「10月19日 周一」、组头「共 20 条」，打开弹层 .date-option.is-selected 数量为 0，首个选项仍是「10月26日 周一 今天 13 条」，sheetSub 仍为「最近 7 天」（index.html:165）。
- 建议：按 state.date 动态生成最近 7 天，或给 prevDay 加窗口下限，并保证弹层总能标出当前选中日期。

### 2. [logic] 底部弹层打开时背景页面仍可滚动，没有滚动锁定；遮罩只是 fixed 覆盖，未阻止底层滚动。

- 证据：app.js:324-349 openSheet/closeSheet 未修改 body 或 .mp-app 的 overflow；styles.css:828-835 .sheet-layer 仅 position:fixed；实测打开 dateSheet 时 body 的 computed overflow 为 "visible"。
- 建议：openSheet 时给 body/.mp-app 加 overflow:hidden（iOS 记录并恢复 scrollY），closeSheet 时还原。

### 3. [redundancy] 同一天与同一条数在页面顶部重复出现：日期卡显示「10月26日 周一」，紧接的组头又写「10月26日 周一 · 共 13 条」，而统计条已经给出「记录 13 条」。

- 证据：index.html:44-45（dateMain/dateWeek）与 index.html:74-76（groupDate/groupCount）、index.html:54（sumTotal=13）；实测 DOM 文本 groupTitle="10月26日 周一 / 共 13 条"、summary="13/4/9"，两处相隔不足一屏。
- 建议：组头只保留「共 N 条」或只保留班级；日期只在日期卡出现一次，条数只在统计条出现一次。

### 4. [redundancy] 「小一班」在 13 条记录里逐行重复，而组头右侧已经显示班级；同一信息在一屏内出现 14 次。

- 证据：app.js:307 每行拼接 CLASS_NAME；index.html:78 .group-class 已显示「小一班」；实测 .tl-row .tl-class 数量为 13，groupClass 文本为「小一班」。
- 建议：删除行内 .tl-class，班级信息只留在组头；若同日跨班，再按班级分组分段。

### 5. [redundancy] 详情弹层的时间出现三次：副标题、hero 右侧、明细「记录时间」行，内容完全重复。

- 证据：app.js:443 detailSub 输出「10月19日 周一 · 17:26」；app.js:430 detail-time 输出「17:26」；app.js:401 detailRow('记录时间', ...) 输出「10月19日 周一 17:26」；实测 #detailBody 文本三处同时出现。
- 建议：保留 hero 时间 + 副标题日期，删掉「记录时间」明细行（或反之）。

### 6. [design-core] 多个主要可点控件高度只有 40px，低于 brief 规定的 44×44 触控目标。

- 证据：styles.css:70-71 .date-arrow 40×40（实测 prevDay 为 40×40）；styles.css:103 .date-current height:40px；styles.css:801 .state-btn min-height:40px（实测空态 CTA 为 89×40）；styles.css:985 .btn-primary min-height:40px。
- 建议：统一提升到 min-height:44px，或用透明外扩命中区补足 44×44。

### 7. [notes-accuracy] notes.md 两处声明与实际不符：声称有「选择日期的 toast」，但全文件只有返回按钮一个 toast；声称「列表契约不返回 operator」，但契约的列表响应明确包含 operator。

- 证据：notes.md:23「返回与选择日期的 toast」；app.js:499-501 是唯一 showToast 调用；notes.md:42「列表契约不返回 pickup_phone / operator」；backend/api/api-teacher/07-pickup.md:240、253 的列表项均含 "operator"。
- 建议：删掉「选择日期的 toast」；把 operator 说明改为「列表实现未 preload Operator，真机为空」，避免把实现缺口误写成契约缺口。

## 2026-10-27 P1 修复记录

### 1. [logic] 失败态与统计条同时成立 — 已修

失败态下统计条不再是过期条数，三项统一显示「—」占位：`index.html` 三个 `.summary-cell` 各加一个 `.summary-dash`（`aria-hidden`，仅装饰），`styles.css` 在 `[data-state="error"]` 下隐藏 `.summary-value`、显示 `.summary-dash`。实测 `data-state="error"` 时统计条文本为「— 记录 — 接走 — 入园签到」，与「没取到 10月26日 的接送记录」文案不再矛盾；正常态仍是 13/4/9，加载态仍是骨架。`notes.md` 的「失败」一条同步改为「统计条三项改为 — 占位」。

P2 七条按范围未动（日期弹层窗口下限与选中项、弹层滚动锁定、组头/行内班级重复、详情时间三处重复、40px 触控目标、notes 两处表述）。

验证：`od-check` problems=[]；`od-shot` 已重出 `preview.png`；`od-audit` 本页无 ✗（仅原有体积 ⚠）。
