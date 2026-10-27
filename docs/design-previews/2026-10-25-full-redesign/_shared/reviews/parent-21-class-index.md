# parent/21-class-index · 对抗性单页审查

- verdict：**PASS**（score 87/100）
- 源码覆盖：源码的班级名/年级/学年/人数与老师姓名/职务/电话、加载/失败/正常三态均已覆盖；空态、班级相关入口与页脚为按 brief 增补，未发现源码字段或可点入口缺失。

## P2 · 打磨项（7）

### 1. [logic] 失败态把所有错误原因写成「网络不太稳定」，但源码透出的是 err.message，也可能是未绑定孩子或登录失效；文案会误导，重试也不会解决。

- 证据：parent/src/pages/class/index.vue:19-22 catch 里 loadError.value = err?.message || "班级信息加载失败"；server/internal/service/parent_portal_service.go:2906-2908 在 pc.Class == nil 时返回 ErrParentInvalidParams；21-class-index/index.html:186-188 固定「班级信息加载失败 / 网络不太稳定，请稍后再试 / 重试」（_prompt.md:46 指定）。
- 建议：失败态优先保留服务端 message（或按错误码映射）；未绑定/登录失效给「去绑定/重新登录」，只有网络错误才提示网络；同步改 _prompt.md:46 的固定文案。

### 2. [redundancy] 「班级相关」两个入口是在班级信息页重复已存在的全局入口：通知公告已在首页快捷入口与「我的」菜单，成长相册已由成长首页「导出相册」承载；放在「班级相关」下还会让家长误以为是班级范围的内容。

- 证据：21-class-index/index.html:143-173（成长相册/通知公告两行）；parent/01-home/index.html:120-125（通知公告快捷入口）；parent/27-profile/index.html:237-239（通知公告菜单）；parent/09-growth-index/app.js:179（.js-export toast「已打开成长相册」）；21-class-index/notes.md:59,67 自认是调度新增、需产品确认；A27 已记录入口为新增，本条补充其与既有入口重复的问题。
- 建议：删掉这张卡，或换成真正的班级范围内容（按 class_id 过滤的班级通知/班级相册）；若保留为快捷方式，改标题为「快捷入口」并避免与全局入口同名；产品确认。

### 3. [logic] 「2025-2026 学年」与批次时间线（2026-10）不一致；在 2026 年 10 月，这个班应处于 2026-2027 学年，页面看起来停在上一学年。

- 证据：21-class-index/index.html:45「2025-2026 学年」；parent/01-home/index.html:52「2026年10月26日 星期一」；parent/09-growth-index/index.html:138-139 有 2026-10 记录；parent/21-class-index/_prompt.md:27 指定「2025-2026」。
- 建议：批次示例数据统一改为 2026-2027（或省略学年），同时改 _prompt.md:27；避免只改单页造成新一轮漂移。

### 4. [redundancy] 空态用标题、说明、按钮把同一事实说了四遍（暂无/尚未配置/请稍后重新加载/重新加载），且「尚未配置」不是重载能解决的临时状态；真实数据形态里空态等于老师列表为空，重载仍会返回空。

- 证据：21-class-index/index.html:112-114；21-class-index/notes.md:23,69（空态=老师列表为空）；_prompt.md:44 指定了这段文案与按钮。
- 建议：空态只留一句「班级老师尚未配置，请联系园所」+ 一个「联系园所/刷新」动作；若要保留重载，说明「刷新看看」而不是承诺恢复；批次统一改 _prompt.md。

### 5. [design-core] loading 态把 #class-name 所在的 .identity-real 隐藏，但 .identity-card 仍用 aria-labelledby="class-name" 命名；两个骨架容器又都是 aria-hidden="true"，读屏用户只得到 aria-busy，班级卡变成无名区域且没有「正在加载」可读文本。

- 证据：21-class-index/index.html:36、:41、:55、:117；21-class-index/styles.css:597-603（[data-state="loading"] .identity-real { display:none }）。
- 建议：给 .identity-card 固定 aria-label="班级信息"（或保留视觉隐藏的标题），并加 role="status" 的视觉隐藏「正在加载班级信息」；骨架继续 aria-hidden。

### 6. [design-core] 间距/内边距有多处裸 px，且落在 token 尺度外（24/20/10/2px）；其中 6px 明明有 --space-sm 仍写死，和 notes 声称的「间距走 --space-*」不符，落地会把这些值带进源码。

- 证据：21-class-index/styles.css:175,179（margin-top:24px）、:371（margin:20px auto 0）、:415,588（margin:6px 0 0）、:647（padding:10px var(--space-xl)）、:159,166（2px）；tokens.css:61-64（--space-sm:6px 至 --space-xl:16px，无 20/24/10）；notes.md:52 声称间距用 --space-*。
- 建议：6px 改为 var(--space-sm)；20/24px 用 calc(var(--space-xl) * 1.25/1.5) 或新增 --space-2xl；10px 用 var(--space-md)/var(--space-lg)；2px 用 var(--space-xs) 或 0；改完同步 notes。

### 7. [redundancy] 56px 徽记「小一」与紧邻的班级名「小一班」重复同一名称，没有提供额外信息，还和 20px 班级名争主视觉；notes 自己也承认可换成线稿。

- 证据：21-class-index/index.html:39（<span class="class-emblem" aria-hidden="true">小一</span>）、:41（<h2 class="class-name" id="class-name">小一班</h2>）；notes.md:73；_prompt.md:37 允许「小一」或幼儿园/人群线稿。
- 建议：改用 _prompt.md:37 允许的幼儿园/人群线稿或抽象图形，保留「小一班」作为唯一名称来源；徽记仍可保留暖底。
