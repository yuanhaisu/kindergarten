# parent/23-student-index · 对抗性单页审查

- verdict：**WARN**（score 74/100）
- 源码覆盖：源码主体覆盖较全（档案字段、晨检列表/空/局部失败、绑定列表与解绑确认、加载/整页失败都在），但缺 is_isolated 隔离提示与晨检行内加载态，且空态把字段缺失/晨检为空/单孩合并成一个状态。

## P1 · 明显缺陷（2）

### 1. [design-core] 档案卡 2×2 事实格被 max-width:100% 抵消了左右负边距，整块相对卡片内容左移 16px：分隔线左端顶到卡片外缘，右端却比卡片内容右缘短 16px，第二列整体偏左。

- 证据：styles.css:119-125 `.facts{max-width:100%;margin:var(--space-xl) calc(-1*var(--space-xl)) 0}`；DOM 实测 `.facts` x=61.5/right=391.5，而 `.identity`/`.health-note` x=77.5、卡片内容右缘 407.5（左右各偏 16px），右侧列分隔线随之错位。
- 建议：移除 `.facts` 的 `max-width:100%`（或改为 `width: calc(100% + 2 * var(--space-xl))` 保留兜底），让事实格左右出血对称。

### 2. [completeness] 晨检卡缺源码已渲染的 is_isolated 分支（「已在隔离室等待家长」），原型既无该行也无样式；notes 却把该字段误列为「源码与原型都未展示」。

- 证据：parent/src/pages/student/index.vue:118 `<text v-if="r.is_isolated" class="health-isolated">已在隔离室等待家长</text>`；接口 backend/api/api-parent/12-health.md:29 返回 is_isolated；index.html:90-115 仅日期/胶囊/异常/体温/备注，styles.css 中无 `.health-isolated`；notes.md:72 将其列入「源码与原型都未展示」。
- 建议：补隔离提示行与对应样式（warning/danger 语气），并在示例或说明中给出可达状态；同步修正 notes.md:72。

## P2 · 打磨项（5）

### 1. [logic] 解绑确认弹层打开时未锁定页面滚动，也未把背景移出可达树，遮罩后的内容仍可滚动并被读屏访问。

- 证据：app.js:121-135 openDialog 只设置 hidden 与焦点，无 overflow/inert/aria-hidden；实测 html/body 的 overflow 均为 visible，弹层打开时 window.scrollTo(0,300) 使 scrollY 0→136。
- 建议：打开时锁 documentElement/body 的 overflow 并在关闭时恢复；给 .page 加 aria-hidden 或 inert。

### 2. [completeness] 源码晨检卡有独立的行内加载分支（返回页面时重载晨检），原型只有整页骨架与晨检失败态，无法呈现「档案已显示、晨检加载中」。

- 证据：parent/src/pages/student/index.vue:103-104 `<view v-if="healthLoading" class="state-inline">加载中...</view>`；:61-64 onShow 会再次 loadHealth；index.html:90-132 只有 records/empty/error 三块，五个演示芯片也没有该态。
- 建议：补晨检行内加载骨架/文案（可并进第 5 芯片或新增演示态），或在 notes 中明确标注该态未覆盖。

### 3. [logic] 空态把三个源码中相互独立的条件（档案字段缺失、近 7 天无晨检、绑定孩子 ≤1）绑成同一状态，导致「档案完整但晨检为空」与「档案完整但只绑 1 个孩子」两个真实分支不可达；notes 还称隐藏绑定卡与源码 v-if 一致。

- 证据：parent/src/pages/student/index.vue:98（health_note 自身为空才隐藏）、:107（healthRecords 为空是独立空态）、:124（students.length>1 才是绑定卡条件）；原型 styles.css:433-439 与 :578-580 均由 `[data-state="empty"]` 同时触发；notes.md:23 声称与 `students.length > 1` 一致。
- 建议：拆分空态分支，或让「晨检为空」「单孩」在正常档案下独立呈现；修正 notes 的因果说明。

### 4. [notes-accuracy] notes 把数据加载写成「并行 GET profile 与 GET health」，并称档案也有独立 error；与源码实际的请求顺序和错误处理不符。

- 证据：notes.md:3；源码 parent/src/pages/student/index.vue:52-55 是 `Promise.all([fetchStudentProfile, fetchStudents])` 后串行 `await loadHealth()`；:48-57 的 loadData 只有 finally 没有 catch（档案无 error 分支）。
- 建议：改为「profile + students 并行，health 串行；档案无 catch，整页失败态为原型补充」。

### 5. [redundancy] 空态下「近 7 天」在区块头与晨检空文案各出现一次，同一时间口径重复。

- 证据：index.html:88 section-meta「近 7 天」；index.html:123「近 7 天暂无晨检记录」；styles.css:437-439 空态时两者同时可见（DOM 实测 empty 态 health-empty display:flex，section-meta 仍在）。
- 建议：空态时隐藏 section-meta，或把空态文案改为「暂无晨检记录」。

## 2026-10-27 P1 修复记录

- **P1-1 [design-core] 档案卡事实格错位** — 已修。移除 `.facts` 的 `max-width:100%`，让左右负边距出血对称；DOM 实测 `.facts` 14–376 与卡片外缘 14–376 对齐，第二列起点 195（卡片中线），`.identity` / `.health-note` 保持内容区 30–360。
- **P1-2 [completeness] 晨检缺 is_isolated 分支** — 已修。10-23 晨检行体温后补「已在隔离室等待家长」（`--warning` 13px/600，对应源码 `.health-isolated`）；notes.md 修正该字段的误列，并补入正常态描述与 token 映射。

验证：`od-check` → `problems: []`（scrollWidth 390 = clientWidth 390，bodyHeight 1048）；`od-shot` 重出 `preview.png`（390 × 1048）；`od-audit` 本页无 ✗。P2 5 项按批次要求未动。
