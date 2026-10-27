# teacher/07-pickup-index · 对抗性单页审查

- verdict：**WARN**（score 66/100）
- 源码覆盖：源码覆盖了概览卡、四段筛选、入园/接走/再接走/通知四类动作、手动登记弹层（授权人/例外/理由）与整页空/加载/失败态；缺 overview.arrived（已入园/全班在读）统计、提交中与提交失败分支（源码 mapPickupError 的 4 类错误）、弹层内授权人 loading，并把「授权接送人」行换成了契约没有的接走时间/接走人。

## P1 · 明显缺陷（3）

### 1. [logic] 手动登记的「选择学生」弹层可以给未到园的孩子登记接走，绕过本页自己置灰的「未到园不能接走」规则

- 证据：app.js:115-118 对 not_arrived 的「接走」设 aria-disabled=true；但 app.js:238-254 的选人弹层列出全部 10 人（含未到园的 天天），选中后 app.js:362-371 一律 toast「天天 登记成功」。源码 teacher/src/pages/pickup/index.vue:182-185 的「手动登记」只是把筛选切到「全部」，notes.md:31-36 未申报这个改写（_shared/findings.md 接口缺口 #6 只记录了置灰本身，未涉及这条绕过路径）。
- 建议：选人弹层按状态限制：not_arrived 行标注「需先入园」并不可选；或恢复源码「手动登记=切到全部」的行为并在 notes 说明。确认提交补失败分支（源码 index.vue:131-143 mapPickupError）。

### 2. [completeness] 已接走学生行显示契约与源码都不存在的「接走时间 / 接走人」，同时删掉源码每行都有的「授权接送人」

- 证据：app.js:43-53 为 3 名已接走学生写入 pickedAt/pickedBy，app.js:140-141 渲染为「已接走 17:05 · 李梅（妈妈）」；backend/api/api-teacher/07-pickup.md:70-94 的 students 响应没有 picked_at/picked_by，源码 teacher/src/pages/pickup/index.vue:329-331 对所有行都显示「授权接送人：…」。
- 建议：回退为「授权接送人」行；若确需接走时间/接走人，标为需要 records 接口补字段或列表 join 的提案，并在 notes「与现状的差异」声明。

### 3. [logic] 入园 / 手动确认 / 扫码完成都只弹 toast，列表与概览数字不更新，出现「提示已入园/已接走但卡片状态不变」的自相矛盾

- 证据：app.js:221-224 点「入园」只 toast「天天 已入园」，卡片仍为未到园、接走仍 aria-disabled；app.js:362-371、412-415 手动与扫码完成同样只 toast，概览仍「待接走 6」。源码 teacher/src/pages/pickup/index.vue:106-108/118-128/161 每个动作后都调用 loadPickup()。
- 建议：动作后就地更新 STUDENTS 与概览并重渲染，或明确标注为静态演示、不更新数据。

## P2 · 打磨项（5）

### 1. [logic] 加载态仍在列表标题和滞留条上给出确定数字，只有四个统计数字被骨架化

- 证据：styles.css:493-507 的 loading 分支只隐藏 .stat-num 与 .student-list；.list-head（index.html:79-81「未接走学生（6人）」）与 .overdue-strip（index.html:64-67「2 名孩子已过放学时间 17:30 仍在园」）不受影响，加载中仍宣称 6 人和 2 名滞留。
- 建议：loading 态一并隐藏/骨架化列表计数与滞留条，避免与骨架语义冲突。

### 2. [completeness] 概览四项统计与源码不一致：源码首项「已入园」(overview.arrived) 被换成「未到园」，全页没有全班在读/已入园数字，notes 却声称概览包含这 7 项

- 证据：源码 teacher/src/pages/pickup/index.vue:227-231 首项为「已入园 = overview.arrived」，契约 backend/api/api-teacher/07-pickup.md:37-48 返回 total/arrived；原型 index.html:43-62 只有 未到园/待接走/已接走/异常，notes.md:10 写的是「全班在读、已入园、未到园、已接走、未接走、曾滞留、异常」。
- 建议：统计恢复「已入园」或补「全班在读」，并让 notes 与实际四项一一对应。

### 3. [notes-accuracy] notes 声称扫码只有「核验中 / 未通过」两态，实际还有可达的第三态「核验通过」；「通知只做 toast」也不准确

- 证据：notes.md:13/19；index.html:239-250 存在 scanSuccess（核验通过 · 乐乐 · 17:42），app.js:400-404 的「重新扫码」500ms 后即进入该态；notes.md:41 称通知「只做 toast 反馈」，app.js:225-230 实际还把按钮改成禁用态「已通知」。
- 建议：notes 补齐第三态与通知禁用态；若保留成功态，「完成」需同步更新列表与统计（与 P1 的登记不更新一并修）。

### 4. [redundancy] 同一 not_picked_up 状态一页三种叫法；滞留用四处信号重复表达；已接走行内重复一次

- 证据：index.html:51「待接走」、:73「未接走」、:80「未接走学生（6人）」，app.js:60/242-243 选人弹层又叫「已入园」；滞留同时有红边框 styles.css:211、「滞留」pill app.js:136、「放学 17:30…」提示 app.js:152-154、顶部滞留条 index.html:64-67；已接走 pill app.js:135 与正文 app.js:141 各说一次。
- 建议：统一 not_picked_up 口径（统计=筛选=标题）；滞留只保留卡片内一处最强信号；已接走行去掉重复标签。

### 5. [design-core] 筛选分段与登记方式分段的按钮高度 38px，低于批次 44px 触控硬约束

- 证据：styles.css:151 `.filter-btn { height: 38px }`，styles.css:723 `.mode-btn { height: 38px }`；_shared/brief.md:45 要求「触控目标 ≥ 44×44px」（页面其余动作 .a-btn 44px、.bar-action 64px 均达标）。
- 建议：两处分段按钮改 44px，或外扩 44px 透明命中区。

## 2026-10-27 P1 修复记录

- P1-1 已修：选人弹层把未到园的学生渲染成 `.picker-row--off`（标注「需先入园」、`aria-disabled` 且点击只提示不打开登记），与列表「接走」置灰同一条规则；确认提交补了契约里的失败分支——已接走的学生再次登记而未填理由时，保留弹层并 toast「重复接走需填写理由」。
- P1-2 已修：删除 `pickedAt` / `pickedBy`，已接走行与其余行统一展示「授权接送人：…」，回到 `students` 契约（无接走时间/接走人）与源码 `index.vue:329-331` 的写法。
- P1-3 已修：新增 `renderSummary()` / `refreshView()`，入园、手动确认、扫码完成都就地改 `STUDENTS` 并重画列表、筛选计数、概览四项与滞留条；切回「正常」chip 时从初始快照重置，演示可重复。
- P2-1 ~ P2-5 均未处理（按本轮范围只修 P1）。
