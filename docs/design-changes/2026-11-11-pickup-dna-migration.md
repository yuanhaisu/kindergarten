# 接送页 DNA 真实迁移验证

- status: implemented
- date: 2026-11-11
- branch: design/pickup-dna-migration
- pages:
  - parent/pages/pickup/index
  - parent/src/custom-tab-bar（共享底栏）
- baseline: 5634589
- author: yhs

## 问题

DNA 验证集只有 HTML 原型。原型里用了内联 SVG、语义标签、Google Fonts、`:root` 与固定 px，这些在微信小程序里都不成立。需要拿一个真实页面跑通，确认 DNA 在 WXML/WXSS 约束下仍然成立，并暴露必须提前决策的问题。

## 目标 / 非目标

- 目标：把接送页按 DNA 迁移到 uni-app，同时验证底栏、安全区、滚动留白、字体与状态轨。
- 非目标：不改接口与数据结构；不改其它页面；不动 `tokens.json`；不做全量迁移。

## 平台适配（相对原型）

| 原型 | 小程序实现 |
|---|---|
| 内联 `<svg>` 图标 | 沿用 remixicon 字体字形（页面原本就有） |
| `header/section/nav` 等语义标签 | 全部 `view/text` |
| Google Fonts（Instrument Serif / Noto Serif SC） | 系统字体栈：`Songti SC → STSong → Noto Serif CJK SC → serif`，无远程字体 |
| `:root` CSS 变量 | SCSS 局部变量，编译成静态值 |
| 固定 px | 全部 rpx（750 基准） |
| 每页自带悬浮导航 | 复用 `custom-tab-bar`，共享改成悬浮胶囊（`custom: true` 已开启） |
| `bottom: 16px` | `calc(8rpx + env(safe-area-inset-bottom))` |
| iframe/内层 overflow 滚动 | 页面级滚动，未引入 scroll-view |
| 阴影分层 | 卡片改为 1rpx 发丝线 + 白面 |

## 验证结果（真实开发者工具）

- **未入园**：状态卡显示「未入园 / 截至 16:57 未入园」，进度轨隐藏 —— 符合设计。
- **在园中**：状态「在园中」，进度轨「送入 07:55 · 家长小俊彦 → 接走 等待接走」，二维码 CTA 出现；字段与接口一致。
- **底栏安全区**：计算值 `bottom: 38px` = 8rpx(4px) + `env(safe-area-inset-bottom)` 34px，未压 home indicator。
- **滚动留白**：滚到底时 `ledger` 底部 647px < 底栏顶部 661px，不遮挡。
- **共享底栏**：首页切换后仍正常，激活项为「首页」，未破坏其它 tab。
- **未验证**：已接走状态（需要今天一条 type=2 记录）、Android 真机字形。

## 必须提前决策的问题

**中文衬线字体。** 模拟器里标题与状态落到了系统宋体，观感明显比参考产品差，也容易被误读为「字体没加载」。三个选项：

1. 自托管思源宋体子集（可控，但要处理包体与首屏）；
2. 放弃中文衬线，只保留拉丁/数字使用衬线（身份减弱但稳定）；
3. 只在高版本 iOS 使用宋体，Android 回落无衬线（身份两端不一致）。

在定这条之前，不建议把 DNA 铺到更多页面。

## 差异

- 原型里的 `page-title` 68rpx 与 `pass-state` 60rpx 在小程序里按 750 基准换算；桌面浏览器原型不可直接对照像素。
- 原型初版缺「历史记录」，却多了真实页面没有的二维码卡与「可接送人」。已按真实页面补齐历史、移除多余区块，并把二维码 CTA 收进凭证卡。
- 原型用渐变占位照片，接送页无图片，不涉及。

## 后续

1. 定字体策略。
2. 补「已接走」状态与异常记录的视觉验证。
3. 再决定是否把底栏 DNA 推广到其余 tab 页（当前已经是共享组件，改动已生效）。
