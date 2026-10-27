# teacher/03-notice-create · 对抗性单页审查

> 2026-10-27 token 对比度已统一修复（白字压橙 5.18–7.31:1），见 `_shared/REVIEW.md` §3.2。

- verdict：**WARN**（score 70/100）
- 源码覆盖：源码字段与主流程基本覆盖（类型/标题/正文/附件 2/9/多班接收/需确认/定时/模板/必填校验），但缺发布成功后的返回与复位、缺发布失败分支，提交闭环不完整。

## P1 · 明显缺陷（2）

### 1. [logic] 发布成功后既不返回工作台也不复位表单，还把发布按钮重新启用——同一份已发出的通知可以再次点「发布」，造成重复群发。

- 证据：app.js:478-493 的 confirmPublish 只在 900ms 后 showToast('已发布') 并重新启用 confirmPublish/dialogCancel，没有任何返回或重置；源码 teacher/src/pages/notice/create.vue:94-99 是 toast 后 600ms 执行 navigateBack()（无上一页则 reLaunch('/pages/index/index')），_prompt.md:19 也明确「600ms 后返回上一页」。
- 建议：成功后按源码模拟返回（toast「已返回老师工作台」）或至少清空并复位表单、保持发布禁用；若为评审方便留页，请在 notes.md 标注「留页复位」并复位成空表单，不能保留可重复发送的已填内容。

### 2. [design-core] 主按钮、选中类型、错误提示的 token 组合对比度不达 WCAG AA（实测 1.90–3.08:1），主操作文字和选中态在浅暖底上明显偏淡。

- 证据：styles.css:24 .chip.is-on（--accent-deep #e6762a 压 --accent-wash #fff7f0 = 2.83:1）、:56 .field-error（--danger #EF4444 压 --danger-wash #FEE2E2 = 3.08:1）、:108 .publish-btn（--on-brand #fff 压 --accent-gradient #ffa866→#ff8a3d = 1.90–2.35:1）、:129 .sheet-btn--strong 与 :157 .use-btn 同 accent-deep 组合；色值见 tokens.css:81-91/32-33，brief.md:45 要求「对比度满足 WCAG AA」。
- 建议：回 tokens.json 调整语义色（加深 accent-deep/danger，或主按钮改用深色字/更深的 accent），不要逐页打补丁；至少保证主按钮文字、选中胶囊文字、错误文案对底色 ≥4.5:1，并把这组 token 对比度记入批次缺口。

## P2 · 打磨项（6）

### 1. [navigation] 导航栏返回箭头是 <button>，但全页没有绑定任何 handler，点击无反应；同批子页都用 toast 明确返回工作台。

- 证据：index.html:24-26 有 .mp-back 按钮；app.js 全文没有 .mp-back 监听（shell.js:9-27 只处理演示条）；teacher/09-tasks-index/app.js:487-491 与 teacher/19-profile-index/app.js:112-116 都绑定了 toast「已返回老师工作台」；notes.md:9 称该箭头对应源码 uni.navigateTo 入栈。
- 建议：给 .mp-back 绑定与兄弟页一致的返回反馈（toast「已返回老师工作台」，目的地 teacher/01-workbench 存在），或显式 disabled/aria-disabled；不要把可聚焦按钮留成死控件。

### 2. [completeness] 发布失败分支缺失：确认发布后 900ms 无条件走成功，没有源码 catch 的失败 toast 与保留表单路径。

- 证据：app.js:485-492 的 setTimeout 固定 publishing=false、showToast('已发布')；源码 teacher/src/pages/notice/create.vue:100-102 在 catch 里 showToast(e?.message || '发布失败') 并保留表单；页面的「失败」演示芯片只覆盖班级列表加载失败（notes.md:27），不是提交失败。
- 建议：补提交失败演示分支（或在确认发布后按可控条件走失败）：保留已填内容、恢复按钮、toast 错误信息；并让演示条的失败态能区分「班级加载失败」与「发布失败」。

### 3. [logic] 定时发送默认时间是同一天 09:00，而状态栏时间是 9:41——打开定时发送即显示已经过去的发送时间，页面却写「到点后自动发送」。

- 证据：index.html:15 的 mp-time 是 9:41；app.js:15 默认 date '2026-10-27'、time '09:00'；app.js:103 开启后 schedHint 显示「10月27日 09:00 自动发送」；notes.md:76 自述未做「不能选过去时间」校验。
- 建议：与调度确认示例时间，把默认改到 9:41 之后（如 10:30）或次日；若保留 09:00，补「过去时间不可发布」的校验与提示，使状态栏和发送时间自洽。

### 4. [logic] 定时摘要和确认框用 shortDate 省略年份；日期滚轮可选 2025/2027/2028，选跨年日期时底栏和最终确认只显示「10月27日 09:00」，看不到年份。

- 证据：app.js:32 的 shortDate 只拼 M月D日；app.js:136 底栏摘要与 app.js:469 确认框 dlgWhen 都调用 shortDate；app.js:18 的 YEARS 为 [2025,2026,2027,2028]，而日期行 app.js:105 用 fullDate 显示年份。
- 建议：shortDate 在非当前年（或所有情况）补上年份，或确认框改用 fullDate；保证最终确认、底栏摘要与提交的 scheduled_at 年份一致可见。

### 5. [design-core] 弹层没有滚动锁定和焦点管理：openLayer/closeLayer 只切换 .is-open，打开时焦点不进入弹层、关闭后不回到触发元素；日期/时间滚轮是无 tabindex 的 div，键盘无法操作。

- 证据：app.js:276-277 的 openLayer/closeLayer 只 toggle .is-open；app.js:471 只给确认框 setTimeout 聚焦 confirmPublish，关闭时无 restore；index.html:198-221 的 .wheel 是带 aria-label 的 div，无 role/tabindex/keydown；styles.css:118-124 只做 visibility/transform。
- 建议：打开时记录 activeElement、把焦点移到 sheet 首个控件并锁 body 滚动，关闭时恢复焦点与滚动；给滚轮补 role=listbox/option（或提供可聚焦的日期/时间输入作为回退）并支持上下键。

### 6. [redundancy] 班级加载失败时，同一失败在接收范围行、脚下说明、底栏摘要三处重复出现，脚下「请检查网络后重试」与行内「重试」按钮语义重复。

- 证据：index.html:112-114 的 classError 显示「班级列表加载失败」+「重试」；app.js:116 把 classFoot 写成「请检查网络后重试」；app.js:129 的底栏摘要写「班级信息未加载，暂不能发布」；notes.md:27 只描述了行内错误与摘要，未提脚下这句。
- 建议：错误态保留行内「失败原因 + 重试」和底栏后果摘要，删除或改写 cell-foot（例如改为「仅显示你任教的班级」），避免同一失败三处同义重复。

## 2026-10-27 P1 修复记录

| # | 结论 | 改动 / 理由 |
|---|---|---|
| P1-1 发布成功后不复位、可重复群发 | **已修（留页复位）** | `app.js` 新增 `published` 状态：确认发布 900ms 后按源码 `submit()` 的收尾 toast「已发布，返回老师工作台」，新增 `resetDraft()` 把标题 / 正文 / 附件清空、其余字段回默认，`renderAll()` 回写输入框与缩略图；底栏摘要改为「通知已发布 · 表单已复位」，发布按钮保持禁用并显示「已发布」。开始写新草稿（标题 / 正文 / 附件非空）或切演示芯片（`setView` 清零）才解锁，发布处理函数同时加 `published` 守卫。单页原型不虚构工作台跳转，故按审查给出的「留页复位」分支处理，理由记入 `notes.md` 差异 8 与覆盖状态表。 |
| P1-2 主按钮 / 选中胶囊 / 错误文案对比度 1.90–3.08:1 | **暂缓（批次级）** | 三组都落在 token 层：`--on-brand` #FFFFFF 压 `--accent-gradient` #ffa866→#ff8a3d = 1.90–2.35:1（`.publish-btn`）；`--accent-deep` #e6762a 压 `--accent-wash` #fff7f0 = 2.83:1（`.chip.is-on`、`.sheet-btn--strong`、`.use-btn`）；`--danger` #EF4444 压 `--danger-wash` #FEE2E2 = 3.08:1（`.field-error`）。`tokens.css` 由 `scripts/sync-open-design-adapter.mjs` 从 `shared-design-tokens/tokens.json` 生成，文件头明确禁止手改，且 50 个页面的副本逐字节相同（md5 `c03f8da…`，teacher / parent 共用），改本页副本会让本页与其余 49 页脱钩；审查建议也要求「回 tokens.json 调整、不要逐页打补丁」，本页目录内无合规解法。已在 `notes.md` 遗留 7 登记，待批次统一压深品牌橙与语义红并同步三端。 |

**验证**：`node scripts/od-check.mjs …/03-notice-create/index.html --width 390` → `problems: []`、`overflowing: []`、`scrollWidth = clientWidth = 390`，滚动 400px 后 `statusbar top=0 / navbar top=20`；`od-shot --width 390 --dsf 2` 重出 `preview.png`（CSS 390×1125）；`od-audit` 批次 50/50 通过、本页无 ✗（`app.js` 20773 字节 / 20.29KB 触发「超 20KB」体积提示：本轮修复增加约 0.85KB，批次内另有 15 页同类提示，提示不计失败）。CDP（headless Chrome，390×900）16 项断言全过：确认发布后 toast「已发布，返回老师工作台」、标题/正文空、附件 `0/9`、按钮 `disabled` 且文案「已发布」、摘要「通知已发布 · 表单已复位」、确认弹层关闭；禁用态再点发布无反应；输入新标题即解锁；空表单点发布被 `validate()` 拦下不弹确认；完整再发一次仍复位；发布后点当前「正常」芯片即可恢复样例（`mp:state` 分支对 `published` 强制走 `setView`），切失败 / 正常芯片均正常；控制台 0 error。

P2 六项（返回箭头死控件、提交失败分支、定时默认 09:00 已过、跨年日期无年份、弹层滚动锁与焦点管理、失败三处重复）按要求未动。
