# 家长端全量设计稿 · 生产约定

本文件是给页面作者（人或 agent）的硬约束。动手前必须读：

1. `docs/design-dna/2026-11-11-kinderly/DESIGN.md`（DNA v1 冻结规则）
2. 本目录 `_shared/dna.css`（唯一共享样式，禁止改）
3. 基准页：`01-home/`、`11-growth/`、`17-pickup/`、`31-profile/`、`02-login/`

## 文件结构

- 每页一个目录：`NN-slug/index.html`，`<link rel="stylesheet" href="../_shared/dna.css" />`
- 页面特有布局写在 `<style>` 块里；**禁止**在页面里重定义 token 变量或覆盖共享组件
- 不引用外部字体 / 网络图片 / 任何 CDN；字体由 dna.css 本地加载
- 列表页空态：同目录 `empty.html`，复制外壳，只替换内容区为 `.empty`
- 状态变体：用 `?state=xxx` + `[data-state]` + 页尾小脚本（参考 `17-pickup`），不新增重复文件

## 手机外壳

- `.mp-app` → `.mp-statusbar` → `.mp-main`（tab 页加 `.mp-main--tab`）→ `.mp-nav`
- 直接复制 `01-home/index.html` 的状态栏与导航；tab 页只有 4 个：01 首页 / 11 成长 / 17 接送 / 31 我的
- 非 tab 页：`.page-bar`（返回圆钮 + 页面小标题 + 右侧占位）然后 h1；**没有** `.mp-nav`
- 宽度 390，页面水平内边距 20（`.mp-main` 已内置），不得出现横向滚动

## 排版（冻结值，不得改）

- H1：衬线 400，30px / 1.35 / 字距 0；一屏只有一个 h1
- H2：衬线 400，21px / 1.4 / 0；卡片或区块标题
- H3 / 列表标题：衬线 400，18px；设置行、记录行标题用无衬线 15px / 500–600
- 正文：15px / 1.65；衬线长正文：17px / 1.8 / 0（仅反思性长文）
- 次要文字：12.5–13px；说明 11.5px；眉题 11px / 0.14em / 600
- **衬线永远 400**，任何衬线文字不得加粗
- 大数字用 `.num`（衬线 400），不强制字距

## 颜色（文字安全值）

| 用途 | 值 |
|---|---|
| 画布 / 卡片 | `--bone` / `--card`（白）/ `--card-soft` |
| 正文与主色 | `--pine` |
| 次级文字（bone 上） | `--ink-bone` #55616F |
| 次级文字（白卡上） | `--ink-card` #5E6B7A（`.card` 内自动生效） |
| `--ink-soft` #728173 | **只用于图标 / 分隔线 / ≥24px 大字**，不得作小字 |
| `--ink-blue` | 只做装饰，不作文字 |
| 赤陶文字 | `--terracotta-text` #A64E2E；`--terracotta` 只做图标/插画/软底 |
| 橙色 | 只做角标、圆点、warning 图标与 12% 软底；橙底文字用松绿或 `--orange-text` |
| 状态标签 | `.tag--ok/--wait/--muted/--danger/--abnormal/--info/--solid` |

## 组件（只用 dna.css 已有类）

`.card` `.card--soft/--pine/--sage/--flush/--tight`、`.btn`（`--primary/--ghost/--soft/--text/--danger/--block/--lg/--sm`）、`.chip`（`.is-on`）、`.seg`、`.field`/`.input`/`.picker`/`.input--error`、`.tag`、`.badge`、`.avatar`、`.orb`、`.list-row`、`.hairline`、`.hl`（行内高亮）、`.banner--warning/--info/--danger`、`.empty`、`.sk`、`.bar`、`.steps3`、`.sheet`/`.scrim`/`.toast`、`.thumb--sage/gold/clay/blue/pine`。

- **内容标签不用胶囊 chip**：用 `.hl` 行内高亮（这是 DNA 签名特征）
- 筛选才用 `.chip`；分段切换用 `.seg`
- 卡片无阴影，靠 1px 发丝线与明度差；唯一阴影是悬浮导航
- 危险操作用独立确认区或底部抽屉，不用居中弹窗
- 图标：内联 SVG，24 网格，`fill="none" stroke="currentColor"`，线宽 1.6–2.0

## 文案

- 真实业务语，中文，面向家长；不写 lorem ipsum、不写英文占位
- 称呼：乐乐（小一班，5 岁）、李老师、王老师、妈妈（李静）
- 语气克制、不施压（DNA：疲惫状态下的家长）
- 日期统一 2026 年 11 月 12 日（周三）前后

## 页面清单与内容要求

| 目录 | 页面 | 内容要求 |
|---|---|---|
| 03-bind | 绑定孩子 | 姓名、关系 picker、绑定码 6 位、隐私提示；主 CTA 绑定；无绑定码的兜底说明 |
| 04-consent | 隐私政策与用户协议 | 三条同意项（隐私政策 / 儿童个人信息规则 / 用户协议）带勾选与查看；同意并继续；暂不同意 |
| 05-policy | 政策全文 | 长文阅读页；更新日期；6 个章节；无卡片容器，正文 15px；衬线导语 |
| 06-notice-index | 通知列表 | 置顶通知 + 列表；全部/未读/园所/班级筛选；未读点；**empty.html** |
| 07-notice-detail | 通知详情 | 类型眉题、标题、发布人、正文、附件卡、已读人数、报名 CTA |
| 08-leave-index | 请假记录 | 新建入口、本月统计、状态筛选、请假卡片（类型/时间/事由/状态）；**empty.html** |
| 09-leave-create | 新建请假 | 类型 seg、起止日期 picker、事由 textarea（字数）、凭证上传；`?state=error` 展示校验错误 |
| 10-leave-detail | 请假详情 | 状态卡 + 审批时间线（提交/老师/园长）、事由、凭证、撤回 |
| 12-growth-detail | 成长详情 | 大图、标题、正文、照片网格、老师寄语、评论列表、点赞条 |
| 13-growth-album | 成长相册 | 按月分组照片网格 3 列、照片/视频筛选、总数 |
| 14-growth-report | 成长报告 | 学期报告；四个发展维度进度；老师寄语；高光照片；导出按钮 |
| 15-album-index | 班级相册 | 精选相册大卡 + 相册网格 2 列（封面/标题/张数/日期）；**empty.html** |
| 16-album-detail | 相册详情 | 相册头、照片网格 3 列、下载/分享操作、拍摄日期 |
| 18-pickup-qrcode | 接送二维码 | 大二维码占位（内联 SVG）、孩子姓名、6 位码、刷新倒计时、使用说明、一次有效提示 |
| 19-pickup-authorized | 授权人管理 | 授权人列表（头像/姓名/关系/手机/状态）、新增按钮、说明；**empty.html** |
| 20-pickup-authorized-create | 新增授权人 | 姓名、关系 picker、手机号、有效期、照片上传、提交审核、隐私说明 |
| 21-tasks | 亲子任务 | 今日任务卡（虚线三段进度）、待完成/已完成筛选、任务行含积分；**empty.html** |
| 22-tasks-detail | 任务详情 | 标题、积分、说明、步骤清单、打卡按钮、我的提交记录、照片 |
| 23-tasks-wall | 班级墙 | 2 列打卡动态（照片/孩子名/任务/点赞）、筛选、今日参与数 |
| 24-points | 积分与成就 | 松绿大卡显示可用积分、明细列表、成就徽章网格（含未解锁）、规则入口 |
| 25-class | 班级信息 | 班级、人数、教室、两位老师卡片（可联系）、班级公告 |
| 26-calendar | 班级日历 | 月历网格（事件圆点）、选中日期事件卡、图例、下月活动 |
| 27-student | 幼儿信息 | 档案头（头像/姓名/班级/年龄）、基本信息、健康信息、紧急联系人、编辑入口 |
| 28-feedback | 意见反馈 | 类型 chips、textarea、照片上传、联系方式、提交、常见问题卡 |
| 29-settings | 设置 | 消息通知、隐私与协议、账号安全、关于；行尾值/chevron；退出登录 |
| 30-messages | 消息 | 全部/系统/老师 seg、消息行（头像/标题/摘要/时间/未读点）；**empty.html** |

## 自检（交付前必须做）

```bash
node scripts/od-shot.mjs "docs/design-previews/2026-11-12-parent-dna-full/<NN-slug>/index.html" \
  "docs/design-previews/2026-11-12-parent-dna-full/<NN-slug>/preview.png" --width 390 --height 844 --dsf 2
```

- 输出 JSON 必须 `cssWidth: 390`、`cssHeight: 844`
- 自查：无横向溢出、无文字被裁切、无 <400 字重的衬线、无小字用 #728173、无橙色文字压在橙色上
- 页面里不出现 `TODO`、`lorem`、`示例` 字样

## 交付报告

列出创建的文件；标出该页的特殊设计决策（≤3 条）；标出任何未覆盖状态。
