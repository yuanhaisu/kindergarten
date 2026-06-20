# 测试用例目录

> 测试用例 ID 体系: `TC-{MODULE}-{NNNN}`
> 状态: 🔲 未实现 | 🚧 进行中 | ✅ 已通过 | ❌ 失败
> 优先级: P0 (必须) | P1 (应该) | P2 (可以)

## 模块缩写

| 缩写 | 模块 | 缩写 | 模块 |
|------|------|------|------|
| ATH | 认证 | ADT | 审核 |
| CLS | 班级 | ADM | 管理员 |
| STD | 学生 | ROL | 角色 |
| TCH | 教师 | SET | 设置 |
| PRN | 家长 | LOG | 日志 |
| ATT | 出勤 | EXP | 导出 |
| LEV | 请假 | DCT | 字典 |
| PKU | 接送 | CAL | 日历 |
| GRW | 成长 | DASH | 看板 |
| TSK | 任务 | COM | 通用 |
| PNT | 积分 | NTC | 通知 |
| ACH | 成就 | ANN | 公告 |

---

## 认证 (ATH)

| TC ID | 名称 | 优先级 | 类型 | 状态 |
|-------|------|--------|------|------|
| TC-ATH-001 | 密码登录-成功 | P0 | Unit | 🔲 |
| TC-ATH-002 | 密码登录-用户不存在 | P0 | Unit | 🔲 |
| TC-ATH-003 | 密码登录-密码错误 | P0 | Unit | 🔲 |
| TC-ATH-004 | 密码登录-非管理员 | P0 | Unit | 🔲 |
| TC-ATH-005 | Token 刷新 | P0 | Unit | 🔲 |
| TC-ATH-006 | 手机号脱敏 | P1 | Unit | ✅ |

## 班级 (CLS)

| TC ID | 名称 | 优先级 | 类型 | 状态 |
|-------|------|--------|------|------|
| TC-CLS-001 | 创建班级-成功 | P0 | Unit | ✅ |
| TC-CLS-002 | 创建班级-未绑定园所 | P0 | Unit | ✅ |
| TC-CLS-003 | 创建班级-无效年级 | P0 | Unit | ✅ |
| TC-CLS-004 | 创建班级-指定教师成功 | P0 | Unit | ✅ |
| TC-CLS-005 | 创建班级-重复教师 | P0 | Unit | ✅ |
| TC-CLS-006 | 创建班级-教师不存在 | P0 | Unit | ✅ |
| TC-CLS-007 | 删除班级-空班级成功 | P0 | Unit | ✅ |
| TC-CLS-008 | 删除班级-有学生拒绝 | P0 | Unit | ✅ |
| TC-CLS-009 | 班级列表-分页 | P0 | Unit | ✅ |
| TC-CLS-010 | 年级校验 | P1 | Unit | ✅ |
| TC-CLS-011 | 教师职称解析 | P1 | Unit | ✅ |
| TC-CLS-012 | 教师职称标签 | P1 | Unit | ✅ |

## 接送 (PKU)

| TC ID | 名称 | 优先级 | 类型 | 状态 |
|-------|------|--------|------|------|
| TC-PKU-001 | 接送概览-正常日期 | P0 | Unit / E2E R8 | 🔲 |
| TC-PKU-002 | 接送概览-无效日期 | P0 | Unit | 🔲 |
| TC-PKU-003 | 扫码接送-有效二维码 | P0 | Unit / E2E R9 | 🔲 |
| TC-PKU-004 | 手动接送-登记成功 | P0 | Unit | 🔲 |
| TC-PKU-005 | 手动接送-重复登记 | P0 | Unit | 🔲 |
| TC-PKU-006 | 异常标记 | P0 | Unit | 🔲 |
| TC-PKU-007 | 授权人审核-通过 | P0 | Unit / E2E | 🔲 |
| TC-PKU-008 | 授权人审核-拒绝 | P0 | Unit | 🔲 |

## 日历 (CAL)

| TC ID | 名称 | 优先级 | 类型 | 状态 |
|-------|------|--------|------|------|
| TC-CAL-001 | 日历事件-创建成功 | P0 | Unit | 🔲 |
| TC-CAL-002 | 日历事件-标题为空 | P0 | Unit | 🔲 |
| TC-CAL-003 | 日历事件-日期为空 | P0 | Unit | 🔲 |
| TC-CAL-004 | 日历事件-日期范围查询 | P0 | Unit | 🔲 |
| TC-CAL-005 | 日历事件-单条详情 | P1 | Unit | 🔲 |
| TC-CAL-006 | 日历事件-更新 | P1 | Unit | 🔲 |
| TC-CAL-007 | 日历事件-删除 | P1 | Unit | 🔲 |
| TC-CAL-008 | 日历标记点-按天聚合 | P1 | Unit | 🔲 |

## 成长 (GRW)

### 数据模型（8 张表）
`GrowthDailyLog` / `GrowthLogStudent` / `GrowthMedia` / `GrowthMediaStudent` / `GrowthComment` / `GrowthLike` / `GrowthReport` / `GrowthReportDimension`

### API 端点
- **老师端**: 10 个 — 列表/详情/创建/编辑/删除/评论(列表+创建)/点赞列表/关联学生/媒体上传
- **管理端**: 5 个 — 日志列表/详情/删除/审核/统计
- **家长端**: 7 个 — 成长时光流/详情/点赞+取消/评论(列表+创建)/相册/报告

### 核心业务逻辑（Service 层）

| TC ID | 名称 | 优先级 | 类型 | 状态 |
|-------|------|--------|------|------|
| TC-GRW-001 | 创建成长日志-全班模式(target_type=1) | P0 | Unit | 🔲 |
| TC-GRW-002 | 创建成长日志-指定学生(target_type=2) | P0 | Unit | 🔲 |
| TC-GRW-003 | 创建成长日志-标题/内容为空 | P0 | Unit | 🔲 |
| TC-GRW-004 | 创建成长日志-无效日期格式 | P0 | Unit | 🔲 |
| TC-GRW-005 | 编辑成长日志-部分字段更新 | P0 | Unit | 🔲 |
| TC-GRW-006 | 编辑成长日志-他人日志拒绝 | P0 | Unit | 🔲 |
| TC-GRW-007 | 删除成长日志-本人日志成功 | P0 | Unit | ✅ |
| TC-GRW-008 | 删除成长日志-他人日志拒绝 | P0 | Unit | ✅ |
| TC-GRW-009 | 成长日志列表-按日期筛选 | P0 | Unit | 🔲 |
| TC-GRW-010 | 成长日志列表-分页正常 | P0 | Unit | 🔲 |
| TC-GRW-011 | 成长日志列表-空数据 | P1 | Unit | 🔲 |
| TC-GRW-012 | 日志详情-含媒体/标签/评分 | P0 | Unit | 🔲 |
| TC-GRW-013 | 日志详情-日志不存在返回错误 | P1 | Unit | 🔲 |

### 评分与标签

| TC-GRW-020 | 用餐评分-有效范围(1-3) | P1 | Unit | ✅ |
| TC-GRW-021 | 用餐评分-无效值拒绝 | P1 | Unit | 🔲 |
| TC-GRW-022 | 睡眠质量-有效范围(1-3) | P1 | Unit | ✅ |
| TC-GRW-023 | 情绪评分-有效范围(1-3) | P1 | Unit | ✅ |
| TC-GRW-024 | 表现标签-空数组合法 | P1 | Unit | 🔲 |
| TC-GRW-025 | 表现标签-超出允许值拒绝 | P1 | Unit | 🔲 |
| TC-GRW-026 | 评分与标签枚举常量正确性 | P1 | Unit | ✅ |

### 纯函数

| TC-GRW-027 | auditStatusString 所有状态 | P1 | Unit | ✅ |
| TC-GRW-028 | growthContentPreview 截断逻辑 | P1 | Unit | ✅ |
| TC-GRW-029 | IsGrowthNotFound 错误判断 | P1 | Unit | ✅ |

### 评论

| TC-GRW-030 | 评论列表-正常加载 | P1 | Unit | 🔲 |
| TC-GRW-031 | 评论列表-包含嵌套回复 | P1 | Unit | 🔲 |
| TC-GRW-032 | 创建评论-一级评论成功 | P1 | Unit | 🔲 |
| TC-GRW-033 | 创建评论-回复评论(parent_id) | P1 | Unit | 🔲 |
| TC-GRW-034 | 创建评论-内容为空拒绝 | P1 | Unit | 🔲 |
| TC-GRW-035 | 创建评论-日志不存在拒绝 | P1 | Unit | 🔲 |

### 点赞

| TC-GRW-040 | 点赞-首次点赞成功 | P1 | Unit | 🔲 |
| TC-GRW-041 | 点赞-重复点赞去重 | P1 | Unit | 🔲 |
| TC-GRW-042 | 取消点赞-成功 | P1 | Unit | 🔲 |
| TC-GRW-043 | 取消点赞-未点赞幂等 | P1 | Unit | 🔲 |
| TC-GRW-044 | 点赞列表-正常加载 | P1 | Unit | 🔲 |

### 审核（管理端）

| TC-GRW-050 | 审核日志-通过(pass) | P0 | Unit | ✅ |
| TC-GRW-051 | 审核日志-驳回(reject) | P0 | Unit | ✅ |
| TC-GRW-052 | 审核日志-无效action拒绝 | P1 | Unit | ✅ |
| TC-GRW-053 | 审核日志-日志不存在 | P1 | Unit | ✅ |

### 统计

| TC-GRW-060 | 成长统计-日期范围正常 | P1 | Unit | 🔲 |
| TC-GRW-061 | 成长统计-空范围回退默认 | P2 | Unit | 🔲 |
| TC-GRW-062 | 成长统计-班级维度数据正确 | P2 | Unit | 🔲 |

### 边界与异常

| TC-GRW-070 | 未绑定园所拒绝操作 | P0 | Unit | ✅ |
| TC-GRW-071 | 不同园所数据隔离 | P0 | Unit | 🔲 |
| TC-GRW-072 | sync_to_feed=false 不在时光流出现 | P1 | Unit | 🔲 |
| TC-GRW-073 | Bean数据回填(ensureBootstrap)幂等 | P2 | Integration | 🔲 |
| TC-GRW-074 | 并行点赞/评论不产生竞态 | P2 | Integration | 🔲 |

### E2E（管理端 Round 10, 当前跳过）

| TC ID | 名称 | 状态 |
|-------|------|------|
| TC-GRW-E01 | 成长日志列表-至少10条 | ⏸️ 跳过（老师端未发布数据） |
| TC-GRW-E02 | 成长日志详情-打开5条 | ⏸️ |
| TC-GRW-E03 | 审核日志-至少4条审核通过/驳回 | ⏸️ |
| TC-GRW-E04 | 删除日志-删除1条 | ⏸️ |
| TC-GRW-E05 | 成长统计-与列表量级匹配 | ⏸️ |

---

## 老师端（Teacher Mini-Program）产品测试用例

| TC ID 前缀 | 模块 | 页面 | 用例数 |
|-----------|------|------|--------|
| TC-TCH-HOME | 首页工作台 | `pages/index/index` | 17 |
| TC-TCH-NOTI | 通知管理 | `pages/notice/create` | 36 |
| TC-TCH-LEAV | 请假审批 | `pages/leave/index` | 15 |
| TC-TCH-GRWT | 成长记录 | `pages/growth/*` | 39 |
| TC-TCH-PKUT | 接送管理 | `pages/pickup/*` | 34 |
| TC-TCH-CAL | 班级日历 | `pages/calendar/*` | 25 |
| TC-TCH-CLAS | 班级管理 | `pages/class/index` | 6 |
| TC-TCH-TASK | 亲子任务 | `pages/tasks/index` | 17 |
| TC-TCH-DOCT | 入园晨检 | `pages/doctor/checkin` | 4 |
| TC-TCH-MSG | 消息中心 | `pages/messages/index` | 4 |
| TC-TCH-GLB | 全局通用 | 跨模块 | 17 |
| **合计** | **10 模块** | **20 页面** | **214** |

> 详见 [TEACHER_TEST_CASES.md](./TEACHER_TEST_CASES.md)

## 管理端（Admin Web）产品测试用例

| TC ID 前缀 | 模块 | 用例数 |
|-----------|------|:------:|
| TC-ADM-AUTH | 登录认证 | 8 |
| TC-ADM-DASH | 数据看板 | 6 |
| TC-ADM-CLS | 班级管理 | 8 |
| TC-ADM-STD | 学生管理 | 8 |
| TC-ADM-TCH | 老师管理 | 5 |
| TC-ADM-PRT | 家长管理 | 4 |
| TC-ADM-ATT | 出勤管理 | 3 |
| TC-ADM-LEV | 请假管理 | 4 |
| TC-ADM-PKU | 接送管理 | 5 |
| TC-ADM-GRW | 成长记录 | 6 |
| TC-ADM-TSK | 亲子任务 | 5 |
| TC-ADM-PNT | 积分管理 | 4 |
| TC-ADM-ACH | 成就管理 | 3 |
| TC-ADM-NOT | 通知管理 | 5 |
| TC-ADM-ANN | 园所公告 | 4 |
| TC-ADM-AUD | 内容审核 | 3 |
| TC-ADM-SET | 系统设置 | 6 |
| TC-ADM-ROL | 角色权限 | 4 |
| TC-ADM-ADM | 管理员 | 4 |
| TC-ADM-EXP | 数据导出 | 4 |
| TC-ADM-LOG | 操作日志 | 1 |
| TC-ADM-HTH | 晨检统计 | 1 |
| TC-ADM-RBAC | 权限隔离 | 4 |
| TC-ADM-CROSS | 跨端联动 | 6 |
| **合计** | **24 模块** | **107** |

> 详见 [ADMIN_TEST_CASES.md](./ADMIN_TEST_CASES.md)

## 家长端（Parent Mini-Program）产品测试用例

| TC ID 前缀 | 模块 | 用例数 |
|-----------|------|:------:|
| TC-PRN-AUTH | 登录绑定 | 7 |
| TC-PRN-HOME | 首页 | 6 |
| TC-PRN-NOTI | 通知 | 4 |
| TC-PRN-LEAV | 请假 | 7 |
| TC-PRN-GRW | 成长记录 | 10 |
| TC-PRN-PKU | 接送 | 8 |
| TC-PRN-TSK | 亲子任务 | 6 |
| TC-PRN-PNT | 积分 | 3 |
| TC-PRN-ACH | 成就 | 1 |
| TC-PRN-CLS | 班级 | 2 |
| TC-PRN-STD | 学生 | 3 |
| TC-PRN-FBK | 反馈 | 2 |
| TC-PRN-MSG | 消息 | 1 |
| TC-PRN-SET | 设置 | 1 |
| TC-PRN-PROF | 个人中心 | 2 |
| TC-PRN-GLB | 全局通用 | 8 |
| **合计** | **16 模块** | **71** |

> 详见 [PARENT_TEST_CASES.md](./PARENT_TEST_CASES.md)

---

> 更多模块的测试用例将在后续 Phase 中补充。
> 每个 TC-ID 对应的验收标准详见各模块的 Spec 文档 (`docs/specs/`)。
