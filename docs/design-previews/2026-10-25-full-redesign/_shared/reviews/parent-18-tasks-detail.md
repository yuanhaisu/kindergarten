# parent/18-tasks-detail · 对抗性单页审查

- verdict：**WARN**（score 70/100）
- 源码覆盖：覆盖了任务说明/要求/学习关联、未读提示、主回应+补充回应、今日提交、打卡历史、公开申请/取消与给孩子看的弹层；主要缺口是「回应独立于本周期提交」的组合态、补充回应反馈、当前提交的公开操作，以及错误态只画了 404 分支。

## P1 · 明显缺陷（4）

### 1. [logic] 老师回应/未读提示被绑定到「公开已通过」：源码中 responses、hadUnread 与 submitted_current_period 是三个独立条件，原型只在 approved 显示 .feedback/.whisper，submitted 态整块隐藏；「本周期已提交 + 老师已回应 + 公开待审核」这一常见组合看不到回应。反过来 approved 态又隐藏打卡区，「昨天有回应、今天未提交」时无法打卡。

- 证据：styles.css:956-972（submit-card/checkin-card 互斥，feedback/whisper 仅 approved 显示）；detail.vue:448-450、452-455、515（三个 v-if 相互独立）；06-task.md:115-118（responses 为主回应，与公开状态无关）。
- 建议：把回应与未读提示从 submit-card 拆成独立区块，严格按源码三个独立条件渲染；状态演示补一个「有回应 + 本周期待提交」组合态。

### 2. [logic] 当前周期提交被移出历史时间轴后，公开操作一并丢失且状态写错：「我的提交」卡没有任何申请/取消公开按钮，刚打卡就显示「已申请公开，等待老师审核」（实际未发申请，契约中申请是独立 POST）；approved 态也无法撤下当前这条，只能误操作 10月25日 的旧记录。

- 证据：index.html:87-145（submit-card 内无 data-public）；index.html:207/220/232（公开按钮只在 10-25/24/23 历史行）；app.js:77-83（提交后直接 setState('submitted')）；index.html:102-105（静态「已申请公开」文案）；detail.vue:535-543（源码动作在每条 history 行）；06-task.md:253-255（申请是独立 POST）。
- 建议：在「我的提交」卡带出公开状态与申请/取消按钮（两步确认），或把当前提交保留在时间轴顶部；提交后默认按 is_public_requested=false 显示「未公开」+「申请公开」。

### 3. [logic] 历史行的公开状态被压成三个混合态，与源码 publicState 四态不一致：①从未申请（none）的行标成「未公开」并配「申请公开」，源码 none 不显示状态，revoked 才是「未公开」+「重新申请」；②一键完成（无文字无媒体）的行也给「申请公开」，源码会隐藏且接口必 422；③缺 approved 但 public_visible=false 的「老师已通过，暂不对本班展示」。

- 证据：index.html:224-233（「（一键完成）」行 + 「未公开」徽标 + 「申请公开」）；detail.vue:107-130（四态文案与动作）、132-141（content||media_url 才给申请）、114（not-visible 分支）；06-task.md:249-251（一键完成申请 422）、285（public_status 语义）。
- 建议：按 publicState/publicLabel/publicActionLabel/showPublicAction 四函数逐分支实现：none 无状态文案且无内容不给申请、revoked 用「重新申请」、approved 按 public_visible 分流。

### 4. [redundancy] 同一个打卡总数在一页出现三次且空态不归零：任务卡「累计打卡 7 次」、历史标题「共 7 次」、脚注「共 7 条」；切到空态时前两处仍是 7，与同屏「还没有打卡记录」直接矛盾，点空态「去打卡」回正常态又冒出 3 条历史；单位「次/条」也不统一。

- 证据：index.html:72-75、186-187、237-246；styles.css:974-976（空态只隐藏 timeline/history-more，history-count 与 meta-grid 仍可见）；app.js:214-219（空态 CTA 切回 normal）；detail.vue:527-547（源码只有 total>length 时的一条 hint，没有「累计打卡」统计）。
- 建议：只保留历史脚注一处总数（源码口径），空态置 0 或隐藏；统一用「条」；若要保留「累计打卡」必须与 history.total 同源并随状态变化。

## P2 · 打磨项（4）

### 1. [completeness] 补充回应缺反馈组：源码对每条 supplement 单独渲染 REACTIONS，契约也允许对 supplement 单独 reaction（主回应与补充各自一个 my_reaction），原型只在主回应下有一组，家长无法对「老师又补了一句」反馈。

- 证据：index.html:133-142（supplement 只有「给孩子看」）；detail.vue:499-509（supplement 内 reactions）；06-task.md:119、221（补充回应反馈单独存在）。
- 建议：在 supplement 内补一组与主回应同构的三选一反馈（aria-pressed 单选 + 已选态 + toast）。

### 2. [logic] 「给孩子看」弹层永远显示「▶ 听老师说」，即使来源是没有语音的补充回应；点击只把文案改成「正在播放…」且没有超时复位，会一直停在播放态，看起来像音频坏了。

- 证据：index.html:289（固定播放按钮）；index.html:140（补充回应的 data-kid 无语音）；app.js:135-140（openKid 不区分是否有语音）、154-158（只 toggle 文案，无定时复位）；detail.vue:558（源码仅 media_url 存在才渲染播放按钮）。
- 建议：openKid 接收 hasVoice，无语音时隐藏 kidPlay；有语音时按 inline voice 的定时器/结束回调复位播放态。

### 3. [design-core] 两个交互控件低于 44×44 触控下限：多孩条「切换查看」高 36px、照片移除 32×32；其中「切换查看」11px 白字压 --warning 约 2.9:1，未达 WCAG AA。

- 证据：styles.css:170-180（height:36px；background:var(--warning); color:var(--on-brand)）；styles.css:590-605（width/height:32px）；index.html:38-41、165。
- 建议：两者最小 44×44；按钮改用 --fg 等深色文字压 --warning，或在 tokens.json 加深 warning 后统一替换。

### 4. [notes-accuracy] notes 的「与现状的差异」与实现不符：①称 --accent-gradient 只用于「打卡」主按钮，实际语音播放态也用了整条渐变；②称只补了 end_date 与 checkin-history.total 两个源码未渲染事实，实际还渲染了「星星罐 20 星」和历史照片缩略图（字段本身见 A16/A24，这里只报 notes 声明不实）。

- 证据：notes.md:43 vs styles.css:132、457-460；notes.md:53 vs index.html:68-71（星星罐）、198-204（历史缩略图）；detail.vue:529-544（源码历史模板无 media、无 star_goal）。
- 建议：notes 据实列出第二处渐变与所有新增渲染字段并标注需产品确认；或收回第二处渐变、移除未确认字段。

## 2026-10-27 P1 修复记录

1. **[logic] 老师回应/未读提示与公开状态解耦** — 已修：回应从「我的提交」卡拆成独立的「王老师的回应」卡，`.feedback` 在已提交 / 已通过 / 有回应·未提交三态显示，`.whisper` 在已提交 / 已通过显示；演示条新增「有回应·未提交」组合态，该态下回应可见且今日打卡区可用。
2. **[logic] 当前周期提交的公开入口与默认状态** — 已修：「我的提交」卡新增公开状态行（徽标 + 说明 + 申请/取消公开，两步确认）；打卡后默认 none（无徽标、仅「申请公开」），已通过态可直接撤下当前这条，不再只能操作旧历史行。
3. **[logic] 历史行公开状态对齐 publicState 四态** — 已修：10/25 approved+visible「老师已通过 · 本班家长可见」、10/24 approved+not visible「老师已通过 · 暂不对本班展示」、10/23 pending「待审核 · 已申请，等待老师审核」、10/22 revoked「未公开 · 重新申请」、10/21 一键完成行无徽标无入口；none 不再显示「未公开」。
4. **[redundancy] 打卡总数重复三处** — 已修：删除任务卡「累计打卡」与历史标题「共 7 次」，只保留脚注「仅显示最近 5 条提交（共 7 条）」，空态随历史区一起隐藏，单位统一为「条」。

P2 四项（补充回应反馈组、给孩子看的播放态复位、36/32px 触控目标、notes 声明细节）本轮按要求未动。
