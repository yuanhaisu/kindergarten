-- 收款幂等：同一订单下的幂等键唯一。
-- 未携带幂等键的旧请求写 NULL；MySQL 唯一索引允许多个 NULL，旧请求之间不会互相冲突。
-- +goose NO TRANSACTION

-- +goose Up
ALTER TABLE `payments`
  ADD COLUMN `idempotency_key` varchar(64) DEFAULT NULL AFTER `refund_of_payment_id`,
  ADD UNIQUE KEY `uk_payments_order_idem` (`order_id`, `idempotency_key`) USING BTREE;

-- +goose Down
ALTER TABLE `payments`
  DROP KEY `uk_payments_order_idem`,
  DROP COLUMN `idempotency_key`;
