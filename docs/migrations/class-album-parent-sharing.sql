-- 班级相册共享 Phase 1：数据模型迁移
--
-- 背景：openspec/changes/class-album-parent-sharing/
--   设计见 design.md D2（数据模型）与 Migration Plan；行为验收见 specs/。
--   本次只加列/加表/加索引 + 回填，不改既有列、不删列、不动其它业务表。
--
-- 执行时机：新代码部署前。AutoMigrate 只增不改，且不会回填 ——
--   生产不便重启时用本文件；本地开发库可用临时程序逐条执行本文件。
--
-- 幂等：MySQL 8 不支持 ADD COLUMN IF NOT EXISTS / ADD INDEX IF NOT EXISTS，
--   每步先查 information_schema 再 PREPARE 执行，可重复运行。
--   两处回填另加保护：
--     · parent_settings：仅当本次真的新增了 allow_class_share 列时才回填，
--       重复执行不会把家长后来修改的值覆盖回旧 allow_growth_share。
--     · album_photos.file_id：只填 NULL 行；file_url 在 files 中不唯一的
--       保持 NULL（多义不做猜测），执行前请先跑文末「歧义 URL」查询。
--
-- 执行前快照（行数）：albums / album_photos / parent_settings / students
-- 执行后核对：
--   go run ./cmd/dbcheck albums
--   go run ./cmd/dbcheck album_photos
--   go run ./cmd/dbcheck album_students
--   go run ./cmd/dbcheck album_photo_students
--   go run ./cmd/dbcheck parent_settings
--   SELECT COUNT(*) AS albums_total, SUM(status = 'published') AS published FROM albums;
--   SELECT COUNT(*) AS photos_total, SUM(file_id IS NOT NULL) AS backfilled,
--          SUM(file_id IS NULL) AS missing FROM album_photos;
--   SELECT COUNT(*) AS settings_total,
--          SUM(allow_class_share = allow_growth_share) AS matches_old_value,
--          SUM(allow_external_share = 0) AS external_off FROM parent_settings;

-- ════════════════════════════════════════════════════════════════════════
-- 1. albums：发布状态字段 + 家长端复合索引
-- ════════════════════════════════════════════════════════════════════════

SET @ddl := IF(
  (SELECT COUNT(*) FROM information_schema.COLUMNS
    WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'albums' AND COLUMN_NAME = 'status') = 0,
  'ALTER TABLE albums ADD COLUMN status VARCHAR(20) NOT NULL DEFAULT ''draft'' AFTER photo_count',
  'DO 0'
);
PREPARE s FROM @ddl; EXECUTE s; DEALLOCATE PREPARE s;

SET @ddl := IF(
  (SELECT COUNT(*) FROM information_schema.COLUMNS
    WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'albums' AND COLUMN_NAME = 'published_at') = 0,
  'ALTER TABLE albums ADD COLUMN published_at DATETIME(3) NULL AFTER status',
  'DO 0'
);
PREPARE s FROM @ddl; EXECUTE s; DEALLOCATE PREPARE s;

SET @ddl := IF(
  (SELECT COUNT(*) FROM information_schema.COLUMNS
    WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'albums' AND COLUMN_NAME = 'published_by') = 0,
  'ALTER TABLE albums ADD COLUMN published_by BIGINT UNSIGNED NOT NULL DEFAULT 0 AFTER published_at',
  'DO 0'
);
PREPARE s FROM @ddl; EXECUTE s; DEALLOCATE PREPARE s;

-- 家长端列表的唯一高频查询：同园同班 + 已发布 + 按相册日期倒序。
SET @ddl := IF(
  (SELECT COUNT(*) FROM information_schema.STATISTICS
    WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'albums'
      AND INDEX_NAME = 'idx_albums_kg_class_status_date') = 0,
  'ALTER TABLE albums ADD INDEX idx_albums_kg_class_status_date (kindergarten_id, class_id, status, album_date)',
  'DO 0'
);
PREPARE s FROM @ddl; EXECUTE s; DEALLOCATE PREPARE s;

-- 历史相册一律落草稿：不自动发布，避免旧照片直接暴露给家长。
-- 新增列的默认值已让存量行为 draft，这里再兜一次空串（幂等：非空行不受影响）。
UPDATE albums SET status = 'draft' WHERE status = '' OR status IS NULL;

-- ════════════════════════════════════════════════════════════════════════
-- 2. album_photos：file_id（media-assets 引用统一字段）+ 回填
-- ════════════════════════════════════════════════════════════════════════

SET @ddl := IF(
  (SELECT COUNT(*) FROM information_schema.COLUMNS
    WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'album_photos' AND COLUMN_NAME = 'file_id') = 0,
  'ALTER TABLE album_photos ADD COLUMN file_id BIGINT UNSIGNED NULL AFTER album_id',
  'DO 0'
);
PREPARE s FROM @ddl; EXECUTE s; DEALLOCATE PREPARE s;

SET @ddl := IF(
  (SELECT COUNT(*) FROM information_schema.STATISTICS
    WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'album_photos'
      AND INDEX_NAME = 'idx_album_photos_file_id') = 0,
  'ALTER TABLE album_photos ADD INDEX idx_album_photos_file_id (file_id)',
  'DO 0'
);
PREPARE s FROM @ddl; EXECUTE s; DEALLOCATE PREPARE s;

-- 按 file_url 回填 file_id。
-- 两表可能由不同迁移创建，排序规则不一致（utf8mb4_general_ci vs utf8mb4_0900_ai_ci），
-- 显式统一比较排序规则，避免 "Illegal mix of collations"。
-- 子查询只保留「该 URL 在 files 中唯一」的行：同一个 URL 对应多条文件记录时
-- 无法判定引用哪一条，留 NULL 由后续写入补齐，不做随机猜测。
-- 不排除 files 的软删行：URL 是同一个资产，软删只影响素材库可见性。
UPDATE album_photos ap
JOIN (
  SELECT file_url, MIN(id) AS id
  FROM files
  GROUP BY file_url
  HAVING COUNT(*) = 1
) f ON f.file_url = ap.file_url COLLATE utf8mb4_unicode_ci
SET ap.file_id = f.id
WHERE ap.file_id IS NULL;

-- 歧义 URL（执行前检查，结果应为 0；非 0 时这些照片的 file_id 会保持 NULL）：
-- SELECT COUNT(*) FROM (
--   SELECT file_url FROM files GROUP BY file_url HAVING COUNT(*) > 1
-- ) dup JOIN album_photos ap ON ap.file_url = dup.file_url COLLATE utf8mb4_unicode_ci;

-- ════════════════════════════════════════════════════════════════════════
-- 3. 新表：album_students / album_photo_students
--    （相册级参与幼儿 / 照片级幼儿关联）
-- ════════════════════════════════════════════════════════════════════════

CREATE TABLE IF NOT EXISTS album_students (
  id              bigint unsigned NOT NULL AUTO_INCREMENT,
  created_at      datetime(3) DEFAULT NULL,
  updated_at      datetime(3) DEFAULT NULL,
  deleted_at      datetime(3) DEFAULT NULL,
  kindergarten_id bigint unsigned NOT NULL,
  album_id        bigint unsigned NOT NULL,
  student_id      bigint unsigned NOT NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uk_album_student (album_id, student_id),
  KEY idx_album_students_kindergarten_id (kindergarten_id),
  KEY idx_album_students_student_id (student_id),
  KEY idx_album_students_deleted_at (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS album_photo_students (
  id              bigint unsigned NOT NULL AUTO_INCREMENT,
  created_at      datetime(3) DEFAULT NULL,
  updated_at      datetime(3) DEFAULT NULL,
  deleted_at      datetime(3) DEFAULT NULL,
  kindergarten_id bigint unsigned NOT NULL,
  photo_id        bigint unsigned NOT NULL,
  student_id      bigint unsigned NOT NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uk_album_photo_student (photo_id, student_id),
  KEY idx_album_photo_students_kindergarten_id (kindergarten_id),
  KEY idx_album_photo_students_student_id (student_id),
  KEY idx_album_photo_students_deleted_at (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- ════════════════════════════════════════════════════════════════════════
-- 4. parent_settings：班内共享 / 园外分享 两个开关 + 旧值迁移
-- ════════════════════════════════════════════════════════════════════════

-- 先记下「本次是否真要新增列」，回填只在这个值为 1 时执行。
-- 这样重复运行本文件不会把家长后来改过的班内共享覆盖回旧 allow_growth_share。
SET @backfill_parent_settings := (
  SELECT COUNT(*) = 0 FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'parent_settings'
    AND COLUMN_NAME = 'allow_class_share'
);

SET @ddl := IF(
  (SELECT COUNT(*) FROM information_schema.COLUMNS
    WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'parent_settings'
      AND COLUMN_NAME = 'allow_class_share') = 0,
  'ALTER TABLE parent_settings ADD COLUMN allow_class_share TINYINT(1) NOT NULL DEFAULT 1 AFTER allow_growth_share',
  'DO 0'
);
PREPARE s FROM @ddl; EXECUTE s; DEALLOCATE PREPARE s;

SET @ddl := IF(
  (SELECT COUNT(*) FROM information_schema.COLUMNS
    WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'parent_settings'
      AND COLUMN_NAME = 'allow_external_share') = 0,
  'ALTER TABLE parent_settings ADD COLUMN allow_external_share TINYINT(1) NOT NULL DEFAULT 0 AFTER allow_class_share',
  'DO 0'
);
PREPARE s FROM @ddl; EXECUTE s; DEALLOCATE PREPARE s;

-- 旧值迁移：班内共享沿用旧 allow_growth_share（不静默改成开启），园外分享一律关闭。
-- allow_growth_share 列保留一个版本，供回滚与旧键兼容。
UPDATE parent_settings
SET allow_class_share = allow_growth_share,
    allow_external_share = 0
WHERE @backfill_parent_settings = 1;

-- ════════════════════════════════════════════════════════════════════════
-- 5. students：状态变更时间（毕业后 90 天只读窗口）
-- ════════════════════════════════════════════════════════════════════════

SET @ddl := IF(
  (SELECT COUNT(*) FROM information_schema.COLUMNS
    WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'students'
      AND COLUMN_NAME = 'status_changed_at') = 0,
  'ALTER TABLE students ADD COLUMN status_changed_at DATETIME(3) NULL AFTER status',
  'DO 0'
);
PREPARE s FROM @ddl; EXECUTE s; DEALLOCATE PREPARE s;

-- 历史毕业/退学学生不回填时间：读取端按「已过期」处理，
-- 避免用 updated_at 猜测导致多保留或少保留访问期。

-- ════════════════════════════════════════════════════════════════════════
-- 回滚（等价内容见 .claude/runs/OS-ALB-1/rollback.sql，不自动执行）
-- ════════════════════════════════════════════════════════════════════════
-- 注意：album_students / album_photo_students / album_photos.file_id /
--       students.status_changed_at 都是新数据；回滚会丢数据，先确认已被引用。
--   1) 代码回滚优先。旧代码不读新列，但 INSERT parent_settings 时会省略新列；
--      如果 AutoMigrate 已按新模型把两列的 DDL 默认值去掉，旧代码插入会因
--      「Field 'allow_class_share' doesn't have a default value」失败。
--      回滚前必须二选一：恢复默认值（allow_class_share DEFAULT 1、
--      allow_external_share DEFAULT 0），或直接删掉两个新列。
--   2) albums：删索引与三列 —— 历史相册会回到「无状态」，旧代码本就是无状态读取。
--   3) parent_settings：allow_growth_share 仍在，旧代码照常读取旧列。
--   4) DROP TABLE album_students, album_photo_students 只在确认无发布内容后执行。
--
-- 具体语句（需人工逐条执行）：
--   ALTER TABLE albums DROP INDEX idx_albums_kg_class_status_date;
--   ALTER TABLE albums DROP COLUMN status, DROP COLUMN published_at, DROP COLUMN published_by;
--   ALTER TABLE album_photos DROP INDEX idx_album_photos_file_id;
--   ALTER TABLE album_photos DROP COLUMN file_id;
--   ALTER TABLE parent_settings DROP COLUMN allow_external_share, DROP COLUMN allow_class_share;
--   ALTER TABLE students DROP COLUMN status_changed_at;
--   DROP TABLE IF EXISTS album_photo_students;
--   DROP TABLE IF EXISTS album_students;
