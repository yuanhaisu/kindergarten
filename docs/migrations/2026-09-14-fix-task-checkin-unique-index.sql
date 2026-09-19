-- 修复 task_check_ins 的唯一索引
--
-- 问题：TaskCheckIn 的 struct tag 只在 Date 一个字段上标了 uniqueIndex，
--  GORM 因此生成 UNIQUE(date) —— 全平台每天只允许一条打卡记录。
--  第二个家长打卡就是 Error 1062 Duplicate entry。
--  正确约束应为 (task_id, student_id, date)：每个孩子每天每项任务一次。
--
-- ⚠️ AutoMigrate 无法自动修正：它按【索引名】判断存在性，
--  idx_task_student_date 已存在（只是列不对）就会被跳过。
--  必须先手工删除，下次服务启动时 AutoMigrate 会按新 tag 重建。
--
-- 影响范围：仅限【在建库时就有旧索引】的数据库。
--  2026-09-18 核对过本机开发库（localhost:3316）：task_check_ins 的
--  idx_task_student_date 已经是 (task_id, student_id, date) 三元组，
--  即新库由 AutoMigrate 按修好后的 struct 直接建对了，**无需执行本迁移**。
--  先跑第 1 步确认，不要盲目 DROP。
--
-- 列序 (task_id, student_id, date) 覆盖现有全部查询
-- （均以 task_id 打头，无仅按 student_id 过滤的查询）。

-- ── 第 1 步：查看当前索引，确认待删除的名字 ────────────────────────────
-- 若输出中 idx_task_student_date 已是 task_id/student_id/date 三行，到此为止。
SHOW INDEX FROM task_check_ins;

-- ── 第 2 步：删除定义错误的唯一索引 ────────────────────────────────────
ALTER TABLE task_check_ins DROP INDEX idx_task_student_date;

-- ── 第 3 步：二选一 ───────────────────────────────────────────────────
-- (a) 重启后端服务，AutoMigrate 会按新 struct tag 自动重建；或
-- (b) 手动建立，不依赖服务重启：
--
-- ALTER TABLE task_check_ins
--   ADD UNIQUE KEY idx_task_student_date (task_id, student_id, date);

-- ── 第 4 步：验证 ─────────────────────────────────────────────────────
-- SHOW INDEX FROM task_check_ins WHERE Key_name = 'idx_task_student_date';
-- 期望 3 行，Seq_in_index 依次为 1/2/3，对应 task_id / student_id / date。
-- 若只有 1 行且 Column_name = 'date'，说明删除未生效或索引被重建成了旧定义。

-- ── 可选清理：冗余单列索引 ─────────────────────────────────────────────
-- 复合索引的最左前缀已覆盖 task_id；现有查询无仅按 student_id 过滤的场景，
-- 两个单列索引均可删除。收益不大（省写入开销与空间），可延后处理。
--
-- ⚠️ 索引名是 GORM 默认命名规则（idx_<表>_<列>），执行前务必用第 1 步的输出核对：
-- ALTER TABLE task_check_ins DROP INDEX idx_task_check_ins_task_id;
-- ALTER TABLE task_check_ins DROP INDEX idx_task_check_ins_student_id;

-- ── 遗留数据 ──────────────────────────────────────────────────────────
-- 旧索引存在期间，同一日期只有一条打卡能被写入（其余请求均因唯一冲突失败），
-- 因此历史打卡数据大概率是残缺的。索引修好后不会自动补全，
-- 如需演示数据可参考 cmd/seedattendance 的做法另写一次性脚本。
