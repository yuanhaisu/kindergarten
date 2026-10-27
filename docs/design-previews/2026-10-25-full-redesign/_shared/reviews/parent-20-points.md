# parent/20-points · 对抗性单页审查

- verdict：**WARN**（score 72/100）
- 源码覆盖：源码的 hero（level_name/total_points/rank_in_class/class_total）、三段 tab、本周/本月、榜单行、已获得成就（名称+描述+计数）均已覆盖；成就 description 在网格默认态不可见（只在无提示的底部面板），另新增源码没有的积分流水/等级进度/四态（notes 已标注为差异或提案）。

## P1 · 明显缺陷（2）

### 1. [logic] 积分明细标称「共 128 条」，但点一次「加载更多」追加 2 条后即置灰为「已经到底了」，列表实际只有 9 条，总量与到底状态自相矛盾。

- 证据：parent/20-points/index.html:69 panel-meta「共 128 条」；parent/20-points/app.js:128-136 点击后 appendHistoryRows() 追加 2 条（app.js:138-169）就 loadMore.disabled=true 并把文案改成「已经到底了」；backend/api/api-parent/07-points.md 的 history 契约为 total=128、page/page_size 分页（示例 page_size=20）。
- 建议：按契约分页（每次 20 条，显示「已加载 9/128」并允许继续加载），或把总量改成实际 9 条并说明这只是最近记录，不要在 9 条时宣称到底。

### 2. [redundancy] 本周/本月统计卡被提到 tab 之外常驻，导致排行榜 tab 同屏出现两个「本周 45」；成就 tab 的卡头同屏出现「已获得 8 / 12」与「已解锁 8 枚」；且源码里本周/本月本来是积分 tab 的内容，现在积分 tab 只剩新增的流水，原始统计被抽出。

- 证据：parent/20-points/index.html:55-58 stat-grid 位于 tabs(60-64) 之上；:56「本周积分 45」与 :113「我的本周积分 45」同屏；:138「已获得 8 / 12」与 :139「已解锁 8 枚」同屏；:114「我的排名 3」与 :40「班级第 3 / 28」及 :124 已高亮的第 3 行重复；源码 parent/src/pages/points/index.vue:47-50 本周/本月仅在 tab==='overview' 时渲染。
- 建议：把本周/本月放回积分面板（或在排行榜/成就 tab 隐藏），panel-sub/panel-meta 二者只留一处并承载不同信息（如统计周期/口径），避免同一数字在卡头出现两次。

## P2 · 打磨项（5）

### 1. [notes-accuracy] notes 的「遗留」只列了等级阈值(A28)与流水(A29)，漏掉 overview 的 rank_in_class/class_total 根本没有返回、this_month 恒为 0、level 只会算出 1–3；原型却把「班级第 3 / 28」「本月 180」「Lv.5」当作真实数据展示。

- 证据：server/internal/service/parent_portal_service.go:2309-2332 返回 map 仅 total_points/level/level_name/this_week/this_month，其中 this_month 写死 int64(0)，level 上限 3（新手上路/活跃家长/积分达人）；backend/api/api-parent/07-points.md:15-27 契约含 rank_in_class/class_total；parent/20-points/index.html:39-40,57 显示 Lv.5 / 班级第 3 / 28 / 本月 180；notes.md:52-61 未提这些缺口。
- 建议：notes「遗留」补记 rank_in_class/class_total 未实现、this_month 恒 0、level 只有 1–3；后端按契约补齐这些字段，或原型把 hero 数字标注为示例假设。

### 2. [notes-accuracy] 排行榜接口返回扁平字段 student_name/total_points 且没有 is_self，源码却读 item.student?.name / item.points / item.is_self；真机上榜单姓名与分数会是空的、自己那行也不会高亮，原型画成正常榜单，notes 未记录这个缺口。

- 证据：server/internal/service/parent_portal_service.go:2360-2367 返回 student_name/total_points，无 is_self；parent/src/pages/points/index.vue:53-56 读 item.student?.name / item.points，:53 用 item.is_self；parent/20-points/index.html:117-131 正常显示姓名与分数、:123-124 高亮「乐乐 + 我」；notes.md:58-61 未提。
- 建议：notes「遗留」补一条榜单字段形状缺口；后端按契约返回 student{name}/points/is_self，或前端改读扁平字段。

### 3. [logic] 徽章底部面板声明 aria-modal，但打开时不锁背景滚动、不做焦点陷阱；Tab 仍能走到被遮罩的演示条与页面控件，背景也能继续滚动。

- 证据：parent/20-points/app.js:208-212 打开时只切换 hidden 并 focus 关闭按钮；app.js:214-233 无 body 滚动锁与 focus trap；parent/20-points/index.html:223 标注 aria-modal="true"；styles.css:715-741 backdrop/sheet 为 fixed，下方 7 条流水页面本身可滚动。
- 建议：打开时给 html/body 加 overflow:hidden（或给 .mp-main 加 inert/aria-hidden），把焦点限制在 sheet 内，关闭时恢复滚动与焦点。

### 4. [completeness] 排行榜页脚写「仅显示本班前 20 名（共 28 人）」，但只渲染 5 行且没有继续加载入口；同一页的积分明细有「加载更多」，榜单没有。

- 证据：parent/20-points/index.html:133 panel-foot；:117-131 只有 5 个 rank-row；:108 的 loadMore 只在 panelPoints 内，panelRank 无任何展开控件。
- 建议：要么补满 20 行/提供加载更多，要么把页脚改成「仅显示前 5 名（共 28 人）」。

### 5. [completeness] 源码在每张成就卡上直接渲染 description，原型网格只给图标+名称，描述/获得条件/获得日期全藏在无任何提示的点击面板里；4 个未获得徽章连「怎么获得」都看不到。

- 证据：parent/src/pages/points/index.vue:62-65 渲染 a.name 与 a.description；parent/20-points/index.html:142-189 徽章按钮只有 badge-icon+badge-name，无「详情」/chevron 类提示；app.js:224-226 点击才 openSheet，内容见 app.js:188-212。
- 建议：在徽章格上补可见的详情入口（chevron/「详情」），或把 description 作为第二行小字展示，至少让锁定徽章显示获得条件。

## 2026-10-27 P1 修复记录

| P1 | 状态 | 改动 |
|---|---|---|
| 1 积分明细「共 128 条」与实际 9 条到底矛盾 | 已修 | 卡头改为「最近 9 条」（7 条首屏 + 加载更多补 2 条 = 9 条），加载满 9 条后才置灰「已经到底了」；契约的 `total:128` 分页已记入 notes「遗留」，落地时按接口 `total` 分页 |
| 2 本周/本月常驻卡与各 tab 卡头数字重复 | 已修 | `.stat-grid` 移回「积分」面板顶部（与源码一致，切 tab 即隐藏）；排行榜卡头删「我的排名 3」并把 sub 改为「按本周获得积分排序」；成就卡头删「已解锁 8 枚」，只留「已获得 8 / 12」 |

**复验**：`od-check --width 390` → `problems: []`；DOM 实测：排行榜 tab 统计块不可见、成就卡头仅一句计数、加载更多后 7→9 条且按钮置灰；`od-audit` 本页无 ✗；`preview.png` 已重出（390×1172 @2x）。
