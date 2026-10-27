你是幼儿园管理平台「养成星球」的资深移动端设计师。为**家长端**做「设置」页的高保真 HTML 原型。

## 唯一任务
页面：设置（源码：`parent/src/pages/settings/index.vue`）
只做一个方向，不要 A/B 变体，不要多页面。写完三个文件即停止：不要提问、不要联网、不要读取自己的产物做自查循环。

## 这个页面真实的产品事实
- 页面是**子页**：从「我的」页 `uni.navigateTo` 进入 → 导航栏**必须有返回箭头**，标题「设置」，**不画 tabBar**。
- 进入即 `onLoad`：`ensureLogin()` → `GET /parent/settings`。返回 `{ notification: {...}, privacy: {...} }`。
- **消息通知**组（源码实际渲染，6 项开关，顺序如下）：
  `notice_push` 通知推送 · `leave_push` 请假通知 · `pickup_push` 接送通知 · `growth_push` 成长记录 · `task_push` 任务提醒 · `announcement_push` 公告通知
- **隐私**组（接口契约 `backend/api/api-parent/11-settings.md` 有，源码当前**尚未渲染**，本次按契约补齐 2 项）：
  `show_phone_to_teacher` 向老师展示手机号 · `allow_growth_share` 允许展示成长记录
- 开关切换：调 `PUT /parent/settings`，成功后 `uni.showToast("已更新")`，失败 `uni.showToast("更新失败")`。所以点击开关有**短暂的同步中**过程，不是纯本地切换。
- 页面底部：**退出登录**按钮 → 微信原生居中确认框（标题「退出登录」/ 内容「确定要退出登录吗?」/ 取消 + 确定）→ 确认后清登录态并 `reLaunch` 回登录页。
- 源码没有「清除缓存」「关于」「版本号」「账号信息」等入口——**不要发明**这些 cell。

## 示例数据（中文，幼儿园语境）
- 消息通知：通知推送 开 · 请假通知 开 · 接送通知 开 · 成长记录 开 · 任务提醒 关 · 公告通知 开
- 隐私：向老师展示手机号 开 · 允许展示成长记录 关
- 确认框文案：「退出登录」/「退出后需重新微信授权登录。」/ 取消 / 退出登录

## 页面结构（从上到下）
1. 微信外壳（由共享 shell.css 提供，直接用标记）
2. 组标题「消息通知」（弱化：12px、中性灰、weight 500，左边距与卡内文字对齐）
3. 消息通知卡：6 行 cell，每行 = 文案（14px）+ 右侧开关（48×30，滑块 26，开启态用品牌橙）。行高 ≥ 54px，行间 1px 分隔线，首行无上分隔、末行无下分隔
4. 组尾小注（11px 中性灰）：「关闭后消息仍会保留在消息列表，只是不再提醒。」
5. 组标题「隐私」
6. 隐私卡：2 行 cell，同上
7. 组尾小注：「仅影响老师和同班家长看到的信息。」
8. 退出登录卡：单独一组，整行居中、`--danger` 文字、白底、无箭头
9. 底部留白 `calc(24px + env(safe-area-inset-bottom))`

## 视觉与结构硬约束
- `<html lang="zh-CN" data-surface="parent">`；390px 视口；单列纵向流，不做桌面布局
- 引入顺序：`tokens.css` → `../../_shared/shell.css` → `styles.css`
- 外壳**原样使用**共享标记：`mp-statusbar` / `mp-navbar` / `mp-title` / `mp-capsule` / `mp-back`（返回箭头**要有**）/ `mp-demo`。**禁止**把 shell.css 复制进项目，**禁止**在 styles.css 里定义任何 `.mp-*` 选择器（含覆盖）——只允许 `[data-state]` 状态选择器
- 截图里禁止出现 tabBar
- 颜色只写 `var(--*)`，**禁止裸 hex / rgb / rgba**（`<meta name="theme-color">` 除外）；阴影只用 `var(--shadow-sm|md|lg)`
- **无直角**：最扁平元素 ≥ `var(--radius-sm)`；卡片 `var(--radius-card)`；按钮/滑块 `var(--radius-pill)`；开关轨道用 pill
- 暖色面积 ≤ 10%：大面积为 `--bg` 中性底 + `--surface` 白卡；橙色只出现在**开启态开关轨道**、主按钮和焦点态
- 层级只靠字重 + 字号：标题 600 / 正文 400 / 标签 500
- 触控目标 ≥ 44px；正文 ≥ 13px；分隔线用 `--border`
- **禁止 `white-space: nowrap`**；所有容器 `max-width: 100%`；不得出现横向滚动；多列用 `grid-template-columns: repeat(N, minmax(0, 1fr))`
- 开关用**小程序/iOS 风格**（轨道 + 圆形滑块 + `transform: translateX()` 动效，`var(--motion-base)`），不要用原生 `<input type=checkbox>` 的方块样式
- 不用 `<select>`、alert、prompt、hover 才可见的信息

## 必须覆盖的状态（数据区）
页面根 `.mp-app` 带 `data-state`，用属性选择器接管：
- `normal`（默认）：完整两卡 + 退出登录
- `empty`：设置项为空（接口返回空对象）→ 卡内换成居中空态：线框图标 +「还没有可设置的提醒项」+ 主按钮「重新加载」；退出登录卡保留
- `loading`：骨架屏 2–3 行（灰条呼吸动画），隐藏真实列表
- `error`：整页居中错误块：`--danger-wash` 圆 + 感叹号 +「设置加载失败」+「网络不太稳定，请稍后再试」+「重试」按钮
- `disabled`：开关提交中（PUT 未返回）→ 该行降级：文字转 `--muted`、开关 `opacity: .5` 且不可点、行尾出现 11px「同步中…」

## 交互（app.js，必须有真实反馈）
- 点开关（整行都可点，行 ≥ 44px）：乐观翻转动效 → 行进入「同步中…」→ 约 450ms 后回正常并 toast「已更新」
- 点「退出登录」：弹出**微信风格居中对话框**（半透明遮罩 + `--surface` 圆角卡 + 标题 + 说明 + 取消/退出登录 两个按钮，确认按钮用 `--danger` 文字或浅底）；「退出登录」→ toast「已退出登录」并关闭
- 空态「重新加载」/ 错误态「重试」：1 秒降级为「重试中…」后回到正常态，toast 提示
- 返回箭头 → toast「返回上一页」
- toast 轻提示条（底部居中、深色半透明胶囊），不要 alert
- 状态切换由共享 `shell.js` 负责，不要重写 demo 条，也不要重复绑定 chip

## 输出文件（项目根目录）
1. `index.html` —— 结构
2. `styles.css` —— 样式（引用 tokens.css）
3. `app.js` —— 交互
tokens.css 已由流程提供，**不要重写、不要覆盖**。
每个文件 < 20KB。不要引用任何外部资源、不要内联 base64 图片、不用字体 CDN。
