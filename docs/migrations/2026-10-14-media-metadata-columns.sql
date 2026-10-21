-- 素材管理基线 L1/L2 元数据列
--
-- 背景：素材管理设计基线（docs/media-management-design.md + openspec/specs/media-assets/）
--  要求描述分两层：
--    L1 文件层  files.caption（素材是什么）、files.tags（受控标签，字典驱动）
--    L2 引用层  growth_media.ref_desc（这次为什么用）
--  首批由 wire-growth-media-upload 落地。
--
-- 执行时机：**新代码部署前**。正常路径是后端启动时 GORM AutoMigrate 自动补列
--  （AutoMigrate 对「新增列」有效——它只增不改，这里正是需要「增」的场景）；
--  本文件是生产环境不便重启时的等价手工路径。
--
-- ⚠️ 三列均可空、无默认值、无索引需求（当前无按 caption/tags 检索的查询；
--    素材库检索功能启动时再评估索引）。存量行保持 NULL，读取端按空值处理。

-- ── 第 1 步：确认当前列（应缺 caption/tags/ref_desc）──────────────────
-- SHOW COLUMNS FROM files LIKE 'caption';
-- SHOW COLUMNS FROM files LIKE 'tags';
-- SHOW COLUMNS FROM growth_media LIKE 'ref_desc';

-- ── 第 2 步：新增列（幂等：MySQL 8 无 ADD COLUMN IF NOT EXISTS，重复执行报 1060 可忽略）──
ALTER TABLE files
  ADD COLUMN caption varchar(255) NULL COMMENT '素材固有描述（L1）',
  ADD COLUMN tags    varchar(255) NULL COMMENT '受控标签，逗号分隔，候选来自 sys_dict（L1）';

ALTER TABLE growth_media
  ADD COLUMN ref_desc varchar(255) NULL COMMENT '引用层描述：这次为什么用（L2）';

-- ── 第 3 步：验证 ─────────────────────────────────────────────────────
-- go run ./cmd/dbcheck files         # 应看到 caption / tags 两列
-- go run ./cmd/dbcheck growth_media  # 应看到 ref_desc 一列
--
-- ── 回滚（若需）────────────────────────────────────────────────────────
-- ALTER TABLE files DROP COLUMN caption, DROP COLUMN tags;
-- ALTER TABLE growth_media DROP COLUMN ref_desc;
