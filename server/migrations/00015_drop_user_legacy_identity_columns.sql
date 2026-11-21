-- 旧跨端身份退役（#173）：
--   客户端身份的唯一来源是 user_identities(app_id, open_id)；users.open_id /
--   users.union_id 已无任何读写，保留会形成「一个身份两处存」的双源。
--
-- 不回填旧数据：v1 尚未发布，开发库按需重置；且旧 users 行没有 app_id，
-- 无法可靠映射到 user_identities 的客户端身份。MySQL 删除列时会一并删除
-- 单列索引 idx_users_open_id / idx_users_union_id。
-- DDL 隐式提交，不包事务。
-- +goose NO TRANSACTION

-- +goose Up

ALTER TABLE `users`
  DROP COLUMN `open_id`,
  DROP COLUMN `union_id`;

-- +goose Down

-- 只恢复结构，不恢复数据：旧列的值已随 DROP 丢失，回滚后均为 NULL。
ALTER TABLE `users`
  ADD COLUMN `open_id` varchar(128) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci DEFAULT NULL,
  ADD COLUMN `union_id` varchar(128) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci DEFAULT NULL,
  ADD KEY `idx_users_open_id` (`open_id`) USING BTREE,
  ADD KEY `idx_users_union_id` (`union_id`) USING BTREE;
