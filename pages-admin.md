# 管理后台（网页端）页面清单（Nuxt `pages/` 目录结构）

> 设计稿标识说明：**✅已有设计稿** / **⏳暂无设计稿**
>
> 当前（该设计文件）尚未读取到管理后台顶层画板，因此暂时全部标记为 **⏳暂无设计稿**（来自 `产品方案.md` 功能拆解）。

```text
pages/
  admin/                                # 管理后台（网页端）
    login.vue                           ⏳
    dashboard.vue                       ⏳ 数据看板（已读率/活跃度/完成率等）
    classes/
      index.vue                         ⏳ 班级管理
      [id].vue                          ⏳ 班级详情（老师/学生/家长）
    staff/
      teachers.vue                      ⏳ 老师账号管理
      roles.vue                         ⏳ 角色/权限（如需要）
    students/
      index.vue                         ⏳ 学生与家长信息
      [id].vue                          ⏳ 学生详情/绑定关系
    content/
      notice-audit.vue                  ⏳ 通知与内容审核
      announcements.vue                 ⏳ 园所公告发布
    attendance/
      overview.vue                      ⏳ 出勤统计（含请假原因分布）
      export.vue                        ⏳ 报表导出
    pickup/
      records.vue                       ⏳ 全园接送记录查询
      exceptions.vue                    ⏳ 异常预警/回溯
    analytics/
      parent-engagement.vue             ⏳ 家长互动数据（已读率/活跃等）
      task-completion.vue               ⏳ 任务完成率分析
    settings/
      index.vue                         ⏳ 园所配置/协议/基础信息
```


数据看板-》