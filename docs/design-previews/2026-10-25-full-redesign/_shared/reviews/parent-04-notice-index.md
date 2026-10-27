# parent/04-notice-index · 对抗性单页审查

- verdict：**PASS**（score 86/100）
- 源码覆盖：源码的统计条、4 个分类 tab、卡片五要素（标题/未读点/两行预览/发布人/日期/箭头）与加载、空、失败、禁用四态均已覆盖，卡片跳转意图指向 notice/detail（05 存在）、子页无 tabBar 也正确；缺口在实现侧未返回卡片依赖的 content_preview/creator/attachments_count，以及空态下筛选点击没有实际结果。

## P2 · 打磨项（4）

### 1. [completeness] 列表卡片的「两行预览」和「发布人」在真机上会是空白：原型与源码都依赖 content_preview / creator，契约也承诺 attachments_count，但 ParentPortalService.ListNotices 只返回全文 content 和原始英文 category_label，三个字段一个都没有（A1–A42 未收录）。

- 证据：server/internal/service/parent_portal_service.go:1161-1168 的 map 只有 id/category/category_label/title/content/need_confirm/status/published_at/created_at/is_read/is_confirmed；backend/api/api-parent/02-notice.md:25,29-30 明确返回 content_preview、attachments_count、creator；parent/src/pages/notice/index.vue:123 读 item.content_preview、:129 读 item.creator?.nickname（缺省空串）；原型 index.html:112-113 同结构，落地后每张卡第二行与「类型 · 发布人 · 时间」中的发布人会空掉。
- 建议：后端按契约补齐 content_preview（如 content 截 80 字）、creator{nickname,avatar}、attachments_count；短期前端兜底 item.content_preview || item.content.slice(0,80)，creator 缺失时显示「园所通知」而非空串。同时 notes.md:27「接口返回 attachments_count」只对契约成立，应像 line 29 一样注明实现未返回。

### 2. [design-core] 选中筛选胶囊与「未读」统计数字的橙白组合对比度只有 2.80:1，低于 WCAG AA（13px 需 4.5:1，20px 大字也需 3:1），品牌色压在文字上不可读性偏高。

- 证据：styles.css:160-165 选中胶囊 background: var(--accent)、color: var(--on-brand)，字号 var(--text-base) 13px/600；styles.css:94-96 未读数字 color: var(--accent)、20px/600 压 var(--surface)；tokens.css:18,20,22 为 #FFFFFF / #F97316 / #EA580C。实测计算：白压 #F97316 = 2.80:1，#F97316 压白 = 2.80:1，hover 的 #EA580C 白字也只有 3.54:1。批次 token 缺口 #2/#3 只覆盖 --muted/--hint，未覆盖 accent-on-brand 这一对。
- 建议：回 shared-design-tokens/tokens.json 增加一个「承载文字的品牌深色」（如 accentInk #C2410C：压白 5.2:1、压 #FFF7ED 4.9:1），选中胶囊改为该底色+白字或 wash 底+深字，未读数字改用该色；--accent 只留给小圆点、图标等非文字元素。三端统一改 token，不要逐页打补丁。

### 3. [logic] 空态下 4 个筛选胶囊保持可点，但点击只移动高亮，列表、统计和 #filter-empty 都不变，属于「看起来能点但没反应」；加载/失败态都禁用了筛选，空态却没有一致处理。

- 证据：app.js:95-100 只在 loading/error 调 setFilterAvailability(false)，empty 走 else 被启用；app.js:86-89 selectCategory 仅在 currentState()==='normal' 时 refreshNormalView()；app.js:74-76 filterEmpty 只在 refreshNormalView 内切换。实测空态点「活动」：仅 aria-pressed 与高亮变化，统计仍 0/0/0，仍显示「暂无通知」，#filter-empty 始终 hidden。
- 建议：空态同样 setFilterAvailability(false)（与 loading/error 对齐），或在空态选中分类时显示 #filter-empty「该分类暂无通知 + 查看全部」，保证每次点击都有明确结果。

### 4. [notes-accuracy] notes.md:23 断言统计数字在切换分类时与源码「保持一致语义」，实际只有「全部/任务/活动」一致；「通知」tab 源码 total 取接口全量 8，原型重算为可见 4 条，结论句给出了确定性的错误信息（line 28 虽已把口径列为待确认，但前后矛盾）。

- 证据：notes.md:23 与 notes.md:28；app.js:45-49 setStats(list.length, unread, today)；parent/src/pages/notice/index.vue:19-22 的 total 来自 total.value，:42-57 仅在 other tab 前端过滤后 total 仍取 res.data.total；server/internal/service/parent_portal_service.go:1136-1143 的 Count 随 category 过滤（故任务/活动与原型一致）。实测切「通知」：原型统计 4/0/0，源码同场景 total=8。
- 建议：把 line 23 改为「除『通知』tab 外与源码一致；『通知』tab 源码 total 为全量，原型按可见列表重算（见待确认项）」，并把 line 28 的待确认结论前置，避免读者误以为全部分类都已对齐。
