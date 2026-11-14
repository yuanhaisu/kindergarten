# distribution-commission Specification

## Purpose
在不建立分销商账号体系的前提下，保住订单的渠道归因与佣金事实：归因一旦确认不可改写，佣金只按真实到账计算。

## Requirements

### Requirement: 订单归因一旦确认不可改写

订单 MAY 关联一个分销商。归因 MUST 在订单确认时确定，确认后 MUST NOT 被修改或清空。分销商 MUST NOT 拥有登录凭证。

#### Scenario: 订单确认后改归因被拒
- **WHEN** 平台方尝试修改一张已确认订单的分销商
- **THEN** 请求被拒绝，订单归因保持不变
- 参考: TC-NEW-DST-0001

#### Scenario: 分销商无登录入口
- **WHEN** 使用任意手机号或账号尝试登录系统
- **THEN** 不存在以分销商身份登录的入口与凭证
- 参考: TC-NEW-DST-0002

### Requirement: 佣金按实收回款计算

佣金 MUST 以订单的实收金额为基数、以收款登记确认日作为计入时点。未到账的应收金额 MUST NOT 产生佣金。

#### Scenario: 部分收款只按已收部分计佣
- **WHEN** 一张应收 2000 元的订单仅登记 1000 元收款，佣金比例为 10%
- **THEN** 产生的佣金为 100 元，而不是 200 元
- 参考: TC-NEW-DST-0003

### Requirement: 退款冲回佣金

一笔收款被退款（负数登记）时，由该笔收款产生的佣金 MUST 被冲回；已冲回的佣金 MUST NOT 参与结算。

#### Scenario: 收款退款后佣金冲回
- **WHEN** 一笔已计佣的收款被全额退款
- **THEN** 对应佣金记录的金额归零且状态变为不可结算
- 参考: TC-NEW-DST-0004

### Requirement: 结算门槛为到账满三十天

佣金台账 SHALL 按月结算。只有「到账满 30 天且期间无退款」的部分才 MUST 进入可结算状态；未满足条件的佣金 MUST 保持待结算。状态取值：`pending`（待结算）/ `settleable`（可结算）/ `settled`（已结算）/ `reversed`（已冲回）。

#### Scenario: 未满三十天不可结算
- **WHEN** 一笔佣金的收款到账仅 20 天
- **THEN** 该佣金不是可结算状态
- 参考: TC-NEW-DST-0005

#### Scenario: 满三十天后可结算
- **WHEN** 该笔收款到账满 30 天且无退款
- **THEN** 该佣金进入可结算状态
- 参考: TC-NEW-DST-0006

### Requirement: 佣金台账只读

平台方 SHALL 只能查看佣金台账及其推导过程。台账金额 MUST 由收款与佣金比例推导，MUST NOT 允许人工直接改写金额。

#### Scenario: 台账金额不可手工改写
- **WHEN** 平台方在佣金台账中尝试手工修改某条佣金金额
- **THEN** 系统不提供该操作，或拒绝该请求
- 参考: TC-NEW-DST-0007
