# 消息（老师端）

- 源码：teacher/src/pages/messages/index.vue
- 设计系统：user:nurture-planet-local，surface=teacher
- 方向：收件箱式消息中心——按类型定色的图标、品牌橙未读点，点开即读、已读整行降灰。

## 信息层级

1. 导航栏「消息」+ 返回箭头（源码里消息页由入口 `navigateTo` 进入，属子页）
2. 吸顶分类分段：全部 / 通知 / 审批 / 系统，带各类条数（与源码 `tabTypeMap` 一致，无「未读」tab）
3. 列表口径行：`共 N 条 · M 条未读` + 「全部已读」（全局动作）
4. 单张卡片内的 cell 行：类型图标、标题、两行摘要、类型胶囊 + 业务明细（通知的已读进度、审批的状态）、相对时间、右箭头
5. 原型状态条：正常 / 空态 / 加载 / 失败

## 覆盖状态

- 正常：12 条示例（通知 5 / 审批 4 / 系统 3），其中 4 条未读。未读行标题 600、图标满色、右上角品牌橙小圆点；已读行标题 400、图标 62% 不透明度、类型胶囊转灰，不整行铺底色。
- 空态：整页空态（一条消息都没有）与分类空态（某分类 0 条）分开；分类空态只在切换分类后无数据时出现。整页空态 / 加载 / 失败下分类分段整体隐藏，不同屏展示已加载条数。
- 加载：5 行骨架屏；`prefers-reduced-motion` 下停用呼吸动画。
- 错误：整页错误 + 重试，重试回到正常列表。源码当前 catch 后只 toast，此态是为失败可见性补的。
- 禁用：「全部已读」在全量为 0 未读时禁用（`disabled` + `aria-disabled`），按全量未读判定而不是当前分类。

## token 映射

- 页面底色 `var(--bg)`、卡片 `var(--surface)`、分隔线 `var(--border)`
- 未读点 `var(--accent)`；选中分段 `var(--accent-wash)` + `var(--accent-deep)`
- 通知 `var(--info)` / `var(--info-wash)`；待审批 `var(--warning)` / `var(--warning-wash)`；已同意 `var(--success)` / `var(--success-wash)`；失败 `var(--danger)` / `var(--danger-wash)`
- 正文 `var(--fg)` / `var(--fg-2)`、次要 `var(--muted)`、时间与箭头 `var(--hint)`
- 圆角 `var(--radius-card)` / `var(--radius-xl)` / `var(--radius-pill)`；阴影 `var(--shadow-sm)` / `var(--shadow-lg)`
- 字号 `var(--text-lg)` 标题、`var(--text-base)` 正文与摘要、`var(--text-xs)` 时间与胶囊；字重 `var(--weight-normal)` / `var(--weight-medium)` / `var(--weight-semibold)`
- toast 用 `var(--fg)` 深底 + `var(--surface)` 反白

## 与现状的差异

- 未读：源码 `mapMessage` 把 `isRead` 固定为 `true`、未读数固定为 `0`，界面永远没有未读；原型按 `unread` 字段还原未读点、标题加粗与「全部已读」。接真实数据前需要接口下发未读态。
- 分类：保留源码的 全部/通知/审批/系统（`tabTypeMap` → notice / leave / system），没有新增「未读」tab；分段上补了每类条数，源码没有。
- 单条点击：源码只有 leave 跳 `/pages/leave/index`，其余仅 toast；原型保持同样分流（审批 toast「已打开请假审批」，其它「查看：标题」），并即时置读、计数同步。
- 「全部已读」：源码有 `markAllRead` 但模板未渲染入口，原型补到列表上方，并做全量禁用逻辑。
- 失败态：源码 `loadMessages` 捕获后只 toast，页面没有错误态；原型补整页失败 + 重试。
- 底部导航：共享 `mp-tabbar` 4 项，本页「消息」高亮，未读角标 4（与页头「4 条未读」一致）。

## 遗留

- 未读/已读的持久化需要后端字段（当前 `/teacher/messages` 不下发 `is_read` / `unread`），本页只做视觉与本地交互。
- 消息类型没有 task（源码与 `backend/api/api-teacher/10-messages.md` 都只有 notice / leave / system），后续若加任务消息需补图标与文案。
- 系统消息点击后没有详情页可跳，源码同样只 toast；需要产品确认系统消息是否要落到具体页面。
- 分类空态目前不可达（12 条示例覆盖全部三类），保留作为后续真实数据的防御态。
