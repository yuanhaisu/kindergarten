# parent/27-profile · 对抗性单页审查

- verdict：**PASS**（score 85/100）
- 源码覆盖：源码的 12 项菜单（顺序/标题/去向）、身份卡姓名与手机号、tabBar 第 4 项都已覆盖；未覆盖的是昵称/手机号字段缺失时的「家长 / 微信用户 / 家」字段级兜底——现有空态只处理整卡缺失，且属设计提案。

## P2 · 打磨项（6）

### 1. [design-core] 主色文字对比度不达 WCAG AA（brief 第 45 行硬要求）：空态主按钮白字压渐变仅 2.26–2.80:1，失败态「重试」3.35:1，页尾「退出登录」3.08:1，确认框「退出登录」3.76:1，身份卡「家长」胶囊 3.19:1（即使按 prompt 用 surface 底也只有 3.56:1）。

- 证据：styles.css:334-338（.btn--primary 用 --accent-gradient + --on-brand）、styles.css:340-344（.btn--ghost）、styles.css:628-631（.logout-button 用 --danger/--danger-wash）、styles.css:795-797（.dialog-confirm 用 --danger/--surface）、styles.css:194-208（.identity-role 用 --accent-deep 压 11% --accent 混色）；tokens.css:18-24,32-33；_shared/brief.md:45。
- 建议：回 shared-design-tokens/tokens.json 统一提高主色文案对比度（实色深底或深色字），目标 ≥4.5:1；属批次级 token 修正，不要只改本页。

### 2. [completeness] 源码的字段级空值兜底（昵称缺省「家长」、手机号缺省「微信用户」、头像首字「家」）在原型四态里都没有对应；正常卡写死乐乐妈妈/138****6688，空态只覆盖整卡缺失，notes 还称该页状态「源码无」。

- 证据：parent/src/pages/profile/index.vue:45-48；index.html:121-134（正常卡写死）、index.html:137-145（整卡缺失空态）；notes.md:29,54。
- 建议：补「已登录但资料缺失」的兜底表现（可直接沿用源码三处文案），或至少在 notes 里明确这是待产品确认的取舍；不要把整卡空态当作字段兜底。

### 3. [logic] 重试的 1s 计时器不会被「失败」chip 取消：重试中手动点「失败」，计时器到点仍会 applyState('normal')，页面和 chip 一起跳回正常，用户选的错误态被覆盖。

- 证据：app.js:75-77（next === 'error' 时跳过 resetRetry）、app.js:202-207（setTimeout 无条件回 normal）、app.js:266-269（mp:state → applyState）。
- 建议：在 mp:state 入口无条件 resetRetry()（或 applyState 开头统一清理），让任何 chip 选择都取消待执行的重试；仅当计时器仍有效时才回正常态。

### 4. [notes-accuracy] notes 把生命周期写成 onShow、并称本页无网络请求；源码实际是 onLoad + await ensureLogin()，无 token 时 ensureLogin 会走 silentLogin → POST /auth/login/wechat，且 notes line 4 把 GET /common/user/profile 写成登录态来源，源码并未调用。

- 证据：notes.md:3-4；parent/src/pages/profile/index.vue:25-28；parent/src/utils/login.js:16-29,35-45（ensureLogin 无 token 时 silentLogin）。
- 建议：notes 改为 onLoad，并写明「本地缓存优先，无 token 时 ensureLogin 会发起微信登录请求」；GET /common/user/profile 标注为后续提案，不是现状。

### 5. [redundancy] 身份卡「家长」胶囊是源码没有的新增信息：家长端所有账号都是家长，胶囊与昵称/端语境重复，不承载信息；notes 的「与现状的差异」只列了幼儿园名、图标、分组等，未把这枚胶囊标为设计提案。

- 证据：index.html:121-126（乐 / 乐乐妈妈 / 家长）；parent/src/pages/profile/index.vue:44-48（源码只有头像、昵称、手机号）；notes.md:47-54（差异清单未提胶囊）。
- 建议：去掉胶囊，或换成有信息量的标签（孩子班级/园所）并在 notes 差异清单中注明；若保留纯装饰，说明它不承载信息。

### 6. [notes-accuracy] notes 的暖色范围声明与实际不符：称暖色只出现在头像、角色胶囊和右上角轨道，实际 12 个菜单图标块全部是 --accent-wash 底 + --accent-deep 图标，空态徽标也用 --accent-wash；prompt 第 65 行本身就要求橙色出现在图标块。

- 证据：notes.md:7；styles.css:507-517（.menu-icon）、styles.css:262-265（.state-badge--warm）、index.html:175-248（12 个 .menu-icon）；_prompt.md:65。
- 建议：把 notes 的暖色范围改为「头像、角色胶囊、12 个图标块、空态徽标、轨道装饰」并重算面积占比；若仍 ≤10% 再保留结论。
