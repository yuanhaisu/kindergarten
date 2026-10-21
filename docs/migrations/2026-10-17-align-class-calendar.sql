-- 班级日历语义对齐与课表：数据地基
--
-- 背景：align-class-calendar 变更（设计见 openspec/changes/align-class-calendar/design.md）
--   D1 受众范围 scope（class/kindergarten）；D2 发布状态 status 与推送分离；
--   D5 课表两层模型 + 物化槽位（source/source_slot/overridden）；D4 变更要旨（revised_at/change_summary）
--
-- 执行时机：新代码部署前。正常路径是后端启动时 GORM AutoMigrate 自动补列/建表
--  （本变更加列与建表正是 AutoMigrate 有效的场景）；本文件是生产不便重启时的等价手工路径。
--  字典 seed（calendar_event_type）的正常路径是 seeder.go 启动播种（幂等，含 kgID=0 模板），
--  本文件第 4 步为其等价手工路径——两者均按存在性判重，可安全共存。
--
-- ⚠️ 存量数据兼容：既有事件行 scope/status/source 取默认值（class/published/manual），
--    source_slot 为 NULL —— MySQL 唯一索引不约束 NULL，手动事件与课表槽位天然不冲突。
--
-- ⚠️ 外键事实（实施中发现）：calendar_events 上存在真实外键 fk_calendar_events_class
--    （class_id → classes.id, ON DELETE RESTRICT）。园级事件因此不能用 class_id=0 哨兵
--    （会撞 FK 1452）——改为 class_id 可空、园级存 NULL（MySQL 外键对 NULL 不检查，
--    且保留对真实班级行的 RESTRICT 保护）。见第 3b 步。

-- ── 第 1 步：确认现状 ────────────────────────────────────────────────
-- SHOW COLUMNS FROM calendar_events LIKE 'scope';
-- SHOW TABLES LIKE 'timetable_templates';

-- ── 第 2 步：calendar_events 扩列（幂等：重复执行报 1060 可忽略）──────
ALTER TABLE calendar_events
  ADD COLUMN scope          varchar(20)  NOT NULL DEFAULT 'class'     COMMENT '受众范围：class 班级 / kindergarten 园级',
  ADD COLUMN status         varchar(20)  NOT NULL DEFAULT 'published' COMMENT '发布状态：draft 草稿 / published 已发布',
  ADD COLUMN source         varchar(20)  NOT NULL DEFAULT 'manual'    COMMENT '来源：manual 手动 / timetable 课表物化',
  ADD COLUMN source_slot    varchar(40)  NULL                         COMMENT '课表槽位键（如 3-2 = 周三第2节）；手动事件为 NULL',
  ADD COLUMN overridden     tinyint(1)   NOT NULL DEFAULT 0           COMMENT '课表实例被老师覆盖（调整/取消）后为 1，同步不触碰',
  ADD COLUMN revised_at     datetime(3)  NULL                         COMMENT '最近一次实质变更时间（家长端变更提示窗口用）',
  ADD COLUMN change_summary varchar(255) NULL                         COMMENT '变更要旨（如「时间 10:00 → 10:30」）；无变更为 NULL';

-- 物化槽位唯一键（仅约束课表来源；名称与 domain 的 uniqueIndex:uk_calendar_slot 对齐，
-- 避免 AutoMigrate 以不同名重复建索引）
ALTER TABLE calendar_events
  ADD UNIQUE KEY uk_calendar_slot (class_id, event_date, source_slot);

-- ── 第 3 步：新表（课表两层模型）─────────────────────────────────────
-- timetable_templates：年级模板（周模式）。
-- 注意：刻意不建唯一索引 —— 全量替换采用「软删旧 + 插新」，
-- 软删行会占唯一键位（本仓库已知陷阱：软删除行占唯一索引）。
CREATE TABLE IF NOT EXISTS timetable_templates (
  id              bigint unsigned NOT NULL AUTO_INCREMENT,
  created_at      datetime(3) NULL,
  updated_at      datetime(3) NULL,
  deleted_at      datetime(3) NULL,
  kindergarten_id bigint unsigned NOT NULL,
  grade           varchar(32)  NOT NULL COMMENT '年级（对应 classes.grade）',
  weekday         int          NOT NULL COMMENT '1-7（周一=1）',
  period          int          NOT NULL COMMENT '第几节（正整数）',
  title           varchar(200) NOT NULL COMMENT '课程/活动名',
  location        varchar(200) NULL,
  start_time      varchar(10)  NULL,
  end_time        varchar(10)  NULL,
  PRIMARY KEY (id),
  KEY idx_tt_templates_kg_grade (kindergarten_id, grade),
  KEY idx_timetable_templates_deleted_at (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- timetable_overrides：班级微调（周模式，覆盖年级模板）。
CREATE TABLE IF NOT EXISTS timetable_overrides (
  id              bigint unsigned NOT NULL AUTO_INCREMENT,
  created_at      datetime(3) NULL,
  updated_at      datetime(3) NULL,
  deleted_at      datetime(3) NULL,
  kindergarten_id bigint unsigned NOT NULL,
  class_id        bigint unsigned NOT NULL,
  weekday         int          NOT NULL COMMENT '1-7（周一=1）',
  period          int          NOT NULL COMMENT '第几节',
  action          varchar(20)  NOT NULL COMMENT 'replace 替换 / remove 删除 / add 新增',
  title           varchar(200) NULL,
  location        varchar(200) NULL,
  start_time      varchar(10)  NULL,
  end_time        varchar(10)  NULL,
  PRIMARY KEY (id),
  KEY idx_tt_overrides_kg_class (kindergarten_id, class_id),
  KEY idx_timetable_overrides_deleted_at (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- ── 第 3b 步：class_id 改可空（园级事件存 NULL；幂等：重复执行报 1091/1826 等可忽略）──
-- 改 FK 列类型必须先卸 FK 再装回（ON DELETE RESTRICT 保持原样）。
ALTER TABLE calendar_events DROP FOREIGN KEY fk_calendar_events_class;
ALTER TABLE calendar_events MODIFY class_id bigint unsigned NULL;
ALTER TABLE calendar_events ADD CONSTRAINT fk_calendar_events_class
  FOREIGN KEY (class_id) REFERENCES classes (id) ON DELETE RESTRICT ON UPDATE RESTRICT;

-- 存量归一：若曾按 class_id=0 写入过园级事件，改为 NULL
UPDATE calendar_events SET class_id = NULL WHERE class_id = 0 AND scope = 'kindergarten';

-- ── 第 3c 步：creator_id 改可空（系统物化实例无自然人创建者）──────────
-- 课表同步创建的事件没有自然人创建者；表上有外键 fk_calendar_events_creator
-- （creator_id → users.id, RESTRICT），哨兵 0 会撞 FK → 与 class_id 同法：改可空、系统行存 NULL。
ALTER TABLE calendar_events DROP FOREIGN KEY fk_calendar_events_creator;
ALTER TABLE calendar_events MODIFY creator_id bigint unsigned NULL;
ALTER TABLE calendar_events ADD CONSTRAINT fk_calendar_events_creator
  FOREIGN KEY (creator_id) REFERENCES users (id) ON DELETE RESTRICT ON UPDATE RESTRICT;

-- ── 第 4 步：字典 seed（等价 seeder.go 路径；幂等）────────────────────
-- 4a. 现存园所各补一份 calendar_event_type
INSERT INTO sys_dict_types
  (kindergarten_id, type_key, type_name, description, is_system, sort_order, created_at, updated_at)
SELECT kg.id, 'calendar_event_type', '日历事件类型',
       '班级日历事件类型（内置 5 类不可删除，园所可扩展）', 1, 1, NOW(3), NOW(3)
FROM kindergartens kg
WHERE NOT EXISTS (
  SELECT 1 FROM sys_dict_types t
  WHERE t.kindergarten_id = kg.id AND t.type_key = 'calendar_event_type' AND t.deleted_at IS NULL
);

-- 4b. kgID=0 系统模板（对应 seeder.go 的 SeedDicts(db, 0)；供新园所复制）
INSERT INTO sys_dict_types
  (kindergarten_id, type_key, type_name, description, is_system, sort_order, created_at, updated_at)
SELECT 0, 'calendar_event_type', '日历事件类型',
       '班级日历事件类型（内置 5 类不可删除，园所可扩展）', 1, 1, NOW(3), NOW(3)
FROM DUAL
WHERE NOT EXISTS (
  SELECT 1 FROM sys_dict_types t
  WHERE t.kindergarten_id = 0 AND t.type_key = 'calendar_event_type' AND t.deleted_at IS NULL
);

-- 4c. 内置 5 项（色值取 design-data/teacher-06 图例；图标键由前端按 item_key 映射，不入字典）
INSERT INTO sys_dict_items
  (type_id, item_key, item_label, extra, sort_order, is_enabled, created_at, updated_at)
SELECT t.id, s.item_key, s.item_label, s.extra, s.sort_order, 1, NOW(3), NOW(3)
FROM sys_dict_types t
JOIN (
            SELECT 'notice'   AS item_key, '通知'   AS item_label, '{"color":"#3B82F6"}' AS extra, 1 AS sort_order
  UNION ALL SELECT 'activity', '活动',   '{"color":"#22C55E"}', 2
  UNION ALL SELECT 'holiday',  '假期',   '{"color":"#F97316"}', 3
  UNION ALL SELECT 'meeting',  '家长会', '{"color":"#A855F7"}', 4
  UNION ALL SELECT 'checkup',  '体检',   '{"color":"#EF4444"}', 5
) s
WHERE t.type_key = 'calendar_event_type' AND t.deleted_at IS NULL
  AND NOT EXISTS (
    SELECT 1 FROM sys_dict_items i
    WHERE i.type_id = t.id AND i.item_key = s.item_key AND i.deleted_at IS NULL
  );

-- ── 第 5 步：验证 ────────────────────────────────────────────────────
-- go run ./cmd/dbcheck calendar_events        # 应见新列与 uk_calendar_slot（唯一）
-- go run ./cmd/dbcheck timetable_templates
-- go run ./cmd/dbcheck timetable_overrides
-- go run ./cmd/dbcheck --query "SELECT kindergarten_id, type_key, is_system FROM sys_dict_types WHERE type_key='calendar_event_type' AND deleted_at IS NULL"
-- go run ./cmd/dbcheck --query "SELECT t.kindergarten_id, i.item_key, i.item_label, i.extra FROM sys_dict_items i JOIN sys_dict_types t ON t.id=i.type_id WHERE t.type_key='calendar_event_type' AND t.deleted_at IS NULL AND i.deleted_at IS NULL ORDER BY t.kindergarten_id, i.sort_order"
--
-- ── 回滚（若需）──────────────────────────────────────────────────────
-- 注意：回滚 class_id 可空同样要先卸 FK：DROP FOREIGN KEY → MODIFY NOT NULL → ADD CONSTRAINT。
-- ALTER TABLE calendar_events DROP INDEX uk_calendar_slot,
--   DROP COLUMN scope, DROP COLUMN status, DROP COLUMN source, DROP COLUMN source_slot,
--   DROP COLUMN overridden, DROP COLUMN revised_at, DROP COLUMN change_summary;
-- DROP TABLE IF EXISTS timetable_templates;
-- DROP TABLE IF EXISTS timetable_overrides;
-- DELETE i FROM sys_dict_items i JOIN sys_dict_types t ON t.id = i.type_id WHERE t.type_key = 'calendar_event_type';
-- DELETE FROM sys_dict_types WHERE type_key = 'calendar_event_type';
