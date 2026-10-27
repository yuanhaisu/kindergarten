# parent/22-calendar-index · 对抗性单页审查

> 2026-10-27 已修复本页 P0（视力筛查地点改为医务室），修复与验证见 `_shared/REVIEW.md` 2.1。

- verdict：**FAIL**（score 76/100）
- 源码覆盖：源码的变更提示（最多 3 条 +「另有 N 条」/收起）与活动卡（标题、类型、时间、地点、两行描述、加载/空分支）都已覆盖，另有 notes 声明的月历、按日清单、骨架、错误重试、整月空态与「回到今天」；偏离或缺失的是服务端 event.color 未使用（改为固定 5 色，见 A32）、卡片内 event_date 上移到区块头、源码与 notes 都没有的卡内变更行，且月历依赖前端补 start/end（A30）。

## P0 · 用户可见硬伤（1）

### 1. [logic] 视力筛查卡片的当前地点仍是「保健室」，与同屏变更说明「地点 保健室 → 医务室」直接矛盾，家长会按旧地点去保健室。

- 证据：app.js:152-154 该事件同时写了 location: '保健室' 与 change_summary: '地点 保健室 → 医务室'；运行时选中 2026-10-27 后卡片 .event-loc 显示「保健室」，卡内 .event-change 显示「安排调整 地点 保健室 → 医务室」，顶部变更条也是同一句，矛盾同屏可见。
- 建议：把 vision-screening 的 location 改为「医务室」，让当前值与变更结果一致；_prompt.md 的示例数据同样是「保健室 + 保健室→医务室」，若按提示词重生成需一并修正。

## P1 · 明显缺陷（2）

### 1. [logic] 「今天」胶囊只在当前月禁用，选中当前月的其他日期后仍然禁用，无法一键回到今天。

- 证据：app.js:305 仅用 isCurrent 决定 todayBtn.disabled，而 selectDate()（app.js:539-548）不调用 renderToolbar()；运行时点选 2026-10-02 后 #todayBtn.disabled 仍为 true，#eventsDate 已变成「10月2日 周五」。
- 建议：禁用条件改为 isCurrent && selected === TODAY_ISO，或在 selectDate() 末尾重算工具栏，使当前月选中非今天日期时胶囊可用。

### 2. [redundancy] 每张活动卡把类型名渲染两次（标题行胶囊 + 元信息行「类型 · 时间」），同一屏重复。

- 证据：app.js:444 的 .type-pill 与 app.js:446 的 .meta-type 使用同一个 label；运行时每张卡同时存在「活动」胶囊与「活动 · 09:00–10:30」。老师端同模块 teacher/11-calendar-index/app.js:179-185 只在 meta 渲染一次类型。
- 建议：删除标题行的 .type-pill（保留图标 + meta），或删除 meta 里的 .meta-type 只留胶囊；与老师端卡片口径统一。

## P2 · 打磨项（4）

### 1. [redundancy] 同一条变更在顶部变更卡与当日活动卡内各写一遍，首屏两处重复同一「旧值 → 新值」。

- 证据：初始屏变更条「秋季运动会彩排 10月26日 · 时间 10:00 → 10:30」与 10-26 卡内 app.js:431-433 的「安排调整 时间 10:00 → 10:30」同文，「班级家长会」同理；源码 shared-ui/CalendarEventCard.vue 不渲染 change_summary，notes.md 也未声明卡内变更行。
- 建议：二选一：卡内不再重复全文（顶部变更卡承担），或变更卡只做索引、卡内保留明细；若保留两处，卡内改成「时间已调整」等不重复数字的措辞。

### 2. [logic] 演示状态按「空态 → 加载 → 正常」切换后会停在 2026年12月 的空月，而不是回到默认 10 月。

- 证据：app.js:513-519 的 applyMode() 只在 previous === 'empty' 时复位今天；实测依次点「空态、加载、正常」后 #monthTitle 为「2026年12月」并显示「12月暂无活动安排」，aria-pressed 停在「正常」。
- 建议：进入 normal 时若视图不在今天所在月就复位 view/selected（不依赖 previous），或进入 demo 前记录视图、退出时恢复。

### 3. [logic] 翻到非当前月后副标题仍写「本月 N 个活动」，与「2026年11月」的标题口径不一致。

- 证据：app.js:303 无论 isCurrent 与否都拼「本月」；运行时点 #nextMonth 后 #monthTitle 为「2026年11月」、#monthSub 为「本月 2 个活动」。
- 建议：按 isCurrent 分支：当前月用「本月 N 个活动」，其他月用「N 个活动」或「11月 2 个活动」。

### 4. [notes-accuracy] notes 声称错误态「只留错误面板」，实际月份条仍然显示。

- 证据：notes.md:11「月历与当日区一起收起，只留错误面板」；styles.css:1106-1111 的 error 规则只隐藏 .changes-card/.calendar-card/.calendar-skeleton/.events-section，未隐藏 .month-bar。运行时切「失败」后 #monthTitle、#monthSub、翻月按钮均可见，#monthSub 为「本月活动未加载」。
- 建议：要么在 error 态一并隐藏 .month-bar 与 notes 对齐，要么把 notes 改写为「保留月份条并显示未加载提示」。

## 2026-10-27 P1 修复记录

- **P1-1 [logic] 已修**：「今天」pill 的禁用条件由「当前月」改为「已在今天」（`isCurrent && selected === TODAY_ISO`），并在 `selectDate()` 后重算工具栏（app.js renderToolbar / selectDate）。实测选中 10-02 后 pill 可用并回到 10-26，翻到 9 月同样可一键回今天；停在今天时仍置灰。
- **P1-2 [redundancy] 已修**：删除活动卡标题行的 `.type-pill` 及其样式，类型名只在元信息行渲染一次（「活动 · 09:00–10:30」），与老师端日历卡口径一致；`aria-describedby` 仍包含元信息行，类型信息未丢失。
- 同步：notes.md 更新「今天」pill 禁用口径、翻月回今天的描述，并补充卡片类型只展示一次的说明。
- 验证：`od-check --width 390` → `problems: []`；`od-shot` 已重出 `preview.png`（390 / dsf 2）；`od-audit` 本页无 ✗（仅批次通用的「超 20KB」体积提示）；交互脚本实测无 console 错误。
- P2 四项保持未动（变更重复、演示态复位、跨月副标题口径、error 态月份条与 notes 不符）。
