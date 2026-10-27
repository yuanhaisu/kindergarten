# 接送记录 · 老师端（teacher）

## 1. 任务

为「养成星球」老师端小程序设计**接送记录**页的高保真原型（HTML/CSS/JS，390px 宽）。

- 源码：`teacher/src/pages/pickup/records.vue`
- 接口：`GET /api/v1/teacher/pickup/records?date=&student_id=&page=&page_size=`（老师端口径，按班级过滤）
- 设计系统：`user:nurture-planet-local`，`<html data-surface="teacher">`
- 外壳：链接 `../../_shared/shell.css`，使用 canonical `mp-*` 标记
- 不画 tabBar（老师端全局无 tabBar）
- 画返回箭头（本页是 `uni.navigateTo` 进入的子页）

**这一页是什么**：老师按日期回看本班当天的接送流水——谁几点被谁接走、用什么方式核验、有没有例外/异常。它是「查账」页，不是操作页，因此没有底部主按钮。

**设计方向**：园门口的接送流水账。暖色纸感底 + 一条细竖线作为账页边栏，时间在左侧栏位对齐（tabular numerals），姓名与事实在右侧成行；日期卡像账本翻页器的页眉。不要做成通用后台表格，也不要用冷灰卡片堆叠。

---

## 2. 源码事实（必须遵守，不得自由发挥）

### 2.1 源码已有的

- 默认日期为「今天」；`prevDay` / `nextDay` 左右切换日期（源码同时保留了一个 `▾`，但点击无行为）
- 顶部摘要三项：今日接送 **13 人** / 已接走 **4 人** / 入园签到 **9 人**
- 日期展示格式 `YYYY年M月D日`
- 列表卡片字段：时间、类型（送入 / 接走）、状态、学生头像、姓名、班级、接送人、例外 badge、核验方式（扫码 / 手动）、理由、异常 tag
- 状态：加载中「加载中...」、空「暂无接送记录」、失败用 toast
- 一个「筛选」按钮，点击只 toast「筛选功能开发中」——**是死按钮**

### 2.2 源码没有的（本次不要加）

- **没有「只看异常」参数**：后端 `GET /teacher/pickup/records` 只接受 `date` / `student_id` / `page` / `page_size`。**不要画「只看异常」chip。**
- 没有底部导出 / 统计 / 批量操作入口 → **不要画底部操作栏**
- 没有搜索、没有分页交互、没有列表项点击行为
- 没有多日聚合：一次只查一天
- 没有 tabBar

### 2.3 本次有意做的修正（写进 notes，不算发明）

| 源码现状 | 原型做法 | 理由 |
|---|---|---|
| 摘要单位「人」 | 改为「条」 | 统计的是记录数不是人数；13/4/9 是条数 |
| 死按钮「筛选」 | 删除 | 无对应能力 |
| 死箭头 `▾` | 点日期打开底部日期选择弹层 | 让已有日期筛选真正可操作；同时保留左右切换 |
| 扫码 `#9333EA` / 手动 `#2563EB` 硬编码 | 扫码 = `--info`，手动 = 中性描边标签 | 去自造色，核验方式用轻量标签 |
| 列表行不可点 | 点行打开「记录详情」底部弹层 | 详情接口 `GET /teacher/pickup/records/:id` 已存在，`fetchTeacherPickupRecord` 已封装但页面未用；只展示接口真实字段，不新增业务能力 |
| 时间轴只有一条竖线 | 左侧时间栏 + 竖线 + 圆点，按 `recorded_at DESC` 排列 | 记录本身就是时间序列 |

> 详情弹层只放这些真实字段：类型、记录时间、接送人（仅接走）、核验方式、授权状态（仅接走）、异常备注（仅异常）、理由（有 `remark` 时）。**不要放接送人手机号、操作人、GPS 等源码未展示的字段。**

---

## 3. 信息架构（自上而下）

```
[statusbar 20]
[navbar 44 · 返回箭头 · 接送记录 · 胶囊]
[main]
  1. 日期卡
     ‹   10月26日  周一  [今天]  ⌄   ›
     ─────────────────────────────
     13 条 记录 | 4 条 接走 | 9 条 入园签到
  2. 分组头
     10月26日 周一 · 共 13 条            小一班
  3. 时间轴卡（13 行，按时间倒序）
     17:12 | ● 朵朵  小一班   [手动]
           |   接走 · 张伟（舅舅）  [例外]
           |   理由 临时委托舅舅接走，已电话核实母亲
     17:05 | ● 团团  小一班   [扫码]
           |   接走 · 李梅（妈妈）  [异常]   ← 整行 --danger-wash
           |   理由 家长称孩子发烧，需提前接走
     ...
     07:55 | ● 安安  小一班   [扫码]
           |   送入
  4. 空态 / 加载骨架 / 失败块（三选一，随 data-state）
[无底部操作栏]
[mp-demo 右下角]
[toast]
[日期选择弹层 / 记录详情弹层]
```

- 页面左内边距 14px，右 14px，上 12px，底部 `calc(96px + env(safe-area-inset-bottom))` 给 demo 条留位
- 卡片之间间距 12px；分组头与卡片间距 8px
- 分组头只出现一组（一次只查一天），格式必须含 `· 共 N 条`

---

## 4. 外壳（逐字复制，不得改写）

`<head>` 里按顺序：

```html
<link rel="stylesheet" href="tokens.css">
<link rel="stylesheet" href="../../_shared/shell.css">
<link rel="stylesheet" href="styles.css">
```

`<body>` 直接子元素是 `<div class="mp-app" data-state="normal">`，内部依次是状态栏、导航栏、`<main class="mp-main page">`、demo 条、toast、弹层；最后是脚本：

```html
<script src="../../_shared/shell.js"></script>
<script src="app.js"></script>
```

状态栏（`9:41`、三枚 SVG）与导航栏逐字使用下面这段，只改标题与返回 aria-label：

```html
<div class="mp-statusbar" aria-hidden="true">
  <span class="mp-time">9:41</span>
  <span class="mp-status-icons">
    <svg width="17" height="11" viewBox="0 0 17 11" fill="currentColor"><rect x="0" y="7" width="3" height="4" rx="1"/><rect x="4.5" y="5" width="3" height="6" rx="1"/><rect x="9" y="2.5" width="3" height="8.5" rx="1"/><rect x="13.5" y="0" width="3" height="11" rx="1"/></svg>
    <svg width="15" height="11" viewBox="0 0 15 11" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round"><path d="M1 3.6C2.8 2 5.3 1 7.5 1s4.7 1 6.5 2.6"/><path d="M3.5 6.1c1.1-1 2.4-1.5 4-1.5s2.9.5 4 1.5"/><path d="M6 8.6c.5-.4 1-.6 1.5-.6s1 .2 1.5.6"/><circle cx="7.5" cy="10" r=".7" fill="currentColor"/></svg>
    <svg width="24" height="11" viewBox="0 0 24 11" fill="none"><rect x=".5" y=".5" width="20" height="10" rx="3" stroke="currentColor" stroke-opacity=".5"/><rect x="2" y="2" width="15" height="7" rx="1.8" fill="currentColor"/><path d="M22 3.5v4c.9-.3 1.4-1 1.4-2s-.5-1.7-1.4-2Z" fill="currentColor" fill-opacity=".5"/></svg>
  </span>
</div>

<header class="mp-navbar">
  <button class="mp-back" id="backBtn" type="button" aria-label="返回接送管理">
    <svg width="20" height="20" viewBox="0 0 20 20" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M12.5 4.5 7 10l5.5 5.5"/></svg>
  </button>
  <h1 class="mp-title">接送记录</h1>
  <div class="mp-capsule" aria-hidden="true">
    <span class="mp-capsule-dots">···</span>
    <span class="mp-capsule-divider"></span>
    <span class="mp-capsule-ring">◎</span>
  </div>
</header>
```

demo 条（**不要加 inline style**，本页没有底部操作栏）：

```html
<div class="mp-demo" data-open="false">
  <div class="mp-demo-chips" role="group" aria-label="原型状态">
    <button class="mp-demo-chip" data-mp-state="normal" aria-pressed="true">正常</button>
    <button class="mp-demo-chip" data-mp-state="empty" aria-pressed="false">空态</button>
    <button class="mp-demo-chip" data-mp-state="loading" aria-pressed="false">加载</button>
    <button class="mp-demo-chip" data-mp-state="error" aria-pressed="false">失败</button>
  </div>
  <button class="mp-demo-toggle" type="button" aria-expanded="false">原型演示</button>
</div>
```

---

## 5. 组件与视觉规范

### 5.1 颜色（只用 token）

- 页面底：`var(--bg)`（`#f7f3ea` 暖米色，由 `.mp-app` 提供）
- 卡片：`var(--surface)`（`#fffaf0`）+ `var(--shadow-sm)`
- 主文字 `var(--fg)`，次文字 `var(--fg-2)`，辅助 `var(--muted)`，更弱 `var(--hint)`
- 强调橙 `var(--accent)` / `var(--accent-deep)` / `var(--accent-wash)`；**暖色面积不超过整页 10%**，只用在：日期选中底、今天 pill、头像底、空态主按钮
- 语义色只按含义用：
  - 接走（完成）= `var(--success)` + `var(--success-wash)`
  - 送入（中性事件）= `var(--info)` + `var(--info-wash)`
  - 例外 = `var(--warning)` + `var(--warning-wash)`
  - 异常 = `var(--danger)` + `var(--danger-wash)`，**整行异常才用，其他任何地方不要用 danger**
- 描边：`var(--border)`，细线一律用伪元素 + `opacity: .35 ~ .45`，避免冷灰线太重
- 禁止自造颜色（含紫色、蓝紫、任何 hex/rgb）

### 5.2 字体与数字

- 字体继承外壳 `var(--font-sans)`
- 时间、计数、统计数字：`font-variant-numeric: tabular-nums`
- 字号：
  - 日期主文 16px / 600（`--text-xl`）
  - 摘要数字 20px / 700（`--text-3xl`）
  - 学生姓名 15px / 600（`--text-lg`）
  - 类型 / 接送人 12–13px
  - 理由 12px / `var(--leading-body)`
  - 标签 10–11px / 500
- 理由等长文本必须可换行，**全文件禁止 `white-space: nowrap`**

### 5.3 日期卡

```
<section class="date-card" data-od-id="pickup-records-date">
  <div class="date-row">
    <button class="date-arrow" id="prevDay" aria-label="前一天">‹</button>
    <button class="date-current" id="dateBtn" aria-haspopup="dialog" aria-controls="dateSheet">
      <svg calendar 15/> <span class="date-main">10月26日</span>
      <span class="date-week">周一</span>
      <span class="today-pill" id="todayPill">今天</span>
      <svg chevron-down 14/>
    </button>
    <button class="date-arrow" id="nextDay" aria-label="后一天" disabled>›</button>
  </div>
  <div class="summary" ...>
    <div class="summary-cell"><span class="summary-value"><b>13</b><i>条</i></span><span class="summary-label">记录</span><span class="summary-skel"></span></div>
    ... 接走 4 条 ...
    ... 入园签到 9 条 ...
  </div>
</section>
```

- 卡：`border-radius: var(--radius-card)`（12px），`box-shadow: var(--shadow-sm)`，内边距 `6px 6px 0`
- 日期行高 48px；左右箭头 40×40、圆形、透明底，`:active` 用 `var(--accent-wash)`；`disabled` 时 `color: var(--hint)` + `opacity: .4` + 不响应
- 日期按钮：flex 1，高 40，胶囊圆角，默认透明；`:active` / 打开弹层时 `var(--accent-wash)`；内容水平居中，gap 6
- `今天` pill：仅当选中今天显示，10px、高 18、`var(--accent-wash)` 底 + `var(--accent-deep)` 字
- 下划线与摘要：摘要 `display: grid; grid-template-columns: repeat(3, minmax(0, 1fr))`，上边用 1px 伪元素细线，padding `10px 10px 12px`
- 每个 cell 左对齐，数字一行（20px + 单位 10px muted），标签下一行 11px muted；后两格左侧加 1px 竖细线（伪元素，opacity .35）
- 加载态数字换成骨架条（`.summary-skel`），高度 18px，宽度 40–56px

### 5.4 分组头

```
<div class="group-head normal-only">
  <h2 class="group-title"><span id="groupDate">10月26日 周一</span><span class="group-sep">·</span><span class="group-count" id="groupCount">共 13 条</span></h2>
  <span class="group-class">小一班</span>
</div>
```

- 外边距 `14px 2px 8px`；标题 12px / 500 `var(--fg-2)`，数量 `var(--muted)`，中间点前后各 4px
- 右侧「小一班」11px `var(--muted)`
- 允许 `flex-wrap: wrap`，窄屏时班级可换行到下一行，**不得溢出**

### 5.5 时间轴卡

```
<div class="timeline-card normal-only" id="recordList" aria-live="polite">
  <button class="tl-row" type="button" data-idx="0" aria-label="查看 朵朵 17:12 的接送记录">
    <span class="tl-time">17:12</span>
    <span class="tl-rail"><span class="tl-dot tl-dot--pickup"></span></span>
    <span class="tl-body">
      <span class="tl-top">
        <span class="tl-avatar" aria-hidden="true">朵</span>
        <span class="tl-name">朵朵</span>
        <span class="tl-class">小一班</span>
        <span class="tl-method tl-method--manual">
          <svg hand 11/> 手动
        </span>
      </span>
      <span class="tl-meta">
        <span class="tl-type tl-type--pickup">接走</span>
        <span class="tl-dotsep">·</span>
        <span class="tl-person">张伟（舅舅）</span>
        <span class="tl-flag tl-flag--override">例外</span>
      </span>
      <span class="tl-remark"><span class="tl-remark-label">理由</span>临时委托舅舅接走，已电话核实母亲</span>
    </span>
  </button>
  ...
</div>
```

- 卡：`border-radius: var(--radius-card)`，`box-shadow: var(--shadow-sm)`，`overflow: hidden`，上下 padding 4px
- 行：`display: grid; grid-template-columns: 40px 14px minmax(0, 1fr); column-gap: 8px; width: 100%; padding: 0 14px; text-align: left; background: transparent; border: 0;`，`:focus-visible` 用 `outline: 2px solid var(--accent); outline-offset: -2px`
- 时间：12px `var(--muted)`，右对齐，`padding-top: 15px`，tabular-nums
- 竖线：`.tl-rail` 内绝对定位 1px 线，铺满行高；首行 `top: 18px`，末行 `bottom: calc(100% - 18px)`；圆点 8×8 在 `top: 14px`，`box-shadow: 0 0 0 3px var(--surface)` 把线断开
- 圆点颜色：接走 `var(--success)`，送入 `var(--info)`
- 行体：`padding: 12px 0 13px; min-width: 0`；相邻行之间用伪元素 1px 细线（左右各让 14px，`var(--border)`，opacity .4）
- 第一行：头像 30×30 圆形，`var(--accent-wash)` 底，`var(--accent-deep)` 字 12px/600，单字；姓名 15px/600 紧随，班级 11px `var(--hint)`；核验标签推到最右（`margin-left: auto`）
- 核验标签：高 20、胶囊、10px/500、gap 3、内联 SVG 11px
  - 扫码：`var(--info-wash)` 底 + `var(--info)` 字
  - 手动：透明底 + 1px `var(--border)` 描边 + `var(--muted)` 字
- 第二行：`display: flex; flex-wrap: wrap; align-items: center; gap: 5px; margin-top: 5px; font-size: 12px`
  - 类型：接走 `var(--success)`，送入 `var(--info)`，600
  - 接送人：`var(--fg-2)`；送入没有接送人，这一行只有「送入」
  - 例外：`var(--warning-wash)` 底 + `var(--warning)` 字，10px，高 17，胶囊
  - 异常：**实心** `var(--danger)` 底 + `var(--on-brand)` 字，10px，高 17，胶囊
- 理由行：`margin-top: 5px; font-size: 12px; color: var(--muted); line-height: var(--leading-body); overflow-wrap: anywhere;`
  - 前缀「理由」用 10px、`var(--hint)`、`margin-right: 4px`，不加重
- 异常整行：`.tl-row--danger { background: var(--danger-wash); }`，并加 `box-shadow: inset 3px 0 0 var(--danger)`；`:active` 用 `filter: brightness(.97)`
- 行内任何文本容器都要 `min-width: 0`，姓名/接送人可换行

### 5.6 空 / 加载 / 失败

三块都放在 `<main>` 内，由 `.mp-app[data-state]` 控制显示；`normal` 时只显示日期卡 + 分组头 + 时间轴，非 `normal` 时隐藏 `.normal-only`。日期卡在三种状态下都保留（用户仍可切日期）。

- **空态** `.state-empty`：垂直居中块，padding `40px 20px`
  - 56px 圆形图标底 `var(--accent-wash)`，内联 SVG 28px `var(--accent-deep)`（剪贴板/清单）
  - 标题「这天没有接送记录」15px/600
  - 说明 12px `var(--muted)`，动态：周末为「10月25日 周日 · 全园休息」，工作日为「换个日期看看，或下拉刷新」
  - 主按钮「回到今天」：高 40、胶囊、`var(--accent-gradient)` 底 + `var(--on-brand)` 字、`var(--shadow-sm)`
- **加载** `.state-loading`：4 行骨架，每行结构与真实行一致（时间条 32×12、圆点、头像圆 30、两条文字条、标签块），颜色只用 `var(--border)` 的不同透明度，`@keyframes` 在 0.4 / 0.18 之间脉冲；摘要数字同时换骨架
- **失败** `.state-error`：同空态布局，56px 圆形底 `var(--danger-wash)`、图标 `var(--danger)`；标题「接送记录加载失败」；说明「网络异常，没取到 10月26日 的接送记录，请重试」；按钮「重试」同样主按钮样式
- **禁用态**：今天时「后一天」箭头 `disabled`（视觉见 5.3），这是本页唯一的禁用控件；notes 里说明
- 所有状态块都要 `max-width: 100%`，文案换行，不得撑破 390

### 5.7 日期选择弹层

- `.sheet-layer` + `.sheet-backdrop`（`background: var(--fg); opacity: .38`）+ `.sheet`（`var(--surface)`、顶部 24px 圆角、`var(--shadow-lg)`、max-height 76vh）
- 头部：标题「选择日期」15px/600，副标题「最近 6 天」12px muted，右上 36×36 关闭按钮
- 行（每天一个 `<button class="date-option">`）：
  - 左：`10月26日` 14px/600 + `周一` 12px muted
  - 右：今天用 `今天` pill；其余用 `13 条` / `20 条` / `无记录`（11px muted）
  - 选中行：`var(--accent-wash)` 底、日期字 `var(--accent-deep)`、右侧 16px 对勾 `var(--accent)`；未选中透明底
  - 行高 ≥ 48px，行间 1px 细线（opacity .35）
- 点行 → 关闭弹层 → 切到该日期并走 380ms 加载态
- 日期列表（倒序，含一个空态日）：
  - 10月26日 周一 · 今天 · 13 条
  - 10月25日 周日 · 无记录
  - 10月23日 周五 · 20 条
  - 10月22日 周四 · 20 条
  - 10月21日 周三 · 20 条
  - 10月20日 周二 · 20 条

### 5.8 记录详情弹层

- 同一套 `.sheet`，max-height 78vh
- 头部：标题「接送记录详情」，副标题「10月26日 周一 · 17:12」，关闭按钮
- 头部下方一块 hero：
  - 44px 圆形头像（accent-wash / accent-deep）
  - 姓名 18px/600 + 下一行「小一班 · 接走」12px muted
  - 右侧时间 18px/700 tabular-nums
  - 若异常/例外，姓名行尾放对应小标签（复用 5.5 的样式）
- 字段列表 `<dl>`：两列 `grid-template-columns: 76px minmax(0, 1fr)`，行 padding 10px 0，行间 1px 细线 opacity .35
  - 标签 12px `var(--muted)`，值 13px `var(--fg)`，可换行
  - 字段（按记录动态显示/隐藏）：
    - 接送类型：接走 / 送入
    - 记录时间：10月26日 周一 17:12
    - 接送人：张伟（舅舅）——仅接走
    - 核验方式：手动 / 扫码
    - 授权状态：例外放行 · 非授权接送人 / 有效授权接送人——仅接走
    - 异常备注：提前接走（17:05，早于 17:30 放学）——仅异常
    - 理由：临时委托舅舅接走，已电话核实母亲——有 remark 时
- 底部：整宽「关闭」主按钮（40 高、胶囊、accent-gradient）
- 送入记录的详情里没有接送人 / 授权状态两行，不要显示空值「—」占位

### 5.9 Toast 与返回

- `.toast` 参考外壳页：`position: fixed; left: 50%; bottom: calc(96px + env(safe-area-inset-bottom)); background: var(--fg); color: var(--on-brand); border-radius: var(--radius-pill); padding: 10px 16px; z-index: 500;`，`.is-on` 显示
- 返回按钮点击：toast「已返回接送管理」（原型不做真实路由）
- 弹层打开时点遮罩或关闭按钮关闭；Esc 关闭

### 5.10 图标

全部内联 SVG，`fill="none"` + `stroke="currentColor"` + `stroke-width: 1.6~1.8`，尺寸 11–20。需要：返回箭头、日历、上下 chevron、扫码框、手形（手动）、时钟、对勾、关闭、空态剪贴板、失败警示三角。

---

## 6. 演示数据（精确，不要临场改）

### 6.1 固定常量

- 今天：`2026-10-26`，周一；班级：小一班；放学时间 17:30
- 学生（姓名 · 默认授权接送人）：
  - 乐乐 · 张敏（妈妈）
  - 小雨 · 赵静（妈妈）
  - 朵朵 · 陈静（妈妈）
  - 果果 · 孙磊（爸爸）
  - 豆豆 · 周敏（妈妈）
  - 米米 · 吴涛（爸爸）
  - 团团 · 李梅（妈妈）
  - 安安 · 王强（爸爸）
  - 妞妞 · 刘芳（奶奶）
  - 天天 · 何静（妈妈）

### 6.2 今天的 13 条（严格按此表，按时间倒序）

| # | 时间 | 学生 | 类型 | 核验 | 接送人 | 例外 | 异常 | 理由 | 异常备注 |
|---|---|---|---|---|---|---|---|---|---|
| 1 | 17:12 | 朵朵 | 接走 | 手动 | 张伟（舅舅） | 是 | 否 | 临时委托舅舅接走，已电话核实母亲 | — |
| 2 | 17:05 | 团团 | 接走 | 扫码 | 李梅（妈妈） | 否 | 是 | 家长称孩子发烧，需提前接走 | 提前接走（17:05，早于 17:30 放学） |
| 3 | 16:52 | 安安 | 接走 | 手动 | 王强（爸爸） | 否 | 否 | — | — |
| 4 | 16:38 | 妞妞 | 接走 | 扫码 | 刘芳（奶奶） | 否 | 否 | — | — |
| 5 | 08:41 | 米米 | 送入 | 扫码 | — | — | — | — | — |
| 6 | 08:30 | 豆豆 | 送入 | 扫码 | — | — | — | — | — |
| 7 | 08:20 | 果果 | 送入 | 手动 | — | — | — | — | — |
| 8 | 08:18 | 妞妞 | 送入 | 扫码 | — | — | — | — | — |
| 9 | 08:12 | 乐乐 | 送入 | 扫码 | — | — | — | — | — |
| 10 | 08:05 | 小雨 | 送入 | 手动 | — | — | — | — | — |
| 11 | 08:02 | 团团 | 送入 | 扫码 | — | — | — | — | — |
| 12 | 07:58 | 朵朵 | 送入 | 手动 | — | — | — | — | — |
| 13 | 07:55 | 安安 | 送入 | 扫码 | — | — | — | — | — |

- 摘要：13 条记录 / 4 条接走 / 9 条入园签到
- 分组头：`10月26日 周一 · 共 13 条`
- 标题下只有 1 条异常、1 条例外

### 6.3 历史上学日（10/23、10/22、10/21、10/20）

每个上学日生成 20 条：10 条送入 + 10 条接走，全部有效授权、无异常、无理由。

- 送入时间（固定数组，升序）：`07:52, 07:58, 08:03, 08:08, 08:12, 08:18, 08:22, 08:27, 08:33, 08:41`
- 接走时间（固定数组，升序）：`16:34, 16:41, 16:47, 16:52, 16:58, 17:03, 17:09, 17:14, 17:20, 17:26`
- 按学生顺序依次对应（乐乐→小雨→朵朵→果果→豆豆→米米→团团→安安→妞妞→天天）
- 核验方式按索引交替：偶数扫码、奇数手动
- 接送人取该学生的默认授权人
- 摘要：20 / 10 / 10；分组头 `10月23日 周五 · 共 20 条`
- 日期弹层里显示的条数必须与生成结果一致

### 6.4 周末（10/25 周日、10/24 周六）

0 条 → 空态。10/25 的说明文案是「10月25日 周日 · 全园休息」。

---

## 7. 状态与交互

### 7.1 状态机

页面状态由 `selectedDate` + `phase` 推出，并写到 `.mp-app[data-state]`：

- `phase = 'loading'` → `loading`
- `phase = 'error'` → `error`
- 否则 `records.length === 0` → `empty`
- 否则 → `normal`

- 切日期（箭头或弹层）：`phase='loading'` → 380ms 后 `phase='idle'`，渲染该日数据
- demo chip 点击（监听 `mp:state`）：
  - `normal` → 今天 + `idle`
  - `empty` → 10/25 周日 + `idle`（天然空态，文案与日期一致）
  - `loading` → 今天 + 持续 `loading`（不自动结束，便于展示）
  - `error` → 今天 + `error`
- 任何日期切换都清除 demo 的强制状态，回到自然状态
- 每次状态变化同步 chip 的 `aria-pressed`（仅当前状态对应的 chip 为 true；自然空态时「空态」chip 亮）
- 重试：`phase='loading'` → 380ms → `idle`
- 「回到今天」：`selectDate(TODAY)`
- 回到今天后「后一天」箭头 `disabled = true`

### 7.2 交互清单

| 元素 | 行为 |
|---|---|
| 返回箭头 | toast「已返回接送管理」 |
| 左箭头 / 右箭头 | 前一天 / 后一天；超过今天禁用 |
| 日期按钮（整块） | 打开日期选择弹层 |
| 日期弹层某一行 | 关闭弹层，切日期，380ms 骨架 |
| 时间轴某一行 | 打开该条记录详情弹层 |
| 空态「回到今天」 | 切回今天 |
| 失败「重试」 | 380ms 后恢复自然状态 |
| 弹层遮罩 / 关闭 / Esc | 关闭弹层 |
| demo chips | 见 7.1 |

- 打开任一弹层时给 `document.body` 加 `overflow: hidden`？页面本身用 document 滚动，**不要锁 body**，只让 `.sheet-body` 可滚动。
- 弹层用 `hidden` 属性控制；打开时移除 `hidden` 并聚焦关闭按钮。
- 详情弹层数据从当前行的记录对象取，不重新造数据。

### 7.3 可访问性

- 日期按钮 `aria-haspopup="dialog"` + `aria-controls="dateSheet"`
- 弹层 `role="dialog" aria-modal="true" aria-labelledby=...`
- 时间轴行是 `<button>`，`aria-label` 含姓名 + 时间
- 状态块标题用 `<h2>`；列表容器 `aria-live="polite"`
- 装饰性 SVG 加 `aria-hidden="true"`

---

## 8. 硬约束（违反即返工）

1. **禁止 `white-space: nowrap`**（整个 `styles.css` 一处都不许有；外壳自带的不算）
2. 摘要必须是 `grid-template-columns: repeat(3, minmax(0, 1fr))`
3. 所有卡片、行、弹层、文本容器 `max-width: 100%`；390px 宽不得横向滚动
4. 颜色只能 `var(--token)`；HTML / CSS / JS 中不得出现 hex、rgb、hsl（`tokens.css` 除外，且不得改）
5. 不得外链任何资源（字体、图片、脚本）
6. `styles.css` 不得定义 `.mp-*` 类（`[data-state]` 作用域除外）
7. 不得复制 / 修改 `_shared/`，不得创建本地 `_shared`
8. 不得改 `tokens.css`
9. 不得出现 `border-radius: 0`
10. 只写 `index.html`、`styles.css`、`app.js` 三个文件；写完即停，不做自检循环
11. 不画 tabBar；不画底部操作栏；不加「只看异常」；不加搜索/分页/导出
12. 不使用外部图标库或 emoji 充当图标；图标一律内联 SVG

---

## 9. 输出

- `index.html`：完整可打开，外壳逐字对齐第 4 节
- `styles.css`：只写本页样式，按「日期卡 / 分组头 / 时间轴 / 三态 / 弹层 / toast」分段注释
- `app.js`：数据常量 + 状态机 + 渲染 + 弹层 + 事件；无 console 报错

完成后只返回：文件清单、主要设计决策、与源码的差异、遗留问题。
