# 家长端设计稿 · OpenDesign DNA 版

- 日期：2026-11-13（2026-11-15 收口）
- 规则：DNA v1（`docs/design-dna/2026-11-11-kinderly/DESIGN.md`，2026-11-12 冻结）
- 交付主稿：`parent-chosen/` 的 31 页（画廊 `chosen.html`，31 页 / 13 契约状态 + 3 评审补充态）
- 对照基线：`docs/design-previews/2026-11-12-parent-dna-full/`（手工 31 页，**仅作参照，不是移植起点**）
- 设计系统：`user:nurture-planet-dna`（OpenDesign 本地设计系统，见 `open-design/.od/design-systems/nurture-planet-dna/`）

## 结论

**`parent-chosen/` 是本批唯一交付主稿**：31 页，V2 / V3 / V4 三组提案已选入，22 的任务详情 V2 已补齐，13 个状态截图齐全。
`parent/` 的 baseline 与 `parent-variants/` 的 V1 / V5 / V6 / V7 / V8 只作参照，不再作为移植起点。

后端依赖（移植前先改契约，不要把提案当现状）：

- 07 通知详情（V3 报名提案）：报名 / 取消需新的报名 API
- 21 / 22 / 24（V2 任务积分提案）：`06-task` 恢复 `points` / `streak`
- 29 设置（V4 简化提案）：6 个通知细分项按来源合并为 2 个总开关；免打扰时段需 `settings` 补字段。这是刻意的简化取舍，不恢复 9 个开关
- 11 / 13 / 14 成长口径：`04-growth` 需新增 `teacher_note`（可空，非空=有老师寄语）、`stats { records, photos, teacher_notes }`（学期累计，与分页无关）、`album_summary { photos, videos }`（成长记录媒体 + 含孩子的班级相册照片）；当前契约未实现

## 第二轮收口（#88，2026-11-18）

- 孩子切换只在 31；01 无切换、无第二个孩子。31 乐乐 42/32/1245，朵朵 18/12/640，含「添加孩子 · 即将支持」行。
- 11 固定 5/12/3（上拉不变）；14 用 5；13「全部照片」48 照片 / 6 视频（成长记录 + 含孩子的班级相册照片）；31 累计 32。
- 02 v3 是独立的紧凑无插图变体，未选入主稿；chosen 02 去掉手机号登录，保留单一同意行。
- 29 移除 `allow_class_share` 控件（班级相册默认同班可见，无家长关闭入口），保留 `show_phone_to_teacher`；2 个通知总开关 + 免打扰。
- 17 先「选择今天接走人」再出码；18 只有二维码，无短验证码；18 另补未指定 / 加载中 / 已过期 / 网络不可用评审态（不进 13 契约截图）。
- 术语表已同步「班内共享」「成长相册」。

## 怎么审阅

1. 先看 `chosen.html`：交付主稿（07 / 21 / 24 / 29 用提案，其余 baseline），说明见 `CHOSEN.md`
2. 全量对照看 `compare.html`：31 页逐页并排
3. baseline 画廊看 `index.html`：31 页 / 10 状态，仅作对照
4. 逐页结论写在 `COMPARE.md`；机械检查用 `_shared/check-variants.mjs --chosen`、`_shared/audit.mjs --chosen`、`_shared/font-check.mjs --chosen`
5. 8 组变体看 `variants.html`：baseline vs 变体 / 提案并排；清单 `VARIANTS.md`，brief 在 `briefs-variants/`

## 状态

- 全量：**31 / 31 页完成**，chosen 页集 13 个契约状态截图（另有 3 个评审补充态：01 早餐回退 / 11 照片查看 / 31 切换孩子）
- 机械检查：chosen checker 0 失败；`audit --chosen` 0 error；`font-check --chosen` 31 / 31 通过（H1 衬线 30px / 400、390×844 无溢出）
- 产物：`parent-chosen/NN-slug/`，每页 `index.html` / `styles.css` / `app.js` / `preview.png`
- 画廊：`chosen.html`（31 页 / 16 个状态变体）；重建：`node _shared/build-gallery.mjs --chosen`
- 截图：`node _shared/shots-chosen.mjs`（31 张默认 + 16 张状态）
- 手工批次与 `parent/` baseline 保持原样，仅供并排比较
- 8 组变体（请假 / 积分 / 报名 / 设置 / 月度 / 女 / 中班 / 单孩）单独生成，不替换主稿；V2 / V3 / V4 已选入 chosen

## 约定

- chosen 是本批唯一交付主稿；不改产品代码、不迁移 token
- 页面用 `?state=empty|error|away|done|cancelled|detail|switch|photo|meal-fallback` 切换状态
- 字体是本地 `_shared/fonts.css`（Instrument Serif + Noto Serif SC），不引外网
- 截图统一 390×844 @2x
