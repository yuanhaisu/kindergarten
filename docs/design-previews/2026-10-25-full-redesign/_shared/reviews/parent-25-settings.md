# parent/25-settings · 对抗性单页审查

- verdict：**WARN**（score 80/100）
- 源码覆盖：源码 settings/index.vue 的 6 个通知开关（顺序、默认值、整行点击）、加载态、退出登录→确认→清鉴权主链路均已覆盖，隐私 2 键按契约补画（A41），空态/骨架/错误态属源码没有的设计提案，入口「我的→设置 navigateTo」与子页返回箭头、无 tabBar 也一致；未覆盖的源码分支只有开关保存失败时的「更新失败」提示。

## P1 · 明显缺陷（1）

### 1. [logic] 加载态骨架块只有 188px 高且不占满内容区，「退出登录」在首屏加载时被顶到页面中部（y=288），下方留白 500px；同一按钮在 normal/empty/loading 三态分别落在 y≈673/764/288，切换时跳动约 385px，且源码在 loading 期间根本不渲染该按钮。

- 证据：styles.css:379-383 的 `[data-state="loading"] .state-loading{display:block}` 没有 flex 增长（对比 styles.css:295-302 空态用 `flex:1 1 auto`）；实测 844 视口：.state-loading 76→264（高 188），#logoutBtn 288→344，下方空白 844-344=500px；normal 态 logout 672.8，empty 态 764；parent/src/pages/settings/index.vue:49-59 在 `v-if="loading"` 时只渲染「加载中...」，logout 位于 `v-else` 内。
- 建议：给 `[data-state="loading"] .state-loading` 补 `flex: 1 1 auto`（骨架纵向铺满，或与源码一致在 loading 期间隐藏 logout-wrap），保证三种状态下 logout 底部位置稳定、不出现首屏中部悬浮的危险按钮。

## P2 · 打磨项（4）

### 1. [completeness] 错误态把「退出登录」整块隐藏，而源码在加载失败时仍渲染退出按钮；空态/加载态都保留、仅错误态隐藏，四态显隐规则自相矛盾。

- 证据：styles.css:304-306 `[data-state="error"] .logout-wrap{display:none}`；parent/src/pages/settings/index.vue:11-18 的 loadData 只有 `finally` 没有 `catch`，失败后 `loading=false` 会走 `v-else`（index.vue:50-59）渲染卡片与 `logout-btn`（index.vue:58）；对照 styles.css:295-302 空态与 379-383 加载态均未隐藏 logout-wrap。
- 建议：删掉 `[data-state="error"] .logout-wrap{display:none}`，让错误态保留退出登录（与源码、空态、加载态一致）；若担心错误态用户误触，降级为弱化样式而非移除。

### 2. [completeness] 原型没有任何开关保存失败分支：settleSwitch 无条件提示「已更新」，源码 catch 里的「更新失败」toast 与回滚语义在原型中无处可触发，落地时容易照抄成功路径。

- 证据：parent/src/pages/settings/index.vue:26-27 `catch (err) { uni.showToast({ title: err?.message || "更新失败", icon: "none" }) }`；docs/.../25-settings/app.js:94-100 `settleSwitch(row)` 固定 `showToast('已更新')`，全文件无失败路径；notes.md:63 只写「失败路径在真实实现里回滚」，原型五态芯片也没有失败态演示。
- 建议：增加可触发的保存失败演示（例如把第 5 个芯片改为「保存失败」示意，或让「公告通知」行固定失败），补上开关回滚 + 行内/ toast「更新失败」的规格；若坚持不做，在 notes 的覆盖状态表里明确标注该分支未原型化并写清回滚行为。

### 3. [design-core] 两条组尾注与「同步中…」使用 11px，低于「正文 ≥ 13px」的设计核心阈值；说明性正文被压到标签级字号。

- 证据：styles.css:102-110 `.group-note{font-size:var(--text-xs)}`、styles.css:156-162 `.row-sync{font-size:var(--text-xs)}`；parent/25-settings/tokens.css:39 `--text-xs: 11px`（正文 token `--text-base: 13px` 在 41 行）；_shared/brief.md:45 与 158「触控 ≥44×44px；正文 ≥ 13px」；notes.md:78 只论证了对比度，未处理字号。
- 建议：组尾注改用 `var(--text-base)`（13px）；「同步中…」若要留在行尾可用 `--text-sm`（12px）但需批次统一，至少把解释性句子提到 13px。

### 4. [notes-accuracy] notes 声称「演示胶囊与退出按钮矩形无交叠」，在 390×667 视口滚动到底实测两矩形相交，展开演示条时更是直接盖住退出按钮上半部；该断言只在 390×900 下成立。

- 证据：notes.md:40「演示胶囊与退出按钮矩形无交叠」；实测 390×667 滚到底：logout y 586.8–642.8（x 61.5–423.5），演示开关 y 623–651（x 348–418）→ 相交；展开后 chips y 555–617 覆盖 logout 586.8–617 区间；notes.md:86 自述交互实测视口为 390×900。
- 建议：为页面底部补出演示条高度的安全间距（或演示条收起时缩为圆点/上移），并把 notes 的断言改为限定视口（「390×900 下无交叠」），避免以单一高视口结果代表全部宽度。

## 2026-10-27 P1 修复记录

- P1-1 [logic] 加载态骨架只有 188px、退出登录悬浮首屏中部 → **已修**
  - 采用建议中的第二种（与源码一致）：`styles.css` 新增 `[data-state="loading"] .logout-wrap{display:none}`，加载态只留骨架，不再渲染危险按钮。
  - 依据 `parent/src/pages/settings/index.vue:49-59`：`v-if="loading"` 只渲染「加载中...」，`logout-btn` 在 `v-else` 内。
  - 未选 `flex:1 1 auto`：骨架纵向拉伸只是让容器变高，188px 的骨架仍贴顶部，危险按钮被吊到页面底部，中间留 476px 空洞（截图复核读成「布局塌陷」）；改为垂直居中又会让骨架脱离内容落点（同样被判为异常）。隐藏退出登录与源码分支同构，也与批次内其他页加载态（骨架置顶、下方留白）一致。
  - 实测（390×844）：normal/disabled 的 logout 672.8–728.8，empty 764–820，loading 无 logout（骨架 76–264）；390×667 同构（empty 587–643）。loading 不再出现按钮，故与 normal/empty 无按钮位置跳动；normal↔empty 的 91px 差异来自内容高度，属既定「退出登录跟随内容末尾、不吸底」的设计。
  - 验证：`od-check …/25-settings/index.html --width 390` → `problems: []`；`od-shot` → preview.png（390×1000，113,805 bytes）；`od-audit` → 本页无 ✗。notes.md 的「覆盖状态」「与现状的差异」「验收」已同步。
- P2 四项按本轮要求未动，保持原状。
