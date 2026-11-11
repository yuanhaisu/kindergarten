# Kinderly DNA 家长端验证集（5 页）

- status: prototype
- date: 2026-11-11
- pages:
  - parent/pages/index/index
  - parent/pages/pickup/index
  - parent/pages/growth/index
  - parent/pages/leave/create
  - parent/pages/growth/detail
- baseline: 35acd7b
- author: yhs

## 问题

设计大改方向定为「从 Kinderly / Kin 提取设计 DNA，再基于 DNA 重做 UI」。DNA 已在 `docs/design-dna/2026-11-11-kinderly/` 定稿，但还没在我们自己的信息密度和业务文案上验证过：衬线标题、暖纸底、行内高亮标签、悬浮胶囊导航这几条规则，放进家园沟通场景是否仍然成立，不能只靠参考产品的截图判断。

## 目标 / 非目标

- 目标：用 5 个代表性页面验证 DNA 的可迁移性，产出可评审的原型与截图。
- 非目标：不改生产代码、不改 `tokens.json`、不接入 design-sync 门禁、不覆盖全部 29 页。

## 验证集选择

| # | 页面 | 验证什么 |
|---|---|---|
| 01 | 首页 | 多卡片信息流 + 快捷入口 + 深色强调卡 |
| 02 | 接送 | 状态驱动页 + 两步进度轨 + 二维码入口 |
| 03 | 成长列表 | 图文 feed + 行内高亮标签 + 筛选胶囊 |
| 04 | 请假表单 | 表单密度、日期行、聚焦流程是否隐藏全局导航 |
| 05 | 成长详情 | 长文衬线正文 + 评论 + 大图 |

## 验收标准

- 5 页在 390×844 下无横向溢出，主操作可见或可滚动到。
- 标题实际渲染为衬线（中文回落到宋体类），正文 UI 保持无衬线。
- 悬浮胶囊导航与参考产品一致：白色胶囊、sage 激活底、soft shadow。
- 行内高亮标签不是胶囊 chip。
- 每页 `preview.png` 与原型一致。

## 设计改动

- 新批次：`docs/design-previews/2026-11-11-dna-validation/`
- 共享样式：`_shared/dna.css`（DNA token + 组件 + 手机壳）
- 画廊：`index.html`（5 页 iframe 预览）

## 评审记录

- 用 mcp-vision 对 5 页各做了一轮批判性评审，修复了：
  - 首页状态块过挤、FAB 与「去完成」碰撞（移除首页 FAB）
  - 接送页进度轨连线与圆点错位（改为 `calc(50% - 6px)`）
  - 请假页提交按钮在首屏之外（表单页隐藏全局导航、压缩行高）
  - 请假页类型/日期/事由不自洽（改为病假 1 天、字数 18/200）
  - 成长页照片占位被误读为坏图（加「示例照片」标注）
  - 详情页大图过高（220 → 180）

## 差异

- 未使用真实照片，统一用渐变占位并明确标注「示例照片」。
- 未覆盖弹窗、下拉刷新、空态、错误态；这些留给接入真实页面时验证。

## 后续

1. 方向确认后，把 5 页中的规则提炼成 token 变更（`tokens.json` + 主题文件），再决定是否全量铺开。
2. 接入 design-sync 前，先完成批次冻结与 `BATCH` 参数化（见 `design-redesign-dna-plan`）。
