-- 园所入口字段（#177）：租户通道、公开入口码、品牌主题色。
-- #170（公开品牌）/ #176（家长端二维码）/ #178（品牌设置）依赖本迁移。
--
-- v1 所有园所 app_channel=shared；dedicated / group 只保留取值。
-- public_code 为空串表示尚未生成；唯一索引建在 STORED 生成列 public_code_uniq 上，
-- 把空串转成 NULL：未生成码的园所可以并存，非空码全局唯一，软删园所的码也不复用。
-- DDL 隐式提交，不包事务。
-- +goose NO TRANSACTION

-- +goose Up
ALTER TABLE `kindergartens`
  ADD COLUMN `app_channel` varchar(16) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci NOT NULL DEFAULT 'shared' AFTER `description`,
  ADD COLUMN `public_code` varchar(16) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci NOT NULL DEFAULT '' AFTER `app_channel`,
  ADD COLUMN `brand_color` varchar(32) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci NOT NULL DEFAULT '' AFTER `public_code`,
  ADD KEY `idx_kindergartens_app_channel` (`app_channel`) USING BTREE;

-- 存量回填：10 位小写十六进制。与运行时 base32 生成器不要求同构，唯一即可。
UPDATE `kindergartens`
SET `public_code` = LOWER(SUBSTRING(MD5(CONCAT(UUID(), '-', `id`)), 1, 10))
WHERE `public_code` = '';

ALTER TABLE `kindergartens`
  ADD COLUMN `public_code_uniq` varchar(16) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci
    GENERATED ALWAYS AS (IF(`public_code` = '', NULL, `public_code`)) STORED AFTER `public_code`;

ALTER TABLE `kindergartens`
  ADD UNIQUE KEY `uk_kindergartens_public_code` (`public_code_uniq`) USING BTREE;

-- +goose Down
ALTER TABLE `kindergartens`
  DROP INDEX `uk_kindergartens_public_code`;

ALTER TABLE `kindergartens`
  DROP INDEX `idx_kindergartens_app_channel`,
  DROP COLUMN `brand_color`,
  DROP COLUMN `public_code_uniq`,
  DROP COLUMN `public_code`,
  DROP COLUMN `app_channel`;
