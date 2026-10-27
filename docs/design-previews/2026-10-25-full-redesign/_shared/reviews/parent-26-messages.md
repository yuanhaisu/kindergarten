# parent/26-messages · 对抗性单页审查

- verdict：**WARN**（score 72/100）
- 源码覆盖：8 条示例 + 卡片三要素（标题/摘要/时间）+ 正常/空/加载/失败四态齐备，但源码「列表为空 + 加载失败」的整页错误分支与「已有数据时刷新不显示加载」的分支都无法在原型里到达。

## P1 · 明显缺陷（1）

### 1. [logic] 失败态把已加载的消息整块 disabled + pointer-events:none，刷新失败后用户一条旧消息都打不开；源码刻意保留列表可点，notes 却把这个行为说成「对应源码」。

- 证据：26-messages/styles.css:621-624（[data-state="error"] .inbox-groups{opacity:.45;pointer-events:none}）+ 26-messages/app.js:122（setCardDisabled(state === 'error')）；实测点「失败」chip 后 .msg-card.disabled===true、groups pointer-events:none、点卡片无任何反应。对照 parent/src/pages/messages/index.vue:104-122：失败时只在列表顶部插 inline-error，卡片仍绑 @click、无 disabled。notes.md:16 写「已有列表保留但整块降到 45% 且不可点，对应源码『刷新失败不清空列表』的行为」——「不可点」在源码里没有对应物，「不清空」的理由（还能读）也被抵消。
- 建议：失败态去掉 disabled 与 pointer-events:none，只在列表上方叠错误条（与 source 的 inline-error 同构）；若坚持锁列表，需在 notes 标为「与源码不同、待产品确认」，并同步把口径行的「3 条未读 / 共 8 条」降级，避免对刚刚失败的数据下断言。

## P2 · 打磨项（6）

### 1. [logic] 口径行右侧「共 N 条」由渲染出的卡片数算出，不是接口的 total；而源码只拉第一页（page_size 默认 20）且没有触底加载，真机上线会恒显示「共 20 条」并把更早的消息永久藏掉。

- 证据：26-messages/app.js:32-34（totalText.textContent = String(cards.length)）；backend/api/api-common.md:417（响应明明带 total）；server/internal/handler/http/message.go:22（page_size 默认 20）；parent/src/pages/messages/index.vue:73（fetchMessages() 不传任何分页参数，全文件无 onReachBottom/loadMore）。左侧「3 条未读」来自 unread-count 服务端聚合、右侧「共 8 条」来自客户端渲染数，同一行两套口径。
- 建议：「共 N 条」绑定接口 total；notes 补一条「源码无分页/触底加载，第 21 条以后看不到，落地前需产品定」。

### 2. [notes-accuracy] notes 声称「列表本身为空时只剩错误卡」，但页面写死 8 张卡、没有任何清空逻辑，该分支永远不可达；同理「已有数据时刷新」也不会出现加载态（源码只有列表为空才显示 loading）。

- 证据：notes.md:16 末句；index.html:100-230 固定 8 张 .msg-card；app.js:119-124 applyState 只切 data-state、不清列表；实测切到「失败」后 #inboxGroups 仍渲染 8 张卡（opacity .45）。源码 parent/src/pages/messages/index.vue:69（if (messages.length===0) loading=true）与 :100-102（空列表失败走整页「{{loadError}}，点击重试」）。
- 建议：二选一：让「失败」芯片先清空列表以复现源码的整页错误，或新增「空+失败」样本；同时在 notes 删掉未实现的分支描述，并说明与批次「严格 4 态」约定（A22/A42）的取舍。

### 3. [design-core] 已读消息的正文摘要与元信息被降到 --hint（#94A3B8）压白底，对比度仅约 2.57:1，8 张卡里 5 张处于该状态——回看旧消息几乎不可读；降灰应只作用于次要信息。

- 证据：26-messages/styles.css:321-324（.is-read .msg-summary,.is-read .msg-meta{color:var(--hint)}）+ tokens.css:17（--hint:#94A3B8）；实测 computed color rgb(148,163,184)、font-size 12px，#FFFFFF 底对比 2.57:1（--muted 为 4.76:1）。_shared/findings.md 的 token 缺口 3 已记录 --hint 压底不可读的同类问题，但此处压在正文而非脚注。
- 建议：已读正文留在 --muted，只用标题字重/图标位退回中性来区分已读；或把降灰限定在 meta 行（类型/时间）。

### 4. [redundancy] 类型标签与标题措辞重复：8 张卡里 5 张标题已含类型词，meta 行再写一遍，加上左侧彩色图标，同一信息在一张卡里出现三次。

- 证据：index.html:110-112（标题「请假申请已通过」+ .msg-type 请假）、:125-127（「新亲子任务：周末亲子阅读打卡」+ 任务）、:159-161（「新增 3 张成长照片」+ 成长）、:174-176（「秋季亲子运动会报名通知」+ 通知）、:208-210（「请假申请已提交」+ 请假）；仅「乐乐已入园/接送」「秋季体检安排/系统」「乐乐已被接走/接送」三张的标签有增量信息。
- 建议：只对标题不含类型词的条目显示标签（接送/系统），或整体去掉标签、靠图标色 + 标题；保留时至少去掉与标题重复的 5 条。

### 5. [design-core] 加载骨架的类型位是 40px 正圆（pill），加载完成后变成 12px 圆角方块，骨架→内容切换时形状跳变。

- 证据：26-messages/styles.css:524-531（.skel-tile 40px + border-radius:var(--radius-pill)）对照 styles.css:189-200（.msg-tile 同为 40px 但 border-radius:var(--radius-xl)，即 12px）；实测 loading 态 .skel-wrap display:block、skel-tile 为圆形。
- 建议：骨架 tile 改用 var(--radius-xl)，与真实类型位一致；其余骨架块保持 pill。

### 6. [logic] 示例数据的时间链自相矛盾：同一天里「10/27–10/28 请假已批准（09:12）」与「07:58 已刷卡入园」并存，且该申请在 10/23 就「已提交」，读起来是孩子请假在家的同时又被送进了园。

- 证据：index.html:111（「乐乐的请假申请（10月27日-10月28日）已被王老师批准」，时间 09:12，属于「今天」组）；index.html:141（「乐乐已于 07:58 刷卡入园」，同属「今天」组）；index.html:209-210（「请假申请已提交」时间 10月23日）。状态栏 9:41 说明「今天」即 10/27。
- 建议：把请假区间改成 10/28–10/29（或把「已通过」挪到 10/26），或将入园/接走两条换到不同日期，保证同一孩子同一天不会既「请假生效」又「在园」。

## 2026-10-27 P1 修复记录

- **P1-1 [logic] 失败态把已加载消息整块 disabled + pointer-events:none** — **已修**。删除 `styles.css` 的 `[data-state="error"] .inbox-groups{opacity:.45;pointer-events:none}` 与 `app.js` 的 `setCardDisabled()`、点击处 error 态短路；失败态仅保留列表上方的错误卡（与源码 `inline-error` 同构），8 张卡照常可读可点，点击仍置读 + toast。`notes.md` 的失败/重试/错误态三处描述同步改写，「不可点」不再被说成源码行为。
- P2 六项按要求未动。
- 验证：`od-check` problems=[]；失败态实测 `groups.pointer-events=auto`、`opacity=1`、无 disabled、点卡后未读 3→2 且 toast 出现；`od-audit` 50 页全过，本页无 ✗。
