-- 晨检照片引用对齐 media-assets 统一引用契约
--
-- 背景：openspec/changes/morning-health-closed-loop/design.md D3
--   health_check_media 原先直接存 file_url/thumb_url，缺少引用层三件套
--   （file_id / ref_desc / sort_num）。本次补列并按 URL 回填存量 file_id。
--   file_url / thumb_url 保留为读取缓存，文件真相源仍是 files 表。
--
-- 执行时机：新代码部署前。AutoMigrate 能补列但不会回填，本文件同时负责回填。
-- 幂等：每步先查 information_schema，重复执行不会报 1060 / 1061。

-- ── 第 1 步：确认当前列（应缺 file_id/ref_desc/sort_num）──────────────
-- SHOW COLUMNS FROM health_check_media;

-- ── 第 2 步：新增列（幂等）────────────────────────────────────────────
SET @ddl := IF(
  (SELECT COUNT(*) FROM information_schema.COLUMNS
    WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'health_check_media' AND COLUMN_NAME = 'file_id') = 0,
  'ALTER TABLE health_check_media ADD COLUMN file_id BIGINT UNSIGNED NULL AFTER check_id',
  'DO 0'
);
PREPARE s FROM @ddl; EXECUTE s; DEALLOCATE PREPARE s;

SET @ddl := IF(
  (SELECT COUNT(*) FROM information_schema.COLUMNS
    WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'health_check_media' AND COLUMN_NAME = 'ref_desc') = 0,
  'ALTER TABLE health_check_media ADD COLUMN ref_desc VARCHAR(255) NOT NULL DEFAULT '''' AFTER category',
  'DO 0'
);
PREPARE s FROM @ddl; EXECUTE s; DEALLOCATE PREPARE s;

SET @ddl := IF(
  (SELECT COUNT(*) FROM information_schema.COLUMNS
    WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'health_check_media' AND COLUMN_NAME = 'sort_num') = 0,
  'ALTER TABLE health_check_media ADD COLUMN sort_num INT NOT NULL DEFAULT 0 AFTER ref_desc',
  'DO 0'
);
PREPARE s FROM @ddl; EXECUTE s; DEALLOCATE PREPARE s;

SET @ddl := IF(
  (SELECT COUNT(*) FROM information_schema.STATISTICS
    WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'health_check_media' AND INDEX_NAME = 'idx_health_media_file_id') = 0,
  'ALTER TABLE health_check_media ADD INDEX idx_health_media_file_id (file_id)',
  'DO 0'
);
PREPARE s FROM @ddl; EXECUTE s; DEALLOCATE PREPARE s;

-- ── 第 3 步：回填存量 file_id（幂等，未命中的历史行保持 NULL）──────────
-- 两表可能由不同迁移创建，排序规则不一致（0900_ai_ci vs general_ci）；
-- 显式统一到 utf8mb4_unicode_ci，避免 "Illegal mix of collations"。
UPDATE health_check_media m
JOIN files f ON f.file_url = m.file_url COLLATE utf8mb4_unicode_ci AND f.deleted_at IS NULL
SET m.file_id = f.id
WHERE m.file_id IS NULL AND m.deleted_at IS NULL;

-- ── 第 4 步：验证 ─────────────────────────────────────────────────────
-- go run ./cmd/dbcheck health_check_media   # 应看到 file_id/ref_desc/sort_num 与 idx_health_media_file_id
-- SELECT COUNT(*) AS total,
--        SUM(file_id IS NULL) AS missing_file_id
-- FROM health_check_media WHERE deleted_at IS NULL;

-- ── 回滚（若需）────────────────────────────────────────────────────────
-- ALTER TABLE health_check_media DROP INDEX idx_health_media_file_id;
-- ALTER TABLE health_check_media DROP COLUMN file_id, DROP COLUMN ref_desc, DROP COLUMN sort_num;
