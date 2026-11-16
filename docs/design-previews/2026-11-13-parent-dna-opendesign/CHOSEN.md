# 已选方案（2026-11-14，2026-11-15 收口）

用户选择：**V2 / V3 / V4 三组提案进入默认方向**；V1 / V5 / V6 / V7 / V8 保留为备选。
**交付主稿是 `parent-chosen/` 的 31 页 / 13 状态**；`parent/` 的 baseline 与 `parent-variants/` 只作参照，不是移植起点。

| 页面 | 选中版本 | 路径（已并入主稿） | 后端依赖 |
|---|---|---|---|
| 07 通知详情 | V3 报名提案 | `parent-chosen/07-notice-detail/`（源自 `parent-variants/v3-notice-signup/`） | 需新增报名 API（报名 / 取消 / 已报名状态） |
| 21 亲子任务 | V2 任务积分提案 | `parent-chosen/21-tasks/`（源自 `parent-variants/v2-task-points/`） | `06-task` 恢复 `points` / `streak` |
| 22 任务详情 | V2 任务积分提案（已完成） | `parent-chosen/22-tasks-detail/` | 同上 |
| 24 积分与成就 | V2 任务积分提案 | `parent-chosen/24-points/`（源自 `parent-variants/v2-task-points/`） | 同上 |
| 29 设置 | V4 简化版 | `parent-chosen/29-settings/`（源自 `parent-variants/v4-settings-simple/`） | `settings` 增加免打扰字段；6 个通知细分项合并为 2 个总开关 |

## 后端依赖（移植前先改契约）

- **07 V3 报名提案**：报名 / 取消 / 已报名状态需新增报名 API；当前契约没有这些字段。
- **21 / 22 / 24 V2 任务积分提案**：`06-task` 需恢复 `points` / `streak`，任务打卡作为积分来源。
- **29 V4 简化提案**：`settings` 需增加免打扰字段。6 个通知细分项按来源合并为 2 个总开关（接收园所通知 / 接收老师消息），这是刻意的简化取舍，**不恢复** 9 个开关。

## 底部导航

- 「成长」tab 统一为 **V1 的叶子图标**（`M20 4C12 4 5 8 5 16c0 2 1 4 3 4 8 0 12-7 12-16Z` + `M5 20c2-5 6-8 11-10`）。
- 「接送」tab 统一为 **面包车正视图**（`rect x=4.2 y=4.8 w=15.6 h=12 rx=3` + `M4.4 10.6h15.2` + 两个 `r=1.3` 车轮），覆盖 `01-home`、`11-growth`、`17-pickup`、`31-profile`。
- 已覆盖有底部导航的页面：`parent-chosen/01-home`、`11-growth`、`17-pickup`、`31-profile`；变体参照 `parent-variants/v1-leave-day/01-home`、`v1-leave-day/17-pickup`、`v8-single-child/31-profile`。
- 二级页没有底部导航，不需要改。

## Logo 使用

- 设计稿用 **DNA 版**：`_shared/logo-dna.svg`（pine `#173220` / marigold `#DDB14A` / cream `#FCFAF4`；几何沿用官方 logo，只换色）
- 官方 PNG `_shared/logo.png`（复制自 `parent/src/static/logo.png`）保留为品牌资产
- 02 登录：56×56，置于 `养成星球 · NURTURE PLANET` 眉题上方；中间不放任何插图
- 31 我的：页头只留「我的」标题（原右上角 28×28 logo 已移除）；页脚版本行 16×16
- V8 单孩子：页头 24×24（在“单孩子”角标左侧）+ 页脚 16×16
- 其他页面不重复放 logo；品牌 mark 只在登录与账号页出现，避免每页都贴标
- 移植资产：`_shared/logo-dna-168.png` / `logo-dna-84.png` / `logo-dna-48.png`（168/84/48 = 56/28/16 @3x，圆盘外透明）；小程序 `<image>` 不支持 SVG，移植时用这三档
- 重新导出：`node _shared/export-logo-png.mjs`（本机 Chrome 栅格化 `_shared/logo-dna.svg`，不联网）；校验：`node _shared/check-variants.mjs --chosen`（48-logo-png 资产断言）

## 内容对齐（2026-11-15，已并入 chosen）

- 06 家长会：11/21（周六）10:00 小一班教室（原已正确）
- 18 接送码：今日 23:59 前有效；移除“截图不可用”，保留“一次有效 · 核验后失效”
- 09 新建请假：默认 11/13；日期 min/max、结束不得早于开始、最多 30 天、提交说明改 4 小时内回复
- 10 请假详情：11/13 周五、已提交/等待 2 小时、4 小时内回复；删除“园长审批”节点；新增 `?state=cancelled` 已取消态
- 19 可接送人：行点击打开详情 sheet；已通过可编辑/暂停/删除，待审核可撤回/删除，已拒绝可重提/删除；删除提示“今日已指定接走人会失效”
- 20 添加授权人：补身份证（选填）与 5 个授权范围开关；支持 `?mode=edit`（编辑态为示意，实际实现需按选中授权人 id 预填）
- 30 消息：总未读独立于筛选、本组未读、全部已读、消息详情 sheet；接送码与家长会文案对齐

## 合并落点（parent-chosen/）

- `parent-chosen/`：31 页交付主稿；07 / 21 / 24 / 29 用提案，其余用 baseline，22 补 V2 详情。
- 01 首页：10:20、通知角标 2、任务卡改“给家人讲一件今天开心的事 / 今天还没提交”、午睡改“12:30 开始”。
- 01 首页补充：早餐优先展示今日食谱条目（小米粥、水煮蛋，来源「今日食谱」），食谱缺失时回退「早餐吃得很好」（`?state=meal-fallback`）；页头去头像；快捷入口为 请假 / 亲子任务 / 班级日历 / 通知。
- 17 接送：10:20、可接送人改“妈妈、爸爸，共 2 位（外婆待审核）”、接送码说明改“当日有效，核验后失效”。
- 14 报告：页头加“乐乐”，本期积分改 +218。
- 11 成长：统计卡为 本学期 5 成长记录 / 12 照片 / 3 老师寄语，数字按当前筛选的可见记录实时重算（全部 5/12/3、本月 4/12/3、有照片 3/12/2、老师寄语 3/9/3、只看孩子 4/10/2）；筛选为 全部 / 本月 / 有照片 / 老师寄语 / 只看孩子；记录不再显示专属/班级标签；补秋游班级记录（用于「只看孩子」之外）与一条 10 月无照片记录（凑满「全部 5 条」而不改照片/寄语总量）；右上角接入班级相册；照片可点开查看抽屉。
- 22 任务详情：补 V2 提案态 `+10 积分` / `本周 2 / 3 天`，页脚标注 `积分提案`。
- 31 我的：统计卡改“累计数据 / 32 成长记录”，补“消息”菜单；“切换”打开孩子切换底部抽屉（乐乐 / 朵朵），选中后同步孩子行（头像首字、名字、班级园所）。
- 所有 chosen 页面状态栏统一 10:20。
- V1 / V5 / V6 / V7 / V8 作为状态/数据集并存，不替换主稿。
- 画廊：`chosen.html`（31 页 / 13 契约状态 + 3 评审补充态 = 16 个状态变体）；重建：`node _shared/build-gallery.mjs --chosen`。
- 截图：`node _shared/shots-chosen.mjs`（31 默认 + 16 状态）。13 个契约状态之外，「早餐回退」（01-home `?state=meal-fallback`）、「照片查看」（11-growth `?state=photo`）与「切换孩子」（31-profile `?state=switch`）是为本轮评审补的预览，不进契约清单。

## 验收命令

```bash
cd docs/design-previews/2026-11-13-parent-dna-opendesign
node _shared/check-variants.mjs --chosen   # 31 页，0 失败
node _shared/audit.mjs --chosen            # 0 error
node _shared/font-check.mjs --chosen       # 31 / 31 通过
node _shared/build-gallery.mjs --chosen    # chosen.html，31 页 / 16 个状态变体
node _shared/shots-chosen.mjs              # 31 张默认 + 16 张状态截图
```

## 边界

- 交付主稿是 `parent-chosen/` 的 31 页 / 13 状态；`parent/` 的 baseline 与 `parent-variants/` 的备选变体只作参照，**不是移植起点**。
- `BRIEF-REWRITE-PLAN.md` 仍描述 baseline 写法；chosen 的差异以本文件与 `parent-chosen/` 页面为准。
- 选提案意味着后端契约要改（见「后端依赖」）：`06-task` 的 points/streak、通知报名 API、settings 的免打扰字段。移植前先改契约，不要把提案当现状。
- 不恢复 29 的 9 开关方案；那是另一次契约 / 设计决策。
