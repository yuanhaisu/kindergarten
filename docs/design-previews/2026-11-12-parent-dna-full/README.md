# 家长端全量设计稿（DNA v1）

- 日期：2026-11-12
- 基线：`docs/design-dna/2026-11-11-kinderly/DESIGN.md`（DNA v1，2026-11-12 冻结）
- 范围：`parent/src/pages.json` 的全部 **31 页** + 10 个状态变体（空态 / 接送三态 / 表单错误）
- 性质：**设计稿**。不改产品代码，不迁移 token；用户确认后才进入移植

## 怎么review

1. 打开根目录 `index.html`：全部分组的画廊（缩略图可点进单页）
2. 单页直接打开 `NN-slug/index.html`（390×844 手机壳，内部可滚动）
3. 状态变体：
   - 接送：`17-pickup/index.html?state=away` / `?state=done`
   - 新建请假：`09-leave-create/index.html?state=error`
   - 空态：各列表页的 `empty.html`
4. 每页 `preview.png` 为 390×844 @2x 的首屏截图

## 目录约定

- `_shared/dna.css` — DNA v1 的唯一共享样式源（token + 组件），页面不得重定义
- `_shared/CONTRACT.md` — 页面作者的硬约束（排版、颜色、组件、文案、自检）
- `_shared/build-gallery.mjs` — 生成根 `index.html`（`--check` 只检查缺失）
- `_shared/audit.mjs` — 静态审计（h1/导航/外链/占位文案/衬线字重/装饰色误用）
- `PLAN.md` — 批次计划与工作流

## 与旧批次的关系

`docs/design-previews/2026-10-25-full-redesign/parent/` 只有 29 页，且是旧视觉。
本批次只借用它的**内容结构**，视觉全部按 DNA v1 重绘，并补上 `auth/consent`、`auth/policy`。

## 已知边界

- 照片位是色块占位；插画为内联 SVG（120 网格风格），最终资产在移植阶段逐页确定
- 不做暗色模式（DNA v1 决策）
- 字体在本地加载 `_shared/fonts/` 下的两份 woff2，与小程序内 `wx.loadFontFace` 同源
