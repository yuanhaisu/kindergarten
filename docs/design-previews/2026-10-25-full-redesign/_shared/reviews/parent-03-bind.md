# parent/03-bind · 对抗性单页审查

- verdict：**WARN**（score 74/100）
- 源码覆盖：源码的邀请码单字段、确认绑定、扫码即自动提交、成功 toast+1 秒返回、失败保留输入均已覆盖，并新增计数/最小 6 位/说明卡/扫码层等已标注的设计提案；三条后端失败文案只画了「未找到该学生」一条，「已绑定或邀请码无效」「绑定学生失败」缺失。

## P1 · 明显缺陷（3）

### 1. [redundancy] 导航栏标题与 hero 大标题都写「绑定孩子」，两个标题在首屏相距 59px 同时可见，重复同一信息。

- 证据：index.html:28 `<h1 class="mp-title">绑定孩子</h1>`，index.html:40 `<h2 class="hero-title" id="pageTitle">绑定孩子</h2>`；实测 nav 标题 top=41、字号 15px，hero 标题 top=100、字号 30px，文本完全相同。同批次 02-login、07-leave-create、24-feedback、25-settings 的 mp-title 均未在内容区重复。
- 建议：hero 改成任务型标题（如「输入邀请码」/「认领孩子」），或删除 hero 标题，把「绑定孩子」只保留在导航栏。

### 2. [logic] 输入框在第 1 到 5 个字符之间就进入 error 态：红边、红色错误行、aria-invalid=true，并露出会清空输入的「重新填写」，用户还没输完就被判失败。

- 证据：app.js:271-274：length===0→empty，0<length<6→state='error'+COPY.short；app.js:96 `refillBtn.hidden = state !== 'error'`。实测输入 1 个字符 'K' 后：state=error、fieldReportText=「邀请码至少 6 位」、input.is-error=true、aria-invalid=true、refillVisible=true、submitDisabled=true。
- 建议：打字过程中保持 neutral（或只显示「还差 N 位」），把「至少 6 位」与「重新填写」延到 blur 或首次提交失败后再出现。

### 3. [navigation] `pages/auth/bind` 在家长端没有任何入站入口，App 内无法到达该页；原型集里也没有任何页面链接到 03-bind。

- 证据：parent/src/pages.json:17 只注册 path，parent/src 内无 navigateTo/redirectTo/reLaunch 指向该页；`node scripts/_nav-audit.mjs` 输出 /pages/auth/bind inbound=0（与 A38 webview、A39 messages 同为零入站）；parent/src/App.vue:5-9 无 scene/query 路由；原型集 grep 不到 03-bind 链接。
- 建议：在 23-student-index 或 27-profile 增加「绑定新孩子」入口指向 pages/auth/bind；若实际靠外部小程序码/场景值进入，在 notes.md 声明入口与参数。

## P2 · 打磨项（5）

### 1. [logic] 扫码层给 body 加了 `is-locked`，但本页与共享 shell 都没有对应 CSS，背景仍可滚动。

- 证据：app.js:287 添加、app.js:175 移除；03-bind/styles.css 与 _shared/shell.css 均无 `.is-locked` 规则（仅 11-growth-album/styles.css:39 有 `body.is-locked { overflow: hidden; }`）。实测扫码打开时 getComputedStyle(document.body).overflowY='visible'，页面 scrollHeight=1024、innerHeight=844，可滚 180px。
- 建议：补 `body.is-locked { overflow: hidden; }`（建议落到 _shared/shell.css），并验证打开/关闭成对恢复。

### 2. [logic] 扫码层声明 aria-modal=true 但不是真模态：背景未 inert/aria-hidden，焦点能移到背景输入框；遮罩是 .mp-app 内 absolute，不是视口固定。

- 证据：index.html:187 `role="dialog" aria-modal="true"`；app.js:288 只移焦到 scanClose，未改背景 inert/aria-hidden。实测扫码打开时执行 inviteCode.focus() 后 activeElement=inviteCode，.mp-main 的 aria-hidden=null、inert=false。styles.css:588-589 `.scan-screen { position:absolute }`；滚到底时遮罩底与视口底仍露出约 8px 页面底色。
- 建议：遮罩改 `position: fixed`；打开时给背景加 inert/aria-hidden，关闭时恢复；焦点在关闭后回到扫码按钮。

### 3. [redundancy] 「邀请码由班级老师提供」从首屏到页脚重复 5 次，说明卡第 01 条正文几乎是字段 helper 的逐字复制。

- 证据：index.html:41「输入老师提供的邀请码…」、:95「邀请码由班级老师提供，每位孩子一个」、:146「孩子在园里的学号即邀请码，由班级老师提供，每位孩子一个。」、:160「联系班级老师核对邀请码」、:209「邀请码二维码由班级老师提供」；实测 guide 01 文本与 helper 只差「孩子在园里的学号即邀请码，」前缀。
- 建议：helper 作为唯一出处；guide 01 只保留「学号即邀请码」新信息；删掉扫码层脚注的重复来源说明。

### 4. [completeness] 失败态只实现「未找到该学生」一条：后端三种失败里「已绑定或邀请码无效」「绑定学生失败」在原型不可达，且「已绑定」时给「重新填写」是错误建议。

- 证据：app.js:14-21 COPY 只有 notFound；app.js:223-225 对 REJECTED_CODE/DEMO_ERROR_CODE 一律 `errorMessage = COPY.notFound`；源码 bind.vue:22-23 透传 `err?.message`；parent_portal.go:1034-1041 三条文案分别为「未找到该学生，请检查邀请码」「已绑定或邀请码无效」「绑定学生失败」；notes.md:56 自认只演示第一条。
- 建议：补第二个错误分支（第二个演示码/第五个 chip 或失败态切换文案），文案由服务端 message 驱动；「已绑定」时不要只给「重新填写」。

### 5. [notes-accuracy] notes.md 的状态表把客户端校验写成「输入 4 位时自动出现」，实现是第 1 个字符就触发。

- 证据：notes.md:26「客户端校验 | 输入 4 位时自动出现」；实测输入 'K'（1 位）即 state=error、fieldReportText=「邀请码至少 6 位」；app.js:271-274 的条件是任意 0<length<6。
- 建议：把 notes 改成「任意 1–5 位时」；若产品确实想 4 位才提示，改触发条件并在 blur/提交时再校验。

## 2026-10-27 P1 修复记录

- **P1-1 导航栏与 hero 标题重复**：已修。hero 标题由「绑定孩子」改为任务型「认领孩子」，导航栏保留「绑定孩子」；`aria-labelledby` 仍指向该 h2，结构未动。
- **P1-2 输入 1–5 位即进 error**：已修。打字阶段保持中性（`data-state` 不再转 error），提示行只显示「还差 N 位」；「邀请码至少 6 位」红边错误延到失焦或提交才出现；「重新填写」仅服务端失败时显示（`errorKind=server`），位数不足时不再出现，避免清空用户正在输入的码。
- **P1-3 pages/auth/bind 无入站入口**：暂缓（批次级）。入口应加在 23-student-index / 27-profile，属其他页面目录，本页无法单独修，故按约束不越界改；已在 `notes.md` 遗留项 7 记录入口位置与建议文案。

P2 五项未动。`notes.md` 的状态表「客户端校验」行与交互细节已同步为新行为。
