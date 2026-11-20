-- 平台控制台审计：目标园所、请求链路标识与检索索引（design D6）。
-- DDL 隐式提交，不包事务。
-- +goose NO TRANSACTION

-- +goose Up
ALTER TABLE `platform_operation_logs`
  ADD COLUMN `kindergarten_id` bigint unsigned NOT NULL DEFAULT '0' AFTER `target_id`,
  ADD COLUMN `request_id` varchar(64) NOT NULL DEFAULT '' AFTER `ip`,
  ADD KEY `idx_platform_log_kg_time` (`kindergarten_id`, `created_at`) USING BTREE,
  ADD KEY `idx_platform_log_operator_time` (`operator`, `created_at`) USING BTREE;

-- +goose Down
ALTER TABLE `platform_operation_logs`
  DROP KEY `idx_platform_log_operator_time`,
  DROP KEY `idx_platform_log_kg_time`,
  DROP COLUMN `request_id`,
  DROP COLUMN `kindergarten_id`;
