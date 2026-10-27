你是幼儿园管理平台「养成星球」的资深移动端设计师。为老师端做一个页面的高保真 HTML 原型。

## 唯一任务

页面：我的 / 老师个人中心（源码：teacher/src/pages/profile/index.vue，标题取自 pages.json 的 navigationBarTitleText「我的」）
只做一个方向，不要 A/B 变体，不要多页面。写完 index.html / styles.css / app.js 三个文件即停止：不要读取自己的产物，不要做自查循环，不要提问，不要联网。

## 这个页面真实的产品事实

**页面定位**：老师确认「我是谁、带哪个班」以及账号设置入口的地方。它不是仪表盘，数据只做一行摘要。

**数据来源（两段）**：
1. 本地登录态 `getUser()`（登录时写入 storage，秒开）：
   - `nickname` → 姓名，缺省「老师」；头像用姓名首字（真实 `avatar` 是 URL，原型不引外链图片）
   - `roles[0].kindergarten_name` → 园所，缺省「未绑定园所」
   - `roles[0].class_name` → 班级（可能为空）
   - `roles[0].role` → 职务，映射：`teacher`=主班老师、`assistant`=配班老师、`principal`=园长、其它=老师
   - `roles.length === 0` → 无班级/职务标签（本页唯一的「空」形态）
2. `GET /teacher/home/overview`（需登录）只取三个数字：
   - `attendance.present` → 今日出勤
   - `attendance.leave` → 今日请假
   - `pending_approvals` → 待审批

**页面区块顺序（必须保持）**：
1. 微信外壳：状态栏 + 导航栏（标题「我的」+ **返回箭头**，本页由 `navigateTo` 进入）+ 右上胶囊
2. 紧凑个人信息卡：头像（姓名首字）+ 姓名 + 园所 + 两个标签（班级 / 职务）
3. 今日班级数字带：今日出勤 / 今日请假 / 待审批（三列）
4. 账号与设置 cell 列表（一张卡，三行）
5. 退出登录（独立一行/一张卡，危险色文字）
6. 老师端**没有 tabBar**（pages.json 无 tabBar 字段），不要画任何底部导航

**三个设置入口（文案照抄源码，不要改写、不要新增）**：

| 标题 | 说明 | 源码去往 |
|---|---|---|
| 邀请码 | 分享或管理邀请码 | 无落地页，点击 toast「功能开发中」 |
| 账号安全 | 修改手机号、绑定微信 | 无落地页，点击 toast「功能开发中」 |
| 设置 | 关于、意见反馈 | 无落地页，点击 toast「功能开发中」 |

**退出登录**：源码用 `uni.showModal` 居中确认框——标题「退出登录」、说明「确定要退出当前账号吗？」，确认后清 token 并 `reLaunch` 到工作台。原型必须画**微信风格居中确认对话框**（标题 + 说明 + 取消 / 退出登录），确认后 toast「已退出登录」。

**加载与失败**：姓名/园所来自本地 storage，**不会转圈**；只有三个数字走网络。所以加载态只让数字变骨架，个人信息卡与 cell 列表保持可用。源码 catch 后静默归零——这次要改成可见的失败态：「今日数据加载失败」+「重试」。

**已知现状（这次要改掉的）**：
- 顶部是一整块高饱和蓝→渐变 header，白字压在上面，暖色/彩色占了近半屏。
- 统计数字用了蓝/绿/琥珀三种颜色，像小仪表盘。
- 三个功能入口各自一种图标底色（绿/橙/灰），再加一行独立投影的退出卡。
- 页面底部画了自绘的 4 项底栏（首页/班级/消息/我的）——`pages.json` 没有 tabBar，本批次老师端一律不画，**这次不要画**。

## 示例数据（中文，幼儿园语境，数字必须自洽）

- 姓名：**王老师**（头像首字「王」）
- 园所：**阳光幼儿园**
- 班级 / 职务：**小一班** / **主班老师**
- 今日出勤 **26**、今日请假 **2**、待审批 **3**（26 + 2 = 全班 28 人）
- 今日班级区头右侧小字：**小一班 · 共 28 人**
- 空（未绑定）态：园所显示「未绑定园所」，无班级/职务标签，三个数字显示「—」

## 设计方向（一个方向，照做）

主题：**「一张工牌」**。整页像老师工牌 + 微信设置页：上半部分一眼确认身份和今天班里的三个数，下半部分是安静的目录。

1. **个人信息卡（全页唯一渐变）**：用 `var(--accent-gradient-soft)`（浅暖奶油渐变，不是高饱和橙）做卡片底，1px `--border`，`var(--radius-2xl)`。内部：56px 圆形头像用 `var(--accent-gradient)` 实底 + 白色姓名首字；右侧姓名 18px/600 `--fg`，下面园所 13px `--muted`；再下一行两个胶囊标签（班级、职务）：`var(--surface)` 底 + 1px `--border` + 11px `--fg-2`。卡片高度控制在 120–140px，不要半屏。
2. **数字带**：区头「今日班级」15px/600 + 右侧小字「小一班 · 共 28 人」11px `--muted`。卡内三列 `grid-template-columns: repeat(3, minmax(0, 1fr))`，1px `--border` 分隔；数字 20px/600 `--fg`；只有「待审批」的数字用 `--accent-wash` 胶囊底 + `--accent-deep`（它是唯一需要动作的数字），其余不要彩色。
3. **账号与设置**：组标题「账号与设置」12px/500 `--muted`（微信设置页式弱标题），下面一张 `--surface` 卡、`--radius-card`、`--shadow-sm`。三行 cell：左侧 34×34 圆角图标块（`--accent-wash` 底 + `--accent-deep` 线性图标），中间标题 14px/500 + 说明 12px `--muted`，右侧不是箭头而是弱化的「**开发中**」小胶囊（`var(--bg)` 底 + `var(--muted)` 字 11px）——诚实表达这三个入口在源码里没有落地页。行高 ≥ 60px，行间 1px `--border`。
4. **退出登录**：一张独立 `--surface` 卡，居中「退出登录」14px/500 `--danger`，左侧一个描边 logout 图标；不要红色大底。点击弹居中对话框。
5. 层级只靠字重 + 字号；暖色只出现在渐变卡、头像、图标块、待审批胶囊，面积 ≤ 10%。

## 视觉与结构硬约束

- `<html lang="zh-CN" data-surface="teacher">`，390px 手机视口，单列纵向流，不做桌面布局、不做横向滚动。
- 引入顺序（必须一字不差）：
  ```html
  <link rel="stylesheet" href="tokens.css">
  <link rel="stylesheet" href="../../_shared/shell.css">
  <link rel="stylesheet" href="styles.css">
  ```
  末尾依次引入 `<script src="../../_shared/shell.js"></script>` 和 `<script src="app.js"></script>`。
- **外壳唯一实现是 _shared/shell.css**：必须原样使用下面的 `mp-*` 标记，不得自己画状态栏 / 导航栏 / 胶囊。styles.css 里**不得**出现任何 `.mp-*` 选择器定义（唯一例外是 `.mp-app[data-state="..."]` 这类状态选择器）。
- 必须补 `html, body { margin: 0; }` 和 `body { background: var(--bg); }`（共享 shell.css 不重置 body 边距）。
- 颜色只写 `var(--*)`，禁止裸 hex / rgb / 颜色名；需要半透明白时用 `color-mix(in srgb, var(--on-brand) 80%, transparent)` 这类由 token 派生的写法。无直角（最扁平元素 ≥ `var(--radius-sm)`）；阴影只用 `var(--shadow-sm/md/lg)`。
- 字号层级只靠字号 + 字重；正文 ≥13px，说明文字最小 12px；触控目标 ≥ 44×44px。
- 禁止 `white-space: nowrap`；多列统计用 `grid-template-columns: repeat(3, minmax(0, 1fr))`；所有容器 `max-width: 100%`；页面不得出现横向滚动。
- 点击反馈只做 `:active`（`transform: scale(.98)` 或背景压暗），不做 hover 才可见的信息。
- 不要给 `.mp-app` 加 `overflow`，不要给 `body` 设 `overflow: auto`（会破坏 sticky 外壳）。
- 底部内容留 `padding-bottom: calc(96px + env(safe-area-inset-bottom))`（给右下角演示条留位）。

## 状态覆盖（默认正常态）

**空态**（本页的真实含义是「未绑定园所」，不是「列表为空」）：
- 个人信息卡：园所变「未绑定园所」，隐藏班级/职务标签；头像与姓名保留
- 数字带三个数字变「—」，区头右侧小字隐藏；数字带下方出现一句 12px `--muted` 说明「还没有绑定园所，今日班级数据暂不可用」+ 一个 `--accent-wash` 底「刷新资料」小按钮（点击 toast「资料已刷新」）
- 不要发明「去绑定」之类源码没有的入口

**加载**：只让三个数字变骨架块（28×20px，透明度脉冲），区头右侧小字改成「同步中…」；个人信息卡与 cell 列表保持正常可用，不整页遮罩。

**失败**：数字带整块替换为：「今日数据加载失败」+「重试」按钮（`--accent-wash` 底）；个人信息卡与 cell 列表保持正常。点「重试」切回正常态并 toast「今日数据已更新」。

**禁用**：本页没有业务禁用控件；三个设置入口在源码里都无落地页，统一用右侧「开发中」弱化胶囊表达，点击只给 toast。**不要发明**额外的置灰按钮。

## 交互必须真的能点（app.js）

- 返回箭头 → toast「已返回 老师工作台」
- 三个 cell 点击 → toast「功能开发中」
- 退出登录 → 打开居中确认对话框；点「取消」或遮罩关闭；点「退出登录」→ 关闭 + toast「已退出登录」
- 「重试」→ 切回正常态（同步底部 chip 的选中态）+ toast「今日数据已更新」
- 空态「刷新资料」→ toast「资料已刷新」
- 演示条切换由 `_shared/shell.js` 负责，页面不要重复实现；页面只需用 `[data-state]` 属性选择器接管四态样式
- toast 用小程序风格深色小胶囊居中偏下，2 秒自动消失；**禁止 `alert()` / `prompt()` / `confirm()`**
- 对话框支持 Esc 关闭；控制台无报错

## 必须使用的外壳骨架（原样复制，只改标题）

```html
<div class="mp-app" data-state="normal">

  <div class="mp-statusbar" aria-hidden="true">
    <span class="mp-time">9:41</span>
    <span class="mp-status-icons">
      <svg width="17" height="11" viewBox="0 0 17 11" fill="currentColor"><rect x="0" y="7" width="3" height="4" rx="1"/><rect x="4.5" y="5" width="3" height="6" rx="1"/><rect x="9" y="2.5" width="3" height="8.5" rx="1"/><rect x="13.5" y="0" width="3" height="11" rx="1"/></svg>
      <svg width="15" height="11" viewBox="0 0 15 11" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round"><path d="M1 3.6C2.8 2 5.3 1 7.5 1s4.7 1 6.5 2.6"/><path d="M3.5 6.1c1.1-1 2.4-1.5 4-1.5s2.9.5 4 1.5"/><path d="M6 8.6c.5-.4 1-.6 1.5-.6s1 .2 1.5.6"/><circle cx="7.5" cy="10" r=".7" fill="currentColor"/></svg>
      <svg width="24" height="11" viewBox="0 0 24 11" fill="none"><rect x=".5" y=".5" width="20" height="10" rx="3" stroke="currentColor" stroke-opacity=".5"/><rect x="2" y="2" width="15" height="7" rx="1.8" fill="currentColor"/><path d="M22 3.5v4c.9-.3 1.4-1 1.4-2s-.5-1.7-1.4-2Z" fill="currentColor" fill-opacity=".5"/></svg>
    </span>
  </div>

  <header class="mp-navbar">
    <button class="mp-back" type="button" aria-label="返回">
      <svg width="20" height="20" viewBox="0 0 20 20" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M12.5 4.5 7 10l5.5 5.5"/></svg>
    </button>
    <h1 class="mp-title">我的</h1>
    <div class="mp-capsule" aria-hidden="true">
      <span class="mp-capsule-dots">···</span>
      <span class="mp-capsule-divider"></span>
      <span class="mp-capsule-ring">◎</span>
    </div>
  </header>

  <main class="mp-main">
    <!-- 页面内容 -->
  </main>

  <div class="mp-demo" data-open="false">
    <div class="mp-demo-chips" role="group" aria-label="原型状态">
      <button class="mp-demo-chip" data-mp-state="normal" aria-pressed="true">正常</button>
      <button class="mp-demo-chip" data-mp-state="empty" aria-pressed="false">空态</button>
      <button class="mp-demo-chip" data-mp-state="loading" aria-pressed="false">加载</button>
      <button class="mp-demo-chip" data-mp-state="error" aria-pressed="false">失败</button>
    </div>
    <button class="mp-demo-toggle" type="button" aria-expanded="false">原型演示</button>
  </div>

  <div class="toast" id="toast" role="status" aria-live="polite"></div>
</div>
```

## 输出文件（项目根目录）

1. `index.html` —— 结构（含内联 SVG 图标；退出确认对话框放在外壳内）
2. `styles.css` —— 页面样式（引用 tokens.css 的变量；不要定义 `.mp-*`）
3. `app.js` —— 交互：toast、对话框开关、重试/刷新、cell 点击
4. `tokens.css` 已由流程提供，**不要重写、不要覆盖、不要修改**

每个文件 < 20KB。不要引用任何外部资源（无 CDN、无字体外链、无图片外链）。
