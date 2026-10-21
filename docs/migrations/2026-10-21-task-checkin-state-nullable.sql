-- task_checkin_states.last_check_in_date 改为可空
--
-- 背景：首次提交时状态行还没有“上次提交日期”，代码以 NULL 表示。
-- GORM AutoMigrate 只增不改：如果该列曾以 NOT NULL 建过，不会自动变成 nullable。
-- 空库无需执行；已有该表的开发/测试库执行本文件。
--
-- 执行前确认：
--   SHOW COLUMNS FROM task_checkin_states LIKE 'last_check_in_date';
-- 若 Null 列为 YES，到此为止。

ALTER TABLE task_checkin_states
  MODIFY last_check_in_date DATE NULL;

-- 执行后确认：
--   SHOW COLUMNS FROM task_checkin_states LIKE 'last_check_in_date';
-- 期望 Null = YES。
