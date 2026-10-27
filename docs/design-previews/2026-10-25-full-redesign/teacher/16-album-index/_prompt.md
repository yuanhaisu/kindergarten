你是幼儿园管理平台「养成星球」的资深移动端设计师。为**微信小程序老师端**做一个页面的高保真 HTML 原型。

## 唯一任务
页面：班级相册（源码：teacher/src/pages/album/index.vue）
只做一个方向，不要 A/B 变体，不要多页面。写完 index.html / styles.css / app.js 三个文件即停止，不要提问，不要联网，不要读自己的产物做自查循环。

## 这个页面真实的产品事实（不要虚构其它能力）

老师端班级相册页：老师按分类浏览本班相册（每册含标题、分类、照片数、日期、封面），并从本页发起上传。照片按「相册 → 月份」组织。

**源码字段（唯一真相，不要加别的字段）**：
- `albums[]`：`id`、`title` 相册名、`category` = 活动 | 日常 | 作品 | 节日、`photo_count` 照片数、`date` 日期、`cover` 封面图 URL（当前为空，用占位）。
- `stats`：`total_photos` 照片总数、`monthly_new` 本月新增、`album_count` 相册数。源码默认值：386 / 42 / 12。
- `categories`：全部 / 活动 / 日常 / 作品 / 节日，默认「全部」；筛选是**客户端过滤**，不是新接口。
- 源码里的交互函数：`selectCategory(cat)`；`openUpload()` → toast「上传功能开发中」；`goAlbumDetail(id)` → toast「查看相册 {id}」；`openSearch()` → toast「搜索功能开发中」（**模板里没有渲染搜索入口，本原型也不要画搜索**）。
- 列表加载：`loadAlbums()` 目前恒返回空数组（**老师端相册 API 尚未提供**），所以线上默认是「暂无相册」；本原型默认展示正常态样例数据。
- 入口关系：本页由老师工作台 / 更多页 `navigateTo` 进入 → **必须画返回箭头**；导航标题「班级相册」。老师端**没有 tabBar**（任何老师端页面都不要画）。
- 页面没有角色/权限差异，没有「置顶」「公开/私密」「审核」等字段，不要发明。

## 样例数据（请原样使用，数字必须自洽）

当前：2026年10月26日 周一。班级：小一班。老师：王老师。

统计：照片总数 **386**，本月新增 **42**，相册数 **12**。

相册（列表展示最近 6 册；10 月 4 册合计 12+9+14+7 = 42 张，正好等于「本月新增 42」）：

**2026年10月 · 4 册**
1. 秋日郊游 · 活动 · 12张 · 10月24日
2. 美术课作品展 · 作品 · 9张 · 10月21日
3. 晨间活动日常 · 日常 · 14张 · 10月16日
4. 重阳节手工 · 节日 · 7张 · 10月11日

**2026年9月 · 2 册**
5. 中秋节做月饼 · 节日 · 30张 · 9月25日
6. 开学第一周 · 日常 · 48张 · 9月4日

分类筛选后的可见册数：全部 6、活动 1（秋日郊游）、日常 2（晨间活动日常 / 开学第一周）、作品 1（美术课作品展）、节日 2（重阳节手工 / 中秋节做月饼）。筛选后若某个月份分组没有可见相册，整个分组头要隐藏。

## 页面结构（从上到下）

1. **微信外壳**：状态栏 + 导航栏「班级相册」+ 左侧返回箭头 + 右侧胶囊。**不画 tabBar。**
2. **统计卡**（一张白底卡，内部 **3 等分**）：照片总数 386 / 本月新增 42 / 相册数 12。每列 = 小图标块 + 标签 + 数字；列之间用细竖线分隔。用 `grid-template-columns: repeat(3, minmax(0, 1fr))`，数字不得被挤换行。
3. **分类筛选**：一行 5 个等宽胶囊「全部 / 活动 / 日常 / 作品 / 节日」，用 `grid-template-columns: repeat(5, minmax(0, 1fr))`。选中项用品牌色实底或浅底 + 品牌色文字（推荐浅底 `var(--accent-wash)` + `var(--accent-deep)` 文字 + 1px `var(--accent)` 描边，克制）。点击真的过滤下面的相册。
4. **月份分组 + 相册卡**（单列，组间留白）：
   - 分组头：左侧「2026年10月」，右侧「4 册」，下面一条细分割线；筛选后数字与分组显隐同步更新。
   - 相册卡（白底圆角卡）：
     - **封面区 = 三列照片网格**，`grid-template-columns: repeat(3, minmax(0, 1fr))`，三个正方形格子（`aspect-ratio: 1 / 1`），格间距 6px，每格圆角 `var(--radius-lg)`，**用 CSS 渐变块占位，禁止外链图片、禁止 base64 图片**。三格用同一册的三种相邻渐变（可参考下面的渐变配方）。
     - 相册信息行：标题（15px / 600，可换行，不截断） + 右侧「12张」小胶囊（`var(--surface)` 底或 `var(--accent-wash)` 浅底，中性优先）。
     - 底部元信息行：日期「10月24日」在左，分类标签在右（活动/日常/作品/节日各用一种语义浅底 + 深色文字：活动=success、日常=info、作品=accent、节日=warning；见下方 token 规则）。
     - 点相册信息区域 → toast「查看相册「秋日郊游」」（对应源码 `goAlbumDetail` 占位；**不要跳转**）。
     - 点任意一张封面照片 → **底部弹层模拟大图**：遮罩 + 底部升起的白色圆角面板，含拖动条、相册名 + 日期、一块 4:3 的大渐变图（与所点格子同色系放大）、「第 1 张 / 共 12 张」计数、左右切换按钮与关闭按钮。到第一张时「上一张」明显禁用、到最后一张时「下一张」明显禁用（这就是本页的禁用态）。遮罩点击、关闭按钮、Esc 都能关闭。打开弹层不能让页面横向溢出。
5. **底部固定操作栏**：主按钮「＋ 上传照片」（品牌渐变，高度 48px，宽度撑满内容区）。点击 toast「上传功能开发中」（源码文案原样）。内容区底部留 `calc(76px + env(safe-area-inset-bottom))` 避让；演示条用行内 `style="bottom: calc(64px + 12px + env(safe-area-inset-bottom))"`。

> 说明：源码 `openUpload()` 已存在但模板未渲染入口；按批次要求把上传放在底部固定主按钮，notes 里会标注需产品确认。

## 照片渐变占位配方（只用 token，禁止裸色）

六个渐变类，可在各册三格间轮换，保证同册三格是同一色系的不同深浅：
```css
.ph-1 { background: linear-gradient(150deg, var(--accent-soft), var(--accent-wash)); }
.ph-2 { background: linear-gradient(150deg, var(--info-wash), color-mix(in srgb, var(--info) 24%, var(--surface))); }
.ph-3 { background: linear-gradient(150deg, var(--success-wash), color-mix(in srgb, var(--success) 22%, var(--surface))); }
.ph-4 { background: linear-gradient(150deg, var(--warning-wash), color-mix(in srgb, var(--warning) 22%, var(--surface))); }
.ph-5 { background: linear-gradient(150deg, color-mix(in srgb, var(--accent) 26%, var(--surface)), var(--accent-wash)); }
.ph-6 { background: linear-gradient(150deg, var(--info-wash), var(--success-wash)); }
```
格子里可以放一个低透明度的小相机/照片线框图标（`currentColor` 用 `var(--hint)`），不要放文字标签。

## 状态覆盖（数据区四态 + 演示条）

用共享 `mp-demo` 演示条（正常 / 空态 / 加载 / 失败），由 `shell.js` 把状态写到 `.mp-app` 的 `data-state` 上：
- **正常**：统计卡 + 分类筛选 + 两个月份分组 + 6 张相册卡 + 底部上传按钮。
- **空态（还没有照片）**：隐藏统计卡、分类筛选、月份分组；显示居中空态：图标 + 「还没有照片」+ 一句「上传后家长可在成长时光里看到」（**只一句**）+ 主按钮「上传照片」（点击 toast「上传功能开发中」）。底部固定上传按钮保留。
- **加载态**：隐藏统计卡数字与相册卡；显示统计骨架（3 列）+ 筛选骨架 + 2 张相册卡骨架（每张含三格正方形占位块 + 两行文本条），带轻微呼吸动画（`prefers-reduced-motion` 下降级为静态）。
- **失败态**：隐藏统计卡、分类筛选、列表；显示「相册加载失败」+「网络异常，未获取到照片」+「重试」按钮（点击后 600ms 回到正常态）。
- **禁用态**：大图弹层里第一张的「上一张」、最后一张的「下一张」真 disabled（`disabled` 属性 + 明显降级），其余按钮不得出现假禁用。

## 视觉与结构硬约束（逐条照做）

- `<html lang="zh-CN" data-surface="teacher">`，390px 手机视口，单列纵向流，不做桌面布局。
- 引入顺序固定：
  ```html
  <link rel="stylesheet" href="tokens.css">
  <link rel="stylesheet" href="../../_shared/shell.css">
  <link rel="stylesheet" href="styles.css">
  ```
  页面 `styles.css` 里**不得出现任何 `.mp-*` 选择器定义**（唯一例外是 `[data-state="..."]` 前缀的状态选择器）。不要复制、不要改写 shell.css；状态栏 SVG 必须使用下面给出的原样代码。
- 颜色只能写 `var(--*)`，**禁止任何裸 hex / rgb / hsl**；无直角（最扁平元素 ≥ `var(--radius-sm)`，按钮/标签/胶囊用 `var(--radius-pill)`，卡片用 `var(--radius-card)`）；阴影只用 `var(--shadow-sm|md|lg)`。
- 暖色面积 ≤ 10%：大面积是 `var(--bg)` / `var(--surface)`；品牌渐变只允许出现在底部「上传照片」主按钮和弹层里的主/次按钮上；活动/日常/作品/节日标签用各自的 `-wash` 浅底 + 深色文字，不要用大面积实色。
- 语义色只用 token：活动 `var(--success)` + `var(--success-wash)`；日常 `var(--info)` + `var(--info-wash)`；作品 `var(--accent-deep)` + `var(--accent-wash)`；节日 `var(--warning)` + `var(--warning-wash)`；失败 `var(--danger)` + `var(--danger-wash)`。
- 触控目标 ≥ 44px；正文 ≥ 13px（`var(--text-base)`）；标题 15–16px、字重 600；统计数字 20px 左右、字重 700。
- **禁止 `white-space: nowrap` 用于可能超宽的文本**（相册标题、日期、标签都可能长，必须正常换行或省略号）。
- 所有多列统计/网格用 `grid-template-columns: repeat(N, minmax(0, 1fr))`；所有容器 `max-width: 100%`；页面不得出现横向滚动。
- 弹层用 `position: fixed`，内部面板 `width: min(390px, 100%)` 并水平居中，不得越过视口左右边界。
- 无 JS 报错；不用 alert / prompt；点击必须有 `:active` 反馈（缩放或压暗）；不要 hover 才出现的信息。
- 每个文件 < 20KB；不引用任何外部资源（字体、图片、CDN 都不行）。
- 不写营销话术；按钮文案用 2–4 字动词。

## 外壳标记（原样照抄，尺寸不许改）

```html
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
      <h1 class="mp-title">班级相册</h1>
      <div class="mp-capsule" aria-hidden="true">
        <span class="mp-capsule-dots">···</span>
        <span class="mp-capsule-divider"></span>
        <span class="mp-capsule-ring">◎</span>
      </div>
    </header>

    <main class="mp-main">…页面内容…</main>

    <div class="mp-demo" data-open="false" style="bottom: calc(64px + 12px + env(safe-area-inset-bottom))">
      <div class="mp-demo-chips" role="group" aria-label="原型状态">
        <button class="mp-demo-chip" data-mp-state="normal" aria-pressed="true">正常</button>
        <button class="mp-demo-chip" data-mp-state="empty" aria-pressed="false">空态</button>
        <button class="mp-demo-chip" data-mp-state="loading" aria-pressed="false">加载</button>
        <button class="mp-demo-chip" data-mp-state="error" aria-pressed="false">失败</button>
      </div>
      <button class="mp-demo-toggle" type="button" aria-expanded="false">原型演示</button>
    </div>

    <div class="toast" role="status" aria-live="polite"></div>
  </div>
  <script src="../../_shared/shell.js"></script>
  <script src="app.js"></script>
</body>
```

## 输出文件（项目根目录）
1. index.html —— 结构
2. styles.css —— 样式（引用 tokens.css）
3. app.js —— 交互
tokens.css 已由流程提供，不要重写，不要覆盖。每个文件 < 20KB。不要引用任何外部资源。
