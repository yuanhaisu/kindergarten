# Spec Delta

## Purpose

定义单实例部署下后端必须满足的运行契约：健康检查、优雅停机、迁移顺序、日志、备份与单实例边界。

## ADDED Requirements

### Requirement: 存活与就绪检查分离

后端 SHALL 提供 `/health` 表示进程存活，且 MUST NOT 依赖 MySQL/Redis；SHALL 提供 `/ready` 表示可接收流量，MySQL 或 Redis 不可用时 MUST 返回未就绪。

#### Scenario: 依赖不可用时就绪失败

- **WHEN** MySQL 或 Redis 不可用
- **THEN** `/ready` 返回未就绪，`/health` 仍表示进程存活
- 参考: TC-NEW-OPS-0001

#### Scenario: 依赖恢复后就绪成功

- **WHEN** MySQL 与 Redis 恢复可用
- **THEN** `/ready` 返回就绪
- 参考: TC-NEW-OPS-0002

### Requirement: 收到终止信号后优雅停机

后端在收到 SIGTERM 或 SIGINT 时 SHALL 停止接收新请求，并在有界超时内等待在途请求完成；MUST NOT 立即中断在途请求。

#### Scenario: 在途请求完成后再退出

- **WHEN** 后端收到 SIGTERM 且在途请求尚未完成
- **THEN** 后端停止接收新请求，等待在途请求完成后退出
- 参考: TC-NEW-OPS-0003

#### Scenario: 超时后强制退出

- **WHEN** 在途请求超过约定超时仍未完成
- **THEN** 后端结束等待并退出，避免部署无限挂起
- 参考: TC-NEW-OPS-0004

### Requirement: 迁移先于新版本启动

发布流程 SHALL 在启动新版本前执行版本化数据库迁移。应用运行期 MUST NOT 执行建表或改表；迁移失败时 MUST 中止发布，新版本 MUST NOT 开始服务。

#### Scenario: 迁移失败中止启动

- **WHEN** 迁移 job 返回失败
- **THEN** 流水线中止发布，新版本不启动
- 参考: TC-NEW-OPS-0005

#### Scenario: 启动期不再自动建表

- **WHEN** 应用启动
- **THEN** 应用只读取现有 schema，不执行 AutoMigrate 或种子写入
- 参考: TC-NEW-OPS-0006

### Requirement: 生产日志可采集且可关联

生产环境 SHALL 把结构化日志输出到 stdout，并包含 `request_id`；日志 MUST 被集中采集且保留至少 30 天；日志 MUST NOT 包含明文手机号、openid 或任何密钥。

#### Scenario: 请求日志带 request_id

- **WHEN** 任一请求完成
- **THEN** 对应日志包含可关联的 request_id
- 参考: TC-NEW-OPS-0007

#### Scenario: 敏感信息脱敏

- **WHEN** 日志记录手机号或 openid
- **THEN** 输出为脱敏形式，不出现明文
- 参考: TC-NEW-OPS-0008

### Requirement: 单实例运行边界明确

每个环境 SHALL 只运行一个后端实例。进程内定时任务、识别队列、课表同步与导出临时文件 MUST 按单实例假设运行；在未引入共享存储或粘性路由前，MUST NOT 启用水平扩展。

#### Scenario: 部署声明单实例

- **WHEN** 部署生产或预发
- **THEN** 后端副本数为 1，进程内任务只有一个实例执行
- 参考: TC-NEW-OPS-0009

#### Scenario: 扩展前必须先改造

- **WHEN** 有人尝试把后端副本数调到大于 1
- **THEN** 发布检查阻止该变更，并提示先完成共享存储/粘性路由改造
- 参考: TC-NEW-OPS-0010

### Requirement: 生产数据具备可恢复备份

生产 MySQL SHALL 启用每日自动备份与 binlog；SHALL 每周生成一份存放在数据库实例之外的逻辑备份；恢复目标 MUST 为 RPO ≤ 5 分钟、RTO ≤ 2 小时；SHALL 每季度执行一次恢复演练。

#### Scenario: 每日备份存在

- **WHEN** 检查最近 24 小时的生产备份
- **THEN** 存在成功的自动备份与可用的 binlog
- 参考: TC-NEW-OPS-0011

#### Scenario: 恢复演练通过

- **WHEN** 执行季度恢复演练
- **THEN** 能从备份恢复到临时实例并通过 dbcheck
- 参考: TC-NEW-OPS-0012
