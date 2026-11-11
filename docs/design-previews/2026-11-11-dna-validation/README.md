# 2026-11-11 DNA 验证集 · 家长端 5 页

Kinderly / Kin 设计 DNA 在家园沟通场景下的可迁移性验证。**不是** 2026-10-25 批次的替代品，也不受 design-sync 门禁约束——它是设计领先的探索产物，方向确认后才会进入 token 与代码。

- DNA 规则：[`../../design-dna/2026-11-11-kinderly/DESIGN.md`](../../design-dna/2026-11-11-kinderly/DESIGN.md)
- 变更单：[`../../design-changes/2026-11-11-dna-validation.md`](../../design-changes/2026-11-11-dna-validation.md)

## 页面

| # | 页面 | 原型 |
|---|---|---|
| 01 | 首页 | [`parent/01-home/`](parent/01-home/) |
| 02 | 接送 | [`parent/02-pickup/`](parent/02-pickup/) |
| 03 | 成长列表 | [`parent/03-growth/`](parent/03-growth/) |
| 04 | 请假表单 | [`parent/04-leave-create/`](parent/04-leave-create/) |
| 05 | 成长详情 | [`parent/05-growth-detail/`](parent/05-growth-detail/) |

画廊：[`index.html`](index.html)

## 怎么用

直接双击 `index.html`，或单独打开任一 `parent/*/index.html`。页面按 390×844 的手机壳渲染，**桌面浏览器**下居中显示。

## 规则落点

- `_shared/dna.css` 是唯一的共享样式来源：颜色、字体、圆角、阴影、导航、卡片、按钮、胶囊、行内高亮。
- 每页 `styles.css` 只写页面独有的布局，不重复定义 token。
- 字体通过 Google Fonts 加载 Instrument Serif + Noto Serif SC；离线时回落到宋体，布局不受影响。

## 刻意不做

- 真实照片：统一用渐变占位并标注「示例照片」。
- 空态 / 错误态 / 弹窗：留给接入真实页面时验证。
- 批次的 `design.config.json`、lock、门禁：等方向确认、新批次正式立项时再补。
