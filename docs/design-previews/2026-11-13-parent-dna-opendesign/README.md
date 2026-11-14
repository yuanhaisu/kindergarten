# 家长端设计稿 · OpenDesign DNA 版

- 日期：2026-11-13
- 规则：DNA v1（`docs/design-dna/2026-11-11-kinderly/DESIGN.md`，2026-11-12 冻结）
- 对照基线：`docs/design-previews/2026-11-12-parent-dna-full/`（手工 31 页，**不替换**）
- 设计系统：`user:nurture-planet-dna`（OpenDesign 本地设计系统，见 `open-design/.od/design-systems/nurture-planet-dna/`）

## 怎么审阅

1. 先看 `chosen.html`：已选方案（07/21/24/29 用提案，其余 baseline），说明见 `CHOSEN.md`
2. 全量对照看 `compare.html`：31 页逐页并排
3. 画廊看 `index.html`：31 页 / 10 状态，按 7 个模块分组
4. 逐页结论写在 `COMPARE.md`；机械检查用 `_shared/audit.mjs` 与 `_shared/font-check.mjs`
5. 8 组变体看 `variants.html`：baseline vs 变体/提案并排；清单 `VARIANTS.md`，brief 在 `briefs-variants/`

## 状态

- 试点：01 首页、17 接送三态 **已通过**
- 全量：**31 / 31 页完成**，10 个状态变体
- 机械检查：`audit` 0 error；`font-check` 31/31 通过（H1 衬线 30px/400、390×844 无溢出）
- 产物：`parent/NN-slug/`，每页 `index.html` / `styles.css` / `app.js` / `preview.png`
- 画廊：`index.html`；试点对照：`pilot.html`；逐页结论：`COMPARE.md`
- 手工批次保持原样，供并排比较
- 8 组变体（请假/积分/报名/设置/月度/女/中班/单孩）单独生成，不替换 baseline；见 `VARIANTS.md`

## 约定

- 本批是探索稿，不改产品代码、不迁移 token、不替换手工稿
- 页面用 `?state=empty|error|away|done` 切换状态
- 字体是本地 `_shared/fonts.css`（Instrument Serif + Noto Serif SC），不引外网
- 截图统一 390×844 @2x
