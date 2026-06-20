# 家长端（Parent Mini-Program）产品测试用例

> 版本: 1.0 | 最后更新: 2026-06-15
> ID 体系: `TC-PRN-{MODULE}-{NNNN}`
> 状态: 🔲 未执行 | ✅ 已通过 | ❌ 失败
> 优先级: P0（核心流程）| P1（重要功能）| P2（边缘场景）
> 测试类型: FUNC | UX | SEC | MP（小程序特性）| CROSS（跨端联动）

## 前置条件

1. **家长已登录**: 微信小程序 `wx.login()` → code2session 成功，持有有效 JWT token（`role=parent`）
2. **家长已绑定学生**: 通过邀请码或手机号绑定至少一名学生
3. **园所已初始化**: 所在园所有效，班级、老师数据完整
4. **网络正常**: 微信小程序网络正常

## 测试限制

| 限制项 | 说明 |
|--------|------|
| 家长仅微信登录 | 无密码通道，测试需微信开发者工具或真实小程序 |
| 多学生切换 | 家长可绑定多个学生，通过 `switch-student` 切换 |
| 4 Tab 导航 | 首页/成长/任务/我的，custom-tab-bar |

---

## 1. 登录与绑定（AUTH）— `pages/auth/*`

| TC ID | 名称 | P | 类型 | 状态 |
|-------|------|---|------|------|
| TC-PRN-AUTH-001 | 静默登录-App启动自动登录 | P0 | MP | 🔲 |
| TC-PRN-AUTH-002 | 手机号绑定 | P0 | FUNC | 🔲 |
| TC-PRN-AUTH-003 | 邀请码绑定学生 | P0 | FUNC | 🔲 |
| TC-PRN-AUTH-004 | 绑定多个学生 | P1 | FUNC | 🔲 |
| TC-PRN-AUTH-005 | 解绑学生 | P1 | FUNC | 🔲 |
| TC-PRN-AUTH-006 | Token 自动刷新 | P0 | MP | 🔲 |
| TC-PRN-AUTH-007 | Token 过期-重新登录 | P1 | MP | 🔲 |

---

## 2. 首页（HOME）— `pages/index/index`（Tab 1）

| TC ID | 名称 | P | 类型 | 状态 |
|-------|------|---|------|------|
| TC-PRN-HOME-001 | 首页-概览数据加载 | P0 | FUNC | 🔲 |
| TC-PRN-HOME-002 | 首页-学生切换 | P0 | FUNC | 🔲 |
| TC-PRN-HOME-003 | 首页-园所公告展示 | P1 | FUNC | 🔲 |
| TC-PRN-HOME-004 | 首页-成长时光流 | P0 | FUNC | 🔲 |
| TC-PRN-HOME-005 | 首页-下拉刷新 | P1 | MP | 🔲 |
| TC-PRN-HOME-006 | 首页-8个功能入口（对齐设计稿） | P1 | UX | 🔲 |

---

## 3. 通知（NOTI）— `pages/notice/*`

| TC ID | 名称 | P | 类型 | 状态 |
|-------|------|---|------|------|
| TC-PRN-NOTI-001 | 通知列表-按学生维度 | P0 | FUNC | 🔲 |
| TC-PRN-NOTI-002 | 通知详情-查看完整内容 | P0 | FUNC | 🔲 |
| TC-PRN-NOTI-003 | 确认收到（need_confirm=true） | P0 | FUNC | 🔲 |
| TC-PRN-NOTI-004 | 通知列表-空数据 | P1 | UX | 🔲 |

---

## 4. 请假（LEAV）— `pages/leave/*`

| TC ID | 名称 | P | 类型 | 状态 |
|-------|------|---|------|------|
| TC-PRN-LEAV-001 | 请假列表-查看历史 | P0 | FUNC | 🔲 |
| TC-PRN-LEAV-002 | 创建请假-选择学生+日期+原因 | P0 | FUNC | 🔲 |
| TC-PRN-LEAV-003 | 创建请假-必填字段为空拒绝 | P0 | FUNC | 🔲 |
| TC-PRN-LEAV-004 | 请假详情-查看审批状态 | P0 | FUNC | 🔲 |
| TC-PRN-LEAV-005 | 编辑请假-未审批的可修改 | P1 | FUNC | 🔲 |
| TC-PRN-LEAV-006 | 取消请假 | P1 | FUNC | 🔲 |
| TC-PRN-LEAV-007 | 跨端-老师审批后状态更新 | P0 | CROSS | 🔲 |

---

## 5. 成长记录（GRW）— `pages/growth/*`（Tab 2）

| TC ID | 名称 | P | 类型 | 状态 |
|-------|------|---|------|------|
| TC-PRN-GRW-001 | 成长时光流-按时间线展示 | P0 | FUNC | 🔲 |
| TC-PRN-GRW-002 | 成长详情-含评分/标签/媒体 | P0 | FUNC | 🔲 |
| TC-PRN-GRW-003 | 点赞-首次点赞 | P1 | FUNC | 🔲 |
| TC-PRN-GRW-004 | 取消点赞 | P1 | FUNC | 🔲 |
| TC-PRN-GRW-005 | 评论列表 | P1 | FUNC | 🔲 |
| TC-PRN-GRW-006 | 发表评论 | P1 | FUNC | 🔲 |
| TC-PRN-GRW-007 | 成长相册-按月份/分类 | P1 | FUNC | 🔲 |
| TC-PRN-GRW-008 | 成长报告-维度评分 | P1 | FUNC | 🔲 |
| TC-PRN-GRW-009 | 跨端-老师发布 sync_to_feed=true→流中可见 | P0 | CROSS | 🔲 |
| TC-PRN-GRW-010 | 跨端-sync_to_feed=false→流中不可见 | P1 | CROSS | 🔲 |

---

## 6. 接送（PKU）— `pages/pickup/*`

| TC ID | 名称 | P | 类型 | 状态 |
|-------|------|---|------|------|
| TC-PRN-PKU-001 | 接送二维码-生成展示 | P0 | FUNC | 🔲 |
| TC-PRN-PKU-002 | 接送记录-历史列表 | P0 | FUNC | 🔲 |
| TC-PRN-PKU-003 | 授权接送人-列表 | P0 | FUNC | 🔲 |
| TC-PRN-PKU-004 | 添加授权接送人 | P0 | FUNC | 🔲 |
| TC-PRN-PKU-005 | 编辑授权接送人 | P1 | FUNC | 🔲 |
| TC-PRN-PKU-006 | 删除授权接送人 | P1 | FUNC | 🔲 |
| TC-PRN-PKU-007 | 跨端-老师审核授权人后状态更新 | P0 | CROSS | 🔲 |
| TC-PRN-PKU-008 | 跨端-老师扫码接送后记录更新 | P0 | CROSS | 🔲 |

---

## 7. 亲子任务（TSK）— `pages/tasks/*`（Tab 3）

| TC ID | 名称 | P | 类型 | 状态 |
|-------|------|---|------|------|
| TC-PRN-TSK-001 | 任务列表-按学生维度 | P0 | FUNC | 🔲 |
| TC-PRN-TSK-002 | 任务详情-含要求/奖励 | P0 | FUNC | 🔲 |
| TC-PRN-TSK-003 | 打卡签到 | P0 | FUNC | 🔲 |
| TC-PRN-TSK-004 | 打卡历史-查看 | P1 | FUNC | 🔲 |
| TC-PRN-TSK-005 | 跨端-老师新建任务→家长可见 | P0 | CROSS | 🔲 |
| TC-PRN-TSK-006 | 跨端-家长打卡→老师统计更新 | P0 | CROSS | 🔲 |

---

## 8. 积分（PNT）— `pages/points/index`

| TC ID | 名称 | P | 类型 | 状态 |
|-------|------|---|------|------|
| TC-PRN-PNT-001 | 积分概览-总分/趋势 | P1 | FUNC | 🔲 |
| TC-PRN-PNT-002 | 积分历史-明细列表 | P1 | FUNC | 🔲 |
| TC-PRN-PNT-003 | 积分排行-班级/全园 | P1 | FUNC | 🔲 |

---

## 9. 成就（ACH）

| TC ID | 名称 | P | 类型 | 状态 |
|-------|------|---|------|------|
| TC-PRN-ACH-001 | 成就列表-已获得/未获得 | P1 | FUNC | 🔲 |

---

## 10. 班级（CLS）— `pages/class/index`

| TC ID | 名称 | P | 类型 | 状态 |
|-------|------|---|------|------|
| TC-PRN-CLS-001 | 班级信息-老师/学生 | P1 | FUNC | 🔲 |
| TC-PRN-CLS-002 | 班级日历-查看事件 | P1 | FUNC | 🔲 |

---

## 11. 学生（STD）— `pages/student/index`

| TC ID | 名称 | P | 类型 | 状态 |
|-------|------|---|------|------|
| TC-PRN-STD-001 | 学生资料-查看 | P1 | FUNC | 🔲 |
| TC-PRN-STD-002 | 编辑学生资料 | P1 | FUNC | 🔲 |
| TC-PRN-STD-003 | 学生列表-多学生切换 | P1 | FUNC | 🔲 |

---

## 12. 反馈（FBK）— `pages/feedback/index`

| TC ID | 名称 | P | 类型 | 状态 |
|-------|------|---|------|------|
| TC-PRN-FBK-001 | 反馈列表 | P2 | FUNC | 🔲 |
| TC-PRN-FBK-002 | 提交反馈 | P2 | FUNC | 🔲 |

---

## 13. 消息（MSG）— `pages/messages/index`

| TC ID | 名称 | P | 类型 | 状态 |
|-------|------|---|------|------|
| TC-PRN-MSG-001 | 消息列表 | P1 | FUNC | 🔲 |

---

## 14. 设置（SET）— `pages/settings/index`

| TC ID | 名称 | P | 类型 | 状态 |
|-------|------|---|------|------|
| TC-PRN-SET-001 | 个人设置-查看/编辑 | P1 | FUNC | 🔲 |

---

## 15. 个人中心（PROF）— `pages/profile/index`（Tab 4）

| TC ID | 名称 | P | 类型 | 状态 |
|-------|------|---|------|------|
| TC-PRN-PROF-001 | 个人信息-头像/昵称 | P1 | FUNC | 🔲 |
| TC-PRN-PROF-002 | 已绑定学生列表 | P1 | FUNC | 🔲 |

---

## 16. 全局通用（GLOBAL）

| TC ID | 名称 | P | 类型 | 状态 |
|-------|------|---|------|------|
| TC-PRN-GLB-001 | 多学生数据隔离-切换后数据正确 | P0 | FUNC | 🔲 |
| TC-PRN-GLB-002 | 园所隔离-跨园所不可见 | P0 | SEC | 🔲 |
| TC-PRN-GLB-003 | 下拉刷新（Tab 页） | P1 | MP | 🔲 |
| TC-PRN-GLB-004 | 页面栈-正常前进/返回 | P1 | MP | 🔲 |
| TC-PRN-GLB-005 | 小程序切后台再回来 | P2 | MP | 🔲 |
| TC-PRN-GLB-006 | 未认证请求 → 401 | P1 | SEC | 🔲 |
| TC-PRN-GLB-007 | 空数据-空状态提示 | P2 | UX | 🔲 |
| TC-PRN-GLB-008 | 无效 ID 参数校验 | P1 | FUNC | 🔲 |

---

## 测试覆盖汇总

| 模块 | 用例数 | P0 | P1 | P2 |
|------|:------:|:--:|:--:|:--:|
| AUTH 登录绑定 | 7 | 3 | 3 | 1 |
| HOME 首页 | 6 | 2 | 4 | 0 |
| NOTI 通知 | 4 | 3 | 1 | 0 |
| LEAV 请假 | 7 | 4 | 2 | 1 |
| GRW 成长记录 | 10 | 2 | 7 | 1 |
| PKU 接送 | 8 | 4 | 2 | 2 |
| TSK 亲子任务 | 6 | 4 | 1 | 1 |
| PNT 积分 | 3 | 0 | 3 | 0 |
| ACH 成就 | 1 | 0 | 1 | 0 |
| CLS 班级 | 2 | 0 | 2 | 0 |
| STD 学生 | 3 | 0 | 3 | 0 |
| FBK 反馈 | 2 | 0 | 0 | 2 |
| MSG 消息 | 1 | 0 | 1 | 0 |
| SET 设置 | 1 | 0 | 1 | 0 |
| PROF 个人中心 | 2 | 0 | 2 | 0 |
| GLOBAL 全局 | 8 | 1 | 4 | 3 |
| **合计** | **71** | **23** | **37** | **11** |
