-- Saa S 商业化第一期：平台账号、套餐目录、订单收款、功能授权、分销佣金。
-- 权威来源：openspec/changes/add-saas-commercial-layer/ 与 docs/adr/0011-0016。
-- +goose NO TRANSACTION

-- +goose Up
CREATE TABLE `platform_admins` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  `username` varchar(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci NOT NULL,
  `password_hash` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci NOT NULL,
  `name` varchar(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci NOT NULL DEFAULT '',
  `status` tinyint NOT NULL DEFAULT '1',
  PRIMARY KEY (`id`) USING BTREE,
  UNIQUE KEY `uk_platform_admin_username` (`username`) USING BTREE,
  KEY `idx_platform_admins_deleted_at` (`deleted_at`) USING BTREE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE `distributors` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  `name` varchar(128) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci NOT NULL,
  `contact` varchar(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci NOT NULL DEFAULT '',
  `phone` varchar(32) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci NOT NULL DEFAULT '',
  `commission_rate` int NOT NULL DEFAULT '0',
  `status` tinyint NOT NULL DEFAULT '1',
  PRIMARY KEY (`id`) USING BTREE,
  KEY `idx_distributors_deleted_at` (`deleted_at`) USING BTREE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE `plans` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  `code` varchar(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci NOT NULL,
  `name` varchar(128) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci NOT NULL,
  `tier` int NOT NULL DEFAULT '1',
  `price_cents` bigint NOT NULL DEFAULT '0',
  `term_months` int NOT NULL DEFAULT '6',
  `status` varchar(16) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci NOT NULL DEFAULT 'active',
  PRIMARY KEY (`id`) USING BTREE,
  UNIQUE KEY `uk_plan_code` (`code`) USING BTREE,
  KEY `idx_plans_deleted_at` (`deleted_at`) USING BTREE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE `plan_modules` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  `plan_id` bigint unsigned NOT NULL,
  `module_code` varchar(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci NOT NULL,
  PRIMARY KEY (`id`) USING BTREE,
  UNIQUE KEY `uk_plan_module` (`plan_id`,`module_code`) USING BTREE,
  KEY `idx_plan_modules_deleted_at` (`deleted_at`) USING BTREE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE `plan_tiers` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  `plan_id` bigint unsigned NOT NULL,
  `tier` int NOT NULL,
  `min_students` int NOT NULL DEFAULT '0',
  `max_students` int NOT NULL DEFAULT '0',
  `price_multiplier` int NOT NULL DEFAULT '100',
  PRIMARY KEY (`id`) USING BTREE,
  UNIQUE KEY `uk_plan_tier` (`plan_id`,`tier`) USING BTREE,
  KEY `idx_plan_tiers_deleted_at` (`deleted_at`) USING BTREE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE `orders` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  `order_no` varchar(32) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci NOT NULL,
  `kindergarten_id` bigint unsigned NOT NULL,
  `plan_id` bigint unsigned NOT NULL,
  `plan_snapshot` json NOT NULL,
  `student_tier` int NOT NULL DEFAULT '1',
  `amount_due_cents` bigint NOT NULL DEFAULT '0',
  `status` varchar(16) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci NOT NULL DEFAULT 'unpaid',
  `distributor_id` bigint unsigned DEFAULT NULL,
  `commission_rate` int NOT NULL DEFAULT '0',
  `invoice_info` json DEFAULT NULL,
  `starts_at` datetime(3) NOT NULL,
  `expires_at` datetime(3) NOT NULL,
  `confirmed_at` datetime(3) DEFAULT NULL,
  `cancelled_at` datetime(3) DEFAULT NULL,
  `cancel_reason` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci NOT NULL DEFAULT '',
  `created_by` varchar(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci NOT NULL DEFAULT '',
  PRIMARY KEY (`id`) USING BTREE,
  UNIQUE KEY `uk_order_no` (`order_no`) USING BTREE,
  KEY `idx_orders_kg` (`kindergarten_id`) USING BTREE,
  KEY `idx_orders_status` (`status`) USING BTREE,
  KEY `idx_orders_distributor` (`distributor_id`) USING BTREE,
  KEY `idx_orders_expires_at` (`expires_at`) USING BTREE,
  KEY `idx_orders_deleted_at` (`deleted_at`) USING BTREE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE `payments` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  `order_id` bigint unsigned NOT NULL,
  `kindergarten_id` bigint unsigned NOT NULL,
  `amount_cents` bigint NOT NULL,
  `method` varchar(32) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci NOT NULL DEFAULT 'bank_transfer',
  `received_at` datetime(3) NOT NULL,
  `voucher_url` varchar(512) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci NOT NULL DEFAULT '',
  `remark` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci NOT NULL DEFAULT '',
  `recorded_by` varchar(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci NOT NULL DEFAULT '',
  PRIMARY KEY (`id`) USING BTREE,
  KEY `idx_payments_order` (`order_id`) USING BTREE,
  KEY `idx_payments_kg` (`kindergarten_id`) USING BTREE,
  KEY `idx_payments_received_at` (`received_at`) USING BTREE,
  KEY `idx_payments_deleted_at` (`deleted_at`) USING BTREE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE `entitlements` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  `kindergarten_id` bigint unsigned NOT NULL,
  `module_code` varchar(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci NOT NULL,
  `source` varchar(16) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci NOT NULL DEFAULT 'subscription',
  `order_id` bigint unsigned DEFAULT NULL,
  `starts_at` datetime(3) NOT NULL,
  `expires_at` datetime(3) NOT NULL,
  PRIMARY KEY (`id`) USING BTREE,
  KEY `idx_ent_kg_module` (`kindergarten_id`,`module_code`) USING BTREE,
  KEY `idx_entitlements_expires_at` (`expires_at`) USING BTREE,
  KEY `idx_entitlements_order` (`order_id`) USING BTREE,
  KEY `idx_entitlements_deleted_at` (`deleted_at`) USING BTREE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE `commission_ledger` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  `order_id` bigint unsigned NOT NULL,
  `payment_id` bigint unsigned NOT NULL,
  `distributor_id` bigint unsigned NOT NULL,
  `kindergarten_id` bigint unsigned NOT NULL,
  `base_amount_cents` bigint NOT NULL,
  `rate` int NOT NULL,
  `amount_cents` bigint NOT NULL,
  `status` varchar(16) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci NOT NULL DEFAULT 'pending',
  `eligible_at` datetime(3) NOT NULL,
  `reversal_of` bigint unsigned DEFAULT NULL,
  PRIMARY KEY (`id`) USING BTREE,
  KEY `idx_commission_order` (`order_id`) USING BTREE,
  KEY `idx_commission_payment` (`payment_id`) USING BTREE,
  KEY `idx_commission_distributor` (`distributor_id`) USING BTREE,
  KEY `idx_commission_kg` (`kindergarten_id`) USING BTREE,
  KEY `idx_commission_status` (`status`) USING BTREE,
  KEY `idx_commission_eligible_at` (`eligible_at`) USING BTREE,
  KEY `idx_commission_reversal_of` (`reversal_of`) USING BTREE,
  KEY `idx_commission_ledger_deleted_at` (`deleted_at`) USING BTREE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

ALTER TABLE `kindergartens`
  ADD COLUMN `status` varchar(16) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci NOT NULL DEFAULT 'active' AFTER `address`;

-- +goose Down
ALTER TABLE `kindergartens` DROP COLUMN `status`;

DROP TABLE IF EXISTS `commission_ledger`;
DROP TABLE IF EXISTS `entitlements`;
DROP TABLE IF EXISTS `payments`;
DROP TABLE IF EXISTS `orders`;
DROP TABLE IF EXISTS `plan_tiers`;
DROP TABLE IF EXISTS `plan_modules`;
DROP TABLE IF EXISTS `plans`;
DROP TABLE IF EXISTS `distributors`;
DROP TABLE IF EXISTS `platform_admins`;
