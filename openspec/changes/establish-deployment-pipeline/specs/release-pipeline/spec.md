# Spec Delta

## Purpose

定义软件版本从构建到预发、生产、回滚的发布契约，以及小程序审核周期下的 API 兼容约束。

## ADDED Requirements

### Requirement: 发布由不可变 tag 触发

版本发布 SHALL 由 `vYYYY.MM.DD-N` 形式的 tag 触发。同一个 tag MUST 对应唯一且不可变的后端镜像与管理端静态包；重复构建同一 tag MUST NOT 覆盖已发布的产物。

#### Scenario: tag 触发构建

- **WHEN** 维护者推送一个新 tag `v2026.11.03-1`
- **THEN** 流水线产出该 tag 的后端镜像与管理端静态包，并以该 tag 作为唯一版本标识
- 参考: TC-NEW-DEP-0001

#### Scenario: 重复 tag 被拒绝

- **WHEN** 已存在的 tag 被重新推送
- **THEN** 流水线拒绝覆盖已有产物，或产出不同内容时失败
- 参考: TC-NEW-DEP-0002

### Requirement: 预发自动部署，生产必须人工批准

发布 tag 构建成功后 SHALL 自动部署到预发环境；生产环境部署 MUST 经过显式人工批准。未获得批准时，流水线 MUST NOT 启动生产部署。

#### Scenario: 自动部署预发

- **WHEN** tag 构建成功
- **THEN** 流水线自动部署预发并通过 `/ready` 检查
- 参考: TC-NEW-DEP-0003

#### Scenario: 未批准不进入生产

- **WHEN** 预发部署成功但生产审批未通过
- **THEN** 生产环境保持原版本，部署任务不启动
- 参考: TC-NEW-DEP-0004

### Requirement: 就绪失败自动回滚

每次部署 SHALL 在启动后验证 `/ready`。若新版本在约定窗口内未就绪，流水线 MUST 恢复到上一个已发布的后端镜像与管理端静态包；回滚 MUST NOT 留下部分更新的环境。

#### Scenario: 新版本未就绪自动回滚

- **WHEN** 新版本启动后 `/ready` 在窗口内持续失败
- **THEN** 流水线把后端镜像与管理端静态包恢复到上一个发布版本
- 参考: TC-NEW-DEP-0005

#### Scenario: 回滚后旧版本可服务

- **WHEN** 自动回滚完成
- **THEN** `/ready` 恢复成功，旧版本继续提供原行为
- 参考: TC-NEW-DEP-0006

### Requirement: 后端兼容最旧线上小程序版本

后端 API SHALL 向后兼容仍在线上运行的最旧小程序版本。删除或重命名字段、改变字段语义 MUST 拆为两次发布：先发布兼容版本，再在兼容窗口结束后清理；`/api/v1` MUST NOT 因日常发布升版。

#### Scenario: 旧版小程序调用新后端

- **WHEN** 仍在线上运行的旧版小程序调用新版后端
- **THEN** 旧版依赖的字段与语义保持可用，请求成功
- 参考: TC-NEW-DEP-0007

#### Scenario: 破坏性变更被拆为两次发布

- **WHEN** 某字段需要重命名
- **THEN** 第一次发布同时提供新旧字段，第二次发布在兼容窗口结束后才移除旧字段
- 参考: TC-NEW-DEP-0008

### Requirement: 生产发布前必须具备数据库快照

生产发布 SHALL 在部署新版本前确认数据库快照或等效备份已成功生成；无法确认时 MUST 中止生产发布。

#### Scenario: 快照失败中止发布

- **WHEN** 生产发布前的快照任务失败
- **THEN** 流水线中止发布，生产保持原版本
- 参考: TC-NEW-DEP-0009

#### Scenario: 快照成功后继续

- **WHEN** 快照成功
- **THEN** 流水线继续执行迁移与部署
- 参考: TC-NEW-DEP-0010
