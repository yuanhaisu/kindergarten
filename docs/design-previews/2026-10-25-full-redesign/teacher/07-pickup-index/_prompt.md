你是幼儿园管理平台「养成星球」的资深移动端设计师。为**微信小程序老师端**做一个页面的高保真 HTML 原型。

## 唯一任务
页面：接送管理（源码：teacher/src/pages/pickup/index.vue）
只做一个方向，不要 A/B 变体，不要多页面。写完 index.html / styles.css / app.js 三个文件即停止，不要提问，不要联网，不要读自己的产物做自查循环。

## 这个页面真实的产品事实（不要虚构其它能力）

老师端接送台。老师站在园门口，一手拿手机，逐个把本班孩子「送入 / 接走」登记掉，并处理放不下学的滞留孩子。

**顶部今日概览**（后端 GET /teacher/pickup/overview 真实字段）：
`total 全班在读` / `arrived 已入园` / `not_arrived 未到园` / `picked_up 已接走` / `not_picked_up 未接走（待接）` / `overdue_not_picked 曾滞留学生数` / `abnormal 今日异常记录数`
注意 `arrived = picked_up + not_picked_up`。

**筛选**（后端 status 参数，源码 4 个选项）：
`not_picked_up 未接走`（默认） / `all 全部` / `arrived 已入园` / `picked_up 已接走`
源码用「点一下循环切换下一个」的按钮，这次改成 4 个并排的分段选项（同一组真实选项，不是新功能）。

**学生行**（GET /teacher/pickup/students 真实字段）：
`name` 姓名 / `avatar` 头像 / `authorized_persons` 有效授权接送人（逗号或「、」连接，空时后端返回「暂无」）/ `arrive_time` 当日首次入园时刻（形如 08:12，**后端已返回但现页面没展示**）/ `pickup_status` = `not_arrived` 未到园 | `not_picked_up` 已入园待接走 | `picked_up` 已接走 / `is_overdue` 已过放学时间仍在园 / `pickup_deadline` 放学时间（形如 17:30）/ `last_remark` 上次登记备注 / `is_authorized_override` 上次为授权例外

**每行动作**（源码原样）：
- `pickup_status = not_arrived` 时出现「入园」按钮 → 直接登记送入
- 始终有「接走」按钮；已接走时文案变「再接走」
- `is_overdue` 为真时出现「通知」按钮 → 一键通知滞留孩子家长
- 后端规则：**没登记入园的孩子不能直接接走**（会报「该生今日尚未入园，请先登记入园（送入）后再登记接走」）。所以未到园那一行，把「接走」做成禁用态（灰、aria-disabled），这是把后端既有规则前置到 UI，不是新功能。

**手动登记**（源码弹窗）：
- 「选授权人」模式：radio 卡片列表，每张显示 `name` / `relation · phone`；懒加载该生有效授权人
- 「标例外」模式：填「来接者姓名」（placeholder 如「邻居王五」）—— 这就是**异常放行**
- 两种模式都有「理由」textarea（重复接走 / 提前接走时必填）
- 底部 取消 / 确认登记
- 选授权人模式下没选人、标例外模式下没填姓名时，确认按钮禁用

**扫码**（源码 uni.scanCode → POST /teacher/pickup/scan）：
扫家长出示的接送二维码，后端从码里自解释学生与接走人，回包有 `student.name` / `person.name` / `person.relation` / `type_label`（送入 / 接走）/ `is_authorized` / `is_authorized_override` / `recorded_at`。
核验失败会返回可操作的原因，源码把它映射成：
- 「该生今日尚未入园，请先登记入园（送入）后再登记接走」
- 「重复接走需填写理由」
- 「所选接送人授权已失效，请重新选择或标例外」
核验方式只有两种值：`扫码` / `手动`。

**入口关系**：本页由老师工作台 `navigateTo` 进入 → **必须画返回箭头**。老师端**没有 tabBar**（任何老师端页面都不画）。「接送记录」按钮跳 `pages/pickup/records`。

## 样例数据（请原样使用，数字必须自洽：arrived = picked_up + not_picked_up）

今日：2026年10月26日 周一，现在 17:42，本班放学时间 17:30

概览：total 10 · arrived 9 · not_arrived 1 · picked_up 3 · not_picked_up 6 · overdue_not_picked 2 · abnormal 1

学生（10 个，顺序如下；默认只显示 `not_picked_up` 的 6 个）：

| 姓名 | 授权接送人 | 入园 | 状态 | 标记 | 备注 |
|---|---|---|---|---|---|
| 乐乐 | 李梅（妈妈）、王强（爸爸） | 08:12 | not_picked_up | 滞留 | 放学 17:30，请尽快联系家长 |
| 小雨 | 张敏（妈妈） | 08:05 | not_picked_up | 例外 | 上次登记：邻居代接，已电话确认 |
| 朵朵 | 暂无 | 07:58 | not_picked_up | — | — |
| 果果 | 刘芳（奶奶）、刘明（爸爸） | 08:20 | not_picked_up | 滞留 | 放学 17:30，请尽快联系家长 |
| 豆豆 | 陈静（妈妈） | 08:30 | not_picked_up | — | — |
| 米米 | 赵霞（妈妈） | 08:41 | not_picked_up | — | — |
| 团团 | 李梅（妈妈） | 08:02 | picked_up | — | 已接走 17:05 · 李梅（妈妈） |
| 安安 | 王强（爸爸） | 07:55 | picked_up | — | 已接走 16:52 · 王强（爸爸） |
| 妞妞 | 刘芳（奶奶） | 08:18 | picked_up | — | 已接走 16:38 · 刘芳（奶奶） |
| 天天 | 孙丽（妈妈） | 未入园 | not_arrived | — | 今日未登记入园 |

班级：小一班。老师：王老师。
授权人手机号用 138**** 这种打码写法。

## 视觉与结构硬约束（逐条照做）

- `<html lang="zh-CN" data-surface="teacher">`，390px 手机视口，单列纵向流，不做桌面布局
- 微信小程序外壳：**必须**用下面给出的 `mp-*` 标记原样照抄；页面 `styles.css` 里**不得出现任何 `.mp-*` 选择器定义**（唯一例外是 `[data-state="..."]` 前缀的状态选择器）
- 引入顺序固定：
  ```html
  <link rel="stylesheet" href="tokens.css">
  <link rel="stylesheet" href="../../_shared/shell.css">
  <link rel="stylesheet" href="styles.css">
  ```
- 导航栏：**画返回箭头**（`mp-back`），标题「接送管理」；**不画 tabBar**
- 颜色只能写 `var(--*)`，**禁止任何裸 hex / rgb / hsl**；无直角（最扁平元素 ≥ `var(--radius-sm)`，按钮标签用 `var(--radius-pill)`，卡片用 `var(--radius-card)`）；阴影只用 `var(--shadow-sm|md|lg)`
- 暖色面积 ≤ 10%：大面积永远是 `var(--bg)` / `var(--surface)`；渐变只允许出现在底栏的「扫码接送」主按钮（`var(--accent-gradient)`）
- 状态色**只能用语义 token**：滞留/失败 = `var(--danger)` + `var(--danger-wash)`；例外 = `var(--warning)` + `var(--warning-wash)`；已接走 = `var(--success)` + `var(--success-wash)`；未到园 = 中性 `var(--muted)`。「待接走」不要上色，用中性 pill。**不要自造绿黄红**
- 触控目标 ≥ 44px；正文 ≥ 13px（`var(--text-base)`）
- **禁止 `white-space: nowrap` 用于可能超宽的文本**（授权人姓名列表可长，必须正常换行）
- 所有多列统计用 `grid-template-columns: repeat(N, minmax(0, 1fr))`；所有容器 `max-width: 100%`；页面不得出现横向滚动
- 无 JS 报错；不用 alert / prompt；点击必须有 `:active` 反馈（缩放或压暗）
- 每个文件 < 20KB；不引用任何外部资源（字体、图片、CDN 都不行）；头像用首字 + 圆形色块

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
      <h1 class="mp-title">接送管理</h1>
      <div class="mp-capsule" aria-hidden="true">
        <span class="mp-capsule-dots">···</span>
        <span class="mp-capsule-divider"></span>
        <span class="mp-capsule-ring">◎</span>
      </div>
    </header>

    <main class="mp-main page">…</main>

    <!-- 固定底栏：页面自己的类名，不用 mp-* -->
    <div class="actionbar">…</div>

    <div class="mp-demo" data-open="false" style="bottom: calc(64px + 12px + env(safe-area-inset-bottom))">
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
</body>
```

## 页面结构（从上到下）

1. **今日概览区**（不套大卡片的外框感太重，做成一张 `var(--surface)` 卡）
   - 第一行：「今日接送」标题 + 右侧日期「10月26日 周一」（小字、`var(--fg-2)`、tabular-nums）
   - 第二行：4 列数字带，用 `grid-template-columns: repeat(4, minmax(0, 1fr))`：
     `未到园 1` / `待接走 6` / `已接走 3` / `异常 1`
     数字 `var(--text-3xl)` + 600；标签 `var(--text-xs)` + `var(--muted)`。「待接走 6」这个最需要被看见的数字可以用一个 `var(--accent-wash)` 圆角 pill 包住数字（这是页面上仅有的两处暖色之一）。「异常 1」用 `var(--danger)` 数字（1 很小，不构成大面积）
   - 第三行（仅当有滞留时）：一条 `var(--danger-wash)` 底的细提示条，左 `var(--danger)` 小图标 + 文案「2 名孩子已过放学时间 17:30 仍在园」，纯信息，不可点
2. **筛选**：4 个并排分段（未接走 / 全部 / 已入园 / 已接走），`grid-template-columns: repeat(4, minmax(0, 1fr))`，选中项 `var(--surface)` 底 + `var(--fg)` 600 字重 + `var(--shadow-sm)`，容器 `var(--bg)` 内凹胶囊；默认选中「未接走」
3. **列表标题行**：「未接走学生（6人）」，数字随筛选联动
4. **学生卡片列表**（单列，间距 10px）：
   - 左：40px 圆头像（首字，`var(--info-wash)` 类中性底，不要用白字彩色底）
   - 中（`min-width: 0`，可换行）：第一行 姓名（`var(--text-lg)` 600）+ 状态 pill；
     第二行「授权接送人：李梅（妈妈）、王强（爸爸）」（`var(--text-sm)`，`var(--muted)`，可换行）；
     第三行「入园 08:12」（`var(--text-xs)`，`var(--hint)`，tabular-nums）；
     滞留行追加「放学 17:30，请尽快联系家长」（`var(--text-xs)`，`var(--danger)`）；
     有 last_remark 时追加一行（`var(--text-xs)`，`var(--muted)`，前置「上次登记：」）
   - 右：动作列（竖排，右对齐，`flex: none`）
     - 主按钮「接走」/「再接走」：`var(--radius-pill)`，`var(--accent-wash)` 底 + `var(--accent-deep)` 字
     - 滞留时「通知」：`var(--danger)` 描边或 `var(--danger-wash)` 底 + `var(--danger)` 字
     - 未到园时「入园」：`var(--success-wash)` 底 + `var(--success)` 字；同行「接走」置灰禁用（`aria-disabled="true"`，`var(--muted)` 字 + `var(--surface-warm)` 底 + 细边框）
   - 状态 pill：`已接走` = success；`滞留` = danger；`例外` = warning；`未到园` = 中性 `var(--muted)` + 描边；`待接走` = 中性（不上色，`var(--bg)` 底 + `var(--fg-2)` 字）
   - 已接走的卡片整体降音量：白底、无阴影、文字 `var(--muted)`，只留姓名 + 「已接走 17:05 · 李梅（妈妈）」+ 右侧「再接走」
5. **底部固定操作栏**（`.actionbar`，`position: sticky; bottom: 0`，页面安全区 `padding-bottom: env(safe-area-inset-bottom)`，高 64px，`var(--surface)` 底 + 1px 顶边框）
   - 左：两个安静的图标+文字按钮「手动登记」「接送记录」（图标 20px 描边 svg + `var(--text-xs)` 标签，`var(--fg-2)`）
   - 右：「扫码接送」主按钮 —— `var(--radius-pill)`，`var(--accent-gradient)` 底 + `var(--on-brand)` 字，高 44px，扫码图标 + 文案，是整个页面唯一的渐变块
   - 顶部再给一句极小的辅助说明（在内容区最后 / 或概览卡下方）：「家长出示接送码，扫码即可登记」——`var(--text-xs)`、`var(--muted)`

## 必须能点（app.js）

1. **筛选**：点 4 个选项 ⇒ 按 `data-status` 过滤卡片（未接走只看 not_picked_up；已入园 = not_picked_up + picked_up；已接走只看 picked_up；全部都看），标题里的「（N人）」同步更新；结果为 0 时显示「当前筛选下没有学生」的轻空态行
2. **接走 / 再接走** ⇒ 打开「手动登记」底部弹层（见下），标题带上该生姓名
3. **入园** ⇒ 直接 toast「乐乐 已入园」（不弹窗）
4. **通知** ⇒ toast「已通知乐乐家长」，按钮变「已通知」并禁用（只在本页演示，不改别的行）
5. **底栏「手动登记」** ⇒ 先弹一个「选择学生」底部弹层（10 行：姓名 + 状态），选一个进入「手动登记」弹层
6. **底栏「接送记录」** ⇒ toast「已打开 接送记录」
7. **底栏「扫码接送」** ⇒ 打开「扫码核验」底部弹层：
   - 打开瞬间显示「核验中…」骨架/小圆环，约 700ms 后切到 **核验失败**：
     `var(--danger-wash)` 图标 + 「核验未通过」+ 「朵朵 · 小一班」+ 原因「该家长不在朵朵的有效授权接送人内，扫码登记被拒绝」+ 一句「可改为「标例外」手动登记，并补充理由」
     动作：`标例外登记`（主，进手动登记弹层的 override 模式，学生=朵朵） / `重新扫码`（次）
   - 点「重新扫码」⇒ 切到 **核验成功**：`var(--success-wash)` 图标 + 「核验通过」+ 「乐乐 · 接走 · 17:42」+ 接送人「李梅（妈妈）」+ 核验方式「扫码」+ 底部「完成」按钮（关弹层 + toast「乐乐 已接走」）
8. **手动登记弹层**（底部弹出，圆角顶部 `var(--radius-2xl)`，最高 82vh 内部滚动）：
   - 标题「手动登记 · 乐乐」+ 右上关闭
   - 分段：`选授权人` / `标例外`
   - 选授权人模式：radio 卡片（40px 头像 + 姓名 + 「relation · 138****6688」），可选中；选授权人为 **朵朵** 时显示「暂无有效授权接送人，可切换到「标例外」」（真实字段：authorized_persons = 暂无）
   - 标例外模式：一个 label「来接者姓名」+ input（placeholder「如：邻居王五」）
   - 始终有「理由」textarea（placeholder「重复接走 / 提前接走时必填」，maxlength 200）
   - 底部：`取消` / `确认登记`（未选人 / 未填姓名时 `aria-disabled="true"` 且视觉降级；满足条件后点击 toast「登记成功」并关闭）
9. **所有弹层**：点遮罩关闭、ESC 关闭、有 `role="dialog" aria-modal="true"`；打开时不让背景滚动
10. **状态切换时清理**：切到 空/加载/失败 时关闭所有弹层；「重试」按钮走演示 chip（保证 chip 选中态同步，单一状态出口）

## 四态（用 `.mp-app[data-state=...]` 属性选择器，不写 JS 切类）

- **normal**：完整展示
- **empty**：隐藏概览卡 + 滞留条 + 筛选 + 列表标题 + 列表；显示空态块：`var(--bg)` 圆形图标位 + 「今天没有待接的孩子」+ 「全部孩子已完成接送，下拉可刷新」+ 主按钮「查看今日记录」（`var(--accent-wash)` + `var(--accent-deep)`）。底栏保留（扫码与手动登记在空态下依然可用）
- **loading**：概览数字换成骨架条（`var(--border)` 半透明脉冲），列表换成 3 条骨架卡；筛选与标题保留
- **error**：隐藏概览 + 列表；显示错误块：`var(--danger-wash)` 圆图标 + 「接送数据加载失败」+ 一句「网络异常，未获取到今日接送数据，请勿默认孩子都已接走」+ 主按钮「重试」。底栏保留
- **禁用态**：未到园行的「接走」、手动登记未满足条件时的「确认登记」、已通知过的「通知」

## 输出文件（项目根目录）

1. `index.html` —— 结构（外壳 + 内容 + 底栏 + 3 个弹层 + toast 容器）
2. `styles.css` —— 页面样式，只写 `var(--*)`，不出现 `.mp-*` 定义
3. `app.js` —— 上述全部交互

`tokens.css` 已由流程提供，**不要重写、不要覆盖**。每个文件 < 20KB。
