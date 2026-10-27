# teacher/15-class-index · 对抗性单页审查

- verdict：**WARN**（score 70/100）
- 源码覆盖：信息卡（班级/学年/园所/班主任/人数）、今日快照、搜索+四段筛选、名单及加载/空/错误分支都已覆盖；缺「班级动态」快捷入口（工作台有近似替代）与「最新通知+已读率」整块（全端无替代），BottomNav 按批次约定未画。

## P1 · 明显缺陷（5）

### 1. [navigation] 导航栏返回箭头是死控件，点击完全无反应

- 证据：teacher/15-class-index/index.html:22 `<button class="mp-back" aria-label="返回老师工作台">`；app.js 全文无 .mp-back 查询（grep 无命中），_shared/shell.js:9-27 只绑定演示条；同批 teacher/14-leave-index/app.js:484-486 等页都绑了 toast，说明是本页漏绑
- 建议：在 app.js 绑定 .mp-back → toast「返回老师工作台」或回退演示；本页也可从「我的」进入（teacher/src/pages/profile/index.vue:47），aria-label 宜改为「返回」

### 2. [logic] 「接送截止」被画成按学生取值，同一班出现 16:30/17:00 两种；后端该字段是班级级单一值，等于凭空造出按人截止能力

- 证据：index.html:95/103/114/120/123 五人 data-deadline="17:00"，其余 25 人 16:30；server/internal/service/teacher_portal_service.go:1991 只取一次 classPickupDeadline，:2116 对每个学生写同一个 ResolvePickupDeadline(deadline)，ClassStudents 直接复用该接口（:2806-2808）
- 建议：全班统一一个 deadline（示例 16:30），或删掉信息卡「接送截止」行；若产品确需按人定制，先改 backend/api + service

### 3. [design-core] 筛选胶囊 36px、文本按钮 40px，低于 brief 规定的 44×44 触控下限

- 证据：teacher/15-class-index/styles.css:286 `.filter { min-height: 36px }`；styles.css:401 `.link-btn { min-height: 40px }`；_shared/brief.md:45「触控目标 ≥ 44×44px」
- 建议：两处改为 min-height: var(--input-height)（44px）

### 4. [design-core] 同一「已接走」状态在概览是蓝色(info)、在幼儿卡和信息卡是绿色(success)，语义色一页两套

- 证据：index.html:72 概览「已接走」用 dot--info（styles.css:208 → --info）；styles.css:379 卡片 picked_up 点用 --success；styles.css:720-722 信息卡 picked_up 用 --success；preview.png 实测概览为蓝点、卡片为绿点
- 建议：概览/卡片/信息卡统一同一 token（建议已接走=--info、已入园=--success），并同步 notes 的语义色映射

### 5. [completeness] 源码末尾「班级动态」+「最新通知（含已读率）」整块未画，notes 称归消息页，但消息页原型没有任何已读率展示，该数据视图在整套原型中失去落点

- 证据：teacher/src/pages/class/index.vue:148-164（三快捷入口 + notice-card `已读 {{ notice.readRate }}%` 进度条）；notes.md:37 称「分别是工作台与消息页职责」；teacher/18-messages-index/index.html:107 只有「全部已读」，grep teacher/*/index.html 无任何 read_rate/已读率展示；工作台只有未读计数与一条待办（01-workbench/index.html:72,191）
- 建议：在名单上方补「最新通知」两条 + 已读率条，或把已读率明确落到通知详情/消息页并在 notes 标注需产品确认

## P2 · 打磨项（3）

### 1. [redundancy] 幼儿信息卡里「接送状态」行与标题旁状态 pill 完全重复，4 行卡片实际只有 3 项有效信息

- 证据：index.html:173 `<span class="sheet-pill" id="sheetPill">…已接走`；index.html:178 `<dt>接送状态</dt><dd id="sheetStatus">已接走</dd>`；app.js:103,105 用同一个 statusText 同时写入两处
- 建议：删掉「接送状态」行，或换成后端已返回但未用的 last_remark / is_overdue（notes 遗留 3 提到逾期）

### 2. [logic] 加载态与客户端空结果可同时出现：先筛到 0 条再切「加载」，6 张骨架卡与「没有找到符合条件的孩子」并存

- 证据：styles.css:384-393（.no-results 由 data-visible=true 显示）；加载态规则 styles.css:496-520 只隐藏 .grid/.filter-count，未隐藏 .no-results；对照空态 styles.css:490 明确隐藏了 .no-results；app.js:163-176 setState 不清 noResults
- 建议：补 `.mp-app[data-state="loading"] .no-results { display: none }`，或在 setState 时重置 noResults 的 data-visible

### 3. [notes-accuracy] notes 声称字号全部走 --text-*，实际 .kid-time 硬编码 10px，且低于 token 最小档与正文 13px 下限

- 证据：styles.css:375 `.kid-time { font-size: 10px }`；tokens.css:39 `--text-xs: 11px`（最小档）；notes.md:32「字号/字重全部走 --text-* / --weight-*」；_shared/brief.md:45「正文 ≥ 13px」
- 建议：改为 var(--text-xs)，并修正 notes 中「全部走 token」的断言

## 2026-10-27 P1 修复记录

- P1-1 [navigation] 返回箭头死控件 —— **已修**：`app.js` 绑定 `.mp-back` → toast「返回上一页」；`aria-label` 由「返回老师工作台」改为「返回」（本页也可从「我的」进入，不写死目的地）。
- P1-2 [logic] 「接送截止」按学生取值 —— **已修**：删掉 30 张卡上的 `data-deadline`（原 5 人 17:00、25 人 16:30），改为 `.mp-app[data-pickup-deadline="16:30"]` 的班级级单一值，信息卡统一读这一个值。
- P1-3 [design-core] 触控目标低于 44px —— **已修**：`.filter`（原 36px）与 `.link-btn`（原 40px）的 `min-height` 改为 `var(--input-height)`（44px）。
- P1-4 [design-core] 「已接走」一页两套语义色 —— **已修**：幼儿卡圆点与信息卡 pill 的 `picked_up` 从 `--success` 改为 `--info` / `--info-wash`，与概览蓝点统一；已入园仍为 `--success`。
- P1-5 [completeness] 「最新通知 + 已读率」整块缺失 —— **已修**：在概览与名单之间补「最新通知」2 条（标题 / 日期 / 已读率进度条），对应源码 `fetchTeacherNotices({ page:1, page_size:2 })` 的 `read_rate`；加载态换 2 行骨架、失败态隐藏、空态保留（与名单数据独立），点击出 toast。

### 随 P1 的说明

- 通知示例与 `teacher/18-messages-index` 同源（9/28≈32%、18/28≈64%）；消息页 `recipient_count=28` 与班级 30 人的差异属批次级数据口径，已记入 notes 遗留 7，本页不单方面改数。
- notes.md 已同步：信息层级增「最新通知」、覆盖状态补通知区三态、token 映射补通知卡、差异段改写、验证段补 P1 复测结果。

P2 三项（信息卡「接送状态」行重复、加载态与空结果并存、`.kid-time` 10px）按要求未动。
