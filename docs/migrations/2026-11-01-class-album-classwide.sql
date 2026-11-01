-- 班级相册 Phase 2-A：园所级「全班视图」开关
--
-- 背景：openspec/changes/class-album-classwide/
--   设计见 design.md D1/D2；行为验收见 specs/class-album-classwide/spec.md。
--   只加一列，不改既有列、不动其它业务表。
--
-- 执行时机：新代码部署前。AutoMigrate 只增不改，生产不便重启时用本文件；
--   本地开发库由测试夹具 AutoMigrate 自动补列。
--
-- 幂等：MySQL 8 不支持 ADD COLUMN IF NOT EXISTS，先查 information_schema 再 PREPARE。
--   重复执行是 no-op，不会重置管理员已经开启的值。
--
-- 执行后核对：
--   go run ./cmd/dbcheck kindergarten_app_settings
--   SELECT COUNT(*) AS settings_total,
--          SUM(class_album_classwide_enabled = 1) AS enabled
--     FROM kindergarten_app_settings;
--   期望：新列为 TINYINT(1) NOT NULL DEFAULT 0；历史园所 enabled = 0。

SET @ddl := IF(
  (SELECT COUNT(*) FROM information_schema.COLUMNS
    WHERE TABLE_SCHEMA = DATABASE()
      AND TABLE_NAME = 'kindergarten_app_settings'
      AND COLUMN_NAME = 'class_album_classwide_enabled') = 0,
  'ALTER TABLE kindergarten_app_settings ADD COLUMN class_album_classwide_enabled TINYINT(1) NOT NULL DEFAULT 0 AFTER extra_config',
  'DO 0'
);
PREPARE s FROM @ddl; EXECUTE s; DEALLOCATE PREPARE s;

-- 回滚（仅在确认要回到 Phase 1 时手工执行）：
--   ALTER TABLE kindergarten_app_settings DROP COLUMN class_album_classwide_enabled;
-- 注意：AutoMigrate 不会删除该列；代码回滚到 Phase 1 时列可保留不读，无需立即 DROP。
