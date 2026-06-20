# 管理端（Admin Web）产品测试用例

> 版本: 1.0 | 最后更新: 2026-06-15
> ID 体系: `TC-ADM-{MODULE}-{NNNN}`
> 状态: 🔲 未执行 | ✅ 已通过 | ❌ 失败
> 优先级: P0（核心流程）| P1（重要功能）| P2（边缘场景）
> 测试类型: FUNC | UX | SEC | RBAC | CROSS（跨端联动）

## 前置条件

1. **管理员已登录**: 手机号+密码登录，持有有效 JWT token（`role=admin`）
2. **园所已初始化**: 通过 `/api/v1/system/init` 完成初始化
3. **测试数据**: 园所 `id=4`，含 6 个班级、24 名学生、多名老师
4. **网络正常**: 浏览器可访问 `localhost:3000` → API proxy → `localhost:8080`

## 测试限制

| 限制项 | 说明 |
|--------|------|
| Nuxt SPA 路由 | 直接 URL 访问可能 404，需通过侧边栏导航 |
| RBAC 角色隔离 | admin token 不可访问 teacher/parent API（403） |
| 密码登录 | 仅管理员可用密码登录；老师/家长仅微信登录 |

---

## 1. 登录与认证（AUTH）

| TC ID | 名称 | P | 类型 | 状态 |
|-------|------|---|------|------|
| TC-ADM-AUTH-001 | 密码登录-手机号+密码成功 | P0 | FUNC | ✅ |
| TC-ADM-AUTH-002 | 密码登录-手机号为空 | P0 | FUNC | 🔲 |
| TC-ADM-AUTH-003 | 密码登录-密码为空 | P0 | FUNC | 🔲 |
| TC-ADM-AUTH-004 | 密码登录-密码错误 | P0 | FUNC | 🔲 |
| TC-ADM-AUTH-005 | 密码登录-非管理员账号 | P0 | FUNC | 🔲 |
| TC-ADM-AUTH-006 | Token 自动刷新 | P0 | FUNC | 🔲 |
| TC-ADM-AUTH-007 | Token 过期-重新登录 | P0 | FUNC | 🔲 |
| TC-ADM-AUTH-008 | 退出登录-清除 token | P1 | FUNC | 🔲 |

---

## 2. 数据看板（DASH）— `pages/index.vue`

| TC ID | 名称 | P | 类型 | 状态 |
|-------|------|---|------|------|
| TC-ADM-DASH-001 | 看板-总览数据加载 | P0 | FUNC | ✅ |
| TC-ADM-DASH-002 | 看板-出勤趋势（周/月切换） | P1 | FUNC | 🔲 |
| TC-ADM-DASH-003 | 看板-班级排名列表 | P1 | FUNC | 🔲 |
| TC-ADM-DASH-004 | 看板-成长统计 | P1 | FUNC | 🔲 |
| TC-ADM-DASH-005 | 看板-请假原因分布（空数据） | P2 | UX | 🔲 |
| TC-ADM-DASH-006 | 看板-待办事项卡片 | P1 | FUNC | 🔲 |

### ADM-DASH-001 [P0] 总览数据 ✅
- **实测**: 总学生数=24、今日出勤=0、通知已读率=0%、任务完成率=0%
- **预期**: 数据与后端 `/api/v1/admin/dashboard/overview` 一致

---

## 3. 班级管理（CLS）— `pages/classes/index.vue`

| TC ID | 名称 | P | 类型 | 状态 |
|-------|------|---|------|------|
| TC-ADM-CLS-001 | 班级列表-分页展示 | P0 | FUNC | 🔲 |
| TC-ADM-CLS-002 | 创建班级-填写完整信息 | P0 | FUNC | 🔲 |
| TC-ADM-CLS-003 | 创建班级-名称/年级为空拒绝 | P0 | FUNC | 🔲 |
| TC-ADM-CLS-004 | 编辑班级-修改名称/容量 | P1 | FUNC | 🔲 |
| TC-ADM-CLS-005 | 删除班级-空班级可删除 | P0 | FUNC | 🔲 |
| TC-ADM-CLS-006 | 删除班级-有学生班级拒绝 | P1 | FUNC | 🔲 |
| TC-ADM-CLS-007 | 班级详情-含教师列表 | P1 | FUNC | 🔲 |
| TC-ADM-CLS-008 | 调整班级学生-批量分配 | P1 | FUNC | 🔲 |

---

## 4. 学生管理（STD）— `pages/students/index.vue`

| TC ID | 名称 | P | 类型 | 状态 |
|-------|------|---|------|------|
| TC-ADM-STD-001 | 学生列表-分页+搜索 | P0 | FUNC | 🔲 |
| TC-ADM-STD-002 | 创建学生-基本字段 | P0 | FUNC | 🔲 |
| TC-ADM-STD-003 | 创建学生-必填字段为空拒绝 | P0 | FUNC | 🔲 |
| TC-ADM-STD-004 | 导入学生-批量导入 Excel | P1 | FUNC | 🔲 |
| TC-ADM-STD-005 | 导出学生-下载 Excel | P1 | FUNC | 🔲 |
| TC-ADM-STD-006 | 编辑学生-修改信息 | P1 | FUNC | 🔲 |
| TC-ADM-STD-007 | 学生状态变更（在读↔离园） | P1 | FUNC | 🔲 |
| TC-ADM-STD-008 | 删除学生 | P1 | FUNC | 🔲 |

---

## 5. 老师管理（TCH）— `pages/teachers/index.vue`

| TC ID | 名称 | P | 类型 | 状态 |
|-------|------|---|------|------|
| TC-ADM-TCH-001 | 老师列表-含班级/职称 | P0 | FUNC | 🔲 |
| TC-ADM-TCH-002 | 创建老师-生成邀请链接 | P0 | FUNC | 🔲 |
| TC-ADM-TCH-003 | 编辑老师-修改班级/职称 | P1 | FUNC | 🔲 |
| TC-ADM-TCH-004 | 老师状态变更（在职↔离职） | P1 | FUNC | 🔲 |
| TC-ADM-TCH-005 | 生成/重新生成邀请二维码 | P1 | FUNC | 🔲 |

---

## 6. 家长管理（PRT）— `pages/parents/index.vue`

| TC ID | 名称 | P | 类型 | 状态 |
|-------|------|---|------|------|
| TC-ADM-PRT-001 | 家长列表-含关联学生 | P0 | FUNC | 🔲 |
| TC-ADM-PRT-002 | 创建家长账号 | P1 | FUNC | 🔲 |
| TC-ADM-PRT-003 | 编辑家长信息 | P1 | FUNC | 🔲 |
| TC-ADM-PRT-004 | 家长状态变更 | P1 | FUNC | 🔲 |

---

## 7. 出勤管理（ATT）— `pages/attendance/index.vue`

| TC ID | 名称 | P | 类型 | 状态 |
|-------|------|---|------|------|
| TC-ADM-ATT-001 | 出勤概览-按班级/日期筛选 | P0 | FUNC | 🔲 |
| TC-ADM-ATT-002 | 出勤记录-列表+分页 | P0 | FUNC | 🔲 |
| TC-ADM-ATT-003 | 出勤报表-导出 | P1 | FUNC | 🔲 |

---

## 8. 请假管理（LEV）— `pages/leaves/index.vue`

| TC ID | 名称 | P | 类型 | 状态 |
|-------|------|---|------|------|
| TC-ADM-LEV-001 | 请假列表-按状态筛选 | P0 | FUNC | 🔲 |
| TC-ADM-LEV-002 | 请假统计-各状态数量 | P1 | FUNC | 🔲 |
| TC-ADM-LEV-003 | 请假审核-通过 | P0 | FUNC | 🔲 |
| TC-ADM-LEV-004 | 请假审核-拒绝(必填原因) | P0 | FUNC | 🔲 |

---

## 9. 接送管理（PKU）— `pages/pickup/index.vue`

| TC ID | 名称 | P | 类型 | 状态 |
|-------|------|---|------|------|
| TC-ADM-PKU-001 | 接送概览-按班级/日期 | P0 | FUNC | ✅ |
| TC-ADM-PKU-002 | 接送记录-列表+搜索 | P0 | FUNC | 🔲 |
| TC-ADM-PKU-003 | 异常记录-列表 | P1 | FUNC | 🔲 |
| TC-ADM-PKU-004 | 授权接送人-列表 | P1 | FUNC | 🔲 |
| TC-ADM-PKU-005 | 授权接送人-审核通过/拒绝 | P1 | FUNC | 🔲 |

---

## 10. 成长记录（GRW）— `pages/growth/index.vue`

| TC ID | 名称 | P | 类型 | 状态 |
|-------|------|---|------|------|
| TC-ADM-GRW-001 | 成长记录列表-分页 | P0 | FUNC | 🔲 |
| TC-ADM-GRW-002 | 成长记录详情 | P1 | FUNC | 🔲 |
| TC-ADM-GRW-003 | 成长统计 | P1 | FUNC | 🔲 |
| TC-ADM-GRW-004 | 审核成长记录-通过 | P0 | FUNC | 🔲 |
| TC-ADM-GRW-005 | 审核成长记录-驳回 | P0 | FUNC | 🔲 |
| TC-ADM-GRW-006 | 删除成长记录 | P1 | FUNC | 🔲 |

> ⚠️ 管理端**不可创建**成长记录（仅老师端可创建），`POST /admin/growth/logs` → 404

---

## 11. 亲子任务（TSK）— `pages/tasks/index.vue`

| TC ID | 名称 | P | 类型 | 状态 |
|-------|------|---|------|------|
| TC-ADM-TSK-001 | 任务列表 | P0 | FUNC | 🔲 |
| TC-ADM-TSK-002 | 创建任务-指定班级 | P0 | FUNC | ✅ |
| TC-ADM-TSK-003 | 编辑任务 | P1 | FUNC | 🔲 |
| TC-ADM-TSK-004 | 删除任务 | P1 | FUNC | 🔲 |
| TC-ADM-TSK-005 | 任务统计-打卡率 | P1 | FUNC | 🔲 |

---

## 12. 积分管理（PNT）— `pages/points/index.vue`

| TC ID | 名称 | P | 类型 | 状态 |
|-------|------|---|------|------|
| TC-ADM-PNT-001 | 积分概览 | P1 | FUNC | 🔲 |
| TC-ADM-PNT-002 | 积分记录-列表 | P1 | FUNC | 🔲 |
| TC-ADM-PNT-003 | 手动调整积分 | P1 | FUNC | 🔲 |
| TC-ADM-PNT-004 | 积分规则-配置 | P2 | FUNC | 🔲 |

---

## 13. 成就管理（ACH）— `pages/achievements/index.vue`

| TC ID | 名称 | P | 类型 | 状态 |
|-------|------|---|------|------|
| TC-ADM-ACH-001 | 成就列表 | P1 | FUNC | 🔲 |
| TC-ADM-ACH-002 | 创建成就 | P1 | FUNC | 🔲 |
| TC-ADM-ACH-003 | 编辑成就 | P2 | FUNC | 🔲 |

---

## 14. 通知管理（NOT）— `pages/notices/index.vue`

| TC ID | 名称 | P | 类型 | 状态 |
|-------|------|---|------|------|
| TC-ADM-NOT-001 | 通知列表-按班级/状态筛选 | P0 | FUNC | 🔲 |
| TC-ADM-NOT-002 | 创建通知-选择班级 | P0 | FUNC | ✅ |
| TC-ADM-NOT-003 | 审核通知-通过 | P1 | FUNC | 🔲 |
| TC-ADM-NOT-004 | 编辑通知 | P1 | FUNC | 🔲 |
| TC-ADM-NOT-005 | 删除通知 | P1 | FUNC | 🔲 |

---

## 15. 园所公告（ANN）— `pages/announcements/index.vue`

| TC ID | 名称 | P | 类型 | 状态 |
|-------|------|---|------|------|
| TC-ADM-ANN-001 | 公告列表 | P1 | FUNC | 🔲 |
| TC-ADM-ANN-002 | 创建公告 | P1 | FUNC | 🔲 |
| TC-ADM-ANN-003 | 置顶/取消置顶 | P1 | FUNC | 🔲 |
| TC-ADM-ANN-004 | 编辑/删除公告 | P1 | FUNC | 🔲 |

---

## 16. 内容审核（AUD）— `pages/audit/index.vue`

| TC ID | 名称 | P | 类型 | 状态 |
|-------|------|---|------|------|
| TC-ADM-AUD-001 | 审核列表 | P1 | FUNC | 🔲 |
| TC-ADM-AUD-002 | 审核通过/驳回 | P1 | FUNC | 🔲 |
| TC-ADM-AUD-003 | 审核统计 | P2 | FUNC | 🔲 |

---

## 17. 系统设置（SET）— `pages/settings/*`

| TC ID | 名称 | P | 类型 | 状态 |
|-------|------|---|------|------|
| TC-ADM-SET-001 | 基础设置-查看/编辑 | P1 | FUNC | 🔲 |
| TC-ADM-SET-002 | 园所信息-查看/编辑 | P1 | FUNC | 🔲 |
| TC-ADM-SET-003 | 学年管理-创建/切换 | P1 | FUNC | 🔲 |
| TC-ADM-SET-004 | 字典管理-类型 CRUD | P1 | FUNC | 🔲 |
| TC-ADM-SET-005 | 字典管理-条目 CRUD | P1 | FUNC | 🔲 |
| TC-ADM-SET-006 | 字典条目-批量排序 | P2 | FUNC | 🔲 |

---

## 18. 角色权限（ROL）— `pages/roles/index.vue`

| TC ID | 名称 | P | 类型 | 状态 |
|-------|------|---|------|------|
| TC-ADM-ROL-001 | 角色列表 | P1 | FUNC | 🔲 |
| TC-ADM-ROL-002 | 创建/编辑角色 | P1 | FUNC | 🔲 |
| TC-ADM-ROL-003 | 删除角色 | P1 | FUNC | 🔲 |
| TC-ADM-ROL-004 | 权限树-查看 | P2 | FUNC | 🔲 |

---

## 19. 管理员（ADM）— `pages/admins/index.vue`

| TC ID | 名称 | P | 类型 | 状态 |
|-------|------|---|------|------|
| TC-ADM-ADM-001 | 管理员列表 | P1 | FUNC | 🔲 |
| TC-ADM-ADM-002 | 创建管理员 | P1 | FUNC | 🔲 |
| TC-ADM-ADM-003 | 重置密码 | P1 | FUNC | 🔲 |
| TC-ADM-ADM-004 | 编辑/删除管理员 | P1 | FUNC | 🔲 |

---

## 20. 数据导出（EXP）— `pages/export/index.vue`

| TC ID | 名称 | P | 类型 | 状态 |
|-------|------|---|------|------|
| TC-ADM-EXP-001 | 导出出勤数据 | P1 | FUNC | 🔲 |
| TC-ADM-EXP-002 | 导出学生数据 | P1 | FUNC | 🔲 |
| TC-ADM-EXP-003 | 导出成长数据 | P1 | FUNC | 🔲 |
| TC-ADM-EXP-004 | 下载导出文件 | P1 | FUNC | 🔲 |

---

## 21. 操作日志（LOG）— `pages/logs/index.vue`

| TC ID | 名称 | P | 类型 | 状态 |
|-------|------|---|------|------|
| TC-ADM-LOG-001 | 操作日志列表-分页 | P2 | FUNC | 🔲 |

---

## 22. 晨检统计（HTH）— `pages/health/index.vue`

| TC ID | 名称 | P | 类型 | 状态 |
|-------|------|---|------|------|
| TC-ADM-HTH-001 | 晨检统计 | P1 | FUNC | 🔲 |

---

## 23. RBAC 权限隔离

| TC ID | 名称 | P | 类型 | 状态 |
|-------|------|---|------|------|
| TC-ADM-RBAC-001 | Admin Token 访问 Teacher API → 403 | P0 | RBAC | ✅ |
| TC-ADM-RBAC-002 | Admin Token 访问 Parent API → 403 | P0 | RBAC | 🔲 |
| TC-ADM-RBAC-003 | 未登录访问 Admin API → 401 | P0 | RBAC | 🔲 |
| TC-ADM-RBAC-004 | 非超管无权限操作 | P1 | RBAC | 🔲 |

---

## 24. 跨端数据联动（CROSS）

| TC ID | 名称 | P | 类型 | 状态 |
|-------|------|---|------|------|
| TC-ADM-CROSS-001 | Admin 创建通知→老师端可见 | P0 | CROSS | ✅ |
| TC-ADM-CROSS-002 | Admin 创建任务→老师端可见 | P0 | CROSS | ✅ |
| TC-ADM-CROSS-003 | Admin 创建通知→家长端可见 | P0 | CROSS | 🔲 |
| TC-ADM-CROSS-004 | Admin 审核请假→老师端状态更新 | P0 | CROSS | 🔲 |
| TC-ADM-CROSS-005 | 老师创建成长记录→Admin 可见 | P0 | CROSS | ✅ |
| TC-ADM-CROSS-006 | 家长提交请假→Admin 可见 | P0 | CROSS | 🔲 |

---

## 测试覆盖汇总

| 模块 | 用例数 | P0 | P1 | P2 | 已通过 |
|------|:------:|:--:|:--:|:--:|:------:|
| AUTH 登录认证 | 8 | 5 | 3 | 0 | 1 |
| DASH 数据看板 | 6 | 1 | 4 | 1 | 1 |
| CLS 班级管理 | 8 | 3 | 5 | 0 | 0 |
| STD 学生管理 | 8 | 2 | 6 | 0 | 0 |
| TCH 老师管理 | 5 | 1 | 4 | 0 | 0 |
| PRT 家长管理 | 4 | 1 | 3 | 0 | 0 |
| ATT 出勤管理 | 3 | 2 | 1 | 0 | 0 |
| LEV 请假管理 | 4 | 2 | 2 | 0 | 0 |
| PKU 接送管理 | 5 | 1 | 4 | 0 | 1 |
| GRW 成长记录 | 6 | 2 | 3 | 1 | 0 |
| TSK 亲子任务 | 5 | 1 | 4 | 0 | 1 |
| PNT 积分管理 | 4 | 0 | 3 | 1 | 0 |
| ACH 成就管理 | 3 | 0 | 2 | 1 | 0 |
| NOT 通知管理 | 5 | 2 | 3 | 0 | 1 |
| ANN 园所公告 | 4 | 0 | 4 | 0 | 0 |
| AUD 内容审核 | 3 | 0 | 2 | 1 | 0 |
| SET 系统设置 | 6 | 0 | 5 | 1 | 0 |
| ROL 角色权限 | 4 | 0 | 3 | 1 | 0 |
| ADM 管理员 | 4 | 0 | 4 | 0 | 0 |
| EXP 数据导出 | 4 | 0 | 4 | 0 | 0 |
| LOG 操作日志 | 1 | 0 | 0 | 1 | 0 |
| HTH 晨检统计 | 1 | 0 | 1 | 0 | 0 |
| RBAC 权限 | 4 | 3 | 1 | 0 | 1 |
| CROSS 跨端 | 6 | 6 | 0 | 0 | 3 |
| **合计** | **107** | **32** | **71** | **8** | **9** |
