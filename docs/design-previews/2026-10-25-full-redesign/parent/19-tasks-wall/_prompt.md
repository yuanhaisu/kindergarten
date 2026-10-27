你是幼儿园管理平台「养成星球」的资深移动端设计师。为家长端做一个页面的高保真 HTML 原型。

## 唯一任务
页面：**班级墙**（源码：`parent/src/pages/tasks/wall.vue`）
只做一个方向，不要 A/B 变体，不要多页面。写完 `index.html` / `styles.css` / `app.js` 三个文件立刻停止，不要提问，不要联网，不要读取自己的产物做自查循环。

## 这个页面真实的产品事实（不得增删功能）

- **入口**：家长在「亲子任务」列表点「班级墙」进入（`uni.navigateTo`）。所以这是**子页**：导航栏必须有返回箭头，标题「班级墙」，**不画 tabBar**。
- **定位**：本班公开墙，只展示老师审核通过的打卡。顶部规则文案**逐字**用：`老师确认后才会出现在这里 · 仅本班可见`。
- **数据**：`GET /parent/tasks/public-wall?student_id&page&page_size=20`，返回 `{ list, total }`。每项字段：
  `checkin_id`、`task_id`、`task_title`、`student{id,name,avatar}`、`date`（提交日 `YYYY-MM-DD`）、`content`、`media_url`、`created_at`（UTC ISO）。
- **分组**：按 `date`（不是 created_at）分段，标签为 `今天` / `昨天` / `M月D日`（如 `10月24日`）。
- **两种卡片**：有 `media_url` → 照片卡；没有 →「一句话」卡（源码用浅品牌底强调，与照片卡区分）。
  两种都包含：可选正文 `content`、孩子头像（无头像用名字首字圆底兜底）、孩子名（缺失显示「同班家长」）、时间 `HH:MM`（created_at 转本地时区）、任务名 `task_title`。
- **点击图片** = 全屏预览（`uni.previewImage`）。
- **底部**：`正在加载更多...` / `没有更多了`。
- **没有的功能**：点赞、评论、分享、任务筛选、收藏、举报、关注。一个都不要加。
- **四种状态**：首次加载（居中「加载中...」）；失败且无内容（错误文案 + 「重试」）；空（「本班还没有分享」+「家长提交、老师确认之后，会出现在这里」）；**已有内容时刷新失败**（顶部一条 inline 错误条 `{{错误}}，点击重试`，不清空已有内容——源码特意不这么做）。

## 示例数据（7 条，可直接用）

- 今天 · 乐乐（照片）· 任务「本周亲子共读」· 19:30 · `今天第一次自己读完《好饿的毛毛虫》，读到毛毛虫变蝴蝶那页，她自己也张开了手。`
- 今天 · 朵朵（照片）· 任务「生活小能手」· 18:05 · `终于学会自己系鞋带了，系完在客厅转了三圈。`
- 今天 · 小雨（无图）· 任务「家务小帮手」· 17:20 · `晚饭前主动帮奶奶摆好碗筷，说这是老师教的办法：从左往右摆。`
- 昨天 · 轩轩（照片）· 任务「专注力小挑战」· 20:10 · `60 片恐龙拼图，中间一次都没有喊人帮忙。`
- 昨天 · 小雨（照片）· 任务「自然观察日记」· 19:45 · 正文留空（演示「只有图没有字」）
- 10月24日 · 朵朵（无图）· 任务「自理小任务」· 21:00 · `睡前自己把书包收好了，还把水壶灌满放在门口。`
- 10月24日 · 乐乐（照片）· 任务「秋天在哪里」· 16:30 · `在公园捡了七种不同的叶子，回家贴成了一只小刺猬。`

## 设计方向（照这个做，不要另起风格）

方向名：**「本班的墙：日期当钉子，照片当砖」**

1. **墙头**：一行规则（小锁图标 + 12px 灰字），右侧一枚浅橙胶囊「7 份分享」。不要大标题——导航栏已经是「班级墙」，不重复。
2. **日期条**：左侧 3px 竖橙条 + 「今天」14px/600 + 右侧「3 份」11px 灰 + 一条发丝线延伸到右边缘。**吸顶**在导航栏下方（`position: sticky; top: 64px`，背景 `var(--bg)`），让长列表里始终知道在看哪一天。
3. **照片卡**：白卡，4:3 照片，任务名做白色胶囊压在照片**左下角**（`var(--surface)` 底 + `var(--accent-deep)` 字）；正文 13px/1.65；作者行：`var(--avatar-sm)` 圆头像 + 孩子名 13px/500，右侧时间 11px `var(--hint)`。
4. **一句话卡**：`var(--accent-wash)` 底、无图；正文升到 14px、行高 1.7；左上角一枚淡橙引号，任务胶囊放在正文上方。**任务胶囊在两种卡片里位置一致：都在内容块起始处**——照片卡的内容块就是照片本身。
5. **间距**：卡片间距 12px；两个日期段之间 20px；页面左右 `var(--page-pad-x)`。
6. **收尾**：「没有更多了」11px 居中 + 底部安全区留白。
7. **气质**：安静的墙面，不是广告信息流。暖色只出现在规则胶囊底、日期竖条、一句话卡底色、任务胶囊文字，整体仍是白 + 冷雾底。

## 视觉与结构硬约束

- `<html lang="zh-CN" data-surface="parent">`，390px 手机视口，单列纵向流，不做桌面布局。
- 颜色**只能**写 `var(--*)`（可用的完整清单见 `tokens.css`）；**禁止**任何裸 hex / `rgb()` / `rgba()`，包括阴影——阴影只用 `var(--shadow-*)`。
- 无直角：最扁平的块 ≥ `var(--radius-sm)`；卡片 `var(--radius-card)`；胶囊/头像 `var(--radius-pill)`。
- 正文 ≥ 13px；触控目标 ≥ 44px；点击只做 `:active` 缩放或压暗，信息不得只在 hover 时可见。
- 禁止 `white-space: nowrap` 用于可能超宽的文本（孩子名、任务名、正文都可能长）；多列用 `grid-template-columns: repeat(N, minmax(0, 1fr))`。
- 所有容器 `max-width: 100%`，页面不得出现横向滚动；长文本 `overflow-wrap: break-word`。
- 不要给 `.mp-app` 或 `body` 设 `overflow`（会破坏外壳吸顶）。
- 不用 `alert()` / `prompt()`，console 必须干净。

## 微信外壳（唯一实现，逐字照抄类名）

不要自己画状态栏/导航栏/胶囊。`index.html` 里按下面这份写，CSS 只链接共享文件：

```html
<link rel="stylesheet" href="tokens.css">
<link rel="stylesheet" href="../../_shared/shell.css">
<link rel="stylesheet" href="styles.css">
...
<body>
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
      <h1 class="mp-title">班级墙</h1>
      <div class="mp-capsule" aria-hidden="true"><span class="mp-capsule-dots">···</span><span class="mp-capsule-divider"></span><span class="mp-capsule-ring">◎</span></div>
    </header>
    <main class="mp-main"> …页面内容… </main>
    <div class="mp-demo" data-open="false">
      <div class="mp-demo-chips" role="group" aria-label="原型状态">
        <button class="mp-demo-chip" data-mp-state="normal" aria-pressed="true">正常</button>
        <button class="mp-demo-chip" data-mp-state="empty" aria-pressed="false">空态</button>
        <button class="mp-demo-chip" data-mp-state="loading" aria-pressed="false">加载</button>
        <button class="mp-demo-chip" data-mp-state="error" aria-pressed="false">失败</button>
        <button class="mp-demo-chip" data-mp-state="stale" aria-pressed="false">刷新失败</button>
      </div>
      <button class="mp-demo-toggle" type="button" aria-expanded="false">原型演示</button>
    </div>
  </div>
  <script src="../../_shared/shell.js"></script>
</body>
```

- 状态切换由 `shell.js` 写 `.mp-app[data-state]`，页面只用属性选择器接管五态，**不要另写状态 JS**。
- `styles.css` 里**不得出现任何 `.mp-*` 选择器定义**（唯一例外是 `[data-state]` 属性选择器）。不要复制、改写 shell.css。

## 必须实现的交互（`app.js`）

1. 点照片 → 打开全屏预览层（钉在 390px 宽内、`position: fixed`、深色遮罩用 `var(--fg)` + 透明度），点遮罩或右上角关闭按钮关闭；`Esc` 也能关。
2. 空态主按钮「去任务列表」→ 底部 toast「已打开任务列表」；错误态「重试」→ 900ms 后回到正常态并 toast「已重新加载」。
3. 「刷新失败」态：feed 保留，顶部 inline 错误条出现；点它 → toast「已重新加载」并回到正常态。
4. 所有可点元素 `:active` 有反馈；不加点赞/评论等任何源码没有的按钮。
5. 演示条能开合、五个 chip 能切换（shell.js 已处理，页面只需保证五态样式齐全）。

## 输出文件（项目根目录）

1. `index.html` —— 结构（含外壳，按上面的骨架）
2. `styles.css` —— 页面内容区样式（照片占位用 token 渐变 + 伪元素画，不用图片）
3. `app.js` —— 交互

`tokens.css` 已由流程提供，**不要重写、不要覆盖**。每个文件 < 20KB。不要引用任何外部资源。
