你是幼儿园管理平台「养成星球」的资深移动端设计师。为家长端做一个页面的高保真 HTML 原型。

## 唯一任务
页面：孩子信息（源码：parent/src/pages/student/index.vue）
只做一个方向，不要 A/B 变体，不要多页面。写完全部文件即停止：不要提问，不要联网，不要读取自己刚写的产物做自查，不要覆盖 tokens.css。

## 这个页面真实的产品事实

**定位**：核对孩子档案的信息页。入口「我的」→「我的孩子」`navigateTo` 进入，**子页：导航栏必须有返回箭头**，标题「孩子信息」，**不画 tabBar**。

**数据来源**：
- `GET /parent/student/profile` → `{ name, avatar, gender(1男/2女), birthday, age, health_note, class_name, enroll_date }`
- `GET /parent/health/records` → `{ list: [{ check_date, overall_status(0健康/1需观察/2劝返就医), overall_status_label, temperature, abnormal_items:[{ part_label, status_label }], remark, is_isolated }] }`；仅最近 7 天、已完成
- `GET /parent/students` → `{ list: [{ id, name, class_name, is_current }] }`；`DELETE /parent/students/:id` 解绑

**源码区块顺序**：
1. 档案卡：姓名（大标题）→ 班级 / 性别 / 生日 / 年龄 / 健康备注（**有值才显示该行**）
2. 最近晨检卡：每条 日期 + 状态胶囊 +「异常：喉咙发红」+「体温 37.4°C」+ 隔离提示 + 备注；空时「近 7 天暂无晨检记录」；**晨检有独立的行内失败态**，不许拖垮整页档案
3. 已绑定的孩子卡：**仅当绑定超过 1 个孩子才出现**；每行 姓名（班级）+「解绑」，解绑前有微信风格二次确认

**源码没有的**：编辑档案、补录、拨号、相册入口、切换孩子、下拉刷新。一律不要发明。

## 示例数据（用这一组）
孩子：**乐乐** / 小一班 / 女 / 生日 2022-05-18 / 4岁 / 入园 2025-09-01 / 健康备注「对花生过敏，请避免含花生制品」
晨检（3 条，倒序）：
1. 10月23日 周五 · 需观察 · 异常：喉咙发红 · 体温 37.4°C · 备注「喉咙略红，提醒多喝水」
2. 10月22日 周四 · 健康 · 体温 36.5°C
3. 10月21日 周三 · 健康 · 体温 36.4°C
已绑定：乐乐（小一班，当前）、小雨（大二班）
头像：禁止外链图片，用**首字「乐」**落在柔和圆形里。

## 视觉方向：「成长护照」
一页可核对的档案，靠排版而不是靠卡片堆叠：
- **身份带**：64px 首字头像（`var(--accent-wash)` 底 + 1px `var(--accent-soft)` 边 + `var(--accent-deep)` 字）+ 20px/600 姓名 + 安静的班级胶囊「小一班」（`var(--bg)` 底 / `var(--fg-2)` 字 / 12px）。
- **事实格**：同卡内 2×2 grid（`repeat(2, minmax(0, 1fr))`），四个 info-cell：性别 / 生日 / 年龄 / 入园时间；每格「上标签 11–12px `var(--muted)` + 下值 15px `var(--fg)`」，1px `var(--border)` 细线分隔。
- **健康备注 = 收进来的敏感信息**：不用警示色、不加粗、不用大图标。档案卡底部一块 `var(--bg)` 内嵌块（`var(--radius-xl)`），左侧 16px 盾牌线稿 `var(--muted)`，右侧 11px 字距标签「健康备注」+ 13px/400 `var(--fg-2)` 正文；整块无阴影、无橙色。是「安静地放在那里」，不是「被标红」。
- **最近晨检**：标题在卡外（+ 右侧「近 7 天」），卡内 3 行、1px 分隔线；每行：日期 14px/600 + 状态胶囊；异常 13px `var(--fg-2)`；体温 13px `var(--muted)` `tabular-nums`；备注 13px `var(--muted)`。
- **已绑定的孩子**：标题「已绑定的孩子」+ 右侧「2 位」；每行 姓名 + 班级（`var(--muted)`）+ 中性「当前」小胶囊 + 右侧「解绑」小胶囊（`--danger-wash` / `--danger` / `--radius-pill`）。
- 页脚一行小字：「档案信息由园所统一维护」。

状态胶囊：健康 `--success-wash`/`--success`，需观察 `--warning-wash`/`--warning`，劝返就医 `--danger-wash`/`--danger`。

## 状态覆盖（`.mp-app[data-state]` 纯 CSS 切换）
五个芯片：**正常 / 空态 / 加载 / 失败 / 晨检失败**
- **正常**（默认）：上述全部数据。
- **空态**（部分字段缺失，不是整页无数据）：身份带与班级仍在；生日、年龄、入园时间显示「—」（`var(--hint)`）；**健康备注块整块隐藏**；晨检卡换空块（线框图标 +「近 7 天暂无晨检记录」）；**已绑定的孩子卡整卡隐藏**（只剩 1 个孩子）；档案卡下加柔和提示行「部分档案信息暂未填写」+ 文字按钮「刷新档案」。
- **加载**：身份带骨架 + 事实格 4 块 + 晨检 3 行 + 孩子 2 行骨架，呼吸动画；区块计数隐藏。
- **失败**：隐藏所有卡片与页脚，整页居中错误块（`--danger-wash` 圆形感叹号 +「档案加载失败」+「网络不太稳定，请稍后再试」+「重试」）。
- **晨检失败**：其余区块正常，**只有晨检卡**换行内失败：「晨检记录加载失败」+「点击重试」，证明晨检失败不拖垮整页。
- **禁用**：无表单；唯一降级点是「重试 / 刷新档案 / 点击重试」点击后 1 秒 `aria-disabled="true"`（`--border` 底、`--hint` 字、「重试中…」）。notes 里说明本页无其它可禁用控件。

## 交互（app.js，全部真的能点）
- 「解绑」→ 微信风格居中对话框（「确认解绑」+「确定解除与 乐乐（小一班）的绑定吗？」+ 取消 / 确定）；确定后关闭并 toast「已解绑」；取消关闭。遮罩用 `var(--fg)` + `opacity`，**不要写 rgba**。
- 晨检「点击重试」→ 卡内 1 秒小圆环 → 恢复记录 + toast「晨检已更新」。
- 失败态「重试」/ 空态「刷新档案」→ 禁用 1 秒 → 正常态 + toast「已重新加载」。
- 返回箭头 → toast「返回上一页」。
- 可点元素有 `:active` 缩放/压暗；不用 alert/prompt；无 JS 报错。
- 芯片由共享 `shell.js` 写 `data-state` 并派发 `mp:state`；app.js 只监听同步 `aria-busy` 与 `aria-pressed`，**不要自己再绑芯片**。

## 视觉与结构硬约束
- `<html lang="zh-CN" data-surface="parent">`，390px 手机视口，单列纵向流，不做桌面布局。
- 引入顺序固定：`tokens.css` → `../../_shared/shell.css` → `styles.css`。
- 颜色**只能**写 `var(--*)`，禁止裸 hex / rgb / rgba（内联 SVG 只许 `currentColor`）；阴影只用 `var(--shadow-*)`；无直角（≥ `--radius-sm`，卡片 `--radius-card`，按钮/标签/头像 `--radius-pill`）。
- 暖色 ≤ 10%：橙色只出现在头像圈与主按钮，**健康区绝对不要橙色**。
- 层级只靠字重 + 字号：标题 600 / 正文 400 / 标签 500。
- 所有可能超宽的文本禁止 `white-space: nowrap`；多列统计用 `repeat(N, minmax(0, 1fr))`；所有容器 `max-width: 100%`，页面不得横向滚动。
- 触控目标 ≥ 44px；正文 ≥ 13px；不用 hover 才出现的信息。
- 不引用任何外部资源；不写 base64 大图；单文件 < 20KB。

## 外壳（唯一实现，原样使用 `mp-*` 类名）
不要创建 `_shared/`，不要复制 shell.css；只写 `<link rel="stylesheet" href="../../_shared/shell.css">`（由流程提供）。

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
    <h1 class="mp-title">孩子信息</h1>
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
      <button class="mp-demo-chip" data-mp-state="health-error" aria-pressed="false">晨检失败</button>
    </div>
    <button class="mp-demo-toggle" type="button" aria-expanded="false">原型演示</button>
  </div>
</div>
<script src="../../_shared/shell.js"></script>
<script src="app.js"></script>
```

`styles.css` 里**不得出现任何 `.mp-*` 选择器**（`[data-state]` 状态选择器除外）。

## 输出（项目根目录）
`index.html` 结构、`styles.css` 样式（只写内容区；外壳来自 ../../_shared/shell.css）、`app.js` 交互。`tokens.css` 由流程提供，不要重写或覆盖。每个文件 < 20KB，不引用任何外部资源。
