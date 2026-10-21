-- notice-multitarget: 把存量单班 notices.class_id 回填为 notice_targets 目标行
-- 幂等：已存在 (notice_id, class_id) 的行不会重复插入，可反复执行。
-- 前置：新代码已部署（AutoMigrate 建出 notice_targets 表）。
-- 后置核实见文件末尾对账 SQL，两数不等则不得进入 DROP COLUMN 步骤。

INSERT INTO notice_targets (notice_id, class_id, kindergarten_id, created_at, updated_at)
SELECT n.id, n.class_id, n.kindergarten_id, NOW(), NOW()
FROM notices n
WHERE n.deleted_at IS NULL
  AND n.class_id > 0
  AND NOT EXISTS (
    SELECT 1 FROM notice_targets t
    WHERE t.notice_id = n.id AND t.class_id = n.class_id
  );

-- 对账（两数必须相等）：
-- SELECT COUNT(*) FROM notices WHERE deleted_at IS NULL AND class_id > 0;
-- SELECT COUNT(DISTINCT notice_id) FROM notice_targets;
