# parent/17-tasks-index · 对抗性单页审查

- verdict：**PASS**（score 80/100）
- 源码覆盖：源码的区块顺序、卡片字段、双按钮、三分类与空/加载/错误/内联错误/禁用分支均已覆盖；仅缺任务图标上的未读红点，未绑定孩子错误分支虽已实现但演示不可达。

## P2 · 打磨项（6）

### 1. [completeness] 示例数据自相矛盾：一次性任务「周末亲子手工」写「累计打卡 1 次」，却同时是未提交（data-submitted="false"）、已结束，且不出现在「已打卡」。按接口口径，一次性任务周期内提交过 1 次即应视为已提交，卡片状态与计数对不上。

- 证据：index.html:181「一次性任务 · 累计打卡 1 次」；index.html:167 data-submitted="false"、index.html:175「已结束」徽标、index.html:188 主按钮禁用；backend/api/api-parent/06-task.md:51-52（checkin_count=当前孩子提交次数；一次性=周期内是否已提交）；DOM 实测「已打卡」仅渲染 给绿萝浇水/我会自己穿鞋 两张卡。
- 建议：把该卡改为「累计打卡 0 次」（与未提交一致），或补齐提交态：徽标「已提交」、主按钮「查看记录」，并让它出现在「已打卡」。

### 2. [logic] 空态 CTA 只会切换分类：账号确实没有任何任务时，「全部」空态点「查看待办」→「进行中」空态，再点「查看全部」→ 回到「全部」空态，两个空态互相指向，用户没有出口。

- 证据：app.js:43-59（两套空态文案与 CTA）、app.js:260-267（CTA 只调 setFilter 切分类）；index.html:222-225；源码 parent/src/pages/tasks/index.vue:126-128 的空态本无 CTA，属原型新增能力。
- 建议：空态 CTA 改为可恢复动作（重新加载 / 去班级墙 / 返回），或仅在另一分类确有数据时才显示「切分类」CTA。

### 3. [notes-accuracy] notes 声称覆盖「未绑定孩子」错误文案，但演示面板没有该状态入口（data-student-bound 写死 true，5 个状态芯片都不切绑定态）；强行置 false 后唯一按钮 fullRetry 被隐藏，错误面板没有任何可点操作，成为死胡同。

- 证据：notes.md:21；index.html:13（data-student-bound="true"）、index.html:243-247（5 个芯片）；app.js:194-203（fullRetry.hidden = !bound）；DOM 置 false 后 #errorState 内只有一个 hidden 的 #fullRetry，无其他按钮。
- 建议：演示面板增加「未绑定」入口（或 URL 参数），并把该分支主操作改为「去绑定」（或保留重试），不要只留不可点的说明文字。

### 4. [completeness] 源码在任务图标右上角用红点标记补充回应未读（.tc-dot），接口文档也明确要求列表显示该红点；原型只有卡内文字条「老师又给你留了一句悄悄话」，图标上没有红点，扫视列表时未读不够显眼。

- 证据：parent/src/pages/tasks/index.vue:139-142（v-if="item.hasUnread" 的 .tc-dot）；backend/api/api-parent/06-task.md:55（「用它显示任务列表红点」）；index.html:144-146 的 .task-icon 仅含一个 SVG（DOM 实测 shoes 卡 .task-icon 无红点节点），未读仅在 index.html:161 文字条出现。
- 建议：在 .task-icon 右上角加 6–8px 圆点（--danger 或 --accent，aria-hidden），与文字条并存，对齐源码与接口口径。

### 5. [redundancy] 「本周期还有 2 个任务待提交」与「进行中」tab 的红色数字 2 是同一个数的重复呈现；且该提醒在「已打卡」分类下仍显示——用户正在看已提交卡片，上方却提示还有 2 个待提交，位置与语境错位（批次要求该提醒位于「进行中」列表上方）。

- 证据：index.html:51（badge 2）与 index.html:57-60（提醒 2）同屏；app.js:184-191 仅按 state 与 pending 判断、不区分 currentFilter；_prompt.md:46 指定「『进行中』列表上方加一行 12px 待完成提醒」；notes.md:41 只解释「全部」下的保留，未覆盖「已打卡」。
- 建议：仅在 currentFilter==='active' 时显示该提醒（与徽标合并为单一来源），或改成不与徽标同数的全局提醒文案。

### 6. [navigation] 返回按钮的 aria-label 与 toast 都写死「返回首页」，但任务页也可从「我的」进入，返回实际是回上一页；同级子页普遍用「返回」，本页是唯一写「返回首页」的，目的地声明不成立。

- 证据：index.html:23（aria-label="返回首页"）；app.js:280-284（showToast('正在返回首页')）；parent/src/pages/profile/index.vue:11 同样以 navigateTo 打开 /pages/tasks/index；同级页面 15 处 mp-back 为 aria-label="返回"。
- 建议：改为「返回」/中性 toast（或不弹 toast），与批次其余子页一致。
