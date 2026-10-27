# 对抗性单页审查标准（2026-10-27）

> 你的任务是**找缺陷**，不是确认页面做得好。先假设这页有问题，再逐项证伪。
> 只读，不改任何文件。每条 finding 必须有证据（`file:line` 或 DOM 选择器/可见文本）。

## 0. 审查对象

- 页面目录：`docs/design-previews/2026-10-25-full-redesign/<surface>/<NN-slug>/`
- 必读：本页 `index.html` / `styles.css` / `app.js` / `notes.md`、`_shared/brief.md`、对应源码 `.vue`
- 门禁 `od-check` / `od-audit` 已全批次通过，**不要重复跑**；重点看内容、逻辑、设计核心、冗余与导航意图

## 1. 完整度（completeness）

对照源码 `.vue` 逐项核对：

- 源码中的**区块顺序**、字段、按钮、状态分支、空/错/加载分支是否都在
- 源码里的**跳转入口**是否都在原型里有对应 affordance
- 原型是否**发明**了源码没有的业务能力（`notes.md` 明确标为「设计提案 / 需产品确认」的不算缺陷，但要在 finding 里指出提案本身是否合理）
- 示例数据的数字是否自洽（人数、天数、计数、余额链）

## 2. 逻辑（logic）

- 状态机：正常 / 空 / 加载 / 错误 / 禁用是否互斥且能到达；切换后布局不塌
- 表单：必填校验是否真的阻断；错误提示是否就地；提交中是否真禁用；失败是否保留内容
- 交互：每个可点元素是否有 handler，或明确 `disabled` / `aria-disabled`；禁止「看起来能点但没反应」
- 弹层：打开/关闭/遮罩/滚动锁定是否成对；关闭后焦点/滚动是否恢复
- 数字与文案：统计与列表是否对得上；时间/日期是否自洽（含时区）
- `console` 是否干净（静态检查 `app.js` 里明显的空引用/未定义）

## 3. 设计核心（design-core）

品牌北极星：**Tender Cosmos 温柔宇宙**（低对比、无锐角、圆润、琥珀柔光）。

- 外壳必须只链接 `../../_shared/shell.css` 并使用 `mp-*` 标记；`styles.css` 不得定义 `.mp-*`
- 颜色只写 `var(--*)`；不得裸 hex/rgb/hsl（`tokens.css` 除外）
- 无直角：最小 `var(--radius-sm)`；卡片 `var(--radius-card)`；按钮/标签/头像 `var(--radius-pill)`
- 阴影只用 `var(--shadow-*)`
- 暖色面积 ≤ 10%；大面积是 `var(--bg)` / `var(--surface)`
- 层级只靠字重+字号（标题 600 / 正文 400 / 标签 500）
- 触控 ≥ 44×44px；正文 ≥ 13px；底部安全区；无 `white-space: nowrap` 超宽文本；390 视口无横向滚动

## 4. 冗余 / 错位（redundancy）

- 页内同一信息是否重复出现（标题 + 卡头 + 正文三处重复等）
- 信息是否放错区块（统计放进操作区、次要动作抢主视觉）
- 同一数字/状态是否在一页出现两次且口径不同
- 空态/错误态是否与正常态重复堆叠

## 5. 导航意图（navigation）

原型是独立文件，**不要求真实跨页跳转**；审查的是「意图是否正确、目的地是否存在」。

- 页面内每个可点元素的**隐含目的地**是否与源码 `.vue` 的 `uni.navigateTo / switchTab / redirectTo / reLaunch` 一致
- 目的地是否在下方「原型清单」中存在；指向不存在的页面 = 缺陷
- 家长端 tabBar 只有 4 页：`01-home` / `09-growth-index` / `13-pickup-index` / `27-profile`；其余页不画 tabBar，且是子页要有返回箭头
- 老师端仅 `01-workbench` / `15-class-index` / `18-messages-index` / `19-profile-index` 四个常驻页画 4 项共享底栏；其余老师端页面不画（`pages.json` 仍无 tabBar 字段，底栏是源码自绘）

## 6. notes 准确性（notes-accuracy）

- `notes.md` 的声明与 `index.html/styles.css/app.js` 实际是否一致
- 声称「手写兜底」是否与 `_run.json` 的 `handAuthored` 一致
- 声称覆盖的状态是否真的能切换

---

## 原型清单（导航目的地是否存在，以此为准）

### 老师端（22）

`/pages/index/index` → `teacher/01-workbench`；`/pages/auth/login` → `02-login`；`/pages/notice/create` → `03-notice-create`；`/pages/growth/index` → `04-growth-index`；`/pages/growth/create` → `05-growth-create`；`/pages/growth/detail` → `06-growth-detail`；`/pages/pickup/index` → `07-pickup-index`；`/pages/pickup/records` → `08-pickup-records`；`/pages/tasks/index` → `09-tasks-index`；`/pages/tasks/detail` → `10-tasks-detail`；`/pages/calendar/index` → `11-calendar-index`；`/pages/calendar/create` → `12-calendar-create`；`/pages/calendar/detail` → `13-calendar-detail`；`/pages/leave/index` → `14-leave-index`；`/pages/class/index` → `15-class-index`；`/pages/album/index` → `16-album-index`；`/pages/more/index` → `17-more-index`；`/pages/messages/index` → `18-messages-index`；`/pages/profile/index` → `19-profile-index`；`/pages/doctor/checkin` → `20-doctor-checkin`；`/pages/doctor/calendar` → `21-doctor-calendar`；`/pages/webview/index` → `22-webview`

> 已知源码死链：老师端工作台待办跳 `/pages/notice/index`，但 `pages.json` 无此页（只有 `notice/create`）。原型集因此没有老师端通知列表页。这不算单页缺陷，但若某页出现指向「通知列表」的入口，要标注。

### 家长端（28）

`/pages/index/index` → `parent/01-home`；`/pages/auth/login` → `02-login`；`/pages/auth/bind` → `03-bind`；`/pages/notice/index` → `04-notice-index`；`/pages/notice/detail` → `05-notice-detail`；`/pages/leave/index` → `06-leave-index`；`/pages/leave/create` → `07-leave-create`；`/pages/leave/detail` → `08-leave-detail`；`/pages/growth/index` → `09-growth-index`；`/pages/growth/detail` → `10-growth-detail`；`/pages/growth/album` → `11-growth-album`；`/pages/growth/report` → `12-growth-report`；`/pages/pickup/index` → `13-pickup-index`；`/pages/pickup/qrcode` → `14-pickup-qrcode`；`/pages/pickup/authorized` → `15-pickup-authorized`；`/pages/pickup/authorized-create` → `16-pickup-authorized-create`；`/pages/tasks/index` → `17-tasks-index`；`/pages/tasks/detail` → `18-tasks-detail`；`/pages/tasks/wall` → `19-tasks-wall`；`/pages/points/index` → `20-points`；`/pages/class/index` → `21-class-index`；`/pages/calendar/index` → `22-calendar-index`；`/pages/student/index` → `23-student-index`；`/pages/feedback/index` → `24-feedback`；`/pages/settings/index` → `25-settings`；`/pages/messages/index` → `26-messages`；`/pages/profile/index` → `27-profile`；`/pages/webview/index` → `28-webview`

---

## 已记录的批次级问题（不要重复上报为新发现）

接口/契约缺口见 `_shared/findings.md` 的 A1–A42；老师端死链与自绘底栏见本文件第 5 节。

单页审查只报**该页自身**的问题；如果是 A1–A42 已经记录的同一事实，可引用编号，但不计入新 finding。
