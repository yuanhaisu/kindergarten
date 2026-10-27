# teacher/05-growth-create · 对抗性单页审查

- verdict：**WARN**（score 68/100）
- 源码覆盖：新建主流程覆盖较全（日期/记录对象/快速记录/选择幼儿/照片/正文/徽章/底栏），但源码 ?id= 进入的编辑态（只读日期、保存修改、隐藏徽章、锁定可见范围）完全缺失，单张照片点名与批量点名的选图步骤也未还原。

## P1 · 明显缺陷（4）

### 1. [completeness] 源码的编辑态整条分支在原型里完全不存在：从成长详情「编辑」进入时是 update 而非 create，UI 有四处实质差异，原型只画了新建流。

- 证据：teacher/src/pages/growth/create.vue:59-67,145-154,471-486,771,863-887（?id= 进入编辑态：日期箭头只读、主按钮「保存修改」、隐藏授予徽章、底栏换成只读可见性提示）；05-growth-create/index.html:262-278 只有「发布班级记录」一个底栏，app.js:316-320 也只区分班级/个体两种发布文案；notes.md:8-16 信息层级与「覆盖状态」全表均无编辑态。
- 建议：补一个编辑态（演示芯片或第二份对照），至少画出日期箭头禁用、底栏可见性只读提示、徽章卡隐藏、主按钮「保存修改」，并在 notes 里说明与新建态的差异。

### 2. [logic] 「已选幼儿摘要」与「未选幼儿空态」由演示 chip 的 data-state 控制，而不是由真实选择人数控制，导致两条路都自相矛盾：正常流里取消到 0 人看不到空态插画，空态演示里选回 5 人仍显示「还没有选择幼儿」。

- 证据：app.js:182-208 renderPicked 从不切换 pickEmpty；styles.css:1 用 `.mp-app[data-state="empty"] .picked{display:none}` / `.pick-empty{display:flex}` 控制显隐；05-growth-create/index.html:156,159-177。空态 demo 下从 CTA 选人后 confirmPick（app.js:459-468）只更新 state.selected，不调用 setState('normal')，于是卡头「已选 5 人」与卡体「还没有选择幼儿」+「选择幼儿」按钮同屏；正常流里把 5 人全部取消则只剩「已选 0 人 / 还没有选择幼儿 / 编辑」的空摘要行。
- 建议：用 selected.size 驱动 picked/pickEmpty 互斥（0 人显示空态 CTA，>0 显示摘要），并把 data-state=empty 与真实选择解耦；confirmPick 后按人数自动切回摘要态。

### 3. [logic] 「已点名 N 张」不是按点名状态统计，而是写死 min(已完成张数, 2)；默认两张图一张「点名 6」、一张「AI 建议 3」，后者尚未点名，却显示已点名 2 张。

- 证据：app.js:248 `namedTip.textContent = \`已点名 ${Math.min(done, 2)} 张\``；默认数据 app.js:476-481，角标见 05-growth-create/index.html:198（模板初始值「已点名 2 张」）；源码口径是 `form.media.filter(m => namedCount(m)).length`（teacher/src/pages/growth/create.vue:694），namedCount 只数 student_ids。按此默认态应为 1 张；且新增第 3 张完成后仍会停在 2。
- 建议：给媒体模型加 named/student_ids 字段，按「有已确认名单的照片数」计数，去掉 min(…, 2)。

### 4. [logic] 「批量点名」流程走不到任何结果：没有源码的「先勾选照片」步骤，确定后只弹一句成功 toast，照片角标与已点名计数全不变化；单张照片的点名角标也不可点。

- 证据：app.js:444-447 点「批量点名」直接 openSheet('batch')，无选图；app.js:459-468 batch 分支只 `showToast('已为 N 张照片追加 M 位幼儿')`，不修改 state.media；照片角标由 app.js:226-228 渲染为不可点的 span；源码流程见 teacher/src/pages/growth/create.vue:692-705（全选/选择学生/取消）与 824-847（全班当日到场、AI·高/AI/已有标记）。
- 建议：还原「批量模式勾选照片 → 选择学生」两段，确定后更新每张照片的点名名单与角标；单张角标加点击入口，或把这些入口明确禁用并在 notes 标注未实现。

## P2 · 打磨项（4）

### 1. [notes-accuracy] notes 声称九宫格覆盖「失败重试」，但代码里没有任何路径能把媒体置为 failed，失败瓦片与 retryUpload 是永远不可达的死代码。

- 证据：notes.md:14「完成 / 上传中百分比 / 失败重试 / ＋添加」与 notes.md:47「失败后重试入口」；app.js:252-293 的 addMedia/retryUpload 只会 uploading→done，app.js:483-507 applyState 只造 done/uploading；kind:'failed' 只出现在渲染分支 app.js:221-224 与计数 app.js:299-304，无赋值处。
- 建议：在加载/空态里造一张 failed 瓦片演示重试，或删掉 notes 里这两处「失败重试」的声称。

### 2. [logic] 照片说明行是写死的两行，不随照片增删重建：删掉一张照片后仍显示两条说明（其中一条对应已删除的图），新增第 3 张也没有说明行。

- 证据：05-growth-create/index.html:204-213 为静态两个 .desc-row；app.js:212-250 renderMedia 只做 `descList.hidden = done === 0`，从不按 state.media 重建；源码是 `v-for="(m, i) in form.media"` 每张一行（teacher/src/pages/growth/create.vue:743-748）。
- 建议：由 state.media 生成说明行（缩略图取 artFor(index)），删除/新增时一并重建。

### 3. [logic] 徽章「授予给」与记录目标脱节：固定只有乐乐/小雨/朵朵且乐乐默认勾选，切到指定幼儿只选小满/果果时徽章仍默认授予乐乐；notes 还声称没有做默认态演示、保留了提示文案，两者都不成立。

- 证据：05-growth-create/index.html:249-251 三个写死 chip、乐乐 is-on=true，卡内没有任何「不选则默认授予给目标学生」提示；notes.md:53 称「原型只保留提示文案（未做默认态演示）」；源码默认目标取本记录 student_ids（teacher/src/pages/growth/create.vue:107-113），提示文案见 797-799。
- 建议：授予给列表改为跟随记录已选幼儿（全班记录时用班级名单），默认不预选并补源码那句提示，或删掉 notes 的不实声明。

### 4. [design-core] 多组表单控件的触控高度低于 44px 基线，而同页其它控件都用伪元素外扩到 44px，属规范执行不一致。

- 证据：styles.css:1-2：`.tag` min-height 34px、`.chip` 36px、`.badge-chip` 38px、`.desc-input` 36px 均无外扩伪元素；对照同文件 `.mode-btn` 40px + ::before inset -4px（48px）、`.mini-btn` 32px + ::before inset -6px（44px）、`.sw` 24px + ::before inset -10px（44px）。
- 建议：给 tag/chip/badge-chip/desc-input 补同款 ::before 命中区外扩或 min-height:44px，视觉高度可保持不变。

## 2026-10-27 P1 修复记录

- **1. 编辑态缺失 → 已修**：演示条新增「编辑」芯片（`data-mp-state="edit"`）；切入后日期左右箭头禁用、记录对象分段禁用、隐藏「授予徽章」卡、底栏同步开关隐藏并改为只读可见范围提示（「该记录对全班家长可见 / 该记录仅被选中的幼儿家长可见」）、主按钮改「保存修改」；notes 的覆盖状态表已补编辑行。
- **2. 已选摘要 / 空态由 data-state 驱动 → 已修**：删掉 `styles.css` 里 `.mp-app[data-state="empty"]` 的两条显隐规则，改由 `renderPicked()` 按 `selected.size` 切换 `pickedBox` / `pickEmpty` 的 `[hidden]`，`picker` 与 data-state 彻底解耦；空态 CTA 选人确定后立即切回摘要（显「已选 2 人」与摘要行）。
- **3. 已点名 N 张写死 min(done,2) → 已修**：媒体模型由 `badge` 字符串改为 `{ students, suggestions }`；「已点名 N 张」按「有已确认名单的照片数」统计（默认 1 张），角标按 `点名 N / AI 建议 N / 待点名` 渲染；新增、删除、单张点名、批量点名后实时刷新。
- **4. 批量点名无结果、单张角标不可点 → 已修**：还原源码两段式——「批量点名」进入批量模式（照片瓦片现勾选圈，工具条给全选 / 选择学生 / 取消），勾照片后再选学生，确定按并集追加并即时刷新各照片角标与计数；新增单张「照片点名」弹层（点瓦片或角标进入，可改完整名单）；未选照片 / 无照片时给 toast 提示，不再空跑。
- **验证**：`od-check` problems `[]`；`od-audit` 本页 0 ✗（仅 styles.css 22KB、app.js 24KB 体积 ⚠）；CDP 功能测试覆盖 启动态 / 空态选人 / 批量点名 / 单张点名 / 删除 / 编辑态 / 回正常态，值全部符合预期且无 console 报错；preview.png 与三个状态截图经视觉核对通过。
- **P2 未动**：失败的 retry 不可达、说明行静态、授予给默认值、触控 <44px 四项维持原判，待后续处理。
