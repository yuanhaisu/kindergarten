-- 平台账号角色列（T6）：为平台内分权预留，当前所有账号一律 platform，
-- 不参与任何鉴权判断，接口也不返回该字段（design D4）。
-- DDL 隐式提交，不包事务。
-- +goose NO TRANSACTION

-- +goose Up
ALTER TABLE `platform_admins`
  ADD COLUMN `role` varchar(32) NOT NULL DEFAULT 'platform' AFTER `status`;

-- +goose Down
ALTER TABLE `platform_admins`
  DROP COLUMN `role`;
