# Spec Delta

## Purpose

让园所可以按需开启班级相册的「全班」视图：家长在默认「我的孩子」之外可切换浏览全班照片，同时不改变 Phase 1 的 opt-out、自己的孩子始终可见与班级/园所隔离规则。

## ADDED Requirements

### Requirement: 园所开关默认关闭
园所级全班视图开关 MUST 默认关闭；只有管理员显式开启后，家长端才可使用 `scope=all`。开关变更 MUST 写操作审计。

#### Scenario: 新园所默认关闭
- **WHEN** 园所未修改过全班视图设置
- **THEN** 家长端 `scope=all` 返回 403，默认视图仍是 `mine`
- **AND** 覆盖用例 `TC-NEW-CAW-0001`

#### Scenario: 管理员开启与关闭
- **WHEN** 管理员在设置页开启全班视图，随后再次关闭
- **THEN** 两次变更都成功并写入审计；关闭后 `scope=all` 立即返回 403
- **AND** 覆盖用例 `TC-NEW-CAW-0002`

### Requirement: 家长 scope 语义
家长端班级相册列表/详情 MUST 接受 `scope=mine|all`，缺省为 `mine`；`all` 仅在园所开关开启且请求属于当前孩子班级时可用，否则 MUST 返回 403 或 404，不泄露其它班级数据。

#### Scenario: 默认 mine
- **WHEN** 家长请求班级相册未传 scope
- **THEN** 只返回包含当前孩子关联的照片
- **AND** 覆盖用例 `TC-NEW-CAW-0003`

#### Scenario: 开启后可请求 all
- **WHEN** 园所开关开启，家长传 `scope=all`
- **THEN** 返回本班所有已发布相册与可见照片，仍按 opt-out 过滤
- **AND** 覆盖用例 `TC-NEW-CAW-0004`

#### Scenario: 关闭时 all 被拒绝
- **WHEN** 园所开关关闭，家长传 `scope=all`
- **THEN** 返回 403，不返回全班照片
- **AND** 覆盖用例 `TC-NEW-CAW-0005`

### Requirement: 全班视图的隐私过滤
`scope=all` MUST 继续遵守：只含已发布、同园同班、留存窗口内；关联到任何 `allow_class_share=false` 幼儿的照片对其他家长整张隐藏；当前家长自己的孩子始终可见。

#### Scenario: opt-out 合影在全班视图隐藏
- **WHEN** 全班视图中一张照片关联了已关闭班内共享的幼儿 C，查看者不是 C 的家长
- **THEN** 该照片不出现；同相册其他照片仍可见
- **AND** 覆盖用例 `TC-NEW-CAW-0006`

#### Scenario: 自己的孩子始终可见
- **WHEN** 查看者自己的孩子已关闭班内共享
- **THEN** 含该孩子的照片仍对查看者可见
- **AND** 覆盖用例 `TC-NEW-CAW-0007`

### Requirement: 管理端开关配置
管理端设置接口/页面 SHALL 提供全班视图开关的读取与更新；无对应权限的管理员 MUST NOT 修改；开关值 MUST 与家长端行为同源。

#### Scenario: 有权限管理员读取与更新
- **WHEN** 有设置权限的管理员打开设置页并切换开关
- **THEN** 读取值正确，更新成功，家长端行为随之变化
- **AND** 覆盖用例 `TC-NEW-CAW-0008`

#### Scenario: 无权限管理员不可修改
- **WHEN** 无设置权限的管理员尝试更新该开关
- **THEN** 请求被拒绝，开关值不变
- **AND** 覆盖用例 `TC-NEW-CAW-0009`

### Requirement: 家长端切换入口
家长端班级相册页 MUST 在开关开启时显示「我的孩子 / 全班」切换；开关关闭时 MUST NOT 显示不可用的全班入口；默认选中「我的孩子」，切换后重新拉取对应 scope。

#### Scenario: 开关关闭不显示切换
- **WHEN** 园所开关关闭，家长打开班级相册
- **THEN** 只显示「我的孩子」视图，没有全班切换
- **AND** 覆盖用例 `TC-NEW-CAW-0010`

#### Scenario: 开关开启可切换
- **WHEN** 园所开关开启，家长切到「全班」
- **THEN** 列表与详情按 `scope=all` 重新请求并渲染
- **AND** 覆盖用例 `TC-NEW-CAW-0011`
