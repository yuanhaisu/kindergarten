# parent/05-notice-detail · 对抗性单页审查

- verdict：**WARN**（score 80/100）
- 源码覆盖：源码的通知卡（类型/时间/标题/作者/正文/附件/已读）、底部确认栏、加载与错误处理均已覆盖；缺 need_confirm=false 的信息型通知、is_read=false 两个条件分支，以及确认请求失败分支。

## P1 · 明显缺陷（2）

### 1. [completeness] 正常态无条件渲染「确认收到」栏与「已读」条；源码中两者分别是 need_confirm && !is_confirmed、is_read 的条件渲染，原型没有「无需确认的信息型通知」和「未读」变体，会把所有通知都设计成必须回执。

- 证据：D:/yuanhaisu/personal_projects/kindergarten/parent/src/pages/notice/detail.vue:85（v-if detail.need_confirm && !detail.is_confirmed）、:75-76（v-if detail.is_read）；D:/yuanhaisu/personal_projects/kindergarten/docs/design-previews/2026-10-25-full-redesign/parent/05-notice-detail/index.html:152-157 无条件渲染 confirm-bar，同目录 styles.css:579-583 只在 loading/error/empty 隐藏；index.html:161-166 的四个演示芯片无「无需确认/未读」；D:/yuanhaisu/personal_projects/kindergarten/server/internal/domain/notice.go:37 默认 need_confirm=false。
- 建议：增加 need_confirm=false 的信息型变体（或把「正常」拆成「需确认/仅告知」），已读条按 is_read 分支；notes 覆盖状态同步补充。

### 2. [completeness] 原型纸面的类型胶囊、发布人、附件区、已读/已确认时间共 5 处内容，在当前家长端通知详情实现里没有数据源；真机打开会是空标签、空作者名、无附件、『已读』后无时间。

- 证据：D:/yuanhaisu/personal_projects/kindergarten/server/internal/service/parent_portal_service.go:1190-1196 GetNotice 仅返回 id/category/title/content/need_confirm/status/published_at/created_at/is_read/is_confirmed，无 category_label/creator/attachments/read_at/confirmed_at；契约 D:/yuanhaisu/personal_projects/kindergarten/backend/api/api-parent/02-notice.md:46-75 列有这些字段；原型 D:/yuanhaisu/personal_projects/kindergarten/docs/design-previews/2026-10-25-full-redesign/parent/05-notice-detail/index.html:51-61、73-105 全部画实；附件查询 D:/yuanhaisu/personal_projects/kindergarten/server/internal/repository/notice_repo.go:223 存在但未被详情调用。
- 建议：后端按契约补返回（attachments 调 ListAttachmentsByNoticeID）；若本批次不修后端，notes 需把该页标注为依赖未实现字段。另：published_at/read_at 前端直接截 UTC 串（teacher_portal_service.go:473-477），与 A15 同类，落地要本地化格式化。

## P2 · 打磨项（4）

### 1. [logic] 确认收到的失败分支缺失：源码 PUT 失败会 toast「操作失败」并保持按钮可点，原型 0.8 秒后必定成功切到已回执，没有失败/重试路径。

- 证据：D:/yuanhaisu/personal_projects/kindergarten/parent/src/pages/notice/detail.vue:20-29 handleConfirm 有 try/catch 与 err.message || 操作失败；原型 D:/yuanhaisu/personal_projects/kindergarten/docs/design-previews/2026-10-25-full-redesign/parent/05-notice-detail/app.js:76-99 计时器回调无条件 document.body.classList.add('is-confirmed')；notes.md:27 的「禁用」只描述成功路径。
- 建议：演示层补一次失败分支（如第二次点击走失败 toast 并复位按钮），或至少在 notes 声明该分支未建模。

### 2. [redundancy] 已回执态把同一状态和同一时间显示两遍：顶部绿色横幅「已确认收到 10月26日 09:12」与纸面底部已读条「已读 10月26日 08:36 · 已确认 10月26日 09:12」并存，toast 是第三次。

- 证据：D:/yuanhaisu/personal_projects/kindergarten/docs/design-previews/2026-10-25-full-redesign/parent/05-notice-detail/index.html:45-49（横幅）、index.html:102-105（已读条）；同目录 styles.css:599-609 与 611-630 两套规则同时让 banner 与 read-confirmed 可见；notes.md:24 同时描述两者。
- 建议：二选一：保留横幅则已读条去掉「· 已确认 09:12」后缀（或删横幅），同一状态只留一处。

### 3. [notes-accuracy] notes 遗留 1 与契约不符：称 attachments 只有 { id, file_url }、需在契约补 file_name/file_size，但契约已定义 file_type/file_name/file_size；真正缺口是详情实现不返回 attachments，且附件说明本可直接用 file_name。

- 证据：D:/yuanhaisu/personal_projects/kindergarten/docs/design-previews/2026-10-25-full-redesign/parent/05-notice-detail/notes.md:61；D:/yuanhaisu/personal_projects/kindergarten/backend/api/api-parent/02-notice.md:59-66（file_type/file_name/file_size）；D:/yuanhaisu/personal_projects/kindergarten/server/internal/service/parent_portal_service.go:1190-1196 无 attachments 键。
- 建议：改写遗留 1：契约字段已存在，问题在实现未返回；附件 caption 可用 file_name。

### 4. [design-core] 两处语义色文字对比度低于 WCAG AA 4.5:1：类型胶囊 11px 橙字约 3.35:1；已回执绿色 14px 横幅与 12px 已读条文字约 3.00–3.10:1。

- 证据：D:/yuanhaisu/personal_projects/kindergarten/docs/design-previews/2026-10-25-full-redesign/parent/05-notice-detail/styles.css:48-61（--accent-deep on --accent-wash，11px）、273-285（--success on --success-wash，14px）、252-254 与 603-609/620-626（--success on --bg，12px）；同目录 tokens.css:22-23、28-29。按 WCAG 相对亮度公式实测 3.35/3.00/3.10。
- 建议：回 tokens.json 统一加深 --success 与类型胶囊文字色（或胶囊字改用 --fg），不要只改本页。

---

## 2026-10-27 P1 修复记录

1. **[completeness] 确认栏 / 已读条无条件渲染 → 已修。**
   把「正常」拆成三种真实分支并补演示芯片：新增「仅告知」（`data-state="info"`，`need_confirm=false`，换一张健康提醒纸面、无确认栏、无附件区，只保留已读条）与「未读」（`data-state="unread"`，`is_read=false`，隐藏已读条、保留确认栏）。`styles.css` 按状态控制 `.sheet-info` / `.confirm-bar` / `.read-block` 显隐（信息型纸面底部留白收 48px），`app.js` 在切到非「已回执」状态时清掉 `body.is-confirmed`，避免信息型纸面被染绿或冒出并不存在的确认时间。notes.md 的「信息层级」「覆盖状态」「演示芯片」同步更新。
2. **[completeness] 类型胶囊 / 发布人 / 附件 / 已读时间 / 已确认时间无数据源 → 暂缓（需改后端）。**
   契约 `backend/api/api-parent/02-notice.md` 已定义 `category_label` / `creator` / `attachments` / `read_at` / `confirmed_at`，缺口在 `server/internal/service/parent_portal_service.go:1190-1196` 的 `GetNotice` 未返回这些字段；补齐需改 `server/`，超出本页范围。设计保持契约目标形态，notes.md 新增「依赖未实现字段」一节标注该依赖与真机表现。

P2 四条（确认失败分支、横幅与已读条重复、notes 遗留 1 表述、语义色对比度）按批次要求未动。
