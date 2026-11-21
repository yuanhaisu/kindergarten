-- 小程序客户端注册表（#163）：每个客户端一个微信 AppID，v1 只有两条 shared 客户端。
-- 不在迁移里写死 AppID：启动时从配置幂等同步，避免 test/prod 串号。
-- DDL 隐式提交，不包事务。
-- +goose NO TRANSACTION

-- +goose Up
CREATE TABLE `clients` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  `app_id` varchar(128) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci NOT NULL,
  `role` varchar(16) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci NOT NULL,
  `scope_kind` varchar(16) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci NOT NULL DEFAULT 'shared',
  `scope_id` bigint unsigned NOT NULL DEFAULT '0',
  `status` varchar(16) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci NOT NULL DEFAULT 'enabled',
  `release_version` varchar(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci NOT NULL DEFAULT '',
  `credential_ref` varchar(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci NOT NULL DEFAULT '',
  PRIMARY KEY (`id`) USING BTREE,
  UNIQUE KEY `uk_clients_app_id` (`app_id`) USING BTREE,
  KEY `idx_clients_deleted_at` (`deleted_at`) USING BTREE,
  KEY `idx_clients_role` (`role`) USING BTREE,
  KEY `idx_clients_status` (`status`) USING BTREE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- +goose Down
DROP TABLE IF EXISTS `clients`;
