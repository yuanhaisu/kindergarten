# 老师端（微信小程序）页面清单（Nuxt `pages/` 目录结构）

> 设计稿标识说明：**✅已有设计稿** / **⏳暂无设计稿**
>
> 当前确认已有设计稿：**登录页、首页、发布通知页、成长记录页、接送管理页、班级日历页、请假审批页、成长记录详情页**（共 8 页）

```text
pages/
  teacher/                              # 老师端（微信小程序）
    login.vue                           ✅ 已有设计稿（老师端小程序-登录页）
    home.vue                            ✅ 已有设计稿（老师端小程序-首页）
    notices/
      index.vue                         ⏳ 通知列表（分类/搜索/未读）
      create.vue                        ✅ 已有设计稿（老师端小程序-发布通知页）
      [id].vue                          ⏳ 通知详情/已读名单
    leave/
      requests.vue                      ✅ 已有设计稿（老师端小程序-请假审批页）
      [id].vue                          ⏳ 请假审批详情
    calendar/
      index.vue                         ✅ 已有设计稿（老师端小程序-班级日历页）
      event-create.vue                  ⏳ 活动创建/编辑
    growth/
      index.vue                         ✅ 已有设计稿（老师端小程序-成长记录页）
      daily-log-create.vue              ⏳ 每日成长日志填写
      album-upload.vue                  ⏳ 照片/视频批量上传
      student/[studentId].vue           ✅ 已有设计稿（老师端小程序-成长记录详情页）
    pickup/
      scan.vue                          ✅ 已有设计稿（老师端小程序-接送管理页）
      records.vue                       ⏳ 接送记录（班级维度）
      record/[id].vue                   ⏳ 接送记录详情/异常标记
    tasks/
      index.vue                         ⏳ 亲子任务列表/布置入口
      create.vue                        ⏳ 任务布置
      stats.vue                         ⏳ 完成率/排行（可选）
    settings/
      index.vue                         ⏳ 个人/班级设置、协议等
```
