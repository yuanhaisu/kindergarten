-- C-2 / decision §20.10-6：通知「义务集」计数
--
-- 背景：反馈率（feedback_coverage / child_happy_rate / ack_rate）的分母必须来自
-- 「回应发出那一刻确定的义务集（按家长去重的目标家长数）」，不能回看 messages 行数：
--   ① 投递失败不写 messages 行 → 分母漏掉失败方；
--   ② 判重命中不写新行 → 即使全部成功，行数也小于义务集。
-- 两者都会让分母偏小、比率虚高。
--
-- NULL 表示「未能确定义务」（收件人查询整体失败，或本列上线前的历史数据）。
-- 统计把 NULL 排除并单独计数，**不得**当 0——「未知」与「确实没有家长」不可混同。
--
-- 类型用 BIGINT 而不是 INT：Go 的 *int 经 GORM 的 MySQL dialector 映射成 bigint，
-- AutoMigrate 建出来的开发库就是 bigint；按 INT 迁移会让两种库的列类型不一致，
-- 后续 dbcheck / AutoMigrate 对比时会看到无意义的差异（值域本身两者都够）。
--
-- GORM AutoMigrate 会给开发库补列（只增不改），生产/已有库执行本文件。
-- MySQL 不支持 ADD COLUMN IF NOT EXISTS，执行前先确认列是否存在：
--   SHOW COLUMNS FROM parent_tasks    LIKE 'notify_target_count';
--   SHOW COLUMNS FROM task_responses  LIKE 'notify_target_count';
-- 两列都已存在则跳过本文件。若某个库执行过本文件的早期 INT 草案，改用：
--   ALTER TABLE parent_tasks   MODIFY notify_target_count BIGINT NULL;
--   ALTER TABLE task_responses MODIFY notify_target_count BIGINT NULL;

ALTER TABLE parent_tasks
  ADD COLUMN notify_target_count BIGINT NULL COMMENT '发布通知义务家长数（按家长去重）；NULL=未知';

ALTER TABLE task_responses
  ADD COLUMN notify_target_count BIGINT NULL COMMENT '回应通知义务家长数（按家长去重）；NULL=未知';

-- 执行后确认：
--   SHOW COLUMNS FROM parent_tasks   LIKE 'notify_target_count';  -- Null = YES
--   SHOW COLUMNS FROM task_responses LIKE 'notify_target_count';  -- Null = YES
-- 并用 go run ./cmd/dbcheck parent_tasks / task_responses 核对索引未受影响。
