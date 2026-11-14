# 亲子任务 · 任务积分提案（V2 / task-points）

- 基线：`briefs/21-tasks.md`
- 变体组：V2 task-points
- 预览：`index.html?variant=points`
- 空态：`index.html?variant=points&state=empty`
- 性质：**未来提案**，不是当前 `06-task` 契约。

> 提案：需 06-task 恢复 points/streak

## 场景

晚饭后，妈妈想在十几分钟内看看今晚要陪孩子做什么、打卡能得多少积分。页面只突出“今天这一件”，其余任务作为清单，不让她在多个任务之间做选择。

## 与 baseline 的差异

1. 状态栏时间改为 `10:20`。
2. 今日任务卡恢复积分与连续打卡：
   - `+10 积分`（marigold 软色数字，仍以松绿为主色）
   - `本周 2 / 3 天 · 今天还没打卡`
   - 三段虚线进度，2 段亮、1 段暗。
3. 任务清单三行恢复积分：`+10`、`+10`、`+15`；状态标签仍为 `待完成`。
4. 空态保留，但文案改为“乐乐今天的任务都提交了”；不再把本周六的任务算进今天。
5. 页面顶部 H1 旁与页脚各有一处 11px 弱文字标记：`积分提案` / `提案：需 06-task 恢复 points/streak`。

## 不包含

- 排行榜、积分商城、补卡付费。
- `+10` 为提案展示值，不代表当前接口已经返回 points/streak。
- 不引入外链字体、CDN 或网络图片；衬线标题使用项目内自托管的 Noto Serif SC 子集。

## 文件

- `index.html` — 390 × 844 页面结构、状态栏、返回栏、默认/空态。
- `styles.css` — DNA v1 绑定 token（`:root` 原样粘贴）+ 本页布局。
- `app.js` — `?state=empty`、筛选、打卡反馈、班级墙/返回 toast。
- `fonts/noto-serif-sc-subset.woff2` — 页面所需字形的本地衬线子集（SIL OFL 1.1）。
