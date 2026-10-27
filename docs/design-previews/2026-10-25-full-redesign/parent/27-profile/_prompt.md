你是幼儿园管理平台「养成星球」的资深移动端设计师。为家长端做一个页面的高保真 HTML 原型。

## 唯一任务
页面：我的（源码：parent/src/pages/profile/index.vue）
输出目录已有 tokens.css，不要重写。只做一个方向，不要 A/B 变体，不要多页面。写完 index.html / styles.css / app.js 即停止，不要提问，不要联网，不要读自己的产物做自查循环。

## 这个页面真实的产品事实
- 这是家长端 tabBar 的第 4 个 tab 页（pages.json tabBar：首页 / 成长 / 接送 / 我的）。导航栏标题「我的」，**不画返回箭头**；底部必须画 4 项 tabBar，「我的」高亮。
- 页面数据只有登录缓存的 user 对象：`nickname`、`phone`（后端下发脱敏值，如 138****6688）、`avatar`、`roles[]`（家长角色含 `kindergarten_name`）。源码身份卡是**居中**布局：渐变圆形头像（昵称首字）+ 昵称 + 手机号；昵称缺失显示「家长」，手机号缺失显示「微信用户」。
- 身份卡下面是 12 项静态菜单，顺序与标题不得增删改：
  1 亲子任务（/pages/tasks/index）
  2 积分成就（/pages/points/index）
  3 班级日历（/pages/calendar/index）
  4 成长报告（/pages/growth/report）
  5 请假记录（/pages/leave/index）
  6 接送记录（/pages/pickup/index）
  7 授权人管理（/pages/pickup/authorized）
  8 我的孩子（/pages/student/index）
  9 班级信息（/pages/class/index）
  10 通知公告（/pages/notice/index）
  11 意见反馈（/pages/feedback/index）
  12 设置（/pages/settings/index）
  源码 menuItems 每项还带 icon 键（task / points / calendar / report / leave / pickup / auth / child / class / notice / feedback / settings），但模板没渲染图标——原型按这些键补 18px 线性图标。
- 源码点击去向是跳页（tabBar 页用 switchTab，其它用 navigateTo）。本原型是单页，点击给 toast「已打开「<标题>」」即可，不真跳转。
- 退出登录：源码在「设置」子页（clearAuth + reLaunch 到登录页），**不在本页**。本原型按页主要求把它放到底部作为设计提案：危险色按钮，点击弹微信风格居中确认框（标题「退出登录」/ 说明「退出后需重新微信登录，确定退出吗？」/「取消」+「退出登录」）。

## 结构（按此实现）
1. 状态栏（20px）+ 导航栏「我的」（44px，标题居中，右侧胶囊，无返回箭头）。
2. 紧凑身份卡（88–100px 高，`--surface` 白卡 + `--radius-2xl` + `--shadow-sm`）：
   - 左：56px 圆形头像，`--accent-gradient` 底 + 白色昵称首字 22px/700；
   - 右：昵称 16px/600 +「家长」小胶囊（surface 底、accent-deep 字）；第二行 12px `--muted`：`138****6688 · 阳光幼儿园`（允许换行，不裁切）；
   - 卡片右上角加极淡的轨道装饰（两圈同心圆 + 一个小点，accent-wash，aria-hidden），**或**左侧 3px 渐变竖条——二选一，不要都上。
3. 四组 cell 列表（源码顺序不变，只在自然断点分组）：
   - 成长与任务：亲子任务 / 积分成就 / 班级日历 / 成长报告
   - 在园日常：请假记录 / 接送记录 / 授权人管理
   - 孩子与班级：我的孩子 / 班级信息
   - 消息与设置：通知公告 / 意见反馈 / 设置
   每组一张白卡：组标题 12px `--muted` 在卡片上方；每行 52px，30px 圆角图标块（`--accent-wash` 底 + `--accent-deep` 18px 线性图标）+ 14px/500 标题 + 右侧 `›`。
4. 底部「退出登录」危险按钮：`--danger-wash` 底 + `--danger` 字 + `--radius-pill`，高 48px，宽 100%。
5. 11px `--muted` 居中页脚「养成星球 · 家长端」。
6. tabBar 4 项：首页 / 成长 / 接送 / 我的，「我的」`aria-current="page"` 且用 accent 高亮。

## 状态覆盖（数据区）
- 正常：以上完整展示。
- 空：身份卡换成「还没有读取到家长信息」+ 主按钮「重新登录」（点击回到正常态并 toast）；四组菜单保留；退出登录隐藏。
- 加载：身份卡换骨架（56px 圆 + 2 条短条），四组列表换一张 3 行骨架卡；退出登录隐藏。
- 失败：身份卡换「账号信息加载失败」+「网络开小差了，请稍后重试」+「重试」按钮（点击变「重试中…」并禁用约 1 秒，回到正常态并 toast「已重新加载」）；四组菜单保留。
- 演示条用 `mp-demo` / `mp-demo-chip` / `data-mp-state`，固定右下并抬高到 tabBar 之上（行内 `style="bottom: calc(62px + env(safe-area-inset-bottom))"`）。

## 交互（app.js）
- 状态切换与 chip 同步（可监听 shell.js 派发的 `mp:state`）。
- 菜单行点击 → toast「已打开「<标题>」」。
- 退出登录 → 居中确认框（遮罩 + 卡片）：取消关闭；确认后 toast「已退出登录（原型）」并关闭。
- 所有可点元素 `:active` 缩放或压暗；不用 alert / prompt；console 无报错。

## 示例数据
- 昵称：乐乐妈妈；手机号：138****6688；幼儿园：阳光幼儿园。
- 本页不展示孩子信息，**不要**新增「已绑定孩子」列表或统计数字。

## 视觉与结构硬约束
- `<html lang="zh-CN" data-surface="parent">`
- 390px 手机视口；单列纵向流；不做桌面布局；页面不得横向滚动。
- 引入顺序：`tokens.css` → `../../_shared/shell.css` → `styles.css`。外壳一律用 `mp-*` 标记（`mp-statusbar` / `mp-navbar` / `mp-title` / `mp-capsule` / `mp-tabbar` / `mp-demo`）；**不要复制 shell.css、不要改写状态栏 SVG、不要在 styles.css 里定义 `.mp-*`**。
- 颜色只写 `var(--*)`，禁止裸 hex / rgb / rgba；阴影只用 `var(--shadow-*)`；无直角（最小 `--radius-sm`）。
- 暖色面积 ≤ 10%：大面积中性底，橙只出现在头像、图标块、短竖条。
- 正文 ≥ 13px；触控目标 ≥ 44px；层级靠字重 + 字号。
- 禁止 `white-space: nowrap` 用于可能超宽的文本；多列用 `grid-template-columns: repeat(N, minmax(0, 1fr))`；所有容器 `max-width: 100%`。
- 图标只用内联 SVG（currentColor），不引外部资源、不 base64 大图。

## 输出文件（项目根目录）
1. index.html —— 结构
2. styles.css —— 样式（引用 tokens.css）
3. app.js —— 交互

tokens.css 已由流程提供，不要写、不要覆盖。每个文件 < 20KB。
