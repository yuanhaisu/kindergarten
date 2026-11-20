-- 订单信用开通（T4）：零收款确认必须留理由并标记，理由只在详情展示。
-- 依赖 00009 已加的 pay_due_at；与付期无关，单独成迁移。
-- DDL 隐式提交，不包事务。
-- +goose NO TRANSACTION

-- +goose Up
ALTER TABLE `orders`
  ADD COLUMN `credit_granted` tinyint(1) NOT NULL DEFAULT '0' AFTER `pay_due_at`,
  ADD COLUMN `credit_reason` varchar(255) NOT NULL DEFAULT '' AFTER `credit_granted`;

-- +goose Down
ALTER TABLE `orders`
  DROP COLUMN `credit_reason`,
  DROP COLUMN `credit_granted`;
