# teacher/06-growth-detail · 对抗性单页审查

> 2026-10-27 已修复本页 P0（AI 横幅计数改为动态），修复与验证见 `_shared/REVIEW.md` 2.1。

- verdict：**FAIL**（score 63/100）
- 源码覆盖：正常/加载/错误/空评论/单张照片失败五态齐备，作者行、标题正文、九宫格、相关孩子、统计与编辑/分享/授徽章、评论列表与底部输入均已覆盖；缺逐张照片内联的 AI 建议（被收拢成单卡且确认后不回写点名）、授徽章应按全班学生选择（原型只有 3 个相关孩子）、作者时间缺年份。

## P0 · 用户可见硬伤（1）

### 1. [logic] AI 横幅声称「3 条高置信建议」，页内却只有 1 条高置信行；点「全部确认」实际只确认 1 条，且横幅在仍有待确认行时继续显示「3 条」，数字与动作结果自相矛盾。

- 证据：index.html:65 `<strong>3</strong> 条高置信建议`；#aiList 内仅 index.html:132 一条 `.status--ok`（乐乐），index.html:143 是 `.status--pending`（朵朵）；app.js:114 只筛选 `.status--ok`，app.js:121 弹出「已确认 1 条高置信建议」；app.js:90 仅在 rows.length===0 时隐藏横幅。
- 建议：横幅条数改为按 `.status--ok` 行实时计算，并在每次确认/拒绝后同步；或补齐 3 条高置信行，使「全部确认」的条数与 toast 结果一致。

## P1 · 明显缺陷（2）

### 1. [redundancy] 同一「照片张数」在页内出现两个口径：照片区写 9 张，打开任意照片预览却显示「共 8 张」。

- 证据：index.html:61「9 张」、index.html:69 aria-label「共 9 张」、index.html:379 默认「共 9 张」；app.js:26 的 `$$('.photo-open')` 只有 8 个（第 9 格是 index.html:107 的 `.photo-fail`），app.js:31 `const total = tiles.length`，打开照片后预览显示「第 N 张 / 共 8 张」。
- 建议：total 取完整媒体数（含加载失败位）而不是可预览按钮数；或把失败位也纳入 tiles。

### 2. [logic] 切到「空评论」态后，评论卡标题仍写「（5条）」、互动条仍写「5 条评论」，同时显示「还没有家长评论」，同一状态出现两个互斥口径。

- 证据：styles.css:986-987 只隐藏 `.comment-list` 并显示 `.comment-empty`；index.html:215「家长评论（5条）」与 index.html:167「5 条评论」不随状态变化；app.js:224-230 的 `setState` 只切 data-state 与 chip；app.js:232-235 仍按 DOM 中 5 个隐藏 `.comment` 计数。
- 建议：进入 empty 时把 #commentCount 与 #statCommentCount 置 0（或空态真实清空列表），并保证发布首条后计数从 1 起算。

## P2 · 打磨项（5）

### 1. [logic] 预览弹层拿不到被点照片的渐变：按钮类为 `photo-open ph ph--1`，`classList[1]` 恒为 `ph`，`|| 'ph--1'` 兜底是死代码，所有照片预览都落成同一个默认底，与九宫格里各不相同的色块不一致。

- 证据：app.js:33 `previewStage.className = 'preview-stage ph ' + (tile.classList[1] || 'ph--1')`；index.html:71 等按钮类依次为 `photo-open`(0)、`ph`(1)、`ph--1`(2)，因此 ph--N 永远不生效；styles.css:227-234 只定义了 `.ph--1`~`.ph--8`，`.ph` 自身无背景。
- 建议：改为按前缀取类（如 `Array.from(tile.classList).find(c => /^ph--\d+$/.test(c))`）再赋给预览层，删掉不可达的兜底。

### 2. [completeness] 授徽章「授予给」只提供 3 个相关孩子，源码的候选是全班学生；面板却同时写「默认授予本班全部 18 人」，名单与口径对不上，老师无法单独授予其他班级成员。

- 证据：detail.vue:21-28 用 `fetchTeacherClassStudents()` 载入全班 `students`，detail.vue:482-491 全量渲染为可选；index.html:195-199 只有乐乐/小雨/朵朵三个 `data-kid`；index.html:200 文案「未选择孩子，将默认授予本班全部 18 人」。
- 建议：候选改为全班学生名单；若只想展示相关孩子，需在 notes 标明是示例子集并说明真实实现取全班。

### 3. [completeness] 源码按每张照片内联的 AI 识别建议被收拢成一个「第 6 张」卡片，其它照片上的建议无处呈现；点「確認」只删除行，不会把该孩子补进对应照片的点名标签。

- 证据：detail.vue:398-410 在每个 `photo-card` 内渲染 `photo-sugs`（确认后 detail.vue:161-165 写入 `student_ids`，378-381 渲染点名）；index.html:122-152 只有一个 `ai-card` 标「第 6 张」；app.js:93-99 `handleSuggestion` 只 `row.remove()`，照片 6（index.html:95-99）始终没有点名胶囊。
- 建议：建议按照片挂载或在卡内按张分组；确认后同步给对应照片补「点名」标签，保持源码的确认语义。

### 4. [notes-accuracy] notes 把「AI 识别建议从跳转确认改为行内确认/拒绝」列为与现状的差异，但源码本来就是逐张照片内的行内确认/拒绝；notes 也没披露原型反而把建议收拢成了单卡。

- 证据：notes.md:38「AI 识别建议从跳转确认改为行内确认 / 拒绝，减少一次页面往返」；detail.vue:398-410 已存在行内的「确认」「拒绝」按钮；index.html:122-152 只有一个汇总 `ai-card`。
- 建议：改正差异描述为「建议从逐张内联改为单卡汇总」，并标注该改动是待产品确认的设计提案。

### 5. [logic] 授徽章面板默认预选「合作之星」且点击无法取消，导致「请先选一枚徽章」校验永远不可达；源码默认不选、可反选，未选时不展示「授予给」区域。

- 证据：index.html:189 `class="badge-choice is-selected" aria-pressed="true"`；app.js:153-161 每次点击都保证恰好一枚选中；app.js:187-192 的 `if (!badge)` 分支因此永远不触发；源码 detail.vue:50-52 支持再点取消，detail.vue:480 仅在 `grant.achievement_id` 存在时展示授予对象。
- 建议：初始不选中任何徽章，允许再次点击取消选择，并保留「请先选一枚徽章」的阻断分支。

## 2026-10-27 P1 修复记录

> 仅处理 P1 两条，P2 一律未动。

### P1-1 [redundancy] 照片张数两个口径 — 已修

- app.js 新增 `mediaTotal()`，预览张数按 `.photo-cell` 照片位（含加载失败的第 9 格）计算，不再取可预览按钮数 `tiles.length`。实测第 3 / 8 张与重试恢复后的第 9 张均显示「第 N 张 / 共 9 张」，与「9 张」「共 9 张」一致。

### P1-2 [logic] 空评论态与计数互斥 — 已修

- 进入 `data-state="empty"` 时把 5 条演示评论移出 `#commentList`（切回其它状态原样放回），`#commentCount` 与 `#statCommentCount` 随 DOM 归零为「（0条）/ 0」；空态发布首条评论后演示评论不再放回，计数从 1 起算。`setState` 与 shell.js 广播的 `mp:state` 都会触发同步。

验证：`od-check --width 390` problems 为 `[]`；`od-audit` 本页无 ✗（仅既有 25KB 体积提示）；preview.png 已按 390×2 重出。CDP 实测：正常态 5/5；空态 0/0 且评论列表清空；空态发首条后 1/1；空态→正常态往返恢复 5/5。
