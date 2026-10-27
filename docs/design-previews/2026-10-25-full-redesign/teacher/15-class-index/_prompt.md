你是幼儿园管理平台「养成星球」的资深移动端设计师。为**微信小程序老师端**做一个页面的高保真 HTML 原型。

## 唯一任务
页面：我的班级（源码：teacher/src/pages/class/index.vue）
只做一个方向，不要 A/B 变体，不要多页面。写完 index.html / styles.css / app.js 三个文件即停止，不要提问，不要联网，不要读自己的产物做自查循环。

## 这个页面真实的产品事实（不要虚构其它能力）

老师端「我的班级」是班级信息页：老师查看自己带的班、班里的老师、今天的接送概览，以及全班幼儿名单。数据来自三个真实接口，页面一次性 `Promise.all` 拉取：

- `GET /teacher/class/info` → `{ id, name 班名, grade 年级, academic_year 学年, student_count 人数, teacher: { teacher_name 老师姓名, title_label 职务, teacher_avatar, kindergarten_name 园所名 } }`
- `GET /teacher/class/students` → `{ list: [{ student_id, name, avatar, authorized_persons 授权接送人, arrive_time 入园时间, pickup_status, is_overdue, pickup_deadline, last_remark }], total }`
- `GET /teacher/pickup/overview` → `{ date, total, arrived 已入园, not_arrived 未入园, picked_up 已接走, not_picked_up 未接走, overdue_not_picked, abnormal }`

**关键口径（照抄源码，不要自己发挥）**：
- `pickup_status` 只有三个值：`not_arrived` 未入园 / `not_picked_up` 未接走 / `picked_up` 已接走。**没有出勤字段**，页面上的「今日概览」用的是接送统计，不要写成考勤/签到。
- 花名册筛选只有 4 项：全部 / 未入园 / 未接走 / 已接走。搜索**只按姓名**（后端不返回学号，不要画学号搜索）。
- 源码里**没有**「编辑班级」「添加幼儿」「查看家长电话」「置顶」等能力，不要发明。
- 点击幼儿卡片：源码卡片本身没有跳转。原型允许打开一个**只读**的底部简要信息卡，展示该学生对象里已有的字段：头像、姓名、接送状态、入园时间、授权接送人、接送截止时间。不要放任何会改数据的按钮。
- 顶部右上角的铃铛是真实入口：跳 `pages/messages/index`（原型里用 toast 模拟）。铃铛上的红点代表未读，源码有。
- 老师端**没有 tabBar**，本页由工作台 `navigateTo` 进入 → **必须画返回箭头**。

**本页刻意不画**（源码有、但与本页「班级信息」定位重复，属于本批次的设计取舍）：源码的「班级动态」三快捷入口（发布通知 / 记成长 / 接送）与「最新通知」列表。它们分别是工作台与消息页的职责；本页保留右上铃铛作为去消息页的入口即可。不要自作主张把它们加回来。

## 示例数据（请原样使用，数字必须自洽）

- 今天：2026年10月26日 周一
- 班级：小一班；年级：小班；学年：2026-2027 学年；园所：阳光幼儿园
- 老师：李老师 · 主班老师（`title_label`）；班级人数：30 人
- 今日接送：已入园 28 / 未入园 2 / 已接走 6 / 未接走 22（6+22=28，与 30 人对得上）
- 花名册筛选计数：全部 30 / 未入园 2 / 未接走 22 / 已接走 6

30 名幼儿（姓名、状态、入园时间；未入园的时间为「—」，其余给 07:58–08:26 之间的时间）：

| 姓名 | 状态 | 入园时间 |
|---|---|---|
| 乐乐 | 已接走 | 08:02 |
| 小雨 | 已接走 | 08:05 |
| 朵朵 | 未接走 | 08:07 |
| 安安 | 已接走 | 08:00 |
| 贝贝 | 未接走 | 08:11 |
| 辰辰 | 未接走 | 08:09 |
| 晨晨 | 未接走 | 08:14 |
| 豆豆 | 未入园 | — |
| 果果 | 已接走 | 07:58 |
| 涵涵 | 未接走 | 08:16 |
| 浩浩 | 未接走 | 08:04 |
| 嘉嘉 | 未接走 | 08:19 |
| 可心 | 未接走 | 08:06 |
| 可可 | 未接走 | 08:12 |
| 朗朗 | 未接走 | 08:21 |
| 满满 | 未接走 | 08:08 |
| 苗苗 | 未接走 | 08:13 |
| 沐沐 | 未接走 | 08:03 |
| 暖暖 | 未接走 | 08:17 |
| 皮皮 | 未入园 | — |
| 七七 | 未接走 | 08:22 |
| 然然 | 未接走 | 08:10 |
| 睿睿 | 未接走 | 08:18 |
| 糖糖 | 未接走 | 08:15 |
| 彤彤 | 未接走 | 08:20 |
| 晚晚 | 未接走 | 08:24 |
| 西西 | 未接走 | 08:23 |
| 小北 | 未接走 | 08:25 |
| 一诺 | 已接走 | 08:01 |
| 元宝 | 已接走 | 08:26 |

授权接送人示例（轮换使用，真实字段）：妈妈、爷爷 / 爸爸、外婆 / 妈妈 / 爸爸、奶奶 / 外公、妈妈。接送截止时间统一 `16:30`（个别 `17:00` 可接受）。

## 页面结构（从上到下）

1. **微信外壳**：20px 状态栏（9:41 + 图标）+ 44px 导航栏「我的班级」+ 左侧返回箭头 + 右侧胶囊。**不画 tabBar。**
2. **紧凑 hero（唯一渐变区域）**：
   - 左侧：班级徽记（圆形浅底 + 班级首字「小」或一个 SVG 图标，60px 左右）
   - 主标题：小一班（20px / 700 / `var(--on-brand)`）
   - 上行小字：小班 · 2026-2027 学年（12px，白色 70%）
   - 下行小字：阳光幼儿园（12px，白色 70%）
   - 右侧：铃铛按钮（44×44 触控目标，白色 20% 圆形底，右上红点）
   - 底部一行：老师小头像 + 「李老师 · 主班老师」+ 分隔点 + 「30 人」小胶囊
   - **不要大数字堆砌**，hero 高度控制在 ~110px，不要画四个巨大数字。
3. **今日接送概览卡**（普通白卡，不是渐变）：
   - 头部：标题「今日接送」+ 右侧日期「10月26日 周一」
   - 四列统计：`grid-template-columns: repeat(4, minmax(0, 1fr))`，每列 = 小色点 + 标签（11px）+ 数字（20px / 600，不要超过 22px）
   - 顺序：已入园（success）/ 未入园（danger）/ 已接走（info）/ 未接走（warning）
   - 四列之间不要竖线，用留白分隔；卡片圆角 `var(--radius-card)`，阴影 `var(--shadow-sm)`
4. **幼儿名单区**：
   - 区头：左侧「幼儿名单」14px / 600；右侧计数「30 人」12px muted
   - 工具行（上下两行，不要横向滚动）：
     - 搜索框：整行宽，44px 高，左侧放大镜 SVG，占位「搜索学生姓名」
     - 筛选：4 个等宽胶囊（`repeat(4, minmax(0, 1fr))`），文案「全部 / 未入园 / 未接走 / 已接走」，各带小计数。选中项 = `var(--accent)` 底 + `var(--on-brand)` 字；未选中 = surface 底 + 1px border。触控高度 ≥ 36px，pill 圆角。
   - **网格卡片**：3 列 `repeat(3, minmax(0, 1fr))`，gap 8px。每张卡：
     - 圆形头像 40px：头像图不存在时用姓名首字（底色用 `color-mix(in srgb, var(--accent) 16%, var(--surface))`，字色 `var(--accent-deep)`）
     - 姓名 13px / 600，最多一行，超出用 `-webkit-line-clamp: 1`（不要 `white-space: nowrap`）
     - 状态：小圆点 6px + 11px 文案（未入园 danger / 未接走 warning / 已接走 success）
     - 入园时间 10px `var(--hint)`：`08:02 入园`，未入园显示 `还没入园`
     - 卡片背景 `var(--surface)`，圆角 12px，`var(--shadow-sm)`，`min-height: 112px`，整卡是 `<button>`，`:active { transform: scale(.98) }`
   - 名单为空时（空态）：网格隐藏，显示居中空态：一个柔和圆形图标位 + 「班里还没有幼儿档案」+ 主按钮「刷新名单」。
   - 搜索/筛选无结果时：网格位置显示「没有找到符合条件的孩子」+ 文本按钮「清除筛选」（这是客户端筛选，不是后端空列表）。
5. **底部简要信息卡（bottom sheet）**：
   - 点击任一幼儿卡片打开；半透明遮罩 + 底部圆角面板（顶部 24px）
   - 顶部拖拽条；头部：48px 头像 + 姓名 18px / 600 + 状态 pill
   - 四行只读信息：入园时间 / 接送状态 / 授权接送人 / 接送截止，左侧标签 muted，右侧值 fg
   - 底部一个「关闭」按钮（次要样式，`var(--surface-warm)` 底）
   - 遮罩点击 / Esc / 关闭按钮都能关；打开时锁 body 滚动，关闭后恢复
6. **原型演示条**：用共享 `mp-demo`（右下角，默认收起），四个 chip：正常 / 空态 / 加载 / 失败。

## 四种状态（必须真的能切换）

- **正常**：上述完整内容，30 张卡片。
- **空态（名单为空）**：hero 人数变「0 人」、今日接送四个数字全变 0、名单计数「0 人」；工具行隐藏；网格隐藏；显示空态块（图标 + 「班里还没有幼儿档案」+ 「刷新名单」）。其余 hero 与概览卡保留。
- **加载**：hero 变成骨架条（徽记块 + 两行灰条 + 胶囊块）；概览卡变成 4 个骨架数字；名单工具行**保留但明显禁用**（降低不透明度、`aria-disabled="true"`、不可点），网格位置显示 6 张骨架卡；不显示空态/错误。
- **失败**：hero / 概览 / 名单全部隐藏，页面中央显示错误块：图标 + 「班级信息加载失败」+ 一句「网络好像开小差了」+ 主按钮「重试」。点重试先进入加载态，约 800ms 后回到正常态。空态里的「刷新名单」同样先加载再回正常。

## 视觉与结构硬约束

- `<html lang="zh-CN" data-surface="teacher">`；390px 手机视口；单列纵向流；不做桌面布局。
- 颜色只写 `var(--*)`，**禁止裸 hex / rgb / rgba**；无直角（最小圆角 `var(--radius-sm)`）；阴影只用 `var(--shadow-*)`。
- 暖色面积 ≤ 10%：只有 hero 用 `var(--accent-gradient)`，其余都是中性底 + 白卡。
- 字号层级只靠字重 + 字号；正文 ≥ 13px（辅助信息 10–12px 仅限时间/计数）；触控目标 ≥ 44px（筛选 pill 可 36px 高，但间距要够）。
- **禁止 `white-space: nowrap`**；所有容器 `max-width: 100%`；页面不得横向滚动。
- 写完 index.html / styles.css / app.js 即停。不要覆盖 tokens.css。

## 外壳（唯一实现，原样使用；不要复制 shell.css 进页面目录）

在 `<head>` 按顺序链接：
```html
<link rel="stylesheet" href="tokens.css">
<link rel="stylesheet" href="../../_shared/shell.css">
<link rel="stylesheet" href="styles.css">
```
body 末尾先引 `../../_shared/shell.js`，再引 `app.js`。**styles.css 里不得出现任何 `.mp-*` 选择器定义**（`[data-state]` 状态选择器除外）。

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
    <button class="mp-back" type="button" aria-label="返回老师工作台">
      <svg width="20" height="20" viewBox="0 0 20 20" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M12.5 4.5 7 10l5.5 5.5"/></svg>
    </button>
    <h1 class="mp-title">我的班级</h1>
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

## 交互（app.js 必须真的能点）

- 筛选 pill 切换：按 `data-status` 过滤卡片，更新计数，`aria-pressed` 同步；无结果时显示「没有找到符合条件的孩子 / 清除筛选」。
- 搜索输入：即时按姓名过滤，与当前筛选叠加。
- 幼儿卡片：点击打开 bottom sheet，用卡片上的 `data-*` 填充分姓名 / 状态 / 时间 / 授权人 / 截止时间；关闭按钮、遮罩、Esc 都能关；打开时 `body.style.overflow='hidden'`，关闭恢复。
- 铃铛：toast「打开消息」（不用 alert/prompt）。
- 空态「刷新名单」、错误态「重试」：先切到加载态，800ms 后回正常态（同时同步演示 chip 的 aria-pressed）。
- 监听 `document` 的 `mp:state` 事件：状态变回 normal 时重新应用当前搜索/筛选。
- 控制台必须干净：无 error、无 warning。

## 输出文件（写到当前项目根目录）
1. `index.html` —— 结构（30 张幼儿卡静态写全，不要靠 JS 生成）
2. `styles.css` —— 样式（引用 tokens.css；不得出现 `.mp-*` 定义）
3. `app.js` —— 交互
`tokens.css` 已由流程提供，不要重写、不要覆盖。
每个文件 < 20KB。不要引用任何外部资源。
