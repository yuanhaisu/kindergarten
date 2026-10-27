你是幼儿园管理平台「养成星球」的资深移动端设计师。为**微信小程序老师端（保健医角色）**做一个页面的高保真 HTML 原型。

## 唯一任务
页面：入园晨检（源码：teacher/src/pages/doctor/checkin.vue）
只做一个方向，不要 A/B 变体，不要多页面。写完 index.html / styles.css / app.js 三个文件即停止，不要提问，不要联网，不要读自己的产物做自查循环。

## 这个页面真实的产品事实（不要虚构其它能力）

保健医在早晨对已入园的孩子逐个做晨检。孩子入园打卡（pickup_record）时后端自动生成一条待检记录，保健医在队列里点开某个孩子，逐项记录并给出结论；提交后记录落到「已完成」，同时给家长发通知。

**队列数据（GET /doctor/health/queue，字段全部真实）**：
`id` 晨检记录 id / `pickup_record_id` / `student_id` / `student_name` / `class_name` / `class_id` / `check_status` = 0 待检 | 1 已完成 / `overall_status` = 0 健康 | 1 需观察 | 2 劝返 / `pickup_time` 入园时间（`HH:MM`，如 `08:12`）/ `avatar`（可能为空字符串，空时用姓名首字占位）。
队列响应顶层是 `{ list, policy }`。

**园所体温策略（与队列同一次下发，字段真实）**：
`policy.temperature_required` 布尔（true = 全员测温、体温必填）/ `policy.fever_threshold` 数值（本原型 37.3）。
策略条在加载中与加载失败时必须隐藏（源码注释说明了原因：避免在「必填园所」短暂显示错误的「按需测温」）。
本原型：**按需测温（选填）**，阈值 37.3°C。

**检查项（源码真实项，一项都不许加减）**，每项取值 0/1/2：
- 喉咙 `throat_status`：0 正常 / 1 发红 / 2 红肿
- 手部 `hand_status`：0 正常 / 1 红疹 / 2 疱疹
- 口腔 `mouth_status`：0 正常 / 1 溃疡 / 2 疱疹
- 眼睛 `eye_status`：0 正常 / 1 发红 / 2 分泌物
- 皮肤 `skin_status`：0 正常 / 1 红疹 / 2 水疱
源码里**没有**「精神状态 / 情绪 / 咳嗽」等其它检查项，不要加。默认每项都是 0（正常），点异常项在该项 1/2 之间切换；再点同一异常项回到 0；点「正常」直接置 0。

**体温**：`temperature`（number|null），选填（本园按需测温）。输入值 ≥ `fever_threshold`（37.3）时行内提示发烧。源码里策略为必填时，未填体温点提交会被拦下并 toast「本园要求录入体温」——本原型策略是选填，但代码里保留这个校验分支。

**备注**：`remark`，文本，≤200 字，占位「异常情况说明（选填）」。

**异常拍照**：某项异常时该项行尾出现「拍照」按钮；拍完先上传再关联（`POST /doctor/health/check/:id/media`，body = `{ file_id, category, ref_desc }`）。`category` 就是上面的字段名（throat/hand/mouth/eye/skin），已关联的照片显示为缩略图。原型里点「拍照」= 在页面下方「异常照片」区新增一个带类目标签的占位缩略图 + toast「照片已保存」，不要画真实照片、不要生成图片。

**隔离**：`is_isolated` 布尔，勾选文案「已带到隔离室等待家长」。

**提交（PUT /doctor/health/check/:id，body 字段真实）**：`temperature` / `throat_status` / `hand_status` / `mouth_status` / `eye_status` / `skin_status` / `overall_status` / `is_isolated` / `remark`。三个提交按钮对应 `overall_status`：
- 「健康入园」= 0
- 「需观察」= 1
- 「劝返就医」= 2
响应 data 额外含 `notified_count`（已通知家长数）/ `notify_failed_count`（通知失败数）。源码逻辑：`notify_failed_count > 0` 时弹「已记录 / N 位家长通知失败，请线下告知」；否则 toast「健康入园·已通知」这种。提交成功后关闭检查卡并重新拉队列。

**班级筛选**：队列里出现过哪些班就出哪些 chip（本原型：小一班 / 中二班 / 大一班），选「全部」= 不带 class_id 参数。切换即按班重新统计与过滤。

**统计**：总入园 / 待检查 / 已完成（源码就是这三个），随班级筛选变化。

**页面底部入口**：「发布园级事件（体检等，全园可见）」，点击去 `/pages/doctor/calendar`（园级事件页）。这是页面上唯一的次要入口，画成安静的整行 cell，不要做成主按钮。

**角色事实（影响失败态文案）**：Casbin 只给保健医开放 `/doctor/*`，班级老师调用会 403。失败态文案要能说清「没有权限或网络异常」，不能只写「加载失败」。

## 示例数据（中文，幼儿园语境；本原型固定这些数据）

今日 10月26日 周一。总计 18 人：待检 5，已完成 13。

待检 5 人（按入园时间）：
1. 乐乐 · 小一班 · 08:12
2. 小雨 · 小一班 · 08:20
3. 朵朵 · 中二班 · 08:26
4. 安安 · 中二班 · 08:31
5. 果果 · 大一班 · 08:38

已完成 13 人：
- 小一班 3 人：沐沐 08:02 健康 / 一一 08:06 健康 / 糖糖 08:09 需观察
- 中二班 4 人：小宝 08:04 健康 / 米粒 08:11 健康 / 天天 08:15 健康 / 豆豆 08:18 劝返
- 大一班 6 人：辰辰 07:58 健康 / 悦悦 08:01 健康 / 壮壮 08:05 健康 / 苗苗 08:08 健康 / 圆圆 08:14 需观察 / 阿布 08:22 健康

按班的统计（切 chip 时用）：
- 全部：总 18 / 待检 5 / 已完成 13
- 小一班：总 5 / 待检 2 / 已完成 3
- 中二班：总 6 / 待检 2 / 已完成 4
- 大一班：总 7 / 待检 1 / 已完成 6

姓名首字用作头像占位：乐 / 小 / 朵 / 安 / 果 / 沐 / 一 / 糖 / 小 / 米 / 天 / 豆 / 辰 / 悦 / 壮 / 苗 / 圆 / 阿。头像用 `var(--accent-wash)` 底 + `var(--accent-deep)` 字，不要用照片。

## 视觉方向（一句话）

**晨检台，不是表单墙**：一个孩子一张卡，收起时只有姓名/班级/入园时间与「检查」按钮；展开后五组状态 chip 在卡内完成录入，异常才染色（`--danger` 浅底），整页保持奶白底 + 一处暖橙焦点。

## 信息层级（从上到下）

1. **今日卡**（`--surface`，`--radius-card`，`--shadow-sm`）：第一行「10月26日 周一」+ 右侧体温策略胶囊（按需测温 · 体温选填，用 `--info` + `--info-wash`；若必填则 `--warning` + `--warning-wash`）；第二行三列统计（`grid-template-columns: repeat(3, minmax(0, 1fr))`）：总入园 18 / 待检 5 / 已完成 13，数字 20px 600 tabular-nums，「待检」用 `--accent-deep` 且只在有数字时强调，其余 `--fg`；标签 12px `--muted`。
2. **班级筛选**：一行可换行的 chip（**不要横向滚动容器**）。选中 = `--accent-wash` 底 + `--accent-deep` 字 + 1px `--accent` 边框；未选 = `--surface` 底 + 1px `--border` + `--muted` 字。高度 32px，点击有 :active 压暗。
3. **待检区**：区头「待检」+ 计数「5 人」（计数用小胶囊或纯文字，别用 emoji），右侧可放一句 12px 提示「点开逐项检查」。
4. **孩子卡 ×5**：收起态一行——头像 40px 圆形 + 姓名 15px/600 + 元信息 12px `--muted`（小一班 · 08:12 入园）+ 右侧「检查」小胶囊按钮（`--accent-wash` 底 / `--accent-deep` 字）。整行可点，行高 ≥ 56px。
5. **展开的检查卡**（同一个孩子卡内部展开，同一时刻只允许展开一张）：顶部一行孩子姓名 +「收起」文字按钮；下面依次是体温、5 个检查项、备注、异常照片、隔离开关、三个提交按钮。
6. **已完成区**：区头可点折叠/展开，标题「已完成 13」+ 右侧「收起 / 展开」；行高 48px，小头像 28px + 姓名 + 班级 + 右侧状态标签（健康 = `--success-wash`/`--success`；需观察 = `--warning-wash`/`--warning`；劝返 = `--danger-wash`/`--danger`）。已完成行整体降低存在感（字重 400/500，不要和待检抢）。
7. **园级事件入口**（页面最底部）：安静 cell——左侧 28px 圆角方块图标（`--info-wash` 底、`--info` 线图标或单字「园」），主文案「发布园级事件」，副文案「体检等 · 全园可见」，右侧 ›。

## 检查卡内部（展开态）的具体规格

- **体温**：label「体温」+ 右侧小字「选填」（若必填则「必填」并用 `--danger`）；输入框 96×40px，圆角 `--radius-lg`，边框 `--border`，聚焦边框 `--accent`，右侧单位「°C」。数值 ≥ 37.3 时下方出现一行 `--danger-wash` 底、`--danger` 字的小提示「37.8°C 高于阈值 37.3°C」。
- **五个检查项**：每项一行，行内左边 label（48px，13px `--muted`/500），右边 chip 组可换行。chip 高 32px，内边距 0 14px，`--radius-pill`：
  - 未选中：`--surface` 底、1px `--border`、`--muted` 字；
  - 选中「正常」：`--success-wash` 底、1px `--success` 边框、`--success` 字（**不要实心绿**）；
  - 选中异常（发红/红肿/红疹/疱疹/溃疡/分泌物/水疱）：`--danger-wash` 底、1px `--danger` 边框、`--danger` 字（**浅底，不要满屏红**）。
  - 某项处于异常时，该行行尾出现 28×28px 的「拍照」圆形 ghost 按钮（`--accent-wash` 底 + 相机线图标 `--accent-deep`），点击后加一张照片。
  - chip 的 `aria-pressed` 要跟随状态。
- **备注**：label「备注」+ textarea 2 行，高 ≥ 64px，占位「异常情况说明（选填）」，右下角显示 `0/200`。
- **异常照片**：仅当有照片时出现；标题「异常照片」+ 横向排列的 52×52 圆角方片，每个方片内一个相机线图标 + 下方 11px 类目名（喉咙/手部…）。不要画真实照片。
- **隔离开关**：整行 `role="switch"`，左文案「已带到隔离室等待家长」，右侧 44×26 的开关（关 = `--border` 轨道 + 白圆点；开 = `--danger` 轨道 —— 这是唯一允许用 danger 实心的地方，因为是状态而非按钮）。开启时整行底色变 `--danger-wash`、文案 `--danger`。
- **提交按钮**：三段式
  - 第一行：全宽主按钮「健康入园」，`--accent-gradient` 底、白字、44px、`--radius-lg`；`:active` 压暗/缩放。
  - 第二行：两个等宽次按钮「需观察」（`--warning-wash` 底 + `--warning` 字 + 1px `--warning` 边框）、「劝返就医」（`--danger-wash` 底 + `--danger` 字 + 1px `--danger` 边框）。**不要实心红**。
  - 提交中：三个按钮 `disabled`、整体 opacity .55，文字变「提交中…」；没有进度条。
- **提交失败（必须覆盖）**：在按钮上方插入一条 `--danger-wash` 底的提示条：「提交失败 · 网络抖动，请重试」，右侧「重试」文字按钮；卡片保持展开、已填内容不丢。**原型演示规则**：每次页面加载后的第一次提交必定失败（提示条里加一行 11px 的灰字「原型演示：首次提交模拟失败，重试即成功」），第二次提交成功。这是唯一允许的模拟失败点。

## 状态覆盖（数据区必须有）

用共享 `mp-demo` 演示条（正常 / 空态 / 加载 / 失败），由 `shell.js` 写 `data-state` 到 `.mp-app`：
- **正常**：上面的 5 张待检卡 + 13 行已完成。
- **空态**（今日已全部完成）：隐藏待检区与筛选、计数改为 总 18 / 待检 0 / 已完成 18；显示空态块（一个 64px 的圆形 `--success-wash` 图标位 + 「今天的晨检都完成了」+「18 名幼儿已完成，3 人需留意」+ 主按钮「回看已完成」，点击滚到已完成区并为已完成区补上 5 条新完成记录，状态均为「健康」）；已完成区仍显示 18 行。
- **加载**：隐藏待检/已完成/空态；今日卡的三个数字换成骨架条；策略胶囊隐藏；显示 3 张骨架卡（40px 圆 + 两条线），骨架用 `--accent-wash`/`--border` 的柔和块 + 1.4s shimmer 即可，不要刺眼。加载持续 900ms 后自动回正常（点芯片进入时开始计时）。
- **失败**：隐藏待检/已完成/空态；今日卡数字显示「—」（**不能显示 0**，0 会被误读成今天没人）；策略胶囊隐藏；显示失败块：`--danger-wash` 圆形图标位 + 「晨检数据加载失败」+ 12px 说明「网络异常，或当前账号没有保健医权限」+ 主按钮「重试」（点击 700ms 后回正常态）。
- **禁用态**：提交中按钮禁用；无数据的已完成区不渲染。

## 交互必须真的能点（app.js）

- 班级 chip 切换：过滤待检/已完成、更新三列统计与区头计数。
- 点孩子卡收起行 → 展开该卡（同时收起其它卡）；点「收起」→ 收起。展开时 `aria-expanded="true"`。
- 5 组 chip 能切换（正常 ↔ 两个异常），异常时出现拍照按钮；再点当前异常项回到正常。
- 体温输入实时校验：≥37.3 显示发烧提示；非数字不显示。
- 备注输入更新计数 `N/200`。
- 拍照按钮追加一个占位缩略图 + toast「照片已保存」。
- 隔离开关切换。
- 三个提交按钮：模拟 500ms 提交 → （首次失败 → 提示条；重试) → 成功：卡片收起并从待检列表移除、插入已完成列表首部（状态对应 健康/需观察/劝返）、统计与区头计数更新、toast「健康入园 · 已通知家长」（观察到/劝返同理）；若开了隔离，toast 追加「· 已隔离」。
- 已完成区折叠/展开（`aria-expanded`，切换文案「收起/展开」）。
- 园级事件行点击 → toast「已打开园级事件」；返回箭头点击 → toast「返回工作台」。
- 空态「回看已完成」→ 滚到已完成区 + 补齐 5 条记录。
- 演示条四种状态可切；切到加载/失败要能靠按钮回到正常。
- toast：底部居中、`--fg` 半透明底 + 白字、`--radius-pill`、2 秒自动消失，`aria-live="polite"`。不用 alert / prompt。
- 控制台必须零报错。

## 视觉与结构硬约束（逐条照做）

- `<html lang="zh-CN" data-surface="teacher">`，390px 手机视口，单列纵向流，不做桌面布局；老师端**没有 tabBar，不要画**。
- 本页是由 `uni.navigateTo` 进入的子页：**导航栏必须有返回箭头**，标题「入园晨检」。
- 引入顺序固定：
  ```html
  <link rel="stylesheet" href="tokens.css">
  <link rel="stylesheet" href="../../_shared/shell.css">
  <link rel="stylesheet" href="styles.css">
  ```
  页面 `styles.css` 里**不得出现任何 `.mp-*` 选择器定义**（唯一例外是 `[data-state="..."]` 前缀的状态选择器）。不要复制、不要改写 shell.css，也不要复制 `.mcp.json`。
- 颜色只能写 `var(--*)`，**禁止任何裸 hex / rgb / hsl**（包括 `rgba()` 阴影；阴影只用 `var(--shadow-sm|md|lg)`）。无直角：最扁平元素 ≥ `var(--radius-sm)`，按钮/标签/胶囊用 `var(--radius-pill)`，卡片用 `var(--radius-card)`。
- 暖色面积 ≤ 10%：大面积永远是 `var(--bg)` / `var(--surface)`；品牌渐变只允许出现在「健康入园」主按钮和空态/失败态主按钮上。整页不要出现大块橙色。
- 语义色只用 token：健康/正常 = `--success` + `--success-wash`；需观察/必填 = `--warning` + `--warning-wash`；异常/劝返/隔离 = `--danger` + `--danger-wash`；策略/信息 = `--info` + `--info-wash`。**不要自造颜色。**
- 触控目标 ≥ 44×44px（chip 视觉高 32px，但要用 padding 或包裹让可点区 ≥ 44px）；正文 ≥ 13px（`var(--text-base)`）；姓名 15px/600；区头 14px/600。
- **禁止 `white-space: nowrap` 用于可能超宽的文本**（姓名、班级、备注、提示条都可能长，必须正常换行）。唯一例外是外壳自带的 `.mp-title`。
- 多列统计用 `grid-template-columns: repeat(N, minmax(0, 1fr))`；所有容器 `max-width: 100%`；页面不得出现横向滚动。班级 chip 用 `flex-wrap: wrap`，**不要**做 `overflow-x: auto` 的横向滚动条（会被门禁判为越界）。
- 底部内容留 `padding-bottom: calc(88px + env(safe-area-inset-bottom))`，给演示条留位置。
- 图片一律不引用（头像用首字占位、照片用占位方片、图标全部内联 SVG）。
- 每个文件 < 20KB；不引用任何外部资源（字体、图片、CDN 都不行）。
- 不要用 emoji 当图标或按钮文案（源码里的 ✅⚠️🚫 一律改成纯文字 + 内联 SVG）。不写营销话术；按钮文案 2–4 字动词。

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
      <button class="mp-back" id="backBtn" type="button" aria-label="返回工作台">
        <svg width="20" height="20" viewBox="0 0 20 20" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M12.5 4.5 7 10l5.5 5.5"/></svg>
      </button>
      <h1 class="mp-title">入园晨检</h1>
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
