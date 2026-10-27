# 消息（parent）

- 源码：parent/src/pages/messages/index.vue（接口 `GET /common/messages`、`GET /common/messages/unread-count`、`PUT /common/messages/:id/read`，契约见 backend/api/api-common.md）
- 设计系统：user:nurture-planet-local，surface=parent
- 方向：**门口的信箱** —— 按「今天 / 昨天 / 更早」排好的信；未读是刚到的信（品牌色小圆点 + 类型色图标），读过的信整行降灰，但卡片不铺灰底、不整行染色。
- 信息层级（从上到下）：
  1. 状态栏 + 导航栏「消息」（`navigateTo` 子页，带返回箭头，无 tabBar）
  2. 口径行：左「3 条未读」+ 品牌色小圆点，右「共 8 条」；下方 1px 分隔线
  3. 日期分组：今天（3 条，全未读）/ 昨天（2 条，全已读）/ 更早（3 条，全已读）
  4. 消息卡：40px 类型图标位 → 标题（未读带品牌色圆点）→ 两行摘要 → 类型 + 时间；仅可跳转项右侧有 `›`
  5. 原型状态条：正常 / 空态 / 加载 / 失败
- 覆盖状态：
  - **正常**：8 条示例、3 条未读；口径行「3 条未读 · 共 8 条」。
  - **空**：口径行与列表隐藏，空白卡内 44px 收件箱图标 +「暂无消息」+ 一句说明 + 主按钮「刷新」（对应源码 `onPullDownRefresh` 的同一 handler）。
  - **加载**：口径行换短骨架条 + 4 张骨架卡呼吸动画；`prefers-reduced-motion` 下关掉动画。
  - **失败**：列表上方叠错误卡（`--danger-wash` 图标位 +「消息加载失败」+「重试」）；**已有列表照常保留、可读可点**，对应源码「刷新失败不清空列表、只在列表上方插 inline-error」的行为。列表本身为空时只剩错误卡。
  - **重试**：点「重试」后按钮 `disabled`、文案「重试中…」、降到 `--hint`，1 秒后回到正常态并 toast「已重新加载」。
- token 映射：
  - 页面底 `--bg`；卡片/空态/错误卡 `--surface` + `--radius-card` + `--shadow-sm`
  - 未读圆点/口径行未读数字/空态主按钮 `--accent`；任务图标 `--accent-deep`；成长图标 `--accent` + `--accent-wash`
  - 类型图标位：notice=`--info-wash`/`--info`、leave=`--warning-wash`/`--warning`、pickup=`--success-wash`/`--success`、system=`--bg`/`--muted`
  - 未读标题 `--fg`、摘要/类型 `--muted`、时间 `--hint`；已读标题 `--muted`、摘要/类型/时间 `--hint`、图标位退回 `--bg`/`--hint`
  - 分隔线 `--border`；错误 `--danger-wash`/`--danger`；字号 `--text-xs/sm/base/md/lg`；字重 `--weight-medium/semibold`
- 与现状的差异：
  - **未读标记**：源码是卡片头部的红色 `--color-semantic-danger` 圆点 + 整卡 `.unread` 类；原型按批次要求改为**品牌色 6px 小圆点**，靠标题字重与图标色区分，不整行染色。
  - **已读降灰**：源码没有明显降灰；原型按要求把已读标题降到 `--muted`、摘要/时间降到 `--hint`、图标位退回中性。卡片仍是白底，避免列表出现「坏掉」的灰块。
  - **类型展示**：源码列表不渲染 `type`；原型加了类型图标与中文类型标签（notice/leave/pickup/growth/task/system → 通知/请假/接送/成长/任务/系统），让家长一眼分辨消息来源。这是展示层增强，没有新增业务能力。
  - **日期分组**：源码是平铺列表并原样显示 `created_at.slice(0,16)`（如 `2026-10-26T09:12`）；原型按「今天/昨天/更早」分组，今天/昨天只显示时刻，更早显示 `10月24日`，组内顺序仍是接口的 `created_at DESC`。
  - **摘要**：源码是单行 `white-space: nowrap` 截断；原型改 2 行截断，窄屏下更少丢信息。
  - **错误态**：源码列表为空时整页错误、已有数据时在列表上方插 inline 错误条且列表保持可点；原型沿用同一口径——错误卡叠在列表上方，列表不降透明、不锁点击，并把「点击重试」做成按钮。
  - **空态**：源码只有「暂无消息」文字；原型补图标、说明与「刷新」按钮（下拉手势在静态原型不可用）。
- 遗留 / 需产品确认：
  - **入口缺失**：家长端目前没有任何页面 `navigateTo` 到 `pages/messages/index`（老师端有入口）。本页只做页面本身，没有发明铃铛或入口。
  - **消息类型生产情况**：后端当前只有 pickup / task / system 三类消息的写入路径；notice / leave / growth 在 domain 枚举与接口契约里存在，但还没有创建点。原型示例覆盖了 6 类以验证类型图标与标签，真实上线初期可能只出现 3 类。
  - **全部已读**：接口有 `PUT /common/messages/read-all`，但源码没有渲染入口；原型未加，需产品确认是否要放到本页。
  - **未读口径**：页面只用 `unread-count.total`，接口还返回 `by_type`；若后续要做类型筛选或分类角标，需要产品确认展示方式。
  - **跳转校验**：源码只放行 `/pages/` 开头的内部路径，tabBar 页用 `switchTab`；原型用 toast 模拟跳转，未实现真实路由。
  - `created_at` 的时区与格式目前依赖后端字段，原型按本地日期分组，真实实现需统一时区口径。
