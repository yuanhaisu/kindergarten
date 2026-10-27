# 全量页面重设计 · 对抗性审查报告

> 2026-10-27。范围：老师端 22 页 + 家长端 28 页，共 50 页。
> 方法：50 个独立 subagent 逐页审查（完整度 / 逻辑 / 设计核心 / 冗余错位 / 导航意图 / notes 准确性），主会话负责跨页与整体逻辑。单页明细见 `_shared/reviews/`。

## 0. 结论摘要

| 指标 | 结果 |
|---|---|
| PASS | 6 页 |
| WARN | 39 页 |
| FAIL | 5 页 |
| P0（用户可见硬伤） | 5 条 |
| P1（明显缺陷） | 90 条 |
| P2（打磨项） | 253 条 |

**先修 5 条 P0，再处理 90 条 P1；P2 可在进入 Impeccable 落地阶段时逐页清理。**

> 2026-10-27 更新：5 条 P0 已全部修复；P1 已修 85 条、暂缓 8 条（批次级）。详见 §2.1 与 §3.1。

## 1. 整体逻辑审查（主会话）

### 1.1 导航完整性：4 个页面零入边 + 1 个死链

| 页面 | 问题 |
|---|---|
| 老师端 `/pages/webview/index` | 源码零入边，无任何入口 |
| 家长端 `/pages/auth/bind` | 源码零入边，无任何入口 |
| 家长端 `/pages/messages/index` | 零入边（findings A39） |
| 家长端 `/pages/webview/index` | 零入边（findings A38） |
| 老师端 `/pages/notice/index` | 工作台待办硬编码跳转，但 `pages.json` 无此页 → **死链** |
| 老师端「我的」 | 源码自绘 4 项底栏，`pages.json` 无 tabBar；原型按规则不画（已记录） |

其余 45 页在源码里都有入边；原型集覆盖 `pages.json` 全部 50 页（老师 22 / 家长 28）。

### 1.2 跨页数据一致性

- **小一班人数**：`teacher/15-class-index` = 30 人；`teacher/19-profile-index` 与 `parent/20-points` = 28 人。前者 notes 解释为 28 在园 + 2 未入园，但三页都没标口径。
- **家长端今日叙事**：`parent/01-home`「今日在校情况 / 已入园 07:52」 vs `parent/13-pickup-index` 同日「已接走 17:30」，首页缺接送后的收口状态。

### 1.3 跨页冗余

- `teacher/01-workbench` 与 `teacher/19-profile-index` 都展示同一组「26 出勤 / 2 请假 / 3 待审批」今日班级统计（数值一致，但两页重复）。
- `parent/01-home` 与 `parent/13-pickup-index` 都讲今日状态，但一个讲在校、一个讲接送，未互相引用。

### 1.4 设计核心一致性

- **演示芯片不统一**：38 页 4 个（正常/空态/加载/失败），9 页 5 个，3 页 6 个；标签与命名也不一致（`error` vs `failed`、`空态` vs `空评论`、`必填未填` vs `必填未完成`）。
- `parent/08-leave-detail` 没有「正常/空态」，只有「已同意 / 待审批 / 已驳回 / 加载 / 失败」。
- 9 页存在 10–12px 小字（多为徽章/时间戳；brief 要求正文 ≥13px，属边缘）。
- 通过项：50/50 `tokens.css` 与设计系统同源、50/50 预览图 780px、外壳标记 50/50 完整、4 个 tab 页高亮正确、老师端零 tabBar、无 `.mcp.json` / 无 `shell.css` 副本。

### 1.5 交付物一致性

- **3 页 `preview.png` 落后于源码改动**：`teacher/12-calendar-create`、`teacher/19-profile-index`、`parent/12-growth-report`（HTML/CSS/JS 比截图新 39–112 秒）。
- `teacher/03-notice-create` 的 `<script>` 顺序是 `app.js` → `shell.js`，与批次约定相反（notes 已说明是刻意为之）。

## 2. P0 明细（必须修）

### P0-1 · parent/06-leave-index · [logic]

**统计卡『全部请假』的数字从不更新：JS 查 [data-stat="all"]，HTML 写的是 data-stat="total"，空态因此显示 6/0/0、失败态显示 6/—/—，与页面状态及 notes 的 0/0/0、全『—』说法矛盾。**

- 证据：index.html:53（data-stat="total"）；app.js:37 与 :49-52（values 用 all 键并 querySelector('[data-stat="all"]')，返回 null 被静默跳过）；DOM 实测空态 .stat-number 文本为 6/0/0，失败态为 6/—/—；notes.md:22-24 声称空态统计归 0/0/0、失败态统计显示「—」。
- 建议：把 index.html:53 的 data-stat 改为 all，或把 app.js 的键改为 total（同步 updateStats 三处赋值），复测空/失败态统计与列表状态一致。

### P0-2 · parent/22-calendar-index · [logic]

**视力筛查卡片的当前地点仍是「保健室」，与同屏变更说明「地点 保健室 → 医务室」直接矛盾，家长会按旧地点去保健室。**

- 证据：app.js:152-154 该事件同时写了 location: '保健室' 与 change_summary: '地点 保健室 → 医务室'；运行时选中 2026-10-27 后卡片 .event-loc 显示「保健室」，卡内 .event-change 显示「安排调整 地点 保健室 → 医务室」，顶部变更条也是同一句，矛盾同屏可见。
- 建议：把 vision-screening 的 location 改为「医务室」，让当前值与变更结果一致；_prompt.md 的示例数据同样是「保健室 + 保健室→医务室」，若按提示词重生成需一并修正。

### P0-3 · teacher/04-growth-index · [logic]

**底部筛选弹层和 toast 用 position:absolute 挂在内容高 2477px 的 .mp-app 上，弹层实际开在文档底部、视口之外；点两颗筛选 chip 只看到整屏遮罩变暗、看不到面板，发布/返回/打开详情/重试的 toast 也全部在视口外，主操作看起来完全没反应。**

- 证据：styles.css:1078-1085 `.sheet-layer{position:absolute;inset:0}`、styles.css:1223-1226 `.toast{position:absolute}`，而 shell.css:7-20 的 `.mp-app` 仅 `position:relative;min-height:100dvh`，高度被内容撑到 2477px。实测 893px 视口、scrollY=0：打开班级筛选后 `.sheet` top=2218.4/bottom=2501.0，`.toast` top=2371.5，均大于 innerHeight；同批次 teacher/07-pickup-index/styles.css:576-578 等页均用 position:fixed。
- 建议：把 .sheet-layer 与 .toast 改为 position:fixed（与 07/08/10 等页一致），或固定 .mp-app 高度并让 .mp-main 内部滚动；toast/弹层不应挂在可变高度容器上。

### P0-4 · teacher/06-growth-detail · [logic]

**AI 横幅声称「3 条高置信建议」，页内却只有 1 条高置信行；点「全部确认」实际只确认 1 条，且横幅在仍有待确认行时继续显示「3 条」，数字与动作结果自相矛盾。**

- 证据：index.html:65 `<strong>3</strong> 条高置信建议`；#aiList 内仅 index.html:132 一条 `.status--ok`（乐乐），index.html:143 是 `.status--pending`（朵朵）；app.js:114 只筛选 `.status--ok`，app.js:121 弹出「已确认 1 条高置信建议」；app.js:90 仅在 rows.length===0 时隐藏横幅。
- 建议：横幅条数改为按 `.status--ok` 行实时计算，并在每次确认/拒绝后同步；或补齐 3 条高置信行，使「全部确认」的条数与 toast 结果一致。

### P0-5 · teacher/14-leave-index · [logic]

**点击「同意」或「确定驳回」后，新审批记录显示「审批 10月NaN日」，展开审批留痕显示「10月NaN日 00:00」——核心动作产生可见数据错误。**

- 证据：app.js:34 `NOW_STAMP = "2026-10-26 12:00"` 用空格分隔，而 parseDate/parseTime 只按 "T" 切分（app.js:61-69）；approve/reject 把 reviewedAt 写成 NOW_STAMP（app.js:360、375），formatDay/formatStamp（app.js:87-95、75-85）得到 d=NaN。实测：同意 id=1 后 .card-time 文本为「审批 10月NaN日」，.trace 为「10月NaN日 00:00」；驳回 id=3 同样；notes.md:20 声称「已用 CDP 逐条跑通」。
- 建议：把 NOW_STAMP 改为 "2026-10-26T12:00"，或让 parseDate/parseTime 兼容空格分隔；补交互回归断言（审批后卡片时间不得含 NaN）。

## 2.1 修复记录（2026-10-27）

| P0 | 页面 | 修复 | 验证 |
|---|---|---|---|
| P0-1 | parent/06-leave-index | `index.html` 统计键 `data-stat="total"` → `all` | 实测：正常 6/1/2、空态 0/0/0、失败 —/—/— |
| P0-2 | parent/22-calendar-index | `app.js` 视力筛查 `location` 保健室 → 医务室（`_prompt.md` 同步） | 实测：卡片显示「医务室」，变更说明仍为「保健室 → 医务室」 |
| P0-3 | teacher/04-growth-index | `styles.css` `.sheet-layer` / `.toast` 由 `absolute` 改 `fixed` | 实测：弹层 561–844、toast 740–760，均在 844 视口内可见 |
| P0-4 | teacher/06-growth-detail | 横幅静态 3 → 1；`syncAiState` 按 `.status--ok` 实时计算并在无高置信行时隐藏 | 实测：初始 1 条；全部确认后横幅隐藏、toast「已确认 1 条」 |
| P0-5 | teacher/14-leave-index | `NOW_STAMP` 由空格分隔改为 `2026-10-26T12:00` | 实测：同意/驳回后时间「今天 12:00」，无 NaN，计数同步 |

**复验**：5 页 `od-check` 均 `problems: []`；批次 `od-audit` 50/50 通过；8 张预览已重出（5 张修复页 + 3 张原过期页），全批次无过期 `preview.png`。

## 3. P1 明细（按页面）

- **parent/01-home** · [logic] 多孩切换只重绘孩子卡与今日四列，没有刷新「今晚一件事」和「亲子任务」角标；切到小雨后仍显示乐乐的「亲子阅读」任务与角标 3，两个绑定当前孩子的数据块与选中孩子不一致。
  - 证据：docs/design-previews/2026-10-25-full-redesign/parent/01-home/app.js:76-104 的 applyProfile/切换回调只更新 fieldMap 与奖章；index.html:153-166 的任务卡和 :139 的角标 3 为静态内容；源码切换后 await loadHome()（parent/src/pages/index/index.vue:141），任务按 student_id 拉取（index.vue:87-90），后端 pendingTaskCount 也按 CurrentStudent 计算（server/internal/service/parent_portal_service.go:2534）。
  - 建议：为小雨准备独立的任务/角标数据并在切换时一并重绘；若保留简化，至少在 notes 与页面上标注该区不随孩子切换。
- **parent/01-home** · [logic] 空态把「今晚一件事」与动态列表一起隐藏，且通知角标仍为 1、任务角标仍为 3，与空态文案「这周还没有新的通知和成长记录」自相矛盾；空态判定把两套互不相关的数据源耦合在一起。
  - 证据：parent/01-home/styles.css:756-758 同时隐藏 .tonight-section 与 .feed-list；index.html:123/139 的角标 1/3 在空态仍可见，:200 的空态文案称本周无通知；源码中今日任务只依赖 todayTask（parent/src/pages/index/index.vue:307），动态空态只作用于 feed（index.vue:325-327）。
  - 建议：空态只替换动态区，按 todayTask 保留今晚一件事；角标应随空态数据置 0 或隐藏，避免与空态文案冲突。
- **parent/02-login** · [design-core] 主按钮白字压品牌橙渐变，实测对比度仅 2.26–2.80:1；15px/600 属正文级，低于 WCAG AA 4.5:1，本页唯一主操作的可读性不达标。
  - 证据：tokens.css:18 定义 --on-brand:#FFFFFF；tokens.css:24 定义 --accent-gradient: linear-gradient(135deg,#FB923C 0%,#F97316 100%)；styles.css:178-182 按钮用该渐变 + --on-brand + 15px/600；notes.md:72 自述「2.80:1（浅端 2.26:1），不达 AA」。
  - 建议：在 token 层给品牌橙配深色 --on-brand（或把渐变整体压深到 ≥4.5:1），并同步 teacher 端；不要逐页改按钮颜色。
- **parent/02-login** · [redundancy] 失败态同时展示常驻错误条和内容等价的 danger toast，同一失败原因在一屏重复两遍；源码与页面 brief 都明确只要常驻错误条，且 role=alert 与 role=status 会重复播报。
  - 证据：index.html:63-75 错误条 role=alert aria-live=assertive；app.js:154-158 失败 chip 同时 setErrorCopy 与 showToast('微信登录失败，请稍后重试','danger')，app.js:111-115 真实断网失败同样双写；login.vue:37-39 注释「不再用 toast…改渲染成卡片内常驻错误条」；_prompt.md:16 同义；teacher/02-login/notes.md:66 老师端只保留常驻错误条；notes.md:65 先说「失败原因只留在卡片内」又说「额外给一条 danger toast」。实测失败态：error bar display:flex + toast is-on/data-tone=danger/同文案。
  - 建议：删除失败路径的 danger toast（app.js:114、:158），只保留常驻错误条；若需即时反馈，用不重复失败原因的短提示，并避免 role=alert 与 role=status 同时播报。
- **parent/03-bind** · [redundancy] 导航栏标题与 hero 大标题都写「绑定孩子」，两个标题在首屏相距 59px 同时可见，重复同一信息。
  - 证据：index.html:28 `<h1 class="mp-title">绑定孩子</h1>`，index.html:40 `<h2 class="hero-title" id="pageTitle">绑定孩子</h2>`；实测 nav 标题 top=41、字号 15px，hero 标题 top=100、字号 30px，文本完全相同。同批次 02-login、07-leave-create、24-feedback、25-settings 的 mp-title 均未在内容区重复。
  - 建议：hero 改成任务型标题（如「输入邀请码」/「认领孩子」），或删除 hero 标题，把「绑定孩子」只保留在导航栏。
- **parent/03-bind** · [logic] 输入框在第 1 到 5 个字符之间就进入 error 态：红边、红色错误行、aria-invalid=true，并露出会清空输入的「重新填写」，用户还没输完就被判失败。
  - 证据：app.js:271-274：length===0→empty，0<length<6→state='error'+COPY.short；app.js:96 `refillBtn.hidden = state !== 'error'`。实测输入 1 个字符 'K' 后：state=error、fieldReportText=「邀请码至少 6 位」、input.is-error=true、aria-invalid=true、refillVisible=true、submitDisabled=true。
  - 建议：打字过程中保持 neutral（或只显示「还差 N 位」），把「至少 6 位」与「重新填写」延到 blur 或首次提交失败后再出现。
- **parent/03-bind** · [navigation] `pages/auth/bind` 在家长端没有任何入站入口，App 内无法到达该页；原型集里也没有任何页面链接到 03-bind。
  - 证据：parent/src/pages.json:17 只注册 path，parent/src 内无 navigateTo/redirectTo/reLaunch 指向该页；`node scripts/_nav-audit.mjs` 输出 /pages/auth/bind inbound=0（与 A38 webview、A39 messages 同为零入站）；parent/src/App.vue:5-9 无 scene/query 路由；原型集 grep 不到 03-bind 链接。
  - 建议：在 23-student-index 或 27-profile 增加「绑定新孩子」入口指向 pages/auth/bind；若实际靠外部小程序码/场景值进入，在 notes.md 声明入口与参数。
- **parent/05-notice-detail** · [completeness] 正常态无条件渲染「确认收到」栏与「已读」条；源码中两者分别是 need_confirm && !is_confirmed、is_read 的条件渲染，原型没有「无需确认的信息型通知」和「未读」变体，会把所有通知都设计成必须回执。
  - 证据：D:/yuanhaisu/personal_projects/kindergarten/parent/src/pages/notice/detail.vue:85（v-if detail.need_confirm && !detail.is_confirmed）、:75-76（v-if detail.is_read）；D:/yuanhaisu/personal_projects/kindergarten/docs/design-previews/2026-10-25-full-redesign/parent/05-notice-detail/index.html:152-157 无条件渲染 confirm-bar，同目录 styles.css:579-583 只在 loading/error/empty 隐藏；index.html:161-166 的四个演示芯片无「无需确认/未读」；D:/yuanhaisu/personal_projects/kindergarten/server/internal/domain/notice.go:37 默认 need_confirm=false。
  - 建议：增加 need_confirm=false 的信息型变体（或把「正常」拆成「需确认/仅告知」），已读条按 is_read 分支；notes 覆盖状态同步补充。
- **parent/05-notice-detail** · [completeness] 原型纸面的类型胶囊、发布人、附件区、已读/已确认时间共 5 处内容，在当前家长端通知详情实现里没有数据源；真机打开会是空标签、空作者名、无附件、『已读』后无时间。
  - 证据：D:/yuanhaisu/personal_projects/kindergarten/server/internal/service/parent_portal_service.go:1190-1196 GetNotice 仅返回 id/category/title/content/need_confirm/status/published_at/created_at/is_read/is_confirmed，无 category_label/creator/attachments/read_at/confirmed_at；契约 D:/yuanhaisu/personal_projects/kindergarten/backend/api/api-parent/02-notice.md:46-75 列有这些字段；原型 D:/yuanhaisu/personal_projects/kindergarten/docs/design-previews/2026-10-25-full-redesign/parent/05-notice-detail/index.html:51-61、73-105 全部画实；附件查询 D:/yuanhaisu/personal_projects/kindergarten/server/internal/repository/notice_repo.go:223 存在但未被详情调用。
  - 建议：后端按契约补返回（attachments 调 ListAttachmentsByNoticeID）；若本批次不修后端，notes 需把该页标注为依赖未实现字段。另：published_at/read_at 前端直接截 UTC 串（teacher_portal_service.go:473-477），与 A15 同类，落地要本地化格式化。
- **parent/06-leave-index** · [logic] 失败态点『重试』后页面永远停在 error、列表不恢复，却 toast『已重新加载』，是一个自相矛盾的假恢复。
  - 证据：app.js:134-150 只重置按钮并 showToast('已重新加载')，未改 app.dataset.state、未重新渲染列表；DOM 实测点击后 1.4s：data-state=error、.error-state display:flex、.leave-list-area display:none、toast 文本「已重新加载」；notes.md:25 描述为「1s 后恢复」。
  - 建议：重试后切回 normal 并恢复列表/统计；若只演示请求过程，至少把文案改为「已重新发起请求」并保留 error，不要让 toast 声称已恢复。
- **parent/08-leave-detail** · [redundancy] 审批人与时间在已同意/已驳回态重复出现三处，待审批与已撤销态的等待/撤销文案也层层堆叠
  - 证据：index.html:45「王老师 · 10月26日 08:40 完成审批」、:59「王老师已同意 · 10月26日 08:40」、:191-192「王老师」/「10月26日 08:40」；DOM 实测已同意态「王老师」「10月26日 08:40」各出现 3 次；待审批态 :72/:86/:221-222 四处等待说明；已撤销态 :126/:140/:233-234/:250 五处撤销说明
  - 建议：状态头只留结论（可保留一句极短 meta）；时间线独占提交/审批时间；审批卡只留意见正文；待审批/已撤销各保留一处主说明与一处终态说明
- **parent/08-leave-detail** · [design-core] .mini-avatar 没有任何样式规则，学生行渲染成裸文字「乐 乐乐 · 小二班」
  - 证据：index.html:154 有 <span class="mini-avatar">乐</span>；styles.css 全文无 .mini-avatar（只有 :372 的 .avatar--teacher）；DOM 计算样式 background transparent、13×20.8px、border-radius 0；preview 裁图确认首字「乐」是普通文字而非圆形头像，列表页 06 的同类 .leave-avatar 是 40px 圆形底（06/styles.css:221-229）
  - 建议：为 .mini-avatar 补 token 化圆形头像（宽高 --avatar-sm、border-radius: var(--radius-pill)、background: var(--accent-wash)、color: var(--accent-deep)），或删除该 span 仅保留姓名
- **parent/08-leave-detail** · [logic] 日期星期与 2026 批次自相矛盾：10月27日写成周一、10月29日写成周三
  - 证据：08/index.html:162「10月27日 周一 → 10月29日 周三」；parent/07-leave-create/index.html:83-84「2026-10-28 周三」、:94-95「2026-10-29 周四」，同一 10月29日在两页分别为周三/周四；2026-10-27 实为周二、10-29 为周四
  - 建议：改为「10月27日 周二 → 10月29日 周四」，或去掉星期只留日期，避免跨页再冲突
- **parent/09-growth-index** · [completeness] 缺源码的「孩子视角筛选后为空」状态（mode-empty）的标题与说明：孩子视角把所有记录过滤掉时，原型只剩「时光轴 / 2026年 / 0 条」和空白列表，没有任何解释。
  - 证据：parent/src/pages/growth/index.vue:325-331 有独立的 v-else-if='visibleFeeds.length === 0' 空态（标题与说明随视角变化，并带「看全部班级动态」）；本页 index.html:102-116 只有 feeds 全空时的「老师还没有发布成长记录」空态，:365-368 演示条也只有四态；app.js:85-141 的 applyView 只隐藏记录和月份组，不渲染该分支；notes.md:43 只用顶部切换替代 CTA，未覆盖缺失的状态文案。
  - 建议：补一个孩子视角专属空态（或在 applyView 中当 recCount=0 且 feeds 非空时显示）：标题「还没有与乐乐相关的成长记录」、说明「可以先看看班级动态，或等老师发布新的记录。」；CTA 可继续由顶部视角切换承担。
- **parent/10-growth-detail** · [navigation] 返回箭头是看起来可点的按钮，但没有绑定任何 handler，点击后页面与 URL 均不变；同批次 8 个家长端子页都接了返回 toast，_prompt 也要求「导航与返回箭头保持可用」。
  - 证据：index.html:23-25 定义 .mp-back（aria-label=返回）；app.js 全文无 .mp-back/返回 处理，实测点击 URL 不变；_prompt.md:60；对照 05-notice-detail/app.js:53-57 等 8 页均有返回兜底。
  - 建议：给 .mp-back 绑定返回逻辑：独立预览用 history.back()，或 toast「返回成长时光」；加载/失败态保持可用。
- **parent/10-growth-detail** · [logic] 照片预览弹层显示「第 N 张 / 共 M 张」，但弹层内只有关闭按钮，没有上一张/下一张或滑动手势；用户无法在弹层内查看其余照片，计数器成了假承诺。源码 uni.previewImage 支持左右滑，这是照片优先页的核心交互。
  - 证据：index.html:253-266 弹层内唯一按钮 #closePreview；app.js:71-82 openPreview 只替换当前张并写计数，未监听 ArrowLeft/Right；detail.vue:57-59 传全量 urls+current；notes.md:65 自认不支持切图。
  - 建议：补左右箭头与触摸滑动（或至少「上一张/下一张」按钮），计数器随当前 index 更新；若不打算做切图，删掉计数文案。
- **parent/11-growth-album** · [logic] 整页空态下月份条没有隐藏、箭头仍可切换：页面停在「还没有照片」，标题却能一路从 2026年10月 翻到 12 月；此时点「刷新看看」落到的是 2026年12月的月内空态「本月暂无照片」，原本有 18 张的 10 月回不去。空态描述的状态与可用操作互相矛盾，也与 notes 的声明相反。
  - 证据：styles.css:338-339 在 empty 只隐藏 .mb-meta，全文件没有隐藏 .monthbar 的规则（.monthbar 定义在 styles.css:54）；app.js:78-80 的 locked 仅含 loading/error，empty 下 #nextMonth 仍 enabled；DOM 实测：data-state=empty 时 .monthbar display:block，连点两次后 #monthTitle=2026年12月 而 .state-empty 仍显示「还没有照片」，再点 #emptyRefresh → data-state=normal、标题 2026年12月、[data-month-empty] 显示「本月暂无照片」并 toast「已刷新」；notes.md:21 声称空态「月份条与网格隐藏」。
  - 建议：二选一并让实现与 notes 一致：a) 保留整页空态时补 .mp-app[data-state="empty"] .monthbar { display:none; }，并在 empty 下冻结 goMonth（需先有全量照片数据，见另一条 finding）；b) 删除整页空态，把演示 empty 芯片映射为月内空态，月份条保持可用。
- **parent/12-growth-report** · [logic] 空态与失败态的出口把周期写死成「月报/周报」，不读当前选中的周期：在「月报」下进入空态后，页面仍说「这个周期还没有成长报告……先看看月报或学期报告」并给「查看月报」按钮，点击后刚被判为「没有报告」的月报立刻出现数据，toast 还谎称「已切换到月报」；失败态「重试」同样固定跳回周报。
  - 证据：app.js:73 `returnToNormal('monthly', '已切换到月报')`；app.js:83-88 的 mp:state 处理在非 normal 时不重置周期，所以先点 `[data-od-id="report-tab-monthly"]` 再点演示芯片「空态」，分段仍高亮「月报」，而 index.html 的 `.state-empty .state-copy` 仍是「老师通常周五整理周报，先看看月报或学期报告」、`[data-od-id="report-empty-cta"]` 仍是「查看月报」；app.js:79 的 `returnToNormal('weekly', ...)` 让从月报失败态点「重试」跳到周报。
  - 建议：状态出口读取当前选中周期：空态 CTA 指向一个与当前周期不同的周期（周报→月报、月报→学期、学期→返回成长时光），文案随周期生成；「重试」沿用失败前的周期，toast 只在周期真的变化时出现。
- **parent/12-growth-report** · [logic] 三个周期的演示数字只满足单调递增，日速率互相矛盾：学期（9/1–10/25）在扣除月报（9 月）后，10 月 1–25 日的积分/记录/任务日速是月报与周报的 1.7–2.9 倍，等于说 10 月的产出比 9 月高一倍以上，而周报（10/19–10/25）的日速又与 9 月持平。
  - 证据：index.html 三份 `.stats`：周报 45 分/15 条/8 任务（10/19–10/25，7 天）、月报 180 分/62 条/31 任务（9/1–9/30，30 天）、学期 610 分/186 条/96 任务（9/1–10/25，55 天）。学期=9 月+10 月 1–25 日，倒推 10 月 25 天需新增 430 分/124 条/65 任务（日速 17.2/4.96/2.6），而月报日速 6.0/2.07/1.03、周报日速 6.4/2.14/1.14；出勤率 91%→94% 要求 10 月约 98%，当周却只有 80%。notes.md:60 却声明「演示数字自洽」。
  - 建议：按日速率反推学期值（约 340 分 / 115 条记录 / 58 个任务），或把月报改成 10/1–10/25 再重算四格，使学期 ≈ 月报加 10 月增量与周报同速。
- **parent/13-pickup-index** · [completeness] 今日卡只实现「已接走」终态，缺 brief 要求的「仅入园→在园中」分支；空态又把历史一起隐藏，导致「今日尚无记录但有历史」的上学日早晨没有可用状态。
  - 证据：index.html:55-61 把 status-main 写死为「已接走」，无第二分支；_prompt.md:19 明确要求仅有入园记录时显示「在园中 + 07:52 妈妈（李丽）送入」；styles.css:12 让 empty/loading/error 同时隐藏 .pass 与 .history；notes.md:19 把空态定义为「今天还没接送且无历史」。DOM 实测：切到 empty 后 .history display:none，正常态也没有在园中/未入园分支。
  - 建议：按当天记录派生「未入园 / 在园中（最后一条为入园）/ 已接走」三态；empty 只隐藏今日卡、保留历史，或新增「今日无记录 + 历史列表」状态。
- **parent/13-pickup-index** · [logic] 默认态已显示「已接走 17:30」，却仍把「出示接送二维码」作为唯一实心主按钮；弹层继续显示「今日接走人 爸爸（王明）」和当日 23:59 有效期，与已完成的接走事实矛盾。
  - 证据：index.html:58-59 状态「已接走 / 17:30 / 爸爸（王明）」；index.html:83-86 主按钮「出示接送二维码」；index.html:249-268 弹层人证对照与有效期；外壳时钟 index.html:27 固定 9:41，却在今天展示 17:30 已发生的事件。DOM 实测该按钮可正常打开弹层。notes.md:48 只解释了示例时点，未处理该矛盾。
  - 建议：在「已接走」终态隐藏/禁用二维码 CTA（改成「今日接送已完成」或回执），把主按钮只在未接走时露出；或把默认示例改为「在园中」态。
- **parent/14-pickup-qrcode** · [completeness] 错误态把源码的三类业务错误（未获取到接送凭证 / 请先指定今日接走人 / 授权已失效）统一压成「网络不太稳定，请稍后再试」，且只提供「重试」；源码错误卡唯一恢复入口「去指定今日接走人」以及错误态下仍渲染的「更换今日接走人」都被隐藏，授权失效时用户无路可走。
  - 证据：qrcode.vue:23-25、31-37 区分三类错误文案；qrcode.vue:84-88 错误卡含「去指定今日接走人」按钮；qrcode.vue:124-125 两个操作按钮在非 loading 分支（含 error）仍渲染。原型 index.html:143-157 错误卡只有「凭证加载失败/网络不太稳定/重试」；styles.css:529-532 在 error 态隐藏整个 action-stack；notes.md:51 称源码错误卡「只含文案」，与 qrcode.vue:87 不符。
  - 建议：错误态按类型分支：授权失效→「重新指定」跳 15-pickup-authorized，未指定→沿用空态 CTA，网络错误→重试；至少保留源码的「去指定今日接走人」入口与「更换今日接走人」，并在 notes 更正对源码错误卡的描述。
- **parent/14-pickup-qrcode** · [navigation] 导航栏返回箭头是 button，但 app.js 没有为 .mp-back 绑定任何 handler，点击完全无反应；同批次 05/12 都有绑定，属于可点控件失效。
  - 证据：index.html:26-28 渲染 .mp-back；app.js:10-15 的元素查询列表中没有 .mp-back，全文也无 back 绑定；对比 parent/05-notice-detail/app.js:53-58、parent/12-growth-report/app.js:68 均给 .mp-back 绑定了 toast。
  - 建议：在 app.js 中 querySelector('.mp-back') 并绑定 showToast('返回接送记录')，或在 shell.js 统一兜底，避免每页漏绑。
- **parent/15-pickup-authorized** · [logic] 张建军卡片同时出现黄色「待审核」和绿色「可接送」：review_status=0 已使其不可指定（按钮降级、底部说明也写明审核后才能指定），但权限标签仍用 success 绿表达「可接送」，审核状态、权限标签、按钮状态三者互相矛盾；同页刘芳（已撤销）卡片又完全没有权限标签行，两张不可指定卡的处理口径也不一致。
  - 证据：index.html:189-203（张建军 data-review-status="0"、badge--pending「待审核」、permission-tag--pickup「可接送」）；index.html:212-228（刘芳 data-status="0"、badge--revoked「已撤销」、无权限标签）；styles.css:381-384（可接送使用 --success/--success-wash 绿色）。
  - 建议：统一权限标签口径：按有效授权（review_status===1 && status===1）显示，或在待审核/已撤销卡上都标注「配置权限，审核通过后生效」；不能只在待审核卡保留绿色「可接送」。
- **parent/16-pickup-authorized-create** · [logic] 「失败」态下点主「提交」按钮会重新走成功流程，但失败条不消失：成功 toast 与「提交失败，请重试」同屏，状态自相矛盾。
  - 证据：app.js:361-380 的提交处理只拦 loading/submitting，未清理 data-state=failed（对比 app.js:382-390 的 retryBtn 会置 normal）；styles.css:502-504 失败条由 .mp-app[data-state="failed"] 常驻显示。DOM 实测：切「失败」芯片→点 #submitBtn，1.6s 后 data-state 仍为 failed，#failBar display:flex，#toast 同时显示「已提交，待园所审核」。
  - 建议：提交前先退出 failed 态并同步芯片/演示条（与 retryBtn 一致），或提交中隐藏失败条；也可在 failed 态禁用主提交、只留「重试」。
- **parent/18-tasks-detail** · [logic] 老师回应/未读提示被绑定到「公开已通过」：源码中 responses、hadUnread 与 submitted_current_period 是三个独立条件，原型只在 approved 显示 .feedback/.whisper，submitted 态整块隐藏；「本周期已提交 + 老师已回应 + 公开待审核」这一常见组合看不到回应。反过来 approved 态又隐藏打卡区，「昨天有回应、今天未提交」时无法打卡。
  - 证据：styles.css:956-972（submit-card/checkin-card 互斥，feedback/whisper 仅 approved 显示）；detail.vue:448-450、452-455、515（三个 v-if 相互独立）；06-task.md:115-118（responses 为主回应，与公开状态无关）。
  - 建议：把回应与未读提示从 submit-card 拆成独立区块，严格按源码三个独立条件渲染；状态演示补一个「有回应 + 本周期待提交」组合态。
- **parent/18-tasks-detail** · [logic] 当前周期提交被移出历史时间轴后，公开操作一并丢失且状态写错：「我的提交」卡没有任何申请/取消公开按钮，刚打卡就显示「已申请公开，等待老师审核」（实际未发申请，契约中申请是独立 POST）；approved 态也无法撤下当前这条，只能误操作 10月25日 的旧记录。
  - 证据：index.html:87-145（submit-card 内无 data-public）；index.html:207/220/232（公开按钮只在 10-25/24/23 历史行）；app.js:77-83（提交后直接 setState('submitted')）；index.html:102-105（静态「已申请公开」文案）；detail.vue:535-543（源码动作在每条 history 行）；06-task.md:253-255（申请是独立 POST）。
  - 建议：在「我的提交」卡带出公开状态与申请/取消按钮（两步确认），或把当前提交保留在时间轴顶部；提交后默认按 is_public_requested=false 显示「未公开」+「申请公开」。
- **parent/18-tasks-detail** · [logic] 历史行的公开状态被压成三个混合态，与源码 publicState 四态不一致：①从未申请（none）的行标成「未公开」并配「申请公开」，源码 none 不显示状态，revoked 才是「未公开」+「重新申请」；②一键完成（无文字无媒体）的行也给「申请公开」，源码会隐藏且接口必 422；③缺 approved 但 public_visible=false 的「老师已通过，暂不对本班展示」。
  - 证据：index.html:224-233（「（一键完成）」行 + 「未公开」徽标 + 「申请公开」）；detail.vue:107-130（四态文案与动作）、132-141（content||media_url 才给申请）、114（not-visible 分支）；06-task.md:249-251（一键完成申请 422）、285（public_status 语义）。
  - 建议：按 publicState/publicLabel/publicActionLabel/showPublicAction 四函数逐分支实现：none 无状态文案且无内容不给申请、revoked 用「重新申请」、approved 按 public_visible 分流。
- **parent/18-tasks-detail** · [redundancy] 同一个打卡总数在一页出现三次且空态不归零：任务卡「累计打卡 7 次」、历史标题「共 7 次」、脚注「共 7 条」；切到空态时前两处仍是 7，与同屏「还没有打卡记录」直接矛盾，点空态「去打卡」回正常态又冒出 3 条历史；单位「次/条」也不统一。
  - 证据：index.html:72-75、186-187、237-246；styles.css:974-976（空态只隐藏 timeline/history-more，history-count 与 meta-grid 仍可见）；app.js:214-219（空态 CTA 切回 normal）；detail.vue:527-547（源码只有 total>length 时的一条 hint，没有「累计打卡」统计）。
  - 建议：只保留历史脚注一处总数（源码口径），空态置 0 或隐藏；统一用「条」；若要保留「累计打卡」必须与 history.total 同源并随状态变化。
- **parent/19-tasks-wall** · [design-core] 日期条吸顶完全失效：页面滚到 600px 时「今天」日期条已滚出视口，与 notes 宣称的「日期当钉子、吸顶在导航栏下方」相反。
  - 证据：styles.css:168-170 声明 position:sticky; top:64px；但 shell.css:133 的 .mp-main{overflow-x:hidden} 使其成为滚动容器，sticky 相对不滚动的 .mp-main 定位。Chrome 实测：window.scrollTo(0,600) 后 .wall-datebar.getBoundingClientRect().top = -472（预期 64）；把 .mp-main 的 overflow-x 临时改为 clip 后同一测量为 64。notes.md:13、notes.md:46 与 _prompt.md:36 都把吸顶当作已实现。
  - 建议：把 shell.css 的 .mp-main 改为 overflow-x: clip（clip 不建立滚动容器，实测 sticky 恢复为 top 64），或让 .mp-main 成为真正的滚动容器并约束高度；改后必须在 400px+ 滚动位置真实测量日期条 top 恒为 64。
- **parent/20-points** · [logic] 积分明细标称「共 128 条」，但点一次「加载更多」追加 2 条后即置灰为「已经到底了」，列表实际只有 9 条，总量与到底状态自相矛盾。
  - 证据：parent/20-points/index.html:69 panel-meta「共 128 条」；parent/20-points/app.js:128-136 点击后 appendHistoryRows() 追加 2 条（app.js:138-169）就 loadMore.disabled=true 并把文案改成「已经到底了」；backend/api/api-parent/07-points.md 的 history 契约为 total=128、page/page_size 分页（示例 page_size=20）。
  - 建议：按契约分页（每次 20 条，显示「已加载 9/128」并允许继续加载），或把总量改成实际 9 条并说明这只是最近记录，不要在 9 条时宣称到底。
- **parent/20-points** · [redundancy] 本周/本月统计卡被提到 tab 之外常驻，导致排行榜 tab 同屏出现两个「本周 45」；成就 tab 的卡头同屏出现「已获得 8 / 12」与「已解锁 8 枚」；且源码里本周/本月本来是积分 tab 的内容，现在积分 tab 只剩新增的流水，原始统计被抽出。
  - 证据：parent/20-points/index.html:55-58 stat-grid 位于 tabs(60-64) 之上；:56「本周积分 45」与 :113「我的本周积分 45」同屏；:138「已获得 8 / 12」与 :139「已解锁 8 枚」同屏；:114「我的排名 3」与 :40「班级第 3 / 28」及 :124 已高亮的第 3 行重复；源码 parent/src/pages/points/index.vue:47-50 本周/本月仅在 tab==='overview' 时渲染。
  - 建议：把本周/本月放回积分面板（或在排行榜/成就 tab 隐藏），panel-sub/panel-meta 二者只留一处并承载不同信息（如统计周期/口径），避免同一数字在卡头出现两次。
- **parent/22-calendar-index** · [logic] 「今天」胶囊只在当前月禁用，选中当前月的其他日期后仍然禁用，无法一键回到今天。
  - 证据：app.js:305 仅用 isCurrent 决定 todayBtn.disabled，而 selectDate()（app.js:539-548）不调用 renderToolbar()；运行时点选 2026-10-02 后 #todayBtn.disabled 仍为 true，#eventsDate 已变成「10月2日 周五」。
  - 建议：禁用条件改为 isCurrent && selected === TODAY_ISO，或在 selectDate() 末尾重算工具栏，使当前月选中非今天日期时胶囊可用。
- **parent/22-calendar-index** · [redundancy] 每张活动卡把类型名渲染两次（标题行胶囊 + 元信息行「类型 · 时间」），同一屏重复。
  - 证据：app.js:444 的 .type-pill 与 app.js:446 的 .meta-type 使用同一个 label；运行时每张卡同时存在「活动」胶囊与「活动 · 09:00–10:30」。老师端同模块 teacher/11-calendar-index/app.js:179-185 只在 meta 渲染一次类型。
  - 建议：删除标题行的 .type-pill（保留图标 + meta），或删除 meta 里的 .meta-type 只留胶囊；与老师端卡片口径统一。
- **parent/23-student-index** · [design-core] 档案卡 2×2 事实格被 max-width:100% 抵消了左右负边距，整块相对卡片内容左移 16px：分隔线左端顶到卡片外缘，右端却比卡片内容右缘短 16px，第二列整体偏左。
  - 证据：styles.css:119-125 `.facts{max-width:100%;margin:var(--space-xl) calc(-1*var(--space-xl)) 0}`；DOM 实测 `.facts` x=61.5/right=391.5，而 `.identity`/`.health-note` x=77.5、卡片内容右缘 407.5（左右各偏 16px），右侧列分隔线随之错位。
  - 建议：移除 `.facts` 的 `max-width:100%`（或改为 `width: calc(100% + 2 * var(--space-xl))` 保留兜底），让事实格左右出血对称。
- **parent/23-student-index** · [completeness] 晨检卡缺源码已渲染的 is_isolated 分支（「已在隔离室等待家长」），原型既无该行也无样式；notes 却把该字段误列为「源码与原型都未展示」。
  - 证据：parent/src/pages/student/index.vue:118 `<text v-if="r.is_isolated" class="health-isolated">已在隔离室等待家长</text>`；接口 backend/api/api-parent/12-health.md:29 返回 is_isolated；index.html:90-115 仅日期/胶囊/异常/体温/备注，styles.css 中无 `.health-isolated`；notes.md:72 将其列入「源码与原型都未展示」。
  - 建议：补隔离提示行与对应样式（warning/danger 语气），并在示例或说明中给出可达状态；同步修正 notes.md:72。
- **parent/24-feedback** · [logic] 「失败」态把「列表加载失败」和「提交失败」两个互不相干的失败压进同一个 data-state，任一重试都会静默清掉另一条，并把并未加载成功的 3 条记录当成已加载显示
  - 证据：styles.css:476-481（records-ok / records-error 全由 [data-state="failed"] 驱动）与 styles.css:508（bar-error 也由同一属性驱动）；app.js:371-379 recoverRecords() 只做 setState('normal')。实测（chrome-devtools 打开本页）：failed 态同时 bar-error=flex + records-error=flex；点列表「重试」后 bar-error 由 flex→none（提交失败未经任何重试就消失）；点提交条「重试」后 records-error 由 flex→none、records-ok 由 none→block、计数由「加载失败」变「共 3 条」
  - 建议：拆成 listState 与 submitState 两个独立维度（或两个独立 data 属性）：列表错误面板只由列表请求结果控制，提交失败条独立显隐；两个重试各管各的，禁止互相清除
- **parent/24-feedback** · [logic] 「提交中」演示芯片没有出口：1.3s 后不会进入成功态，且 formLocked 让点击提交直接 return，页面永久停在转圈状态
  - 证据：app.js:348-369 applyDemoState() 只 clearTimeout 不创建计时器；app.js:259-260 startSubmit 与 app.js:275-276 handleSubmit 都以 if (formLocked) return 早退。实测：点「提交中」后 2.2s 仍是 state=submitting、view=form、按钮文案「提交中…」，再点 #submit-btn 无任何反应。notes.md:25 与 notes.md:29 却声称「1.3s 后进入成功态」「成功态由「提交中」延时 1.3s …进入」
  - 建议：芯片进入 submitting 时复用 startSubmit 的 1.3s 计时（或把成功态做成第 7 个芯片）；若保持现状，必须在 notes 里改成「成功态只能由真实提交进入」
- **parent/24-feedback** · [logic] 成功后「查看我的反馈」「再提一条」把输入清空却把状态留在 normal：必填框为空时提交按钮仍是品牌渐变可点、底栏仍是普通说明，与 notes 宣称的「清空内容也会即时进入阻断态」相反
  - 证据：app.js:315-320（viewRecords）与 app.js:335-339（writeAnother）都是 setContent('') 之后 setState('normal')。实测点「查看我的反馈」后：textarea.value.length=0、submit aria-disabled=false、按钮 backgroundImage=linear-gradient(135deg, rgb(251,146,60)…)、可见提示为「提交后可在记录里查看进展」；只有再次输入（app.js:396-397）或点提交（app.js:283-287）才转 required。notes.md:22、notes.md:51 声称清空即阻断
  - 建议：清空后统一走校验（setState('required')），或让 setContent('') 自身触发阻断态，保证「空内容=阻断」在成功返回路径上也成立
- **parent/25-settings** · [logic] 加载态骨架块只有 188px 高且不占满内容区，「退出登录」在首屏加载时被顶到页面中部（y=288），下方留白 500px；同一按钮在 normal/empty/loading 三态分别落在 y≈673/764/288，切换时跳动约 385px，且源码在 loading 期间根本不渲染该按钮。
  - 证据：styles.css:379-383 的 `[data-state="loading"] .state-loading{display:block}` 没有 flex 增长（对比 styles.css:295-302 空态用 `flex:1 1 auto`）；实测 844 视口：.state-loading 76→264（高 188），#logoutBtn 288→344，下方空白 844-344=500px；normal 态 logout 672.8，empty 态 764；parent/src/pages/settings/index.vue:49-59 在 `v-if="loading"` 时只渲染「加载中...」，logout 位于 `v-else` 内。
  - 建议：给 `[data-state="loading"] .state-loading` 补 `flex: 1 1 auto`（骨架纵向铺满，或与源码一致在 loading 期间隐藏 logout-wrap），保证三种状态下 logout 底部位置稳定、不出现首屏中部悬浮的危险按钮。
- **parent/26-messages** · [logic] 失败态把已加载的消息整块 disabled + pointer-events:none，刷新失败后用户一条旧消息都打不开；源码刻意保留列表可点，notes 却把这个行为说成「对应源码」。
  - 证据：26-messages/styles.css:621-624（[data-state="error"] .inbox-groups{opacity:.45;pointer-events:none}）+ 26-messages/app.js:122（setCardDisabled(state === 'error')）；实测点「失败」chip 后 .msg-card.disabled===true、groups pointer-events:none、点卡片无任何反应。对照 parent/src/pages/messages/index.vue:104-122：失败时只在列表顶部插 inline-error，卡片仍绑 @click、无 disabled。notes.md:16 写「已有列表保留但整块降到 45% 且不可点，对应源码『刷新失败不清空列表』的行为」——「不可点」在源码里没有对应物，「不清空」的理由（还能读）也被抵消。
  - 建议：失败态去掉 disabled 与 pointer-events:none，只在列表上方叠错误条（与 source 的 inline-error 同构）；若坚持锁列表，需在 notes 标为「与源码不同、待产品确认」，并同步把口径行的「3 条未读 / 共 8 条」降级，避免对刚刚失败的数据下断言。
- **parent/28-webview** · [logic] 来源条、画布与脚注被排成同一页面的上下层，但微信 `<web-view>` 会自动铺满整个页面并覆盖其他组件，真实落地时来源条与脚注必然被网页盖住，页面骨架不可实现；notes 把来源条当成可常驻方案，只讨论了与网页标题重复，没有记录这条平台约束。
  - 证据：index.html:37-58 的 source-bar 与 index.html:60-112 的 web-canvas 为同层兄弟节点；源码 parent/src/pages/webview/index.vue:16 只有一个裸 `<web-view>`；notes.md:36-38 只写「重建 web-view（改 key 或重新赋值 src）」，notes.md:46 只讨论来源条与网页标题重复。微信文档：web-view「会自动铺满整个页面，并覆盖其他组件」。
  - 建议：把来源信息前置到入口页或打开外链前的确认弹层，让目标页保持裸 web-view；若坚持同屏，只能改 cover-view 叠加并接受遮挡网页内容，并在 notes「遗留」写明该约束。
- **teacher/01-workbench** · [navigation] 源码底栏的「班级」「我的」两个主入口在原型全站都没有落点，入口链断裂；notes 声称的兜底（其余在「更多」里）与源码不符。
  - 证据：teacher/src/pages/index/index.vue:272-289 的自绘底栏含 /pages/class/index（:277）与 /pages/profile/index（:285）；teacher/src/utils/nav.js:13-18 的 TEACHER_TABS 同 4 项。原型 index.html:78-160 只有 8 个功能入口，铃铛只去消息，无班级/我的入口；teacher/src/pages/more/index.vue:18-29 的 10 个菜单项也不含这两页，故 notes.md:104「其余在「更多」里」不成立。
  - 建议：在问候区头像或「更多」页补「班级」「我的」入口（不必画 tabBar，但不能没有可达路径），并修正 notes 的兜底说明。
- **teacher/02-login** · [logic] 首屏 normal 态协议默认已勾选，与源码 agreed=ref(false) 的「主动勾选」相反；点「微信授权登录」会直接进入登录中并成功，源码里必须通过的同意门禁在默认路径上完全失效。
  - 证据：teacher/src/pages/auth/login.vue:11 `const agreed = ref(false)`、login.vue:62 注释「主动勾选」；02-login/index.html:12 `data-state="normal"`、index.html:78 `aria-checked="true"`；02-login/app.js:22 `CHECKED` 含 normal、app.js:83-89 仅当不在 CHECKED 时才弹同意提示；preview.png 中勾选框为已勾选。notes.md:25 只写「勾选已选」，未登记该差异（对照 parent/02-login/notes.md:62 已明确登记并给出对调方案）。
  - 建议：把 normal 初值改为未勾选（或对调 normal/empty 两枚芯片的语义），仅在勾选后才允许进入 pending；同时在本页 notes「与现状的差异」补一条，与 parent 登录页同口径说明，避免首屏截图被当成真实初值。
- **teacher/03-notice-create** · [logic] 发布成功后既不返回工作台也不复位表单，还把发布按钮重新启用——同一份已发出的通知可以再次点「发布」，造成重复群发。
  - 证据：app.js:478-493 的 confirmPublish 只在 900ms 后 showToast('已发布') 并重新启用 confirmPublish/dialogCancel，没有任何返回或重置；源码 teacher/src/pages/notice/create.vue:94-99 是 toast 后 600ms 执行 navigateBack()（无上一页则 reLaunch('/pages/index/index')），_prompt.md:19 也明确「600ms 后返回上一页」。
  - 建议：成功后按源码模拟返回（toast「已返回老师工作台」）或至少清空并复位表单、保持发布禁用；若为评审方便留页，请在 notes.md 标注「留页复位」并复位成空表单，不能保留可重复发送的已填内容。
- **teacher/03-notice-create** · [design-core] 主按钮、选中类型、错误提示的 token 组合对比度不达 WCAG AA（实测 1.90–3.08:1），主操作文字和选中态在浅暖底上明显偏淡。
  - 证据：styles.css:24 .chip.is-on（--accent-deep #e6762a 压 --accent-wash #fff7f0 = 2.83:1）、:56 .field-error（--danger #EF4444 压 --danger-wash #FEE2E2 = 3.08:1）、:108 .publish-btn（--on-brand #fff 压 --accent-gradient #ffa866→#ff8a3d = 1.90–2.35:1）、:129 .sheet-btn--strong 与 :157 .use-btn 同 accent-deep 组合；色值见 tokens.css:81-91/32-33，brief.md:45 要求「对比度满足 WCAG AA」。
  - 建议：回 tokens.json 调整语义色（加深 accent-deep/danger，或主按钮改用深色字/更深的 accent），不要逐页打补丁；至少保证主按钮文字、选中胶囊文字、错误文案对底色 ≥4.5:1，并把这组 token 对比度记入批次缺口。
- **teacher/04-growth-index** · [logic] 页脚「已显示最近 N 条」用全量 RECORDS.length 而不是当前渲染行数；时间筛选「本月（10月）」只剩 4 条时，页脚仍写「已显示最近 7 条」，搜索时同样失真。
  - 证据：app.js:396 `listFoot.textContent = '已显示最近 ' + RECORDS.length + ' 条 · 下拉刷新'`。DOM 实测选 2026-10 后 `.record-card` 数量为 4，`#listFoot` 文本仍是「已显示最近 7 条 · 下拉刷新」。
  - 建议：用 filteredRows() 的 rows.length 更新页脚，或改成「已加载 7 条 · 当前显示 4 条」，并在筛选/搜索时同步刷新。
- **teacher/04-growth-index** · [logic] 筛选/搜索零命中被并入 data-state=empty，CSS 因此把发布按钮一并隐藏；此时头部统计仍是 2/3，页面一边说「未找到匹配的记录」一边说今天记了 2 条，且屏幕上没有任何发布入口（空态 CTA 变成了「清除筛选」）。
  - 证据：app.js:386-390 零命中时 `app.dataset.state='empty'`；styles.css:1274-1282 在 empty 下 `display:none` 掉 `.publish-btn`；app.js:371-373 把空态 CTA 改成「清除筛选」。DOM 实测中二班筛选后 `#publishBtn` computed display=none，`#statToday/#statPending` 仍为 2/3；源码 teacher/src/pages/growth/index.vue:132-134 的「快速记录」入口始终在 StateBlock 之外可见。
  - 建议：把「服务端空」和「筛选/搜索无结果」拆成两种状态：后者不隐藏发布按钮，统计要么同步当前结果口径，要么明确标注不受筛选影响。
- **teacher/05-growth-create** · [completeness] 源码的编辑态整条分支在原型里完全不存在：从成长详情「编辑」进入时是 update 而非 create，UI 有四处实质差异，原型只画了新建流。
  - 证据：teacher/src/pages/growth/create.vue:59-67,145-154,471-486,771,863-887（?id= 进入编辑态：日期箭头只读、主按钮「保存修改」、隐藏授予徽章、底栏换成只读可见性提示）；05-growth-create/index.html:262-278 只有「发布班级记录」一个底栏，app.js:316-320 也只区分班级/个体两种发布文案；notes.md:8-16 信息层级与「覆盖状态」全表均无编辑态。
  - 建议：补一个编辑态（演示芯片或第二份对照），至少画出日期箭头禁用、底栏可见性只读提示、徽章卡隐藏、主按钮「保存修改」，并在 notes 里说明与新建态的差异。
- **teacher/05-growth-create** · [logic] 「已选幼儿摘要」与「未选幼儿空态」由演示 chip 的 data-state 控制，而不是由真实选择人数控制，导致两条路都自相矛盾：正常流里取消到 0 人看不到空态插画，空态演示里选回 5 人仍显示「还没有选择幼儿」。
  - 证据：app.js:182-208 renderPicked 从不切换 pickEmpty；styles.css:1 用 `.mp-app[data-state="empty"] .picked{display:none}` / `.pick-empty{display:flex}` 控制显隐；05-growth-create/index.html:156,159-177。空态 demo 下从 CTA 选人后 confirmPick（app.js:459-468）只更新 state.selected，不调用 setState('normal')，于是卡头「已选 5 人」与卡体「还没有选择幼儿」+「选择幼儿」按钮同屏；正常流里把 5 人全部取消则只剩「已选 0 人 / 还没有选择幼儿 / 编辑」的空摘要行。
  - 建议：用 selected.size 驱动 picked/pickEmpty 互斥（0 人显示空态 CTA，>0 显示摘要），并把 data-state=empty 与真实选择解耦；confirmPick 后按人数自动切回摘要态。
- **teacher/05-growth-create** · [logic] 「已点名 N 张」不是按点名状态统计，而是写死 min(已完成张数, 2)；默认两张图一张「点名 6」、一张「AI 建议 3」，后者尚未点名，却显示已点名 2 张。
  - 证据：app.js:248 `namedTip.textContent = \`已点名 ${Math.min(done, 2)} 张\``；默认数据 app.js:476-481，角标见 05-growth-create/index.html:198（模板初始值「已点名 2 张」）；源码口径是 `form.media.filter(m => namedCount(m)).length`（teacher/src/pages/growth/create.vue:694），namedCount 只数 student_ids。按此默认态应为 1 张；且新增第 3 张完成后仍会停在 2。
  - 建议：给媒体模型加 named/student_ids 字段，按「有已确认名单的照片数」计数，去掉 min(…, 2)。
- **teacher/05-growth-create** · [logic] 「批量点名」流程走不到任何结果：没有源码的「先勾选照片」步骤，确定后只弹一句成功 toast，照片角标与已点名计数全不变化；单张照片的点名角标也不可点。
  - 证据：app.js:444-447 点「批量点名」直接 openSheet('batch')，无选图；app.js:459-468 batch 分支只 `showToast('已为 N 张照片追加 M 位幼儿')`，不修改 state.media；照片角标由 app.js:226-228 渲染为不可点的 span；源码流程见 teacher/src/pages/growth/create.vue:692-705（全选/选择学生/取消）与 824-847（全班当日到场、AI·高/AI/已有标记）。
  - 建议：还原「批量模式勾选照片 → 选择学生」两段，确定后更新每张照片的点名名单与角标；单张角标加点击入口，或把这些入口明确禁用并在 notes 标注未实现。
- **teacher/06-growth-detail** · [redundancy] 同一「照片张数」在页内出现两个口径：照片区写 9 张，打开任意照片预览却显示「共 8 张」。
  - 证据：index.html:61「9 张」、index.html:69 aria-label「共 9 张」、index.html:379 默认「共 9 张」；app.js:26 的 `$$('.photo-open')` 只有 8 个（第 9 格是 index.html:107 的 `.photo-fail`），app.js:31 `const total = tiles.length`，打开照片后预览显示「第 N 张 / 共 8 张」。
  - 建议：total 取完整媒体数（含加载失败位）而不是可预览按钮数；或把失败位也纳入 tiles。
- **teacher/06-growth-detail** · [logic] 切到「空评论」态后，评论卡标题仍写「（5条）」、互动条仍写「5 条评论」，同时显示「还没有家长评论」，同一状态出现两个互斥口径。
  - 证据：styles.css:986-987 只隐藏 `.comment-list` 并显示 `.comment-empty`；index.html:215「家长评论（5条）」与 index.html:167「5 条评论」不随状态变化；app.js:224-230 的 `setState` 只切 data-state 与 chip；app.js:232-235 仍按 DOM 中 5 个隐藏 `.comment` 计数。
  - 建议：进入 empty 时把 #commentCount 与 #statCommentCount 置 0（或空态真实清空列表），并保证发布首条后计数从 1 起算。
- **teacher/07-pickup-index** · [logic] 手动登记的「选择学生」弹层可以给未到园的孩子登记接走，绕过本页自己置灰的「未到园不能接走」规则
  - 证据：app.js:115-118 对 not_arrived 的「接走」设 aria-disabled=true；但 app.js:238-254 的选人弹层列出全部 10 人（含未到园的 天天），选中后 app.js:362-371 一律 toast「天天 登记成功」。源码 teacher/src/pages/pickup/index.vue:182-185 的「手动登记」只是把筛选切到「全部」，notes.md:31-36 未申报这个改写（_shared/findings.md 接口缺口 #6 只记录了置灰本身，未涉及这条绕过路径）。
  - 建议：选人弹层按状态限制：not_arrived 行标注「需先入园」并不可选；或恢复源码「手动登记=切到全部」的行为并在 notes 说明。确认提交补失败分支（源码 index.vue:131-143 mapPickupError）。
- **teacher/07-pickup-index** · [completeness] 已接走学生行显示契约与源码都不存在的「接走时间 / 接走人」，同时删掉源码每行都有的「授权接送人」
  - 证据：app.js:43-53 为 3 名已接走学生写入 pickedAt/pickedBy，app.js:140-141 渲染为「已接走 17:05 · 李梅（妈妈）」；backend/api/api-teacher/07-pickup.md:70-94 的 students 响应没有 picked_at/picked_by，源码 teacher/src/pages/pickup/index.vue:329-331 对所有行都显示「授权接送人：…」。
  - 建议：回退为「授权接送人」行；若确需接走时间/接走人，标为需要 records 接口补字段或列表 join 的提案，并在 notes「与现状的差异」声明。
- **teacher/07-pickup-index** · [logic] 入园 / 手动确认 / 扫码完成都只弹 toast，列表与概览数字不更新，出现「提示已入园/已接走但卡片状态不变」的自相矛盾
  - 证据：app.js:221-224 点「入园」只 toast「天天 已入园」，卡片仍为未到园、接走仍 aria-disabled；app.js:362-371、412-415 手动与扫码完成同样只 toast，概览仍「待接走 6」。源码 teacher/src/pages/pickup/index.vue:106-108/118-128/161 每个动作后都调用 loadPickup()。
  - 建议：动作后就地更新 STUDENTS 与概览并重渲染，或明确标注为静态演示、不更新数据。
- **teacher/08-pickup-records** · [logic] 失败态与统计条同时成立：页面提示「没取到 10月26日 的接送记录」，但日期卡内统计仍显示 13/4/9 条，等于宣称加载失败却展示完整数据。
  - 证据：app.js:230-235 只在 phase==='loading' 时跳过 updateSummary，error 仍会执行 app.js:249-258；styles.css:584-588 只隐藏 .normal-only，index.html:50-69 的 .date-card/.summary 不在其中；实测 data-state="error" 时 #sumTotal 文本为 "13"，#errorDesc 为 "网络异常，没取到 10月26日 的接送记录，请重试"。
  - 建议：error 态把统计数字隐藏或改为「—」，或把文案改成「显示的是上次成功结果，点重试更新」；不要让「没取到」和 13 条同时出现。
- **teacher/09-tasks-index** · [completeness] 「范围」只剩 7 个硬编码 pill（班级 + 6 个孩子），源码是从接口全量渲染的本班学生（本页自设 CLASS_SIZE=30），且没有「学生名单加载失败，点击重试；恢复前只能创建班级任务」分支。按真实班级 30 人算，多数孩子无法被指派个体任务。
  - 证据：index.html:197-204 仅 班级/乐乐/小雨/朵朵/果果/豆豆/米米；app.js:55 CLASS_SIZE=30；teacher/src/pages/tasks/index.vue:32、211-220、309-312；backend/api/api-teacher/09-class-and-tasks.md:68「一次全量返回（无分页）」
  - 建议：改用底部 picker/action sheet 承载完整学生列表（可滚动或带搜索），「班级」固定为首项；补学生名单加载失败的提示与重试；notes 记录该差异。
- **teacher/09-tasks-index** · [notes-accuracy] notes 声称覆盖「星星罐未配置时『按班级』置灰」，但该状态不可达也不可见：jar.goal 初始为 60 且保存只接受 >0；.jar-empty-note 恒为 display:none（无任何显示规则），.is-unset 无对应 CSS，保存按钮永远写「更新目标」而非源码按 configured 切换的「设置目标」。
  - 证据：notes.md:17；app.js:58、208、456-465；index.html:158、161；styles.css:1（`.jar-empty-note{display:none…}` 且无 `.jar-card.is-unset` 规则）；teacher/src/pages/tasks/index.vue:348-359
  - 建议：演示条增加「未配置」态或提供切换；补 `.jar-card.is-unset .jar-live{display:none}` 与 `.jar-empty-note{display:block}`；按钮文案按 jarConfigured 切换；同步修正 notes。
- **teacher/10-tasks-detail** · [logic] 朵朵卡片提交时间 10月24日 18:40 处于任务周期 10月20日~10月26日 之内，却被标记「旧周期 · 请逐条回应」，与源码 isCurrentPeriod 的判定相反，周期提示与日期自相矛盾。
  - 证据：index.html:51 周期「10月20日 ~ 10月26日」；index.html:196 朵朵时间「10月24日 18:40」；index.html:199「旧周期 · 请逐条回应」；源码 detail.vue:72-75 isCurrentPeriod 仅在 row.date 落在 period_start/period_end 之外时为旧周期。
  - 建议：把朵朵的日期改到周期外（如 10月18日）以演示旧周期提示，或删除该卡的旧周期标记，确保样例数据与周期判定一致。
- **teacher/10-tasks-detail** · [logic] 编辑弹层里确认「关闭公开」后，页面立即写入 page.dataset.public="off"，随后点「取消」或关闭弹层不会回滚；破坏性改动脱离了表单的保存/取消语义，与源码「先暂存、保存时才提交」不一致。
  - 证据：app.js:214-223 关闭公开的 dialogAction 直接执行 page.dataset.public = "off"；app.js:272-279 与 index.html:539 的「取消」只调用 closeSheet，不恢复状态；源码 detail.vue:478-501 仅在确认后改 edit.allow_public，真正提交发生在 saveEdit。DOM 实测：确认关闭→取消后 page.dataset.public 仍为 off，重开弹层开关 aria-checked=false。
  - 建议：将 allow_public 改为弹层内暂存值，仅在点「保存」时应用到页面状态；「取消」或关闭弹层时恢复原公开状态。
- **teacher/10-tasks-detail** · [logic] 保存编辑时只校验任务标题，「学习关联」清空后仍能保存成功，导致 hero 上「学习关联」标签保留而内容为空；源码要求标题与学习关联同时必填。
  - 证据：app.js:316-323 只判断 title 非空；源码 detail.vue:512-515 同时校验 edit.title 与 edit.learning_link；index.html:55 的 .hero-learning-text 会被 app.js:326 写成空串。DOM 实测：清空 #taskLinkInput 后点保存，toast 显示「已保存」，.hero-learning-text 变为空。
  - 建议：保存前增加 learning_link 非空校验并就地提示；若不希望显示空字段，至少在 hero 中隐藏空的「学习关联」行。
- **teacher/10-tasks-detail** · [completeness] 「补充回应」入口只在果果卡出现；一诺（语音回应）与可心（班级星星回应）同样有主回应，却完全没有补充回应入口，而源码对所有 responded 且 row.response 存在的行都渲染补充区。
  - 证据：index.html:296-315 果果卡有 .supplement-toggle；index.html:359-366（一诺）与 index.html:378-382（可心）在 .response-block 后直接结束，无 data-supplement-toggle；源码 detail.vue:1065-1079 条件为 status === 'responded' && row.response。DOM 实测 responded 列表 3 卡中仅 1 卡存在补充入口。
  - 建议：为所有存在主回应的已回应卡补上「补充回应（个别夸一句）」入口，保持源码的全量可用性。
- **teacher/10-tasks-detail** · [redundancy] 四列统计里的「3 待回应 / 3 已回应」与紧邻分段的「待回应 3 / 已回应 3」是同一组数字重复；统计、分段、审核筛选、批量条四层连续堆叠，390 宽下第一条家长提交在约 633px 处才开始，首屏看不到任何提交内容，与「一页看清谁交了」的目标相悖。
  - 证据：index.html:63-80 四列统计，index.html:82-89 紧接着的分段再次显示 3/3；index.html:91-104 还有审核 chip 与批量条。DOM 实测：.mp-app 宽度 390px，.list[data-list="pending"] top=633px（844px 视口），第一张 .checkin top=633px。
  - 建议：合并统计与分段（计数只保留一处），把审核筛选收进折叠筛选或与分段合并，保证首屏露出至少一条提交卡片。
- **teacher/11-calendar-index** · [logic] 失败态仍显示月份条的派生统计「本月 11 个活动」，与错误块「未获取到本月活动」正面矛盾；加载态也在数据未到达前先报出具体数量，且错误态下月份箭头仍可点击、把月份文字和统计一起改掉而错误块不变。
  - 证据：index.html:45 静态「本月 11 个活动」；app.js:81-84 renderMonth 只对 empty 特判，error/loading 仍输出真实 count；styles.css:561-566 error 只隐藏 cal-card/day-skeleton/day-section，.month-bar 保留；index.html:121「网络异常，未获取到本月活动。」；app.js:257-265 的 shiftMonth 在 error 态同样重渲染 monthLabel/monthSub（如 2026年11月 / 本月 2 个活动）。
  - 建议：error 态把副标题改为「—」或「活动加载失败」，或整条 month-bar 只保留月份与箭头；loading 期间同样不输出具体计数；error 态禁用月份箭头，直到重试成功。
- **teacher/12-calendar-create** · [completeness] 源码 create.vue 同时承担「编辑已有活动」：详情页以 `?id=` 进入后拉详情预填、已发布活动点发布走变更确认弹层、发布设置换成 notify-hint、主按钮变「保存发布」；原型只画了纯新建态，编辑分支 0 覆盖。
  - 证据：teacher/src/pages/calendar/detail.vue:36 `navigateTo({ url: `/pages/calendar/create?id=${event.value.id}` })`；teacher/src/pages/calendar/create.vue:154-179（onLoad 读 id + fetchTeacherCalendarEvent 预填）、140-152（published 时走 substantiveLines + confirmVisible）、233-239（notify-hint）、249-269（变更确认 sheet）、244（editId ? "保存发布" : "立即发布"）；原型 index.html:33 标题固定「添加活动」、index.html:185-187 按钮固定「发布活动」，app.js:1 无 location.search / id / editId 任何处理。
  - 建议：补一个「编辑已发布活动」演示态：预填字段 + 发布设置里的 notify-hint + 点发布后弹出变更清单与通知开关的确认 sheet，主按钮文案切「保存发布」；若有意不做，需在 notes 中明确标注编辑态不在范围并说明原因。
- **teacher/12-calendar-create** · [logic] 「必填未完成时常驻阻断说明」实际只在首次点发布被拦之后才生效：用户清空必填标题时，底部摘要仍显示完整发布预览、发布按钮仍为可用态，与 notes 的「常驻」声明相反。
  - 证据：app.js:1 `bt()`/`pd()` 都以 `rq` 为前提（`var missTitle=rq&&mt()`、`if(rq&&(mt()||md()))return true`），而 `setMode('normal')` 把 rq 置 false；实测清空 #event-title 后 #submit-summary 仍为「活动 · 10月30日 09:00–11:30 · 通知家长」、#btn-publish aria-disabled="false"、#title-error 仍 hidden，点一次发布才变成「还差标题」+ aria-disabled="true"；notes.md:13、:33 声称「常驻阻断说明」。
  - 建议：必填/时间判定改为无条件计算，`rq` 只控制错误文案是否显示；或在 input/change 时把 rq 置 true，保证摘要阻断与按钮禁用随字段实时更新。
- **teacher/13-calendar-detail** · [notes-accuracy] notes.md 多处声明与四件套实际不符：声称信息卡含「参与班级与人数摘要、提醒设置」（实际是 日期/时间/地点/参与范围，页面无人数、无提醒）；声称「编辑（底部弹层）」（实际编辑只弹 toast，页面无编辑弹层）；状态覆盖列了不存在的「已取消/已删除」降级态，却漏掉了实际存在且是核心差异的「草稿」「全园只读」两态。
  - 证据：notes.md:6、notes.md:11、notes.md:14、notes.md:23、notes.md:40；index.html:87-124 仅有 4 个 .info-row（活动日期/活动时间/活动地点/参与范围），全页无「人数/提醒」文本（grep 无命中）；app.js:288 编辑按钮只 toast('打开编辑活动')；index.html:193-198 的 6 个 chip 为 正常/空态/加载/失败/草稿/全园只读，app.js:100 DATA_STATES 中无 cancelled/deleted。
  - 建议：按实际实现重写 notes.md：信息卡字段改为 日期/时间/地点/参与范围；编辑写明「toast 占位，真实实现跳 pages/calendar/create?id=」；状态列表补「草稿」「全园只读」，删除「已取消/已删除」（源码同样没有该状态）。
- **teacher/14-leave-index** · [logic] 状态分段条的「滚动吸顶」不生效：sticky 声明存在，但滚动后过滤器随页面上移消失。
  - 证据：styles.css:106-113 声明 position: sticky; top: 64px；实测 scrollTo(0,250) 后 .filter-sticky 的 rect.top=-72px，scrollTo(0,600) 后 -422px（应停在 64px），同期 .mp-navbar 稳定在 20px。根因是 .mp-main 计算 overflow-y:auto（shell.css:130-134）成为最近滚动容器却不滚动（clientHeight=scrollHeight=1628）。notes.md:10 宣称「滚动吸顶」。
  - 建议：让 sticky 相对文档滚动（把过滤器移出 .mp-main，或让 .mp-main 真正滚动并把 top 改为 0）；在 od-check 中增加 390 视口滚动后 filter 顶部位置断言。
- **teacher/14-leave-index** · [logic] 统计卡口径标注错误：整卡标为「本月请假统计 / 统计为本月全量」，但后端只有「本月总计」按月，待审批/已同意/已拒绝是班级全量；分段计数又只统计当前 20 条，三处数字在真实数据下会互相矛盾。
  - 证据：index.html:40（sr-only 标题「本月请假统计」）、index.html:59（「统计为本月全量 · 列表显示最近 20 条」）；server/internal/service/teacher_portal_service.go:732-742 仅 monthly_total 用 monthStart 过滤；app.js:229-239 的分段计数来自当前 records（8 条）；契约 backend/api/api-teacher/05-leaves.md 的 stats 响应里也只有 monthly_total 带月份语义。
  - 建议：卡片标题改为「请假统计」，只给第四项保留「本月总计」；分段计数改读服务端 ?status= 的 total，或明确标注「最近 20 条内」；删除误导性脚注。
- **teacher/14-leave-index** · [logic] 合并成单列后只按提交时间倒序，待审批不置顶；旧的待审批申请会被近期已审批挤出第一屏，与设计目标相反，也弱于源码的独立待审批区块。
  - 证据：app.js:243-247、274 只按 createdAt 排序，未按 status 优先；源码 teacher/src/pages/leave/index.vue:112-159 用独立「待审批申请」区块保证可见；notes.md:28 宣称「把「现在要处理什么」提到第一屏」。示例数据里 3 条待审批恰好是最新 3 条（app.js:40-42），掩盖了该问题。
  - 建议：全部视图按「待审批优先、再按提交时间倒序」排序，或默认筛选落在「待审批」；列表头写明排序规则。
- **teacher/15-class-index** · [navigation] 导航栏返回箭头是死控件，点击完全无反应
  - 证据：teacher/15-class-index/index.html:22 `<button class="mp-back" aria-label="返回老师工作台">`；app.js 全文无 .mp-back 查询（grep 无命中），_shared/shell.js:9-27 只绑定演示条；同批 teacher/14-leave-index/app.js:484-486 等页都绑了 toast，说明是本页漏绑
  - 建议：在 app.js 绑定 .mp-back → toast「返回老师工作台」或回退演示；本页也可从「我的」进入（teacher/src/pages/profile/index.vue:47），aria-label 宜改为「返回」
- **teacher/15-class-index** · [logic] 「接送截止」被画成按学生取值，同一班出现 16:30/17:00 两种；后端该字段是班级级单一值，等于凭空造出按人截止能力
  - 证据：index.html:95/103/114/120/123 五人 data-deadline="17:00"，其余 25 人 16:30；server/internal/service/teacher_portal_service.go:1991 只取一次 classPickupDeadline，:2116 对每个学生写同一个 ResolvePickupDeadline(deadline)，ClassStudents 直接复用该接口（:2806-2808）
  - 建议：全班统一一个 deadline（示例 16:30），或删掉信息卡「接送截止」行；若产品确需按人定制，先改 backend/api + service
- **teacher/15-class-index** · [design-core] 筛选胶囊 36px、文本按钮 40px，低于 brief 规定的 44×44 触控下限
  - 证据：teacher/15-class-index/styles.css:286 `.filter { min-height: 36px }`；styles.css:401 `.link-btn { min-height: 40px }`；_shared/brief.md:45「触控目标 ≥ 44×44px」
  - 建议：两处改为 min-height: var(--input-height)（44px）
- **teacher/15-class-index** · [design-core] 同一「已接走」状态在概览是蓝色(info)、在幼儿卡和信息卡是绿色(success)，语义色一页两套
  - 证据：index.html:72 概览「已接走」用 dot--info（styles.css:208 → --info）；styles.css:379 卡片 picked_up 点用 --success；styles.css:720-722 信息卡 picked_up 用 --success；preview.png 实测概览为蓝点、卡片为绿点
  - 建议：概览/卡片/信息卡统一同一 token（建议已接走=--info、已入园=--success），并同步 notes 的语义色映射
- **teacher/15-class-index** · [completeness] 源码末尾「班级动态」+「最新通知（含已读率）」整块未画，notes 称归消息页，但消息页原型没有任何已读率展示，该数据视图在整套原型中失去落点
  - 证据：teacher/src/pages/class/index.vue:148-164（三快捷入口 + notice-card `已读 {{ notice.readRate }}%` 进度条）；notes.md:37 称「分别是工作台与消息页职责」；teacher/18-messages-index/index.html:107 只有「全部已读」，grep teacher/*/index.html 无任何 read_rate/已读率展示；工作台只有未读计数与一条待办（01-workbench/index.html:72,191）
  - 建议：在名单上方补「最新通知」两条 + 已读率条，或把已读率明确落到通知详情/消息页并在 notes 标注需产品确认
- **teacher/16-album-index** · [redundancy] 同一页两组册数口径不一致：统计卡写「相册数 12」，列表两个月份组只有 4+2=6 册，且全页没有任何「加载更多/查看全部」入口或「已展示 6/共 12 册」的说明，用户无法触达另外 6 册。
  - 证据：index.html:82 相册数 12；index.html:98「4 册」、index.html:156「2 册」；DOM 实测 .album-card 共 6 张、月份计数为 ["4 册","2 册"]；notes.md:41/50 自认「列表只展示 12 册中的最近 6 册」「之外的 6 册没有入口」。
  - 建议：列表尾部补「查看全部 12 册 / 加载更多」入口并让月份组覆盖全部相册；或把统计改为「已展示 6 / 共 12 册」，避免同页两个总数打架。
- **teacher/16-album-index** · [logic] 大图弹层打开时背景没有真正锁定滚动：app.js 加的 body.is-locked 在本页 CSS 里没有任何规则（同批次 parent/11-growth-album/styles.css:39 有 body.is-locked{overflow:hidden}，本页没有），同时 app.js:49 的 main.scrollTop=0 对一个非滚动容器空转，所以弹层开着时页面仍可滚动，关掉后页面停在漂移后的位置。
  - 证据：app.js:146/161 add/remove('is-locked')；styles.css 全文无 is-locked 规则（DOM 内所有 styleSheet 扫描结果为无匹配）；实测弹层打开后 body 计算样式 overflow=visible，window.scrollTo(0,400) 后 scrollY=400；.mp-main 的 scrollHeight===clientHeight（1562），scrollTop 恒为 0。notes.md:23 称「大图弹层开/关…全部通过」但未覆盖背景滚动。
  - 建议：在 styles.css 补 body.is-locked{overflow:hidden}（或改用 .mp-main 滚动容器并锁它），状态切换时只保留 window.scrollTo(0,0)，删掉无效的 main.scrollTop。
- **teacher/18-messages-index** · [logic] 「全部已读」的启用判定按全量未读，点击守卫却按当前分类未读，导致按钮可点但静默无反应
  - 证据：app.js:312 `var unavailable = countUnread(messages) === 0;` 与 app.js:381 `if (countUnread(messagesFor(currentFilter)) === 0) return;` 作用域不一致。实测：切到「通知」并点掉唯一未读后，摘要为「共 5 条 · 0 条未读」，按钮 disabled=false、aria-disabled=false，点击无新 toast（仍停留在上一次「查看：秋游亲子活动通知」），切回「全部」仍为「共 12 条 · 3 条未读」。notes.md:21 声称按全量未读判定，与实现不符。
  - 建议：把守卫改为 countUnread(messages) === 0，与禁用条件同一作用域并执行全局置读；或把按钮改为仅作用于当前分类并改文案（如「本类已读」），同时让摘要与动作口径一致。
- **teacher/18-messages-index** · [logic] 空态/失败态仍显示分类分段及其条数，与「暂无消息」「消息加载失败」直接矛盾
  - 证据：index.html:87-102 的 .inbox-toolbar 位于 .view.view-normal（index.html:104 起）之外，styles.css:426-437 只隐藏 .view-normal；实测 data-state=empty 与 error 时 .inbox-toolbar 仍 display:block 且可见，文案为「全部12 通知5 审批4 系统3」，同屏显示「暂无消息」/「消息加载失败」。notes.md:18 却称「整页空态」。同批 09-tasks-index/styles.css 在 empty/error/loading 下会隐藏 .filter-count，本页未做。
  - 建议：把 .inbox-toolbar 移入 .view-normal，或在 empty/error/loading 下隐藏 .seg-count 并禁用分段切换；没有数据时不应展示已加载的条数。
- **teacher/19-profile-index** · [logic] 空态唯一主行动「刷新资料」是死动作：点击只弹 toast「资料已刷新」，页面仍停在 empty，园所仍为「未绑定园所」、三个数字仍为「—」，构成虚假成功反馈。
  - 证据：index.html:93 的 #refresh-profile 只在 app.js:104-108 里 toast("资料已刷新")；实测点击后 .mp-app[data-state] 仍为 empty，.stat-empty 仍 display:flex，#refresh-profile 仍在，notes.md:59 亦承认真实行为待定。
  - 建议：让刷新模拟一次成功取数（切回 normal 并更新园所/统计），或改成诚实的「请联系园所管理员绑定」提示/禁用态；不要用「已刷新」掩盖未变化。
- **teacher/19-profile-index** · [design-core] 强调色/危险色文字在浅色底上的对比度系统性不足：待审批「3」2.50:1、空态「刷新资料」与失败态「重试」2.50:1、退出登录/确认按钮 3.62:1、「开发中」徽标 4.01:1；20px/600 的「3」连大字号 3:1 都达不到，12–14px 文本低于 AA 4.5:1。
  - 证据：styles.css:181-182（pill 底/字）、266-268（btn-soft）、362-369（badge）、387（logout）、490（dialog danger）；tokens.css:32（--danger #EF4444）、81-91（teacher --accent-deep #e6762a、--surface #fffaf0）；实测 2.50 / 2.50 / 3.62 / 4.01；brief.md:45 要求对比度满足 WCAG AA。
  - 建议：在 token/派生层统一修：tint 上的文字改用 --fg 或把 accent-deep 加深到 ≥4.5:1；danger 文本用更深的 danger 文本 token（或 fg 文字 + danger 图标）；badge 用 --fg-2；用对比度工具复测，不要逐页打补丁。
- **teacher/20-doctor-checkin** · [logic] 空态「回看已完成」与已渲染内容自相矛盾：进入空态时 18 条完成记录和 completedCount=18 已经全部显示，点击后仍是 18 条，只是把 5 条原本待检的记录打上「新完成」并提示「已补齐 5 条完成记录」；再点一次提示「已完成记录已置顶」但顺序不变。空态行由静态 SEED 生成，不读 records，提交过的结论会被重置；空态统计也写死 18/0/18。
  - 证据：app.js:141-152 emptyRows() 用 SEED，!emptyReviewed 时直接返回全部 18 条；app.js:318-333 renderCompleted() 在空态渲染 emptyRows() 并给 wasPending 行加 is-fresh；app.js:525-536 首次点击 toast「已补齐 5 条完成记录」，二次点击「已完成记录已置顶」；app.js:181 空态统计硬编码 {18,0,18}。运行时空态点击前 #completedCount=18、#completedList li=18、.is-fresh=0；点击后仍 18 条、.is-fresh=5。
  - 建议：空态不要预先渲染完成列表，改为点击后再渲染；或把 CTA 改成「查看已完成」并删掉「补齐/置顶」文案；emptyRows 改为从 records 派生，renderStats 不写死 18/0/18。
- **teacher/20-doctor-checkin** · [redundancy] 园级事件入口被从源码的筛选栏上方移到页面最底部，位于 13–18 条已完成记录之后；它是进入 pages/doctor/calendar 的唯一入口，正常态距页顶约 1344px、空态约 1394px，首屏 844px 内完全不可见，等于把唯一跳转入口埋掉，且 notes 的信息层级把它列为第 6 项、差异清单未提这次重排。
  - 证据：源码 teacher/src/pages/doctor/checkin.vue:207-212 garden-entry 在 filter 214-225 之前，goGardenCalendar 在 :185；原型 index.html:113-120 eventEntry 在 completedSection 103-111 之后；运行时 #eventEntry 绝对 top 正常态 1344、空态 1394，视口 844；notes.md:15 把入口列为第 6 项，notes.md:34-44 未提重排。
  - 建议：移回今日卡下方、班级筛选上方；若产品坚持置底，至少在首屏给快捷入口，或把入口并入今日卡并在 notes 标注这次重排。
- **teacher/20-doctor-checkin** · [design-core] 关键操作与状态文字对比度远低于 WCAG AA：主按钮白字压暖橙渐变约 1.90–2.35:1；「需观察」「劝返」和三档状态标签用语义色压同色浅底约 2.74–3.08:1；「待晨检」约 2.83:1，政策胶囊约 4.24:1，字号仅 11–14px。低对比是品牌方向，但这些文字承载操作结果与幼儿状态，已低于可读阈值。
  - 证据：styles.css:104 .submit-primary、:107-109 .submit-warn/.submit-danger、:132 .primary-btn；styles.css:49 .child-check、:150-157 .status-tag、:13 .policy-pill；tokens.css:18 --on-brand:#FFFFFF、:24 默认 accent-gradient #FB923C→#F97316、:89 teacher 渐变 #ffa866→#ff8a3d、:28-35 语义色。按实际色值计算：白字压 #ffa866=1.90、压 #ff8a3d=2.35；#16A34A/#DCFCE7=3.00；#CA8A04/#FEF9C3=2.74；#EF4444/#FEE2E2=3.07；#e6762a/#fff7f0=2.83；#2563EB/#DBEAFE=4.24。
  - 建议：状态文字改用 --fg/--fg-2 压浅底，或把语义色加深到 ≥4.5:1；主按钮改深色字或加深渐变；在 tokens.json 统一修，不要逐页打补丁。
- **teacher/21-doctor-calendar** · [logic] 变更确认弹层的「发布变更通知」开关不参与保存，用户即使关掉，保存后仍按事件原有 notify 状态通知家长。
  - 证据：app.js:664 只用 setSwitch(changeNotifySwitch, suggestNotify) 做视觉预置；saveConfirm()（app.js:708-724）直接 applyEventChanges(event, pendingEdit.data)，而 readFormData() 在编辑态把 notify 固定为 editingBaseline.notify（app.js:582-584），全文件没有任何保存路径读取 changeNotifySwitch；源码 calendar.vue:140 是 save("published", confirmNotify.value)。示例事件如 evt-makeup-checkup 为 notify:true（app.js:84），改标题时开关预置为关，保存后 event.notify 仍为 true。
  - 建议：saveConfirm() 读取 isSwitchOn(changeNotifySwitch) 并写回 pendingEdit.data.notify 后再 applyEventChanges；补「关掉开关后 event.notify 为 false」的自测。
- **teacher/21-doctor-calendar** · [logic] 编辑草稿时「通知家长」开关被一并隐藏，且保存固定取 baseline，草稿发布会默认推送家长；源码对草稿显示开关且默认不推。
  - 证据：openEditSheet() 无条件执行 notifyRow.hidden = true（app.js:462）；源码 calendar.vue:245-251 只在 editId && original?.status === 'published' 时隐藏，草稿走 v-else 显示 notify-row；startEdit 把 notify 置 false（calendar.vue:122-125），草稿 submitPublish 使用 form.value.notify（calendar.vue:150-166）。原型里示例草稿「小班补检」为 notify:true（app.js:77-90），编辑后发布仍为 true。notes.md:41 却写「新建 / 草稿用表单内开关」，与实现不符。
  - 建议：改为 notifyRow.hidden = event.status === 'published'；草稿编辑时允许开关交互，并让 readFormData 在草稿编辑分支读取 isSwitchOn(notifySwitch)，已发布编辑才使用 baseline。
- **teacher/21-doctor-calendar** · [completeness] 编辑态底部没有「存为草稿」，编辑草稿只能直接发布，已发布事件也无法保存回草稿。
  - 证据：setFooterMode('edit') 同时把 saveDraftBtn.hidden 与 publishNowBtn.hidden 设为 true（app.js:401-407）；源码编辑态始终渲染「存为草稿」，submitDraft() → save("draft", false) 在 editId 存在时走 update（calendar.vue:122-125、252-255）。
  - 建议：编辑态 footer 保留「存为草稿」（取消编辑 / 存为草稿 / 保存发布三键），点击时以 draft 状态保存并关闭弹层。
- **teacher/22-webview** · [design-core] 页内地址条/重新加载/容器脚注与微信 `<web-view>` 全屏覆盖的硬约束冲突：真实 web-view 会自动铺满整个页面并盖住页内其他元素，原型却把地址卡（`index.html:37-55`）、网页容器内的加载/失败面板（`57-100`）、脚注（`102-105`）画成与网页画布同页共存；真实实现时这些 UI 会被 web-view 覆盖。地址条本身也是浏览器语汇，与批次「一眼看出这是小程序、不是移动网页」的约束相悖。
  - 证据：源码 `teacher/src/pages/webview/index.vue:13` 以 `<web-view>` 作为整页唯一内容；原型 `docs/design-previews/2026-10-25-full-redesign/teacher/22-webview/index.html:37-55`（地址条）、`:57-100`（web-card 内加载/失败面板）、`:102-105`（脚注）均在同一页面；`notes.md:38-40` 只把地址条/脚注标为「新增、需产品确认」，未说明 web-view 全屏覆盖导致其不可实现。家长端 28-webview 同构（A38 已记来源条重复），同一约束同样适用。
  - 建议：删除页内地址条、刷新按钮与容器脚注，只保留「容器占位 + 空态」；若必须让用户知道链接，改由原生导航栏/入口页承担，并保持源码 `<web-view>` 独占页面。若产品坚持地址条，需把实现方案从 `<web-view>` 改成自绘 H5 容器，并在 notes 明确标注该平台限制。

## 3.1 P1 修复记录（2026-10-27）

- 覆盖：44 个含 P1 的页面，共 85 条已修、8 条暂缓。
- 每页修完后均通过 `od-check`（`problems: []`）、重出 `preview.png`、本页 `od-audit` 无 ✗；单页记录已追加到 `_shared/reviews/<page>.md`。
- 暂缓项均为**批次级 token / 契约 / 跨页入口**问题，不能在单页目录内合规解决。

| 页面 | 结果 | 已修 | 暂缓 |
|---|---|---|---|
| parent/01-home | fixed | 2 | 0 |
| parent/02-login | fixed | 1 | 1 |
| parent/03-bind | fixed | 2 | 1 |
| parent/05-notice-detail | fixed | 1 | 1 |
| parent/06-leave-index | fixed | 1 | 0 |
| parent/08-leave-detail | fixed | 3 | 0 |
| parent/09-growth-index | fixed | 1 | 1 |
| parent/10-growth-detail | fixed | 2 | 0 |
| parent/11-growth-album | fixed | 1 | 0 |
| parent/12-growth-report | fixed | 2 | 0 |
| parent/13-pickup-index | fixed | 2 | 0 |
| parent/14-pickup-qrcode | fixed | 2 | 0 |
| parent/15-pickup-authorized | fixed | 1 | 0 |
| parent/16-pickup-authorized-create | fixed | 1 | 0 |
| parent/18-tasks-detail | fixed | 4 | 0 |
| parent/19-tasks-wall | fixed | 1 | 0 |
| parent/20-points | fixed | 2 | 0 |
| parent/22-calendar-index | fixed | 2 | 0 |
| parent/23-student-index | fixed | 2 | 0 |
| parent/24-feedback | fixed | 3 | 0 |
| parent/25-settings | fixed | 1 | 0 |
| parent/26-messages | fixed | 1 | 0 |
| parent/28-webview | fixed | 1 | 0 |
| teacher/01-workbench | fixed | 1 | 0 |
| teacher/02-login | fixed | 1 | 1 |
| teacher/03-notice-create | partial | 1 | 1 |
| teacher/04-growth-index | fixed | 2 | 0 |
| teacher/05-growth-create | fixed | 4 | 0 |
| teacher/06-growth-detail | fixed | 2 | 0 |
| teacher/07-pickup-index | fixed | 3 | 0 |
| teacher/08-pickup-records | fixed | 1 | 0 |
| teacher/09-tasks-index | fixed | 2 | 0 |
| teacher/10-tasks-detail | fixed | 5 | 0 |
| teacher/11-calendar-index | fixed | 1 | 0 |
| teacher/12-calendar-create | fixed | 2 | 0 |
| teacher/13-calendar-detail | fixed | 1 | 0 |
| teacher/14-leave-index | fixed | 3 | 0 |
| teacher/15-class-index | fixed | 5 | 0 |
| teacher/16-album-index | fixed | 2 | 0 |
| teacher/18-messages-index | fixed | 3 | 0 |
| teacher/19-profile-index | fixed | 1 | 1 |
| teacher/20-doctor-checkin | fixed | 2 | 1 |
| teacher/21-doctor-calendar | fixed | 3 | 0 |
| teacher/22-webview | fixed | 1 | 0 |

### 暂缓明细（8 条）

- **parent/02-login** · P1-1 [design-core] 主按钮白字压品牌橙渐变实测仅 2.26–2.80:1，15px/600 属正文级，低于 WCAG AA 4.5:1
  - 原因：批次级：修复点在 token 层（shared-design-tokens/tokens.json 的 --on-brand 或 --accent-gradient），并须同步 teacher 端；审查建议明确「不要逐页改按钮颜色」，本页目录内单独改会造成跨端不一致，已登记 notes.md「遗留」第 2 条等待批次统一处理。
- **parent/03-bind** · P1-3 [navigation] pages/auth/bind 在家长端无任何入站入口（源码无 navigateTo 指向，App.vue 不处理场景值）
  - 原因：入口须加在 23-student-index / 27-profile 等其他页面目录，按任务约束只改本页目录不得越界，属批次级统一处理；已在 notes.md 遗留项 7 记录建议入口位置与文案
- **parent/05-notice-detail** · [P1-2 completeness] 原型纸面的类型胶囊、发布人、附件区、已读时间、已确认时间共 5 处内容没有数据源，真机打开会是空标签/空作者/无附件/无时间
  - 原因：缺口在 server/internal/service/parent_portal_service.go:1190-1196 的 GetNotice 未按契约返回 category_label / creator / attachments / read_at / confirmed_at（契约 backend/api/api-parent/02-notice.md:46-75 已定义，附件仓储 ListAttachmentsByNoticeID 也已存在）。补齐需修改 server/ 后端实现，超出本页范围且本批次禁止改源码；设计保持契约目标形态，已在 notes.md 新增「依赖未实现字段」一节标注该依赖与真机表现，并注明 published_at/read_at 需前端本地时区格式化。
- **parent/09-growth-index** · 视角筛选为空状态在原型演示条中不可一键直达（属于状态可达性，不影响该状态本身的实现）。
  - 原因：当前 fixture「只看孩子」恒为 5 条，该分支只能由数据触发；新增第五个演示芯片会偏离批次统一的四态芯片规范（批次级），本页不单独扩，已在 notes.md「遗留 / 需产品确认」注明。
- **teacher/02-login** · 批次级统一（非本页 P1）：parent/02-login 首屏仍停在「正常」（已勾选），与本页首屏「空态」（未勾选）呈现相反；两页芯片语义已一致（正常＝已勾选、空态＝未勾选），仅默认芯片不同
  - 原因：只能改本页与审查清单，parent 页面不在本次范围内；parent 登录 notes 已给出 normal/empty 对调方案，是否全批次统一首屏芯片需批次 owner 决定（批次级）。
- **teacher/03-notice-create** · P1-2 [design-core] 主按钮、选中类型胶囊、错误提示的 token 组合对比度仅 1.90–3.08:1，不达 WCAG AA
  - 原因：批次级 token 问题：三组都在 token 层（--on-brand #FFFFFF 压 --accent-gradient #ffa866→#ff8a3d = 1.90–2.35:1；--accent-deep #e6762a 压 --accent-wash #fff7f0 = 2.83:1；--danger #EF4444 压 --danger-wash #FEE2E2 = 3.08:1），需改 shared-design-tokens/tokens.json 的 onBrand / brand.primaryDark / brand 渐变 / danger 并重建 tokens.css。tokens.css 是 sync-open-design-adapter.mjs 的生成产物（文件头「Do not edit by hand」）且 50 页副本逐字节相同（md5 c03f8da…），逐页改色会让本页与其余 49 页脱钩，审查也明确要求「不要逐页打补丁」；本页目录内无合规解法，已登记 notes.md 遗留 7。
- **teacher/19-profile-index** · P1-2 [design-core] 强调色/危险色文字在浅色底上对比度系统性不足（待审批「3」与「刷新资料/重试」2.50:1、「开发中」徽标 4.01:1、退出登录与确认框危险按钮 3.62:1）
  - 原因：批次级，本页目录内无合规解法。实测复核与审查一致，成因全在 token：--accent-deep(#e6762a) 压 accent 派生浅底、--danger(#EF4444) 压 --surface(#fffaf0)、--muted 压 neutral 浅底。(a) tokens.css 由 scripts/sync-open-design-adapter.mjs 从 shared-design-tokens/tokens.json 生成，文件头禁止手改；(b) 危险红只有 --danger 一个 token，无深色文本变体，页内自造深红即新增裸色值（od-audit 按裸色值判失败）；(c) 其余三处若改 --fg/--fg-2，本页「待审批」胶囊、软按钮、「开发中」徽标会与同批次 teacher/01-workbench、07-pickup-index、20-doctor-checkin 同款组件不一致，正是审查与 _shared/REVIEW.md §6.4 禁止的逐页打补丁。需回 tokens.json 一次性加深 accent-deep、补 danger 文本 token 并重跑三端产物后统一复测；已写入本页 notes.md 遗留。
- **teacher/20-doctor-checkin** · P1-3 [design-core] 关键操作与状态文字对比度远低于 WCAG AA：主按钮白字压暖橙渐变 1.90–2.35:1；「需观察」「劝返」与三档状态标签 2.74–3.08:1；「待晨检」2.83:1；政策胶囊 4.24:1，字号仅 11–14px
  - 原因：批次级 token 问题，本页无法单独定：这些颜色全部来自 tokens.css 的全局 token（--accent-gradient / --success / --warning / --danger / --info 及其 wash，由 shared-design-tokens/tokens.json 生成并下发三端）。审查建议与 _shared/REVIEW.md §6.4 都要求回 tokens.json 一次修、不要逐页打补丁；本页任务约束也禁止手改生成产物、禁止改 _shared/ 与其他页面，逐页改色会造成批次内同类标签不一致。已在 notes「遗留」登记该问题与修复路径，待 token 层统一处理

### 跨页一致性补充

- 小一班人数统一口径：**30 人 = 在园 28 + 未入园 2**。`teacher/15-class-index` 名单计数改为「30 人（含 2 名未入园）」；`teacher/19-profile-index` 改为「在园 28 人」；`parent/20-points` 改为「在园 28 人」。
- 其余跨页项（演示芯片规范、家长端首页/接送页今日叙事、老师端工作台/我的重复统计）仍待批次决策，未在本次 P1 修复中处理。

### 修复后状态

- P0：5 条全部修复（见 §2.1）。
- P1：85/90 已修，8 条暂缓（批次级）。
- P2：253 条仍未处理。
- 批次门禁：`od-audit` 50/50 通过、0 问题；50 张 `preview.png` 均为 780px 宽，无过期预览。

## 3.2 配色方案分层（2026-10-27）

> 默认恢复原设计稿的亮橙；AA 方案作为可选主题，不强制替换视觉。

| 主题 | 说明 | 白字压品牌橙 | 合规 |
|---|---|---|---|
| `bright`（默认） | 原设计稿亮橙，视觉优先 | 2.35 / 2.80:1 | ✗ 不达 AA |
| `deep` | 白字保留，品牌渐变加深为焦糖橙 | 5.18–7.31:1 | ✓ AA |
| `ink` | 保留亮橙渐变，按钮文字改深墨色 | 5.9–7.4:1 | ✓ AA |

- 入口：`shared-design-tokens/tokens.json` + `shared-design-tokens/themes.json`；切换：`?theme=deep|ink|bright` 或 `data-theme`。
- 对比页：`_shared/themes.html`（7 个代表页 × 3 方案）；说明：`_shared/THEMES.md`。
- §3.1 中的 4 条对比度暂缓（parent/02、teacher/03、teacher/19、teacher/20）在 `deep` / `ink` 主题下解决；`bright` 保留为视觉参考。
- 50 张默认预览 + 14 张主题对比图已重出。

## 4. P2 分布（打磨项）

| 页面 | P2 数 |
|---|---|
| parent/21-class-index | 7 |
| teacher/01-workbench | 7 |
| teacher/08-pickup-records | 7 |
| parent/06-leave-index | 6 |
| parent/09-growth-index | 6 |
| parent/13-pickup-index | 6 |
| parent/15-pickup-authorized | 6 |
| parent/17-tasks-index | 6 |
| parent/26-messages | 6 |
| parent/27-profile | 6 |
| teacher/03-notice-create | 6 |
| teacher/09-tasks-index | 6 |
| teacher/11-calendar-index | 6 |
| teacher/12-calendar-create | 6 |
| teacher/16-album-index | 6 |
| teacher/18-messages-index | 6 |
| teacher/19-profile-index | 6 |
| parent/01-home | 5 |
| parent/02-login | 5 |
| parent/03-bind | 5 |
| parent/07-leave-create | 5 |
| parent/10-growth-detail | 5 |
| parent/11-growth-album | 5 |
| parent/12-growth-report | 5 |
| parent/14-pickup-qrcode | 5 |
| parent/16-pickup-authorized-create | 5 |
| parent/20-points | 5 |
| parent/23-student-index | 5 |
| teacher/02-login | 5 |
| teacher/04-growth-index | 5 |
| teacher/06-growth-detail | 5 |
| teacher/07-pickup-index | 5 |
| teacher/13-calendar-detail | 5 |
| teacher/20-doctor-checkin | 5 |
| teacher/22-webview | 5 |
| parent/04-notice-index | 4 |
| parent/05-notice-detail | 4 |
| parent/08-leave-detail | 4 |
| parent/18-tasks-detail | 4 |
| parent/19-tasks-wall | 4 |
| parent/22-calendar-index | 4 |
| parent/24-feedback | 4 |
| parent/25-settings | 4 |
| parent/28-webview | 4 |
| teacher/05-growth-create | 4 |
| teacher/14-leave-index | 4 |
| teacher/17-more-index | 4 |
| teacher/21-doctor-calendar | 4 |
| teacher/10-tasks-detail | 3 |
| teacher/15-class-index | 3 |

> P2 的逐条证据与修法见 `_shared/reviews/<page>.md`。

## 5. 50 页 verdict 总表

| 页面 | verdict | score | P0 | P1 | P2 |
|---|---|---|---|---|---|
| parent/01-home | WARN | 68 | 0 | 2 | 5 |
| parent/02-login | WARN | 78 | 0 | 2 | 5 |
| parent/03-bind | WARN | 74 | 0 | 3 | 5 |
| parent/04-notice-index | PASS | 86 | 0 | 0 | 4 |
| parent/05-notice-detail | WARN | 80 | 0 | 2 | 4 |
| parent/06-leave-index | FAIL | 58 | 1 | 1 | 6 |
| parent/07-leave-create | PASS | 84 | 0 | 0 | 5 |
| parent/08-leave-detail | WARN | 70 | 0 | 3 | 4 |
| parent/09-growth-index | WARN | 78 | 0 | 1 | 6 |
| parent/10-growth-detail | WARN | 75 | 0 | 2 | 5 |
| parent/11-growth-album | WARN | 75 | 0 | 1 | 5 |
| parent/12-growth-report | WARN | 75 | 0 | 2 | 5 |
| parent/13-pickup-index | WARN | 74 | 0 | 2 | 6 |
| parent/14-pickup-qrcode | WARN | 78 | 0 | 2 | 5 |
| parent/15-pickup-authorized | WARN | 74 | 0 | 1 | 6 |
| parent/16-pickup-authorized-create | WARN | 78 | 0 | 1 | 5 |
| parent/17-tasks-index | PASS | 80 | 0 | 0 | 6 |
| parent/18-tasks-detail | WARN | 70 | 0 | 4 | 4 |
| parent/19-tasks-wall | WARN | 70 | 0 | 1 | 4 |
| parent/20-points | WARN | 72 | 0 | 2 | 5 |
| parent/21-class-index | PASS | 87 | 0 | 0 | 7 |
| parent/22-calendar-index | FAIL | 76 | 1 | 2 | 4 |
| parent/23-student-index | WARN | 74 | 0 | 2 | 5 |
| parent/24-feedback | WARN | 75 | 0 | 3 | 4 |
| parent/25-settings | WARN | 80 | 0 | 1 | 4 |
| parent/26-messages | WARN | 72 | 0 | 1 | 6 |
| parent/27-profile | PASS | 85 | 0 | 0 | 6 |
| parent/28-webview | WARN | 72 | 0 | 1 | 4 |
| teacher/01-workbench | WARN | 72 | 0 | 1 | 7 |
| teacher/02-login | WARN | 76 | 0 | 1 | 5 |
| teacher/03-notice-create | WARN | 70 | 0 | 2 | 6 |
| teacher/04-growth-index | FAIL | 55 | 1 | 2 | 5 |
| teacher/05-growth-create | WARN | 68 | 0 | 4 | 4 |
| teacher/06-growth-detail | FAIL | 63 | 1 | 2 | 5 |
| teacher/07-pickup-index | WARN | 66 | 0 | 3 | 5 |
| teacher/08-pickup-records | WARN | 75 | 0 | 1 | 7 |
| teacher/09-tasks-index | WARN | 64 | 0 | 2 | 6 |
| teacher/10-tasks-detail | WARN | 70 | 0 | 5 | 3 |
| teacher/11-calendar-index | WARN | 80 | 0 | 1 | 6 |
| teacher/12-calendar-create | WARN | 72 | 0 | 2 | 6 |
| teacher/13-calendar-detail | WARN | 82 | 0 | 1 | 5 |
| teacher/14-leave-index | FAIL | 55 | 1 | 3 | 4 |
| teacher/15-class-index | WARN | 70 | 0 | 5 | 3 |
| teacher/16-album-index | WARN | 70 | 0 | 2 | 6 |
| teacher/17-more-index | PASS | 88 | 0 | 0 | 4 |
| teacher/18-messages-index | WARN | 72 | 0 | 2 | 6 |
| teacher/19-profile-index | WARN | 65 | 0 | 2 | 6 |
| teacher/20-doctor-checkin | WARN | 72 | 0 | 3 | 5 |
| teacher/21-doctor-calendar | WARN | 75 | 0 | 3 | 4 |
| teacher/22-webview | WARN | 76 | 0 | 1 | 5 |

## 6. 建议修复顺序

1. **5 条 P0**：teacher/04 弹层定位、teacher/14 日期 NaN、parent/06 统计键、teacher/06 数字自相矛盾、parent/22 地点矛盾。
2. **跨页一致性**：小一班人数口径、演示芯片规范、家长端今日叙事、老师端工作台/我的重复统计。
3. **P1 按页面批量修**：优先 FAIL/WARN 且 P1≥3 的页（teacher/05、10、15，parent/03、08、18、24）。
4. **P2**：进入 Impeccable 落地阶段逐页清理；token 级问题（对比度、小字号）回 `tokens.json` 一次修。
