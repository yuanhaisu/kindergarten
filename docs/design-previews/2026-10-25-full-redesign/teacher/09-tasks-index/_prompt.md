你是幼儿园管理平台「养成星球」的资深移动端设计师。为**微信小程序老师端**做一个页面的高保真 HTML 原型。

## 唯一任务
页面：亲子任务（源码：teacher/src/pages/tasks/index.vue）
只做一个方向，不要 A/B 变体，不要多页面。写完 index.html / styles.css / app.js 三个文件即停止，不要提问，不要联网，不要读自己的产物做自查循环。

## 这个页面真实的产品事实（不要虚构其它能力）

老师端亲子任务管理页。老师查看本班已发布的任务、每个任务的参与情况与待回应数量，点进任务去回应家长提交，并在这里新建任务、维护班级星星罐。

**作用域（重要）**：任务只能建给**当前登录老师自己的班级**（本原型：小一班）。源码里**没有「目标班级」选择器**，不要画班级多选。唯一的范围差异是 `target_student_id`：0 = 全班任务，>0 = 某一个孩子的个体任务。

**任务列表**（GET /teacher/tasks，字段全部真实）：
`id` / `title` 标题 / `description` 说明 / `learning_link` 学习关联 / `target_student_id` 0=班级、>0=个体 / `frequency` = once 一次性 | daily 每日 | weekly 每周 / `start_date`、`end_date` / `status` = 1 进行中、0 已结束 / `allow_public` 家长可申请公开 / `recognition_ritual` = sticker 贴纸 | voice 语音 | class_star 班级星星 / `star_scope` = task 按任务 | class 按班级 / `star_goal` / `star_count`（**可能为 null：星数查询失败时后端返回 null，此时隐藏星星信息，不能画成 0**） / `pending_responses` 待回应总数 / `pending_period` 本周期待回应数 / `period_start`、`period_end` / `created_at`。

**每个任务的参与统计**（GET /teacher/tasks/:id/stats，字段真实）：
`total_students`（**恒为班级在读人数 30，个体任务也是 30**）/ `submission_count` 提交数 / `responded_count` 已回应 / `pending_count` 待回应。
- 班级任务展示「已参与 21/30」+ 一条细进度条（submission_count / total_students）。
- 个体任务**不要**画 N/30，改显示「已提交 1 条」，因为 30 是全班人数，对个体任务没有意义。
- 列表接口暂未聚合这组数字；原型按产品要求展示，这是真实统计端点已有的字段，不要发明别的统计。

**待回应用语**：列表只有 `pending_responses`（全部待回应，含旧周期遗留）与 `pending_period`（本周期）。两者不等时显示「待回应 5（本周期 3）」，相等时只显示「待回应 5」。**没有「待审核」字段**，不要写待审核；公开申请审核只在任务详情里按条处理。

**星星展示规则**（照抄源码逻辑，不要自己发挥）：
- `star_count == null` → 不显示任何星星信息。
- `star_scope = class` → 只显示「班级星星罐 · 本任务已集 8 星」，不画任务级星星进度。
- `star_scope = task` 且 `star_goal > 0` → 显示「★ 12/20」；`star_count > star_goal` 时文案「星星已超额 · 12/10」；相等时「星星已集满 · 12/12」。
- `star_goal = 0` → 不显示星星信息。

**新建任务**（POST /teacher/tasks，源码表单字段原样保留）：
- `title` 任务标题（必填）
- `description` 任务说明
- `learning_link` 和幼儿园学习的关系（**必填**）
- `target_student_id` 范围：班级 / 某个孩子（个体任务**不能**公开）
- `frequency` 频率：一次性 / 每日 / 每周
- `start_date`、`end_date` 起止日期（默认今天 ~ 7 天后；结束不能早于开始）
- `recognition_ritual` 回应仪式：贴纸 / 语音 / 班级星星
- `star_scope` 星星罐范围：按任务 / 按班级（**按班级需要本学期班级星星罐已配置，否则置灰**；选按班级时任务星星目标清空为 0，提示用班级罐目标）
- `star_goal` 星星目标（仅按任务时显示；0=不启用；回应仪式选「班级星星」且按任务时必须 > 0）
- `allow_public` 允许家长申请公开（仅班级任务显示；开启后提示「家长可申请把内容放上班级墙，需你逐条审核」）
- 提交按钮「创建任务」

**班级星星罐**（GET/PUT /teacher/class/star-jar，字段真实）：
`term` 学期 / `configured` 是否已配置 / `goal` 目标颗数 / `stars` 已集星数 / `progress` 后端算好的进度（**可能为 null**）/ `over_goal` 是否超额。
页面需要展示「已集 46 / 60」、一条进度条、当前学期、目标输入框（目标须为 >0 的整数）与「更新目标 / 设置目标」按钮。未配置时显示「本学期还没设目标；设好之后才能创建『按班级』任务」。
班级罐只展示集体进度，**不允许个人排名**。

**入口关系**：本页由老师工作台 `navigateTo` 进入 → **必须画返回箭头**。老师端**没有 tabBar**（任何老师端页面都不画）。任务卡点击进入 `pages/tasks/detail`（回应页）。

## 样例数据（请原样使用，数字必须自洽）

当前：2026年10月26日 周一。班级：小一班（30 名在读），老师：王老师。学期：2026 秋季学期。

班级星星罐：configured=true，term=2026 秋季学期，stars=46，goal=60，progress=77，over_goal=false。

任务（4 个，顺序如下；默认「全部」显示这 4 个）：

| # | 标题 | 频率 | 仪式 | 范围 | 起止 | 状态 | 参与 | 待回应 | 星星 |
|---|---|---|---|---|---|---|---|---|---|
| 1 | 每日阅读 10 分钟 | 每日 | 贴纸 | 班级 | 10月20日–11月2日 | 进行中 | 21/30（70%） | 5（本周期 3） | ★ 12/20 |
| 2 | 自己的事情自己做 | 每周 | 班级星星 | 班级 | 10月19日–11月1日 | 进行中 | 26/30（87%） | 2 | 班级罐 · 本任务已集 8 星 |
| 3 | 乐乐 · 练习系鞋带 | 一次性 | 语音 | 个体（乐乐） | 10月24日–10月30日 | 进行中 | 已提交 1 条 | 1 | ★ 3/5 |
| 4 | 周末亲子散步 | 一次性 | 贴纸 | 班级 | 10月12日–10月18日 | 已结束 | 28/30（93%） | 0 | 无（star_goal=0） |

任务说明：
1. 每天陪孩子读 10 分钟绘本，拍一张共读照片。
2. 自己穿鞋、收拾玩具，一周记录一次即可。
3. 练习系鞋带，录一段 15 秒语音。
4. 和孩子散步 20 分钟，找一找秋天的落叶。

分段筛选计数：全部 4 / 进行中 3 / 已结束 1。班级任务总待回应 8 条。

## 页面结构（从上到下）

1. 微信外壳：状态栏 + 导航栏「亲子任务」+ 返回箭头 + 右侧胶囊。**不画 tabBar。**
2. **顶部三段筛选**：全部 / 进行中 / 已结束（并排等宽分段控件，选中项用品牌色浅底 + 品牌色文字；每段带计数）。这是把真实 `status` 字段做成客户端视图筛选，不是新后端能力。
3. **任务卡列表**（单列，卡片间距 12px）：
   - 顶部：标题（可两行）+ 状态 pill（进行中 = 品牌色浅底；已结束 = 中性灰底）
   - 元信息一行：频率 · 仪式 · 日期区间；个体任务追加「个体任务」标签。允许换行，不能截断。
   - 说明：最多两行，超出省略。
   - 参与：班级任务「已参与 21/30」+ 右侧「70%」+ 一条 6px 细进度条（轨道浅色、填充品牌色）；个体任务「已提交 1 条」，不画进度条。
   - 星星：按上面的真实规则显示为一行小字（品牌色深色文字），不要额外再画第二条进度条。
   - 底部行：左侧待回应 badge（warning 浅底 + warning 文字，明显但不刺眼；含「本周期 N」时一并显示）；右侧「查看回应 ›」。整张卡可点，点击 toast「正在打开「标题」回应页」。
4. **班级星星罐卡**（列表下方，次级设置区）：标题「班级星星罐」+ 学期；「已集 46 / 60」；进度条；目标输入框 +「更新目标」。输入非法（空 / 0 / 非整数）toast 提示；保存后更新数字与进度并 toast「已保存」。
5. **底部固定操作栏**：一个主按钮「＋ 新建任务」（品牌渐变），点击打开底部弹层。内容区底部留出 `calc(76px + env(safe-area-inset-bottom))` 的避让。

## 新建任务底部弹层（真实字段，必须能用）

从底部升起，最大高度约 88vh，内部可滚动，含拖动条、标题「新建任务」、副标题「小一班 · 家长会在小程序里看到」。字段顺序：
1. 任务标题（input，placeholder「如：每日阅读 10 分钟」）
2. 任务说明（textarea，placeholder「如：每天陪孩子读 10 分钟绘本」）
3. 和幼儿园学习的关系（input，**必填**，placeholder「如：延续本周的春天主题」）
4. 范围：胶囊选择「班级」+ 孩子名（乐乐 / 小雨 / 朵朵 / 果果 / 豆豆 / 米米）。选孩子后显示提示「个体任务只有该孩子家长可见，不能公开」，并隐藏公开开关。
5. 频率：三段控件 一次性 / 每日 / 每周（默认「一次性」）
6. 起止日期：两个日期 cell，显示「10月26日 周一」这类文案；点击调用浏览器原生日期选择（原型可用隐藏 date input 实现，但外观必须是 cell，不能出现桌面下拉框），结束日期不能早于开始。
7. 回应仪式：三段控件 贴纸 / 语音 / 班级星星（默认贴纸）
8. 星星罐范围：两段控件 按任务 / 按班级（默认按任务）。选「按班级」时隐藏星星目标输入，并提示「使用班级罐目标 60 颗，本任务不再单独设目标」。
9. 星星目标（仅按任务显示）：数字 input，placeholder「0 = 不启用；选班级星星仪式时必填」。
10. 允许家长申请公开（仅班级任务显示）：switch + 说明。
11. 底部：取消 / 创建任务（主按钮）。标题或学习关联为空时「创建任务」禁用（真正 disabled，样式明显降级）。点击创建：通过校验 → 关闭弹层 → toast「已创建」→ 新任务插到列表顶部（进行中、参与 0/30、待回应 0），筛选计数同步更新。

## 状态覆盖（数据区四态 + 演示条）

用共享 `mp-demo` 演示条（正常 / 空态 / 加载 / 失败），由 `shell.js` 写 `data-state` 到 `.mp-app`：
- **正常**：上面的 4 张任务卡 + 星星罐。
- **空态**：隐藏任务卡与筛选计数，显示「还没有亲子任务」+ 一句「创建后家长可在小程序里查看并回应」+ 主按钮「新建任务」（打开同一个弹层）。星星罐仍保留。
- **加载**：隐藏任务卡，显示 3 张骨架卡（标题、元信息、进度条位置）；筛选计数隐藏；星星罐显示骨架。
- **失败**：隐藏任务卡，显示一句话「任务加载失败」+「网络异常，未获取到任务列表」+「重试」按钮；筛选计数隐藏；重试按钮模拟 600ms 后回到正常态。
- **禁用态**：创建按钮初始禁用；按班级在星星罐未配置时禁用（本原型已配置，但代码要体现 disabled 类）；个体任务下公开开关隐藏。

## 视觉与结构硬约束（逐条照做）

- `<html lang="zh-CN" data-surface="teacher">`，390px 手机视口，单列纵向流，不做桌面布局。
- 引入顺序固定：
  ```html
  <link rel="stylesheet" href="tokens.css">
  <link rel="stylesheet" href="../../_shared/shell.css">
  <link rel="stylesheet" href="styles.css">
  ```
  页面 `styles.css` 里**不得出现任何 `.mp-*` 选择器定义**（唯一例外是 `[data-state="..."]` 前缀的状态选择器）。不要复制或改写 shell.css。
- 颜色只能写 `var(--*)`，**禁止任何裸 hex / rgb / hsl**；无直角（最扁平元素 ≥ `var(--radius-sm)`，按钮/标签/胶囊用 `var(--radius-pill)`，卡片用 `var(--radius-card)`）；阴影只用 `var(--shadow-sm|md|lg)`。
- 暖色面积 ≤ 10%：大面积永远是 `var(--bg)` / `var(--surface)`；品牌渐变只允许出现在底部「新建任务」主按钮和创建弹层主按钮上。进度条填充用 `var(--accent)`，轨道用 `var(--accent-wash)` 或 `var(--border)`，不要自造绿/黄/红进度色。
- 语义色只用 token：进行中/主进度 = `var(--accent)` + `var(--accent-wash)`；待回应 = `var(--warning)` + `var(--warning-wash)`；已结束/次要 = `var(--muted)` / `var(--border)`；失败 = `var(--danger)` + `var(--danger-wash)`。**不要自造颜色。**
- 触控目标 ≥ 44px；正文 ≥ 13px（`var(--text-base)`）；标题 15–16px、字重 600。
- **禁止 `white-space: nowrap` 用于可能超宽的文本**（任务标题、日期区间、待回应文案都可能长，必须正常换行）。
- 所有多列统计/分段用 `grid-template-columns: repeat(N, minmax(0, 1fr))`；所有容器 `max-width: 100%`；页面不得出现横向滚动。
- 底部内容留 `padding-bottom: calc(76px + env(safe-area-inset-bottom))`；演示条用行内 `style="bottom: calc(64px + 12px + env(safe-area-inset-bottom))"` 避开固定操作栏。
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
      <button class="mp-back" type="button" aria-label="返回老师工作台">
        <svg width="20" height="20" viewBox="0 0 20 20" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M12.5 4.5 7 10l5.5 5.5"/></svg>
      </button>
      <h1 class="mp-title">亲子任务</h1>
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
