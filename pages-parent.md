# 家长端（微信小程序）页面清单（Nuxt `pages/` 目录结构）

> 设计稿标识说明：**✅已有设计稿** / **⏳暂无设计稿**
>
> 当前（该设计文件）尚未读取到家长端顶层画板，因此暂时全部标记为 **⏳暂无设计稿**（来自 `产品方案.md` 功能拆解）。

```text
pages/
  parent/                               # 家长端（微信小程序）
    login.vue                           ⏳（如需要手机号登录/绑定）
    home.vue                            ⏳
    notices/
      index.vue                         ⏳ 通知列表
      [id].vue                          ⏳ 通知详情/确认已读
    leave/
      create.vue                        ⏳ 发起请假
      records.vue                       ⏳ 请假记录
    calendar/
      index.vue                         ⏳ 班级日历
    growth/
      timeline.vue                      ⏳ 成长时间轴（浏览/点赞/留言）
      album.vue                         ⏳ 成长相册
      report.vue                        ⏳ AI 成长报告（周/月）
      export.vue                        ⏳ 相册导出
    pickup/
      authorizations.vue                ⏳ 授权接送人管理（增删/有效期）
      records.vue                       ⏳ 接送记录（孩子维度）
      qr.vue                            ⏳ 孩子二维码（如需要展示）
    tasks/
      index.vue                         ⏳ 亲子任务列表
      [id].vue                          ⏳ 任务详情/打卡
      achievements.vue                  ⏳ 积分/徽章
      leaderboard.vue                   ⏳ 班级排行（可选）
    settings/
      index.vue                         ⏳
```
