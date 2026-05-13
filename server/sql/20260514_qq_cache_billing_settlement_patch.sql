-- QQ缓存管理：系统售卖计费结算补丁（幂等）

SET @has_billing_settled_at := (
  SELECT COUNT(1)
  FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME = 'sys_qq_cache_records'
    AND COLUMN_NAME = 'billing_settled_at'
);
SET @sql_add_billing_settled_at := IF(
  @has_billing_settled_at = 0,
  'ALTER TABLE `sys_qq_cache_records` ADD COLUMN `billing_settled_at` datetime(3) NULL COMMENT ''计费结算时间'' AFTER `device_id`',
  'SELECT 1'
);
PREPARE stmt_add_billing_settled_at FROM @sql_add_billing_settled_at;
EXECUTE stmt_add_billing_settled_at;
DEALLOCATE PREPARE stmt_add_billing_settled_at;

SET @has_billing_settled_by := (
  SELECT COUNT(1)
  FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME = 'sys_qq_cache_records'
    AND COLUMN_NAME = 'billing_settled_by'
);
SET @sql_add_billing_settled_by := IF(
  @has_billing_settled_by = 0,
  'ALTER TABLE `sys_qq_cache_records` ADD COLUMN `billing_settled_by` bigint unsigned NULL COMMENT ''计费结算管理员ID'' AFTER `billing_settled_at`',
  'SELECT 1'
);
PREPARE stmt_add_billing_settled_by FROM @sql_add_billing_settled_by;
EXECUTE stmt_add_billing_settled_by;
DEALLOCATE PREPARE stmt_add_billing_settled_by;

SET @has_idx_billing_settled_at := (
  SELECT COUNT(1)
  FROM information_schema.STATISTICS
  WHERE TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME = 'sys_qq_cache_records'
    AND INDEX_NAME = 'idx_sys_qq_cache_records_billing_settled_at'
);
SET @sql_add_idx_billing_settled_at := IF(
  @has_idx_billing_settled_at = 0,
  'ALTER TABLE `sys_qq_cache_records` ADD INDEX `idx_sys_qq_cache_records_billing_settled_at` (`billing_settled_at`)',
  'SELECT 1'
);
PREPARE stmt_add_idx_billing_settled_at FROM @sql_add_idx_billing_settled_at;
EXECUTE stmt_add_idx_billing_settled_at;
DEALLOCATE PREPARE stmt_add_idx_billing_settled_at;

SET @has_idx_billing_settled_by := (
  SELECT COUNT(1)
  FROM information_schema.STATISTICS
  WHERE TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME = 'sys_qq_cache_records'
    AND INDEX_NAME = 'idx_sys_qq_cache_records_billing_settled_by'
);
SET @sql_add_idx_billing_settled_by := IF(
  @has_idx_billing_settled_by = 0,
  'ALTER TABLE `sys_qq_cache_records` ADD INDEX `idx_sys_qq_cache_records_billing_settled_by` (`billing_settled_by`)',
  'SELECT 1'
);
PREPARE stmt_add_idx_billing_settled_by FROM @sql_add_idx_billing_settled_by;
EXECUTE stmt_add_idx_billing_settled_by;
DEALLOCATE PREPARE stmt_add_idx_billing_settled_by;

INSERT INTO `sys_apis` (`created_at`, `updated_at`, `api_group`, `method`, `path`, `description`)
SELECT NOW(), NOW(), 'QQ缓存', 'POST', '/qqCache/billing/settle', '管理端结算QQ缓存计费数量'
WHERE NOT EXISTS (
  SELECT 1 FROM `sys_apis` WHERE `path` = '/qqCache/billing/settle' AND `method` = 'POST' AND `deleted_at` IS NULL
);

INSERT INTO `sys_apis` (`created_at`, `updated_at`, `api_group`, `method`, `path`, `description`)
SELECT NOW(), NOW(), 'QQ缓存', 'GET', '/qqCache/billing/history', '管理端查询QQ缓存计费结算历史'
WHERE NOT EXISTS (
  SELECT 1 FROM `sys_apis` WHERE `path` = '/qqCache/billing/history' AND `method` = 'GET' AND `deleted_at` IS NULL
);

INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`)
SELECT 'p', '888', '/qqCache/billing/settle', 'POST', '', '', ''
WHERE NOT EXISTS (
  SELECT 1 FROM `casbin_rule` WHERE `ptype` = 'p' AND `v0` = '888' AND `v1` = '/qqCache/billing/settle' AND `v2` = 'POST'
);

INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`)
SELECT 'p', '888', '/qqCache/billing/history', 'GET', '', '', ''
WHERE NOT EXISTS (
  SELECT 1 FROM `casbin_rule` WHERE `ptype` = 'p' AND `v0` = '888' AND `v1` = '/qqCache/billing/history' AND `v2` = 'GET'
);

INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`)
SELECT 'p', '100', '/qqCache/billing/settle', 'POST', '', '', ''
WHERE NOT EXISTS (
  SELECT 1 FROM `casbin_rule` WHERE `ptype` = 'p' AND `v0` = '100' AND `v1` = '/qqCache/billing/settle' AND `v2` = 'POST'
);

INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`)
SELECT 'p', '100', '/qqCache/billing/history', 'GET', '', '', ''
WHERE NOT EXISTS (
  SELECT 1 FROM `casbin_rule` WHERE `ptype` = 'p' AND `v0` = '100' AND `v1` = '/qqCache/billing/history' AND `v2` = 'GET'
);
