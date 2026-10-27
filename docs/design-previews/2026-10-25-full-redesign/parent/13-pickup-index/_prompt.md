你是幼儿园管理平台「养成星球」的资深移动端设计师。为家长端微信小程序做一个页面的高保真 HTML 原型。

## 唯一任务
页面：接送记录（源码：parent/src/pages/pickup/index.vue）
只做一个方向，不要 A/B 变体，不要多页面。写完 index.html / styles.css / app.js 三个文件后立刻停止：不要读回产物，不要自查循环，不要提问，不要联网。

## 产品事实（照此设计，不要发明）
这是家长端**底部 tab 页**（4 个 tab 中的第 3 个「接送」），`pages/pickup/index`，导航标题「接送记录」，`enablePullDownRefresh: true`。`onShow` 时调 `GET /parent/pickup/records?student_id=<当前孩子>`，一次返回该孩子全部接送记录。**没有筛选、没有分页控件、没有「家长扫码」功能**（扫码是老师端的事，家长只出示二维码）。

响应字段：`id`、`date`（YYYY-MM-DD）、`type`（1=入园，2=接走）、`type_label`（「入园」/「接走」）、`pickup_name`（如「爸爸（王明）」）、`pickup_relation`（英文枚举 father/mother/grandmother…）、`recorded_at`（ISO 时间，取第 11–16 位为 HH:MM）、`is_abnormal`（布尔）。

源码现有 UI：页头「接送记录」+ 右侧「接送二维码」按钮（`uni.navigateTo('/pages/pickup/qrcode')`）；加载中「加载中...」；无记录「暂无接送记录」；有记录时每条一张卡：类型标签 + `date` 和 HH:MM + `pickup_name (pickup_relation)`。底部 BottomNav 高亮「接送」。请求失败只弹 toast（错误文案来自后端）。

本页是**入口 tab 页，导航栏不画返回箭头**。

## 页面结构（自上而下，顺序固定）
1. **今日接送卡**（核心，全页唯一允许大面积暖色的区域）：把当天记录汇总成一张「接送凭证」。
   - 顶部：日期「10月26日 周一」+ 孩子上下文「乐乐 · 小一班」。
   - 主状态：「已接走」大号状态 + 接走时间「17:30」+「爸爸（王明）」；若只有入园记录则显示「在园中」+「07:52 妈妈（李丽）送入」。
   - 下方用竖向时间轴列当天各条记录：圆点 + 类型 + HH:MM + 接送人 + 关系中文标签。入园用 success 色，接走用 accent 色。
   - 底部主按钮「出示接送二维码」（打开二维码底部弹层），旁边小字「核验后一次有效」。这是页面唯一的实心主按钮。
2. **历史记录**：按日期分组（日期 + 星期作为组标题），每组内是当天的入园/接走行：
   - 左侧：类型胶囊（入园 = success-wash / success；接走 = info-wash / info）；`is_abnormal` 为真时加 warning 色「异常」小标。
   - 主文案：`pickup_name`；次文案：接送关系中文（父亲/母亲/奶奶/爷爷）。
   - 右侧：`HH:MM` + `›`。
   - 点击任意一行 → toast「已打开 10月23日 接走记录」（源码没有详情页，不要做详情弹层）。
   - 组标题右侧一个 44×44 刷新图标按钮 → toast「已刷新接送记录」。
3. 历史记录之后不要加「加载更多」「查看全部」；源码没有分页控件。

## 示例数据（今天 = 2026年10月26日 星期一；孩子 = 乐乐，小一班）
- 今日：入园 07:52 妈妈（李丽）· 母亲；接走 17:30 爸爸（王明）· 父亲 → 主状态「已接走」
- 10月23日 周五：入园 07:48 妈妈（李丽）；接走 17:41 奶奶（张秀兰）· 奶奶
- 10月22日 周四：入园 08:02 妈妈（李丽）；接走 18:05 爸爸（王明）
- 10月21日 周三：入园 07:51 妈妈（李丽）；接走 17:22 爷爷（王建国）· 爷爷，标「异常」
- 10月20日 周二：入园 07:55 妈妈（李丽）；接走 17:15 妈妈（李丽）
「异常」仅来自接口 `is_abnormal`，不要写异常原因。

## 四态（用 `.mp-app[data-state]` 接管；默认正常态）
- 正常：如上。
- 空（今天还没接送、也没有历史）：今日卡与列表都隐藏，显示居中空态 —— 柔和圆底图标（接送/二维码意象，内联 SVG）、标题「今天还没有接送记录」、一句「老师核验接送码后，记录会出现在这里」、主按钮「出示接送二维码」。
- 加载：今日卡与列表隐藏，显示骨架屏 —— 一张凭证卡骨架 + 3 行列表骨架，用 var(--surface-warm) / var(--border) 做 shimmer 动画，不要进度条。
- 失败：居中「接送记录加载失败」+「网络好像不太稳定」+ 描边按钮「重试」；点重试恢复正常态并 toast「已重新加载」。
- 禁用态：加载/失败态时「出示接送二维码」按钮 disabled（降透明度、不可点）。

## 交互（app.js 必须真的能点；无 alert/prompt、无 console 错误）
- 点「出示接送二维码」→ 打开底部弹层：拖动条、标题「接送二维码」、人证对照「乐乐 → 爸爸（王明）· 父亲」、一个 CSS/SVG 绘制的二维码占位（只用 var(--fg) 方块 + var(--surface) 底，不要真二维码图片）、提示「请向老师出示此码，核验后一次有效」、「有效期至 10月26日 23:59」、次要按钮「刷新二维码」（toast）、文字按钮「更换今日接走人」（toast「已打开授权接送人」）。
- 点遮罩 / 关闭按钮 / Esc 关闭弹层；弹层打开时背景不滚动。
- 底部 tabBar 4 项：首页 / 成长 / 接送 / 我的；「接送」为当前项（`aria-current="page"`，accent 色）。点其它项 toast「已切换到首页」等；点接送 toast「当前已在接送」。
- 演示状态条用 `mp-demo`，行内 `style="bottom: calc(62px + env(safe-area-inset-bottom))"`。

## 视觉硬约束
- `<html lang="zh-CN" data-surface="parent">`，390px 视口，单列纵向流。
- 引入顺序：`tokens.css` → `../../_shared/shell.css` → `styles.css`；末尾先 `<script src="../../_shared/shell.js"></script>` 再 `<script src="app.js"></script>`。
- **外壳原样使用共享 shell**：先读 `D:/yuanhaisu/personal_projects/kindergarten/docs/design-previews/2026-10-25-full-redesign/_shared/shell.md` 与同目录 `shell.css`，照抄 `mp-statusbar` / `mp-time` / `mp-status-icons` / `mp-navbar` / `mp-title` / `mp-capsule` / `mp-main` / `mp-tabbar` / `mp-tab` / `mp-demo` 的标记与 SVG，不许自造状态栏或 tabBar；**不要把 shell.css 复制进项目、不要改写状态栏 SVG**。
- styles.css 里不得出现 `.mp-*` 定义（`[data-state]` 选择器除外）。
- 颜色只写 var(--*)；禁止裸 hex/rgb；无直角（≥ var(--radius-sm)）；阴影只用 var(--shadow-*)；暖色面积 ≤ 10%（只有今日卡的 accent-wash 和主按钮）。
- 标题 600 / 正文 400 / 标签 500；正文 ≥ 13px；触控目标 ≥ 44×44。
- 禁止 `white-space: nowrap`（除非该行加注释 `od-audit-allow: nowrap`）；多列用 `grid-template-columns: repeat(N, minmax(0, 1fr))`；所有容器 `max-width: 100%`；不得横向滚动。
- 不引外网资源、不内联 base64 大图、不用图表库。

## 输出文件（写到当前项目根目录，每个 < 20KB）
1. index.html —— 结构
2. styles.css —— 样式（引用 tokens.css）
3. app.js —— 交互
tokens.css 已由流程提供，**不要重写、不要覆盖**。不要生成 preview.png 或 notes.md。
