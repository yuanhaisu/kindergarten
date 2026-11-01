# Design

## Context

Phase 1 的家长班级相册固定按当前孩子过滤（`parent-class-album`）；`kindergarten_app_settings` 目前只有 `current_semester` 与 `ExtraConfig` JSON。Phase 2-A 要增加一个园所级、默认关闭的全班视图开关，并让家长端在同一页面切换「我的孩子 / 全班」。

## Goals / Non-Goals

**Goals**：园所可控的全班视图；家长显式切换；不改变 Phase 1 隐私规则；管理端可配置；真实后端可验证。

**Non-Goals**：发布通知、视频、园外分享/下载、打包导出、人脸模糊。

## Decisions

### D1. 用独立列存开关，不塞进 ExtraConfig

`kindergarten_app_settings` 新增 `ClassAlbumClasswideEnabled bool`（列 `class_album_classwide_enabled`，默认 false）。理由：家长相册服务直接读该列即可，不需要复制 `settingsPayload` 的 JSON 解析；admin 设置仍走现有 SettingsService 读写该列。替代方案（ExtraConfig 新分组）需要在家长服务与 settings 服务各维护一份解析，易漂移。

### D2. 家长 API 增加 `scope` 与可用性字段

- `GET /parent/class-albums` 与 `/:id` 增加 `scope=mine|all`，缺省 `mine`。
- 列表响应新增 `classwide_enabled: bool`，家长端据此决定是否显示切换；详情不需要。
- 开关关闭且 `scope=all` → 403；其它 scope 值 → 422；跨班/跨园 → 404。
- `scope=all` 仍只查已发布、同班、留存窗口内；照片过滤改为「不做 my-child 过滤，但保留 opt-out 整张隐藏与自己的孩子例外」。

### D3. 管理端契约与页面

`GET/PUT /admin/settings` 增加 `album.classwide_enabled`；复用现有设置更新权限与审计（`settings` 模块），不新增权限码。admin 设置页在「成长/通知」附近增加开关与说明。

### D4. 家长端

`parent/src/pages/album/index.vue` 读取列表响应的 `classwide_enabled`：
- false：不渲染切换，只请求 `mine`。
- true：渲染「我的孩子 / 全班」分段；切换时重置分页并请求 `scope=all|mine`；详情链接带上当前 scope。
- 空态/错误态沿用 Phase 1；全班视图无照片时显示「本班暂无可见照片」。

## 跨端触点清单

| 层 | 是否修改 | 内容 |
| --- | --- | --- |
| Go domain | 会改 | `KindergartenAppSetting` 新列 |
| repository | 会改 | 设置 repo 读取/保存新列；家长相册查询按 scope |
| service | 会改 | SettingsService 读写；ParentPortalService scope 过滤 |
| handler | 会改 | 设置响应/请求；家长相册解析 scope |
| Casbin policy | 不改 | 复用现有 admin settings / parent 路由权限 |
| router.go | 不改 | 端点已存在，仅新增 query/字段 |
| backend/api/ | 会改 | `api-parent/13-class-albums.md`、`api-admin/18-settings.md` |
| 老师端 | 不改 | 无老师侧变化 |
| 家长端 | 会改 | `parent/src/pages/album/index.vue`、`parent/src/api/parent-class-album.js` |
| 管理端 | 会改 | `admin/pages/settings/index.vue`、`admin/api/settings.ts` |
| design token | 不改 | 复用 Phase 1 分段/卡片样式 |
| 测试 | 会改 | Go 单测 + TC CAW-0001..0011；DevTools/admin 真实交互 |
| 指标 | 不改 | Phase 1 的 `album_parent_view` 增加 scope 字段（可选，不新增事件） |

## Migration Plan

1. SQL：`ALTER TABLE kindergarten_app_settings ADD COLUMN class_album_classwide_enabled TINYINT(1) NOT NULL DEFAULT 0;`（AutoMigrate 只增不改；生产手工执行并用 `go run ./cmd/dbcheck kindergarten_app_settings` 核实）。
2. 默认 false：历史园所不自动开放；管理员显式开启。
3. 上线顺序：后端 → 管理端 → 家长端；家长端旧版本不传 scope，行为仍是 mine。
4. 回滚：代码回滚即可；新列保留（不读），家长端回到 mine；管理端开关隐藏。

## Risks / Trade-offs

- **默认关闭导致功能“看不见”**：符合隐私优先；用管理端说明与园所运营沟通。
- **`scope=all` 的隐私边界**：仍依赖 Phase 1 的 opt-out 过滤；未逐张标记的合影可能包含其他孩子，Phase 1 已记录为残余，Phase 2 不做人脸模糊。
- **家长端列表多一个字段**：不额外请求；旧客户端忽略。
- **管理端权限**：复用现有 settings 权限，不新增码；若园所角色没有 settings 权限，开关不可见。
