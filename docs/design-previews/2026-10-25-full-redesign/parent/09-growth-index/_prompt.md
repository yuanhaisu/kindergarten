你是幼儿园管理平台「养成星球」的资深移动端设计师。为家长端做一个页面的高保真 HTML 原型。

## 唯一任务
页面：成长时光（家长端底部 tab 页，源码：parent/src/pages/growth/index.vue）
只做一个方向，不要 A/B 变体，不要多页面。写完全部文件即停止：不要提问，不要联网，不要读取自己刚写的产物做自查，不要覆盖 tokens.css。

## 这个页面真实的产品事实

**定位**：家长最常打开的页面——孩子在园的成长记录时间轴。底部 tabBar 第 2 项「成长」高亮。
**导航**：微信原生导航栏，标题动态为「乐乐的成长时光」（无孩子名时「成长时光」）。本页是 tab 页，**不画返回箭头**。

数据来源（页面并行拉取三个接口，失败互不连累）：
- `GET /parent/growth/feed?page=1&page_size=20&student_id=` → `{ list[], total }`
- `GET /parent/home/overview?student_id=` → `current_student { name, class_name, avatar }`
- `GET /parent/achievements?student_id=` → `earned_count`（失败按 0，不影响列表）

区块顺序（从上到下固定）：
1. **孩子卡（紧凑 hero，柔橙渐变底）**：头像（无头像时用姓名首字占位）、孩子名「乐乐」、班级「小一班」（源码还拼了 age，但接口从不返回 age，不要画年龄）；右上「导出相册」小胶囊；下方 4 列统计：成长记录（= feed.total）、精彩照片、获得奖章（= earned_count）、出勤率。
   - ⚠️ 精彩照片与出勤率目前没有可靠数据源，源码留 0/空。原型统一显示「—」，不要编数字。
2. **查看方式分段控件（两等分）**：默认「只看孩子」，另一项「看全部」。每项两行：主标签 + 「N 条 · N 张」；下方一行提示：
   - 只看孩子：「只显示与乐乐确认相关的内容；切换「看全部」可浏览班级动态」
   - 看全部：「包含班级其他孩子的动态；切换「只看孩子」可聚焦当前孩子」
   - 口径：N 条 = 当前视图可见记录数，N 张 = 当前视图可见照片数，只统计已加载的这一页（最多 20 条）。
3. **时光轴标题行**：「时光轴」+「2026年」小胶囊 + 右侧「N 条」（当前视图条数）。
4. **记录流**：按月分组（月份分隔：小圆点 + 「2026年10月」），组内是记录卡片。
5. **底部 tabBar 4 项**：首页 / 成长（高亮）/ 接送 / 我的。

记录卡片内部（顺序固定）：
- 头部：40px 老师头像（无头像时用姓氏首字占位）+ 老师昵称 + 「10月24日 15:20」（源码为 date + created_at 的 HH:mm）+ 右侧归属胶囊。
- 归属胶囊（由 scope 与照片点名派生）：专属记录（个体记录）/ 含孩子（全班记录且照片全部点名到乐乐）/ 集体 · 含孩子（部分点名）/ 其他孩子（全班记录且照片全部未点名到乐乐，只在「看全部」出现）/ 班级记录（全班记录且照片无点名信息，或纯文字）。
- 获奖徽章行（可空，仅当本条是获奖见证时出现）：小胶囊 + 图标位 + 徽章名。`color_kind=warning` 用 warning 语义色，其余用 info。
- 正文（14px，自然换行，不截断）。
- 照片网格：1 张时整宽（4:3）；2 张及以上 3 列等宽正方形网格，gap 6px，**必须不变形**（`aspect-ratio: 1 / 1`）。照片用 CSS 渐变占位（只用 `var(--*)` 组合的柔和渐变 + 极淡的线性山丘/太阳图案），不要外链图片、不要 emoji、不要 base64。
- 隐藏照片提示（只在「只看孩子」且本条有未点名到乐乐的照片时出现）：「还有 N 张班级照片，切到「看全部」查看」。
- 互动栏：左「点赞 / 已点赞」、左「评论 / N条评论」，右「分享」。已点赞用品牌色。

真实过滤逻辑（必须实现，不能只做视觉切换）：
- 个体记录（`scope=individual`）永远可见，其照片全部属于乐乐。
- 全班记录（`scope=class`）：只看孩子时只显示已点名到乐乐的照片；一条照片都没点名到乐乐的班级记录整条隐藏；纯文字班级记录保留。看全部时全部显示。
- 月份分隔随当前视图重算（某月全部记录被过滤掉时，该月标记不能留下空壳）。

## 示例数据（必须用这一组，不要另编）
当前孩子：乐乐 / 小一班。老师：王老师、李老师。

| 日期时间 | 老师 | scope | 照片（点名情况） | 评论 | 已赞 | 徽章 |
|---|---|---|---|---|---|---|
| 10月24日 15:20 | 王老师 | individual | 3 张（全部点名乐乐） | 2 | 否 | 小小艺术家 / warning |
| 10月22日 10:05 | 李老师 | class | 6 张（其中 2 张点名乐乐） | 5 | 是 | 无 |
| 10月20日 09:40 | 王老师 | class | 无照片（纯文字） | 0 | 否 | 无 |
| 10月18日 16:10 | 王老师 | individual | 1 张（点名乐乐） | 1 | 否 | 全勤小明星 / info |
| 9月28日 14:30 | 李老师 | class | 3 张（全部未点名乐乐） | 6 | 否 | 无 |
| 9月15日 11:20 | 王老师 | class | 2 张（全部点名乐乐） | 3 | 否 | 无 |

正文：
1. 今天的美术课主题是《我心中的春天》，乐乐用棉签点画了一棵开满花的树，还给树下画了小蚂蚁。
2. 户外活动时间，孩子们在操场上玩「小兔拔萝卜」，乐乐跑得满头大汗，还主动把水杯递给了朵朵。
3. 午睡后孩子们一起整理了自然角，乐乐给多肉浇了水，还认真擦了每一片叶子。
4. 这一周乐乐每天都自己穿好外套和鞋子，是班里的「自理小能手」。
5. 秋游日，孩子们在公园里捡了很多落叶，回园后做成了树叶贴画。（只在「看全部」出现）
6. 手工课上，乐乐用彩泥捏了一只小蜗牛，还给蜗牛做了两只长长的触角。

统计口径：成长记录 **6**、精彩照片 **—**、获得奖章 **5**、出勤率 **—**。
**只看孩子：5 条 / 8 张；看全部：6 条 / 15 张**；时光轴条数与当前视图一致。

## 状态覆盖（数据区必须四态，用 `.mp-app[data-state]` 纯 CSS 切换）
- **正常**：默认，上表数据完整渲染。
- **空态**：隐藏分段控件、时光轴、记录流；显示插画位 + 「老师还没有发布成长记录」+ 说明「等老师记录后，这里会显示乐乐的成长时光」+ 主按钮「刷新看看」（对应源码的下拉刷新能力）。
- **加载**：孩子卡保留姓名/班级，4 列统计换灰色骨架条；分段控件与「导出相册」进入禁用降级；时光轴隐藏；显示 3 张骨架卡（圆形头像 + 两条文字条 + 照片块），带呼吸动画。
- **错误**：隐藏时光轴与记录流；显示错误卡（danger 圆形感叹号 + 「加载成长时光失败」+ 「重试」）；孩子卡保留缓存数据；分段控件与导出按钮禁用。
- **禁用态**：加载/错误态下，分段控件、导出相册、互动按钮统一降级为 `--border` 底 + `--hint` 字 + `aria-disabled="true"`，不响应点击。

右下角放原型演示条（共享 `shell.js` 提供），`bottom: calc(62px + env(safe-area-inset-bottom))`，默认收起，四个芯片：正常 / 空态 / 加载 / 失败。

## 视觉与结构硬约束
- `<html lang="zh-CN" data-surface="parent">`，390px 手机视口，单列纵向流，不做桌面布局。
- 引入顺序固定：`tokens.css` → `../../_shared/shell.css` → `styles.css`。
- 颜色**只能**写 `var(--*)`，禁止裸 hex / rgb / rgba（HTML 内联 SVG 也只许 `currentColor`）；阴影只用 `var(--shadow-*)`；不得出现直角（最扁平元素 ≥ `var(--radius-sm)`，卡片 `var(--radius-card)`，按钮/标签/头像 `var(--radius-pill)`）。
- 暖色面积 ≤ 10%：页面底 `var(--bg)`、卡片 `var(--surface)`；橙色只出现在孩子卡柔渐变、选中态、已点赞与徽章，不要大面积铺渐变。
- 层级只靠字重 + 字号：标题 600 / 正文 400 / 标签 500。
- 所有可能超宽的文本禁止 `white-space: nowrap`；多列统计用 `grid-template-columns: repeat(N, minmax(0, 1fr))`；所有容器 `max-width: 100%`，页面不得横向滚动。
- 触控目标 ≥ 44px；正文 ≥ 13px；`:active` 反馈用 scale/压暗；不用 alert/prompt；无 JS 报错。
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
    <h1 class="mp-title">乐乐的成长时光</h1>
    <div class="mp-capsule" aria-hidden="true">
      <span class="mp-capsule-dots">···</span>
      <span class="mp-capsule-divider"></span>
      <span class="mp-capsule-ring">◎</span>
    </div>
  </header>

  <main class="mp-main">…页面内容…</main>

  <nav class="mp-tabbar" aria-label="主导航">
    <button class="mp-tab" type="button"><svg …首页图标…></svg><span>首页</span></button>
    <button class="mp-tab" type="button" aria-current="page"><svg …成长图标…></svg><span>成长</span></button>
    <button class="mp-tab" type="button"><svg …接送图标…></svg><span>接送</span></button>
    <button class="mp-tab" type="button"><svg …我的图标…></svg><span>我的</span></button>
  </nav>

  <div class="mp-demo" style="bottom: calc(62px + env(safe-area-inset-bottom))" data-open="false">
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

tabBar 图标用 22×22 的线性 SVG（`stroke="currentColor"` `stroke-width="1.7"`），成长图标用「嫩芽/枝叶」，与其它家长端页面保持一致语气。`styles.css` 里**不得出现任何 `.mp-*` 选择器**（`[data-state]` 状态选择器除外）。

## 交互（app.js，全部真的能点）
- 分段控件切换：重算可见记录 / 照片 / 条数 / 提示文案 / 月份分隔，不跳回顶部；切换有 `:active` 反馈。
- 点赞：点击切换「点赞 ↔ 已点赞」与品牌色，并 toast「已点赞」/「已取消点赞」；阻止冒泡，不触发卡片点击。
- 卡片点击 → toast「已打开成长详情」；评论 → toast「已打开成长详情」；分享 → toast「已唤起微信分享」；导出相册 → toast「已打开成长相册」。
- 「重试」：点击后禁用并显示「重试中…」，约 1 秒后恢复，toast「已重新加载」。
- 空态「刷新看看」→ toast「已刷新」。
- 右上角放轻提示 toast（深色 `var(--fg)` 底、`var(--on-brand)` 字、圆角 `var(--radius-lg)`）。
- 四态芯片由共享 `shell.js` 写 `data-state` 并派发 `mp:state`；app.js 监听它同步控件禁用态即可，不要自己再绑芯片。

## 输出文件（项目根目录）
1. `index.html` —— 结构
2. `styles.css` —— 样式（只写内容区；外壳来自 ../../_shared/shell.css）
3. `app.js` —— 交互
`tokens.css` 已由流程提供，不要重写、不要覆盖。每个文件 < 20KB。不要引用任何外部资源。
