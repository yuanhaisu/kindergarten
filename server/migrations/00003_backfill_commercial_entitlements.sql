-- 为存量园所回填商业化授权，保证上线后现有环境功能不变：
--   1) 每园一条永久 base 授权（来源 gift）
--   2) 每园每条 premium 模块一条 30 天试用授权（来源 trial）
-- premium 模块清单必须与 server/internal/service 的模块注册表保持一致。
-- +goose NO TRANSACTION

-- +goose Up
INSERT INTO `entitlements`
  (`created_at`, `updated_at`, `deleted_at`, `kindergarten_id`, `module_code`, `source`, `order_id`, `starts_at`, `expires_at`)
SELECT NOW(3), NOW(3), NULL, k.`id`, 'base', 'gift', NULL, '2000-01-01 00:00:00', '2099-12-31 23:59:59'
FROM `kindergartens` k
WHERE k.`deleted_at` IS NULL
  AND NOT EXISTS (
    SELECT 1 FROM `entitlements` e
    WHERE e.`kindergarten_id` = k.`id` AND e.`module_code` = 'base' AND e.`deleted_at` IS NULL
  );

INSERT INTO `entitlements`
  (`created_at`, `updated_at`, `deleted_at`, `kindergarten_id`, `module_code`, `source`, `order_id`, `starts_at`, `expires_at`)
SELECT NOW(3), NOW(3), NULL, k.`id`, m.`code`, 'trial', NULL, NOW(3), DATE_ADD(NOW(3), INTERVAL 30 DAY)
FROM `kindergartens` k
CROSS JOIN (
  SELECT 'growth' AS `code`
  UNION ALL SELECT 'album'
  UNION ALL SELECT 'growth_report'
  UNION ALL SELECT 'diet'
) m
WHERE k.`deleted_at` IS NULL
  AND NOT EXISTS (
    SELECT 1 FROM `entitlements` e
    WHERE e.`kindergarten_id` = k.`id`
      AND e.`module_code` = m.`code`
      AND e.`source` = 'trial'
      AND e.`deleted_at` IS NULL
  );

-- +goose Down
DELETE FROM `entitlements` WHERE `source` IN ('gift', 'trial') AND `order_id` IS NULL;
