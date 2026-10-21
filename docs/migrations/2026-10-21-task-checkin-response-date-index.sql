-- task_check_ins 增加 (task_id, response_id, date) 复合索引
--
-- 背景：批量回应在事务内用
--   SELECT ... WHERE task_id = ? AND response_id = 0 AND date BETWEEN ? AND ? FOR UPDATE
-- 加锁。没有这个索引时，InnoDB 沿 (task_id) 索引扫描，
-- locking read 会把该任务的全部提交行（含已回应、含区间外）一起锁住：
-- 既阻塞同任务其它周期的单条回应，也放大锁等待超时。
--
-- 该索引同样服务于「按任务 + 周期查待回应」的普通查询。
-- 新库由 AutoMigrate（domain.TaskCheckIn 的 index:idx_task_response_date）自动创建。
-- 已有库执行前先检查，已存在则跳过（MySQL 的 CREATE INDEX 不支持 IF NOT EXISTS，
-- 重复执行会报 Duplicate key name 并中断迁移批次）：
--   SHOW INDEX FROM task_check_ins WHERE Key_name = 'idx_task_response_date';
--
-- 执行后确认：
--   SHOW INDEX FROM task_check_ins WHERE Key_name = 'idx_task_response_date';
-- 期望三列顺序：task_id(1) → response_id(2) → date(3)。

CREATE INDEX idx_task_response_date ON task_check_ins (task_id, response_id, date);
