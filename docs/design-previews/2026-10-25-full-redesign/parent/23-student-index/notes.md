# 孩子信息（parent）

- 源码：`parent/src/pages/student/index.vue`（`onLoad` → `ensureLogin()` → 并行 `GET /parent/student/profile` 与 `GET /parent/health/records?student_id=`；档案与晨检各自维护 loading / error，晨检失败只降级晨检卡）
- 入口：`pages/profile/index.vue` →「我的孩子」`uni.navigateTo`，因此本页是子页：导航栏有**返回箭头**，标题「孩子信息」，**无 tabBar**
- 设计系统：`user:nurture-planet-local`，surface=parent
- 生成方式：OpenDesign 首跑成功（1208s；项目 `2fa736d2` / run `c8d484e9`，三件套齐全，未走手写兜底）。脚本自报「daemon status never settled」，但 `.od/runs/c8d484e9-4ca9-4428-99f8-5b75e33af310/state.json` 落定 `succeeded`（exitCode 0）；交付时 `index.html` / `styles.css` / `app.js` 与 `.od/projects/2fa736d2-b205-460c-a349-05d3023c6ec0/` 逐字节一致，不是陈旧拷贝；`styles.css` 随后有下述 3 处手工微调
- 方向：**「成长护照」**——身份带 + 2×2 档案事实格 + 被收进浅底块的健康备注 + 安静的晨检时间线 + 监护人名单；暖色只做记号，敏感信息靠降级排版而不是警示色表达。

## 信息层级

1. 微信外壳：20px 状态栏 + 44px 导航栏（**返回箭头** + 标题「孩子信息」+ 胶囊）；**无 tabBar**
2. 身份带：64px 首字头像（暖底 + 1px 描边）→「乐乐」20px/600 → 班级胶囊「小一班」
3. 档案事实 2×2：性别 / 生日 / 年龄 / 入园时间（1px 分隔线；label 11px `--muted`，value 17px `--fg`）
4. 健康备注：浅底内嵌块（盾牌图标 + 11px letter-spaced label + 13px 正文），无告警色、无加粗；「部分档案信息暂未填写 + 刷新档案」提示条（仅空态，位于档案卡与晨检之间）
5. 最近晨检：区块头「最近晨检」+ 右侧「近 7 天」；卡片内 3 条时间线（日期 + 星期 + 状态胶囊；异常原因 / 体温 / 保健医备注）
6. 已绑定的孩子（`students.length > 1` 时）：2 行「姓名 + 班级 + 当前胶囊 + 解绑」；页脚「档案信息由园所统一维护」

## 覆盖状态

| 状态 | 表现 |
| --- | --- |
| 正常（默认） | 乐乐 / 女 / 2022-05-18 / 4岁 / 入园 2025-09-01 / 花生过敏；近 7 天 3 条晨检（10-23 需观察 37.4°C，附「已在隔离室等待家长」；10-22 与 10-21 健康）；已绑定乐乐（当前）+ 小雨 |
| 空态（部分字段缺失） | 生日/年龄/入园时间以「—」占位；健康备注隐藏；晨检空态「近 7 天暂无晨检记录」；已绑定孩子卡隐藏（与源码 `v-if="students.length > 1"` 一致）；出现「部分档案信息暂未填写 + 刷新档案」 |
| 加载 | 档案骨架（头像圆 + 姓名/班级条 + 2×2 事实条）+ 晨检 3 行骨架 + 已绑定 2 行骨架；`aria-busy="true"`；无业务文案残留；呼吸动画在 `prefers-reduced-motion` 下停止 |
| 失败 | 隐藏档案卡/晨检/绑定名单/页脚；整页居中错误块（`--danger-wash` 圆 + 感叹号 +「档案加载失败」+「网络不太稳定，请稍后再试」+「重试」） |
| 晨检失败（第 5 芯片） | 档案卡与绑定名单正常；晨检卡内只留「晨检记录加载失败」+「点击重试」；点击后 1 秒旋转 spinner + `aria-busy`，再回正常并 toast「晨检已更新」 |
| 禁用 | 页面无表单；「重试 / 刷新档案 / 点击重试」点击后 1 秒 `aria-disabled="true"`、`--border` 底、`--hint` 字、文案「重试中…」，不响应重复点击 |

演示芯片 5 个：正常 / 空态 / 加载 / 失败 / 晨检失败（`normal|empty|loading|error|health-error`），由共享 `shell.js` 写 `data-state`。

## 交互（已实测）

- 页面级「重试」→ 立即禁用并显示「重试中…」→ 1 秒后回正常、芯片 `aria-pressed` 同步、toast「已重新加载」
- 空态「刷新档案」→ 同上（`empty → normal`）
- 晨检「点击重试」→ 卡片 `data-retrying="true"` + spinner 显示 → 1 秒后回正常、toast「晨检已更新」
- 「解绑」→ 微信样式弹层「确认解绑 / 确定解除与 乐乐（小一班）的绑定吗？」；打开后焦点落「取消」，Tab / Shift+Tab 在弹层内循环，Esc、遮罩点击、取消均可关闭；「确定」关闭并 toast「已解绑」
- 返回箭头 → toast「返回上一页」；芯片切换正常；`aria-busy` 随 loading 同步；console 无 error/warn

## token 映射

| 元素 | Token |
| --- | --- |
| 页面底 / 卡片面 / 分隔线 | `--bg` / `--surface` / `--border` |
| 身份头像 | `--accent-wash` 底 + 1px `--accent-soft` 描边 + `--accent-deep` 首字，`--radius-pill` |
| 姓名 / 班级胶囊 | `--fg` 20px/600 / `--bg` 底 + `--fg-2` 字，`--radius-pill` |
| 事实格 label / value / 缺失占位 | `--muted` 11px / `--fg` 17px / `--hint` |
| 健康备注 | `--bg` 块 + `--radius-xl`；图标与 label `--muted`；正文 `--fg-2` 13px/400 |
| 晨检日期 / 体温 / 备注 | `--fg` 15px/600 / `--muted` 13px / `--muted` 13px |
| 晨检隔离提示 | `--warning` 13px/600（对应源码 `.health-isolated`） |
| 状态胶囊 | `--success-wash`+`--success`；`--warning-wash`+`--warning`；`--danger-wash`+`--danger` |
| 绑定行 / 当前胶囊 / 解绑 | `--fg` / `--bg`+`--fg-2` / `--danger-wash`+`--danger` |
| 主按钮 | `--accent` 底 + `--fg` 字（深字在橙底约 5:1，优于白字）+ `--shadow-sm` |
| 弹层遮罩 / 弹层 | `var(--fg)` + `opacity: .34`（不用 rgba）；`--surface` + `--shadow-lg`；确认键 `--danger` |
| toast | `--fg` 底 + `--surface` 字 |
| 骨架 / 呼吸 | `--border` 条 + 1.5s opacity 呼吸；`prefers-reduced-motion` 停止 |
| 间距 / 字号 / 圆角 | `--space-*`、`--text-xs…3xl`、`--radius-card/xl/pill` |

## 与现状的差异

1. **三态从一行文字升级为整页状态**：源码只有行内「加载中...」，`loadData` 无 catch，档案请求失败会停在空白；原型补骨架屏、带 CTA 的空态提示与整页错误态。
2. **补头像与入园时间**：源码只渲染姓名与 5 行文本，未使用接口返回的 `avatar` / `enroll_date`；原型用姓氏首字头像（原型禁外链）并按调度要求补「入园时间」。
3. **档案重排为 2×2 事实格**：班级从文本行提到身份带胶囊；性别/生日/年龄/入园时间做成网格；健康备注从普通行下沉为独立内嵌块。
4. **健康备注弱化（隐私）**：源码与其它字段同权重；原型改为浅底块、11px 标签、13px 正文、无告警色与加粗——可读但不抢眼。
5. **当前孩子胶囊**：接口 `students[].is_current` 源码未渲染；原型在绑定列表标出「当前」，避免解绑时认错孩子。
6. **空态定义**：源码用 `v-if` 直接隐藏缺失字段；原型保留事实格并以「—」占位 + 「部分档案信息暂未填写 + 刷新档案」，让缺失可见可恢复（该 CTA 为原型补充）。
7. **晨检局部失败升级为独立演示态**：源码确有独立 `healthError` 与「点击重试」，但只是行内文字；原型做成第 5 个芯片 + 行内错误块 + 重试 spinner，档案与绑定列表保持可见（局部降级而非整页失败）。
8. **解绑确认带上班级**：源码 `uni.showModal` 文案只有孩子姓名；原型弹层补班级，并加焦点圈定 / Esc / 遮罩关闭。确认后仅 toast「已解绑」，未模拟列表移除与当前孩子切换。
9. **生成后 3 处手工微调**：主按钮文字 `--on-brand`（白，约 2.9:1）→ `--fg`（深，约 5:1）；事实格 `gap` 由 `--space-xs` 提到 `--space-sm`、底部补 `--space-xs` 内边距；页脚 `--hint`/`--text-xs` → `--muted`/`--text-sm`。
10. **暖色克制**：橙色只落在头像底、刷新/重试文字与主按钮；晨检用功能色系，健康备注与正文保持中性。

## 遗留 / 需批次决策

1. **接口有、原型未展示的档案字段**：`student_no` / `address` / `parents[]`（父母姓名手机）。本页按「档案字段以源码为准」未画；需产品确认档案页是否补家长与地址。`is_isolated` 不属于此列——源码晨检行已渲染（`parent/src/pages/student/index.vue:118`），原型在 10-23 行补「已在隔离室等待家长」（`--warning` 语气）。
2. **健康备注可写但本页无编辑入口**：`PUT /parent/student/profile` 接收 `avatar` / `health_note`，源码未做编辑 UI；若要让家长维护过敏信息，需先定权限与审核口径。
3. **空态「刷新档案」是原型补的 CTA**（源码无下拉刷新、无刷新按钮）；需确认保留，还是改为下拉刷新。
4. **第 5 个演示芯片「晨检失败」偏离批次 4 态约定**（先例：07 请假、16 授权人、17 亲子任务也是 5 芯片）；批次需统一：允许「局部失败」作为第 5 态，还是合并进「失败」。
5. **解绑后的真实流程未模拟**：源码解绑成功会从 `students` 移除；若解绑的是当前孩子，`currentStudent.js` 会切换本地当前孩子并重载。原型只 toast，落地需按源码补切换与空绑定兜底。
6. **年龄口径**：示例「4岁」为 2026-10 的当前年龄（生日 2022-05-18）；入园 2025-09-01 时为 3 岁 3 个月；晨检「10月23日 周五」与 2026 年日历一致（2025-10-23 是周四）。视觉复核模型按入学年份误算过一次，示例数据本身自洽。
7. **头像兜底**：接口 `avatar` 可能为空且原型禁外链，落地保留姓氏首字占位。
8. **晨检固定 7 天窗口、无历史入口**（契约与源码一致）；要更多历史需扩 `backend/api/api-parent/12-health.md`。
9. **绑定卡只在 >1 时出现**（源码 `v-if`），单孩家庭看不到「已绑定的孩子」区块；若要在单孩时也展示（例如提示「仅绑定 1 个孩子」），需产品决定。

## 验收

- `node scripts/od-check.mjs …/23-student-index/index.html --width 390` → `problems: []`（`scrollWidth 390 = clientWidth 390`，`overflowing: []`，`bodyHeight 1048`；滚动 148px 后 `statusTop 0` / `navTop 20`，吸顶正常；无 console error）
- `node scripts/od-shot.mjs … --width 390 --dsf 2` → `preview.png`（390 × 1048，134,386 bytes，最终 CSS 渲染）
- `node scripts/od-audit.mjs docs/design-previews/2026-10-25-full-redesign` → 本页无 ✗（无裸色值、无 nowrap、无 `.mp-*` 重定义、已链接 `../../_shared/shell.css`、无 `_shared/` 副本、无 `.mcp.json`）
- 视觉复核（mcp-vision 读 `preview.png` 及空/加载/失败/晨检失败四态截图）：状态栏 + 导航栏 + **返回箭头** + 胶囊齐全；**无 tabBar**；事实格舒展、无裁切；健康备注为弱化浅底块、未用告警色；晨检/绑定/页脚层级清晰；无横向溢出、无直角卡片
- 交互实测（CDP 真实 click）：页面级重试 1 秒降级与恢复、空态刷新、晨检重试 spinner 与恢复、解绑弹层（焦点落「取消」、Tab/Shift+Tab 循环、Esc 关闭）、返回 toast 均正常；console 无 error/warn
- 未修改 `teacher/` 与 `parent/` 源码；输出目录无 `.mcp.json`、无 `_shared/` 副本、未复制 `shell.css`
