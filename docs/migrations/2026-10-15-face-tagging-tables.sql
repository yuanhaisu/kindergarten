-- 照片点名与人脸识别：数据地基
--
-- 背景：photo-tagging-face-recognition 变更（设计见 openspec/changes/photo-tagging-face-recognition/design.md）
--   D2：识别任务表驱动轮询（DB 表而非 Redis 队列）
--   D6：建议/确认/拒绝状态机（growth_media_students 扩列）
--   底库注册状态（student_faces 新表；人脸特征值在云侧，本表只存照片 URL 与云端 PersonRef）
--
-- 执行时机：新代码部署前。正常路径是后端启动时 GORM AutoMigrate 自动建表/补列
--  （本变更全是「新增」，正是 AutoMigrate 有效的场景）；本文件是生产不便重启时的等价手工路径。
--
-- ⚠️ 存量数据兼容：growth_media_students.status 为 NULL 的历史行视为「确认」（读取端按 NULL=1 处理）。
--    无回填脚本——该表此前的写入仅来自测试，生产为空。

-- ── 第 1 步：确认现状 ────────────────────────────────────────────────
-- SHOW COLUMNS FROM growth_media_students LIKE 'status';
-- SHOW TABLES LIKE 'student_faces';
-- SHOW TABLES LIKE 'recognition_jobs';

-- ── 第 2 步：growth_media_students 扩列 ──────────────────────────────
ALTER TABLE growth_media_students
  ADD COLUMN status     tinyint NULL COMMENT '0建议 1确认 2拒绝；NULL=历史行视为确认',
  ADD COLUMN confidence double  NULL COMMENT '识别置信度（人工点名 NULL）',
  ADD INDEX idx_growth_media_students_status (status);

-- ── 第 3 步：新表 student_faces（底库照片与注册状态）────────────────
CREATE TABLE IF NOT EXISTS student_faces (
  id          bigint unsigned NOT NULL AUTO_INCREMENT,
  created_at  datetime(3) NULL,
  updated_at  datetime(3) NULL,
  deleted_at  datetime(3) NULL,
  student_id  bigint unsigned NOT NULL,
  face_url    varchar(500) NOT NULL,
  provider    varchar(20)  NOT NULL DEFAULT 'tencent',
  person_ref  varchar(64)  NULL,
  status      tinyint      NOT NULL DEFAULT 0 COMMENT '0待注册 1已注册 2注册失败',
  last_error  varchar(255) NULL,
  PRIMARY KEY (id),
  UNIQUE KEY idx_student_face (student_id, face_url),
  KEY idx_student_faces_deleted_at (deleted_at),
  KEY idx_student_faces_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- ── 第 4 步：新表 recognition_jobs（识别任务队列）────────────────────
CREATE TABLE IF NOT EXISTS recognition_jobs (
  id              bigint unsigned NOT NULL AUTO_INCREMENT,
  created_at      datetime(3) NULL,
  updated_at      datetime(3) NULL,
  media_id        bigint unsigned NOT NULL,
  kindergarten_id bigint unsigned NOT NULL,
  status          varchar(16) NOT NULL DEFAULT 'pending' COMMENT 'pending/processing/done/failed',
  retries         bigint      NOT NULL DEFAULT 0,
  last_error      varchar(255) NULL,
  worker_id       varchar(64) NULL,
  next_run_at     bigint      NOT NULL DEFAULT 0 COMMENT 'Unix 秒；退避重试用',
  PRIMARY KEY (id),
  UNIQUE KEY idx_recognition_jobs_media_id (media_id),
  KEY idx_job_claim (status, next_run_at),
  KEY idx_recognition_jobs_kindergarten_id (kindergarten_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- ── 第 5 步：验证 ────────────────────────────────────────────────────
-- go run ./cmd/dbcheck growth_media_students   # 应见 status / confidence 两列与 status 索引
-- go run ./cmd/dbcheck student_faces
-- go run ./cmd/dbcheck recognition_jobs
--
-- ── 回滚（若需）──────────────────────────────────────────────────────
-- ALTER TABLE growth_media_students DROP COLUMN status, DROP COLUMN confidence, DROP INDEX idx_growth_media_students_status;
-- DROP TABLE IF EXISTS student_faces;
-- DROP TABLE IF EXISTS recognition_jobs;

-- ── 追加（2026-10-16，D11 大合影分型）────────────────────────────────
-- recognition_jobs 加 face_count：识别检测到的人脸数（0=未识别；1=单人照；≥阈值=大合影不产建议）
ALTER TABLE recognition_jobs
  ADD COLUMN face_count int NOT NULL DEFAULT 0 COMMENT '检测到的人脸数（D11 照片分型）';
-- 验证：go run ./cmd/dbcheck recognition_jobs   # 应见 face_count 列
