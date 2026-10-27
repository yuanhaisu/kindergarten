# teacher/18-messages-index · 对抗性单页审查

> 2026-10-27 后补：本页已加共享 4 项底栏（消息高亮 + 未读角标）；下文为当时审查记录。

- verdict：**WARN**（score 72/100）
- 源码覆盖：已覆盖全部/通知/审批/系统四类筛选与条数、12 条列表（通知已读进度、请假状态）、空/加载/错误/禁用及「全部已读」；缺源码的触底分页（onReachBottom）与下拉刷新，分类空态在现有数据下不可达。

## P1 · 明显缺陷（2）

### 1. [logic] 「全部已读」的启用判定按全量未读，点击守卫却按当前分类未读，导致按钮可点但静默无反应

- 证据：app.js:312 `var unavailable = countUnread(messages) === 0;` 与 app.js:381 `if (countUnread(messagesFor(currentFilter)) === 0) return;` 作用域不一致。实测：切到「通知」并点掉唯一未读后，摘要为「共 5 条 · 0 条未读」，按钮 disabled=false、aria-disabled=false，点击无新 toast（仍停留在上一次「查看：秋游亲子活动通知」），切回「全部」仍为「共 12 条 · 3 条未读」。notes.md:21 声称按全量未读判定，与实现不符。
- 建议：把守卫改为 countUnread(messages) === 0，与禁用条件同一作用域并执行全局置读；或把按钮改为仅作用于当前分类并改文案（如「本类已读」），同时让摘要与动作口径一致。

### 2. [logic] 空态/失败态仍显示分类分段及其条数，与「暂无消息」「消息加载失败」直接矛盾

- 证据：index.html:87-102 的 .inbox-toolbar 位于 .view.view-normal（index.html:104 起）之外，styles.css:426-437 只隐藏 .view-normal；实测 data-state=empty 与 error 时 .inbox-toolbar 仍 display:block 且可见，文案为「全部12 通知5 审批4 系统3」，同屏显示「暂无消息」/「消息加载失败」。notes.md:18 却称「整页空态」。同批 09-tasks-index/styles.css 在 empty/error/loading 下会隐藏 .filter-count，本页未做。
- 建议：把 .inbox-toolbar 移入 .view-normal，或在 empty/error/loading 下隐藏 .seg-count 并禁用分段切换；没有数据时不应展示已加载的条数。

## P2 · 打磨项（6）

### 1. [completeness] 通知已读进度读取契约中不存在的字段 recipient_count，按原型实现会渲染「已读 9/undefined」

- 证据：app.js:53/68/103/138/153 的 extra 使用 recipient_count，app.js:236 拼接 `extra.read_count + '/' + extra.recipient_count`；backend/api/api-teacher/10-messages.md:98 定义的是 total_count（并另有 read_rate）。
- 建议：改用 extra.total_count，或直接消费 read_rate；并在 notes 的字段映射中写明数据来源。

### 2. [notes-accuracy] notes 两处称源码失败时「只 toast」，源码实际是静默清空列表、没有任何提示

- 证据：notes.md:20 与 notes.md:39 均写「源码 catch 后只 toast」；teacher/src/pages/messages/index.vue:98-99 的 catch 只有 `if (reset) allMessages.value = []`，无 toast、无日志——失败会被渲染成「暂无消息」空态。
- 建议：改为「源码 catch 静默清空、无 toast、无错误态，失败会伪装成空列表」，这正是原型补整页错误态的理由。

### 3. [notes-accuracy] 列表主行由源码的固定 sender 改为消息 title、标签由 category/status 改为类型徽章，notes 的「与现状的差异」未记录这一信息层级变更

- 证据：index.vue:185 主行渲染 msg.sender（TYPE_CONFIG 固定为「班级通知/审批提醒/系统消息」），index.vue:194-196 标签渲染 msg.category；原型 app.js:273 主行渲染 item.title、app.js:227 输出类型徽章；notes.md:33-40 的差异清单无此条。
- 建议：在「与现状的差异」补记：主行改用契约真实字段 title（源码未用），标签改为类型徽章 + category/status 明细，并说明原因。

### 4. [logic] 示例消息中有两处星期与真实日期不符，属用户可见的数据错误

- 证据：app.js:130「10月18日（周六）9:30…家长会」，但 2026-10-18 是周日；app.js:170「周一早高峰堵车…（10月13日 至 10月13日）」，但 2026-10-13 是周二。同页 app.js:45 的「10月30日（周五）」则正确。
- 建议：按 2026 年日历改正：10月18日改为周日（或改到 10月17日周六）；10月13日改为周二（或把请假日改到 10月12日周一）。

### 5. [redundancy] 请假消息同一行三处重复「审批/请假」语义：类型徽章「审批」+ 状态「待审批」+ 标题「…请假申请」

- 证据：app.js:227 徽章输出「审批」，app.js:239-244 紧随输出「待审批」（如 m1：app.js:34 标题「乐乐妈妈提交的请假申请」、app.js:39 status_label「待审批」），实际渲染为「审批 待审批」。
- 建议：请假行去掉类型徽章（左侧图标已表意），只保留状态；或把状态并入徽章，避免同义并列。

### 6. [logic] 时间标签为写死文案，不遵守源码 formatRelativeTime 的规则，7 天内也显示绝对日期

- 证据：app.js:111-112 m7 的 created_at 为 2026-10-20T16:40 却写死 timeLabel「10月20日」；index.vue:24-39 的规则是 <7 天返回「N天前」。同页 app.js:87 又使用源码不会产出的「昨天 18:20」格式，两套规则并存。
- 建议：由 created_at 计算标签并沿用源码规则；若要改用微信式「昨天 HH:mm」，需在 notes 声明并统一 7 天内规则。

## 2026-10-27 P1 修复记录

- P1-1 [logic]「全部已读」启用判定与点击守卫口径不一 —— **已修**。`app.js` 的点击守卫由 `countUnread(messagesFor(currentFilter))` 改为 `countUnread(messages)`，与 `updateMeta` 的禁用条件同看全量未读，点击后执行全局置读。实测：切到「通知」读完唯一未读后按钮仍可点，点击 toast 变为「已全部标为已读」，切回「全部」为「共 12 条 · 0 条未读」且按钮转禁用。
- P1-2 [logic] 空态/失败态仍显示分类分段及条数 —— **已修**。把 `.inbox-toolbar` 移入 `.view-normal`，随 `[data-state="empty|loading|error"] .view-normal { display: none }` 一并消失，不需要重复写状态选择器。实测三态下分段控件 rect 为 0×0、四个条数均不可见，不再与「暂无消息」「消息加载失败」同屏矛盾。
- 附带修复（P1-1 入口可达性）：`.inbox-toolbar` 的 `position: sticky` 原为 `top: 64px`，但滚动容器是 `.mp-main`（自带 `overflow-x: hidden`，其上沿已在导航栏之下），sticky 框会下移 64px 且不占流 —— 实测正好压住「共 N 条 · M 条未读 + 全部已读」整行，`elementFromPoint` 命中的是 `.seg-btn`，按钮不可点。改为 `top: 0` 后 toolbar 占 72–144、口径行 146–190、列表 198 起，按钮可点击。注：同批 `teacher/14-leave-index`、`parent/19-tasks-wall` 也用 `top: 64px`，属批次级，本轮未动；sticky 在本原型外壳下不真正吸顶（窗口滚动、`.mp-main` 不滚），要修需改 `_shared/shell.css`，同样批次级。
- 验证：`node scripts/od-check.mjs ... --width 390` → `problems: []`；`node scripts/od-audit.mjs ...` → 本页 `issues: []`；`preview.png` 已用 `od-shot --width 390 --dsf 2` 重出。
