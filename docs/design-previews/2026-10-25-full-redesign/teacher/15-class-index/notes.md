# 我的班级（teacher）

> 生成方式：手写（OpenDesign 两次失败：两次 run 都只产出 index.html，styles.css / app.js 始终缺失；第二次 run 51s 早停）

- 源码：`teacher/src/pages/class/index.vue`（接口 `teacher/src/api/teacher-class.js`、`teacher/src/api/teacher-pickup.js`）
- 设计系统：`user:nurture-planet-local`，`surface=teacher`
- 方向：把「班级信息 + 今日接送 + 全班名单」压成一屏可扫读的班级主页——顶部紧凑渐变 hero 交代「哪个班、谁带的、多少人」，接送概览只做四格数字，主体留给 3 列幼儿网格；点孩子只弹只读简要卡，不新增任何写操作。

## 信息层级

1. 微信外壳：状态栏 + 导航栏 + 共享 4 项底栏（本页「班级」高亮）；返回箭头回上一页，不写死目的地
2. 班级 hero（全页唯一渐变）：班级徽记、`name`、`grade · academic_year`、`kindergarten_name`、老师行（`teacher_name · title_label`）、人数胶囊、右上铃铛入口
3. 今日接送概览：日期 + 四列（已入园 / 未入园 / 已接走 / 未接走），数字来自 `GET /teacher/pickup/overview`
4. 最新通知：`GET /teacher/notices`（page_size=2）的两条通知 + 已读率进度条，进度条宽度按 `read_rate` 百分比取值
5. 幼儿名单：区头计数 + 搜索（仅姓名）+ 四段筛选（全部 / 未入园 / 未接走 / 已接走，带计数）+ 3 列网格
6. 幼儿简要信息卡（bottom sheet，只读）：头像、姓名、状态 pill、入园时间、接送状态、授权接送人、接送截止（班级级单一值）
7. 原型演示条（共享 `mp-demo`，默认收起）

## 覆盖状态

- 正常：hero 30 人；概览 28 / 2 / 6 / 22；最新通知 2 条（已读 32% / 64%）；名单 30 张卡（与 6 + 22 = 28、28 + 2 = 30 自洽）。
- 空态（名单为空）：hero 人数 0、概览四个数字 0、名单计数 0；工具行与网格隐藏；显示「班里还没有幼儿档案」+「刷新名单」；最新通知与名单数据独立，空态下仍保留。
- 加载：hero 骨架条；概览数字换骨架；最新通知换 2 行骨架；工具行保留但 `opacity:.42` + `grayscale(1)` + `pointer-events:none` + `aria-disabled="true"`，筛选计数隐藏（计数尚未返回）；名单位置 6 张骨架卡。
- 失败：hero / 概览 / 最新通知 / 名单全部隐藏，中央「班级信息加载失败 / 网络好像开小差了 / 重试」；重试先进入加载态，800ms 后回正常。
- 客户端空结果：搜索/筛选无命中时显示「没有找到符合条件的孩子 / 清除筛选」，与后端空列表区分。

## token 映射

- 页面底 `var(--bg)`；卡片 `var(--surface)`；hero `var(--accent-gradient)` + `var(--on-brand)`。
- 圆角：hero `var(--radius-2xl)`、卡片 `var(--radius-card)`、幼儿卡/搜索 `var(--radius-xl)`、胶囊 `var(--radius-pill)`；无直角。
- 阴影只用 `var(--shadow-sm)` / `var(--shadow-md)` / `var(--shadow-lg)`。
- 语义色：已入园 `var(--success)`、未入园 `var(--danger)`、已接走 `var(--info)`、未接走 `var(--warning)`，浅底用对应 `-wash`。
- 文字：`var(--fg)` / `var(--fg-2)` / `var(--muted)`；字号/字重全部走 `--text-*` / `--weight-*`。
- 最新通知卡：`var(--surface)` + `var(--radius-xl)` + `var(--shadow-sm)`；图标浅底 `var(--accent-wash)` / `var(--accent-deep)`；进度条轨道 `var(--border)`、填充 `var(--accent)`；文字 `var(--text-base)` / `var(--text-xs)`。
- 品牌色只出现在 hero、选中筛选、主按钮、头像浅底、通知图标与进度条；半透明文字/描边均由 token 经 `color-mix` 派生，无裸色值。

## 与现状的差异

- 源码在班级信息之后还有「班级动态」三快捷入口（发布通知 / 记成长 / 接送）与「最新通知」列表。快捷入口归工作台，原型不画；「最新通知」带 `read_rate` 已读率，消息页没有等价的百分比视图，本版补在概览与名单之间（两条示例与消息页同源：9/28≈32%、18/28≈64%）。右上铃铛仍作为去消息页入口。
- 接送截止按后端事实画成**班级级单一值**（16:30，放在 `.mp-app[data-pickup-deadline]`），不按学生取值；源码 `ClassStudents` 复用接送查询，每个学生拿到的是同一个 deadline。
- 源码幼儿卡片本身无点击行为；原型新增只读简要卡，字段全部来自学生对象（`arrive_time`、`pickup_status`、`authorized_persons`，`pickup_deadline` 取班级级值），不放任何改数据按钮。
- 源码名单是纵向列表；原型改为 3 列网格卡，便于一屏看到更多人。
- 源码的搜索只按姓名，原型保持；后端不返回学号，未画学号搜索。
- 源码 `BottomNav`（首页 / 班级 / 消息 / 我的）在本页有渲染；2026-10-27 已按产品决定补成共享 `mp-tabbar`，本页「班级」高亮。

## 验证

- `od-check --width 390`：`problems: []`，`overflowing: []`，`scrollWidth = clientWidth = 390`；滚动 400px 后 `statusbar top=0 / navbar top=20`，吸顶正常。
- 交互实测（headless Chrome + CDP，21 项断言全过）：30 张卡渲染；筛选「未入园」= 2（豆豆/皮皮）；筛选 + 搜索「豆」= 1；搜索无命中显示空结果块；清除筛选恢复 30；点「乐乐」信息卡字段与卡片 `data-*` 一致；Esc 关闭并恢复 body 滚动；未入园孩子入园时间显示「—」；铃铛 toast 1.6s 后消失；空 / 加载 / 失败三态显示正确；重试先 loading 后 normal 且 chip 同步；卡片/胶囊无直角；hero 使用渐变；控制台无 error / warning。
- P1 修复后复测（headless Chrome + CDP，19 项断言全过）：返回箭头 toast「返回上一页」；最新通知 2 条 + 进度条宽度 32% / 64% 与文案一致；通知卡点击 toast；概览/卡片/信息卡「已接走」同色（`--info`）；逐个打开 30 张卡，接送截止统一 16:30；筛选胶囊与「清除筛选」高度 44px；加载态通知换 2 行骨架、失败态隐藏通知区、空态保留通知区；筛选 / 搜索 / 清除沿用原逻辑；控制台无 error。
- 四态截图人工复核：空态与失败态布局正常；加载态 hero/概览/通知/名单均为骨架，工具行灰色禁用且不显示真实计数。
- `node scripts/od-audit.mjs docs/design-previews/2026-10-25-full-redesign`：本页无 ✗；提示 `index.html 23KB`、`styles.css 21KB` 超过 20KB（批次内另有多页同类提示，本页体积来自 30 张幼儿卡的静态样例数据与通知块样式）。
- 页面目录无 `.mcp.json`、无 `_shared/` 副本；`styles.css` 不含 `.mp-*` 定义。

## 遗留 / 需产品确认

1. **老师端底部导航已补**：`pages.json` 仍无 `tabBar`，但 `class/index.vue` 渲染了自绘 `BottomNav`（4 项）。2026-10-27 已按产品决定在 01/15/18/19 四页补共享 `mp-tabbar`；落地前需把源码的自绘底栏抽成统一组件或改 `pages.json` 的自定义 tabBar，避免两套实现。
2. **多老师展示**：`backend/api/api-teacher/09-class-and-tasks.md` 的示例是 `teachers[]`（主班/副班/生活），实现 `GET /teacher/class/info` 返回的是单个 `teacher` 对象。原型按实现画一位老师；若产品要展示副班/生活老师，需要后端先补数组。
3. **逾期未接**：学生对象有 `is_overdue`，概览有 `overdue_not_picked` / `abnormal`，但源码班级页未使用。原型没有新增逾期标记；若需要，应在批次层面统一逾期文案与颜色（不要新造第四种状态色）。
4. **头像**：真实 `avatar` 可能是 URL；原型不引外部资源，一律用姓名首字圆形底。接入时替换为图片即可，尺寸与圆角 token 已定。
5. **授权接送人**：原型把 `authorized_persons` 当展示字符串；若后端返回对象数组，需要前端先拼成「妈妈、爷爷」再填入信息卡。
6. 名单总数与 `student_count` 在原型里都取 30；真实数据若两者不一致，建议区头以列表 `total` 为准，hero 人数以 `student_count` 为准。
7. **已读率口径**：本页按 `read_rate` 百分比展示，消息页按「已读 9/28」计数展示；两页示例数据已对齐（9/28≈32%、18/28≈64%）。消息页 `recipient_count=28` 与班级 30 人仍不一致，属批次级演示数据口径，需统一后再接真实数据。
