# 班级信息（parent）

- 源码：`parent/src/pages/class/index.vue`（`onLoad` 拉一次：`ensureLogin()` → `GET /parent/class/info?student_id=`；三态 `loading / loadError / info`）
- 入口：`pages/profile/index.vue` →「班级信息」`uni.navigateTo`，因此本页是子页：导航栏有返回箭头，标题「班级信息」，**无 tabBar**
- 设计系统：`user:nurture-planet-local`，surface=parent
- 生成方式：OpenDesign 首跑成功（416s；项目 `3bb17ad4` / run `8113d427`，三件套齐全，未走手写兜底）。脚本先以「产物稳定」推断终态，随后 `.od/runs/<runId>/state.json` 落定 `succeeded`。终态比对：`app.js` / `tokens.css` 与 `.od/projects/3bb17ad4/` 逐字节一致；`index.html` / `styles.css` 的差异**仅为**下述两处手工微调（人数单位、职务胶囊文字色，见「与现状的差异」第 6 条），不是陈旧的未终态拷贝
- 方向：**「班级名片」**——把一屏稀疏数据做成一张记得住的名片：班级徽记 + 学年 + 在班人数，下面是一份安静、可信的老师名单；暖色只做记号，不做底色。

## 信息层级

1. 微信外壳：20px 状态栏 + 44px 导航栏（**返回箭头** + 标题「班级信息」+ 胶囊）；**无 tabBar**
2. 班级名片卡：56px 柔和暖底徽记「小一」→「小一班」20px/600 →「小班 · 2025-2026 学年」；右侧竖排「在班 / 28 人」，用 1px 分隔线隔开
3. 老师区块标题：「班级老师」+ 右侧「3 位」
4. 老师名单卡（三行，1px 分隔线）：48px 姓氏首字头像 → 姓名 + 职务胶囊（主班老师用 `--accent-wash` + 橙点，副班/生活为中性底）→ 脱敏电话
5. 班级相关卡：「成长相册」「通知公告」两行 cell（线性图标 + 标题 + 橙色 `›`）
6. 页脚一行小字：「班级信息由园所统一维护」

## 覆盖状态

| 状态 | 表现 |
| --- | --- |
| 正常（默认） | 小一班 / 小班 / 2025-2026 学年 / 在班 28 人；王老师（主班）、李老师（副班）、刘老师（生活）及脱敏电话；两个入口 |
| 空态 | 班级名片保留（班级本身存在）；老师卡内换成 56px 线框人形 +「暂无老师信息」+「班级老师尚未配置，请稍后重新加载」+「重新加载」主按钮；入口卡与页脚保留 |
| 加载 | 班级名片换骨架（徽记块 + 标题/副标题条 + 人数块），老师卡换 3 行骨架（圆形头像 + 姓名/电话条），呼吸动画；老师计数、入口卡隐藏；页脚保留 |
| 失败 | 隐藏班级卡、老师卡、入口卡与页脚；整页居中错误块（`--danger-wash` 圆 + 感叹号 +「班级信息加载失败」+「网络不太稳定，请稍后再试」+「重试」） |
| 禁用 | 本页无表单；唯一降级点是「重试 / 重新加载」点击后的 1 秒：`aria-disabled="true"`、`--border` 底、`--hint` 字、文案「重试中…」，不响应重复点击 |

演示芯片 4 个：正常 / 空态 / 加载 / 失败（`normal|empty|loading|error`），由共享 `shell.js` 写 `data-state`。

## 交互（已实测）

- 「成长相册」→ toast「已打开成长相册」；「通知公告」→ toast「已打开通知公告」
- 「重试」/「重新加载」→ 立即禁用并显示「重试中…」，1 秒后回到正常态、同步芯片 `aria-pressed`、toast「已重新加载」
- 返回箭头 → toast「返回上一页」（原型反馈；与批次其它页一致）
- 四态芯片切换正常，`aria-busy` 随 loading 同步；console 无 error/warn

## token 映射

| 元素 | Token |
| --- | --- |
| 页面底 / 卡片面 / 分隔线 | `--bg` / `--surface` / `--border` |
| 班级徽记 | `--accent-gradient-soft` + 1px `--accent-soft` 描边 + `--fg` 字，圆角 `--radius-2xl` |
| 班级名 / 副标题 / 人数 | `--fg` 20px / `--muted` 12px / `--fg` 20px + `--muted` 单位 |
| 卡片圆角 / 阴影 | `--radius-card` / 名片 `--shadow-md`，名单与入口 `--shadow-sm` |
| 老师头像 | `--bg` + 1px `--border` + `--fg-2` 首字，`--radius-pill` |
| 职务胶囊 | 主班 `--accent-wash` + `--fg` + `--accent` 圆点；副班/生活 `--bg` + `--fg-2` |
| 电话 | `--muted` 13px，`tabular-nums` |
| 入口图标 / 箭头 | `--fg-2` / `--accent-deep` |
| 空态 / 错误 | `--bg` 图标圆 + `--muted`；`--danger-wash` + `--danger` |
| 主按钮 | `--accent-gradient` + `--fg` 字（深字在橙底上对比约 5:1，优于白字）+ `--shadow-md` |
| 骨架 / 呼吸 | `--border` 条 + `calc(var(--motion-slow) * 3)` alternate；`prefers-reduced-motion` 下停止 |
| 间距 / 字号 / 层级 | `--space-*`、`--text-xs…3xl`、`--weight-medium/semibold` |

## 与现状的差异

1. **三态从一行文字升级为完整页面状态**：源码加载/失败都只是居中的一行字（失败还会停在空白）；原型给骨架卡、带 CTA 的空态、带重试的错误态。
2. **老师行补上头像与职务胶囊**：接口契约有 `avatar` / `title_label`，源码只渲染纯文字且 `avatar` 完全没用；原型用姓氏首字占位（禁止外链图片），把「主班/副班/生活」做成胶囊，主班用极浅橙底区分。
3. **班级名片区重排**：源码是「标题 + 一行 grade · year · count 人」；原型拆成徽记、标题、学年、独立人数位，把「在班 28 人」从一串文字里提出来，一眼可读。
4. **新增「班级相关」入口卡（调度要求，非源码）**：源码没有相册/公告入口；原型补两个 cell，指向家长端真实存在的 `pages/growth/album` 与 `pages/notice/index`，标题用真实页面名。**未加**班级群二维码、聊天、拨号、教师详情。
5. **页脚说明**：源码无；补一行「班级信息由园所统一维护」，解释信息归属，不引入新功能。
6. **生成后两处微调**：人数由「28 / 在班」改为「在班 / 28 人」（原读法缺单位）；副班/生活职务胶囊文字由 `--muted` 改为 `--fg-2`，提升小字对比度。
7. **暖色更克制**：源码整页白卡 + 多处 `info` 蓝电话；原型把电话降为中性灰，橙色只留在徽记、主班胶囊、入口箭头与主按钮上。

## 遗留 / 需批次决策

1. **契约与实现漂移（建议批次层面记录）**：`backend/api/api-parent/08-class.md` 承诺 `teachers[].title_label` 与 `phone`，源码页面也渲染这两个字段；但 `ParentPortalService.ClassInfo`（`server/internal/service/parent_portal_service.go:2905`）实际只返回 `id / title / nickname / avatar`，**不返回 `title_label` 与 `phone`**。原型按契约展示；若后端不补齐，线上职务胶囊与电话会是空行。需决定：补 service 字段，还是页面按 `title` 自行映射并隐藏电话。
2. **入口卡属于 IA 补充**：两个入口不在 `class/index.vue` 里，是本次调度要求新增的；落地前需产品确认是否放在班级信息页，还是只在「我的」保留。
3. **无拨号能力**：源码只展示脱敏电话，`uni.makePhoneCall` 未被调用；原型不做点击拨号暗示。若要「联系老师」，需先确认脱敏号码能否拨通。
4. **空态口径**：班级信息接口不会返回「无班级」，空态实际等于「老师列表为空」；原型保留班级名片只清空老师区，符合真实数据形态。
5. **无下拉刷新**：`pages.json` 未对本页 `enablePullDownRefresh`，源码只在 `onLoad` 拉一次；原型不画下拉刷新，重试是唯一恢复入口。
6. **头像仍是首字占位**：接口 `avatar` 可能为空且原型禁外链；落地时若有真实头像应替换，保留首字兜底。
7. **禁用态覆盖有限**：页面没有表单/开关，只有重试按钮可降级；已如实覆盖并在 notes 说明，没有为了凑「禁用态」发明控件。
8. **徽记「小一」与班级名重复**：这是刻意的姓名牌式缩写记号（与班级名形成主次），不是额外数据；若审阅觉得冗余，可换成幼儿园线稿图标。

## 验收

- `node scripts/od-check.mjs …/21-class-index/index.html --width 390` → `problems: []`（`scrollWidth 390 = clientWidth 390`，`overflowing: []`，无 console error）
- `node scripts/od-shot.mjs … --width 390 --dsf 2` → `preview.png`（390 × 1000，105,175 bytes）
- `node scripts/od-audit.mjs docs/design-previews/2026-10-25-full-redesign` → 本页无 ✗（无裸色值、无 nowrap、无 `.mp-*` 重定义、已链接 `../../_shared/shell.css`、无 `_shared/` 副本、无 `.mcp.json`）
- 视觉复核（mcp-vision 读 `preview.png` 及空/加载/失败三态截图）：状态栏 + 导航栏 + 返回箭头 + 胶囊齐全；**无 tabBar**；三卡层级清晰、文字无裁切、无横向滚动；头像为正圆；无直角；暖色只做点缀；「原型演示」未遮挡内容
- 交互实测（CDP）：入口 toast、重试 1 秒降级与恢复、空态重新加载、芯片切换、返回 toast 均正常；console 无 error/warn
- 未修改 `teacher/` 与 `parent/` 源码；输出目录无 `.mcp.json`、无 `_shared/` 副本、未复制 `shell.css`
