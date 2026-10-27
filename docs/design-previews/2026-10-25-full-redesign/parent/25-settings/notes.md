# 设置（parent）

- 源码：`parent/src/pages/settings/index.vue`（`onLoad` → `ensureLogin()` → `GET /parent/settings`；只渲染 `settings.notification` 的 6 个开关 + 一个「退出登录」按钮）
- 契约：`backend/api/api-parent/11-settings.md`（`GET/PUT /parent/settings`，`notification` 6 键 + `privacy` 2 键）
- 入口：`parent/src/pages/profile/index.vue` →「设置」`uni.navigateTo`，因此本页是子页：**有返回箭头**、标题「设置」、**无 tabBar**
- 设计系统：`user:nurture-planet-local`，surface=parent
- 生成方式：**OpenDesign 首跑成功**（569s；项目 `ccc4a2c6` / run `fec20ac4`，`state.json` 终态 `succeeded`，三件套齐全，未走手写兜底）。交付目录与 `.od/projects/ccc4a2c6/` 逐字节比对：`styles.css` / `app.js` / `tokens.css` 一致；`index.html` 已用项目终态文件覆盖（差异仅为 run 结束后内层 agent 追加的 `data-od-id` 标注）。覆盖后另做 4 处手工修（见「与现状的差异」第 7–9 条），故当前三个文件与项目目录**有意不同**
- 方向：**「安静的开关板」**——微信原生设置页的克制感：两个弱化组标题 + 两张白色 cell 卡，橙色只做「开着」的信号，退出登录单独一张卡、红色文字，不抢主视觉。

## 信息层级

1. 微信外壳：20px 状态栏 + 44px 导航栏（**返回箭头** + 标题「设置」+ 胶囊）；**无 tabBar**
2. 组标题「消息通知」（12px / `--muted` / 500，弱化成标签而非标题）
3. 消息通知卡：6 行 cell（56px 行高，行间 1px 内缩分隔线）：通知推送 / 请假通知 / 接送通知 / 成长记录 / 任务提醒 / 公告通知，右侧 48×30 开关
4. 组尾注（11px）：关闭后消息仍会保留在消息列表，只是不再提醒
5. 组标题「隐私」+ 隐私卡 2 行：向老师展示手机号 / 允许展示成长记录
6. 组尾注：仅影响老师和同班家长看到的信息
7. 退出登录卡：独立一组、横向居中、`--danger` 系文字、白底 `--radius-card`、无箭头
8. 页脚留白 `calc(24px + env(safe-area-inset-bottom))`

## 覆盖状态

| 状态 | 表现 |
| --- | --- |
| 正常（默认） | 两卡 8 行开关（任务提醒、允许展示成长记录默认关，其余开）+ 两条组尾注 + 退出登录 |
| 空态 | 隐藏两个分组，居中空态卡：64px 线框铃铛圆 +「还没有可设置的提醒项」+「重新加载」主按钮；退出登录保留 |
| 加载 | 骨架屏：组标题条 + 三行骨架（文字条 + 开关块），`breathe` 呼吸动画；真实列表隐藏，退出登录一并隐藏（与源码 `v-if="loading"` 分支一致，见「与现状的差异」第 10 条） |
| 失败 | 隐藏两个分组与退出登录，整页居中错误块：`--danger-wash` 圆 + 感叹号 +「设置加载失败」+「网络不太稳定，请稍后再试」+「重试」 |
| 禁用 | 开关提交中：该行文字转 `--muted`、开关 `opacity:.5`、`pointer-events:none`、行尾出现 11px「同步中…」；演示条第 5 个芯片把「接送通知」固定在此态以便审阅 |

前四态由共享 `shell.js` 写 `data-state`，页面用属性选择器接管；第 5 个芯片 `disabled` 是本页对「禁用态」的演示入口（相应说明见「遗留」第 5 条）。

## 交互（Playwright 真实浏览器实测）

- 点开关整行（`role="switch"` + `aria-checked`）：立即翻转 → 行进入「同步中…」（450ms）→ 回正常 + toast「已更新」；同步中再点无效
- 「退出登录」→ 居中确认框（遮罩 `--fg` 46% + 1.5px 模糊、`aria-modal`、滚动锁）；取消 / 点遮罩 / Esc 关闭并归还焦点；确认 → toast「已退出登录」
- 键盘：初始焦点落在对话框容器（`tabindex="-1"`），Tab → 取消，Shift+Tab → 退出登录，焦点被限制在框内
- 空态「重新加载」/ 失败「重试」：按钮 1 秒降级为「重试中…」并回正常态 + toast「设置已刷新」
- 返回箭头 → toast「返回上一页」；四/五态芯片切换正常，各态 `scrollWidth - clientWidth = 0`
- 实测结果：`aria-checked` 翻转与回落、toast 文案、对话框开/取消/确认、四态显隐（groups/empty/loading/error/logout-wrap 的 `display`）、演示胶囊与退出按钮矩形无交叠，全部符合预期；console 无 error

## token 映射

| 元素 | Token |
| --- | --- |
| 页面底 / 卡片面 / 分隔线 | `--bg` / `--surface` / `--border` |
| 组标题 / 组尾注 | `--muted`（12px / 11px），`--weight-medium` / `--weight-normal` |
| 行文案 | `--fg` + `--text-md`，`--leading-body` |
| 开关（开） | `--accent-gradient` 轨道 + `--surface` 滑块 + `--shadow-sm` |
| 开关（关） | `color-mix(in srgb, var(--muted) 78%, var(--surface))` 轨道（无独立 token，用 mix 保持中性灰且不写裸色） |
| 卡片圆角 / 阴影 | `--radius-card` / `--shadow-md`（退出登录卡 `--shadow-sm`） |
| 主按钮 | `--accent-gradient` + `--fg` 深字（橙底深字约 5.4:1）+ `--shadow-md` + `--radius-pill` |
| 退出登录 / 确认按钮文字 | `color-mix(in srgb, var(--danger) 72%, var(--fg))` ≈ `#B43C41`，白底 5.7:1 |
| 错误态 | `--danger-wash` 圆 + `--danger` 图标 |
| 骨架 / 呼吸 | `color-mix(var(--muted) 24%)` 条 + `breathe 1.6s`；`prefers-reduced-motion` 下停止 |
| 间距 / 字号 / 动效 | `--space-*`、`--text-xs…xl`、`--motion-fast/base`、`--ease`、`--page-pad-x/y` |

## 与现状的差异

1. **补齐「隐私」分组**：源码把 `privacy` 原样放在 state 里并随 PUT 回传，但模板从不渲染；契约明确有 `show_phone_to_teacher` / `allow_growth_share`。原型按契约画出 2 行，让这组设置第一次可见。
2. **开关行从「文字 + 挤在右边」升级为完整 cell**：整行 56px 可点（≥44 触控）、行间内缩分隔线、`role="switch"` 语义；源码是 `justify-content: space-between` 的裸行。
3. **开关换成小程序/iOS 语汇**：48×30 轨道 + 26px 滑块 + `translateX` 动效；源码用原生 `<switch color="#F97316">`，在 H5/预览里是另一套观感。
4. **「保存中」第一次有反馈**：源码 `toggleNotification` 先 `await PUT` 再翻转，期间开关不动、没有任何提示；原型改为乐观翻转 + 行内「同步中…」+ 期间禁点，失败路径在真实实现里回滚。
5. **退出登录从粉色胶囊改为独立卡片**：源码是 `#fff0ef` 底 + `#e6534f` 字的 88rpx 胶囊，与「消息通知」卡贴在一起；原型把它拆成独立一组、紧跟隐私组（不吸底），视觉上与设置项区分开，确认框仍保持微信原生居中框。
6. **三态从「一行文字」升级为完整页面状态**：源码加载态只有居中「加载中...」，且 `loadData` 的 `finally` 吞掉异常后设置区会空白；原型给骨架、空态 CTA、错误态重试。
7. **手工修 1（退出登录不再吸底）**：初版 `.logout-wrap { margin-top: auto }` 把按钮吸到屏幕底部，390×1000 画布上留下约 220px 空洞，视觉复核判定像「布局塌陷」。改为紧跟最后一组、间距 24px（微信设置页也是列表末尾接危险操作）。同时撤掉为避让它而加在 `mp-demo` 上的行内 `bottom` 覆盖——现在外壳完全用 shell.css 默认值，页面只有 `[data-state]` 一处状态选择器。
8. **手工修 2（对话框初始焦点）**：改为聚焦对话框容器（`tabindex="-1"` + `.dialog:focus{outline:none}`），并补 Tab/Shift+Tab 从容器进入首/末按钮的分支。原实现聚焦「取消」，程序化焦点会画出 2px 橙色焦点环，在危险操作旁会被读成「推荐此项」。
9. **手工修 3（文案/结构微调）**：确认框说明用「退出后需重新微信授权登录。」替代源码的「确定要退出登录吗?」（标题已重复了这个问题）；空态/错误态文案改为固定人话，不透出 `err.message`。
10. **手工修 4（加载态不再渲染退出登录）**：初版在加载态保留退出按钮，但骨架只有 188px，危险按钮被顶到首屏中部（y≈288），下方空出 500px，与 normal/empty 的底部位置相差近 400px。源码 `v-if="loading"` 时只渲染「加载中...」、`logout-btn` 位于 `v-else` 内，故改为 `[data-state="loading"] .logout-wrap{display:none}`，加载态只留骨架；退出按钮回到内容末尾，不再出现在加载态。

## 遗留 / 需批次决策

1. **「清除缓存」「关于」源码与契约都没有**：调度清单里提到这两项，但 `settings/index.vue` 与 `11-settings.md` 均无对应入口/字段，按「不虚构功能」未画。若产品要，需要先加接口（缓存大小、版本号）再排 UI。
2. **隐私分组需要新增提交逻辑**：源码只有 `toggleNotification`，`privacy` 是原样回传。原型画出了这 2 个开关，但落地必须在页面里补 `togglePrivacy` 并确认后端 `PUT` 对 `privacy` 的校验（契约示例里有，实现未见显式分支）。
3. **开关就是「微信订阅消息」吗**：源码这 6 个开关只改服务端偏好，不涉及 `wx.requestSubscribeMessage`。若产品以为关掉就真收不到推送，中间还缺一步订阅授权，需明确「偏好 ≠ 订阅」。
4. **退出登录没有二次校验**：`clearAuth()` 后直接 `reLaunch` 到登录页，`onLoad` 会立即静默 `ensureLogin`，等于退出后又自动登录（`pages/auth/login` 的静默逻辑）。落地需决定是否真的清 session / 是否要用户手动点登录。
5. **5 个演示芯片与「严格 4 态」约定不一致**：第 5 个 `disabled` 承载「禁用态」（本页唯一可降级控件是开关）。与 findings 里 A22 / A35 同类问题，请批次统一：允许第 5 态，或把禁用态并进「正常」用一个静态禁用行示意。
6. **组尾注是新增文案**：「关闭后消息仍会保留在消息列表」等两句解释在源码里不存在，是本次设计补充；若与真实推送逻辑不符需产品改口径。
7. **对比度**：`--muted` 压白卡约 4.65:1（11px 组尾注刚好过 AA 线），沿用 findings「token 缺口 2」的结论，不在本页单独打补丁。

## 验收

- `node scripts/od-check.mjs …/25-settings/index.html --width 390` → `problems: []`（`scrollWidth 390 = clientWidth 390`、`overflowing: []`、无 console error）
- `node scripts/od-shot.mjs … --width 390 --dsf 2` → `preview.png`（390 × 1000，113,805 bytes）
- `node scripts/od-audit.mjs docs/design-previews/2026-10-25-full-redesign` → 本页无 ✗（无裸色值、无 nowrap、无 `.mp-*` 重定义、已链接 `../../_shared/shell.css`、无 `_shared/` 副本、无 `.mcp.json`）
- 视觉复核（mcp-vision 读 `preview.png` + 空/加载/错误/对话框四张态图）：状态栏 + 导航栏 + **返回箭头** + 胶囊齐全；**无 tabBar**；无直角；文字无裁切、无横向滚动；橙色仅在开启态开关与主按钮；「原型演示」不遮挡内容
- 交互实测（Playwright headless，390×900）：见「交互」一节，全部通过；改焦点逻辑后复测对话框与键盘路径
- 2026-10-27 P1 复测（390×844 / 390×667 两档，逐态量矩形）：normal 与 disabled 的退出按钮 672.8–728.8，empty 764–820（667 档 587–643），loading 不再出现退出按钮（骨架 76–264）；状态切换不再有近 385px 的跳动，首屏中部无悬浮的危险按钮
- 未修改 `teacher/` 与 `parent/` 源码；输出目录无 `.mcp.json`、无 `_shared/` 副本、未复制 `shell.css`、`styles.css` 无 `.mp-*` 定义
