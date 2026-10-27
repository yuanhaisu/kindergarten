# 请假记录（parent）

- 源码：`parent/src/pages/leave/index.vue`（`onLoad` / `onPullDownRefresh` → `GET /parent/leaves`；`switchStatus` 带 `status` 重拉；`goCreate` → `/pages/leave/create`，`goDetail` → `/pages/leave/detail?id=`）
- 设计系统：`user:nurture-planet-local`，surface=parent
- 生成方式：OpenDesign（首跑成功，301s，四件套齐全，未走手写兜底）
- 方向：**「一条请假 = 一张进度卡」**——顶部三个数字给全局，状态分段给入口，下面是可扫读的进度卡；品牌橙只落在待审批数字、选中分段和右下新建按钮上。

## 信息层级

1. 微信外壳：20px 状态栏 + 44px 导航栏（返回箭头，标题「请假记录」，右侧胶囊）
2. 统计卡（3 等分）：全部请假 6 / 待审批 1（橙色）/ 已同意 2，竖向发丝分隔
3. 状态分段（4 等宽 44px 胶囊，`repeat(4, minmax(0,1fr))`）：全部 / 待审批 / 已同意 / 已拒绝
4. 请假卡片（6 条）：40px 圆形首字头像「乐」+ 学生名「乐乐」+ 状态徽标（待审批暖黄 / 已同意绿 / 已驳回红 / 已取消灰）→ 类型标签 +「起止日期 · N 天」→ 事由（2 行截断）→ 待审批显示「提交于 …」，已同意/已驳回显示审批意见块（`审批意见 · 王老师 日期` + 正文，驳回用 `--danger-wash` 底）
5. 列表脚注：`共 6 条请假记录`
6. 右下固定主操作：48px「＋ 请假」胶囊（`--accent-gradient`）

## 覆盖状态

| 状态 | 表现 |
| --- | --- |
| 正常（默认） | 6 条记录：待审批 1、已同意 2、已驳回 2、已取消 1；统计 6 / 1 / 2；脚注「共 6 条请假记录」；右下「＋ 请假」常驻 |
| 空 | 列表、脚注、FAB 全部隐藏；空态卡（日历图标 +「暂无请假记录」+「提交请假后，老师的审批结果会显示在这里」+「发起请假」主按钮）；统计归 0 / 0 / 0；分段仍可点（点了也不会有数据） |
| 加载骨架 | 列表隐藏，换 3 张骨架卡（头像圆 + 标题/正文/元信息三条呼吸条）；统计数字换成小骨架条；分段整体 `opacity:.5` + `pointer-events:none`；FAB 保留（加载中也可以先发起新请假） |
| 加载失败 | 列表、脚注、FAB 全隐藏；错误卡（`--danger-wash` 圆底感叹号 +「请假记录加载失败」+「网络不太稳定，请稍后再试」+「重试」主按钮）；统计显示「—」 |
| 禁用 | ① 加载态下 4 个分段全部 `disabled`；② 失败态「重试」点击后立即 `disabled`、文案「重试中…」，1s 后回到正常态、恢复列表与统计（沿用重试前的筛选，演示芯片同步回到「正常」）并 toast「已重新加载」；③ 筛选结果为空时显示筛选空态（「该状态下暂无请假记录」+ 描边「查看全部」），不误判成整页空态 |

演示芯片：正常 / 空态 / 加载 / 失败（`data-mp-state` = normal / empty / loading / error）。筛选空态是一个页内分支，不是第五个演示状态。

## token 映射

| 元素 | Token |
| --- | --- |
| 页面底 / 卡片面 | `--bg` / `--surface` |
| 统计数字 / 待审批强调 | `--fg` / `--accent` |
| 卡片圆角 / 分段圆角 / 徽标圆角 | `--radius-card` / `--radius-pill` / `--radius-pill` |
| 卡片阴影 | `--shadow-sm`（FAB `--shadow-lg`，toast `--shadow-lg`） |
| 卡片标题 / 事由 | `--fg`（`--text-md` 600 / `--text-base` 400） |
| 日期、脚注、审批元信息 | `--muted`（`--hint` 在暖白底对比不足，按 `_shared/findings.md` 不用作正文色） |
| 状态徽标 | 待审批 `--warning-wash`+`--warning`；已同意 `--success-wash`+`--success`；已驳回 `--danger-wash`+`--danger`；已取消 `--bg`+`--muted` |
| 头像 | `--bg`+`--muted`；待审批卡片 `--accent-wash`+`--accent-deep` |
| 分段选中 | `--accent` 底 + `--on-brand` 字（按下 `--accent-deep`） |
| 审批意见块 | 已同意 `--bg` 底；已驳回 `--danger-wash` 底；`--radius-sm` |
| FAB / 重试 / 发起请假 | `--accent-gradient` + `--on-brand` + `--shadow-md` |
| 骨架 | `--bg` 条 + `--bg` 圆，`skeleton-breathe 1.4s var(--ease)` 呼吸，`prefers-reduced-motion` 下停止 |
| 间距 / 字号 / 动效 | `--space-xs…xl`、`--text-xs…3xl`、`--weight-*`、`--page-pad-x/y`、`--motion-fast`、`--ease` |

## 与现状的差异

- **新建入口从统计卡里挪到右下 FAB**：源码把「+ 请假」塞进统计卡第 4 格，统计行变成 3 数字 + 1 按钮，权重混乱，且列表一长就滚走。原型统计卡回归 3 等分，新建改成常驻 48px FAB；`.mp-demo` 用唯一允许的行内 `bottom: calc(84px + env(safe-area-inset-bottom))` 上移避让（`_shared/shell.md` 允许的这一处）。
- **状态分段从「左对齐自适应宽」改为「4 等宽」**：源码 `.tab-row` 是 flex + padding，四个胶囊宽度随文案变化，点「全部」和「已拒绝」的手指落点差一倍；原型 44px 高、等宽四列，落点一致，也刚好铺满 390px。
- **事由从单行 nowrap 省略改为 2 行截断**：源码 `.reason` 是 `white-space: nowrap; text-overflow: ellipsis`，长理由只剩前半句（且批次约束禁止 nowrap）。原型 `-webkit-line-clamp: 2` + `overflow-wrap: anywhere`，保留足够上下文又不撑高卡片。
- **天数并入日期行**：源码把「N 天」放在学生名下方、日期单独一行；原型合成「10月28日 ~ 10月29日 · 2天」，一条元信息行读完时间跨度，卡片少一行。
- **头像占位从「?」蓝色块改为姓名首字**：源码 `avatar` 字段列表接口根本不返回，永远落到 `?` + 蓝底；原型用姓名首字（「乐」）+ 中性底，待审批卡片染成 `--accent-wash`，比问号更像真实头像占位。
- **新增审批意见块**：源码列表只显示状态徽标，要看老师说了什么必须点进详情；原型在已同意/已驳回卡片直接显示「审批意见 · 王老师 日期 + 正文」，驳回用红底洗色（字段来源见遗留 2）。
- **新增类型标签**：病假/事假/其他 从 `create.vue` 的类型选择器取来（见遗留 1）。
- **加载/空/失败从一行文字升级为完整状态**：源码三者都是一行居中文字（失败只有 toast，无持久错误分支）；原型分别给骨架卡、带 CTA 的空态卡、带重试的错误卡。
- **统计口径保持全量**：源码切 tab 会带 `status` 重拉，`stats` 因此变成子集计数（例如停在「已同意」时待审批恒为 0）；原型筛选走前端隐藏、统计固定为全量 6 / 1 / 2，避免数字随筛选跳动（见遗留 3）。

## 遗留

1. **「请假类型」没有后端字段，需产品确认**：`create.vue` 有 病假/事假/其他 三个选择器（`activeType`），但提交时从未带上，`domain.LeaveRequest` 与 `leave_service.go` 的列表项都没有 `leave_type`。原型按设计意图画了类型标签；要落地须先在 `backend/api/api-parent`（请假创建/列表）补字段，并让 `create.vue` 真正提交。
2. **「审批意见」列表接口不返回**：`review_remark` 目前只有详情接口带出（`detail.vue` 的 `status-remark`），`leaveListItemFromRow` 没有该字段。原型在卡片上直接显示属设计提案；落地需要在列表项补 `review_remark` + `reviewer.nickname` + `reviewed_at`，否则这块要么空、要么得点进详情看。
3. **统计口径需要在真实实现里定死**：源码 `switchStatus` 带 `status` 重拉，`stats` 又是从返回列表算的，等于「筛了什么，统计就只剩什么」。原型选了全量统计。建议后端列表固定返回全量 stats（或前端保留一份全量列表再筛选），二选一，别沿用现状的隐式漂移。
4. **「已取消」记录只在「全部」里可见**：`statusTabs` 只有 全部/待审批/已同意/已拒绝，`status === 3`（已取消）没有对应 tab。原型按源码保留这个结构，并放了一条已取消样例让灰徽标可见；是否补「已取消」tab 需产品确认（补了就是 5 个分段，390px 下每个约 78px，仍可点）。
5. **下拉刷新未在原型体现**：源码 `onPullDownRefresh` 可用，原型是静态页，没有下拉手势与刷新动画；真实实现保留即可。
6. **取消请假入口不在本页**：源码只有详情页对 `status === 0` 提供取消；原型列表卡不做取消按钮，点卡片统一 toast「已打开请假详情」。
7. **未做的**：搜索、日期区间筛选、多孩子切换、批量操作、附件缩略图——源码均无，原型也不发明。

## 验收

- `node scripts/od-check.mjs docs/design-previews/2026-10-25-full-redesign/parent/06-leave-index/index.html --width 390` → `problems: []`（`scrollWidth 390 = clientWidth 390`，`overflowing: []`，`bodyHeight 1268`；滚动 368px 后 `statusTop: 0 / navTop: 20`，吸顶正常，无 console error）
- `node scripts/od-shot.mjs … --width 390 --dsf 2` → `preview.png`（390 × 1268，225 KB）
- `node scripts/od-audit.mjs docs/design-previews/2026-10-25-full-redesign` → 本页无 ✗（无裸色值、无 nowrap、无 `.mp-*` 重定义、已链接 `../../_shared/shell.css`、无 `_shared/` 副本、无 `.mcp.json`）
- 视觉复核（mcp-vision 读 `preview.png`）：状态栏/导航栏/返回箭头/胶囊齐全，无 tabBar；6 张卡片文字完整无裁切、无横向滚动；卡片/标签/按钮全圆角；橙色只出现在待审批数字、选中分段与 FAB；FAB 与「原型演示」胶囊同排不重叠
