你是幼儿园管理平台「养成星球」的资深移动端设计师。为家长端做一个页面的高保真 HTML 原型。

## 唯一任务
页面：积分与成就（源码：parent/src/pages/points/index.vue，路由 pages/points/index，标题「积分与成就」）
只做一个方向，不要 A/B 变体，不要多页面。写完 index.html / styles.css / app.js 即停止，不要读自己的产物自查，不要提问，不要联网。

## 这个页面真实的产品事实
这是从「我的」进入的子页（back 箭头回我的），不是 tabBar 页，**不要画底部 tabBar**。

区块顺序（严格按此）：
1. 积分 hero：等级名 level_name、总积分 total_points、班级排名 rank_in_class / class_total、距下一等级 next_level_points。
2. 两个统计：本周积分 this_week、本月积分 this_month。
3. 三段切换：积分 / 排行榜 / 成就（默认「积分」）。
4. 「积分」面板 = 积分明细流水：type（earn 获得 / spend 消耗）、source_label（任务打卡 / 周全勤 / 月全勤 / 成长日志被推荐 / 管理员调整）、amount、balance、description、created_at。
5. 「排行榜」面板：本周班级榜，rank、student.name、points、is_self；接口最多返回本班前 20 名（class_total=28）。
6. 「成就」面板：徽章网格，earned（已获得，含 earned_at、description、points）与 not_earned（未获得，只有 name、description、category）。后端 `progress` 字段未实现，**绝对不要画进度条/进度百分比**，未获得只表达「未解锁」。

没有兑换商城、没有领取按钮、没有分享按钮——源码里不存在，不要发明。

## 示例数据（必须完全照用，数字要自洽）
孩子：乐乐（中二班）。总积分 1245，等级 Lv.5「超级小星星」，下一等级 1500 分（还差 255），班级排名 3/28，本周 45，本月 180，明细共 128 条。

积分明细（余额链必须成立，本周 5 条合计正好 45）：
- 10月25日 20:30｜任务打卡｜+10｜余额 1245｜完成任务【每日阅读10分钟】
- 10月24日 19:10｜成长日志被推荐｜+10｜余额 1235｜《第一次自己系鞋带》被老师推荐
- 10月23日 20:05｜任务打卡｜+10｜余额 1225｜完成任务【自己整理书包】
- 10月22日 18:40｜任务打卡｜+5｜余额 1215｜完成任务【帮妈妈摆碗筷】
- 10月20日 08:30｜周全勤｜+10｜余额 1210｜上周五天全勤
- 10月18日 15:20｜管理员调整｜-20｜余额 1200｜活动积分更正
- 10月15日 09:00｜月全勤｜+30｜余额 1220｜9月全勤奖励
「加载更多」追加两条：10月12日 任务打卡 +10（余额 1190）、10月8日 成长日志被推荐 +10（余额 1180），随后按钮变禁用「已经到底了」。

排行榜（本周，前 5 行即可）：1 李小花 60；2 张小强 50；3 乐乐 45（is_self，标「我」）；4 陈小雨 40；5 赵朵朵 35。头部写「我的本周积分 45」，底部写「仅显示本班前 20 名（共 28 人）」。

成就（earned_count 8 / total_count 12，网格必须正好 12 枚）：
已获得：全勤之星（连续一个月全勤，10月1日，+10）、小小艺术家（发布 10 条成长记录，10月5日，+10）、阅读小达人（完成 20 次阅读打卡，10月8日，+10）、运动小健将（参加 10 次户外活动，9月28日，+10）、礼貌小天使（主动问好满 30 天，10月12日，+10）、劳动小能手（完成 15 次自理任务，10月15日，+10）、分享之星（和同伴分享 10 次，10月18日，+10）、坚持之星（连续打卡 30 天，10月22日，+10）。
未获得：艺术小天才（参与 10 次美术/手工活动）、故事大王（在班级分享 5 个故事）、小小科学家（完成 8 次自然观察）、时间小管家（连续 21 天按时入园）。

## 视觉方向
「星轨账本」：整体是低对比的冷静账本 + 少量琥珀色星轨。hero 用中性 surface 卡片，绝不做整块橙色渐变；橙色只出现在等级胶囊、进度条、正积分数字、选中态与徽章上（暖色面积 ≤ 10%）。大数字 1245 是视觉锚点，字号大、字重 700、颜色 --fg。徽章是 4 列网格（`repeat(4, minmax(0, 1fr))`），未获得用灰度（--muted / --hint / --surface-warm），**不要彩色蒙版**，已获得用 --accent-wash 圆底 + --accent-deep 图标。所有圆角 ≥ --radius-sm，卡片 --radius-card。

## 视觉与结构硬约束
- `<html lang="zh-CN" data-surface="parent">`；390px 手机视口；单列纵向流；禁止桌面多栏。
- 微信外壳必须用下面这份共享标记，且只通过 `<link rel="stylesheet" href="../../_shared/shell.css">` 引入；**不要复制 shell.css，不要改写状态栏 SVG，styles.css 里不得出现 `.mp-*` 定义**：
```html
<div class="mp-app" data-state="normal">
  <div class="mp-statusbar" aria-hidden="true"><span class="mp-time">9:41</span><span class="mp-status-icons"><svg width="17" height="11" viewBox="0 0 17 11" fill="currentColor"><rect x="0" y="7" width="3" height="4" rx="1"/><rect x="4.5" y="5" width="3" height="6" rx="1"/><rect x="9" y="2.5" width="3" height="8.5" rx="1"/><rect x="13.5" y="0" width="3" height="11" rx="1"/></svg><svg width="15" height="11" viewBox="0 0 15 11" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round"><path d="M1 3.6C2.8 2 5.3 1 7.5 1s4.7 1 6.5 2.6"/><path d="M3.5 6.1c1.1-1 2.4-1.5 4-1.5s2.9.5 4 1.5"/><path d="M6 8.6c.5-.4 1-.6 1.5-.6s1 .2 1.5.6"/><circle cx="7.5" cy="10" r=".7" fill="currentColor"/></svg><svg width="24" height="11" viewBox="0 0 24 11" fill="none"><rect x=".5" y=".5" width="20" height="10" rx="3" stroke="currentColor" stroke-opacity=".5"/><rect x="2" y="2" width="15" height="7" rx="1.8" fill="currentColor"/><path d="M22 3.5v4c.9-.3 1.4-1 1.4-2s-.5-1.7-1.4-2Z" fill="currentColor" fill-opacity=".5"/></svg></span></div>
  <header class="mp-navbar">
    <button class="mp-back" type="button" aria-label="返回我的"><svg width="20" height="20" viewBox="0 0 20 20" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M12.5 4.5 7 10l5.5 5.5"/></svg></button>
    <h1 class="mp-title">积分与成就</h1>
    <div class="mp-capsule" aria-hidden="true"><span class="mp-capsule-dots">···</span><span class="mp-capsule-divider"></span><span class="mp-capsule-ring">◎</span></div>
  </header>
  <main class="mp-main">…</main>
  <div class="mp-demo" data-open="false"><div class="mp-demo-chips" role="group" aria-label="原型状态"><button class="mp-demo-chip" data-mp-state="normal" aria-pressed="true">正常</button><button class="mp-demo-chip" data-mp-state="empty" aria-pressed="false">空态</button><button class="mp-demo-chip" data-mp-state="loading" aria-pressed="false">加载</button><button class="mp-demo-chip" data-mp-state="error" aria-pressed="false">失败</button></div><button class="mp-demo-toggle" type="button" aria-expanded="false">原型演示</button></div>
</div>
<script src="../../_shared/shell.js"></script>
```
- 子页必须有返回箭头；无 tabBar。
- 颜色只写 var(--*)，禁止裸 hex/rgb（含 SVG fill/stroke 也只用 currentColor 或 var()）；阴影只用 var(--shadow-*)；禁止 border-radius: 0。
- 禁止 `white-space: nowrap`；可省略文本用 text-overflow: ellipsis + overflow hidden。
- 所有容器 max-width: 100%；多列统计用 `grid-template-columns: repeat(N, minmax(0, 1fr))`；页面不得横向滚动。
- 触控目标 ≥ 44px；正文 ≥ 13px；对比度满足 WCAG AA。
- 点击反馈只用 :active 缩放/压暗；不用 hover 才出现的信息；不用 alert / prompt。
- 数据区四态：正常（默认）/ 空 / 加载骨架 2–3 行 / 失败（一句话 + 重试，1 秒后恢复）。空态按当前 tab 给不同文案：积分「还没有积分记录」+「完成任务、坚持出勤都会获得积分」+按钮「查看亲子任务」；排行榜「班级还没有排名」；成就「还没有获得徽章」+「老师授予徽章后会出现在这里」+按钮「查看成长记录」。空/加载/失败用 `[data-state]` 属性选择器切换。
- 交互（app.js 必须真实可点）：三段 tab 切换；积分行点击 toast「已打开积分详情」；徽章点击弹出微信风格底部面板（图标、名称、描述、获得时间或「尚未获得」、+10 积分），点遮罩/关闭按钮收起；「加载更多」追加 2 条后变禁用「已经到底了」；排行榜行点击 toast；失败态重试；底部安全区 `padding-bottom: calc(24px + env(safe-area-inset-bottom))`。
- 演示状态条由 shell.js 负责，页面不要重复实现。

## 输出文件（项目根目录）
1. index.html —— 结构（链接顺序：tokens.css → ../../_shared/shell.css → styles.css）
2. styles.css —— 页面内容区样式（不得出现 .mp-* 定义）
3. app.js —— 交互
tokens.css 已由流程提供，不要重写、不要覆盖。每个文件 < 20KB。不要引用任何外部资源、不要用图片文件。
