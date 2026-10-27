# parent/15-pickup-authorized · 对抗性单页审查

- verdict：**WARN**（score 74/100）
- 源码覆盖：源码覆盖完整：安全提示、添加行、今日接走人横幅（已指定/未指定）、人员卡的字段与操作、加载/空/失败分支都在；仅真实头像图片未呈现（notes 已声明），页脚说明与错误态属于增强而非遗漏。

## P1 · 明显缺陷（1）

### 1. [logic] 张建军卡片同时出现黄色「待审核」和绿色「可接送」：review_status=0 已使其不可指定（按钮降级、底部说明也写明审核后才能指定），但权限标签仍用 success 绿表达「可接送」，审核状态、权限标签、按钮状态三者互相矛盾；同页刘芳（已撤销）卡片又完全没有权限标签行，两张不可指定卡的处理口径也不一致。

- 证据：index.html:189-203（张建军 data-review-status="0"、badge--pending「待审核」、permission-tag--pickup「可接送」）；index.html:212-228（刘芳 data-status="0"、badge--revoked「已撤销」、无权限标签）；styles.css:381-384（可接送使用 --success/--success-wash 绿色）。
- 建议：统一权限标签口径：按有效授权（review_status===1 && status===1）显示，或在待审核/已撤销卡上都标注「配置权限，审核通过后生效」；不能只在待审核卡保留绿色「可接送」。

## P2 · 打磨项（6）

### 1. [logic] 「删空 → 失败 → 重试」会把页面带回 normal 状态，但列表仍是 0 张卡：工具行显示「0 位授权接送人」、横幅回到「今日尚未指定接走人」，而不是页面设计好的空态，正常/空态互斥被打破。

- 证据：app.js:403-411（重试计时器结束后只调用 setPageState('normal')，没有 restoreInitialRoster）；app.js:324-329（删到最后一张时 setPageState('empty')）；app.js:145-149（只有 mp:state 回到 normal 才恢复列表 DOM）。
- 建议：重试成功后按 getCards().length 判断：为 0 时 setPageState('empty')，否则 normal；或让重试统一走 restoreInitialRoster 后再定位状态。

### 2. [logic] 删除「今日接走人」时确认框只说「确定删除 X 吗？」，没有像换人指定那样提示今日接送二维码会失效；删除后 designatedId 被置空，二维码实际已不可用，用户可能在不知情下作废已发给门卫的凭证。

- 证据：app.js:218-224（删除框正文只有确认删除）；app.js:205-209（换人框明确写「今日已生成的接送二维码将全部失效」）；app.js:315-319（删除被指定者时 designatedId=null），对应源码 parent/src/pages/pickup/authorized.vue:71。
- 建议：删除对象带 is-designated 时，在确认正文追加「删除后今日接送二维码将失效，需要重新指定」。

### 3. [logic] 两个居中确认弹层打开时没有锁定背景滚动：遮罩是 fixed，但 body 仍可滚动，关闭时也没有配套的滚动恢复（因为没有锁）；在 390×844 视口下弹层背后会跟着滑动，与微信 showModal 的原生行为不一致。

- 证据：styles.css:717-726（.modal-layer 仅 position:fixed/inset，没有任何 body overflow 处理）；app.js:172-198（openDialog 只切 inert/aria-hidden）；app.js:158-170（closeDialogs 无滚动锁/恢复逻辑）。
- 建议：打开时给 document.documentElement（或 .mp-app）加 overflow:hidden 并记录原值，closeDialogs 成对恢复。

### 4. [navigation] 返回控件的隐含目的地写死为「接送」，但源码里本页只从「我的 → 授权人管理」和「接送二维码 → 去指定今日接走人」进入，pages/pickup/index 没有任何入口到本页，返回「接送」这个上下文并不存在。

- 证据：index.html:63 aria-label="返回接送"；app.js:391-393 toast「已返回接送」；实际入口 parent/src/pages/profile/index.vue:17、parent/src/pages/pickup/qrcode.vue:72；parent/src/pages/pickup/index.vue 无 authorized 跳转。
- 建议：改成中性「返回上一页」，或按入口上下文显示「返回我的 / 返回接送二维码」，不要指向不存在的接送入口。

### 5. [design-core] 安全提示正文、关系/手机号、页脚说明都用 --text-sm=12px，低于批次「正文 ≥13px」的下限（仅徽标可 11px）；这三处都是需要用户阅读的正文，尤其手机号在门卫核验场景下偏小。

- 证据：styles.css:95-101（.safety-text font-size:var(--text-sm)）、styles.css:350-357（.person-meta）、styles.css:467-477（.footer-note）；tokens.css:40（--text-sm: 12px）；_shared/review-rubric.md:40 与 _shared/brief.md:45 要求正文 ≥13px。
- 建议：这三处改用 --text-base（13px）；若产品坚持 12px，需在 notes.md 里声明为批次例外，而不是默认使用。

### 6. [redundancy] 「今日接走人」在首屏被重复编码四次：横幅写「今日接走人 / 王建国」，首卡又有同文徽标，再加橙色描边与头像染色；同一状态的多重信号彼此抢注意力，首卡与横幅在视觉上互相复制。

- 证据：index.html:95-103（横幅 kicker「今日接走人」+ 姓名「王建国」）；index.html:120-126（首卡 data-today-badge「今日接走人」）；styles.css:266-268（is-designated 橙色内描边）；styles.css:284-287（被指定头像 accent 染色）。
- 建议：收掉最弱的一处信号：建议保留横幅（全局状态）+ 徽标（卡片内可读定位），把头像染色降为中性；或把徽标文案改为「当前」避免与横幅同文。

## 2026-10-27 P1 修复记录

- **P1-1 [logic] 张建军卡的绿色「可接送」与待审核状态矛盾，且与刘芳卡口径不一致**：已修。权限标签统一按有效授权（`review_status=1 && status=1`）渲染——删除张建军（待审核）卡上的绿色「可接送」标签；刘芳（已撤销）卡原本就没有该行，两张不可指定卡现在口径一致：状态只由「待审核 / 已撤销」徽标 + 降级按钮 + 页脚说明表达，前三张有效卡保留「可接送 / 看记录 / 看成长」。未改 `app.js`，也未发明源码没有的能力（源码同样只在审核通过且有效时允许指定）；notes.md 的「覆盖状态 · 正常 / 禁用」与「与现状的差异 5」已同步该口径。P2 六项按任务要求未动。

验证：`od-check` `problems: []`（390 宽，scrollWidth 390 = clientWidth 390，bodyHeight 1186，滚动后 statusTop 0 / navTop 20）；`od-shot` 重出 `preview.png`（390×1186，dsf 2，209,721 bytes）；`od-audit` 批次 50 页全部通过、本页无 ✗；mcp-vision 复核预览：张建军/刘芳卡无绿色权限标签，前三张有效卡标签正常，无裁切、无错位。
