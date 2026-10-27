# parent/01-home · 对抗性单页审查

- verdict：**WARN**（score 68/100）
- 源码覆盖：源码覆盖基本完整：hero、今日四态、多孩切换、4 个入口、今晚一件事、动态流及正常/空/加载/失败四态均有实现；仅按 brief 明确省略 needBindPhone 整页绑定分支（parent/src/pages/index/index.vue:250-269）、下拉刷新与 scene 深链转场，未凭空新增业务模块。

## P1 · 明显缺陷（2）

### 1. [logic] 多孩切换只重绘孩子卡与今日四列，没有刷新「今晚一件事」和「亲子任务」角标；切到小雨后仍显示乐乐的「亲子阅读」任务与角标 3，两个绑定当前孩子的数据块与选中孩子不一致。

- 证据：docs/design-previews/2026-10-25-full-redesign/parent/01-home/app.js:76-104 的 applyProfile/切换回调只更新 fieldMap 与奖章；index.html:153-166 的任务卡和 :139 的角标 3 为静态内容；源码切换后 await loadHome()（parent/src/pages/index/index.vue:141），任务按 student_id 拉取（index.vue:87-90），后端 pendingTaskCount 也按 CurrentStudent 计算（server/internal/service/parent_portal_service.go:2534）。
- 建议：为小雨准备独立的任务/角标数据并在切换时一并重绘；若保留简化，至少在 notes 与页面上标注该区不随孩子切换。

### 2. [logic] 空态把「今晚一件事」与动态列表一起隐藏，且通知角标仍为 1、任务角标仍为 3，与空态文案「这周还没有新的通知和成长记录」自相矛盾；空态判定把两套互不相关的数据源耦合在一起。

- 证据：parent/01-home/styles.css:756-758 同时隐藏 .tonight-section 与 .feed-list；index.html:123/139 的角标 1/3 在空态仍可见，:200 的空态文案称本周无通知；源码中今日任务只依赖 todayTask（parent/src/pages/index/index.vue:307），动态空态只作用于 feed（index.vue:325-327）。
- 建议：空态只替换动态区，按 todayTask 保留今晚一件事；角标应随空态数据置 0 或隐藏，避免与空态文案冲突。

## P2 · 打磨项（5）

### 1. [logic] 错误态点「重试」约 1 秒后按钮恢复并 toast「已重新加载」，但页面仍停留在错误卡，用户被告知成功却看不到任何恢复。

- 证据：parent/01-home/app.js:118-130 的超时回调只恢复按钮并 showToast('已重新加载')，不改 data-state；styles.css:767-770 错误态仍隐藏今日卡与动态区。
- 建议：重试完成后切回 normal 态；或模拟失败时 toast「仍无法连接，请稍后再试」，不要给出假成功反馈。

### 2. [completeness] 动态预览没有做两行截断：源码用 -webkit-line-clamp:2，brief 也明确要求「两行截断」，当前长预览会把卡片无限撑高。

- 证据：parent/01-home/styles.css:642-648 的 .feed-preview 只有 overflow-wrap，无 line-clamp/overflow；对应源码 parent/src/pages/index/index.vue:751-759 有 -webkit-line-clamp:2；_prompt.md:64 明确要求两行截断。
- 建议：给 .feed-preview 补 display:-webkit-box; -webkit-box-orient:vertical; -webkit-line-clamp:2; overflow:hidden。

### 3. [logic] 示例时间与状态栏 9:41 不自洽：「5小时前」是 04:41，不可能有「今天的美术课」；10-25「午睡后」的记录按源码 formatTime 应显示约 18 小时前，而不是日期。

- 证据：parent/01-home/index.html:14（9:41）、:186（5小时前）、:192（2026-10-25）、:191（午睡后…给多肉浇水）；parent/src/pages/index/index.vue:161-163 的 formatTime 在 24 小时内返回「N小时前」；notes.md:48 还声称与 formatTime 一致。
- 建议：把「5小时前」改到工作时段内（如「1小时前」）或把第三条日期改为 10-24，使显示符合 24 小时内显示小时、超过才显示日期的规则。

### 4. [notes-accuracy] notes 把首页数据拉取写在 onLoad，并漏掉任务请求；实际是 onShow 触发 loadHome，onLoad 只处理 scene 深链，且 loadHome 同时拉 overview/feeds/tasks 三路。

- 证据：parent/01-home/notes.md:3 写「onLoad 并行拉 HomeOverview + ListHomeFeeds」；parent/src/pages/index/index.vue:172 为 onShow(loadHome)，:177-180 的 onLoad 仅解析 scene，:87-90 三路请求含 fetchTasks。
- 建议：把 notes 改为「onShow 触发 loadHome，并行拉 overview/feeds/tasks（任务按当前孩子 student_id 维度）」。

### 5. [design-core] toast 固定在视口 46% 的屏幕中部，会压住今日卡/快捷入口内容；brief 要求轻提示固定在 tabBar 上方。

- 证据：parent/01-home/styles.css:773-777 的 .toast 为 position:fixed; top:46%; left:50%；_prompt.md:88 要求「固定在 tabBar 上方」。
- 建议：改为 bottom: calc(62px + env(safe-area-inset-bottom)) 并与演示条错开，避免覆盖主数据卡。

## 2026-10-27 P1 修复记录

- **P1-1 已修**：多孩切换现在同步重绘「今晚一件事」与亲子任务角标——小雨改为独立任务「亲子运动 10 分钟」+ 角标 1（含 sr 文案），乐乐保持「亲子阅读 15 分钟」+ 角标 3，切换后不再残留上一个孩子的任务数据（app.js profiles / applyProfile）。
- **P1-2 已修**：空态只替换动态区，「今晚一件事」按今日任务保留；空态下通知角标与任务角标（含 sr 文案）一并隐藏，消除与「这周还没有新的通知和成长记录」的矛盾（styles.css 四态规则 + app.js applyBadgesForState）。

