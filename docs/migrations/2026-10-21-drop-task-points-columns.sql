-- 清理亲子任务改造后遗留的旧列
--
-- 背景：A 切片移除了任务积分，domain / backend/api 契约 / service 均已不再读写：
--   parent_tasks.points
--   task_check_ins.points
--   task_check_ins.image_url（改用 media_file_id + media_url）
--
-- GORM AutoMigrate 只增不改，旧列不会自动消失，必须手工 DROP。
--
-- 执行前确认（2026-10-21 开发库实测）：
--   task_check_ins: 10 行，points 全 0，image_url 全空
--   parent_tasks:   17 行，points 全为默认值 1
-- 若其他环境的 points 有非零值或 image_url 有内容，先备份再执行。
--
-- 执行后确认：
--   SHOW COLUMNS FROM parent_tasks LIKE 'points';          -- 应为空
--   SHOW COLUMNS FROM task_check_ins
--     WHERE Field IN ('points','image_url');               -- 应为空

ALTER TABLE parent_tasks
  DROP COLUMN points;

ALTER TABLE task_check_ins
  DROP COLUMN points,
  DROP COLUMN image_url;
