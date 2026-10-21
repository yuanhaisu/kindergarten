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
| PERM | 权限强制 | TBL | 课表 |

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
| TC-NEW-ATH-0001 | 微信登录-老师端首次建号 | P0 | Unit / E2E | 🚧 |
| TC-NEW-ATH-0002 | 微信登录-老师端再次登录命中本端 | P0 | Unit / E2E | 🚧 |
| TC-NEW-ATH-0003 | 微信登录-家长端首次建号 | P0 | Unit / E2E | 🚧 |
| TC-NEW-ATH-0004 | 微信登录-换端登录不命中另一端 | P0 | Unit / E2E | 🚧 |
| TC-NEW-ATH-0005 | 微信登录-该端凭据未配置显式报错 | P0 | Unit | ✅ |
| TC-NEW-ATH-0006 | 微信登录-换取会话被拒透传 errcode | P0 | Unit | ✅ |
| TC-NEW-ATH-0007 | 微信登录-邀请码绑定预创建教师 | P0 | Unit / E2E | 🚧 |
| TC-NEW-ATH-0008 | 微信登录-邀请码无效或已使用 | P0 | Unit / E2E | 🚧 |
| TC-NEW-ATH-0009 | 跨端关联-两端均有 unionid | P1 | Unit | ✅ |
| TC-NEW-ATH-0010 | 跨端关联-两端 unionid 不同判非同人 | P1 | Unit | ✅ |
| TC-NEW-ATH-0011 | 跨端关联-无 unionid 按手机号兜底 | P1 | Unit | ✅ |
| TC-NEW-ATH-0012 | 跨端关联-无 unionid 且手机号不符不关联 | P1 | Unit | ✅ |
| TC-NEW-ATH-0013 | 登录契约-响应结构保持稳定 | P0 | Contract | ✅ |
| TC-NEW-ATH-0014 | 跨端关联-关联不修改账号归属 | P0 | Unit | ✅ |

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

### 接送托管证据（pickup-custody-evidence）

| TC ID | 名称 | 优先级 | 类型 | 状态 |
|-------|------|--------|------|------|
| TC-NEW-PKU-001 | 家长指定今日接走人-指定成功 | P0 | Unit / E2E | ✅ |
| TC-NEW-PKU-002 | 指定未审核通过者被拒 | P0 | Unit | ✅ |
| TC-NEW-PKU-003 | 指定已撤销或已过期者被拒 | P0 | Unit | ✅ |
| TC-NEW-PKU-004 | 重复指定覆盖旧值 | P0 | Unit | ✅ |
| TC-NEW-PKU-010 | 凭证携带接走人可读信息 | P0 | Unit / E2E | ✅ |
| TC-NEW-PKU-011 | 生成时校验接走人资格 | P0 | Unit | ✅ |
| TC-NEW-PKU-012 | 凭证不泄露可枚举学生标识 | P0 | Unit | ✅ |
| TC-NEW-PKU-013 | 凭证当日过期 | P0 | Unit | ✅ |
| TC-NEW-PKU-014 | 凭证一次有效 | P0 | Unit | ✅ |
| TC-NEW-PKU-020 | 扫码成功并归因到接走人 | P0 | Unit / E2E | ✅ |
| TC-NEW-PKU-021 | 接走人授权在凭证生成后被撤销 | P0 | Unit | ✅ |
| TC-NEW-PKU-022 | 凭证已失效被拒 | P0 | Unit | ✅ |
| TC-NEW-PKU-030 | 手动登记-从授权人列表选人 | P0 | Unit | ✅ |
| TC-NEW-PKU-031 | 手动登记-标记为非授权或临时接送 | P0 | Unit | ✅ |
| TC-NEW-PKU-032 | 例外不阻断登记 | P0 | Unit | ✅ |
| TC-NEW-PKU-040 | 同日多次往返可登记 | P0 | Unit | ✅ |
| TC-NEW-PKU-041 | 连续同向被拒 | P0 | Unit | ✅ |
| TC-NEW-PKU-042 | 状态以最后一次转移为准 | P0 | Unit | ✅ |
| TC-NEW-PKU-050 | 首次接走理由可选 | P0 | Unit | ✅ |
| TC-NEW-PKU-051 | 重复接走缺理由被拒 | P0 | Unit | ✅ |
| TC-NEW-PKU-052 | 重复接走带理由成功 | P0 | Unit | ✅ |
| TC-NEW-PKU-053 | 重复送入不强制理由 | P0 | Unit | ✅ |
| TC-NEW-PKU-060 | 提前接走缺理由被拒 | P0 | Unit | ✅ |
| TC-NEW-PKU-061 | 提前接走带理由成功 | P0 | Unit | ✅ |
| TC-NEW-PKU-062 | 放学后接走不要求理由 | P0 | Unit | ✅ |
| TC-NEW-PKU-063 | 放学时刻按班级配置 | P0 | Unit | ✅ |
| TC-NEW-PKU-070 | 送入不记录送托人 | P0 | Unit | ✅ |
| TC-NEW-PKU-071 | 送入不要求理由 | P0 | Unit | ✅ |
| TC-NEW-PKU-080 | 老师手动标记异常 | P0 | Unit | ✅ |
| TC-NEW-PKU-081 | 老师手动取消异常 | P0 | Unit | ✅ |
| TC-NEW-PKU-082 | 系统不自动置异常 | P0 | Unit | ✅ |
| TC-NEW-PKU-090 | 老师看到本班滞留名单 | P0 | Unit | ✅ |
| TC-NEW-PKU-091 | 管理员看到全园滞留名单 | P0 | Unit | ✅ |
| TC-NEW-PKU-100 | 一键通知成功投递 | P0 | Unit | ✅ |
| TC-NEW-PKU-101 | 通知留痕可查 | P0 | Unit | ✅ |
| TC-NEW-PKU-110 | 超时未接被计数 | P0 | Unit | ✅ |
| TC-NEW-PKU-111 | 随后被接走不重复计数 | P0 | Unit | ✅ |
| TC-NEW-PKU-120 | 开启通知时推送家长 | P0 | Unit | ✅ |
| TC-NEW-PKU-121 | 关闭通知时不推送 | P0 | Unit | ✅ |
| TC-NEW-PKU-122 | 授权例外的接走也按配置通知 | P0 | Unit | ✅ |

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

> 本组用例对应 `openspec/changes/align-class-calendar/specs/class-calendar/spec.md` 的 18 个 Scenario；
> 单测见 `server/internal/service/calendar_semantics_test.go` / `dict_calendar_type_test.go` / `garden_calendar_test.go`，
> 端页面与跨端数据流已随组 7/8/10 验收（变更 `align-class-calendar`）。

| TC ID | 名称 | 优先级 | 类型 | 状态 |
|-------|------|--------|------|------|
| TC-NEW-CAL-0001 | 类型字典-内置类型按字典渲染 | P0 | Unit / E2E | ✅ |
| TC-NEW-CAL-0002 | 类型字典-园所自定义类型全端生效 | P1 | E2E | ✅ |
| TC-NEW-CAL-0003 | 类型字典-内置类型不可删除 | P1 | Unit | ✅ |
| TC-NEW-CAL-0004 | 受众范围-园级事件全园可见 | P0 | Unit | ✅ |
| TC-NEW-CAL-0005 | 受众范围-班级事件跨班不可见 | P0 | Unit | ✅ |
| TC-NEW-CAL-0006 | 受众范围-老师对园级事件只读 | P0 | Unit | ✅ |
| TC-NEW-CAL-0007 | 发布状态-草稿对家长不可见 | P0 | Unit | ✅ |
| TC-NEW-CAL-0008 | 发布状态-发布后家长可见（含字典标签/颜色） | P0 | Unit | ✅ |
| TC-NEW-CAL-0009 | 发布状态-已发布回退草稿 | P1 | Unit | ✅ |
| TC-NEW-CAL-0010 | 推送-创建时确认通知生成通知并回写记录位 | P0 | Unit | ✅ |
| TC-NEW-CAL-0011 | 推送-选择不通知不生成但事件照常生效 | P0 | Unit | ✅ |
| TC-NEW-CAL-0012 | 推送-园级通知面向全园家长 | P0 | Unit | ✅ |
| TC-NEW-CAL-0013 | 推送-变更保存确认通知（含【变更】标题） | P0 | Unit | ✅ |
| TC-NEW-CAL-0014 | 变更提示-时间变更产生显式要旨 | P0 | Unit | ✅ |
| TC-NEW-CAL-0015 | 变更提示-无变更不提示、已发生不下发 | P1 | Unit | ✅ |
| TC-NEW-CAL-0016 | 园级发布-园长发布成功（scope=kindergarten、class_id 为 NULL） | P0 | Unit | ✅ |
| TC-NEW-CAL-0017 | 园级发布-老师路径仅班级级/园级编辑被拒 | P0 | Unit | ✅ |
| TC-NEW-CAL-0018 | 园级发布-保健医发布体检 | P0 | Unit / E2E | ✅ |

## 课表 (TBL)

> 本组用例对应 `openspec/changes/align-class-calendar/specs/class-timetable/spec.md` 的 10 个 Scenario，
> 实现见 `server/internal/service/timetable_sync_test.go`（全部已覆盖）。

| TC ID | 名称 | 优先级 | 类型 | 状态 |
|-------|------|--------|------|------|
| TC-NEW-TBL-0001 | 开关-关闭隐藏课表实例（数据保留）、开启恢复 | P0 | Unit | ✅ |
| TC-NEW-TBL-0002 | 模板-开关开启后可维护并同步 | P0 | Unit | ✅ |
| TC-NEW-TBL-0003 | 模板-年级模板生效到无微调班级 | P0 | Unit | ✅ |
| TC-NEW-TBL-0004 | 微调-班级微调覆盖模板（仅本班） | P0 | Unit | ✅ |
| TC-NEW-TBL-0005 | 物化-课表按窗口出现在两端口历 | P0 | Unit | ✅ |
| TC-NEW-TBL-0006 | 物化-窗口滚动无断档且重跑幂等 | P0 | Unit | ✅ |
| TC-NEW-TBL-0007 | 物化-同步不产生任何通知 | P0 | Unit | ✅ |
| TC-NEW-TBL-0008 | 覆盖-老师调整被保留 | P0 | Unit | ✅ |
| TC-NEW-TBL-0009 | 覆盖-取消不复活（软删行占位） | P0 | Unit | ✅ |
| TC-NEW-TBL-0010 | 覆盖-模板变更只影响未来未覆盖实例 | P0 | Unit | ✅ |

## 通知 (NTC)

> 本组用例对应 `openspec/changes/notice-multitarget/specs/notice-targeting/spec.md` 的 13 个 Scenario，
> 实现见 `server/internal/repository/notice_target_test.go` 与 `server/internal/service/notice_multitarget_test.go`。

| TC ID | 名称 | 优先级 | 类型 | 状态 |
|-------|------|--------|------|------|
| TC-NEW-NTC-0001 | 多班发布-老师一次发给两个班 | P0 | Unit | ✅ |
| TC-NEW-NTC-0002 | 多班发布-非任教班被拒 | P0 | Unit | ✅ |
| TC-NEW-NTC-0003 | 多班发布-目标为空被拒 | P0 | Unit | ✅ |
| TC-NEW-NTC-0004 | 年级圈选-快照展开 | P0 | Unit | ✅ |
| TC-NEW-NTC-0005 | 年级圈选-与班级混用去重 | P0 | Unit | ✅ |
| TC-NEW-NTC-0006 | 多孩合并-两孩分属两个目标班只出一条 | P0 | Unit | ✅ |
| TC-NEW-NTC-0007 | 可见范围-仅一孩命中目标也可见 | P0 | Unit | ✅ |
| TC-NEW-NTC-0008 | 可见范围-家长不越界可见 | P0 | Unit | ✅ |
| TC-NEW-NTC-0009 | 可见范围-老师命中任教班可查 | P0 | Unit | ✅ |
| TC-NEW-NTC-0010 | 可见范围-非目标班老师不可见 | P0 | Unit | ✅ |
| TC-NEW-NTC-0011 | 已读率分母-双孩家长计一 | P0 | Unit | ✅ |
| TC-NEW-NTC-0012 | 已读率-两端数值一致 | P0 | Unit | ✅ |
| TC-NEW-NTC-0013 | 迁移-存量单班通知行为等价 | P0 | Unit | ✅ |

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
| TC-GRW-005 | 编辑成长日志-部分字段更新 | P0 | Unit | ✅ |
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

## 晨检 (HTH)

> 来源：openspec/changes/morning-health-closed-loop/specs/morning-health/spec.md
> 服务端测试见 `server/internal/service/health_check_service_test.go`。

| TC ID | 名称 | 优先级 | 类型 | 状态 |
|-------|------|--------|------|------|
| TC-NEW-HTH-0001 | 家长查看本人孩子已完成晨检 | P0 | Unit / E2E | ✅ |
| TC-NEW-HTH-0002 | 家长不可查看他人孩子 | P0 | Unit / E2E | ✅ |
| TC-NEW-HTH-0003 | 待检与跳过不作为结论展示 | P0 | Unit | ✅ |
| TC-NEW-HTH-0004 | 异常项以可读文字呈现 | P1 | Unit / E2E | ✅ |
| TC-NEW-HTH-0005 | 需观察通知全部已绑定家长 | P0 | Unit / E2E | ✅ |
| TC-NEW-HTH-0006 | 劝返且隔离时标题标注紧急 | P1 | Unit | ✅ |
| TC-NEW-HTH-0007 | 健康结论不通知 | P0 | Unit | ✅ |
| TC-NEW-HTH-0008 | 重复提交不产生重复通知 | P0 | Unit | ✅ |
| TC-NEW-HTH-0009 | 通知失败可见且不回滚结论 | P0 | Unit | ✅ |
| TC-NEW-HTH-0010 | 异常部位拍照并关联 | P0 | Unit / E2E | ✅ |
| TC-NEW-HTH-0011 | 跨园所关联照片被拒 | P0 | Unit / E2E | ✅ |
| TC-NEW-HTH-0012 | 晨检详情可读取照片 | P1 | E2E | ✅ |
| TC-NEW-HTH-0013 | 多部位异常产出标签并集 | P0 | Unit | ✅ |
| TC-NEW-HTH-0014 | 体温达到阈值产出发烧标签 | P0 | Unit | ✅ |
| TC-NEW-HTH-0015 | 非法部位取值被拒 | P0 | Unit | ✅ |
| TC-NEW-HTH-0016 | 客户端症状标签不生效 | P0 | Unit / Contract | ✅ |
| TC-NEW-HTH-0017 | 跨园所读取详情被拒 | P0 | Unit / E2E | ✅ |
| TC-NEW-HTH-0018 | 跨园所提交结论被拒 | P0 | Unit / E2E | ✅ |
| TC-NEW-HTH-0019 | 待检队列不含他园记录 | P0 | Unit | ✅ |
| TC-NEW-HTH-0020 | 已完成记录重复提交被拒 | P0 | Unit / E2E | ✅ |
| TC-NEW-HTH-0021 | 劝返不自动标记接送异常 | P0 | Unit | ✅ |
| TC-NEW-HTH-0022 | 健康与需观察不标记接送异常 | P1 | Unit | ✅ |

## 体温策略 (HTP)

> 来源：openspec/changes/health-temperature-policy/specs/health-temperature-policy/spec.md
> 服务端测试见 `server/internal/service/settings_service_test.go` 与 `health_check_service_test.go`。

| TC ID | 名称 | 优先级 | 类型 | 状态 |
|-------|------|--------|------|------|
| TC-NEW-HTP-0001 | 未配置时使用默认策略（按需测温 / 37.3） | P0 | Unit | ✅ |
| TC-NEW-HTP-0002 | 管理员更新全员必测与阈值 | P0 | Unit / E2E | ✅ |
| TC-NEW-HTP-0003 | 阈值越界被拒 | P0 | Unit | ✅ |
| TC-NEW-HTP-0004 | 按需测温策略随队列下发且页面可见 | P0 | Unit / E2E | ✅ |
| TC-NEW-HTP-0005 | 全员必测策略在医生页可见 | P0 | E2E | ✅ |
| TC-NEW-HTP-0006 | 必填模式下缺体温被拒 | P0 | Unit / E2E | ✅ |
| TC-NEW-HTP-0007 | 必填模式下有体温通过 | P0 | Unit | ✅ |
| TC-NEW-HTP-0008 | 未到园豁免体温 | P0 | Unit | ✅ |
| TC-NEW-HTP-0009 | 按需测温允许不录且存 NULL | P0 | Unit | ✅ |
| TC-NEW-HTP-0010 | 达到园所阈值判为发烧 | P0 | Unit | ✅ |
| TC-NEW-HTP-0011 | 低于园所阈值不判发烧 | P0 | Unit | ✅ |
| TC-NEW-HTP-0012 | 阈值变更不回溯旧记录 | P1 | Unit | ✅ |

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

### 成长时光 feed 呈现（家长端）

> 对应 OpenSpec 变更 `align-parent-growth-feed`。字段溯源见 `docs/growth-feed-field-review.md`。

| TC ID | 名称 | 优先级 | 类型 | 状态 |
|-------|------|--------|------|------|
| TC-GRW-080 | 卡片显示发布者昵称 | P0 | Unit | ✅ |
| TC-GRW-081 | 卡片显示发布时刻 | P0 | Unit | ✅ |
| TC-GRW-082 | 卡片显示正文 | P0 | Unit | ✅ |
| TC-GRW-083 | 有照片的记录显示照片 | P0 | Unit | ✅ |
| TC-GRW-084 | 无照片记录不渲染照片区 | P1 | Unit | ✅ |
| TC-GRW-085 | 点赞态正确反映本人操作 | P0 | Unit | ✅ |
| TC-GRW-086 | 评论数真实 | P1 | Unit | ✅ |
| TC-GRW-087 | 同月记录归入同一分组 | P1 | UI | 🔲 |
| TC-GRW-088 | 跨月记录分组隔离 | P1 | UI | 🔲 |
| TC-GRW-089 | 个体记录列表项不含相关孩子 | P0 | Unit | ✅ |
| TC-GRW-090 | 详情给出相关孩子 | P0 | Integration | 🔲 |
| TC-GRW-091 | 全班记录详情无相关孩子 | P1 | Integration | 🔲 |
| TC-GRW-092 | 列表响应不含冗余字段 | P0 | Unit | ✅ |

### 成长时光孩子卡片统计（家长端）

> 对应 P0（成就徽章止血），详见 `docs/achievement-badge-review.md` §7。

| TC ID | 名称 | 优先级 | 类型 | 状态 |
|-------|------|--------|------|------|
| TC-GRW-093 | 孩子卡片「获得奖章」显示真实累计数 | P1 | Integration | ✅ |
| TC-GRW-094 | 孩子卡片「成长记录」显示真实总数 | P1 | Integration | ✅ |
| TC-GRW-095 | 成就接口失败不影响成长时光列表加载 | P1 | Integration | 🔲 |

### 家长端成长记录可见性（对应 OpenSpec 变更 `fix-growth-visibility-teacher-preview`）

> 规则定义见 `openspec/specs/growth-visibility/spec.md`（归档前为变更内 delta）。
> 0001/0002/0004 由纯函数单测 `server/internal/service/parent_growth_visibility_test.go` 覆盖；
> 其余场景为 DB 行为，待集成验证（TC 类型标注为 Integration 的项）。

| TC ID | 名称 | 优先级 | 类型 | 状态 |
|-------|------|--------|------|------|
| TC-NEW-VIS-0001 | 可见性由记录与孩子的关联链决定（条件构造） | P0 | Unit | ✅ |
| TC-NEW-VIS-0002 | 发给其他孩子的个体记录对家长不可见（条件构造） | P0 | Unit | ✅ |
| TC-NEW-VIS-0003 | 非本班记录与媒体不可见 | P0 | Integration | 🔲 |
| TC-NEW-VIS-0004 | 首页动态不出现其他孩子被记录的个体记录（条件构造） | P0 | Unit | ✅ |
| TC-NEW-VIS-0005 | 首页收到发给当前孩子的个体记录 | P0 | Integration | 🔲 |
| TC-NEW-VIS-0006 | 相册按月份过滤时仍遵守可见性 | P0 | Integration | 🔲 |
| TC-NEW-VIS-0007 | 相册不包含其他孩子的个体记录媒体 | P0 | Integration | 🔲 |
| TC-NEW-VIS-0008 | 报告：指定他人孩子 ID 被拒绝 | P0 | Integration | 🔲 |
| TC-NEW-VIS-0009 | 报告：跨园所孩子 ID 被拒绝 | P0 | Integration | 🔲 |
| TC-NEW-VIS-0010 | 报告：未指定孩子时回退当前选中孩子 | P1 | Integration | 🔲 |
| TC-NEW-VIS-0011 | 报告：生成中的报告不可见 | P1 | Integration | 🔲 |
| TC-NEW-VIS-0012 | 报告：已生成报告正常返回 | P1 | Integration | 🔲 |

### 老师端发布与预览（对应 OpenSpec 变更 `fix-growth-visibility-teacher-preview`）

> 规则定义见 `openspec/specs/teacher-growth-authoring/spec.md`。
> 0006/0007 依赖 DB 与页面交互（集成/UI）；其余为响应形状与渲染核对。

| TC ID | 名称 | 优先级 | 类型 | 状态 |
|-------|------|--------|------|------|
| TC-NEW-TGR-0001 | 详情呈现发布者与互动统计 | P0 | Integration | 🔲 |
| TC-NEW-TGR-0002 | 列表呈现照片预览与同步状态 | P0 | Integration | 🔲 |
| TC-NEW-TGR-0003 | 列表呈现发布者 | P0 | Integration | 🔲 |
| TC-NEW-TGR-0004 | 不渲染无来源的浏览统计 | P1 | UI | 🔲 |
| TC-NEW-TGR-0005 | 同步状态由 sync_to_feed 驱动 | P1 | UI | 🔲 |
| TC-NEW-TGR-0006 | 编辑提交更新原记录（计数不变） | P0 | Integration | 🔲 |
| TC-NEW-TGR-0007 | 编辑态载入既有内容 | P0 | UI | 🔲 |

### 老师端成长记录目标学生选择（对应 OpenSpec 变更 `teacher-growth-student-picker`）

> 规则定义见 `openspec/specs/growth-target-students/spec.md`（归档前为变更内 delta）。
> 0001/0002 的查询口径由 service 层 `classStudentIDs` 承担（集成验证）；
> 0003/0004/0005/0007 为页面交互，走 CDP 真实点击验证。

| TC ID | 名称 | 优先级 | 类型 | 状态 |
|-------|------|--------|------|------|
| TC-NEW-TGP-0001 | 候选只列出本班在读学生 | P0 | Integration | 🔲 |
| TC-NEW-TGP-0002 | 非在读学生不出现在候选 | P0 | Integration | 🔲 |
| TC-NEW-TGP-0003 | 点击学生立即高亮 | P0 | UI | 🔲 |
| TC-NEW-TGP-0004 | 再次点击取消选中 | P0 | UI | 🔲 |
| TC-NEW-TGP-0005 | 按姓名过滤候选 | P0 | UI | 🔲 |
| TC-NEW-TGP-0006 | 搜索无结果与空班级的提示区分 | P1 | UI | 🔲 |
| TC-NEW-TGP-0007 | 已选计数随操作更新 | P0 | UI | 🔲 |
| TC-NEW-TGP-0008 | 编辑时回显已选学生 | P0 | Integration | 🔲 |
| TC-NEW-TGP-0009 | 改名次后关联随保存更新 | P0 | Integration | 🔲 |
| TC-NEW-TGP-0010 | 改为全班记录清除学生关联 | P0 | Integration | 🔲 |
| TC-NEW-TGP-0011 | 目标学生关联更新幂等 | P0 | Integration | 🔲 |

### 素材管理基线（对应 `openspec/specs/media-assets/`，架构见 `docs/media-management-design.md`）

> 基线规格：7 需求 / 11 场景。以下 TC 是**跨业务复用的基线验收项**——每个新接入素材的业务都应引用对应 TC 验证，而不是各自新造。
> 0001/0002/0003/0005~0010 在首个接入方（`wire-growth-media-upload`）实现时覆盖；0004（标签字典驱动）与 0011（统一通道）随素材库/各业务接入逐步覆盖。

| TC ID | 名称 | 优先级 | 类型 | 状态 |
|-------|------|--------|------|------|
| TC-NEW-MAS-0001 | 同一文件多条引用各有描述 | P0 | Integration | 🔲 |
| TC-NEW-MAS-0002 | 编辑某条引用不影响其他引用 | P1 | Integration | 🔲 |
| TC-NEW-MAS-0003 | 新业务接入不改文件层 | P1 | Integration | 🔲 |
| TC-NEW-MAS-0004 | 标签来自字典 | P1 | Integration | 🔲 |
| TC-NEW-MAS-0005 | 替换后引用指向新文件 | P1 | Integration | 🔲 |
| TC-NEW-MAS-0006 | 移除引用不删文件 | P0 | Integration | 🔲 |
| TC-NEW-MAS-0007 | 有引用的文件不可删除 | P0 | Integration | 🔲 |
| TC-NEW-MAS-0008 | 无引用的文件可删除（软删） | P1 | Integration | 🔲 |
| TC-NEW-MAS-0009 | 跨业务可见性各自生效 | P0 | Integration | 🔲 |
| TC-NEW-MAS-0010 | 文件层不提供可见性开关 | P1 | Unit | 🔲 |
| TC-NEW-MAS-0011 | 新业务复用统一通道 | P0 | Integration | 🔲 |

### 成长记录媒体上传（对应 OpenSpec 变更 `wire-growth-media-upload`，首个遵循媒体基线的接入方）

> 规则定义见 `openspec/specs/growth-media-pipeline/spec.md`（归档前为变更内 delta）。
> 0003/0004/0006 的拒绝分支由 `validateGrowthMediaPart` 单测 + API 级验证覆盖；
> 0001/0002/0005/0007~0011 为上传与持久化行为，走 API 级 multipart 直传验证。

| TC ID | 名称 | 优先级 | 类型 | 状态 |
|-------|------|--------|------|------|
| TC-NEW-GMP-0001 | 选图后产生服务端可用媒体 | P0 | Integration | 🔲 |
| TC-NEW-GMP-0002 | 上传失败可重试 | P1 | UI | 🔲 |
| TC-NEW-GMP-0003 | 超过单文件上限被拒 | P0 | Integration | 🔲 |
| TC-NEW-GMP-0004 | 不支持的文件格式被拒 | P0 | Integration | 🔲 |
| TC-NEW-GMP-0005 | 发布后照片挂到记录上 | P0 | Integration | 🔲 |
| TC-NEW-GMP-0006 | 发布未上传成功的素材被拒 | P0 | Unit | 🔲 |
| TC-NEW-GMP-0007 | 编辑回显已有照片 | P0 | Integration | 🔲 |
| TC-NEW-GMP-0008 | 追加照片保存后生效 | P0 | Integration | 🔲 |
| TC-NEW-GMP-0009 | 移除照片后从记录中删除 | P0 | Integration | 🔲 |
| TC-NEW-GMP-0010 | 跨记录引用素材被拒 | P0 | Integration | 🔲 |
| TC-NEW-GMP-0011 | 跨班素材被拒 | P0 | Integration | 🔲 |

## 成就授予（ACH）— 老师端授予徽章

> 对应 OpenSpec 变更 `add-achievement-grant`。产品决策见 `docs/achievement-badge-review.md` §5。
> 授予者=老师手动（Q1）· 关联=`StudentAchievement.LogID` 见证记录（Q2）· 口径=累计数（Q3）。

| TC ID | 名称 | 优先级 | 类型 | 状态 |
|-------|------|--------|------|------|
| TC-ACH-001 | 授予成功（本班学生+本园启用徽章） | P0 | Integration | ✅ |
| TC-ACH-002 | 徽章不存在/未启用被拒 | P0 | Integration | ✅ |
| TC-ACH-003 | 跨园所/跨班授予被拒 | P0 | Integration | ✅ |
| TC-ACH-004 | 重复授予被拒（409 非 500） | P0 | Integration | ✅ |
| TC-ACH-005 | 绑定见证记录 | P0 | Integration | ✅ |
| TC-ACH-006 | 不绑定见证记录 | P1 | Integration | ✅ |
| TC-ACH-007 | 见证记录学生不可见被拒 | P0 | Integration | ✅ |
| TC-ACH-008 | 获得数随授予累计 | P1 | Integration | ✅ |
| TC-ACH-009 | 未授予时获得数为零 | P1 | Integration | 🔲 |

### 成长卡片获奖徽章（家长端）

> 对应 OpenSpec 变更 `add-achievement-grant`。设计依据见 `docs/achievement-badge-review.md` §1.1。

| TC ID | 名称 | 优先级 | 类型 | 状态 |
|-------|------|--------|------|------|
| TC-GRW-096 | 见证记录卡片显示徽章 | P0 | Integration | ✅ |
| TC-GRW-097 | 非见证记录不显示徽章 | P0 | Unit | ✅ |
| TC-GRW-098 | 无见证徽章不出现在任何卡片 | P1 | Unit | ✅ |

### 成长时光两模式范围（家长端，对应 OpenSpec 变更 `add-parent-growth-feed-scope`）

> 规则见 `openspec/specs/growth-feed/spec.md`（变更 `add-parent-growth-feed-scope` 已归档）。
> 099~101 为 service 纯函数/组装单测；102~104 为家长端运行时验证。
> 2026-10-20 测试环境实测：102/103/104 全部通过。103 使用新增 R6 fixture（log `991000522`，media `991000522`/`991000523`，仅前者确认到学生 27）；在新 IDE 实例用真实鼠标点击切到只看孩子，截图 `docs/design-previews/2026-10-20-parent-growth-two-modes/runtime-child.png` 仅显示 1 张照片并出现「还有 1 张班级照片…」，`runtime-all.png` 显示 2 张照片。

| TC ID | 名称 | 优先级 | 类型 | 状态 |
|-------|------|--------|------|------|
| TC-GRW-099 | 个体记录列表条目返回 individual scope | P0 | Unit | ✅ |
| TC-GRW-100 | 全班记录列表条目返回 class scope | P0 | Unit | ✅ |
| TC-GRW-101 | scope 不由照片点名推导 | P0 | Unit | ✅ |
| TC-GRW-102 | 个体记录未点名照片在只看孩子仍显示 | P0 | UI | ✅ |
| TC-GRW-103 | 全班记录只显示孩子相关照片并提示 | P0 | UI | ✅ |
| TC-GRW-104 | 全班记录无孩子照片时只看孩子隐藏、看全部可见 | P1 | UI | ✅ |

## 管理端权限强制（PERM）— 角色权限端到端生效

> 对应 OpenSpec 变更 `wire-admin-permissions`。行为规格见 `openspec/changes/wire-admin-permissions/specs/admin-rbac/spec.md`。

| TC ID | 名称 | 优先级 | 类型 | 状态 |
|-------|------|--------|------|------|
| TC-NEW-PERM-0001 | 权限码解析-角色关联优先 | P0 | Unit | ✅ |
| TC-NEW-PERM-0002 | 权限码解析-super_admin 通配 | P0 | Unit | ✅ |
| TC-NEW-PERM-0003 | 权限码解析-空集拒绝 | P0 | Unit | ✅ |
| TC-NEW-PERM-0004 | 服务端强制-有码放行 | P0 | Unit | ✅ |
| TC-NEW-PERM-0005 | 服务端强制-缺管理码 403 | P0 | Unit | ✅ |
| TC-NEW-PERM-0006 | 服务端强制-未登记接口拒绝并留痕 | P0 | Unit | ✅ |
| TC-NEW-PERM-0007 | 登记表缺口使启动失败 | P0 | Unit | ✅ |
| TC-NEW-PERM-0008 | profile 下发权限码 | P0 | Integration | ✅ |
| TC-NEW-PERM-0009 | 菜单按权限过滤 | P0 | E2E | ✅ |
| TC-NEW-PERM-0010 | 通配账号显示全部菜单 | P1 | E2E | ✅ |
| TC-NEW-PERM-0011 | 越权直达重定向首个可访问页 | P0 | E2E | ✅ |
| TC-NEW-PERM-0012 | 零权限账号进入无权限页 | P0 | E2E | ✅ |
| TC-NEW-PERM-0013 | 园所管理员名下保持全量访问 | P0 | Integration | ✅ |
| TC-NEW-PERM-0014 | 教学主任按配置收紧 | P0 | Integration | ✅ |
| TC-NEW-PERM-0015 | 角色页不再显示未生效警告 | P1 | E2E | ✅ |

## 照片点名（PTG）— 照片与孩子的关联

> 对应 OpenSpec 变更 `photo-tagging-face-recognition`。行为规格见 `openspec/changes/photo-tagging-face-recognition/specs/photo-tagging/spec.md`。

| TC ID | 名称 | 优先级 | 类型 | 状态 |
|-------|------|--------|------|------|
| TC-NEW-PTG-0001 | 点名后关系持久化（重建语义/复活软删行） | P0 | Unit / Integration | ✅ |
| TC-NEW-PTG-0002 | 不点名不影响发布 | P0 | Unit / Integration | ✅ |
| TC-NEW-PTG-0003 | 批量点名一次完成（追加语义） | P0 | E2E | ✅ |
| TC-NEW-PTG-0004 | 重复批量点名不叠加 | P0 | Unit | ✅ |
| TC-NEW-PTG-0005 | 单张追加点名 | P0 | Unit / Integration / E2E | ✅ |
| TC-NEW-PTG-0006 | 单张移除点名 | P0 | Unit / Integration | ✅ |
| TC-NEW-PTG-0007 | 未确认的建议不影响家长端出参 | P0 | Unit | ✅ |
| TC-NEW-PTG-0008 | 确认建议后生效（0→1） | P0 | Unit | ✅ |
| TC-NEW-PTG-0009 | 拒绝建议后消失（0→2） | P0 | Unit | ✅ |
| TC-NEW-PTG-0010 | 全班记录默认只看我的孩子 | P0 | E2E | ✅ |
| TC-NEW-PTG-0011 | 切换看全部 | P0 | E2E | ✅ |
| TC-NEW-PTG-0012 | 无孩子被点名的空态引导 | P1 | E2E | ✅ |
| TC-NEW-PTG-0013 | 个体记录不做筛选 | P0 | E2E | ✅ |
| TC-NEW-PTG-0014 | 跨班点名被拒（白名单越界） | P0 | Unit | ✅ |
| TC-NEW-PTG-0015 | 一键确认只作用于高置信（低置信保留） | P0 | Unit / E2E | ✅ |
| TC-NEW-PTG-0016 | 列表中可见待确认建议数 | P1 | Unit / E2E | ✅ |
| TC-NEW-PTG-0017 | 无待确认不呈现角标 | P2 | E2E | ✅ |
| TC-NEW-PTG-0018 | 大合影不产逐人建议，呈现「大合影 · 一键全班」入口 | P0 | Unit / E2E | ✅ |
| TC-NEW-PTG-0019 | 一键全班只含当日到场（请假/缺勤者排除） | P0 | Unit / E2E | ✅ |
| TC-NEW-PTG-0020 | 全班是默认值而非锁定：可再对个别孩子增减 | P0 | Unit / E2E | ✅ |
| TC-NEW-PTG-0021 | 个体记录的大合影保留建议（仅目标孩子；分型仅对全班记录生效） | P1 | Unit | ✅ |

## 人脸底库（FRG）— 学生人脸照片注册

> 对应 OpenSpec 变更 `photo-tagging-face-recognition`。行为规格见 `openspec/changes/photo-tagging-face-recognition/specs/student-face-registry/spec.md`。

| TC ID | 名称 | 优先级 | 类型 | 状态 |
|-------|------|--------|------|------|
| TC-NEW-FRG-0001 | 录入照片并完成注册（1-5 张） | P0 | Unit / Integration | ✅ |
| TC-NEW-FRG-0002 | 客户端不接触人脸服务凭证（密钥只在后端） | P0 | 审查 | ✅ |
| TC-NEW-FRG-0003 | 注册状态可视化（未录入/已注册/失败） | P0 | Unit | ✅ |
| TC-NEW-FRG-0004 | 失败可重试且重试后转已注册 | P0 | Unit | ✅ |
| TC-NEW-FRG-0005 | 底库按园隔离（跨园不匹配） | P0 | Unit / 实测 | ✅ |
| TC-NEW-FRG-0006 | 底库变更不影响既有记录与点名 | P0 | Unit | ✅ |

## 识别队列（RCQ）— 异步识别与配额护栏

> 对应 OpenSpec 变更 `photo-tagging-face-recognition`。行为规格见 `openspec/changes/photo-tagging-face-recognition/specs/growth-recognition-queue/spec.md`。

| TC ID | 名称 | 优先级 | 类型 | 状态 |
|-------|------|--------|------|------|
| TC-NEW-RCQ-0001 | 发布不被识别阻塞（事务后入队） | P0 | Integration | ✅ |
| TC-NEW-RCQ-0002 | 识别完成后建议可见（tagging-status） | P0 | Integration | ✅ |
| TC-NEW-RCQ-0003 | 速率受限（令牌桶节流） | P0 | Unit | ✅ |
| TC-NEW-RCQ-0004 | 预算触顶暂停且任务保留 | P0 | Unit | ✅ |
| TC-NEW-RCQ-0005 | 小园不被大园饿死（按园轮转认领） | P0 | Unit | ✅ |
| TC-NEW-RCQ-0006 | 服务故障不阻断主流程（发布/点名可用） | P0 | Unit / Integration | ✅ |
| TC-NEW-RCQ-0007 | 失败任务可诊断（上限重试后落 failed） | P0 | Unit | ✅ |
| TC-NEW-RCQ-0008 | 灰度开关生效（未开启不建任务） | P0 | Unit | ✅ |
| TC-NEW-RCQ-0009 | 建议按记录范围过滤（跨班/范围外命中不产建议） | P0 | Unit / Integration | ✅ |
| TC-NEW-RCQ-0011 | 建议轮询瞬时失败后自愈（退避重试，建议最终可见） | P2 | E2E | ✅ |

## 亲子任务闭环（TSK-LOOP）

> 对应 OpenSpec 变更 `2026-10-20-redesign-parent-task-loop`。行为规格见 `openspec/changes/2026-10-20-redesign-parent-task-loop/specs/parent-task-loop/spec.md`。
> A 切片状态：0101–0146 全部登记，尚未逐条执行，统一标记为 🔲。

| TC ID | 名称 | 优先级 | 类型 | 状态 |
|-------|------|--------|------|------|
| TC-NEW-TSK-0101 | 班级任务对本班家庭可见 | P0 | FUNC | 🔲 |
| TC-NEW-TSK-0102 | 个体任务仅目标家庭可见 | P0 | FUNC | 🔲 |
| TC-NEW-TSK-0103 | 多孩家庭按当前孩子隔离 | P0 | FUNC | 🔲 |
| TC-NEW-TSK-0104 | 一键完成提交 | P0 | FUNC | 🔲 |
| TC-NEW-TSK-0105 | 文字或照片提交 | P0 | FUNC | 🔲 |
| TC-NEW-TSK-0106 | 未开始或已结束不可提交 | P0 | FUNC | 🔲 |
| TC-NEW-TSK-0107 | 每日/每周/一次性频率 | P0 | FUNC | 🔲 |
| TC-NEW-TSK-0108 | 待回应只在老师端可见 | P0 | FUNC | 🔲 |
| TC-NEW-TSK-0109 | 单条回应闭环 | P0 | FUNC | 🔲 |
| TC-NEW-TSK-0110 | 周期汇总回应 | P0 | FUNC | 🔲 |
| TC-NEW-TSK-0111 | 批量回应部分失败可见 | P1 | FUNC | 🔲 |
| TC-NEW-TSK-0112 | 家长一键反馈 | P1 | FUNC | 🔲 |
| TC-NEW-TSK-0113 | 反馈覆盖率与正向率 | P1 | FUNC | 🔲 |
| TC-NEW-TSK-0114 | 贴纸回应 | P0 | FUNC | 🔲 |
| TC-NEW-TSK-0115 | 语音回应 | P0 | FUNC | 🔲 |
| TC-NEW-TSK-0116 | 班级星星回应 | P0 | FUNC | 🔲 |
| TC-NEW-TSK-0117 | 星星罐按任务/按班级 | P0 | FUNC | 🔲 |
| TC-NEW-TSK-0118 | 只有主回应班级星星计入 | P0 | FUNC | 🔲 |
| TC-NEW-TSK-0119 | 公开默认私密 | P0 | FUNC | 🔲 |
| TC-NEW-TSK-0120 | 申请后老师审核可见 | P0 | FUNC | 🔲 |
| TC-NEW-TSK-0121 | 撤回与个体任务不可公开 | P0 | FUNC | 🔲 |
| TC-NEW-TSK-0122 | 提交不产生积分 | P0 | FUNC | 🔲 |
| TC-NEW-TSK-0123 | 任务契约与规则无积分 | P0 | FUNC | 🔲 |
| TC-NEW-TSK-0124 | 发布通知与 action_path | P0 | FUNC | 🔲 |
| TC-NEW-TSK-0125 | 回应按家长聚合 | P0 | FUNC | 🔲 |
| TC-NEW-TSK-0126 | 家长端小程序码 | P1 | FUNC | 🔲 |
| TC-NEW-TSK-0127 | 多班活动创建 | P0 | FUNC | 🔲 |
| TC-NEW-TSK-0128 | 参与与回应统计 | P0 | FUNC | 🔲 |
| TC-NEW-TSK-0129 | 非法媒体被拒 | P0 | FUNC | 🔲 |
| TC-NEW-TSK-0130 | 删除任务级联 | P0 | FUNC | 🔲 |
| TC-NEW-TSK-0131 | 学习关联必填 | P0 | FUNC | 🔲 |
| TC-NEW-TSK-0132 | 要求列表逐条展示 | P1 | FUNC | 🔲 |
| TC-NEW-TSK-0133 | 首页待办按孩子与周期统计 | P0 | FUNC | 🔲 |
| TC-NEW-TSK-0134 | 取消后再次申请必须重新审核 | P0 | FUNC | 🔲 |
| TC-NEW-TSK-0135 | 汇总后个别补充回应 | P1 | FUNC | 🔲 |
| TC-NEW-TSK-0136 | 给孩子看展示态 | P1 | FUNC | 🔲 |
| TC-NEW-TSK-0137 | 班级星星罐目标独立于任务 | P0 | FUNC | 🔲 |
| TC-NEW-TSK-0138 | 按活动批量编辑与统计 | P0 | FUNC | 🔲 |
| TC-NEW-TSK-0139 | 多班活动事务与跨园所校验 | P0 | FUNC | 🔲 |
| TC-NEW-TSK-0140 | 补充回应未读与读标记 | P1 | FUNC | 🔲 |
| TC-NEW-TSK-0141 | 独立任务不能自设活动分组 | P0 | FUNC | 🔲 |
| TC-NEW-TSK-0142 | 有提交后结构字段只读 | P0 | FUNC | 🔲 |
| TC-NEW-TSK-0143 | 关闭任务公开开关 | P0 | FUNC | 🔲 |
| TC-NEW-TSK-0144 | 跨班公开墙不可见 | P0 | FUNC | 🔲 |
| TC-NEW-TSK-0145 | 更新为个体任务强制关闭公开 | P0 | FUNC | 🔲 |
| TC-NEW-TSK-0146 | 更新个体任务目标跨班被拒 | P0 | FUNC | 🔲 |
