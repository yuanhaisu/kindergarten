# 已选方案（2026-11-14）

用户选择：**V2 / V3 / V4 三组提案进入默认方向**；V1 / V5 / V6 / V7 / V8 保留为备选，不替换 baseline。

| 页面 | 选中版本 | 路径 | 后端依赖 |
|---|---|---|---|
| 07 通知详情 | V3 报名提案 | `parent-variants/v3-notice-signup/07-notice-detail/` | 需新增报名 API/状态 |
| 21 亲子任务 | V2 任务积分提案 | `parent-variants/v2-task-points/21-tasks/` | `06-task` 恢复 `points` / `streak` |
| 22 任务详情 | baseline（已含 `+10`）/ 后续补 V2 版 | `parent/22-tasks-detail/` | 同上 |
| 24 积分与成就 | V2 任务积分提案 | `parent-variants/v2-task-points/24-points/` | 同上 |
| 29 设置 | V4 简化版 | `parent-variants/v4-settings-simple/29-settings/` | `settings` 增加免打扰字段 |

## 底部导航

- 「成长」tab 统一为 **V1 的叶子图标**（`M20 4C12 4 5 8 5 16c0 2 1 4 3 4 8 0 12-7 12-16Z` + `M5 20c2-5 6-8 11-10`）。
- 已覆盖有底部导航的页面：`parent/01`、`parent/11`、`parent/17`、`parent/31`；`parent-variants/v1/01`、`v1/17`、`v8/31`。
- 二级页没有底部导航，不需要改。

## Logo 使用

- 设计稿用 **DNA 版**：`_shared/logo-dna.svg`（pine `#173220` / marigold `#DDB14A` / cream `#FCFAF4`；几何沿用官方 logo，只换色）
- 官方 PNG `_shared/logo.png`（复制自 `parent/src/static/logo.png`）保留为品牌资产
- 02 登录：56×56，置于 `养成星球 · NURTURE PLANET` 眉题上方
- 02 登录中间图案：抽象轨道（松绿小星球无脸 + 赤陶细轨道 + marigold 星点），不再用笑脸星球
- 31 我的：页头右上角 28×28；页脚版本行 16×16
- V8 单孩子：页头 24×24（在“单孩子”角标左侧）+ 页脚 16×16
- 其他页面不重复放 logo；品牌 mark 只在登录与账号页出现，避免每页都贴标
- 移植时导出 `logo-dna.png` @3x；小程序 `<image>` 不支持 SVG

## 合并落点（parent-chosen/）

- `parent-chosen/`：31 页已选方案；07 / 21 / 24 / 29 用提案，其余用 baseline。
- 01 首页：10:20、通知角标 2、任务卡改“给家人讲一件今天开心的事 / 今天还没提交”、午睡改“12:30 开始”。
- 17 接送：10:20、可接送人改“妈妈、爸爸，共 2 位（外婆待审核）”、接送码说明改“当日有效，核验后失效”。
- 14 报告：页头加“乐乐”，本期积分改 +218。
- 31 我的：统计卡改“累计数据 / 32 成长记录”，补“消息”菜单；保留多孩子“切换”。
- 所有 chosen 页面状态栏统一 10:20。
- V1 / V5 / V6 / V7 / V8 作为状态/数据集并存，不替换主稿。
- 画廊：`chosen.html`（31 页 / 10 状态）；重建：`node _shared/build-gallery.mjs --chosen`。
- 截图：`node _shared/shots-chosen.mjs`（31 默认 + 10 状态）。

## 边界

- `parent/` 的 31 页 baseline **未整体替换**；当前是「baseline + 选中提案」并列。
- 若要把三组提案合并成新的 31 页主稿，需要另开一次合并（复制 07/21/24/29 到 `parent/` 或新建 `parent-chosen/`），并把 `BRIEF-REWRITE-PLAN.md` 的 baseline 改成提案口径。
- 选提案意味着后端契约要改：`06-task` 的 points/streak、通知报名 API、settings 的免打扰字段。移植前先改契约，不要把提案当现状。
