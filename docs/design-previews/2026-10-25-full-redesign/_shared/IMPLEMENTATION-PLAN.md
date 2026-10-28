# 全量设计落地切分规划

> 范围：`docs/design-previews/2026-10-25-full-redesign/` 的 51 页在 `teacher/` 与 `parent/` 的落地。
> 基线：`77c1b90`（4 个试点页已落地；老师端通知列表在此前批次已落地）。
> 状态：规划稿，尚未开始执行。

## 0. 已完成基线

| 端 | 设计页 | 代码页 | 说明 |
| --- | --- | --- | --- |
| 老师 | 01-workbench | `teacher/src/pages/index/index.vue` | 试点页，已 push |
| 老师 | 03-notice-create | `teacher/src/pages/notice/create.vue` | 试点页，已 push |
| 老师 | 23-notice-index | `teacher/src/pages/notice/index.vue` | 此前批次已落地，待 V01 回归复核 |
| 家长 | 12-growth-report | `parent/src/pages/growth/report.vue` | 试点页，已 push |
| 家长 | 14-pickup-qrcode | `parent/src/pages/pickup/qrcode.vue` | 试点页，已 push |

剩余：老师端 20 页、家长端 26 页，共 46 页。

## 1. 切分原则

1. **一片一验收**：每片独立可构建、可在 DevTools 打开、有明确验收页与关键交互。
2. **共用件先行**：图标、页面骨架、状态块、CSS 兼容写法先统一，避免 46 页重复踩坑。
3. **先主路径后长尾**：先两端 tab 根页与最高频模块，再列表/表单，最后低频工具页与边界页。
4. **后端缺口独立成片**：UI 片不夹带契约变更；缺口未补前页面用 `—`、隐藏或降级占位。
5. **一片 1–3 页**：超过 3 页拆开；共享文件（`shared-ui/`、`pages.json`、token）一次只由一个片改。
6. **共享组件只改 `shared-ui/`**：改完跑 `sync-ui.js`；不要直接编辑两端 `components/`（会被覆盖）。
7. **每片只 stage 本片文件**：工作区已有大量与本轮无关的脏文件，不得带入提交。

## 2. 统一 Definition of Done

每片完成必须同时满足：

- [ ] 页面路由已存在于 `pages.json`，未擅自新增页面；若必须新增，先走 OpenSpec。
- [ ] `cd teacher && npm run build:mp-weixin` 或 `cd parent && npm run build:mp-weixin` 通过。
- [ ] `node scripts/check-tokens.mjs teacher|parent` 无新增违规。
- [ ] DevTools 正常态：真实数据渲染，console 0 error。
- [ ] 空态、错误态至少各验一次（断网或停后端）。
- [ ] 关键交互真点击验证（tab 切换、返回、提交、弹层），不能只用 MCP 导航代替。
- [ ] 跳转符合 `pages.json`：家长端 tab 页用 `switchTab`；老师端无 tabBar，不存在 `switchTab`。
- [ ] 本片一个 commit（或一组同范围 commit），message 写清页面范围。
- [ ] 不提交 `configs/app.local.ini`、`.env`、凭据、`.mcp.json`。

## 3. 依赖关系

```
F0 ──┬─> W1 主路径 ──> W3 业务模块 ──> W4 低频页 ──> W5 登录/专科 ──> W6 边界页 ──> V01 ──> V02
     ├─> B01/B02（可与页面片并行）；B04 必须先于 P02；B05 必须先于 P01
     └─> B03（独立，需 OpenSpec）

X00（零入边决策）─> P11 / P12 / X01 / X02
D01 在 W3 完成后做；D02 每片顺手修，W7 终检。
```

## 4. 切片总表

| ID | 波次 | 范围（设计页 → 代码页） | 页数 | 规模 | 前置 | 备注 |
| --- | --- | --- | --- | --- | --- | --- |
| F0 | W0 | 落地基线：AppIcon / PageShell / StateBlock / CSS 兼容手册 | - | L | 无 | 只改 `shared-ui/`；试点 5 页回归不回退 |
| X00 | W0 | 零入边页面决策：老师 webview、家长 bind / messages / webview | - | S | 无 | 产品决策：补入口、隐藏或保留深链 |
| P01 | W1 | 家长 01-home → `pages/index/index.vue` | 1 | M | F0、B05 | 首页四入口、今日卡片、tabBar |
| P02 | W1 | 家长 09/10/11 → `growth/index`、`growth/detail`、`growth/album` | 3 | L | F0、B04 | 点赞态以 B04 结论为准；复用 report 的日期/维度处理 |
| P03 | W1 | 家长 13/15/16 → `pickup/index`、`pickup/authorized`、`pickup/authorized-create` | 3 | L | F0 | 复用 qrcode 的关系映射；D01 前允许先复制 |
| P04 | W1 | 家长 27/25 → `profile/index`、`settings/index` | 2 | M | F0 | 设置项、退出登录 |
| T01 | W1 | 老师 04/05/06 → `growth/index`、`growth/create`、`growth/detail` | 3 | L | F0 | 与既有「发布工作台」改造合并，不得回退已验证行为 |
| T02 | W1 | 老师 07/08 → `pickup/index`、`pickup/records` | 2 | M | F0 | 接送码、记录状态 |
| B01 | W2 | 成长报告四格统计 | - | M | 无（页面 12 已存在） | Go service + `backend/api/api-parent/` + `report.vue`；触点 ≥3 层，走 OpenSpec |
| B02 | W2 | 班级家长人数 | - | S | 无（页面 03 已存在） | `ListTeacherClasses` + `backend/api/api-teacher/` + `notice/create.vue`；走 OpenSpec |
| B03 | W2 | 草稿编辑定时/附件 | - | M | 无 | `UpdateNotice` 补字段；走 OpenSpec；可选，需产品确认 |
| B04 | W1* | 点赞落库复核 + 两端防连点 | - | S | 无 | **提前到 P02 之前**；先复现 POST true / GET false；确认后按 systematic-debugging 修 |
| B05 | W1* | 首页 feed 按 student_id + relation + 契约键名 | - | S | 无 | **提前到 P01 之前**；修复多孩串数据 |
| P05 | W3 | 家长 04/05 → `notice/index`、`notice/detail` | 2 | M | F0 | 已读/未读、确认回执 |
| P06 | W3 | 家长 06/07/08 → `leave/index`、`leave/create`、`leave/detail` | 3 | L | F0 | 与老师请假 T04 共用类型映射 |
| P07 | W3 | 家长 17/18/19 → `tasks/index`、`tasks/detail`、`tasks/wall` | 3 | L | F0 | 打卡态、任务墙 |
| P08 | W3 | 家长 20/24 → `points/index`、`feedback/index` | 2 | M | F0 | 积分流水、反馈提交 |
| P09 | W3 | 家长 21/23 → `class/index`、`student/index` | 2 | M | F0 | 班级信息、幼儿档案 |
| T03 | W3 | 老师 09/10 → `tasks/index`、`tasks/detail` | 2 | M | F0 | 发布任务、查看打卡 |
| T04 | W3 | 老师 14 → `leave/index` | 1 | S | F0 | 审批流 |
| T06 | W4 | 老师 15/16 → `class/index`、`album/index` | 2 | M | F0 | 班级相册 |
| T07 | W4 | 老师 11/12/13 → `calendar/index`、`calendar/create`、`calendar/detail` | 3 | L | F0 | 事件类型、日期选择 |
| T08 | W4 | 老师 17/19/18 → `more/index`、`profile/index`、`messages/index` | 3 | M | F0 | 更多聚合入口 + 消息 |
| P10 | W4 | 家长 22 → `calendar/index` | 1 | M | F0 | 只读日历 |
| P11 | W4 | 家长 26 → `messages/index` | 1 | S | F0、X00 | 零入边，先定入口 |
| T09 | W5 | 老师 02 → `auth/login` | 1 | S | F0 | 静默登录已是现状，主要视觉 |
| T10 | W5 | 老师 20/21 → `doctor/checkin`、`doctor/calendar` | 2 | M | F0 | 保健医角色入口 |
| P12 | W5 | 家长 02/03 → `auth/login`、`auth/bind` | 2 | M | F0、X00 | 绑定手机号流程 |
| X01 | W6 | 老师 22 → `webview/index` | 1 | S | F0、X00 | 个人主体不可用：只做空态/错误态 + 隐藏入口 |
| X02 | W6 | 家长 28 → `webview/index` | 1 | S | F0、X00 | 需业务域名；未配置则空态 |
| V01 | W7 | 51 页遍历回归 + 真实交互抽检 + 真机 | - | L | 全部页面片 | MCP navigate 只验渲染；交互用 automator / `scripts/cdp.py` |
| V02 | W7 | 截图视觉比对与差异清单 | - | M | V01 | 截图链路若仍失败，改 DOM/CSS 断言 + 人工真机截图 |
| D01 | W7 | 共享逻辑收口：StateBlock / relationLabels / normalize / PageShell | - | M | W3 完成 | 只改 `shared-ui/` + 引用页 |
| D02 | W7 | token/样式债清理：裸 rpx、硬编码色、reduced-motion | - | M | 全部页面片 | 每片顺手修，本片终检 |

合计 34 片（含后补的 B05），覆盖 46 个剩余页面，无重复、无遗漏。

## 5. 关键片说明

### F0 落地基线

产出四件事：

1. `AppIcon`：统一图标入口，优先 iconfont 或 PNG 资产；WXML 不支持内联 SVG。先覆盖工作台 8 入口 + 导航/状态图标。
2. `PageShell`：原生导航下的页面 padding、底部安全区、`BottomNav` 占位约定。
3. `StateBlock`：loading / empty / error / retry 四态统一。
4. `CSS-COMPAT.md`：`color-mix`、`dvh`、`aspect-ratio`、`sticky`、`backdrop-filter`、`scroll-snap`、flex gap、内层滚动的替代写法。

验收：试点 5 页不回退；两端 build；`check-tokens` 无新增违规。

### B01 / B02 / B03

三者都同时改 Go service、`backend/api/` 契约、端页面，触点 ≥3 层，按 `CLAUDE.md` 应走 OpenSpec：

```
/opsx:propose → 用户批准 → /opsx:apply → 验证 → /opsx:archive
```

- B01：`GetGrowthReport` 增加出勤/积分/任务/成长记录统计；未完成前 `report.vue` 四格保持 `—`。
- B02：`ListTeacherClasses` 增加家长人数字段；未完成前通知页不显示「N 位家长」。
- B03：`UpdateNotice` 落 `scheduled_at` 与附件；编辑草稿页目前隐藏这两项。若产品不要求编辑定时/附件，可删掉此片。

### B04 点赞

先复现，不先改：

1. 家长端对同一条成长记录点赞 → 记录返回的 `id`。
2. 立即 GET 列表/详情，看 `is_liked` 是否仍为 false。
3. 若确认不落库，按 `superpowers:systematic-debugging` 定位后端；同时给两端 `toggleLike` / `handleLike` 加防连点。

### P02 / P03 / P06 / P07 复用边界

- P02 复用 `growth/report.vue` 的日期本地化、维度排序；不要重复实现。
- P03 复用 `pickup/qrcode.vue` 的 `relationLabels`、状态胶囊；D01 再收口到 `shared-ui/`。
- P06 与 T04 共用请假类型/状态映射；先由 P06 定义，T04 引用。
- P07 与 T03 共用任务状态/打卡态；先由 P07 定义，T03 引用。

### T01 与既有改造合并

`teacher/src/pages/growth/index.vue` 已有「发布工作台」首屏改造（今天条、快速记录、搜索下移、删 FAB）。本片按设计 04 重做时必须保留这些已验证行为，不得回退；若设计稿与既有行为冲突，以设计稿为准并在 commit 说明差异。

### X00 / X01 / X02

- X00 先出结论：4 个零入边页面各自「补入口 / 隐藏 / 保留深链」。
- 老师端个人主体：`<web-view>` 不可用，X01 只做空态/错误态，并隐藏入口或标注「暂不可用」。
- 家长端非个人主体：`<web-view>` 条件可用，但需业务域名；X02 先确认域名配置，未配置则空态。

## 6. 风险与决策点

| # | 风险 | 影响 | 应对 |
| --- | --- | --- | --- |
| 1 | 截图链路仍失败（IDE 合成层） | V02 无视觉证据 | 改用 DOM/CSS 断言 + 真机人工截图；V02 前先修 `scripts/shot.py` 或接受降级 |
| 2 | 老师端 webview 不可用 | X01 无法交付真实页面 | 产品接受空态；入口隐藏或标注 |
| 3 | 家长端 webview 业务域名未配置 | X02 无法交付真实页面 | 先查配置；未配置则空态并记录 |
| 4 | 后端字段缺失 | B01/B02 阻塞页面完整度 | 按 OpenSpec 补契约；未补前 `—`/隐藏 |
| 5 | 共享组件被 `sync-ui.js` 覆盖 | 改动丢失 | 只改 `shared-ui/`，改完同步 |
| 6 | 工作区脏文件多 | 误提交 | 每片只 stage 本片路径；提交前 `git status` 核对 |
| 7 | 后端被系统低内存保护停止 | 验证中断 | 验证前确认 8080；必要时重启 `bcxw6cwn2` |
| 8 | 一次并行多个片改共享文件 | 冲突/回退 | 共享文件一次只由一个片改；页面片可并行但需错开共享文件 |

## 7. 建议执行节奏

1. 先做 F0 + X00，一次把基线与边界决策定掉。
2. W1 按 B04 → B05 → P01 → P02 → P03 → P04 → T01 → T02 顺序做（B04 前置 P02，B05 前置 P01）；每片一个 commit。
3. W2 后端片可与 W1 交错，但 B03 需先走 OpenSpec。
4. 每波结束跑一次：两端 build、`check-tokens`、本波新增页面遍历、真实交互抽检。
5. W3 完成后再做 D01，避免过早抽象。
6. 全部页面片完成后做 V01 → V02 → D02 终检。

## 8. 仓库级遗留（不纳入本轮切片）

- 历史泄漏的 RDS 凭据仍需轮换 + 清理 git 历史；在此之前仓库保持 private。
- 工作区现有 84 个与本轮无关的脏文件（`.claude/helpers` 删除、主题脚本、`package-lock.json` 等），不属于任何切片，不得带入提交。
