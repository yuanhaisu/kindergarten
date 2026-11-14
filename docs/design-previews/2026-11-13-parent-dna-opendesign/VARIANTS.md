# 8 组变体 · 生成与对照

- 日期：2026-11-14
- 基线：`parent/` 的 31 页 + `BRIEF-REWRITE-PLAN.md` 0.1/0.2 的 baseline
- 变体 brief：`briefs-variants/<组>/<页面>.md`
- 合成 brief：`briefs-variants/<组>/<页面>.composed.md`（变体 delta + baseline 原文）
- 产物：`parent-variants/<组>/<页面>/`（`index.html` / `styles.css` / `app.js` / `preview.png`）
- 执行器：`node _shared/run-variants.mjs --pilot | --group <组> | --page <组>/<页面> | --all`

## 8 组

| 组 | 变体 | 页面 | 状态/参数 | 性质 |
|---|---|---|---|---|
| V1 | 11/12 请假在家 | 01-home、17-pickup | `?state=leave` | 场景变体；08/10 的 baseline 仍按 11/13，不在本轮 |
| V2 | 任务积分提案 | 21-tasks、24-points | `?variant=points` | 提案；需 06-task 恢复 points/streak |
| V3 | 通知报名提案 | 07-notice-detail | `?variant=signup` | 提案；需新增报名 API |
| V4 | 设置简化版 | 29-settings | `?variant=simple` | 简化提案；免打扰需后端补字段 |
| V5 | 报告月度版 | 14-growth-report | `?period=month` | 周期变体；+180 与 24 对齐 |
| V6 | 女性 / 5 岁数据集 | 27-student | `?dataset=female-5y` | 数据集变体；11/12 的“她”作为同数据集说明 |
| V7 | 5 岁 / 中班数据集 | 27-student | `?dataset=age5-class` | 数据集变体；01/25/31 的班级标签作为说明 |
| V8 | 单孩子 | 31-profile | `?children=1` | 数据/状态变体；隐藏“切换” |

## 已选（2026-11-14）

用户选择 **V2 / V3 / V4** 三组提案进入默认方向；V1 / V5 / V6 / V7 / V8 保留备选。详见 `CHOSEN.md`。

## 生成状态

| 页面 | 状态 | 截图 |
|---|---|---|
| V1 01-home | **已生成**（请假态） | `parent-variants/v1-leave-day/01-home/preview.png` |
| V1 17-pickup | **已生成**（请假中） | `parent-variants/v1-leave-day/17-pickup/preview.png` |
| V2 21-tasks | **已生成**（积分提案） | `parent-variants/v2-task-points/21-tasks/preview.png` |
| V2 24-points | **已生成**（任务打卡 +10） | `parent-variants/v2-task-points/24-points/preview.png` |
| V3 07-notice-detail | **已生成**（报名提案） | `parent-variants/v3-notice-signup/07-notice-detail/preview.png` |
| V4 29-settings | **已生成**（简化版） | `parent-variants/v4-settings-simple/29-settings/preview.png` |
| V5 14-growth-report | **已生成**（本月版） | `parent-variants/v5-report-month/14-growth-report/preview.png` |
| V6 27-student | **已生成**（女 / 5 岁） | `parent-variants/v6-female-child/27-student/preview.png` |
| V7 27-student | **已生成**（5 岁 / 中班） | `parent-variants/v7-age5-class/27-student/preview.png` |
| V8 31-profile | **已生成**（单孩子） | `parent-variants/v8-single-child/31-profile/preview.png` |

- 生成时间：2026-11-14 16:16–17:57
- `check-variants`：**10 / 10 通过**
- `build-variants-gallery --check`：**文件齐全**
- 画廊：`variants.html`（12746 bytes，10 页）

## 说明

- 变体不替换 baseline；31 页默认仍是 `BRIEF-REWRITE-PLAN.md` 的 baseline。
- V6 / V7 都是 27，输出到不同组目录，避免覆盖。
- 变体页面顶部/页脚要有 11px“变体 / 提案”标注，避免被当成现状。
- 生成后建 `variants.html`，与 `index.html`（31 页 baseline）并列。
