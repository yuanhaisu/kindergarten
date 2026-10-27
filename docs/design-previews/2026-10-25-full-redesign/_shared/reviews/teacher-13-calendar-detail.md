# teacher/13-calendar-detail · 对抗性单页审查

- verdict：**WARN**（score 82/100）
- 源码覆盖：源码的头部（类型/状态/标题/分段说明/全园只读提示）、信息卡、家长预览弹层、编辑跳转与删除确认分支均已覆盖，加载态还升级为骨架屏；缺口是源码「加载失败仅 toast、页面仍渲染空壳」的真实降级形态未呈现（改为整页错误块），信息卡另多了源码没有的「参与范围」行。

## P1 · 明显缺陷（1）

### 1. [notes-accuracy] notes.md 多处声明与四件套实际不符：声称信息卡含「参与班级与人数摘要、提醒设置」（实际是 日期/时间/地点/参与范围，页面无人数、无提醒）；声称「编辑（底部弹层）」（实际编辑只弹 toast，页面无编辑弹层）；状态覆盖列了不存在的「已取消/已删除」降级态，却漏掉了实际存在且是核心差异的「草稿」「全园只读」两态。

- 证据：notes.md:6、notes.md:11、notes.md:14、notes.md:23、notes.md:40；index.html:87-124 仅有 4 个 .info-row（活动日期/活动时间/活动地点/参与范围），全页无「人数/提醒」文本（grep 无命中）；app.js:288 编辑按钮只 toast('打开编辑活动')；index.html:193-198 的 6 个 chip 为 正常/空态/加载/失败/草稿/全园只读，app.js:100 DATA_STATES 中无 cancelled/deleted。
- 建议：按实际实现重写 notes.md：信息卡字段改为 日期/时间/地点/参与范围；编辑写明「toast 占位，真实实现跳 pages/calendar/create?id=」；状态列表补「草稿」「全园只读」，删除「已取消/已删除」（源码同样没有该状态）。

## P2 · 打磨项（5）

### 1. [redundancy] 草稿态把「家长看不到」说了三遍且措辞各不相同：状态标签「草稿（家长不可见）」+ 标题下黄色提示条「草稿 · 家长端暂时看不到这个活动，发布后才可见。」+ 预览弹层「草稿 · 家长暂时看不到」。源码只有徽标与弹层两处，黄色提示条是新增的第三份。

- 证据：index.html:47-50 与 app.js:130（徽标「草稿（家长不可见）」）；index.html:55-60（draft-note 黄条）；index.html:213 与 app.js:154（sheetDraft）；源码 detail.vue:86-89、135-137 仅两处。
- 建议：保留徽标与弹层提示，删除标题下黄条；若产品确要黄条，则把徽标文案缩为「草稿」，只留一处解释，避免同一卡内上下紧邻两次「家长不可见」。

### 2. [redundancy] 信息卡新增源码没有的第 4 行「参与范围」；在「全园只读」态该行值「全园」与上方提示条「全园事件（由园长 / 保健医发布），老师端只读」重复表达同一事实，且 notes.md 未把它记为设计提案。

- 证据：index.html:115-123（参与范围行）、app.js:150（scope→本班/全园）、index.html:78-83（全园提示条）；源码 detail.vue:98-114 只有 活动日期/活动时间/活动地点 三行。
- 建议：二选一：删掉「参与范围」行，scope 只留在全园提示条；或保留该行并把全园提示条收缩为「仅园长/保健医发布，老师端只读」，同时在 notes.md 标注这是超出源码的展示项。

### 3. [logic] 加载态整页只剩 aria-hidden="true" 的骨架屏，没有可读文案、也没有 aria-busy；源码本来是有「加载中...」文字的，屏幕阅读器在加载期间得到完全空白。

- 证据：index.html:139（#skeleton aria-hidden="true"）；styles.css:738-739（loading 时 .detail display:none、.skeleton display:flex）；源码 detail.vue:78（text「加载中...」）。
- 建议：保留骨架屏视觉但增加 role="status" 的视觉隐藏「加载中」文本（或去掉骨架的 aria-hidden），并在加载期间给 .mp-app 标 aria-busy="true"。

### 4. [logic] 骨架屏信息区只有 3 行占位，实态信息卡是 4 行，加载完成时卡片高度跳变约一行；骨架与最终布局不一致。

- 证据：DOM 实测 #skeleton .sk-row = 3、.info-row = 4；index.html:153-175（3 条 sk-row）对比 index.html:87-124（4 条 info-row）；styles.css:397-405。
- 建议：骨架屏补第 4 条 sk-row（sk-icon + 两条 sk-line），使 sk-info 高度与实态一致。

### 5. [redundancy] 同一个活动日期在一页里出现两种口径：信息卡「2026年10月26日 周一」，家长预览卡「2026-10-26」；源码详情页与共享卡片都直接用 event_date 原值，中文格式是原型自己加的，两处不统一。

- 证据：index.html:94 与 app.js:103-107（formatDate → 2026年10月26日 周一）；index.html:219 与 app.js:159（parentMeta 用 event.date 原值 2026-10-26）；源码 detail.vue:101 与 CalendarEventCard.vue:55 均为 event_date 原值。
- 建议：预览卡必须保留原值以忠实反映 CalendarEventCard；建议信息卡也改回 event_date 原值（与源码一致），若坚持中文写法则在 notes.md 说明两处格式差异的来源。

## 2026-10-27 P1 修复记录

- P1-1 [notes-accuracy] notes.md 多处声明与四件套实际不符 —— **已修**：按实际实现重写 `notes.md`，未改 index.html / styles.css / app.js。
  - 信息卡字段改为「活动日期 / 活动时间（含是否全天）/ 活动地点 / 参与范围（本班 / 全园）」，删掉不存在的「参与班级与人数摘要」「提醒设置」（index.html / app.js 均无「人数 / 提醒」文本）。
  - 「方向」与「信息层级」中的「编辑（底部弹层）」改为「toast『打开编辑活动』占位（源码 `goEdit()` 跳 `pages/calendar/create?id=`）」；删除仍为居中确认框，表述保留。
  - 状态覆盖删掉不存在的「已取消 / 已删除」，补上实际存在的「草稿」（状态标签「草稿（家长不可见）」+ 标题下黄条 + 预览弹层提示，`scope` 为本班时编辑 / 删除仍可用）与「全园只读」（全园提示条 + 参与范围显示「全园」，`scope === "kindergarten"` 时底部操作区隐藏），并写清空 / 加载 / 失败三态的实际呈现。
  - 同源错述一并校正：删掉并不存在的第 6 层「元信息：创建人 / 创建时间」与遗留项「参与人数为示例数据」（接口 `GET /teacher/calendar/events/detail/:id` 不返回这些字段，页面亦无对应节点）；「与现状的差异」原称「新增预览弹层」「删除确认从 toast 改为居中对话框」均不成立（源码 `detail.vue:129-140` 已有预览弹层、`detail.vue:38-53` 已用 `uni.showModal` 确认框），改写为加载骨架 / 失败错误块 / 空说明块 / 信息卡第 4 行四条真实差异。
  - `styles.css 20KB`、token 映射两节经核对无误，保持原文。

P2 五项未动（草稿提示三重复、参与范围行冗余、加载态 aria、骨架 3 vs 4 行、日期两种口径全部保留原状）。

验证：本次为纯 notes.md 修订，页面四件套无改动；`node scripts/od-check.mjs .../13-calendar-detail/index.html --width 390` → `problems: []`（scrollWidth = clientWidth = 390）；`node scripts/od-audit.mjs docs/design-previews/2026-10-25-full-redesign` → 本页 `issues: []`、`warnings: []`，批次「结构全部通过（体积提示不计失败）」；`preview.png` 已按 390 宽 / dsf 2 重出（149,823 bytes）。
