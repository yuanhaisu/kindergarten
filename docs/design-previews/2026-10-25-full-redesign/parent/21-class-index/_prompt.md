你是幼儿园管理平台「养成星球」的资深移动端设计师。为家长端做一个页面的高保真 HTML 原型。

## 唯一任务
页面：班级信息（源码：parent/src/pages/class/index.vue）
只做一个方向，不要 A/B 变体，不要多页面。写完全部文件即停止：不要提问，不要联网，不要读取自己刚写的产物做自查，不要覆盖 tokens.css。

## 这个页面真实的产品事实

**定位**：家长想确认「孩子在哪一班、谁在带」时的信息页。入口在「我的」→ 班级信息（`uni.navigateTo` 进入），所以是**子页，导航栏左侧必须有返回箭头**，标题「班级信息」，**不画 tabBar**。

**数据来源**：`GET /parent/class/info?student_id=` →
`{ id, name, grade, academic_year, student_count, teachers: [{ id, nickname, avatar, title_label, phone }] }`
（`title_label` 取值「主班老师 / 副班老师 / 生活老师」；`phone` 为脱敏号码如 `138****1234`。任一字段为空时整行不出现，不要留空标签。）

**源码现在的区块顺序**：
1. 班级卡：班级名（大标题）；下一行 `grade · academic_year · student_count人`。
2. 班级老师卡：区块标题「班级老师」；每位老师一行：昵称 / 职务 / 电话。

**源码没有的**：没有班级群二维码、没有聊天、没有在线客服、没有拨号按钮、没有教师详情页。这些一律不要发明。老师行是**纯展示**，不要给它加点击后弹窗/拨号暗示。

**本页允许补齐的两个入口（调度要求，非源码）**：在老师卡下方加一张「班级相关」入口卡，两个 cell，指向家长端**真实存在**的页面：
- 成长相册（`pages/growth/album`）
- 通知公告（`pages/notice/index`）
文案用真实页面标题「成长相册」「通知公告」，不要写成「班级群」「班级圈」。点击只 toast「已打开成长相册 / 已打开通知公告」。

## 示例数据（必须用这一组，不要另编）
班级：**小一班** / 年级 **小班** / 学年 **2025-2026** / 在班 **28 人**。
老师：
1. 王老师 · 主班老师 · 138****1234
2. 李老师 · 副班老师 · 139****5678
3. 刘老师 · 生活老师 · 137****9012

头像：原型禁止外链图片，`avatar` 用**姓氏首字**（王 / 李 / 刘）落在柔和圆形里；接口 avatar 为空时同样走首字兜底。

## 视觉方向：「班级名片」
这一页真实数据只有三组字段，设计不能靠堆装饰，要靠**一张记得住的名片 + 一份可信的名单**：
- 上层是班级身份卡：一个 56px 左右的班级徽记（可用「小一」二字或幼儿园/人群线稿，底色 `var(--accent-gradient-soft)` 这类柔和暖底，面积极小），右侧是班级名与「小班 · 2025-2026 学年」，再右侧或下方是「在班 28 人」的紧凑数字位（数字大、标签小）。
- 下层是老师名单：白色卡片内三行，48px 首字头像 + 昵称 + 职务胶囊 + 脱敏电话；用 1px 分隔线，不堆卡片阴影。主班老师的职务胶囊可用 `var(--accent-wash)` + `var(--accent-deep)`，其余用中性底，让「谁是主班」一眼可见。
- 再下层「班级相关」两行 cell：左线性图标 + 标题 + 右侧 `›`，不写副标题。
- 页脚一行小字：「班级信息由园所统一维护」。

## 状态覆盖（数据区必须四态，用 `.mp-app[data-state]` 纯 CSS 切换）
- **正常**：默认，上表数据完整渲染。
- **空态**：班级卡保留（班级本身存在）；老师卡内显示空块：线框人形图标 + 「暂无老师信息」+ 一句「班级老师尚未配置，请稍后重新加载」+ 主按钮「重新加载」；入口卡保留。
- **加载**：班级卡换骨架（徽记方块 + 两条文字条 + 数字块），老师卡换 3 行骨架（圆形头像 + 两条文字条），带呼吸动画；老师计数与入口卡隐藏。
- **错误**：隐藏班级卡、老师卡、入口卡与页脚；显示居中错误块（`--danger-wash` 圆形感叹号 + 「班级信息加载失败」+ 「网络不太稳定，请稍后再试」+ 「重试」按钮）。
- **禁用**：无表单可禁用；唯一可禁用点是「重试」/「重新加载」点击后 1 秒内变为「重试中…」的 `aria-disabled` 降级态（`--border` 底、`--hint` 字）。在 notes 里说明本页无其它可禁用控件。

右下角放原型演示条（共享 `shell.js` 提供），默认位置（无固定底栏，不写行内 bottom），默认收起，四个芯片：正常 / 空态 / 加载 / 失败。

## 交互（app.js，全部真的能点）
- 「成长相册」cell → toast「已打开成长相册」；「通知公告」cell → toast「已打开通知公告」。
- 「重试」：点击后禁用并显示「重试中…」，约 1 秒后切回正常态、同步芯片 aria-pressed，toast「已重新加载」。
- 空态「重新加载」：同样 1 秒后回到正常态并 toast。
- 所有可点元素有 `:active` 缩放/压暗反馈；不用 alert/prompt；无 JS 报错。
- 四态芯片由共享 `shell.js` 写 `data-state` 并派发 `mp:state`；app.js 监听同步即可，不要自己再绑芯片。

## 视觉与结构硬约束
- `<html lang="zh-CN" data-surface="parent">`，390px 手机视口，单列纵向流，不做桌面布局。
- 引入顺序固定：`tokens.css` → `../../_shared/shell.css` → `styles.css`。
- 颜色**只能**写 `var(--*)`，禁止裸 hex / rgb / rgba（HTML 内联 SVG 也只许 `currentColor`）；阴影只用 `var(--shadow-*)`；不得出现直角（最扁平元素 ≥ `var(--radius-sm)`，卡片 `var(--radius-card)`，按钮/标签/头像 `var(--radius-pill)`）。
- 暖色面积 ≤ 10%：页面底 `var(--bg)`、卡片 `var(--surface)`；橙色只出现在班级徽记、主班职务胶囊与可点的入口箭头，不要大面积铺渐变。
- 层级只靠字重 + 字号：标题 600 / 正文 400 / 标签 500。
- 所有可能超宽的文本禁止 `white-space: nowrap`；多列统计用 `grid-template-columns: repeat(N, minmax(0, 1fr))`；所有容器 `max-width: 100%`，页面不得横向滚动。
- 触控目标 ≥ 44px；正文 ≥ 13px；不用 hover 才出现的信息。
- 不引用任何外部资源；不写 base64 大图；单文件 < 20KB。

## 外壳（唯一实现，必须原样使用 `mp-*` 类名）
不要创建 `_shared/` 目录，也不要复制 shell.css——只写 `<link rel="stylesheet" href="../../_shared/shell.css">`，该文件由流程提供。

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
    <h1 class="mp-title">班级信息</h1>
    <div class="mp-capsule" aria-hidden="true">
      <span class="mp-capsule-dots">···</span>
      <span class="mp-capsule-divider"></span>
      <span class="mp-capsule-ring">◎</span>
    </div>
  </header>

  <main class="mp-main">…页面内容…</main>

  <div class="mp-demo" data-open="false">
    <div class="mp-demo-chips" role="group" aria-label="原型状态">
      <button class="mp-demo-chip" data-mp-state="normal" aria-pressed="true">正常</button>
      <button class="mp-demo-chip" data-mp-state="empty" aria-pressed="false">空态</button>
      <button class="mp-demo-chip" data-mp-state="loading" aria-pressed="false">加载</button>
      <button class="mp-demo-chip" data-mp-state="error" aria-pressed="false">失败</button>
    </div>
    <button class="mp-demo-toggle" type="button" aria-expanded="false">原型演示</button>
  </div>
</div>
<script src="../../_shared/shell.js"></script>
<script src="app.js"></script>
```

`styles.css` 里**不得出现任何 `.mp-*` 选择器**（`[data-state]` 状态选择器除外）。

## 输出文件（项目根目录）
1. `index.html` —— 结构
2. `styles.css` —— 样式（只写内容区；外壳来自 ../../_shared/shell.css）
3. `app.js` —— 交互
`tokens.css` 已由流程提供，不要重写、不要覆盖。每个文件 < 20KB。不要引用任何外部资源。
