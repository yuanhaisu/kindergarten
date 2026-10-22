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
> A 切片状态：2026-10-21 已按 A 切片执行 API 级核心链路；✅ = 实测通过，⚠️ = 仅覆盖主路径，⏸ = B/C 切片未实现或不在 A 范围。

| TC ID | 名称 | 优先级 | 类型 | 状态 |
|-------|------|--------|------|------|
| TC-NEW-TSK-0101 | 班级任务对本班家庭可见 | P0 | FUNC | ✅ |
| TC-NEW-TSK-0102 | 个体任务仅目标家庭可见 | P0 | FUNC | ✅ |
| TC-NEW-TSK-0103 | 多孩家庭按当前孩子隔离 | P0 | FUNC | ✅ |
| TC-NEW-TSK-0104 | 一键完成提交 | P0 | FUNC | ✅ |
| TC-NEW-TSK-0105 | 文字或照片提交 | P0 | FUNC | ✅ |
| TC-NEW-TSK-0106 | 未开始或已结束不可提交 | P0 | FUNC | ✅ |
| TC-NEW-TSK-0107 | 每日/每周/一次性频率 | P0 | FUNC | ✅ |
| TC-NEW-TSK-0108 | 待回应只在老师端可见 | P0 | FUNC | ✅ |
| TC-NEW-TSK-0109 | 单条回应闭环 | P0 | FUNC | ✅ |
| TC-NEW-TSK-0110 | 周期汇总回应 | P0 | FUNC | ✅ |
| TC-NEW-TSK-0111 | 批量回应部分失败可见 | P1 | FUNC | ⚠️ 并发路径已验证，写入失败路径未构造 |
| TC-NEW-TSK-0112 | 家长一键反馈 | P1 | FUNC | ⏸ |
| TC-NEW-TSK-0113 | 反馈覆盖率与正向率 | P1 | FUNC | ⏸ |
| TC-NEW-TSK-0114 | 贴纸回应 | P0 | FUNC | ✅ |
| TC-NEW-TSK-0115 | 语音回应 | P0 | FUNC | ✅ |
| TC-NEW-TSK-0116 | 班级星星回应 | P0 | FUNC | ✅ |
| TC-NEW-TSK-0117 | 星星罐按任务/按班级 | P0 | FUNC | ⏸ |
| TC-NEW-TSK-0118 | 只有主回应班级星星计入 | P0 | FUNC | ⏸ |
| TC-NEW-TSK-0119 | 公开默认私密 | P0 | FUNC | ✅ |
| TC-NEW-TSK-0120 | 申请后老师审核可见 | P0 | FUNC | ⏸ |
| TC-NEW-TSK-0121 | 撤回与个体任务不可公开 | P0 | FUNC | ✅ / ⏸ |
| TC-NEW-TSK-0122 | 提交不产生积分 | P0 | FUNC | ✅ |
| TC-NEW-TSK-0123 | 任务契约与规则无积分 | P0 | FUNC | ✅ |
| TC-NEW-TSK-0124 | 发布通知与 action_path | P0 | FUNC | ⏸ |
| TC-NEW-TSK-0125 | 回应按家长聚合 | P0 | FUNC | ⏸ |
| TC-NEW-TSK-0126 | 家长端小程序码 | P1 | FUNC | ⏸ |
| TC-NEW-TSK-0127 | 多班活动创建 | P0 | FUNC | ⏸ |
| TC-NEW-TSK-0128 | 参与与回应统计 | P0 | FUNC | ⏸ |
| TC-NEW-TSK-0129 | 非法媒体被拒 | P0 | FUNC | ✅ |
| TC-NEW-TSK-0130 | 删除任务级联 | P0 | FUNC | ✅ |
| TC-NEW-TSK-0131 | 学习关联必填 | P0 | FUNC | ✅ |
| TC-NEW-TSK-0132 | 要求列表逐条展示 | P1 | FUNC | ✅ |
| TC-NEW-TSK-0133 | 首页待办按孩子与周期统计 | P0 | FUNC | ✅ |
| TC-NEW-TSK-0134 | 取消后再次申请必须重新审核 | P0 | FUNC | ⏸ |
| TC-NEW-TSK-0135 | 汇总后个别补充回应 | P1 | FUNC | ⏸ |
| TC-NEW-TSK-0136 | 给孩子看展示态 | P1 | FUNC | ⏸ |
| TC-NEW-TSK-0137 | 班级星星罐目标独立于任务 | P0 | FUNC | ⏸ |
| TC-NEW-TSK-0138 | 按活动批量编辑与统计 | P0 | FUNC | ⏸ |
| TC-NEW-TSK-0139 | 多班活动事务与跨园所校验 | P0 | FUNC | ⏸ |
| TC-NEW-TSK-0140 | 补充回应未读与读标记 | P1 | FUNC | ⏸ |
| TC-NEW-TSK-0141 | 独立任务不能自设活动分组 | P0 | FUNC | ⏸ |
| TC-NEW-TSK-0142 | 有提交后结构字段只读 | P0 | FUNC | ✅ |
| TC-NEW-TSK-0143 | 关闭任务公开开关 | P0 | FUNC | ✅ |
| TC-NEW-TSK-0144 | 跨班公开墙不可见 | P0 | FUNC | ⏸ |
| TC-NEW-TSK-0145 | 更新为个体任务强制关闭公开 | P0 | FUNC | ✅ |
| TC-NEW-TSK-0146 | 更新个体任务目标跨班被拒 | P0 | FUNC | ✅ |

### A 切片执行记录（2026-10-21）

环境：本机 Go 后端 `:8080` + 开发库（kindergarten 4）；老师 61（class 6）、老师 65（class 4）、家长 11（孩子 4）、家长 20（孩子 1）、家长 61（孩子 27/32）、家长 12（孩子 21，异班）。证据为 curl 实测，非仅 MCP 导航。

**实测通过（A）**：0101–0110、0114–0116、0119、0122–0123、0129–0133、0142–0143、0145–0146。

**并发互斥补测（对应 0107 频率约束 + `task_check_ins` 唯一键，不是 0111）**：同一任务 / 同一孩子 / 同一天 10 个并发提交，只成功 1 次（201×1 / 422×9，无 5xx），打卡历史仅 1 条，列表 `submitted_current_period=true`、`checkin_count=1`。脚本为本机一次性脚本（未入库，文件名沿用当时误记的任务号）。

**0111「批量回应部分失败可见」现状（2026-10-21 修复后）**：并发竞争路径已实测——两次并发「一键送星（本周期）」，赢家 `responded_count=2`，输家返回 `responded_count=0 / id=0` 且**不留孤儿回应**，DB 中只有一条 `task_responses`（`period`，`period_start=period_end=当天`）。写入失败（DB 报错）路径未构造；该路径现在返回 500（提交保持待回应），`failed_ids` 只在部分成功时可见。

**执行中发现并修复的缺陷**：
1. `parent_portal_service.go` 更新频率状态用了不存在的列 `last_checkin_date`（实际列 `last_check_in_date`），任何打卡都 500 → 已修，并同步修正 `docs/migrations/2026-10-21-task-checkin-state-nullable.sql` 与 handoff。
2. 家长端任务接口在 `student_id` 不属于该家长时静默回退到第一个孩子 → 新增 `parentTaskCtx`：任务链路 `student_id` 必填且必须归属当前家长（列表/详情/提交/历史统一 422/403），并同步前端首页与任务页先落盘当前孩子。
3. `CheckinHistory` 只按 `task_id + student_id` 查询，未套任务可见范围：请求别的孩子的个体任务历史会返回 200 空列表 → 已改为先过 `taskScopeQuery`，跨家庭直接访问返回 404。
4. 本地缓存的孩子被解绑后，任务页一次 403 后停住（无重试、无错误态），详情页仍渲染提交入口 → 任务列表/详情在 403 时刷新 overview 并重试一次（现统一走 `withStudentScope`）；详情增加错误态，无任务数据时不渲染提交入口。
5. 任务列表用 `onLoad` 拉数据，从详情页提交后返回不重跑，卡片一直停在「待完成 / 0+N次」直到手动下拉刷新 → 改为 `onShow`（`parent/src/pages/tasks/index.vue`），返回即刷新。
6. 老师端任务列表同样用 `onLoad`，详情页回应后返回「待回应 N」不刷新；别处创建的任务也不会出现（本页 `addTask` 创建后会主动重拉，不受影响）→ 改为 `onLoad(loadStudents)` + `onShow(loadTasks)`（`teacher/src/pages/tasks/index.vue`），学生列表仍只取一次。

**未执行/未实现（⏸）**：
- 0112/0113 家长反馈、0120/0121 公开申请审核与撤回、0124–0125 通知聚合、0126 小程序码、0134 再次申请、0135 补充回应、0136 给孩子看、0140 读标记（B 切片）
- 0117/0118 星星罐进度、0127/0128 活动、0137 班级罐目标、0138/0139 多班活动、0141 活动分组（C 切片）
- 0144 公开墙（B 切片；公开墙查询未实现）

**UI 交互验证（2026-10-21，家长端，微信开发者工具）**：`parent/dist/dev/mp-weixin`（dev watcher 构建）+ 微信开发者工具模拟器（appid `wxdf989efff8796c53`，家长 61 / 孩子 27，中二班），通过 CDP 在渲染层派发真实鼠标事件驱动，页面文案与状态均取自渲染 DOM（非接口回读）：

1. 首页「今晚一件事」卡片 → 点击进入 `pages/tasks/detail`，详情渲染标题、学习关联（和幼儿园学习相关 / 本周班级主题是「我会整理」）、今日提交与打卡历史。
2. 点「今天完成了（一键提交）」→ 页面切为「打卡历史 2026-10-21 已提交」，提交入口消失；服务端 `task_check_ins` 新增 1 条（id 16）。
3. 返回首页 →「今晚一件事」换成另一条任务（onShow 刷新生效）。
4. 任务列表点「去提交」→ 详情 → 一键提交 → 返回列表，「进行中」不再出现该任务（缺陷 5 的 `onShow` 修复在真机渲染层复验通过）。
5. 详情页在文本框输入「微信端文字提交验证」→ 点「打卡」→ 打卡历史渲染该文字；服务端 `task_check_ins` id 18 的 `content` 一致。
6. 老师 API 回应后重新进入详情 → 渲染「老师回应 / 贴纸 / 今天收得很认真，继续加油！」。

工具备注：`wechat_automator` 的 `page_data` / `element_info` / `screenshot` 与 `wechat_navigate` 在本机 daemon 上持续 90–120s 超时（`system_info`、`evaluate` 正常），改用 `_verify/wx_cdp.mjs`（CDP `Runtime.evaluate` + `Input.dispatchMouseEvent`）读取渲染文案并派发真实点击。派发触摸事件不触发 tap，鼠标事件才有效。

**UI 交互验证（2026-10-21，老师端，微信开发者工具）**：`teacher/dist/dev/mp-weixin`（appid `wxbfb5f81a87f73a61`，登录老师 11 王小明妈妈 / 小一班），同样用 CDP 真实鼠标事件驱动：

1. 工作台点「亲子任务」→ 列表渲染创建表单（范围/频率/日期/回应仪式/星星罐/公开开关）与任务卡片，待回应数与服务端一致（1 / 2 / 0）。
2. 进入任务详情 → 编辑区、待回应/已回应分栏、贴纸（star/heart/thumb）与录音入口、幼儿姓名与家长提交内容均正常渲染。
3. 点贴纸 `star` → 待回应清空；服务端 `task_responses` 生成 `single` / `sticker` / `star`（id 9）。
4. 当时的「一键全部送星」→ 两个家庭的提交同时转为已回应；服务端两条提交共享同一 `response_id=10`（`response_type=period`）。
   **注意**：该次点击发生在**一次性任务**（task 34）上，且当时尚未写入 `period_start/period_end`（两列为 NULL）。此后的修复已要求「只有每日/每周的班级任务」才显示汇总入口，once/个体任务会被拒绝（422），因此这条证据只证明「按钮有响应」，不证明「本周期」语义。周期语义的现行证据是 API 级：daily 任务 36 批量后 `task_responses.id=11`，`period_start=period_end=2026-10-21`，两条提交共享该回应。
5. 返回列表：修复前仍显示旧计数且新任务不出现（缺陷 6），改 `onShow` 后计数与列表即时正确。

**仍未覆盖**：
- 照片提交的选图流程（`wx.chooseMedia` 会拉起系统文件对话框，未自动化）；上传口径由 API 层用例覆盖。
- 真机（非模拟器）渲染差异。
- 老师端创建表单的逐项填写与提交（字段已渲染并确认，数据由 API 层创建）、语音回应（需录音设备）、公开申请/审核链路（B 切片未实现）。
- 数据库旧列已清理：`docs/migrations/2026-10-21-drop-task-points-columns.sql` 已执行，`parent_tasks.points`、`task_check_ins.points`、`task_check_ins.image_url` 已不存在；执行前后行数不变（17 / 10，2026-10-21 迁移当天快照，非当前行数）。

**批量回应并发与周期边界（2026-10-21 已修复并实测）**：

- `BatchRespondTask` / `RespondTask` 改为事务内 `SELECT ... FOR UPDATE` 锁定待回应行，并检查 `RowsAffected`；一条都没抢到时回滚，不产生孤儿 `task_responses`。
- 周期汇总按频率取区间（每日=当天、每周=滚动 7 天，锚点取 `min(今天, 任务结束日)`，已结束任务仍能关闭最后一个周期），只关闭区间内提交，并写入 `PeriodStart` / `PeriodEnd`。
- **一次性任务与个体任务拒绝汇总回应**（422「一次性任务与个体任务需逐条回应」），前端对应隐藏批量按钮并给出提示。
- 全部写入失败返回 500（提交保持待回应），不再伪装成「没有待回应」；只有读取阶段无候选时才返回 **200** + 空操作 `id=0`。
- `task_check_ins` 增加 `(task_id, response_id, date)` 复合索引（`AutoMigrate` 已建，`docs/migrations/2026-10-21-task-checkin-response-date-index.sql` 供已有库），避免 `FOR UPDATE` 锁住整个任务的提交行。
- 任务详情返回 `period_start` / `period_end`；老师端把区间外的待回应标注为「旧周期 · 请逐条回应」，避免「计数不减 / 按钮点了 no-op」的死循环感。
- 任务**列表**同样返回 `pending_period` / `period_start` / `period_end`：`pending_responses` 是全量（含旧周期），`pending_period` 是**本周期内**的待回应条数（与批量接口的筛选条件逐字相同：`task_id = ? AND response_id = 0 AND date >= ? AND date <= ?`）。两者不等时列表显示「待回应 N（本周期 M）」。对不支持汇总回应的任务（一次性、个体、以及 `frequency` 脏数据）`pending_period` 恒等于 `pending_responses`——这些任务调用批量接口会 422，所以列表**不能**把它读成「可以一键关闭」。三个 COUNT（total / 全量待回应 / 本周期待回应）失败时整个列表返回错误，由前端错误卡提示重试；不回退成 0，也不回退成全量（回退值看起来与正常态一模一样）。
- **API 实测（2026-10-21 当日快照，复跑会因日期不同得到不同区间）**：
  - 等值路径：daily / once 各建一条当日任务 + 今日提交 → `pending=1 period=1`，daily 区间 `period_start=period_end=当天`。
  - **错位路径**：daily 任务（task 53，起点 10 天前、终点当天）+ 今日真实提交 + 一条 10 天前的探针提交（一次性 Go 程序直插）→ 列表 `pending=2 / period=1`；批量 `closed=1`（只关本周期）；之后 `pending=1 / period=0`。
  - 被测任务（49/50/51/52/53）验证后都已 `DELETE /teacher/tasks/:id`：`ParentTask` / `TaskCheckIn` 走软删除，`task_response_supplements` / `task_response_reactions` / `task_check_in_states` / `task_read_marks` 是 `Unscoped()` 硬删除，按 id 复查查不到，证据以脚本输出为准。
  - 两个脚本都**未入库**：`_verify/pending_period_mismatch.mjs` 依赖已删除的一次性探针程序，重跑前需按同样逻辑重建（插一条 `date` 为 10 天前的 `TaskCheckIn`）。
- 老师端「一键送星」加 `batchBusy` 门闩：请求进行中（**含随后的列表刷新**，刷新最长等 3 秒，避免 `uni.request` 无 timeout 时门闩锁死 60s）的重复点击被拦住，不会紧跟一条 `responded_count=0` 的提示被误读成失败；0 条文案改为「本周期已无待回应提交」。
- 任务详情 `load()` 加请求序号：切「待回应/已回应」会并发两次加载，后返回的旧请求不再覆盖当前页签的列表。
- **老师端 UI 交互复验（2026-10-21，微信开发者工具 `teacher/dist/dev/mp-weixin`，CDP 派发真实鼠标事件，文案取自渲染 DOM）**：
  1. 列表三种分支：`daily · 待回应 2（本周期 1）`（有旧周期遗留）、`once · 待回应 1`（不支持汇总，无括号）、`daily · 待回应 1`（等值，无括号）。
  2. 详情：今日提交与旧周期行并列，后者标「旧周期 · 请逐条回应」；点「一键送星」后今日行消失、旧行保留；接口复核 `pending=1 / period=0`。
  3. 返回列表（`onShow` 刷新）：自动变为 `待回应 1（本周期 0）`，无需下拉。
  4. `once` 任务详情：无批量按钮，显示「一次性任务与个体任务需逐条回应」。
  5. 停掉后端后返回列表：「网络异常」+「显示的是上次拉取的结果，点击重试」，旧数据保留；点错误卡后列表重新加载、错误卡消失。
  **未覆盖**：toast 文案（小程序原生 UI，不在 DOM 内，无法读取）；连点门闩只能通过「第二次点击后接口状态不变」间接确认。
- **老师端 UI 复验（2026-10-22，改后代码：`loadSeq` + 刷新 3 秒封顶）**：
  1. 登录后工作台日期 `2026年10月22日 星期四`，跨日后显示正常。
  2. 列表：`once · 待回应 1`、`daily · 待回应 1`。
  3. 详情点「一键送星」→ 待回应变「暂无提交」、批量按钮消失；切「已回应」显示该条与「已回应」标记；切回「待回应」仍为空（无过期请求覆盖）。
  4. 重新进入列表：`daily · 待回应 0`，与 `GET /teacher/tasks` 一致。
  **未复验**：旧周期行与加载失败错误卡（代码未变，沿用 10-21 的证据）；连点门闩的中间态仍不可观测。
  **工具备注**：本轮遇到 `navigateTo:fail timeout`——详情 pageframe 建好但空白、导航被回滚，重试一次即成功。失败的 pageframe 会残留在 CDP target 列表**首位**，导致 `wx_cdp.mjs` 默认选错目标读到空 DOM；脚本已支持 `WX_TARGET=pages/xxx` 指定页面。另：`wx.navigateBack` 连续执行会多弹一层，读页面栈前先确认。
- 业务日期与进程时区：`CreateTask` / `UpdateTask` 原用 `time.Local` 解析 `start_date` / `end_date`，与 `taskPeriodRange` 不同源；现统一用 `shanghaiLoc()`。但仅改解析还不够——MySQL DSN 是 `loc=Local`，驱动在写 DATE 列前执行 `t.In(time.Local)`，服务器 `TZ=UTC` 时「2026-10-21 00:00 +08」会被格式化成前一天。因此 `timezone.go` 的 `init()` 把 `time.Local` 对齐到 `Asia/Shanghai`，让解析、计算、落库共用同一墙钟；`teacherPortalDay` 也改用 `shanghaiLoc()`。`_ "time/tzdata"` 随两者移入该文件。
  **证据等级**：`TestProcessTimezoneAlignedWithBusiness` 断言 `time.Local` 已指向 `Asia/Shanghai` 且业务日期经 `t.In(Local)` 后日历日不变；**但 `TZ=UTC` 的写库场景无法在本机复现**——实测 Windows 上 Go 忽略 `TZ` 环境变量（`time.Local` 始终 +08），所以这条只证明了初始化生效，没证明跨时区落库。
 **前提与代价**（审查补充）：
 - 修复成立的前提是 **DSN 的 `loc` 与进程时区一致**。`MYSQL_DSN` 环境变量会整体替换配置，漏写 `loc` 时驱动默认 UTC，`init` 对驱动无效、DATE 仍会差一天。已在 `ProvideGorm` 启动时检测并 `slog.Warn`（不自动改写 DSN——那会把既有 DATETIME 的解释一起改掉）。
 - 若某次部署曾以 `TZ=UTC` + `loc=Local` 运行，`created_at` 等 **DATETIME** 列存的是 UTC 墙钟；对齐到 +08 后这些历史行会被按 +08 解析，显示偏移 8 小时。**DATE 列只取日期部分，不受影响**。当前部署本来就是 +08，无影响。
 - `cmd/*` 下的独立二进制多数不导入 `service`，不会继承这次对齐（`fixpwd` / `initcheck` / `setpwd` 导入，会继承）。非 +08 机器上 seed/refreshtasks 与 API 的日期口径会分叉。
- 实测：并发批量 → 赢家 `responded_count=2`、输家 `0`，DB 仅一条 `period` 回应；并发单条 → 一次 **201**、一次 **409**「该提交已被其他回应关闭，请刷新后重试」，无重复回应。
- 周期边界实测用的是**直插库的探针行**（task 36，date=2026-10-11，非真实家长提交）：批量后该行保持 `response_id=0`；随后单条并发用例把它作为目标关闭为 `response_id=12`，所以现在查库看到的是 12，不是回归。

**代码审查新增的已知缺口（2026-10-21，未修复）**：

- `parentTaskCtx` 把 `ResolveContext` 的任何错误都映射成 403，数据库故障会伪装成「未绑定孩子信息」。
- 打卡成功后详情页整页重载；重载失败会把已经提交成功的页面打成错误态。
