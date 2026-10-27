# teacher/21-doctor-calendar · 对抗性单页审查

- verdict：**WARN**（score 75/100）
- 源码覆盖：覆盖源码全部表单字段、5 类字典、草稿/发布/编辑/删除/通知确认与空/加载/失败/提交禁用态，但编辑态漏了「存为草稿」、草稿编辑漏了通知开关，变更确认行也缺旧值。

## P1 · 明显缺陷（3）

### 1. [logic] 变更确认弹层的「发布变更通知」开关不参与保存，用户即使关掉，保存后仍按事件原有 notify 状态通知家长。

- 证据：app.js:664 只用 setSwitch(changeNotifySwitch, suggestNotify) 做视觉预置；saveConfirm()（app.js:708-724）直接 applyEventChanges(event, pendingEdit.data)，而 readFormData() 在编辑态把 notify 固定为 editingBaseline.notify（app.js:582-584），全文件没有任何保存路径读取 changeNotifySwitch；源码 calendar.vue:140 是 save("published", confirmNotify.value)。示例事件如 evt-makeup-checkup 为 notify:true（app.js:84），改标题时开关预置为关，保存后 event.notify 仍为 true。
- 建议：saveConfirm() 读取 isSwitchOn(changeNotifySwitch) 并写回 pendingEdit.data.notify 后再 applyEventChanges；补「关掉开关后 event.notify 为 false」的自测。

### 2. [logic] 编辑草稿时「通知家长」开关被一并隐藏，且保存固定取 baseline，草稿发布会默认推送家长；源码对草稿显示开关且默认不推。

- 证据：openEditSheet() 无条件执行 notifyRow.hidden = true（app.js:462）；源码 calendar.vue:245-251 只在 editId && original?.status === 'published' 时隐藏，草稿走 v-else 显示 notify-row；startEdit 把 notify 置 false（calendar.vue:122-125），草稿 submitPublish 使用 form.value.notify（calendar.vue:150-166）。原型里示例草稿「小班补检」为 notify:true（app.js:77-90），编辑后发布仍为 true。notes.md:41 却写「新建 / 草稿用表单内开关」，与实现不符。
- 建议：改为 notifyRow.hidden = event.status === 'published'；草稿编辑时允许开关交互，并让 readFormData 在草稿编辑分支读取 isSwitchOn(notifySwitch)，已发布编辑才使用 baseline。

### 3. [completeness] 编辑态底部没有「存为草稿」，编辑草稿只能直接发布，已发布事件也无法保存回草稿。

- 证据：setFooterMode('edit') 同时把 saveDraftBtn.hidden 与 publishNowBtn.hidden 设为 true（app.js:401-407）；源码编辑态始终渲染「存为草稿」，submitDraft() → save("draft", false) 在 editId 存在时走 update（calendar.vue:122-125、252-255）。
- 建议：编辑态 footer 保留「存为草稿」（取消编辑 / 存为草稿 / 保存发布三键），点击时以 draft 状态保存并关闭弹层。

## P2 · 打磨项（4）

### 1. [design-core] 固定的「原型演示」胶囊压住最后一张事件卡的「删除」按钮，滚到底也无法点击。

- 证据：.page 底部只留 var(--space-xl)=16px（styles.css:61、tokens.css:64）；.mp-demo 距底 16px、toggle 高 28px（_shared/shell.css:167-186），占据视口底部 16–44px；.event-actions 高 44px（styles.css:598-618），最后一张卡（10/20 组「口腔检查」）的按钮行落在 16–60px，删除文字被覆盖，preview.png 底部右侧可见。
- 建议：给内容区增加演示条避让，如 .page 底部改为 calc(var(--space-xl) + 48px + env(safe-area-inset-bottom))，或让演示条默认折叠/移位以避开操作区。

### 2. [completeness] 变更确认行只显示新值，日期/时间/地点看不到原值，用户无法判断改动幅度。

- 证据：buildChanges() 对日期、时间、地点只 push 新值（app.js:628-644）；源码 substantiveLines() 生成「日期 旧 → 新」「时间 旧 → 新」「地点 旧 → 新」（calendar.vue:75-84）。
- 建议：变更行按「旧值 → 新值」渲染（如 2026-10-26 → 2026-10-29），标题/内容/类型保留「已更新」文案。

### 3. [logic] 日期组头被拆成「选中」和「折叠」两个按钮：点组头只选中不折叠，取消选中也不收起，与 _prompt 的硬性交互要求不一致；已选中的组还能被 chevron 折叠，形成实心圆标着不可见事件的矛盾状态。

- 证据：_prompt.md:37 要求「点击日期组头即选中/取消选中该组，并折叠/展开该组事件（aria-expanded 同步）」；toggleDaySelect() 只在选中时展开，取消选中不折叠（app.js:758-782）；折叠由独立 .day-toggle → toggleCollapse() 处理（app.js:785-796），且折叠后仍保留 is-selected 实心圆。app.js:756-757 注释自称要避免「实心圆标着看不见的事件」，实际可被 chevron 折叠违背。
- 建议：把选中与折叠合并到组头主按钮：选中/取消时同步 aria-expanded 与折叠状态；若坚持拆分，需在 notes 标为「设计提案/需产品确认」并更新验收口径。

### 4. [navigation] 返回箭头的隐含目的地写成「返回工作台」，但本页在源码里只能从晨检页进入，返回应回晨检。

- 证据：index.html:24 的 aria-label="返回工作台"；全仓 teacher/src 只有 teacher/src/pages/doctor/checkin.vue:185 一处 navigateTo({ url: "/pages/doctor/calendar" })；pages.json:138-141 是普通 navigateTo 子页。
- 建议：改为 aria-label="返回晨检"（或「返回上一页」），与真实入口一致。

## 2026-10-27 P1 修复记录

1. **变更确认弹层的「发布变更通知」开关不参与保存** → **已修**。`saveConfirm()` 先把 `isSwitchOn(changeNotifySwitch)` 写回 `pendingEdit.data.notify` 再 `applyEventChanges`，弹层开关成为最终 notify 值。自测（临时副本注入只读引用，不落仓库文件）：标题变更→预置关→直接保存得 `notify=false`；日期变更→预置开→手动关保存得 `false`；标题变更→手动开保存得 `true`。
2. **编辑草稿时「通知家长」开关被隐藏、保存固定取 baseline** → **已修**。`openEditSheet()` 改为 `notifyRow.hidden = event.status === 'published'`，草稿编辑显示开关并默认关闭（对齐源码 `startEdit` 的 `notify=false`）；`readFormData()` 仅在「已发布编辑」分支取 baseline，草稿编辑读 `isSwitchOn(notifySwitch)`。自测：草稿「小班补检」不改开关直接保存发布 → `notify=false`；打开开关再发布 → `notify=true`。
3. **编辑态缺少「存为草稿」** → **已修**。编辑态 footer 变为「取消编辑 / 存为草稿 / 保存发布」三键（新增 `.sheet-footer.is-edit` 三列均分，≤350px 回落单列；DOM 顺序同步调整）；新增 `handleEditDraftSave()`：以 draft 保存、不弹变更确认、`notify` 恒为 false，toast「已存草稿」，已发布事件可借此退回草稿。自测：编辑已发布事件点「存为草稿」→ 卡片出现草稿标签、总数与组内计数不变。

验证：`od-check --width 390` → `problems: []`；`od-audit` 本页无 ✗（仅体积 ⚠，因文件本身超 20KB 指南线，与本次修复无关）；`preview.png` 已按 390 宽 / dsf 2 重出；创建态仍为「存为草稿 / 立即发布」两键两列，无 console 报错。P2 四项未改动。
