# parent/24-feedback · 对抗性单页审查

- verdict：**WARN**（score 75/100）
- 源码覆盖：源码的 4 个区块（页头「意见反馈」+反馈/取消切换、类型三选+500 字内容表单、空列表提示、反馈卡片列表含状态/日期/园所回复）都有对应，类型/状态/回复字段与 0/1/2 状态语义齐全；缺的是 contact/images 上传与分页 total（notes 遗留 3/4、批次 A33 已记录），页头切换按 A34 改为常驻表单，另有源码没有的字数统计/三行展开/空态 CTA 属已声明提案。

## P1 · 明显缺陷（3）

### 1. [logic] 「失败」态把「列表加载失败」和「提交失败」两个互不相干的失败压进同一个 data-state，任一重试都会静默清掉另一条，并把并未加载成功的 3 条记录当成已加载显示

- 证据：styles.css:476-481（records-ok / records-error 全由 [data-state="failed"] 驱动）与 styles.css:508（bar-error 也由同一属性驱动）；app.js:371-379 recoverRecords() 只做 setState('normal')。实测（chrome-devtools 打开本页）：failed 态同时 bar-error=flex + records-error=flex；点列表「重试」后 bar-error 由 flex→none（提交失败未经任何重试就消失）；点提交条「重试」后 records-error 由 flex→none、records-ok 由 none→block、计数由「加载失败」变「共 3 条」
- 建议：拆成 listState 与 submitState 两个独立维度（或两个独立 data 属性）：列表错误面板只由列表请求结果控制，提交失败条独立显隐；两个重试各管各的，禁止互相清除

### 2. [logic] 「提交中」演示芯片没有出口：1.3s 后不会进入成功态，且 formLocked 让点击提交直接 return，页面永久停在转圈状态

- 证据：app.js:348-369 applyDemoState() 只 clearTimeout 不创建计时器；app.js:259-260 startSubmit 与 app.js:275-276 handleSubmit 都以 if (formLocked) return 早退。实测：点「提交中」后 2.2s 仍是 state=submitting、view=form、按钮文案「提交中…」，再点 #submit-btn 无任何反应。notes.md:25 与 notes.md:29 却声称「1.3s 后进入成功态」「成功态由「提交中」延时 1.3s …进入」
- 建议：芯片进入 submitting 时复用 startSubmit 的 1.3s 计时（或把成功态做成第 7 个芯片）；若保持现状，必须在 notes 里改成「成功态只能由真实提交进入」

### 3. [logic] 成功后「查看我的反馈」「再提一条」把输入清空却把状态留在 normal：必填框为空时提交按钮仍是品牌渐变可点、底栏仍是普通说明，与 notes 宣称的「清空内容也会即时进入阻断态」相反

- 证据：app.js:315-320（viewRecords）与 app.js:335-339（writeAnother）都是 setContent('') 之后 setState('normal')。实测点「查看我的反馈」后：textarea.value.length=0、submit aria-disabled=false、按钮 backgroundImage=linear-gradient(135deg, rgb(251,146,60)…)、可见提示为「提交后可在记录里查看进展」；只有再次输入（app.js:396-397）或点提交（app.js:283-287）才转 required。notes.md:22、notes.md:51 声称清空即阻断
- 建议：清空后统一走校验（setState('required')），或让 setContent('') 自身触发阻断态，保证「空内容=阻断」在成功返回路径上也成立

## P2 · 打磨项（4）

### 1. [notes-accuracy] notes 对实现细节的两处描述与代码不符：表单锁定的透明度写 .62（实际 .58）、按钮文字色写 --on-brand（实际用 --surface，且全页从未使用 --on-brand）

- 证据：notes.md:25「inert + opacity:.62」vs styles.css:551-554（opacity: 0.58）；notes.md:38「--accent-gradient + --on-brand」vs styles.css:575-579（color: var(--surface)）；styles.css 全文无 --on-brand 引用
- 建议：校正 notes 的数值与变量名，或把实现改成与 notes 一致（若按 token 契约应用 --on-brand 则改 CSS）

### 2. [design-core] 两处小字对比度低于 WCAG AA 4.5：必填阻断提示 12px --danger 压白仅 3.76:1；「园所回复」标题 11px --accent-deep 压 --surface-warm 仅 3.42:1

- 证据：页面实测 getComputedStyle + 对比度计算：.hint-required rgb(239,68,68)/rgb(255,255,255)=3.76（12px/500，styles.css:540-547）；.record-reply-title rgb(234,88,12)/rgb(255,250,240)=3.42（11px/600，styles.css:347-361）
- 建议：阻断提示改用 --fg 或更深的 danger 变体（≥4.5）；回复标题加深色阶或抬到 12-13px/700，或改压白底

### 3. [redundancy] 「共 N 条」计数位被当成状态位复用：loading 显示「加载中」、failed 显示「加载失败」，与骨架/错误面板重复表达同一状态，且这两态下计数口径直接消失

- 证据：app.js:101-110 syncCount() 按 state 改写文案；index.html:80 该插槽初始语义是「共 3 条」。实测 failed 时该位显示「加载失败」，同时 .records-error 面板已显示「反馈记录加载失败」
- 建议：计数位恒定只表达数量（未知时隐藏），加载/失败交给骨架与错误面板表达

### 4. [logic] 必填错误不就地：textarea 的 aria-invalid=true 没有任何视觉样式，输入框边框/底色不变，唯一提示在固定底栏

- 证据：样式表中无 aria-invalid 选择器（仅 app.js:125 设置属性），styles.css:113-129 的 .textarea-wrap 无 invalid 分支；index.html:70 字段本体无错误文案。实测 required 态 .textarea-wrap 边框仍为 var(--border)
- 建议：给 .textarea-wrap 增加 [aria-invalid="true"] 的 --danger 边框/浅底，并在 .field 内补一行就地错误文案（可与底栏提示并存）

## 2026-10-27 P1 修复记录

三条 P1 全部在原型内修掉，未改源码 / 契约；P2 四条按规则不动。

### P1-1 [logic] 列表失败与提交失败混在同一个 data-state —— 已修

拆成两条独立通道：`.mp-app[data-list-state]`（`ok / loading / empty / failed`）只由列表请求结果改写，`.mp-app[data-submit-state]`（`idle / required / submitting / failed`）只由提交请求结果改写；`app.js` 新增 `setListState()` / `setSubmitState()`，`setState()` 退化为「演示芯片场景 → 同时铺开两个维度」。`recoverRecords()` 只回写 `listState`，提交重试只回写 `submitState`，互不清除，也不会再把未加载成功的记录显示成已加载。场景不再成立时 `data-state` 置为 `custom`，芯片不再谎报。实测：失败态点列表重试 → 提交失败条仍在；点提交重试 → 列表错误面板仍在、records-ok 仍隐藏。

### P1-2 [logic] 「提交中」芯片没有出口 —— 已修

`applyDemoState('submitting')` 改为直接复用 `startSubmit(SAMPLE_CONTENT)` 的 1.3s 计时，不再只设状态不建计时器；`formLocked` 也随之由同一路径管理。实测：点「提交中」立即转圈，1.3s 后进入整页成功态，摘要卡内容为示例文案，与 notes「1.3s 后进入成功态」一致。

### P1-3 [logic] 成功后返回表单，空内容仍是可点的品牌渐变按钮 —— 已修

`showSuccess()` / `viewRecords()` / `writeAnother()` 在 `setContent('')` 之后统一 `setSubmitState('required')`（并清掉芯片场景），不再 `setState('normal')`。实测：点「查看我的反馈」或「再提一条」后 `submit-state=required`、`aria-disabled=true`、按钮为 `--bg` 灰底、`aria-invalid=true`，底栏显示「还需填写：反馈内容」。

### 验证

- `node scripts/od-check.mjs parent/24-feedback/index.html --width 390` → `problems: []`
- `node scripts/od-audit.mjs docs/design-previews/2026-10-25-full-redesign` → 50 页全部通过，本页无 ✗
- CDP 行为回归 14 项全过（初始态、六芯片、两条重试互不影响、提交中→成功、成功返回阻断、真实提交、成功后计数 4 条），无 console error
- `preview.png` 已按 390×2 重出（默认态渲染无变化，故与旧图一致）
