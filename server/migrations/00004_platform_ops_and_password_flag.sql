-- 商业化第一期补充：园长账号强制改密标记 + 平台方操作留痕。
-- +goose NO TRANSACTION

-- +goose Up
ALTER TABLE `users`
  ADD COLUMN `must_change_password` tinyint(1) NOT NULL DEFAULT '0' AFTER `id_card_number`;

CREATE TABLE `platform_operation_logs` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  `operator` varchar(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci NOT NULL,
  `module` varchar(50) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci NOT NULL,
  `action` varchar(50) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci NOT NULL,
  `target_type` varchar(50) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci NOT NULL DEFAULT '',
  `target_id` bigint unsigned NOT NULL DEFAULT '0',
  `detail` text CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci,
  `ip` varchar(50) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci NOT NULL DEFAULT '',
  PRIMARY KEY (`id`) USING BTREE,
  KEY `idx_platform_log_operator` (`operator`) USING BTREE,
  KEY `idx_platform_log_module` (`module`) USING BTREE,
  KEY `idx_platform_log_target` (`target_type`,`target_id`) USING BTREE,
  KEY `idx_platform_operation_logs_deleted_at` (`deleted_at`) USING BTREE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- +goose Down
DROP TABLE IF EXISTS `platform_operation_logs`;
ALTER TABLE `users` DROP COLUMN `must_change_password`;
