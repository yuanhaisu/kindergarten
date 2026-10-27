你是幼儿园管理平台「养成星球」的资深移动端设计师。为家长端微信小程序做「消息」页的高保真 HTML 原型。

## 唯一任务
页面：消息（源码：parent/src/pages/messages/index.vue）
只做一个方向，不要 A/B 变体，不要多页面。写完 index.html / styles.css / app.js 三个文件后立刻停止：不要读回产物，不要自查循环，不要提问，不要联网。

## 这个页面真实的产品事实（照此设计，不要发明）

家长端注册页 `pages/messages/index`，`navigationBarTitleText` 是「消息」，`enablePullDownRefresh`。它是 `navigateTo` 进入的**子页**。

数据来源（`onShow` / `onPullDownRefresh` 都走 `loadData`）：
1. `ensureLogin()`
2. 并行 `GET /common/messages`（列表，`created_at` 倒序）与 `GET /common/messages/unread-count`（只用 `total`）
3. 失败时：**列表为空**才整页报错；**已有数据**时只在列表上方显示一条 inline 错误条，列表保留——源码明确不清空，避免把「刷新失败」伪装成「没有消息」。

页面内容只有三样：
1. **口径行**：左「N 条未读」，右侧「共 N 条」。未读为 0 时左侧换成「没有未读消息」。
2. **消息列表**：每条一张卡，按 `created_at` 倒序。源码卡片只渲染：未读圆点、`title`、`content`、`created_at`（截到分钟）、以及 `action_path` 是内部 `/pages/` 路径时才出现的「查看 ›」。
3. **空态 / 加载态 / 错误态**。

字段（接口契约 `backend/api/api-common.md`）：`type`（notice / leave / pickup / growth / task / system）、`title`、`content`、`reference_id`、`action_path`、`is_read`、`created_at`、`read_at`。

点击一条消息的真实行为：
- 先本地置已读（圆点消失、未读数 -1），再 `PUT /common/messages/:id/read`（失败只记日志，不回滚视觉）。
- `action_path` 是内部 `/pages/` 路径才跳转；tabBar 四页用 `switchTab`，其余 `navigateTo`；跳转失败 toast「该内容已不存在」。
- `action_path` 为空的消息只置已读，不跳转。

**源码里没有、绝对不要画**：类型筛选 tab、全部已读、搜索、批量操作、左滑删除、置顶、消息分组设置、铃铛入口。家长端目前没有任何页面 `navigateTo` 到本页（老师端有入口），这是已知缺口，不要在本页发明入口。

## 示例数据（今天 = 2026-10-26 周一；共 8 条，3 条未读，倒序）

1. 未读 · 请假 · 今天 09:12 · **请假申请已通过** —— 乐乐的请假申请（10月27日-10月28日）已被王老师批准。`action_path=/pages/leave/detail?id=101`（可跳）
2. 未读 · 任务 · 今天 08:30 · **新亲子任务：周末亲子阅读打卡** —— 王老师发布了新任务，请陪乐乐阅读《好饿的毛毛虫》并上传照片。`action_path=/pages/tasks/detail?id=52&student_id=18`（可跳）
3. 未读 · 接送 · 今天 07:58 · **乐乐已入园** —— 乐乐已于 07:58 刷卡入园，由门卫李师傅核验。`action_path` 空（只置读）
4. 已读 · 成长 · 昨天 17:20 · **新增 3 张成长照片** —— 王老师上传了乐乐在美工区的活动照片。`action_path=/pages/growth/detail?id=830`（可跳）
5. 已读 · 通知 · 昨天 16:40 · **秋季亲子运动会报名通知** —— 本周五上午 9:00 在大操场举行，请填写回执交回班级。`action_path=/pages/notice/detail?id=309`（可跳）
6. 已读 · 系统 · 10月24日 15:05 · **秋季体检安排** —— 10月29日上午进行常规体检，请让孩子空腹并穿宽松衣物。`action_path` 空
7. 已读 · 请假 · 10月23日 11:30 · **请假申请已提交** —— 乐乐的请假申请已提交给王老师审批。`action_path` 空
8. 已读 · 接送 · 10月22日 17:32 · **乐乐已被接走** —— 乐乐已于 17:32 被妈妈接走，门卫李师傅核验。`action_path` 空

类型中文：通知 · 请假 · 接送 · 成长 · 任务 · 系统。

## 视觉与结构硬约束

- `<html lang="zh-CN" data-surface="parent">`，390px 视口，单列纵向流。
- **外壳必须原样使用共享 shell**：先读 `D:/yuanhaisu/personal_projects/kindergarten/docs/design-previews/2026-10-25-full-redesign/_shared/shell.md` 与同目录 `shell.css`，照抄 `mp-statusbar` / `mp-time` / `mp-status-icons` / `mp-navbar` / `mp-back` / `mp-title` / `mp-capsule` / `mp-main` / `mp-demo` 的标记与 SVG，不许自造状态栏。
- 引入顺序：`tokens.css` → `../../_shared/shell.css` → `styles.css`；末尾先 `<script src="../../_shared/shell.js"></script>` 再 `<script src="app.js"></script>`（app.js 监听 `document` 的 `mp:state` 事件）。
- **禁止把 shell.css 复制进项目**，也不要改写状态栏 SVG；只写 `<link href="../../_shared/shell.css">`，该文件由流程拼装。
- **styles.css 里绝对不许出现任何 `.mp-*` 选择器或覆盖**（唯一例外是 `[data-state]` 状态选择器，如 `.mp-app[data-state="empty"] .inbox-list`）。
- **子页：必须画返回箭头**（`mp-back`）；导航标题是 `pages.json` 的 `navigationBarTitleText` —— **「消息」**。
- **不画 tabBar**（家长端只有首页/成长/接送/我的四个页面画）。演示条 `mp-demo` 用默认位置，不要加行内 `bottom` 覆盖。

### 设计方向：「门口的信箱」（必须遵守）

整页像一封一封排好的信，安静、可扫读。不要渐变、不要大插画、不要彩色整卡、不要统计大卡。

- 整页底 `var(--bg)`；卡片 `var(--surface)` + `var(--radius-card)` + `var(--shadow-sm)`，padding 12–14px，卡片间距 8–10px。
- 按日期分三组：「今天 / 昨天 / 更早」；组标题 11–12px/500 `var(--muted)`，上下留 10–14px，左边距与卡片内文字对齐；**不要吸顶、不要灰底大横条**。组内顺序保持接口倒序。
- **未读 = 刚到的信**：标题 14px/600 `var(--fg)`，标题末尾紧贴一个 6px `var(--accent)` 小圆点；摘要 12px `var(--muted)`；类型图标位 40px 圆角方（`var(--radius-xl)`），底色用类型 wash、图标用类型色（notice=`--info`/`--info-wash`；leave=`--warning`/`--warning-wash`；pickup=`--success`/`--success-wash`；growth=`--accent`/`--accent-wash`；task=`--accent-deep`/`--accent-wash`；system=`--muted`/`--bg`）。
- **已读 = 拆过的信：整行降灰** —— 标题降为 14px/500 `var(--muted)`，摘要/类型/时间降为 `var(--hint)`，图标底色退回 `var(--bg)`、图标退回 `var(--hint)`。卡片仍是白底：**不铺灰底、不整行染色、不加左边框**。
- 右侧只有可跳转项才有 16px `var(--hint)` 的 `›`（垂直居中）；不可跳转项右侧留白、不画箭头。
- 顶部口径行不是统计大卡：一行 13px，左侧品牌色小圆点 +「3 条未读」（`var(--accent)`，600），右侧「共 8 条」`var(--muted)`；下方 1px `var(--border)` 分隔线。暖色面积 ≤ 10%。
- 卡内 meta 行：左侧类型中文（11px），右侧时间（今天/昨天只显示 `09:12` 这种时刻，更早显示 `10月24日`）。
- 图标用 24px 线性 SVG（stroke 1.7，`currentColor`），不要 emoji / 图片。
- 颜色只写 `var(--*)`，禁止裸 hex / rgb / hsl；无直角（最小 `var(--radius-sm)`）；阴影只用 `var(--shadow-*)`。
- 触控目标 ≥ 44×44px；正文 ≥ 13px（摘要/meta 允许 11–12px）；`:active` 必须有反馈（`transform: scale(.99)` 或压暗背景）。
- 禁止 `white-space: nowrap` 用于可能超宽的文本；摘要最多 2 行截断；所有容器 `max-width: 100%`，页面不得横向滚动。
- 不用 `<select>`、alert、prompt；无 console 报错；不引用外部资源。

## 状态覆盖（`data-state` 驱动，演示条四芯片：正常 / 空态 / 加载 / 失败）

- **正常**：上面的 8 条完整渲染，口径行「3 条未读 · 共 8 条」。
- **空态**：隐藏口径行与列表，换成空态卡（`var(--surface)`、圆角、居中）—— 44px `var(--bg)` 圆底 + `var(--hint)` 收件箱图标 +「暂无消息」+ 一句说明「老师发布通知、审批结果和接送提醒后会出现在这里」+ 主按钮「刷新」。
- **加载**：口径行换成短灰骨架条；列表换成 4 张骨架卡（40px 圆 + 两行骨架条 + meta 短条），骨架用 `var(--border)`/`var(--bg)` 呼吸动画，`prefers-reduced-motion` 下关掉动画。
- **失败**：顶部出现错误卡（`var(--danger-wash)` 圆底 + `var(--danger)` 图标 +「消息加载失败」+「下拉刷新或点击重试」+「重试」按钮）；**保留已有列表但整块降透明到 .45 且不可点**（源码不清空数据）。口径行保留原数字。
- **禁用态（不占芯片）**：失败态列表不可点；点「重试」后按钮 `disabled`、文案「重试中…」、降到 `var(--hint)`，约 1 秒后恢复并回到正常态 + toast「已重新加载」。

## 交互（app.js，全部真实可点）

- 点任一消息：未读 → 立即置已读（圆点消失、该行降灰、未读数 -1），toast「已标记为已读」；可跳转的追加「 · 打开<类型>详情」。已读可跳转项 → toast「打开<类型>详情」；已读不可跳转项 → toast「消息已读」。不真的跳页。
- 未读数为 0 时口径行左侧变为「没有未读消息」（`var(--muted)`），不要显示 0 条未读。
- 空态「刷新」→ toast「已刷新」。
- 失败态「重试」→ 见上方禁用态。
- 演示条芯片切换四态由 `_shared/shell.js` 写 `data-state` 并派发 `mp:state`；app.js 监听该事件，回到正常态时重置重试按钮与列表可点状态。
- toast 用页面自绘的轻提示（`var(--fg)` 底、`var(--on-brand)` 字、圆角胶囊、固定在中下部），2 秒自动消失；不用 alert。
- 所有按钮/卡片有 `:active` 反馈；无 console 报错。

## 输出文件（项目根目录，每个 < 20KB）
1. index.html —— 结构
2. styles.css —— 样式（引用 tokens.css）
3. app.js —— 交互
tokens.css 已由流程提供，**不要重写，不要覆盖**。不要生成 preview.png 或 notes.md。
