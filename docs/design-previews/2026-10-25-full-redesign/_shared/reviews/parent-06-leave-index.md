# parent/06-leave-index · 对抗性单页审查

> 2026-10-27 已修复本页 P0（统计键 total→all），修复与验证见 `_shared/REVIEW.md` 2.1。

- verdict：**FAIL**（score 58/100）
- 源码覆盖：源码的统计卡、四个状态分段、六张卡片的姓名/天数/事由/起止日期/状态徽标、新建入口以及空/加载/失败态均已覆盖；未覆盖下拉刷新手势，卡片 4/6 的审批/撤销元信息缺失，新增审批意见依赖列表接口尚未返回的 review_remark。

## P0 · 用户可见硬伤（1）

### 1. [logic] 统计卡『全部请假』的数字从不更新：JS 查 [data-stat="all"]，HTML 写的是 data-stat="total"，空态因此显示 6/0/0、失败态显示 6/—/—，与页面状态及 notes 的 0/0/0、全『—』说法矛盾。

- 证据：index.html:53（data-stat="total"）；app.js:37 与 :49-52（values 用 all 键并 querySelector('[data-stat="all"]')，返回 null 被静默跳过）；DOM 实测空态 .stat-number 文本为 6/0/0，失败态为 6/—/—；notes.md:22-24 声称空态统计归 0/0/0、失败态统计显示「—」。
- 建议：把 index.html:53 的 data-stat 改为 all，或把 app.js 的键改为 total（同步 updateStats 三处赋值），复测空/失败态统计与列表状态一致。

## P1 · 明显缺陷（1）

### 1. [logic] 失败态点『重试』后页面永远停在 error、列表不恢复，却 toast『已重新加载』，是一个自相矛盾的假恢复。

- 证据：app.js:134-150 只重置按钮并 showToast('已重新加载')，未改 app.dataset.state、未重新渲染列表；DOM 实测点击后 1.4s：data-state=error、.error-state display:flex、.leave-list-area display:none、toast 文本「已重新加载」；notes.md:25 描述为「1s 后恢复」。
- 建议：重试后切回 normal 并恢复列表/统计；若只演示请求过程，至少把文案改为「已重新发起请求」并保留 error，不要让 toast 声称已恢复。

## P2 · 打磨项（6）

### 1. [completeness] 示例数据中卡片 4 的「李老师 9月29日」和卡片 6 的「家长 9月22日自行撤销」两段元信息在页面上丢失，导致这两张卡没有 meta 行（高度 101px，其他卡 144–181px），也违背 notes 的「已同意/已驳回显示审批意见块」。

- 证据：_prompt.md:35（卡片 4 · 李老师 9月29日）、:37（卡片 6 · 家长 9月22日自行撤销）；index.html:142-157 卡片 4 无 .approval-note，index.html:182-197 卡片 6 无任何 meta；DOM 实测卡片高度 1–6 为 144/162/181/101/181/101px；notes.md:13 声称已同意/已驳回显示审批意见块。
- 建议：卡片 4 补「审批意见 · 李老师 9月29日」（无 remark 时只显示 meta），卡片 6 补「家长 9月22日自行撤销」；或把两条明确改为无意见样例并同步 notes。

### 2. [notes-accuracy] notes 称落地审批意见块还要给列表补 reviewer.nickname 和 reviewed_at，但这两个字段列表接口已经返回，真正缺的只有 review_remark。

- 证据：notes.md:62；server/internal/service/leave_service.go:80-85（reviewed_at 与 reviewer.nickname 已在 leaveListItemFromRow 输出）；_prompt.md:19 也写明 created_at/reviewer.nickname/reviewed_at「列表接口有返回，源码未展示」。
- 建议：把 notes 遗留 2 改为「只需补 review_remark，reviewer/reviewed_at 已存在」，避免按错误前提改后端契约。

### 3. [logic] 筛选空态分支不可达：四个分段都有数据（6/1/2/2），任何点击都不会出现「该状态下暂无请假记录」，但 notes 把它列为已覆盖状态。

- 证据：app.js:75-76 仅在 visibleCount===0 时显示 filterEmpty；DOM 实测 all=6、pending=1、approved=2、rejected=2；notes.md:25 声称「筛选结果为空时显示筛选空态」，notes.md:27 又称它是页内分支。
- 建议：让演示数据留一个空筛选（例如把一条已拒绝改成待审批），或加独立演示开关；否则删掉 notes 的覆盖声明。

### 4. [navigation] 非 tabBar 子页唯一的退出入口——返回箭头没有 handler，点击无任何反应。

- 证据：index.html:38-40 为 <button class="mp-back">；app.js 全文无 .mp-back 绑定；DOM 点击后无 toast、无 URL 变化；对照 08-leave-detail/app.js:108-112 有「已返回请假记录」toast。
- 建议：在 app.js 给 .mp-back 加 toast（如「已返回我的」）或真跳父页；若刻意不做，至少加 aria-disabled 并说明。

### 5. [design-core] 11px 状态徽标的前景/底色对比度只有 2.74–3.08:1（待审批 2.74、已同意 3.00、已驳回 3.08），低于 brief 要求的 WCAG AA 4.5:1；待审批统计数字 20px 的 --accent/白底也只有 2.80:1，低于大字号 3:1。

- 证据：styles.css:269（徽标 font-size: var(--text-xs)=11px）、:275-288（warning/success/danger 三对 token）；tokens.css:28-33（#CA8A04/#FEF9C3、#16A34A/#DCFCE7、#EF4444/#FEE2E2）；实测对比度 2.74/3.00/3.08:1；styles.css:51 与 :88-90、tokens.css:11/20（#F97316 对 #FFFFFF=2.80:1）；brief.md:45 要求对比度满足 WCAG AA。
- 建议：回到 shared-design-tokens/tokens.json 调深 --warning/--success/--danger（或徽标改用更深的文字色），并复核 --accent 作 20px 数字的 3:1；不要在页面里逐页打补丁。

### 6. [redundancy] 『全部请假 6』与『共 6 条请假记录』在默认态重复同一个总数；筛选后统计仍为 6、脚注变成 1，而脚注没有「当前筛选」限定，两个口径并排容易被读成矛盾。

- 证据：index.html:53（统计 total 6）与 :205（footer count 6）；app.js:77 把 footerCount 改为当前可见数；notes.md:57 说明统计固定全量、列表前端筛选。
- 建议：默认态删掉脚注重复计数，或把脚注改为「当前筛选 1 条 / 全部 6 条」，确保统计与脚注只保留一个总数口径。

## 2026-10-27 P1 修复记录

- P1-1 [logic] 失败态『重试』假恢复 —— **已修**：`app.js` 在 1s 模拟请求结束后把 `.mp-app[data-state]` 切回 `normal`（新增 `setAppState`，同步演示芯片 `aria-pressed`），列表与统计随之恢复、筛选沿用重试前的选择；toast 仍为「已重新加载」，与 notes 的「1s 后恢复」一致。DOM 实测：切到失败态 → 点重试（按钮 disabled、文案「重试中…」、页面保持 error）→ 1.2s 后 `data-state=normal`、`.leave-list-area` 可见、统计 6/1/2、芯片「正常」`aria-pressed=true`、toast「已重新加载」；在「已拒绝」筛选下重试后仍停留该筛选（脚注 2）。`node scripts/od-check.mjs` → `problems: []`。
