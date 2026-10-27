# teacher/16-album-index · 对抗性单页审查

- verdict：**WARN**（score 70/100）
- 源码覆盖：源码的统计卡、分类筛选、加载/空分支、相册卡四字段（标题/张数/日期/分类）与两处占位 toast 均有对应实现，四态与筛选实测可用；缺下拉刷新（onPullDownRefresh）意象，整卡点击被拆成「封面开大图 + 信息条看相册」两条路径，封面由源码 1 大 2 小改为三等分。

## P1 · 明显缺陷（2）

### 1. [redundancy] 同一页两组册数口径不一致：统计卡写「相册数 12」，列表两个月份组只有 4+2=6 册，且全页没有任何「加载更多/查看全部」入口或「已展示 6/共 12 册」的说明，用户无法触达另外 6 册。

- 证据：index.html:82 相册数 12；index.html:98「4 册」、index.html:156「2 册」；DOM 实测 .album-card 共 6 张、月份计数为 ["4 册","2 册"]；notes.md:41/50 自认「列表只展示 12 册中的最近 6 册」「之外的 6 册没有入口」。
- 建议：列表尾部补「查看全部 12 册 / 加载更多」入口并让月份组覆盖全部相册；或把统计改为「已展示 6 / 共 12 册」，避免同页两个总数打架。

### 2. [logic] 大图弹层打开时背景没有真正锁定滚动：app.js 加的 body.is-locked 在本页 CSS 里没有任何规则（同批次 parent/11-growth-album/styles.css:39 有 body.is-locked{overflow:hidden}，本页没有），同时 app.js:49 的 main.scrollTop=0 对一个非滚动容器空转，所以弹层开着时页面仍可滚动，关掉后页面停在漂移后的位置。

- 证据：app.js:146/161 add/remove('is-locked')；styles.css 全文无 is-locked 规则（DOM 内所有 styleSheet 扫描结果为无匹配）；实测弹层打开后 body 计算样式 overflow=visible，window.scrollTo(0,400) 后 scrollY=400；.mp-main 的 scrollHeight===clientHeight（1562），scrollTop 恒为 0。notes.md:23 称「大图弹层开/关…全部通过」但未覆盖背景滚动。
- 建议：在 styles.css 补 body.is-locked{overflow:hidden}（或改用 .mp-main 滚动容器并锁它），状态切换时只保留 window.scrollTo(0,0)，删掉无效的 main.scrollTop。

## P2 · 打磨项（6）

### 1. [completeness] 空态把源码中恒常渲染的统计卡与分类筛选一并隐藏。源码只有列表区有 v-if，统计卡和分类胶囊在空列表时仍然显示；notes 只在「覆盖状态」里写「隐藏统计与列表」，没在「与现状的差异」里说明这是有意改结构，也没点出隐藏是为了回避源码里「386 张照片 + 暂无相册」的自相矛盾。

- 证据：index.vue:63-95 统计卡、index.vue:98-110 分类筛选均在列表 v-if 之外；styles.css:279-281 [data-state="empty"] .album-normal{display:none}；DOM 实测 empty 态下 normalDisplay=none；notes.md:19。
- 建议：二选一并写清：空态保留统计卡与筛选（贴源码），或在 notes 差异里注明「空态隐藏统计以回避源码硬编码 386/12 与空列表的冲突」，顺带记录源码统计是写死的假数。

### 2. [redundancy] 空态同屏出现两个「上传照片」按钮，一个是空态区内的浅色按钮、一个是底部渐变大主按钮，二者点击结果完全相同（同一句 toast），主行动被拆成两个视觉权重不同的入口。

- 证据：index.html:215 空态 .btn-soft；index.html:226-230 底部 .album-actionbar 的 .btn-upload；DOM 实测两个 [data-action="upload"] 同时可见（y=358-406 与 y=788-836）；notes.md:19「底部上传按钮保留」未说明为何需要两个。
- 建议：空态只保留一个 CTA：要么空态按钮承担主行动、同时隐藏底部栏，要么空态只留说明文字、由底部栏统一承担。

### 3. [completeness] 封面结构从源码的「1 张大图 + 右侧 2 张小图」改成「3 张等分正方形」，notes 的「与现状的差异」没有记录这次结构改动；而且三格共用同一个 ph-* 渐变类，正文声称的「拼出多张照片的体感」实际读不出多图，更像同一张图复制三份。

- 证据：index.vue:127-159 源码为 cover-main(flex:1) + cover-side(180rpx 两行缩略图)；styles.css:168-173 改为 repeat(3,1fr) 等分、styles.css:187-189 只按 nth-child 改 background-size/position；index.html:104-106 三格同为 ph-4（其余卡片同样同色）；notes.md:5 与 notes.md:36-43 差异清单。
- 建议：在「与现状的差异」补记封面结构变更；若要表达多图，三格用不同色调/裁切或加「+9」角标，或恢复源码 1 大 2 小。

### 4. [completeness] 空态文案「上传后家长可在成长时光里看到」断言了一条源码和契约都不存在的跨端可见性：老师端相册接口尚未提供，家长端「成长时光/相册」读的是成长模块的 /parent/growth/album，与老师端相册不是同一数据源。

- 证据：index.html:214；index.vue:43-54 loadAlbums 恒置空、无接口；notes.md:47「老师端相册 API 尚未提供」；parent/src/api/parent-growth.js:52-54 家长端相册走 /parent/growth/album；backend/api/api-teacher/ 下没有任何相册接口。
- 建议：改成不承诺下游的数据描述（如「上传后可在本页按月份查看」）；若产品确实要求家长端可见，先在 backend/api 定义老师端相册接口并确认与成长时光的关联。

### 5. [design-core] 相册信息按钮的 aria-label 覆盖了按钮内全部可见文本，读屏用户只知道「查看相册 秋日郊游」，听不到卡片上唯一的「12张 / 10月24日 / 活动」三项信息（这三项在页面其他位置都没有）。

- 证据：index.html:108 aria-label="查看相册 秋日郊游" 包裹 109-110 的 12张/10月24日/活动；DOM 实测可访问名为 label 本身、textContent 为「秋日郊游12张 10月24日活动」；六张卡同样写法。
- 建议：去掉 aria-label，让可见文本构成可访问名；或把张数/日期/分类并入 label，例如「秋日郊游，12张，10月24日，活动」。

### 6. [design-core] 页面主按钮「＋ 上传照片」是白字压亮橙渐变，实测对比度仅 1.90–2.35:1，远低于 WCAG AA 对 15px/600 文字的 4.5:1；这是全页最醒目的控件，也是老师端共用 token 的问题，不应逐页打补丁。

- 证据：styles.css:427-428 background:var(--accent-gradient)/color:var(--on-brand)；tokens.css:18 --on-brand:#FFFFFF，tokens.css:89 教师端 --accent-gradient = #ffa866→#ff8a3d；DOM 实测 color rgb(255,255,255)、font-size 15px、font-weight 600，对 #ff8a3d 对比度 2.35:1（浅端约 1.90:1）。
- 建议：回 shared-design-tokens/tokens.json 调整：改用深色文字（--fg / 深橙）或加深渐变到 --accent-deep 量级，保证 ≥4.5:1；改完 npm run build 并同步三端产物。

## 2026-10-27 P1 修复记录

1. [redundancy] 同页册数口径不一致、另外 6 册不可达 —— **已修**：补入「2026年8月（3 册）」「2026年7月（3 册）」两个月份分组（默认 `hidden` 折叠），列表尾部新增「已展示最近 6 册 · 共 12 册」+「查看全部 12 册」按钮，点击原位展开并隐藏入口；分类筛选跨全部 12 册（先展开再算册数，避免筛选结果被折叠吞掉）。现在 4+2+3+3 = 12 册 = 统计卡「相册数」，12 册照片数合计 386 = 「照片总数」，「本月新增 42」仍等于 10 月 4 册之和。
2. [logic] 大图弹层打开时背景未真正锁滚 —— **已修**：`styles.css` 补 `body.is-locked { overflow: hidden }`（本页滚动容器就是 body），`app.js` 删除对非滚动容器 `.mp-main` 的 `scrollTop = 0` 空转，状态切换只保留 `window.scrollTo(0, 0)`；实测弹层打开时 CDP 真实滚轮输入不再移动页面，关闭后恢复。

验证：`od-check --width 390` → `problems: []`；`od-shot` 重出 preview.png（390×dsf2，1711px）；`od-audit` 批次 50/50 页无 ✗（本页仅「index.html 26KB / styles.css 20KB 超 20KB」体积提示，按规则不计失败）。P2 六条未动。
