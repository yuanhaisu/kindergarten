你是幼儿园管理平台「养成星球」的资深移动端设计师。为家长端做一个页面的高保真 HTML 原型。

## 唯一任务
页面：成长相册（源码：parent/src/pages/growth/album.vue）
只做一个方向，不要 A/B 变体，不要多页面。写完全部文件即停止：不要提问，不要联网，不要读取自己刚写的产物做自查，不要覆盖 tokens.css。

## 这个页面真实的产品事实

**定位**：家长从「成长时光」页的「导出相册」进入的二级页，按月查看班级成长照片墙。本页是 `uni.navigateTo` 打开的子页。
**导航**：微信原生导航栏，标题固定「成长相册」（来自 pages.json），**必须画返回箭头**（`mp-back`）；**不画底部 tabBar**（本页不是 tab 页）。

数据来源：`GET /parent/growth/album?year=&month=`，前端按 `data.groups[]`（每组 `{ date, media[] }`）与 `data.total` 渲染；每张 `media` 有 `id / file_url / thumb_url`，点击调用 `uni.previewImage({ urls, current })`——**预览范围是当天这一组照片**，不是整月。

区块顺序（从上到下固定）：
1. **月份切换条**：左箭头 ‹、中间「2026年10月」、右箭头 ›；右侧显示当月总张数「18张照片」。源码用两个圆形箭头按钮 + 月份 + 张数。
   - 本页顶部还必须给出**时间范围**（如「10月9日 - 10月24日」）：由当月照片的最早/最晚分组日期派生，**不新增接口**，写成月份下方的次要信息行。
2. **日期分组**：每组一个小标题（源码直接渲染 `group.date`，只显示日期如「10月24日」，**不要编星期几**），下方 3 列等宽正方形照片网格，组间距与组内间距都要清楚。
3. **3 列照片网格**：`grid-template-columns: repeat(3, minmax(0, 1fr))`，每格 `aspect-ratio: 1 / 1`、**必须不变形**，圆角用 `var(--radius-lg)`，间距 4px（对应源码 `$spacing-xs`）。
4. **照片点击**：原型不跳系统预览器，改为**从底部滑出的预览弹层**（详见交互）。弹层内可在当天照片之间左右切换。

真实约束（不要发明）：
- 页面只有「按月份翻页 + 按日期分组的照片墙 + 点图预览」三件事；**没有**上传、删除、多选、下载、点赞、评论、筛选、搜索。
- 月份箭头前后都能翻，源码不限制未来月份；翻到没有数据的月份就显示该月空态。
- 加载中源码只显示「加载中...」，原型升级为骨架屏；源码没有错误分支，原型补错误态属状态覆盖，不算新增业务。
- 照片 `file_type` 后端支持 video，但源码相册页把 media 一律当图片渲染，**本页不要加视频角标/播放键**。

## 示例数据（必须用这一组，不要另编）

**2026年10月（默认月）：共 18 张，时间范围「10月9日 - 10月24日」**
| 日期 | 张数 | 备注（仅内部语义，页面上不显示文案） |
|---|---|---|
| 10月24日 | 3 张 | 美术课点画 |
| 10月22日 | 6 张 | 户外活动 |
| 10月18日 | 4 张 | 手工彩泥 |
| 10月15日 | 3 张 | 自然角浇水 |
| 10月9日 | 2 张 | 秋游捡落叶 |

**2026年9月（可切换到的有数据月）：共 5 张，时间范围「9月12日 - 9月26日」**
| 日期 | 张数 |
|---|---|
| 9月26日 | 4 张 |
| 9月12日 | 1 张 |

其余月份（如 2026年11月、2026年8月）没有数据 → 显示「本月暂无照片」的月份空态。

## 视觉方向

- 底色是中性冷雾白，照片墙是主角：暖色只出现在月份切换条的选中/悬停反馈、主按钮、预览弹层的少量强调，**暖色面积 ≤ 10%**，不要大面积渐变背景。
- 日期标题克制：小号深色日期 + 右侧极淡的「N 张」；组与组之间用留白分隔，不要横线卡片堆叠。
- 三列网格是页面骨架，圆角统一、间距均匀，像真实相册一样密铺。
- 照片用 **CSS 渐变 + 极淡几何图形占位**（只用 `var(--*)` 组合：天空蓝 `--info-wash`、叶子绿 `--success-wash`、沙黄 `--warning-wash`、薄荷、以及少量 `--accent-wash`），每格左上/右上角一个柔光圆、底部一道弧形山丘剪影，5 种色调轮换；**不要外链图片、不要 emoji、不要 base64**。
- 预览弹层：浮起的圆角卡片（四角都圆），上方是大图占位（正方形），左右各一个 44px 圆形切换按钮（当天只有 1 张时隐藏），底部一行「共 N 张 · 左右切换查看」与「关闭」按钮；背后是压暗遮罩（用 `var(--fg)` + `opacity`，不要写 rgba）。

## 状态覆盖（数据区必须四态，用 `.mp-app[data-state]` 纯 CSS 切换）

- **正常**：默认，渲染上面 18 张照片与 5 个日期分组。
- **空态（全局演示空态）**：隐藏月份网格，显示插画位 + 「还没有照片」+ 说明「老师上传成长记录后，照片会按日期整理在这里」+ 主按钮「刷新看看」。
- **加载**：保留月份标题与两个箭头（但两个箭头禁用降级），张数/时间范围换成灰色骨架条；显示 2 组骨架：日期条 + 3×3 灰色方块，带呼吸动画。
- **错误**：隐藏网格；显示错误卡（danger 圆形感叹号 + 「加载相册失败」+ 说明「网络不太稳定，请稍后再试」+ 「重试」）；月份箭头禁用降级。
- **禁用态**：加载/错误态下两个月份箭头统一变为 `--border` 底 + `--hint` 字，`aria-disabled="true"`，不响应点击；正常/空态恢复。
- **月份空态（注意与全局空态区分）**：翻到没有数据的月份时，保留月份切换条，网格位置显示更轻的块：「本月暂无照片」+ 一句「换个月份看看，或等老师上传新的照片」+ 文字按钮「看上个月」。

## 月份切换必须真的可用（app.js）

- 在 HTML 里静态放两个月份面板：`data-month-panel="2026-10"`（18 张、5 组）与 `data-month-panel="2026-09"`（5 张、2 组），默认 10 月可见、9 月 `hidden`。
- 点 ‹ / › 时在 2026年8月 ~ 2026年12月之间移动：命中上面两个月就显示对应面板并更新标题/张数/时间范围；没命中的月份隐藏所有面板，显示「月份空态」块，张数行显示「0 张照片」。
- 月份标题格式「2026年10月」；张数+范围行格式「18 张照片 · 10月9日 - 10月24日」；无数据月只显示「0 张照片」。
- 「看上个月」按钮执行一次 ‹ 的动作。

## 交互（app.js，全部真的能点）

- **点照片 → 打开底部预览弹层**：弹层标题「10月24日 · 第 1 / 3 张」，大图占位切换为被点照片的色调；‹ › 在**同一日期分组内**循环切换并更新标题；点遮罩、点「关闭」、按 Esc 都能关闭；关闭后焦点回到刚才那张照片。
- 分组只有 1 张照片时，弹层内左右按钮 `hidden`。
- 返回箭头（`.mp-back`）点击 → toast「已返回成长时光」（不伪造真实跳转）。
- 「重试」：点击后禁用并显示「重试中…」，约 900ms 后恢复并回到正常态，toast「已重新加载」。
- 空态「刷新看看」→ 回到正常态，toast「已刷新」。
- 月份切换后 toast 不需要；但点「看上个月」时 toast「已切换到 2026年10月」。
- toast 放右上角（深色 `var(--fg)` 底、`var(--on-brand)` 字、圆角 `var(--radius-lg)`），2 秒消失。
- 四态芯片由共享 `shell.js` 写 `data-state` 并派发 `mp:state`；app.js 监听它同步月份箭头禁用态，**不要自己再绑芯片**；切到非 normal 态时若弹层开着要自动关闭。

## 视觉与结构硬约束

- `<html lang="zh-CN" data-surface="parent">`，390px 手机视口，单列纵向流，不做桌面布局。
- 引入顺序固定：`tokens.css` → `../../_shared/shell.css` → `styles.css`。
- 颜色**只能**写 `var(--*)`，禁止裸 hex / rgb / rgba（HTML 内联 SVG 也只许 `currentColor`）；阴影只用 `var(--shadow-*)`；不得出现直角（最扁平元素 ≥ `var(--radius-sm)`，卡片 `var(--radius-card)`，按钮/标签/头像 `var(--radius-pill)`）。
- 层级只靠字重 + 字号：标题 600 / 正文 400 / 标签 500。
- 所有可能超宽的文本禁止 `white-space: nowrap`；多列统计用 `grid-template-columns: repeat(N, minmax(0, 1fr))`；所有容器 `max-width: 100%`，页面不得横向滚动。
- 触控目标 ≥ 44px；正文 ≥ 13px；`:active` 反馈用 scale/压暗；不用 alert/prompt；无 JS 报错。
- 不引用任何外部资源；不写 base64 大图；单文件 < 20KB（index.html 可略超，属批次已知体积提示）。

## 外壳（唯一实现，必须原样使用 `mp-*` 类名）
不要创建 `_shared/` 目录，也不要复制 shell.css——只写 `<link rel="stylesheet" href="../../_shared/shell.css">`，该文件由流程提供。**本页不画 tabBar。**

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
    <h1 class="mp-title">成长相册</h1>
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
1. `index.html` —— 结构（含两个月份面板、四态区块、预览弹层）
2. `styles.css` —— 样式（只写内容区；外壳来自 ../../_shared/shell.css）
3. `app.js` —— 交互（月份切换、弹层预览、toast、四态联动）
`tokens.css` 已由流程提供，不要重写、不要覆盖。每个文件 < 20KB。不要引用任何外部资源。
