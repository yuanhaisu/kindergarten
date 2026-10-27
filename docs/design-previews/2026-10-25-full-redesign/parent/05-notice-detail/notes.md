# 通知详情（parent）

- 源码：`parent/src/pages/notice/detail.vue`（`onLoad` 取 `id` → `GET /parent/notices/:id`，成功后静默 `PUT /parent/notices/:id/read`）
- 设计系统：`user:nurture-planet-local`，surface=parent
- 生成方式：OpenDesign（首跑成功，四件套齐全，未走手写兜底）
- 方向：**「一页正式通知，不是一张信息卡」**——白纸、留白、字距舒展，读起来像贴在班级门口的那张通知单；品牌橙只出现在类型标签、头像底色和底部那一个主按钮上。

## 信息层级

1. 微信外壳：20px 状态栏 + 44px 导航栏（返回箭头，标题「通知详情」，右侧胶囊）
2. 纸面顶行：`category_label` 类型胶囊 + `published_at`（右对齐，等宽数字）
3. 标题 `title`（20px/600，`max-width: 22em`，自然折行）
4. 发布人行：32px 圆形头像（姓名首字「李」）+ `creator.nickname`
5. 细分隔线 → 正文 5 段（15px / 行高 1.85 / 段间 14px）
6. 附件（`attachments.length > 0` 时才出现）：小节标签「附件」+「2 张」+ 2 个 96px 方形占位块（图片图标 + 「图片 1 / 图片 2」）
7. 已读信息块（`is_read = true` 时才出现）：`var(--bg)` 圆角条 + 对勾 +「已读 10月26日 08:36」
8. 底部固定确认栏（`need_confirm = true` 且未回执时才出现）：整宽 48px「确认收到」主按钮（`--accent-gradient`）

## 覆盖状态

| 状态 | 表现 |
| --- | --- |
| 正常（默认） | `need_confirm=true`、`is_read=true`、未确认：完整通知 + 已读信息块（只有「已读」，无确认时间）+ 底部「确认收到」栏；暖色只有类型胶囊、头像底、主按钮 |
| 仅告知（演示芯片「仅告知」，`data-state="info"`） | `need_confirm=false` 的信息型通知：换成另一张纸面（健康提醒 / 王老师 / 无附件），**不渲染底部确认栏**，正文下方只留「已读 10月27日 08:12」；页面底部留白收到 48px |
| 未读（演示芯片「未读」，`data-state="unread"`） | `is_read=false`：已读信息块整块不出现，底部「确认收到」栏照常；在未读态点确认后，已读条随回执一起出现（`body.is-confirmed`） |
| 已回执（演示芯片「已回执」，`data-state="empty"`） | 纸面顶部出现 `--success-wash` 横幅「✓ 已确认收到 · 10月26日 09:12」；已读信息块转 `--success` 并追加「· 已确认 10月26日 09:12」；底部确认栏隐藏；正文底部留白收到 48px |
| 加载骨架 | 纸面与确认栏隐藏，换纸面骨架：标签条 + 时间条 + 标题两行 + 头像圆/姓名条 + 4 行正文 + 2 个 96px 附件方块；`--border` 呼吸动画，`prefers-reduced-motion` 下停止动画 |
| 加载失败 | 纸面、骨架、确认栏全隐藏；错误卡在首屏垂直居中（`--danger-wash` 圆底 + `--danger` 感叹号 +「通知加载失败」+「网络异常或这条通知已被撤回」+ 描边「重试」） |
| 禁用 | ① 点「确认收到」后按钮立即 `disabled`、文案「确认中…」、`opacity:.6`、`pointer-events:none`，0.8s 后切到已回执并 toast「已确认收到」；② 加载 / 失败 / 仅告知态不渲染确认栏；加载/失败态也不渲染附件 |

演示芯片：正常 / 仅告知 / 未读 / 已回执 / 加载 / 失败（`data-mp-state` = normal / info / unread / empty / loading / error）。本页没有列表空态，因此把「空态」映射为「已回执」，不另造第七个状态（与批次内登录页的处理一致）。

## token 映射

| 元素 | Token |
| --- | --- |
| 页面底 / 纸面 | `--bg` / `--surface` |
| 类型胶囊 | `--accent-wash` 底 + `--accent-deep` 字 + `--radius-pill` |
| 发布时间 / 附件计数 / 占位说明 | `--muted`（`--hint` 在白底仅约 2.7:1，按 `_shared/findings.md` 的口径不用作文字色） |
| 标题 / 正文 | `--fg` / `--fg-2`（正文 15px = `--text-lg`，行高 1.85） |
| 头像 | 32px `--accent-wash` + `--accent-deep` 首字 |
| 分隔线 | 1px `--border` |
| 附件方块 | `--bg` 底 + 1px `--border` 描边 + `--radius-xl` + `--hint` 图标 |
| 已读条 | `--bg` 底 + `--radius-xl`；已确认时文字/图标转 `--success` |
| 已回执横幅 | `--success-wash` 底 + `--success` 字 |
| 主按钮 | `--accent-gradient` + `--on-brand` + `--radius-pill` + `--shadow-md` |
| 确认栏 / 纸面 | `--surface` + `--shadow-lg` / `--shadow-sm` |
| 错误态 | `--danger-wash` + `--danger`；重试按钮 `--surface` 底 + `--border` 描边 |
| 骨架 | `--border` + `--bg`；`--motion-fast` / `--ease` |

## 与现状的差异

- **从「卡片堆」回归「一张纸」**：源码是「内容卡片 + 底部固定按钮」，原型把标题、发布人、正文、附件、已读状态收进同一张纸面，用一条细分隔线代替多张卡片的层层嵌套，减少家长读长文时的视觉噪音。
- **正文行宽与分段受控**：源码 `white-space: pre-wrap` 一段到底；原型把正文拆成 5 个 `<p>`（称呼 / 时间地点 / 当天安排 / 陪同要求 / 雨天顺延），段间 14px，行高 1.85，`max-width: 34em`，长英文用 `overflow-wrap: anywhere` 兜底，不出现一条贯穿全屏的密排文字。
- **附件从 100px 裸图改为占位块 + 说明**：源码直接 `image` + `aspectFill`；原型用 96px 方块 + 图片图标 + 「图片 N」，明确「这里是图片，还没加载」。
- **已读/已确认合并为一条信息带**：源码 `read-info` 是一行纯文本；原型加对勾图标与 `--bg` 底板，已确认时整块转绿，扫一眼能看出回执状态。
- **确认栏上移为浮层**：源码 `position: fixed` 白底无边；原型加 1px 顶边、`--radius-xl` 上圆角与 `--shadow-lg`，与纸面留出层次，按钮高 48px（源码 88rpx = 44px，略放大以稳触控）。
- **新增失败态**：源码只有「加载中…」一行文字，没有错误分支；原型按共享约束补错误卡与「重试」，并在首屏垂直居中。
- **新增「已回执」横幅**：源码确认后只是按钮消失、`read-info` 多一段文字；原型在纸面顶部加一条 `--success-wash` 横幅，让「已确认」这个状态在首屏可见（属设计补充，见遗留 3）。

## 遗留

1. **附件没有文件名/大小字段**：`attachments[]` 只有 `{ id, file_url }`，所以占位说明只能写「图片 1 / 图片 2」，无法显示真实文件名或「PNG · 1.2MB」。若产品要展示，需在 `backend/api/api-parent`（通知详情）补 `file_name` / `file_size` / `mime_type`。
2. **「2 张」计数是派生值**：由 `attachments.length` 得到，不是接口字段；若后续要为附件增加非图片类型（如 PDF），这个计数文案要跟着改。
3. **已回执横幅属设计补充，需产品确认**：源码没有这块 UI，确认成功后仅按钮消失 + `read-info` 追加「已确认」。原型加横幅是为了让状态可见；若产品认定「不打扰」，删掉 `.confirm-banner` 即可，其余逻辑不受影响。
4. **确认不可撤销**：源码 `handleConfirm` 只有正向接口，没有撤销/重新回执；原型严格照此，确认后确认栏永久消失。
5. **正文分段规则**：`content` 是带换行的纯文本，原型按自然段拆成 5 个 `<p>`；真实落地时需要按 `\n\n` / `\n` 统一解析，否则长文可能仍显示为一段。
6. **未做**：`uni.previewImage` 的原图预览、下拉刷新（本页 `pages.json` 未开）、长按复制（源码没有）。

## 依赖未实现字段（P1 审查，2026-10-27）

纸面里的类型胶囊文案、发布人、附件区、已读时间、已确认时间共 5 处，在当前家长端通知详情实现里**没有数据源**：`GetNotice`（`server/internal/service/parent_portal_service.go:1190-1196`）只返回 `id / category / title / content / need_confirm / status / published_at / created_at / is_read / is_confirmed`。

契约 `backend/api/api-parent/02-notice.md` 已经定义了 `category_label`、`creator`、`attachments`（含 `file_type` / `file_name` / `file_size`）、`read_at`、`confirmed_at`，所以这不是设计缺口，而是实现未按契约返回（附件可直接调 `ListAttachmentsByNoticeID`，该仓储方法已存在）。后端补齐前，本页真机打开会表现为：类型胶囊 / 发布人 / 附件区为空、「已读」后无时间。

另：`published_at` / `read_at` / `confirmed_at` 前端目前直接截 UTC 串（`slice(0, 16)`），落地时要换成本地时区格式化。

## 验收

- `node scripts/od-check.mjs …/index.html --width 390` → `problems: []`（`scrollWidth 390 = clientWidth 390`，`overflowing: []`，滚动 400px 后 `statusTop: 0 / navTop: 20`，无 console error）
- `node scripts/od-audit.mjs docs/design-previews/2026-10-25-full-redesign` → 本页无 ✗（无裸色值、无 nowrap、无 `.mp-*` 重定义、已链接 `../../_shared/shell.css`、无 `_shared/` 副本、无 `.mcp.json`）
- 交互实测（CDP）：确认按钮 → 文案「确认中…」+ `disabled` → 0.8s 后确认栏 `display:none`、横幅出现、toast「已确认收到」；附件点击有描边反馈 + toast；错误态「重试」回正常态
- 状态矩阵实测（CDP，2026-10-27）：正常 / 仅告知 / 未读 / 已回执 / 加载 / 失败逐个切换，`scrollWidth 390 = clientWidth 390`、无越界元素；仅告知态确认栏 `display:none`、已读条保留且为 `--muted`；未读态已读条 `display:none`、确认栏保留；从已回执切到仅告知时 `body.is-confirmed` 被清掉，信息型纸面不会被染绿；未读态点确认后横幅 + 已读条一起出现
