你是幼儿园管理平台「养成星球」的资深移动端设计师。为家长端做一个页面的高保真 HTML 原型。

## 唯一任务
页面：成长详情（源码：parent/src/pages/growth/detail.vue）
只做一个方向，不要 A/B 变体，不要多页面。写完全部文件即停止：不要提问，不要联网，不要读取自己刚写的产物做自查，不要覆盖 tokens.css。

## 这个页面真实的产品事实

**定位**：家长从「成长时光」列表点进某一条成长记录的阅读页。`uni.navigateTo` 进入的子页 → 导航栏**必须有返回箭头**，标题「成长详情」；**不画 tabBar**（家长端只有首页/成长/接送/我的四个 tab 页画，本页不是）。

入口：成长列表卡片 → `pages/growth/detail?id=...`。
数据：进页并行拉两个接口——`GET /parent/growth/feed/:id?student_id=`（记录详情）与 `GET /parent/growth/feed/:id/comments`（评论列表，返回 `{ list }`）；失败互不连累。
点赞：`POST /parent/growth/feed/:id/like` / `DELETE` 同一路径，返回 `is_liked`、`likes_count`。**注意：后端目前不落库（POST 返回 true，重新 GET 仍是 false），原型只做本地即时反馈，不得出现「已同步 / 已保存 / 已点赞成功」之类的持久化暗示。**
发评论：`POST /parent/growth/feed/:id/comments`，体 `{ content, parent_id }`；成功后清空输入、toast「评论成功」、重新拉评论列表。
回复：点某条评论的「回复」→ 输入栏上方出现「回复中 / 取消」，`parent_id` = 被回复评论 id。

区块顺序（自上而下，源码固定）：
1. **作者卡**：圆形头像（无头像用姓氏首字占位）+ 昵称（`creator.nickname`，缺省显示「老师」）+ 时间行 `date · HH:mm`（例「10月22日 10:05」）；右侧获奖徽章（可空，`badge.name`；`color_kind=warning` 用 warning 语义色，其余用 info 语义色）。源码**没有标题字段**——所谓「标题」只体现为徽章与表现标签，不要编一个标题。
2. **当日状态 4 列**（仅当 `breakfast_rating` 存在时出现）：早餐 / 午餐 / 午睡 / 情绪。早餐、午餐、情绪取值为「差 / 中 / 好」；午睡为「90分钟」（`nap_duration`）。四等分，`grid-template-columns: repeat(4, minmax(0, 1fr))`，手机宽度下不能挤爆。
3. **正文卡**：
   - 正文 `content`：保留换行（`white-space: pre-wrap`），15px / 行高 1.8。
   - 表现标签 `performance_tags[]`：暖色浅底胶囊，自动换行。
   - **照片区**：3 列网格，每张 `aspect-ratio: 1 / 1`（**必须不变形**），gap 6px，圆角。点任意一张 → 打开**底部弹层预览**（标题「照片预览」，内容区是同一张占位图放大，底部/标题旁标「第 N 张 / 共 M 张」，有关闭按钮，点遮罩或 Esc 关闭）。点名到当前孩子的照片，左下角有「我的孩子」小胶囊。
   - **照片筛选**（只对全班记录出现，`target_type !== 2`）：两个胶囊「只看我的孩子」（默认选中）/「看全部（6）」。**切换必须真的过滤照片，不能只做视觉切换**：默认只显示点名到乐乐的照片；切「看全部」显示全部。个体记录（`target_type === 2`）不出现筛选条。
   - **照片空态**（默认视图里没有拍到自家孩子时）：一句「这些照片里暂时没有拍到您的孩子」+ 主按钮「查看全部照片（6）」，点了切到全部视图。本示例数据下不可达，但样式与结构要做出来（可默认隐藏）。
   - **相关孩子**（`related_students`，仅「发给指定学生」的记录有值）：「相关孩子（N人）：乐乐、朵朵」。本示例是全班记录，这块**不出现**，也不要为它编数据。
4. **互动栏**：左侧心形 + 点赞数（已赞时图标与数字用品牌色，`aria-pressed="true"`）、气泡 + 评论数；右侧「评论」文字按钮（点了聚焦底部输入栏）。
5. **评论列表**：标题「评论 (4)」；每条：昵称 + 关系标签（`relation` 为 mother→「妈妈」、father→「爸爸」，无值（如老师）不显示）+ 时间（`10月22日 10:26`）+ 正文 + 「回复」；子回复缩进挂在父评论下（左侧细竖线 + 浅底小卡）。**没有评论点赞、没有删除、没有置顶**——源码没有的能力一律不画。
6. **底部固定评论栏**：无回复时是一行——圆角输入框（placeholder「写评论...」）+「发送」胶囊按钮；回复态在输入行上方多一行「回复中 · 取消」。输入为空时「发送」按钮降级（浅底 / 低对比、`aria-disabled="true"`、不响应点击），因为源码空文本直接 return。底部 `padding-bottom: calc(8px + env(safe-area-inset-bottom))`，内容区底部必须留出**等于栏高 + 安全区**的空隙，最后一条评论不能被遮住。

## 示例数据（必须用这一组，不要另编）

当前孩子：乐乐 / 小一班（本页不直接展示孩子名，仅「我的孩子」角标与评论语境使用）。

记录：李老师 · 10月22日 10:05 · 全班记录（`target_type=1`）
获奖徽章：**运动小达人**（warning）
当日状态：早餐 好 / 午餐 好 / 午睡 90分钟 / 情绪 好
正文两段：
1. 户外活动时间，孩子们在操场上玩「小兔拔萝卜」，乐乐跑得满头大汗，还主动把水杯递给了朵朵。
2. 回到教室后，乐乐自己换了汗湿的背心，把外套叠好放进柜子，还提醒旁边的小朋友喝水。
表现标签：大动作、关心同伴、自理能力
照片：6 张，其中 2 张点名到乐乐（带「我的孩子」角标）；默认「只看我的孩子」显示 2 张，切「看全部（6）」显示 6 张。
互动：点赞 12（未赞）、评论 4。

评论（4 条，其中 1 条有 1 条子回复）：
| # | 昵称 | 关系 | 时间 | 正文 |
|---|---|---|---|---|
| 1 | 乐乐妈妈 | 妈妈 | 10月22日 10:26 | 看到乐乐把水杯递给朵朵，在家里都没这么主动过。 |
| 1-回复 | 李老师 | — | 10月22日 10:32 | 今天他还帮小朋友捡了好几次萝卜。 |
| 2 | 朵朵妈妈 | 妈妈 | 10月22日 10:41 | 朵朵回家也说今天玩得很开心，谢谢李老师。 |
| 3 | 小雨爸爸 | 爸爸 | 10月22日 12:03 | 照片里的笑容太有感染力了。 |
| 4 | 李老师 | — | 10月22日 12:30 | 下午还有手工课，晚点再发一组照片。 |

照片占位：只用 CSS 渐变（组合 `var(--accent-wash)`、`var(--info-wash)`、`var(--success-wash)`、`var(--bg)` 等 token 变量）做出柔和色块 + 极淡的线性山丘/太阳图案；**禁止外链图片、禁止 emoji、禁止 base64 大图**。6 张的色调要有区分，但整体低对比、不喧哗。在照片预览弹层里，「我的孩子」照片同样要有角标或标题说明。

## 状态覆盖（数据区必须四态，用 `.mp-app[data-state]` 纯 CSS 切换）
- **正常**：默认，上表数据完整渲染。
- **空态 = 评论为空**（演示 chip 标签「空评论」）：记录、照片、互动栏全部保留；评论列表位置换成评论空态块——图标位 + 一句「还没有评论」+ 说明「来说说这条记录里你看到的小细节」+ 主按钮「写第一条评论」（点了聚焦底部输入栏）。不要把整条记录也清空。
- **加载**：整页骨架——作者行（圆头像 + 两条文字条）、4 个状态块、正文 3 行、3×2 照片网格、互动栏、2 条评论骨架；`--border` 呼吸动画，`prefers-reduced-motion` 下停止。底部评论栏进入禁用降级（输入框 disabled、发送灰）。
- **失败**：整页错误卡（`--danger-wash` 圆底 + `--danger` 感叹号 +「这条记录加载失败」+「网络异常，或老师已删除这条记录」+ 描边「重试」）；底部评论栏禁用；导航与返回箭头保持可用。
- **禁用态**：加载/失败态下，发送、点赞、照片筛选、照片点击统一 `aria-disabled="true"` + 降级配色，不响应点击。

右下角放原型演示条（共享 `shell.js` 提供），行内 `style="bottom: calc(84px + env(safe-area-inset-bottom))"`（浮在评论栏之上），默认收起，四个芯片：正常 / 空评论 / 加载 / 失败。

## 视觉与结构硬约束
- `<html lang="zh-CN" data-surface="parent">`，390px 手机视口，单列纵向流，不做桌面布局。
- 引入顺序固定：`tokens.css` → `../../_shared/shell.css` → `styles.css`。
- 颜色**只能**写 `var(--*)`，禁止裸 hex / rgb / rgba（HTML 内联 SVG 也只许 `currentColor`）；阴影只用 `var(--shadow-*)`；不得出现直角（最扁平元素 ≥ `var(--radius-sm)`，卡片 `var(--radius-card)`，按钮/标签/头像/输入框 `var(--radius-pill)`）。
- 暖色面积 ≤ 10%：页面底 `var(--bg)`、卡片 `var(--surface)`；橙色只出现在作者头像底、表现标签、已点赞态、发送按钮与「我的孩子」角标，不要整块铺渐变。
- 层级只靠字重 + 字号：标题 600 / 正文 400 / 标签 500。
- 所有可能超宽的文本禁止 `white-space: nowrap`；多列统计用 `grid-template-columns: repeat(N, minmax(0, 1fr))`；所有容器 `max-width: 100%`，页面不得横向滚动。
- 触控目标 ≥ 44px（「回复」这类文字按钮可用内边距撑到 44px 高）；正文 ≥ 13px；`:active` 反馈用 scale / 压暗；不用 alert / prompt；无 JS 报错。
- 不引用任何外部资源；单文件 < 20KB。
- **`styles.css` 里不得出现 `.mp-*` 定义**，不得复制 `shell.css`，不得创建 `_shared/` 目录，不得写 `.mcp.json`。

## 外壳（唯一实现，必须原样使用 `mp-*` 类名）
只写 `<link rel="stylesheet" href="../../_shared/shell.css">`，该文件由流程提供。

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
    <h1 class="mp-title">成长详情</h1>
    <div class="mp-capsule" aria-hidden="true">
      <span class="mp-capsule-dots">···</span>
      <span class="mp-capsule-divider"></span>
      <span class="mp-capsule-ring">◎</span>
    </div>
  </header>

  <main class="mp-main">…页面内容…</main>

  <!-- 底部固定评论栏（页面自己实现，不是外壳） -->

  <div class="mp-demo" data-open="false" style="bottom: calc(84px + env(safe-area-inset-bottom))">
    <div class="mp-demo-chips" role="group" aria-label="原型状态">
      <button class="mp-demo-chip" type="button" data-mp-state="normal" aria-pressed="true">正常</button>
      <button class="mp-demo-chip" type="button" data-mp-state="empty" aria-pressed="false">空评论</button>
      <button class="mp-demo-chip" type="button" data-mp-state="loading" aria-pressed="false">加载</button>
      <button class="mp-demo-chip" type="button" data-mp-state="error" aria-pressed="false">失败</button>
    </div>
    <button class="mp-demo-toggle" type="button" aria-expanded="false">原型演示</button>
  </div>
</div>
```

## 交互（app.js 必须真的能点）
1. 点赞：本地切换 `is_liked` 与数字 ±1，只改 UI；不要请求、不要提示已保存。
2. 照片：点开底部弹层，显示对应序号与张数；关闭按钮 / 点遮罩 / Esc 都能关。
3. 照片筛选：两个胶囊切换，**真的增减照片节点**（不是只换选中样式）。
4. 回复：点某条「回复」→ 输入栏出现「回复中 · 取消」；取消恢复普通态。
5. 输入：有内容才启用发送；发送后清空输入、退出回复态、toast「评论成功」、把新评论插到列表顶部（纯本地演示）。
6. 点互动栏右侧「评论」聚焦输入框。
7. 状态切换条四态可切。

## 输出文件（项目根目录）
1. index.html —— 结构
2. styles.css —— 样式（引用 tokens.css，外壳样式一律不加）
3. app.js —— 交互
tokens.css 已由流程提供，不要重写，不要覆盖。
每个文件 < 20KB。
