# Spec Delta

## Purpose

判定「同一自然人在两端的两条 User 是同一个人」：unionid 优先、手机号兜底，并保证关联只是数据层判定，不改变登录接口的对外形状。

## ADDED Requirements

### Requirement: unionid 优先判定跨端同人
当两端登录会话都携带 unionid 时，系统 SHALL 以 unionid 作为跨端同人判定的首选依据。

#### Scenario: 两端均有 unionid
- **WHEN** 同一自然人在两端登录后，两条 User 的微信会话都带 unionid 且相同
- **THEN** 系统判定二者为同一自然人并建立关联；后续可据此取到另一端账号
- **TC**: TC-NEW-ATH-0009

#### Scenario: 两端 unionid 不同
- **WHEN** 两条 User 的 unionid 均非空但取值不同
- **THEN** 系统判定二者为不同自然人，不建立关联
- **TC**: TC-NEW-ATH-0010

### Requirement: 手机号兜底判定跨端同人
当 unionid 缺失（如某端为个人主体小程序、微信不下发 unionid）时，系统 SHALL 以手机号作为跨端同人判定的兜底依据。

#### Scenario: 一端无 unionid 但两端手机号一致
- **WHEN** 老师端 User 无 unionid（个人主体小程序）、家长端 User 有或无 unionid，且两条 User 的手机号一致
- **THEN** 系统判定二者为同一自然人并建立关联
- **TC**: TC-NEW-ATH-0011

#### Scenario: 无 unionid 且手机号不一致或缺失
- **WHEN** 至少一端无 unionid，且两条 User 的手机号不一致或任一侧为空
- **THEN** 系统判定二者为不同自然人（或无法判定），不建立关联、不猜测
- **TC**: TC-NEW-ATH-0012

### Requirement: 关联不改变登录接口对外形状
跨端同人判定 SHALL 只影响数据层的关联状态，`POST /auth/login/wechat` 的请求与响应契约 SHALL 保持不变，不新增、不删除、不改名任何接口字段。

#### Scenario: 登录响应契约保持稳定
- **WHEN** 任一端完成微信登录
- **THEN** 响应结构与本次变更前一致，调用方无需感知关联逻辑的存在
- **TC**: TC-NEW-ATH-0013

### Requirement: 关联判定不修改账号归属
建立跨端关联 SHALL NOT 合并两条 User、SHALL NOT 迁移任一端的角色、班级或孩子绑定关系，亦不得改变任一端后续登录的命中结果。

#### Scenario: 关联建立后两端仍各自命中
- **WHEN** 两条 User 已被判定为同一自然人后，任一端再次登录
- **THEN** 仍命中本端原有 User；两侧角色、班级、孩子绑定关系均不变
- **TC**: TC-NEW-ATH-0014
