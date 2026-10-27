# teacher/19-profile-index · 对抗性单页审查

> 2026-10-27 token 对比度已统一修复（accent-deep / danger 加深），见 `_shared/REVIEW.md` §3.2。

- verdict：**WARN**（score 65/100）
- 源码覆盖：源码的工牌信息（头像/姓名/园所/班级与职务标签）、三项统计、三个设置入口、退出登录和空/加载/错误三态都已覆盖，缺「有角色但 class_name 为空」的标签组合及 nickname/avatar 缺省分支；自绘底栏按批次规则未画。

## P1 · 明显缺陷（2）

### 1. [logic] 空态唯一主行动「刷新资料」是死动作：点击只弹 toast「资料已刷新」，页面仍停在 empty，园所仍为「未绑定园所」、三个数字仍为「—」，构成虚假成功反馈。

- 证据：index.html:93 的 #refresh-profile 只在 app.js:104-108 里 toast("资料已刷新")；实测点击后 .mp-app[data-state] 仍为 empty，.stat-empty 仍 display:flex，#refresh-profile 仍在，notes.md:59 亦承认真实行为待定。
- 建议：让刷新模拟一次成功取数（切回 normal 并更新园所/统计），或改成诚实的「请联系园所管理员绑定」提示/禁用态；不要用「已刷新」掩盖未变化。

### 2. [design-core] 强调色/危险色文字在浅色底上的对比度系统性不足：待审批「3」2.50:1、空态「刷新资料」与失败态「重试」2.50:1、退出登录/确认按钮 3.62:1、「开发中」徽标 4.01:1；20px/600 的「3」连大字号 3:1 都达不到，12–14px 文本低于 AA 4.5:1。

- 证据：styles.css:181-182（pill 底/字）、266-268（btn-soft）、362-369（badge）、387（logout）、490（dialog danger）；tokens.css:32（--danger #EF4444）、81-91（teacher --accent-deep #e6762a、--surface #fffaf0）；实测 2.50 / 2.50 / 3.62 / 4.01；brief.md:45 要求对比度满足 WCAG AA。
- 建议：在 token/派生层统一修：tint 上的文字改用 --fg 或把 accent-deep 加深到 ≥4.5:1；danger 文本用更深的 danger 文本 token（或 fg 文字 + danger 图标）；badge 用 --fg-2；用对比度工具复测，不要逐页打补丁。

## P2 · 打磨项（6）

### 1. [design-core] 三个设置 cell 的 :active 按下态不可见：规则把背景设为 --surface-warm，但老师端 --surface-warm 与 --surface 同为 #fffaf0，且 -webkit-tap-highlight-color: transparent 关掉系统高亮、cell 也没有 transform，按下时与静止态完全相同。

- 证据：styles.css:331 .cell:active{background:var(--surface-warm)}；tokens.css:12 与 84 均为 #fffaf0；styles.css:35 关闭 tap highlight；实测两者相等、cell 背景 transparent、无 transform；styles.css:395 的 .logout:active 背景同样无效（只剩 scale(.99)）。
- 建议：用可辨识的按下底色（如 var(--bg) 或 accent 与 surface 的 color-mix）并/或加 transform: scale(.99)；同时修正 .logout:active；在 390 宽下用真实交互复测。

### 2. [redundancy] 空态把「未绑定园所」说了三遍并保留无对象的「今日班级」标题：园所行、三列「—」、说明条「还没有绑定园所…」同时出现，用户在没有班级时仍看到「今日班级」。

- 证据：index.html:50（未绑定园所）、62（今日班级）、71/75/79（三个 —）、84-94（说明 + 刷新资料）；实测 empty 态 .stat-card 仍 display:grid、.stat-empty 为 flex、.today-title 可见。
- 建议：空态只保留一处说明：用一张空态卡替代统计卡（或隐藏整个 today 区），不要同时保留标题 + 三列「—」+ 说明条；园所行可保留「未绑定园所」，但统计区标题应隐藏或改成「暂无班级」。

### 3. [navigation] 确认退出登录后没有去向：源码 clearAuth() 后 reLaunch 到工作台，原型只关闭弹窗并 toast「已退出登录」，页面仍停在「我的」，用户看到已退出却还在个人页。

- 证据：teacher/src/pages/profile/index.vue:89-92（clearAuth + uni.reLaunch('/pages/index/index')）；app.js:70-75（closeDialog(false) + toast）；实测确认后 data-state 仍为 normal、#logout 仍可见，未表达任何跳转意图。
- 建议：确认后切到 teacher/01-workbench（或至少 toast「已退出登录，返回工作台」并复位页面），与源码的 reLaunch 意图一致。

### 4. [logic] 确认退出时焦点留在已隐藏的对话框按钮上，aria-hidden 被 Chrome 拒绝并输出 warning：closeDialog(false) 先设 aria-hidden=true，焦点仍停在 #logout-confirm。

- 证据：app.js:42-50（先 setAttribute('aria-hidden','true')，仅 restoreFocus 时才还焦点）、70-75（确认路径传 false）；实测确认后 document.activeElement.id === 'logout-confirm' 且 #logout-overlay[aria-hidden='true']；console msgid=2：「Blocked aria-hidden on an element because its descendant retained focus…」。
- 建议：先把焦点移到安全元素（触发按钮/body/新页面）再设 aria-hidden，或用 inert；确认路径应把焦点交给目标页/触发器；补焦点陷阱（Tab 不逃出弹窗）。

### 5. [completeness] 把「未绑定园所」等同于「无角色」：源码只替换园所文案，标签仍按 hasRole/className/roleLabel 渲染；原型空态一刀切隐藏标签，且没有「有角色但 class_name 为空（园长/无班级）」的标签组合。

- 证据：teacher/src/pages/profile/index.vue:20-37（schoolName 仅回退文案；className/roleLabel 独立）、121-129（hasRole 下分别 v-if className / roleLabel）；index.html:52-55（normal 永远两个标签）、styles.css:403（empty 隐藏整行）、notes.md:20 宣称空态隐藏班级/职务标签。
- 建议：标签按角色数据条件渲染，空态只影响园所文案与统计；在 notes 的覆盖状态中补「仅职务」组合（园长/无班级），或提供对应演示态。

### 6. [notes-accuracy] notes 的「本页内容高度 885px」与实测不符：390 宽下内容底部为退出登录 604.3px + .page 底部 96px = 700.3px；885 是 885 视口下的 100dvh/app 最小高度，不是内容高度，同一 bullet 的 604px 与 885px 也自相矛盾。

- 证据：notes.md:49；实测 .logout bottom 604.28（normal）、.page padding-bottom 96、内容 700.28；empty 态 logout bottom 649.6 + 96 = 745.6（390×700 会滚动）；document.documentElement.scrollHeight 为 844（min-height 844）。
- 建议：把 885 改为实测内容高度约 700px；如需写容器高度，注明 885 是 100dvh/app 高度；补测空态在 390×700 下会滚动（内容约 745.6px）。

## 2026-10-27 P1 修复记录

- **P1-1 [logic] 空态「刷新资料」是死动作 → 已修**。`app.js` 抽出 `setState(state)`（切 `[data-state]` 的同时同步演示 chip 的 `aria-pressed`，失败态「重试」也改用它），「刷新资料」改为**重跑一次取数**（等于源码 `loadProfile()` 的 getUser + overview）：点击后按钮 `disabled` + `aria-busy` + 文案「刷新中…」→ 页面进 `loading`（三列骨架 + 区头「同步中…」）→ 900ms 后回落 `normal`，园所回「阳光幼儿园」、三列回 26 / 2 / 3、班级与职务标签回来，toast「资料已更新」。另加 `mp:state` 监听：刷新未到点时手动切 chip 会 `clearTimeout` 并复原按钮，定时器不再覆盖用户刚选的状态。原「点一下弹『资料已刷新』、页面仍停在 empty、园所仍『未绑定园所』、三个数字仍是『—』」的虚假成功反馈已消除。
- **P1-2 [design-core] 强调色/危险色文字对比度系统性不足 → 暂缓（批次级）**。实测复核与审查一致：待审批「3」20px/600 = 2.50:1、空态「刷新资料」与失败态「重试」12px/600 = 2.50:1、「开发中」徽标 12px/500 = 4.01:1、退出登录 14px/500 与确认框危险按钮 14px/600 = 3.62:1。成因：2.50 是 `--accent-deep`(#e6762a) 压本页派生的 accent 18% 浅底；3.62 是 `--danger`(#EF4444) 压 `--surface`(#fffaf0)；4.01 是 `--muted` 压 neutral 浅底。**本页目录内无合规解法**：(a) `tokens.css` 由 `scripts/sync-open-design-adapter.mjs` 从 `shared-design-tokens/tokens.json` 生成，文件头禁止手改；(b) 两处危险红只有 `--danger` 一个 token，没有深色文本变体，页内自造深红等于新增裸色值（`od-audit` 会按裸色值判失败）；(c) 其余三处若改用 `--fg` / `--fg-2` 字色，本页「待审批」胶囊、软按钮、「开发中」徽标会与同批次 teacher/01-workbench、07-pickup-index、20-doctor-checkin 的同款组件不一致，正是审查与 `_shared/REVIEW.md` §6.4 明确禁止的「逐页打补丁」。已在 notes.md「遗留」登记：需回 `tokens.json` 一次性加深 `accent-deep`、补 danger 文本 token，重跑三端产物后统一用对比度工具复测。

P2 六项（cell 按下态不可见、空态三重复述、退出后无去向、确认后 aria-hidden 焦点、未绑定园所≠无角色、notes 885px 表述）按要求未动。

**验证**：`od-check --width 390` → `problems: []`、`overflowing: []`、`scrollWidth = clientWidth = 390`、无 console error；CDP 实点击复测：空态点「刷新资料」→ `loading`（chip 同步为「加载」、骨架 `block`、区头「同步中…」`inline`）→ 1.15s 后 `normal`（园所「阳光幼儿园」、26 / 2 / 3、两个标签回来、toast「资料已更新」、chip 同步为「正常」）；刷新途中切「失败」chip → 停在 error、按钮恢复可点（定时器已作废）；失败态「重试」仍回正常态并同步 chip。`od-shot` 已重出 `preview.png`（780×2000 = 390 宽 @2x）；批次 `od-audit` 50/50 通过，本页无 ✗、无体积警告（app.js 4.7KB）。

