# parent/12-growth-report · 对抗性单页审查

- verdict：**WARN**（score 75/100）
- 源码覆盖：对照 report.vue：周期切换（默认周报）、学生姓名/日期、四格统计、能力维度、亮点、关注、评语、加载分支均已覆盖，并补了骨架屏、空态、失败重试与禁用态；缺 avatar 图片态、四个区块各自的 v-if 空分支、每维度 max_score 归一化，以及出勤率缺省「-」的展示。

## P1 · 明显缺陷（2）

### 1. [logic] 空态与失败态的出口把周期写死成「月报/周报」，不读当前选中的周期：在「月报」下进入空态后，页面仍说「这个周期还没有成长报告……先看看月报或学期报告」并给「查看月报」按钮，点击后刚被判为「没有报告」的月报立刻出现数据，toast 还谎称「已切换到月报」；失败态「重试」同样固定跳回周报。

- 证据：app.js:73 `returnToNormal('monthly', '已切换到月报')`；app.js:83-88 的 mp:state 处理在非 normal 时不重置周期，所以先点 `[data-od-id="report-tab-monthly"]` 再点演示芯片「空态」，分段仍高亮「月报」，而 index.html 的 `.state-empty .state-copy` 仍是「老师通常周五整理周报，先看看月报或学期报告」、`[data-od-id="report-empty-cta"]` 仍是「查看月报」；app.js:79 的 `returnToNormal('weekly', ...)` 让从月报失败态点「重试」跳到周报。
- 建议：状态出口读取当前选中周期：空态 CTA 指向一个与当前周期不同的周期（周报→月报、月报→学期、学期→返回成长时光），文案随周期生成；「重试」沿用失败前的周期，toast 只在周期真的变化时出现。

### 2. [logic] 三个周期的演示数字只满足单调递增，日速率互相矛盾：学期（9/1–10/25）在扣除月报（9 月）后，10 月 1–25 日的积分/记录/任务日速是月报与周报的 1.7–2.9 倍，等于说 10 月的产出比 9 月高一倍以上，而周报（10/19–10/25）的日速又与 9 月持平。

- 证据：index.html 三份 `.stats`：周报 45 分/15 条/8 任务（10/19–10/25，7 天）、月报 180 分/62 条/31 任务（9/1–9/30，30 天）、学期 610 分/186 条/96 任务（9/1–10/25，55 天）。学期=9 月+10 月 1–25 日，倒推 10 月 25 天需新增 430 分/124 条/65 任务（日速 17.2/4.96/2.6），而月报日速 6.0/2.07/1.03、周报日速 6.4/2.14/1.14；出勤率 91%→94% 要求 10 月约 98%，当周却只有 80%。notes.md:60 却声明「演示数字自洽」。
- 建议：按日速率反推学期值（约 340 分 / 115 条记录 / 58 个任务），或把月报改成 10/1–10/25 再重算四格，使学期 ≈ 月报加 10 月增量与周报同速。

## P2 · 打磨项（5）

### 1. [completeness] 源码四个内容区块都带 v-if，报告缺字段时整块不渲染；原型在正常态无条件渲染四个区块，report 没有 highlights / concerns / teacher_comment 时会留下空标题和空引文框，也没有「部分缺失」的状态。

- 证据：report.vue:86 `v-if="report.dimensions?.length"`、101 `v-if="report.highlights?.length"`、111 `v-if="report.concerns?.length"`、121 `v-if="report.teacher_comment"`；index.html 三个 `.report` 中的 `[data-od-id="report-highlights"]` / `report-concerns` / `report-comment` 都静态存在，内容写死。
- 建议：为每个区块补空分支（无数据整块隐藏，或显示一行「本期暂无」），或在 notes 的覆盖状态里明确声明只画整页空态、区块级缺失未画。

### 2. [completeness] 源码按每个维度的 d.max_score 归一化柱长并只显示原始分，原型把分值直接当百分比（--score → height%）并静态写「满分 100」；max_score 不是 100 时柱高会画错，而这个契约字段在 notes 的遗留清单里没有出现。

- 证据：report.vue:93 `width: (d.score / d.max_score * 100) + '%'`；styles.css:411 `height: calc(var(--score) * 1%)`；index.html 三处 `.sec-meta` 固定「满分 100」，`.dim-bar` 的 `--score` 直接取 68–96。
- 建议：演示数据显式带上 max_score，`--score` 存归一化百分比；「满分 100」改为按 max_score 渲染，并在 notes 遗留补一条维度量纲说明。

### 3. [redundancy] 「周报/月报/学期」在同一屏出现两次：分段控件的选中项与纸面封面徽标，两者随切换同步变化，徽标不提供任何新信息。

- 证据：index.html `button[data-od-id="report-tab-weekly"]` 文案「周报」且 aria-pressed=true；紧邻其下的 `section[data-period="weekly"] .period-tag` 同样显示「周报」，月报/学期同构，三段重复。
- 建议：徽标改为承载增量信息（如「第 8 周」「10月第 4 周」或班级名），或删掉徽标只留日期范围。

### 4. [navigation] 返回按钮的隐含目的地被写死为「成长时光」，但源码里成长报告也能从「我的」进入，真实返回是 navigateBack 回到上一页；从「我的」进入时这个提示是错的。

- 证据：app.js:68 `showToast('返回成长时光')`；`parent/src/pages/profile/index.vue:14` 菜单项 `{ title: "成长报告", path: "/pages/growth/report" }`，`parent/src/pages/growth/index.vue:30` 另有一处 `uni.navigateTo({ url: "/pages/growth/report" })`。
- 建议：提示改为中性的「返回上一页」（或按来源区分），不要固定成某一个入口页。

### 5. [completeness] 源码里学生头像优先渲染 report.student.avatar 真实图片，无图才用占位；原型只画了姓名首字「乐」的字母头像，avatar 图片态在三个周期里都没有任何表现。

- 证据：report.vue:56-57 `<image v-if="report.student?.avatar" :src="report.student.avatar">` + `header-avatar-placeholder` 兜底；index.html `.head-avatar` 固定文本「乐」，weekly/monthly/term 三份封面完全相同。
- 建议：封面头像支持图片地址，URL 为空时回退首字；notes 的信息层级里注明当前是占位表现。

## 2026-10-27 P1 修复记录

- **P1-1 空态/失败态出口写死周期 — 已修**：app.js 记录当前周期；空态说明与 CTA 随周期生成（周报「查看月报」→ 月报「查看学期报告」→ 学期「返回成长时光」，学期出口只 toast），「重试」回到失败前选中的周期，toast 不再谎称切换。实测：月报空态 CTA 切到学期并提示「已切换到学期报告」；学期失败后重试仍停在学期。
- **P1-2 三个周期演示数字日速矛盾 — 已修**：学期四格改为 92% / 340 分 / 58/80 / 115 条（340 = 月报 180 + 10 月 1–25 日按周报同速的 160；记录 115 = 62 + 53；任务 58 = 31 + 27），三个周期日均速率一致；出勤改按校历工作日自洽（周报 4/5、月报 20/22、学期 36/39）。notes.md 同步更新。
- **P2 未处理**：按任务约束，P2 五项保持原样。
