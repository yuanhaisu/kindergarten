-- 订单付款到期日：独立于服务期，回填老订单后保证非空（design D1）。
-- 先回填再置 NOT NULL：AutoMigrate 只加列不回填，开发库必须走本迁移。
-- DDL 隐式提交，不包事务。
-- +goose NO TRANSACTION

-- +goose Up
ALTER TABLE `orders`
  ADD COLUMN `pay_due_at` datetime(3) NULL AFTER `expires_at`;

UPDATE `orders` SET `pay_due_at` = `starts_at` WHERE `pay_due_at` IS NULL;

ALTER TABLE `orders`
  MODIFY COLUMN `pay_due_at` datetime(3) NOT NULL,
  ADD KEY `idx_orders_pay_due_at` (`pay_due_at`) USING BTREE;

-- +goose Down
ALTER TABLE `orders`
  DROP KEY `idx_orders_pay_due_at`,
  DROP COLUMN `pay_due_at`;
