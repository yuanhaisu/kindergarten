# parent/16-pickup-authorized-create · 对抗性单页审查

- verdict：**WARN**（score 78/100）
- 源码覆盖：源码的 4 个字段（姓名/手机号/关系/身份证）、5 个权限开关、审核提示与提交按钮全部覆盖；唯一未画的是成功后的 navigateBack（notes #9 说明为评审停留），无其他源码能力缺失。

## P1 · 明显缺陷（1）

### 1. [logic] 「失败」态下点主「提交」按钮会重新走成功流程，但失败条不消失：成功 toast 与「提交失败，请重试」同屏，状态自相矛盾。

- 证据：app.js:361-380 的提交处理只拦 loading/submitting，未清理 data-state=failed（对比 app.js:382-390 的 retryBtn 会置 normal）；styles.css:502-504 失败条由 .mp-app[data-state="failed"] 常驻显示。DOM 实测：切「失败」芯片→点 #submitBtn，1.6s 后 data-state 仍为 failed，#failBar display:flex，#toast 同时显示「已提交，待园所审核」。
- 建议：提交前先退出 failed 态并同步芯片/演示条（与 retryBtn 一致），或提交中隐藏失败条；也可在 failed 态禁用主提交、只留「重试」。

## P2 · 打磨项（5）

### 1. [logic] 关系选择面板打开时没有锁定背景滚动，遮罩后面的页面仍可滚动。

- 证据：app.js:207-223 openSheet 只设置 hidden/data-open/aria-expanded；styles.css:619-647 的 .sheet-layer 无滚动锁样式。DOM 实测：#relationBtn 点击后 body、html 的 computed overflow 仍为 visible（页面可滚 175px）。
- 建议：打开时给 body 加 overflow:hidden（或锁类）并在 closeSheet、状态切换、卸载时成对恢复；面板内部保留 overscroll-behavior: contain。

### 2. [redundancy] 同一信息重复：审核规则在页尾说明条与底部固定栏各写一遍，滚到底部两段同屏；身份证「选填」也在标签和 placeholder 各出现一次。

- 证据：index.html:139 与 index.html:168 均为「审核通过后才可接送」语义；DOM 实测 scrollY=175 时 .notice 664–724 与 #submitNote 758–776 同时在视口内。index.html:83「选填」+ index.html:85 placeholder「选填，便于门卫核验」。
- 建议：审核规则只保留一处（建议说明条），底部常驻行在填齐时改为更短的进度提示；身份证选填状态只留标签右侧，placeholder 去掉「选填」。

### 3. [notes-accuracy] notes.md 三处声明与实际不符：①称橙色落在「关系值」但关系值是 --fg；②称上下文条显示「乐乐 · 小一班」但页面没有班级；③称「加载」是表单真实状态但源码没有首屏加载，loading 只包住提交。

- 证据：notes.md:6 vs styles.css:284（.relation-value color: var(--fg)）；notes.md:59 vs index.html:50（仅「为 乐乐 添加接送人」）；notes.md:28 vs parent/src/pages/pickup/authorized-create.vue:26,37,46（loading 仅在 handleSubmit 置真/假）。
- 建议：按实现修正 notes；若确实要橙色关系值或展示班级，则改页面并补数据来源，不要把提案写成现状。

### 4. [design-core] 正文低于批次 13px 下限：审核说明条正文 12px，权限副文案 11px。

- 证据：styles.css:399 的 .notice-text 使用 var(--text-sm)（tokens.css:40=12px）；styles.css:337 的 .perm-sub 使用 var(--text-xs)（tokens.css:39=11px）；_shared/brief.md:45 要求「正文 ≥ 13px」。
- 建议：说明条正文提到 13px；副文案若确需 11px，需在批次规范里明确豁免（当前 _prompt.md:58 的 12px 与 brief 冲突），否则统一到 13px。

### 5. [redundancy] 滚到页尾时「原型演示」胶囊压住审核说明条右下角（与卡片相交约 70×19.6px），notes 的验收结论却说未遮挡内容。

- 证据：index.html:197 行内 bottom: calc(112px + env(safe-area-inset-bottom))；DOM 实测 scrollY=175 时 .mp-demo-toggle（704–732 / 348–418）与 .notice（664.4–723.6 / 61.5–423.5）相交，与 .notice-text 文本盒相交 60.5×8.6px；notes.md:74 声称「均未遮挡内容」。
- 建议：把演示胶囊移到导航栏下方/左侧空白区，或与页尾说明条错开；同时修正 notes 的验收描述。

## 2026-10-27 P1 修复记录

- **P1-1 [logic] 失败态点主「提交」后失败条不消失** — 已修。`app.js` 提交处理在通过 busy 守卫后先判断 `data-state === 'failed'`，与 `retryBtn` 一致地置回 `normal` 并同步芯片与演示胶囊位置，再进入校验/提交流程；失败条与 172px 底部预留随之撤销，不再与「已提交，待园所审核」toast 同屏。CDP 实测：失败态点提交 1.7s 后 `data-state=normal`、`#failBar display:none`、芯片回「正常」、toast 正常显示。`notes.md` 失败态一行已同步该交互。
- 验证：`od-check --width 390` → `problems: []`；`od-shot` 重出 `preview.png`（390×1019，201,680 bytes）；`od-audit` 批次 50 页通过、本页无 ✗。P2 五项未动。
