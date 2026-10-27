# parent/02-login · 对抗性单页审查

> 2026-10-27 token 对比度已统一修复（白字压橙 5.18–7.31:1），见 `_shared/REVIEW.md` §3.2。

- verdict：**WARN**（score 78/100）
- 源码覆盖：源码可见结构（品牌区、登录卡内错误条/主按钮/协议行、卡外脚注）全部覆盖，normal/empty/loading/error 四态均可切换；未表现 onLoad 已登录直达与成功后 reLaunch 首页两个离开动作（按页面 brief 只保留成功轻提示），错误文案覆盖通用与断网两档，未覆盖 toLoginError 的 err.message 兜底。

## P1 · 明显缺陷（2）

### 1. [design-core] 主按钮白字压品牌橙渐变，实测对比度仅 2.26–2.80:1；15px/600 属正文级，低于 WCAG AA 4.5:1，本页唯一主操作的可读性不达标。

- 证据：tokens.css:18 定义 --on-brand:#FFFFFF；tokens.css:24 定义 --accent-gradient: linear-gradient(135deg,#FB923C 0%,#F97316 100%)；styles.css:178-182 按钮用该渐变 + --on-brand + 15px/600；notes.md:72 自述「2.80:1（浅端 2.26:1），不达 AA」。
- 建议：在 token 层给品牌橙配深色 --on-brand（或把渐变整体压深到 ≥4.5:1），并同步 teacher 端；不要逐页改按钮颜色。

### 2. [redundancy] 失败态同时展示常驻错误条和内容等价的 danger toast，同一失败原因在一屏重复两遍；源码与页面 brief 都明确只要常驻错误条，且 role=alert 与 role=status 会重复播报。

- 证据：index.html:63-75 错误条 role=alert aria-live=assertive；app.js:154-158 失败 chip 同时 setErrorCopy 与 showToast('微信登录失败，请稍后重试','danger')，app.js:111-115 真实断网失败同样双写；login.vue:37-39 注释「不再用 toast…改渲染成卡片内常驻错误条」；_prompt.md:16 同义；teacher/02-login/notes.md:66 老师端只保留常驻错误条；notes.md:65 先说「失败原因只留在卡片内」又说「额外给一条 danger toast」。实测失败态：error bar display:flex + toast is-on/data-tone=danger/同文案。
- 建议：删除失败路径的 danger toast（app.js:114、:158），只保留常驻错误条；若需即时反馈，用不重复失败原因的短提示，并避免 role=alert 与 role=status 同时播报。

## P2 · 打磨项（5）

### 1. [logic] 失败态下取消勾选协议，页面切到 empty 并隐藏错误条，但上一条 danger toast 仍在显示「微信登录失败，请稍后重试」，两个互斥状态叠加。

- 证据：app.js:168-177 change 处理器只隐藏 data-tone=warning 的 toast；实测「失败 chip → 取消勾选」后 data-state=empty、.error-bar display:none、toast is-on=true、data-tone=danger、文案仍为「微信登录失败，请稍后重试」。
- 建议：在 change 处理器里无条件 hideToast(true) 再切状态，或仅在 state==='error' 时保留 danger toast；补一条「错误态→取消勾选」的状态机断言。

### 2. [design-core] 主按钮字重 600、字距 0.04em，与页面 brief 的 15px/700、源码 AppButton 的 700/0.02em、teacher 登录页的 700/0.02em 不一致，两端同一 CTA 粗细不同。

- 证据：styles.css:182 font-weight: var(--weight-semibold)（tokens.css:49 = 600）、styles.css:184 letter-spacing: 0.04em；_prompt.md:40「按钮 15px/700」；AppButton.vue:34-35 font-weight:700; letter-spacing:0.02em；teacher/02-login/styles.css:170-172 同为 --weight-bold/0.02em。
- 建议：改为 font-weight: var(--weight-bold) + letter-spacing: 0.02em，与源码和 teacher 对齐；若坚持 600，需在 notes 标为设计提案并跨端统一。

### 3. [notes-accuracy] notes 声称元数据要求「本页由 navigateTo 进入的子页，画返回箭头」并要批次 owner 修元数据；但批次规则和本页 manifest prompt 都明确登录页是入口页、不画返回箭头，该冲突不存在。

- 证据：notes.md:71；_prompt.md:21,37「没有返回箭头」「不画返回箭头（入口页）」；parent/02-login/.file-versions/0eb547304658805aad788d32/manifest.json:13 prompt 原文「不画返回箭头、不画 tabBar（入口页）」；_shared/shell.md:96「入口页（老师工作台、家长首页、登录页）不画」；_shared/subagent-task.md:47 同。
- 建议：把 notes 第 71 条改为「已核对：入口页不画返回箭头，批次规则与 manifest prompt 一致」，删掉「需批次 owner 修元数据」的错误待办。

### 4. [notes-accuracy] notes 对错误条文字颜色的描述前后矛盾：第 13 条说整段文字用 color-mix 且只有 4.69:1，第 53 条说标题用 --fg；实际标题用 color-mix、说明用 --fg-2，而源码两者都用前景色。

- 证据：notes.md:13「文字用 color-mix(--danger 72%, --fg)…实测 4.69:1」；notes.md:53 把「错误标题」映射到 --fg；styles.css:147 标题用 color-mix，styles.css:157 .error-desc 用 var(--fg-2)；login.vue:122-131 源码 .error-title/.error-detail 均为 $color-text-primary。
- 建议：把 notes 改成实际值（标题 mix 4.69:1 / 说明 --fg-2），或把标题改回 --fg 以贴合源码；不要保留三处互相矛盾的说法。

### 5. [logic] 演示面板的关闭逻辑调用不存在的 window.OdShell.closeDemo；_shared/shell.js 没有 OdShell，选完 chip 后面板仍展开、焦点被抢到「原型演示」按钮，错误态 toast 入场时还会短暂压住 chips。

- 证据：app.js:124-136 与 :163 调用 window.OdShell.closeDemo；_shared/shell.js 全文无 OdShell；实测 typeof window.OdShell==='undefined'；打开演示条后点失败 chip：data-open=true、aria-expanded=true、document.activeElement.className='mp-demo-toggle'；toast 入场起始 bottom=772，chips top=764（短暂重叠 8px，终态 764 相切）。
- 建议：在页面内直接切换 demoPanel.dataset.open 与 aria-expanded，或把 closeDemo 补进 _shared/shell.js；不要用 focus() 抢焦点；若面板保持展开，把 toast 的 bottom 抬到 chips 之上。

## 2026-10-27 P1 修复记录

| # | 结论 | 改动 / 理由 |
|---|---|---|
| P1-1 主按钮白字压橙对比度 2.26–2.80:1 | **暂缓（批次级）** | 需回 `shared-design-tokens/tokens.json` 给品牌橙配深色 `--on-brand`（或整体压深渐变）并同步 teacher 端；审查建议明确「不要逐页改按钮颜色」，本页目录内无法自洽解决，登记于 notes.md「遗留」第 2 条等批次统一处理。 |
| P1-2 失败态错误条与 danger toast 重复 | **已修** | 删除 `app.js` 两条失败路径的 danger toast（断网分支、失败芯片），失败态只保留卡片内常驻错误条；`role=alert` 不再与 `role=status` 重复播报（P2-1 的 toast 滞留场景随之消失，但 P2 未纳入本次范围）。 |

**验证**：`od-check --width 390` → `problems: []`；`od-shot --width 390 --dsf 2` → `preview.png` 重出（CSS 390×1000）；`od-audit` → 批次 50/50 通过、本页无 ✗。CDP 交互探针 10 条断言全过：失败芯片与断网失败下 `data-state=error`、`.error-bar` display:flex、toast `hidden=true`/`is-on=false`；成功路径仍正常弹「登录成功」；错误态取消勾选后无残留 toast；console 0 error。notes.md 的覆盖状态表与「与现状的差异」已同步删除 danger toast 描述。
