# Proposal

## Why

Phase 1 的家长班级相册默认只显示「我的孩子」相关照片。园所希望在不放宽隐私规则的前提下，提供一个可选的「全班」视图，让家长看到班级活动全貌；是否开放必须由园所控制，而不是默认对所有园所生效。

## What Changes

- 园所级开关：`kindergarten_app_settings` 新增 `class_album_classwide_enabled`，默认 **false**；管理端设置页可开启/关闭，写操作审计。
- 家长端班级相册接口增加 `scope=mine|all`：默认 `mine`；`all` 仅在园所开关开启时可用，否则返回 403；`all` 仍遵守 Phase 1 的 opt-out 整张隐藏、自己的孩子始终可见、已发布/同园同班/留存窗口。
- 家长端班级相册页增加「我的孩子 / 全班」切换；开关关闭时只显示「我的孩子」，不显示不可用的全班入口。
- 管理端设置契约与页面增加该开关；老师端不变。
- 契约变更声明：本次将修改 `backend/api/api-parent/13-class-albums.md`、`backend/api/api-admin/18-settings.md`。

**明确不做**：发布通知、视频、园外分享/下载、打包导出、人脸模糊（Phase 2 后续 epic）。

## Capabilities

### New Capabilities

- `class-album-classwide`: 园所级全班视图开关、家长 `scope` 查询语义、全班视图的隐私过滤，以及管理端配置与家长端切换。

### Modified Capabilities

（无：Phase 1 的 `parent-class-album` 默认行为不变，本变更只新增可选 scope。）

## Impact

- Go：`domain.KindergartenAppSetting` 新列；`SettingsService` 读写；`ParentPortalService` 相册 scope 查询；handler/契约；迁移 SQL。
- 管理端：`admin/pages/settings` 增加开关；`admin/api/settings` 类型。
- 家长端：`parent/src/pages/album/index.vue` 分段切换；API wrapper 传 `scope`。
- 数据：AutoMigrate 只增不改，生产需手工 SQL + `dbcheck`；默认 false，历史园所不自动开放。
