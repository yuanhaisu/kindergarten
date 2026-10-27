# parent/07-leave-create · 对抗性单页审查

- verdict：**PASS**（score 84/100）
- 源码覆盖：已逐项对照 create.vue：区块顺序（类型/时间/原因/接送人/附件）、必填校验、日期下限与开始日期后移时结束日期自动跟随、提交中禁用、成功 toast 全覆盖，孩子上下文与附件 UI 为 A3/A2 明确标注的设计提案，加载/失败/成功复位为批次新增并已在 notes 说明，未发现源码能力缺失或源码入口遗漏。

## P2 · 打磨项（5）

### 1. [logic] 日期弹层翻月会无条件清空已选日期：翻到下月再翻回原月，选中态消失、「确定」重新变灰，用户以为选过的日期被丢掉。

- 证据：app.js:542-552 的 moveMonth 在每次翻月时执行 sheetPending = null，再 renderCalendar()→syncConfirmState()（app.js:537-540 在 !sheetPending 时把 #confirmDate 置 disabled）。浏览器实测：选中 2026-10-28（确定可用）→ 下一月 → 上一月回到 10 月，28 日无选中样式、确定按钮 disabled。
- 建议：翻月时保留 sheetPending；若当前待选日期落在新月份之外再清空，或将选中值与显示月份解耦，syncConfirmState 只在日期不可选时禁用确定。

### 2. [logic] 提交成功复位后 data-state 回到 normal、演示条「正常」仍高亮，但表单已清空，常驻行立刻变成红色阻断文案「还需填写：开始日期、结束日期、请假原因」、按钮置灰——页面对外声称处于正常态，实际呈现的是全空报错态，状态机名实不符。

- 证据：app.js:353-361 成功定时器里先 clearFormAfterSuccess() 再把 app.dataset.state 设为 'normal' 并 syncChips('normal')；app.js:243-256 在 normal 且未填齐时输出 `还需填写：…` 并加 is-blocking（styles.css:781-784 红色）。浏览器实测：toast「提交成功」后日期显示「请选择」、类型无选中、常驻行 class 为 submit-hint is-blocking、颜色 rgb(239,68,68)、按钮 aria-disabled=true，同时「正常」芯片处于按下态；notes.md:21 对正常态的定义是病假选中+日期已填的示例数据。与 A4 记录的「留页复位 vs navigateBack」不是同一件事，这里是复位后的状态/芯片错配。
- 建议：复位后不要标记为 normal：要么按 A4 直接 navigateBack（或显示「已提交」中性态并取消芯片选中），要么复位后保持 neutral 状态、常驻行显示中性提示，避免成功后第一眼出现报错式红字。

### 3. [design-core] 附件区按 4 列网格排布，但业务上限只有 3 张：默认只有 1 张缩略 + 1 个添加块，只占半行；加到 3 张后添加块隐藏，第 4 列永远空着，右侧留下一条无内容的空槽，视觉上像少了一张图。

- 证据：styles.css:513-519 .upload-grid 为 grid-template-columns: repeat(4, minmax(0,1fr))；app.js:232 与 app.js:303 在 photos.length >= 3 时把 #addPhotoBtn 置 hidden，app.js:307-309 硬性拦截第 4 张。浏览器实测：列宽 76.5px×4，默认 2 个格子（1 照片 + 添加），满额 3 张时第 4 列空白。notes.md:14 也写着「附件 4 列方格」，与「最多 3 张」自相矛盾。
- 建议：改为 3 列网格（与上限 3 张对齐），或保持 4 列但在满额时用 1 个占位格补齐；同步修正 notes.md 的「4 列」表述。

### 4. [design-core] 报错/阻断文字用 --danger #EF4444：在白色卡面上仅 3.76:1、在 --danger-wash 失败条上仅 3.08:1，12–13px 中等字重不属于大字号文本，未达到 brief 要求的 WCAG AA 4.5:1；字数接近上限的 11px 提示用 --accent-deep 也只有 3.56:1。

- 证据：_shared/brief.md:45 要求「对比度满足 WCAG AA」；tokens.css:32-33 --danger:#EF4444 / --danger-wash:#FEE2E2。styles.css:368-375 .field-error 用 var(--danger)+var(--text-sm) 12px；styles.css:781-784 .submit-hint.is-blocking 用 var(--danger)+13px；styles.css:851-878 .submit-failure p 为 13px var(--danger) 压 var(--danger-wash)；styles.css:436-439 .reason-foot.is-near 为 11px var(--accent-deep)（#EA580C，3.56:1）。计算值：3.76:1 / 3.08:1 / 3.56:1。
- 建议：回 tokens.json 把 danger 文字色调深（如 #B91C1C，白底 6.47:1、wash 上 5.30:1）并重建产物；失败条保留浅底但文字改用深色 token；is-near 改用更深的强调文字色。属 token 级修复，不要逐页打补丁。

### 5. [design-core] 日历星期表头和附件文件名把字号硬编码为 10px，绕过 token 体系，且低于 tokens 最小档 --text-xs 11px，390 下中文 10px 明显偏小、可读性差。

- 证据：styles.css:551 .photo-name { font-size: 10px }（「病假条/照片 2」等附件名）；styles.css:1048 .weekday-grid { font-size: 10px }（日一二三四五六）；tokens.css:39-41 最小档 --text-xs:11px、正文 --text-base:13px；浏览器 computed style 确认两处均为 10px。
- 建议：改用 var(--text-xs)（11px）；若 7 列日历表头确实需要更小字号，应在 shared-design-tokens/tokens.json 新增专用 token 并重建，而不是在页面里写死 px。
