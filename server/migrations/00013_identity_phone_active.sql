-- 一人一账号的身份底座（#164）：
--   1) users.phone_active 是 STORED 生成列 + 唯一索引，让「活跃且非空」的手机号唯一，
--      同时让空手机号（待补全）与软删账号不占用唯一键；
--   2) user_identities 把「账号 × 客户端」的微信身份从 users.open_id/union_id 拆出来。
-- DDL 隐式提交，不包事务。
-- +goose NO TRANSACTION

-- +goose Up

-- 预检必须在 DDL 之前：活跃手机号有重复时，直接建唯一索引会被 MySQL 拒绝，
-- 且 1062 的报错文本会把原始手机号带进日志。这里只统计组数与行数并用脱敏
-- 消息中止迁移；不自动删除或合并任何数据。
DROP PROCEDURE IF EXISTS `kg_preflight_phone_active`;

-- +goose StatementBegin
CREATE PROCEDURE `kg_preflight_phone_active`()
BEGIN
  DECLARE dup_groups INT DEFAULT 0;
  DECLARE dup_rows INT DEFAULT 0;
  DECLARE msg VARCHAR(255);

  SELECT COUNT(*), COALESCE(SUM(cnt), 0)
    INTO dup_groups, dup_rows
  FROM (
    SELECT COUNT(*) AS cnt
    FROM `users`
    WHERE `deleted_at` IS NULL AND `phone` IS NOT NULL AND `phone` <> ''
    GROUP BY `phone`
    HAVING COUNT(*) > 1
  ) AS dup;

  IF dup_groups > 0 THEN
    SET msg = CONCAT(
      'phone_active 唯一索引未建立：活跃手机号重复 ', dup_groups,
      ' 组 / ', dup_rows,
      ' 行。请运行 cmd/phoneaudit 人工清理后重试；迁移未删除任何数据。');
    SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = msg;
  END IF;
END
-- +goose StatementEnd

CALL `kg_preflight_phone_active`();

DROP PROCEDURE `kg_preflight_phone_active`;

-- IF(deleted_at IS NULL AND phone <> '', phone, NULL)：
--   * 软删行 → deleted_at 非空 → NULL
--   * 空/未填手机号 → NULL
--   * NULL 不参与唯一比较，因此上述两类行可以有多条。
-- 生成列必须与唯一索引在**同一条 ALTER** 里，MySQL 8 保证单条 DDL 原子性，
-- 不会留下「列已加、索引未加」的半迁移状态。
ALTER TABLE `users`
  ADD COLUMN `phone_active` varchar(20)
    GENERATED ALWAYS AS (IF(`deleted_at` IS NULL AND `phone` <> '', `phone`, NULL)) STORED,
  ADD UNIQUE KEY `uk_users_phone_active` (`phone_active`) USING BTREE;

CREATE TABLE `user_identities` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `user_id` bigint unsigned NOT NULL,
  `app_id` varchar(128) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci NOT NULL,
  `open_id` varchar(128) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci NOT NULL,
  `union_id` varchar(128) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci NOT NULL DEFAULT '',
  PRIMARY KEY (`id`) USING BTREE,
  UNIQUE KEY `uk_user_identities_app_open` (`app_id`,`open_id`) USING BTREE,
  UNIQUE KEY `uk_user_identities_user_app` (`user_id`,`app_id`) USING BTREE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- +goose Down

DROP TABLE IF EXISTS `user_identities`;

ALTER TABLE `users`
  DROP INDEX `uk_users_phone_active`,
  DROP COLUMN `phone_active`;

DROP PROCEDURE IF EXISTS `kg_preflight_phone_active`;
