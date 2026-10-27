# parent/10-growth-detail · 对抗性单页审查

- verdict：**WARN**（score 75/100）
- 源码覆盖：源码区块与分支基本全覆盖（作者/当日状态/正文/标签/照片筛选与空态/互动/评论与子回复/底部输入栏/加载/失败/空态），仅「相关孩子」按条件不出现、照片筛选空态因示例数据不可达，二者 notes 已说明。

## P1 · 明显缺陷（2）

### 1. [navigation] 返回箭头是看起来可点的按钮，但没有绑定任何 handler，点击后页面与 URL 均不变；同批次 8 个家长端子页都接了返回 toast，_prompt 也要求「导航与返回箭头保持可用」。

- 证据：index.html:23-25 定义 .mp-back（aria-label=返回）；app.js 全文无 .mp-back/返回 处理，实测点击 URL 不变；_prompt.md:60；对照 05-notice-detail/app.js:53-57 等 8 页均有返回兜底。
- 建议：给 .mp-back 绑定返回逻辑：独立预览用 history.back()，或 toast「返回成长时光」；加载/失败态保持可用。

### 2. [logic] 照片预览弹层显示「第 N 张 / 共 M 张」，但弹层内只有关闭按钮，没有上一张/下一张或滑动手势；用户无法在弹层内查看其余照片，计数器成了假承诺。源码 uni.previewImage 支持左右滑，这是照片优先页的核心交互。

- 证据：index.html:253-266 弹层内唯一按钮 #closePreview；app.js:71-82 openPreview 只替换当前张并写计数，未监听 ArrowLeft/Right；detail.vue:57-59 传全量 urls+current；notes.md:65 自认不支持切图。
- 建议：补左右箭头与触摸滑动（或至少「上一张/下一张」按钮），计数器随当前 index 更新；若不打算做切图，删掉计数文案。

## P2 · 打磨项（5）

### 1. [logic] 新评论用 prepend 插到列表顶部，但列表按时间升序（10:26→12:30），「刚刚」会排在 10:26 之前且落在视口外；源码重新拉取评论（后端 created_at ASC）时新评论应在末尾。notes 却写成「本地追加」。

- 证据：app.js:172 dom.commentsList.prepend(comment)；index.html:140/160/170/179 时间递增；parent_portal_service.go:1418 Order("created_at ASC")；notes.md:53；_prompt.md:123 要求插到顶部。
- 建议：改为 append 并 scrollIntoView，与源码升序一致；若要置顶，整页评论改倒序并同步示例数据。

### 2. [redundancy] 孩子的当日状态被并进作者卡：section 的 aria-labelledby 指向「李老师」，屏幕阅读器会把早餐/午餐/午睡/情绪读成李老师的区块；视觉上 16px/600 的「好/90分钟」与作者名同级、压过 15px 正文，次要状态抢主视觉。源码将状态标签独立成段。

- 证据：index.html:37 section aria-labelledby="authorName"，41-54 作者名与 day-status 同卡；styles.css .day-value{font-size:var(--text-xl);font-weight:600} 与 .author-name 同为 16px，.record-copy 15px；detail.vue:131-137 独立 status-tags。
- 建议：把当日状态拆回独立卡；或 section 改 aria-label="记录概况" 并把 day-value 降到 14px/500，作者卡只承载身份。

### 3. [logic] 时间口径不自洽：记录发布于 10:05，却展示「午睡 90分钟」（上午 10 点不应有午睡）；同时源码把后端 UTC 的 created_at 直接 slice(11,16)，中国时区会显示早 8 小时——按本地读是数据矛盾，按 UTC 读是时钟错误。

- 证据：index.html:42（10月22日 10:05）与 52（午睡 90分钟）；detail.vue:120 created_at?.slice(11,16)；parent_portal_service.go:1300 CreatedAt.UTC().Format(RFC3339)；_prompt.md:35-37 固定了这组样例。
- 建议：后端返回带时区时间、前端本地化格式化；示例时间改为与状态行一致的下午时间（如 18:05），避免发布时间与「当日状态」混读。

### 4. [design-core] 表现标签 11px 文字用 --accent-deep (#EA580C) 压 --accent-wash (#FFF7ED)，实测对比度 3.35:1，低于 WCAG AA 4.5:1；标签承载正文信息，不是装饰。

- 证据：styles.css .tag{color:var(--accent-deep);background:var(--accent-wash);font-size:var(--text-xs)}；tokens.css:22-23、44（--text-xs=11px）；headless Chrome 计算样式实测 3.35:1。
- 建议：文字改用 --fg 或加深 accent；或提高字号到 ≥13px 后复测至 ≥4.5:1。

### 5. [notes-accuracy] notes 与实际实现多处不符：① #8 称「原型严格照此，没有发明」，但子回复也带「回复」按钮并支持对回复再回复，源码子回复只有昵称+正文；② 第 53 行称「本地追加」，代码是 prepend；③ 第 15 行写「乐乐的妈妈」，UI 是「乐乐妈妈」。

- 证据：notes.md:64 vs index.html:151 + app.js:177-201；detail.vue:209-213 子回复仅 reply-user/reply-content；notes.md:53 vs app.js:172；notes.md:15 vs index.html:138。
- 建议：按代码更新 notes（或删掉子回复的回复入口并标注为提案）；统一昵称写法。

## 2026-10-27 P1 修复记录

| # | 结论 | 改动 |
| --- | --- | --- |
| P1-1 navigation | 已修 | `.mp-back` 绑定点击 → toast「返回成长时光」（原型不真跳页，与同批次 8 个家长端子页一致）；不做状态门禁，加载/失败态下同样可用，已实测两态均弹 toast。 |
| P1-2 logic | 已修 | 照片预览弹层补「上一张 / 下一张」按钮 + ←/→ 键 + 左右滑动，`previewList`/`previewIndex` 驱动 `第 N 张 / 共 M 张` 与画面同步；首尾张对应按钮禁用，单张时隐藏翻页行；焦点陷阱改为在关闭/翻页键之间循环，计数器加 `aria-live="polite"`。 |

验证：`od-check` problems `[]`；`od-shot` 390×1434 @2x 重出；`od-audit` 50 页全部通过、本页无 ✗。CDP 实测：返回 toast（正常/加载/失败三态）、2 张默认筛选下 1→2→1 切换与禁用边界、切「看全部」后第 5/6 张、Tab 焦点循环、Esc/遮罩关闭、body 滚动解锁，均通过。notes.md 已同步预览弹层与返回行为说明。
