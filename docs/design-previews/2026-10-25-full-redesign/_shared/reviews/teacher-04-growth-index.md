# teacher/04-growth-index · 对抗性单页审查

> 2026-10-27 已修复本页 P0（弹层/toast 改 fixed），修复与验证见 `_shared/REVIEW.md` 2.1。

- verdict：**FAIL**（score 55/100）
- 源码覆盖：源码的今日统计、快速记录入口、搜索、StateBlock 三态、月分组卡片与同步/AI 标签都有对应，缺的是源码卡片的「今天 HH:mm / 完整日期时间」日期口径，且新增的班级筛选在示例数据里无法区分。

## P0 · 用户可见硬伤（1）

### 1. [logic] 底部筛选弹层和 toast 用 position:absolute 挂在内容高 2477px 的 .mp-app 上，弹层实际开在文档底部、视口之外；点两颗筛选 chip 只看到整屏遮罩变暗、看不到面板，发布/返回/打开详情/重试的 toast 也全部在视口外，主操作看起来完全没反应。

- 证据：styles.css:1078-1085 `.sheet-layer{position:absolute;inset:0}`、styles.css:1223-1226 `.toast{position:absolute}`，而 shell.css:7-20 的 `.mp-app` 仅 `position:relative;min-height:100dvh`，高度被内容撑到 2477px。实测 893px 视口、scrollY=0：打开班级筛选后 `.sheet` top=2218.4/bottom=2501.0，`.toast` top=2371.5，均大于 innerHeight；同批次 teacher/07-pickup-index/styles.css:576-578 等页均用 position:fixed。
- 建议：把 .sheet-layer 与 .toast 改为 position:fixed（与 07/08/10 等页一致），或固定 .mp-app 高度并让 .mp-main 内部滚动；toast/弹层不应挂在可变高度容器上。

## P1 · 明显缺陷（2）

### 1. [logic] 页脚「已显示最近 N 条」用全量 RECORDS.length 而不是当前渲染行数；时间筛选「本月（10月）」只剩 4 条时，页脚仍写「已显示最近 7 条」，搜索时同样失真。

- 证据：app.js:396 `listFoot.textContent = '已显示最近 ' + RECORDS.length + ' 条 · 下拉刷新'`。DOM 实测选 2026-10 后 `.record-card` 数量为 4，`#listFoot` 文本仍是「已显示最近 7 条 · 下拉刷新」。
- 建议：用 filteredRows() 的 rows.length 更新页脚，或改成「已加载 7 条 · 当前显示 4 条」，并在筛选/搜索时同步刷新。

### 2. [logic] 筛选/搜索零命中被并入 data-state=empty，CSS 因此把发布按钮一并隐藏；此时头部统计仍是 2/3，页面一边说「未找到匹配的记录」一边说今天记了 2 条，且屏幕上没有任何发布入口（空态 CTA 变成了「清除筛选」）。

- 证据：app.js:386-390 零命中时 `app.dataset.state='empty'`；styles.css:1274-1282 在 empty 下 `display:none` 掉 `.publish-btn`；app.js:371-373 把空态 CTA 改成「清除筛选」。DOM 实测中二班筛选后 `#publishBtn` computed display=none，`#statToday/#statPending` 仍为 2/3；源码 teacher/src/pages/growth/index.vue:132-134 的「快速记录」入口始终在 StateBlock 之外可见。
- 建议：把「服务端空」和「筛选/搜索无结果」拆成两种状态：后者不隐藏发布按钮，统计要么同步当前结果口径，要么明确标注不受筛选影响。

## P2 · 打磨项（5）

### 1. [completeness] 卡片日期口径与源码不一致且与月分组重复：源码今天显示「今天 11:20」、其余显示完整「2026-10-24 11:20」，原型一律只显示「10-24」，丢了时间与「今天」标记，又在「2026年10月」分组内重复月份。

- 证据：app.js:310 `record.date.slice(5)`；源码 teacher/src/pages/growth/index.vue:46-54 的 formatDate 有「今天 HH:mm」与 slice(0,16) 两个分支。DOM 实测卡片日期为 10-24/10-24/10-23/10-21/09-30…，月分组标题为「2026年10月」「2026年9月」。
- 建议：分组内显示「今天 11:20」或「24日 11:20」，其余显示「MM-DD HH:mm」；避免在月组内重复月份。

### 2. [logic] 新增的班级筛选在当前示例数据下无法验证：7 条记录全部 classKey='small1'，卡片也不显示班级，选「小一班」与「全部班级」结果逐条相同，只有选「中二班」才会得到空态，用户看不出这个 chip 到底在筛什么。

- 证据：app.js:27/43/59/75/91/107/123 全部 `classKey: 'small1'`，app.js:127-131 却提供「中二班」选项；DOM 实测「全部班级」与「小一班」的卡片 id 串完全相同（1024,1023,1022,1021,930,926,922）。相关契约缺口见 _shared/findings.md 接口缺口 #1/#3。
- 建议：示例数据混入中二班记录并在卡片上显示班级名/徽标；若产品确认不做服务端筛选，删掉班级 chip 只保留时间筛选，避免展示一个不可验证的控件。

### 3. [logic] 弹层打开时没有滚动锁定：app.js 给 .mp-app 加 sheet-open 类，但本页 styles.css 和 _shared/shell.css 都没有任何 .sheet-open 规则，dialog 打开后背景仍可滚动。

- 证据：app.js:466/474/491 添加与移除 `sheet-open`；grep 本页 styles.css 与 _shared/shell.css 无 `.sheet-open` 命中（同批次 teacher/12-calendar-create/styles.css:5、parent/07-leave-create/styles.css:28 用 `body.sheet-open{overflow:hidden}`）；运行时遍历全部样式表也没有含 sheet-open 的规则。
- 建议：在 styles.css 加 `body.sheet-open{overflow:hidden}`，并把类加到 document.body（或给滚动容器加 overflow:hidden），关闭时移除。

### 4. [design-core] 筛选/搜索空态的说明文字 .empty-hint 没有任何样式规则，继承正文 13px 与 --fg，比主文案（14px、--fg-2）更深更重，次要提示抢了主文案的层级；notes 的覆盖状态表也没提这行提示。

- 证据：index.html:105 与 app.js:367-370 写入提示，但 styles.css 中不存在 `.empty-hint` 选择器。运行时 computed：`#emptyHint` color rgb(30,41,59)/13px，`#emptyText` color rgb(51,65,85)/14px。
- 建议：补 `.empty-hint{max-width:250px;margin:var(--space-sm) 0 0;color:var(--muted);font-size:var(--text-sm);line-height:1.7}`，保持「主文案深、提示浅」的层级，并在 notes 里记录该分支。

### 5. [notes-accuracy] notes 的「外壳回归」声称自造类已全部替换、只换实现不改设计，但 styles.css 的 500px 媒体查询仍指向旧类 .app（HTML 中只有 .mp-app），这条桌面手机框规则永不命中，桌面预览下 880px 手机框与圆角不生效；notes 未记录该遗留。

- 证据：styles.css:1319-1322 `.app { height: min(880px, calc(100dvh - 48px)); border-radius: var(--radius-2xl); }`；index.html:13 只有 `class="mp-app"`；notes.md:65-76 的替换表声称 `.app → .mp-app`。运行时 `document.querySelector('.app')` 无匹配。
- 建议：把媒体查询选择器改为 .mp-app（如需要恢复桌面手机框），或删除死规则并在 notes 的遗留里记明。

## 2026-10-27 P1 修复记录

> 本轮只修 P1 两项，P2 五项一律未动。`od-check`（390px）`problems: []`；`od-audit` 本页无 ✗（仅保留原有的 styles.css/app.js 超 20KB 体积提示）；`preview.png` 已按 390×dsf2 重出。

### 1. [logic] 页脚「已显示最近 N 条」用全量 RECORDS.length，筛选/搜索后失真 — 已修

`renderNormal()` 改为按当前渲染行数写页脚：筛选/搜索生效时「已加载 7 条 · 当前显示 N 条」，未筛选时保留「已显示最近 7 条 · 下拉刷新」，且筛选、搜索、清除、演示芯片切换都会同步刷新。DOM 实测：选「本月（10月）」后 4 张卡片、页脚「已加载 7 条 · 当前显示 4 条」；搜索「王老师」后 5 张卡片、页脚「已加载 7 条 · 当前显示 5 条」。

### 2. [logic] 筛选/搜索零命中并入 empty、发布按钮被 CSS 隐藏、统计口径矛盾 — 已修

新增独立状态 `data-state="noresult"`：只隐藏 `.feed` 与 `.list-foot`，`.publish-btn` 与筛选条保留；`empty` 仅保留给服务端真空态（发布按钮仍收起，由「发布第一条」CTA 承担入口）。统计口径与源码一致（覆盖已加载的一页、不随筛选变化），并在筛选/搜索生效时把统计行小字改为「仅统计已加载 · 不受筛选影响」，与「未找到匹配的记录」不再互相打架。DOM 实测中二班零命中：`state=noresult`、发布按钮 `display:flex` 且可点、统计仍 2/3 但带新标注、空态按钮为「清除筛选」；输入无结果关键词同样 `noresult`、按钮为「清除搜索」；演示芯片「空态」仍按原设计隐藏发布按钮。无 console 报错、无横向溢出。
