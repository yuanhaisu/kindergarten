-- 佣金按净额结算：原始佣金行记录累计冲回金额，退款收款行记录被退的原收款。
-- 不删除、不改写历史行；存量数据无冲回记录，reversed_amount_cents 默认 0 即正确。
-- +goose NO TRANSACTION

-- +goose Up
ALTER TABLE `commission_ledger`
  ADD COLUMN `reversed_amount_cents` bigint NOT NULL DEFAULT '0' AFTER `amount_cents`;

ALTER TABLE `payments`
  ADD COLUMN `refund_of_payment_id` bigint unsigned DEFAULT NULL AFTER `order_id`,
  ADD KEY `idx_payments_refund_of` (`refund_of_payment_id`) USING BTREE;

-- +goose Down
ALTER TABLE `payments`
  DROP KEY `idx_payments_refund_of`,
  DROP COLUMN `refund_of_payment_id`;

ALTER TABLE `commission_ledger`
  DROP COLUMN `reversed_amount_cents`;
