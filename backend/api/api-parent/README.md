# 家长端小程序 — API 模块文档

> 基础路径：`/api/v1/parent` | 鉴权：`Authorization: Bearer <token>`（角色 `parent`）
>
> 开发进度与恢复执行规则见 [PROGRESS.md](PROGRESS.md)。

## 模块文件索引

| # | 文件 | 模块 | 接口数 |
|---|------|------|--------|
| 1 | [01-home.md](01-home.md) | 首页 | 4 |
| 2 | [02-notice.md](02-notice.md) | 通知公告 | 3 |
| 3 | [03-leave.md](03-leave.md) | 请假 | 5 |
| 4 | [04-growth.md](04-growth.md) | 成长时光 | 8 |
| 5 | [05-pickup.md](05-pickup.md) | 接送管理 | 6 |
| 6 | [06-task.md](06-task.md) | 亲子任务 | 4 |
| 7 | [07-points.md](07-points.md) | 积分与成就 | 4 |
| 8 | [08-class.md](08-class.md) | 班级 | 2 |
| 9 | [09-student.md](09-student.md) | 学生管理 | 5 |
| 10 | [10-feedback.md](10-feedback.md) | 意见反馈 | 2 |
| 11 | [11-settings.md](11-settings.md) | 设置 | 2 |

## 设计稿对照

| 设计页面 | 对应模块 |
|---------|---------|
| 首页 | 01-home |
| 通知公告页 | 02-notice |
| 我要请假页 | 03-leave |
| 成长时光页 | 04-growth（feed / album） |
| 成长报告页 | 04-growth（report） |
| 评论详情页 / 写评论弹窗 | 04-growth（comments） |
| 接送授权页 | 05-pickup（records / authorized-persons） |
| 添加授权页 | 05-pickup（POST authorized-persons） |
| 亲子任务页 | 06-task + 07-points |
| 亲子任务详情页 | 06-task（detail / checkin） |
| 班级日历页 | 08-class（calendar） |
| 联系老师页 | 08-class（info） |
| 个人中心页 | 09-student + 10-feedback + 11-settings |
