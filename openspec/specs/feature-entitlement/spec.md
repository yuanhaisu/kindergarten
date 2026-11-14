# feature-entitlement Specification

## Purpose
让「园所买了什么」成为可执行的真实边界：按模块授权、运行时强制、到期后以可逆方式收紧，而不是整站停服。

## Requirements

### Requirement: 授权按模块独立到期

每个授权 SHALL 独立记录模块、来源与到期日，来源取值：`trial`（试用）/ `subscription`（订阅）/ `gift`（赠送）。多个授权叠加时，园所对每个模块的可用性 MUST 逐模块判断，不以园所级单一到期日为准。

#### Scenario: 一模块在期另一模块已过期
- **WHEN** 园所的基础包授权仍在期内，膳食模块授权已过期
- **THEN** 膳食模块不可用，其余模块不受影响
- 参考: TC-NEW-ENT-0001

#### Scenario: 同一模块叠加时取并集
- **WHEN** 同一模块同时存在一条试用授权与一条订阅授权
- **THEN** 只要任一授权在期内，该模块即可用
- 参考: TC-NEW-ENT-0002

### Requirement: 服务端按授权强制 premium 模块

未获得有效授权的 premium 模块接口 MUST 拒绝访问；base 模块 MUST 始终可用。客户端隐藏入口 MUST NOT 替代服务端校验。

#### Scenario: 未购模块接口被拒
- **WHEN** 园所未获得膳食模块授权，其账号调用膳食模块接口
- **THEN** 请求被拒绝并返回明确的未授权提示
- 参考: TC-NEW-ENT-0003

#### Scenario: base 模块始终可用
- **WHEN** 园所没有任何 premium 授权
- **THEN** 考勤、通知、接送等 base 模块接口仍可正常调用
- 参考: TC-NEW-ENT-0004

### Requirement: 客户端按授权呈现入口

管理端、老师端与家长端 SHALL 依据本园的授权清单隐藏未购 premium 模块的入口。入口隐藏 MUST 与授权变更在同一请求周期内保持一致（授权变更后再次进入即生效）。

#### Scenario: 未购模块入口不可见
- **WHEN** 园所未购买成长模块，管理员进入管理端
- **THEN** 侧栏不显示成长记录相关菜单
- 参考: TC-NEW-ENT-0005

#### Scenario: 授权变更后入口即时更新
- **WHEN** 平台方为园所新增成长模块授权后，管理员刷新管理端
- **THEN** 成长记录菜单出现
- 参考: TC-NEW-ENT-0006

### Requirement: 到期后给予十五天宽限期

订阅或试用到期后，系统 SHALL 给予 15 天宽限期；宽限期内全部模块保持原有可用性，界面 SHALL 提示即将到期。宽限期 MUST 由配置提供，不得硬编码。

#### Scenario: 宽限期内仍可写入
- **WHEN** 某 premium 模块授权已到期 10 天
- **THEN** 该模块仍可正常新增数据，界面显示到期提醒
- 参考: TC-NEW-ENT-0007

### Requirement: 宽限期结束后降级为停写保读

宽限期结束后，已过期的 premium 模块 MUST 拒绝写操作，但 MUST 保留读操作；base 模块 MUST 不受影响；系统 MUST NOT 整站停服。

#### Scenario: 降级后新增被拒
- **WHEN** 某 premium 模块授权过期已超过宽限期，园所老师尝试新增一条记录
- **THEN** 写操作被拒绝并提示已到期
- 参考: TC-NEW-ENT-0008

#### Scenario: 降级后历史仍可读
- **WHEN** 同一模块处于降级状态，家长查看历史内容
- **THEN** 历史内容正常返回
- 参考: TC-NEW-ENT-0009

### Requirement: 续费即时恢复写能力

新的授权生效后，相关模块 MUST 立即恢复写能力，MUST NOT 等待缓存自然过期。

#### Scenario: 收款后立刻可写
- **WHEN** 平台方为已降级的园所登记收款并确认订单生效
- **THEN** 园所立即可以再次新增该模块数据
- 参考: TC-NEW-ENT-0010

### Requirement: 开通即获得三十天全功能试用

平台方开通园所后，系统 SHALL 自动发放 30 天全功能试用授权，覆盖全部 premium 模块；平台方 SHALL 可人工延长一次，最长至 60 天。试用期产生的业务数据 MUST 在试用结束后永久保留。

#### Scenario: 新开通园所获得试用
- **WHEN** 平台方开通一所园所
- **THEN** 该园所立即拥有全部 premium 模块的 30 天授权
- 参考: TC-NEW-ENT-0011

#### Scenario: 试用结束后数据保留
- **WHEN** 试用到期并进入降级状态
- **THEN** 试用期内产生的数据仍可读取，不被删除
- 参考: TC-NEW-ENT-0012

### Requirement: 在园幼儿数上限提示

当园所在园幼儿数超过所购档位上限时，系统 SHALL 向园所与平台方提示升级；在读幼儿的既有业务（考勤、成长、接送等）MUST NOT 被阻断，新增幼儿 MUST NOT 被阻断——超限只作提示与销售线索，不作闸门。

#### Scenario: 超限后既有业务不受影响
- **WHEN** 园所幼儿数超过所购档位上限
- **THEN** 现有幼儿的考勤与成长记录仍可正常读写，界面提示升级
- 参考: TC-NEW-ENT-0013
