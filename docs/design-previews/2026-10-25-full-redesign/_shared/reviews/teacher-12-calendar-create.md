# teacher/12-calendar-create · 对抗性单页审查

- verdict：**WARN**（score 72/100）
- 源码覆盖：新建态覆盖完整（标题/类型/描述/日期/起止时间/全天/地点/通知、五态演示、日期与时间滚轮弹层），但源码同一页承担的编辑态整条分支全缺：`?id=` 预填、已发布活动保存前的变更确认弹层、notify-hint 与「保存发布」文案均未画。

## P1 · 明显缺陷（2）

### 1. [completeness] 源码 create.vue 同时承担「编辑已有活动」：详情页以 `?id=` 进入后拉详情预填、已发布活动点发布走变更确认弹层、发布设置换成 notify-hint、主按钮变「保存发布」；原型只画了纯新建态，编辑分支 0 覆盖。

- 证据：teacher/src/pages/calendar/detail.vue:36 `navigateTo({ url: `/pages/calendar/create?id=${event.value.id}` })`；teacher/src/pages/calendar/create.vue:154-179（onLoad 读 id + fetchTeacherCalendarEvent 预填）、140-152（published 时走 substantiveLines + confirmVisible）、233-239（notify-hint）、249-269（变更确认 sheet）、244（editId ? "保存发布" : "立即发布"）；原型 index.html:33 标题固定「添加活动」、index.html:185-187 按钮固定「发布活动」，app.js:1 无 location.search / id / editId 任何处理。
- 建议：补一个「编辑已发布活动」演示态：预填字段 + 发布设置里的 notify-hint + 点发布后弹出变更清单与通知开关的确认 sheet，主按钮文案切「保存发布」；若有意不做，需在 notes 中明确标注编辑态不在范围并说明原因。

### 2. [logic] 「必填未完成时常驻阻断说明」实际只在首次点发布被拦之后才生效：用户清空必填标题时，底部摘要仍显示完整发布预览、发布按钮仍为可用态，与 notes 的「常驻」声明相反。

- 证据：app.js:1 `bt()`/`pd()` 都以 `rq` 为前提（`var missTitle=rq&&mt()`、`if(rq&&(mt()||md()))return true`），而 `setMode('normal')` 把 rq 置 false；实测清空 #event-title 后 #submit-summary 仍为「活动 · 10月30日 09:00–11:30 · 通知家长」、#btn-publish aria-disabled="false"、#title-error 仍 hidden，点一次发布才变成「还差标题」+ aria-disabled="true"；notes.md:13、:33 声称「常驻阻断说明」。
- 建议：必填/时间判定改为无条件计算，`rq` 只控制错误文案是否显示；或在 input/change 时把 rq 置 true，保证摘要阻断与按钮禁用随字段实时更新。

## P2 · 打磨项（6）

### 1. [logic] 失败态自身不自洽：点「重试」不进入提交中、瞬间弹「已重新提交」；失败态里点「存草稿」成功后，顶部仍挂着「发布失败：网络异常，请重试」。

- 证据：app.js:1 `hr()` 只执行 `fb.hidden=true; mode='normal'; showToast('已重新提交')`，无 busy/延迟时序；实测 failed→#retry-btn 后立即 state=normal、toast=「已重新提交」、无任何 loading；failed→#btn-draft 后 1.2s，toast=「已存草稿」但 #failure-bar.hidden=false、文案仍为「发布失败：网络异常，请重试」。
- 建议：重试复用发布的 busy/延迟时序并在成功后清 failure-bar；`hd()` 草稿成功时也隐藏 failure-bar（或把 mode 从 failed 复位），避免成功操作与错误条同时出现。

### 2. [design-core] token 引用不合规：`--weight-regular` 在 tokens.css 中不存在（只有 `--weight-normal`），导致必填星号实际继承父级 500 字重；`.submit-dock`/`.picker-sheet` 顶部圆角硬编码 24px，绕过 `--radius-2xl`。

- 证据：styles.css:95 `.req{font-weight:var(--weight-regular)}`、styles.css:208 同；tokens.css:47 只定义 `--weight-normal:400`，无 `--weight-regular`；实测 `.field-label .req` computed font-weight=500（期望 400，因 invalid at computed-value 时间接继承 .field-label 的 500）；styles.css:346 与 styles.css:530 均写 `border-radius: 24px 24px 0 0`，而 tokens.css:57 有 `--radius-2xl: 24px`。
- 建议：两处 `--weight-regular` 改为 `--weight-normal`；圆角改为 `var(--radius-2xl) var(--radius-2xl) 0 0`。

### 3. [notes-accuracy] notes 的「信息层级」与「遗留」描述了原型里根本不存在的发布设置项（参与班级、说明、提醒设置、参与班级多选），还声称做了「同日重叠」时间冲突校验。

- 证据：notes.md:12「发布设置：参与班级、说明、提醒设置」、notes.md:38「参与班级多选的候选用示例数据，未接班级接口」、notes.md:37「时间冲突校验只做了同日重叠的提示」；原型 index.html:144-170 的发布设置只有「通知家长」开关 +「存为草稿不会通知家长」便笺；app.js:1 `tc()` 仅有 `data.end<=data.start` 先后校验，无冲突检测；源码 create.vue:231-240 也没有参与班级/说明/提醒设置。
- 建议：删除三处不存在的描述；若确实想保留「参与班级/提醒设置」作为设计提案，需在原型中画出来并在 notes 标注「设计提案 · 需产品确认」。

### 4. [redundancy] 开启「全天活动」后，开始时间与结束时间两行都渲染「全天」，加上开关本身与底部摘要，同一事实在一屏内重复四次。

- 证据：app.js:1 `renderTimes()` 在 allDay 时把 #start-value 与 #end-value 同时设为「全天」；实测 allDay 开：#start-value=「全天」、#end-value=「全天」、#submit-summary=「活动 · 10月30日 全天 · 通知家长」，而这两行已 aria-disabled="true" 置灰（index.html:99-112 行标签仍写「开始时间」「结束时间」）。
- 建议：全天开启时隐藏或合并时间两行（如只留一行「全天 · 不设具体时刻」），底部摘要保留一次即可。

### 5. [logic] 演示条里的「空态」被实现成校验错误态：切过去立即出现两条必填红错，页面没有真正的空态呈现。

- 证据：app.js:1 `setMode('empty')` 设 `rq=true` 并渲染 EMPTY；实测点「空态」后 #title-error.hidden=false、#date-error.hidden=false、#submit-summary=「还差标题和日期」、#date-value 变为 danger 红；index.html:219 该 chip 文案为「空态」。
- 建议：把 chip 改名为「必填未完成」（与 07/16/17 已有口径一致），或为空态单独设计不立即标红的呈现（一句说明 + 主行动）。

### 6. [design-core] 错误文字对比度不达 WCAG AA：行内必填错误 12px/3.62:1，底部失败条 12px/3.08:1。

- 证据：styles.css:321-331 `.field-error{color:var(--danger);font-size:var(--text-sm)}`、styles.css:351-365 `.failure-bar` 同；tokens.css:32 `--danger:#EF4444`、tokens.css:84 teacher `--surface:#fffaf0`、tokens.css:33 `--danger-wash:#FEE2E2`；实测 #title-error computed 12px rgb(239,68,68) on rgb(255,250,240)=3.62:1，#failure-bar 12px on rgb(254,226,226)=3.08:1（AA 要求 4.5:1）。
- 建议：错误文本改用更深的 danger 墨色（建议回 tokens.json 新增 `--danger-ink` 或调 `--danger` 至对 surface ≥4.5:1），不要靠加粗/放大硬凑。

## 2026-10-27 P1 修复记录

### 1. [completeness] 编辑态分支 0 覆盖 —— 已修

新增「编辑」演示态（chip `data-mp-state="edit"`）：按 `?id=` 预填已发布活动（11月2日 09:00–12:00），导航标题切「编辑活动」、发布设置的通知开关换成「该活动已发布…」提示、主按钮变「保存发布」、保存前弹「通知家长？」确认层（本次变更清单 + 通知开关，日期/时间/地点类变更预置开启），无变更时跳过弹层静默保存、摘要显示「无变更，不推送通知」。

### 2. [logic] 阻断说明只在首次点发布被拦后才生效 —— 已修

`bt()`/`pd()` 去掉 `rq`/`tq` 前置条件，必填与时间先后改为无条件实时计算（`rq` 只控制行内错误文案显隐）；清空标题即刻变「还差标题」+ 主按钮 `aria-disabled="true"`，重新输入立即恢复。

验证：`od-check` problems=[]；CDP 真实交互（发布/草稿/编辑预填/确认层开关/取消/Escape/遮罩/无变更静默保存）50 项全通过、无控制台错误；`od-audit` 本页无 ✗（仅 app.js 22KB 体积 ⚠）。
