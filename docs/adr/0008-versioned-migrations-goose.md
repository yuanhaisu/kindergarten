# 0008 — 用 goose 版本化迁移取代启动期 AutoMigrate

- Status: Accepted
- Date: 2026-11-03

## Context

当前服务启动即执行 `AutoMigrate` 并播种，生产库账号被迫长期持有 DDL 权限；没有 schema 版本记录，无法回答线上到底是哪一版。`AutoMigrate` 只增不改，破坏性变更仍靠 `docs/migrations/` 下的手工 SQL。

## Decision

引入 goose，用版本化迁移管理 schema：

- 从已执行完全部 `docs/migrations/` 手工 SQL 的实际开发库导出基线迁移（`cmd/schemadump`），在空库上验证 `goose up`；
- 生产禁用启动期 `AutoMigrate` 与种子；
- 发布流水线在应用重启前运行一次性迁移 job；
- 应用运行账号只有 DML 权限，迁移账号凭据仅存在于 CI；
- 种子改为显式、幂等命令。

## Consequences

- 每次 schema 变更必须提交迁移文件；基线迁移是一次性工作。
- 应用不再自愈 schema，部署顺序变为「迁移成功后启动」。
- 回滚不向后执行迁移，破坏性变更依赖发布前快照。
- `server/migrations/` 成为 schema 的唯一版本真相源。

## Alternatives considered

- **保留 `AutoMigrate`**：实现最省事，但没有版本记录、DDL 权限常驻、种子混在启动路径。
- **仅手工 SQL**：有记录但无执行器，环境间易漂移。
- **Atlas**：能力更强，但对当前规模过重。
