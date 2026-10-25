# 老师端 API 拆分索引

> 从 `../api-teacher.md` 按 `teacher/teacher-mini-program-implementation-plan.md` 的开发模块拆分。

## 通用约定

- 基础路径：`/api/v1/teacher`
- 所有接口均需 `Authorization: Bearer <token>`。
- 访问用户角色必须为 `teacher`。
- 文档中的 Response 均为 `data` 部分，省略外层 `{ "code": 0, "message": "success" }` 包装。
- 老师端接口实现时必须按当前登录老师限定幼儿园和班级范围，避免访问非本班数据。

## 模块文件

| 开发模块 | API 文件 | 范围 |
| --- | --- | --- |
| 模块 3：首页工作台 | [`03-home.md`](./03-home.md) | 首页概览、待办、待办快捷操作 |
| 模块 4：通知管理 | [`04-notices.md`](./04-notices.md) | 通知 CRUD、撤回、提醒、已读状态、模板 |
| 模块 5：请假审批 | [`05-leaves.md`](./05-leaves.md) | 请假列表、统计、详情、同意、拒绝 |
| 模块 6：成长记录 | [`06-growth.md`](./06-growth.md) | 成长记录、评论、点赞、关联学生、媒体上传 |
| 模块 7：接送管理 | [`07-pickup.md`](./07-pickup.md) | 接送概览、学生接送状态、扫码/手动登记、异常 |
| 模块 8：班级日历 | [`08-calendar.md`](./08-calendar.md) | 日历事件、按日查看、日期标记点 |
| 模块 9：班级与亲子任务 | [`09-class-and-tasks.md`](./09-class-and-tasks.md) | 班级信息、学生、亲子任务、回应/公开审核、反馈汇总、班级星星罐与统计 |
| 模块 10：消息聚合列表 | [`10-messages.md`](./10-messages.md) | 通知发布记录 + 请假审批 + 系统消息聚合 |

## 分步执行与恢复

每次执行一个模块时，只加载：

1. 本索引文件；
2. [`PROGRESS.md`](./PROGRESS.md)；
3. 当前模块对应的 API 文件；
4. 必要时加载开发计划中的对应模块段落。

恢复执行时：

1. 先查看 [`PROGRESS.md`](./PROGRESS.md) 的当前状态；
2. 找到第一个未完成或进行中的模块；
3. 只加载该模块 API 文件继续实现；
4. 完成后更新 `PROGRESS.md` 的状态、完成时间和备注。

## 原始文档

原始整本文档仍保留在 [`../api-teacher.md`](../api-teacher.md)，用于全量核对和后续补充。
