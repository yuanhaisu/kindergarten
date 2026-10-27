# parent/09-growth-index · 对抗性单页审查

- verdict：**WARN**（score 78/100）
- 源码覆盖：源码的孩子卡、视角切换、四态、月分组与记录卡字段均已覆盖；缺「孩子视角筛选后为空」的 mode-empty 标题/说明分支（源码另有「看全部班级动态」CTA，原型用顶部切换替代）。

## P1 · 明显缺陷（1）

### 1. [completeness] 缺源码的「孩子视角筛选后为空」状态（mode-empty）的标题与说明：孩子视角把所有记录过滤掉时，原型只剩「时光轴 / 2026年 / 0 条」和空白列表，没有任何解释。

- 证据：parent/src/pages/growth/index.vue:325-331 有独立的 v-else-if='visibleFeeds.length === 0' 空态（标题与说明随视角变化，并带「看全部班级动态」）；本页 index.html:102-116 只有 feeds 全空时的「老师还没有发布成长记录」空态，:365-368 演示条也只有四态；app.js:85-141 的 applyView 只隐藏记录和月份组，不渲染该分支；notes.md:43 只用顶部切换替代 CTA，未覆盖缺失的状态文案。
- 建议：补一个孩子视角专属空态（或在 applyView 中当 recCount=0 且 feeds 非空时显示）：标题「还没有与乐乐相关的成长记录」、说明「可以先看看班级动态，或等老师发布新的记录。」；CTA 可继续由顶部视角切换承担。

## P2 · 打磨项（6）

### 1. [navigation] tabBar 四个按钮全部没有点击处理，点「首页 / 接送 / 我的」无任何反应，属「看起来能点但没反应」。

- 证据：index.html:344-361 的四个 button.mp-tab 无 data-tab、无 onclick；app.js 全文无 .mp-tab 选择器；同批 parent/13-pickup-index/app.js:119-129 对 .mp-tab 绑定了 data-tab 并 toast，13-pickup-index/index.html:221-235 每个 tab 都带 data-tab。
- 建议：照 13 页模式给四个 tab 补 data-tab 与 click 反馈（当前页提示「当前已在成长」，其余提示「已切换到首页/接送/我的」），或统一由 shell.js 处理。

### 2. [logic] 卡片的 keydown 处理没有像 click 那样排除内部按钮，键盘在「点赞 / 评论 / 分享」上按 Enter/Space 会被卡片拦截并提示「已打开成长详情」，点赞等操作无法用键盘触发。

- 证据：app.js:182-192：click 先判断 event.target.closest('button')（184 行），keydown（187-191 行）没有同样判断就 preventDefault + toast；index.html:164-173 的按钮嵌在 article.rec[tabindex='0'] 内，keydown 会冒泡到卡片。
- 建议：keydown 加 if (event.target !== record) return;（或同样判断 closest('button')），保留卡片本身的 Enter/Space 行为。

### 3. [design-core] 两处说明性正文使用 11px 的 --text-xs，低于 rubric「正文 ≥ 13px」，在 390px 视口下阅读吃力。

- 证据：styles.css:232-239（.seg-hint）与 :369-383（.rec-more）均 font-size: var(--text-xs)；tokens.css:39 定义 --text-xs: 11px；对应文案 index.html:93 与 :195 是完整说明句。
- 建议：两处说明文字改用 --text-base（13px，满足正文字号下限），颜色可保持 --muted。

### 4. [redundancy] 孩子卡的「成长记录 6 / 精彩照片 —」与紧邻分段的「只看孩子 5 条 · 8 张 / 看全部 6 条 · 15 张」是总数与本页当前视角两种口径，界面未标注，尤其「—」与「15 张」易被读成自相矛盾。

- 证据：index.html:58-79 孩子卡统计（成长记录 6、精彩照片 —）；:84-91 分段计数；notes.md:22 与 :49 说明前者是 feed.total / 无照片来源、后者只统计已加载页（A8 已记录这两项为设计提案）；页面没有任何「本页 / 全部」限定词。
- 建议：在分段计数前加「本页」限定（如「本页 5 条 · 8 张」），或在孩子卡统计下注明「统计全部记录」；照片统计保持「—」时在 hint 里说明「总数暂不提供」。

### 5. [redundancy] 年份「2026年」在时光轴胶囊和两个月分组标题里重复出现三次，胶囊没有提供新增信息。

- 证据：index.html:132 的 .year-pill 文本「2026年」；:139「2026年10月」；:273「2026年9月」。
- 建议：去掉 year-pill，或让月分组只在跨年边界显示年份（如「10月」，仅第一个月保留「2026年10月」）。

### 6. [navigation] 分享按钮发明了源码没有的分享能力，并把本应冒泡到卡片的点击拦掉；源码的隐含目的地是成长详情，notes 也未把分享标为设计提案。

- 证据：parent/src/pages/growth/index.vue:403-406 分享项没有 @click.stop，父级 :341 是 @click='goDetail(item.id)'；本页 app.js:167-179 对 .js-share 执行 stopPropagation 并 toast「已唤起微信分享」；notes.md 的「与现状的差异」没有分享条目。
- 建议：要么按源码把分享点击落到详情（toast「已打开成长详情」），要么在 notes.md「遗留/需产品确认」里明确这是待定的分享入口及落点。

## 2026-10-27 P1 修复记录

- **P1-1 [completeness] 孩子视角筛选后的空态（mode-empty）** — 已修。index.html 在时光轴内补 `.mode-empty` 提示卡（标题「还没有与乐乐相关的成长记录」，说明「可以先看看班级动态，或等老师发布新的记录。」）；`applyView()` 在「存在记录但当前视角 0 条」时隐藏 `.months`、显示提示卡，同时保留时光轴标题与「0 条」，与源码 `visibleFeeds.length === 0` 分支一致。全局空态仍由 `data-state="empty"` 独占，两者互斥；「看全部」入口继续由顶部视角切换承担，未重复行内按钮（notes.md 已同步）。当前 fixture「只看孩子」恒为 5 条，该分支需数据触发；是否加第五个演示芯片属批次级规范，本页未扩。

