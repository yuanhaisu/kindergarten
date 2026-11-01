# Spec Delta

## Purpose

定义班内共享与园外分享两个开关的默认值、旧值迁移与撤回语义，并约束老师端家长联系方式只在家长允许时展示。

## ADDED Requirements

### Requirement: 默认值与旧值迁移
未保存过设置的家长 MUST 默认开启班内共享、关闭园外分享；已有设置行 MUST 保留旧 `allow_growth_share` 的值作为班内共享，园外分享一律默认关闭。迁移 MUST NOT 把旧值静默重置为开启。

#### Scenario: 新家长使用新默认值
- **WHEN** 一位从未保存过设置的新家长首次打开设置
- **THEN** 班内共享为开，园外分享为关
- **AND** 覆盖用例 `TC-NEW-PPS-0001`

#### Scenario: 已有设置保留旧值
- **WHEN** 一位家长的旧 `allow_growth_share` 为关闭
- **THEN** 迁移后班内共享仍为关闭，园外分享为关闭
- **AND** 覆盖用例 `TC-NEW-PPS-0002`

### Requirement: 班内共享开关
班内共享 MUST 控制幼儿照片是否出现在本班其他家长可见的班级相册中。家长关闭后，含该幼儿的照片 MUST 立即从其他家长视图隐藏（整张照片），该幼儿自己的家长 MUST 仍能看到；重新开启后 MUST 恢复可见。

#### Scenario: 关闭后立即影响历史与未来照片
- **WHEN** 家长关闭班内共享
- **THEN** 其他家长立即看不到包含该幼儿的既有照片与之后发布的新照片
- **AND** 覆盖用例 `TC-NEW-PPS-0003`

#### Scenario: 重新开启恢复可见
- **WHEN** 家长重新开启班内共享
- **THEN** 包含该幼儿的照片重新出现在其他家长视图
- **AND** 覆盖用例 `TC-NEW-PPS-0004`

### Requirement: 未接入的共享开关不呈现
任何尚未接入实际展示路径的共享开关 MUST NOT 在设置页呈现为可用开关，避免家长误以为已经生效。Phase 1 MUST 只呈现班内共享；园外分享 MUST 保持默认关闭且不出现在设置页，直到园外分享能力上线。

#### Scenario: Phase 1 设置页只展示班内共享
- **WHEN** 家长打开设置页
- **THEN** 能看到班内共享开关，看不到可用的园外分享开关
- **AND** 覆盖用例 `TC-NEW-PPS-0005`

### Requirement: 老师端家长联系方式受控
当家长关闭 `show_phone_to_teacher` 时，老师端家长联系方式接口与页面 MUST NOT 返回或展示该家长手机号；开启时 SHALL 返回。该开关 MUST 只影响手机号，不影响老师查看幼儿姓名与班级归属。

#### Scenario: 关闭时老师看不到手机号
- **WHEN** 家长关闭 `show_phone_to_teacher`，老师查看该幼儿家长联系方式
- **THEN** 接口不返回该家长手机号，页面不展示手机号
- **AND** 覆盖用例 `TC-NEW-PPS-0006`

#### Scenario: 开启时老师可见
- **WHEN** 家长开启 `show_phone_to_teacher`
- **THEN** 老师端能看到该家长手机号
- **AND** 覆盖用例 `TC-NEW-PPS-0007`

#### Scenario: 多家长时逐个过滤
- **WHEN** 一名幼儿有两位家长，其中一位关闭手机号展示
- **THEN** 老师端只看到另一位家长手机号，关闭者的手机号不出现
- **AND** 覆盖用例 `TC-NEW-PPS-0008`
