-- QQ缓存销售提取：销售角色、提取批次、菜单、API、权限补丁（幂等）
-- 适用：MySQL / MariaDB

START TRANSACTION;

CREATE TABLE IF NOT EXISTS `sys_qq_cache_extract_batches` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) NULL DEFAULT NULL,
  `updated_at` datetime(3) NULL DEFAULT NULL,
  `deleted_at` datetime(3) NULL DEFAULT NULL,
  `extractor_id` bigint unsigned NOT NULL DEFAULT 0 COMMENT '销售用户ID',
  `extractor_name` varchar(128) NOT NULL DEFAULT '' COMMENT '销售名称快照',
  `extract_count` int NOT NULL DEFAULT 0 COMMENT '本次提取账号数量',
  `settled_count` int NOT NULL DEFAULT 0 COMMENT '已结算账号数量',
  `status` varchar(32) NOT NULL DEFAULT 'pending_settlement' COMMENT '结算状态',
  `extracted_at` datetime(3) NULL DEFAULT NULL COMMENT '提取时间',
  `settled_at` datetime(3) NULL DEFAULT NULL COMMENT '整批结算时间',
  `settled_by` bigint unsigned NULL DEFAULT NULL COMMENT '结算管理员ID',
  PRIMARY KEY (`id`),
  KEY `idx_sys_qq_cache_extract_batches_deleted_at` (`deleted_at`),
  KEY `idx_sys_qq_cache_extract_batches_extractor_id` (`extractor_id`),
  KEY `idx_sys_qq_cache_extract_batches_status` (`status`),
  KEY `idx_sys_qq_cache_extract_batches_extracted_at` (`extracted_at`),
  KEY `idx_sys_qq_cache_extract_batches_settled_at` (`settled_at`),
  KEY `idx_sys_qq_cache_extract_batches_settled_by` (`settled_by`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

SET @has_sales_settled_at := (
  SELECT COUNT(1)
  FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME = 'sys_qq_cache_records'
    AND COLUMN_NAME = 'sales_settled_at'
);
SET @sql_add_sales_settled_at := IF(
  @has_sales_settled_at = 0,
  'ALTER TABLE `sys_qq_cache_records` ADD COLUMN `sales_settled_at` datetime(3) NULL COMMENT ''销售结算时间'' AFTER `billing_settled_by`',
  'SELECT 1'
);
PREPARE stmt_add_sales_settled_at FROM @sql_add_sales_settled_at;
EXECUTE stmt_add_sales_settled_at;
DEALLOCATE PREPARE stmt_add_sales_settled_at;

SET @has_sales_settled_by := (
  SELECT COUNT(1)
  FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME = 'sys_qq_cache_records'
    AND COLUMN_NAME = 'sales_settled_by'
);
SET @sql_add_sales_settled_by := IF(
  @has_sales_settled_by = 0,
  'ALTER TABLE `sys_qq_cache_records` ADD COLUMN `sales_settled_by` bigint unsigned NULL DEFAULT NULL COMMENT ''销售结算管理员ID'' AFTER `sales_settled_at`',
  'SELECT 1'
);
PREPARE stmt_add_sales_settled_by FROM @sql_add_sales_settled_by;
EXECUTE stmt_add_sales_settled_by;
DEALLOCATE PREPARE stmt_add_sales_settled_by;

SET @has_idx_sales_settled_at := (
  SELECT COUNT(1)
  FROM information_schema.STATISTICS
  WHERE TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME = 'sys_qq_cache_records'
    AND INDEX_NAME = 'idx_sys_qq_cache_records_sales_settled_at'
);
SET @sql_add_idx_sales_settled_at := IF(
  @has_idx_sales_settled_at = 0,
  'ALTER TABLE `sys_qq_cache_records` ADD INDEX `idx_sys_qq_cache_records_sales_settled_at` (`sales_settled_at`)',
  'SELECT 1'
);
PREPARE stmt_add_idx_sales_settled_at FROM @sql_add_idx_sales_settled_at;
EXECUTE stmt_add_idx_sales_settled_at;
DEALLOCATE PREPARE stmt_add_idx_sales_settled_at;

SET @has_idx_sales_settled_by := (
  SELECT COUNT(1)
  FROM information_schema.STATISTICS
  WHERE TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME = 'sys_qq_cache_records'
    AND INDEX_NAME = 'idx_sys_qq_cache_records_sales_settled_by'
);
SET @sql_add_idx_sales_settled_by := IF(
  @has_idx_sales_settled_by = 0,
  'ALTER TABLE `sys_qq_cache_records` ADD INDEX `idx_sys_qq_cache_records_sales_settled_by` (`sales_settled_by`)',
  'SELECT 1'
);
PREPARE stmt_add_idx_sales_settled_by FROM @sql_add_idx_sales_settled_by;
EXECUTE stmt_add_idx_sales_settled_by;
DEALLOCATE PREPARE stmt_add_idx_sales_settled_by;

INSERT INTO `sys_authorities` (`authority_id`, `authority_name`, `parent_id`, `default_router`, `created_at`, `updated_at`)
SELECT 600, '销售', 100, 'qqCacheExtract', NOW(), NOW()
WHERE NOT EXISTS (
  SELECT 1 FROM `sys_authorities` WHERE `authority_id` = 600 AND `deleted_at` IS NULL
);

INSERT INTO `sys_base_menus`
(`created_at`, `updated_at`, `menu_level`, `parent_id`, `path`, `name`, `hidden`, `component`, `sort`, `active_name`, `keep_alive`, `default_menu`, `title`, `icon`, `close_tab`, `transition_type`)
SELECT NOW(), NOW(), 0, 0, 'qq-cache-extract', 'qqCacheExtract', 0, 'view/register/qqCacheExtract.vue', 7, '', 0, 0, '缓存提取', 'download', 0, ''
WHERE NOT EXISTS (
  SELECT 1 FROM `sys_base_menus` WHERE `name` = 'qqCacheExtract' AND `deleted_at` IS NULL
);

INSERT INTO `sys_authority_menus` (`sys_authority_authority_id`, `sys_base_menu_id`)
SELECT '600', CAST(m.`id` AS CHAR)
FROM `sys_base_menus` m
WHERE m.`name` = 'qqCacheExtract'
  AND m.`deleted_at` IS NULL
  AND NOT EXISTS (
    SELECT 1 FROM `sys_authority_menus` am
    WHERE am.`sys_authority_authority_id` = '600'
      AND am.`sys_base_menu_id` = CAST(m.`id` AS CHAR)
  );

INSERT INTO `sys_apis` (`created_at`, `updated_at`, `api_group`, `method`, `path`, `description`)
SELECT NOW(), NOW(), 'QQ缓存', 'GET', '/qqCache/sales/summary', '销售查询缓存提取汇总'
WHERE NOT EXISTS (
  SELECT 1 FROM `sys_apis` WHERE `path` = '/qqCache/sales/summary' AND `method` = 'GET' AND `deleted_at` IS NULL
);

INSERT INTO `sys_apis` (`created_at`, `updated_at`, `api_group`, `method`, `path`, `description`)
SELECT NOW(), NOW(), 'QQ缓存', 'POST', '/qqCache/sales/extract', '销售按数量提取QQ缓存'
WHERE NOT EXISTS (
  SELECT 1 FROM `sys_apis` WHERE `path` = '/qqCache/sales/extract' AND `method` = 'POST' AND `deleted_at` IS NULL
);

INSERT INTO `sys_apis` (`created_at`, `updated_at`, `api_group`, `method`, `path`, `description`)
SELECT NOW(), NOW(), 'QQ缓存', 'POST', '/qqCache/sales/history', '销售查询提取历史'
WHERE NOT EXISTS (
  SELECT 1 FROM `sys_apis` WHERE `path` = '/qqCache/sales/history' AND `method` = 'POST' AND `deleted_at` IS NULL
);

INSERT INTO `sys_apis` (`created_at`, `updated_at`, `api_group`, `method`, `path`, `description`)
SELECT NOW(), NOW(), 'QQ缓存', 'GET', '/qqCache/sales/summaryList', '管理端按销售查询提取汇总'
WHERE NOT EXISTS (
  SELECT 1 FROM `sys_apis` WHERE `path` = '/qqCache/sales/summaryList' AND `method` = 'GET' AND `deleted_at` IS NULL
);

INSERT INTO `sys_apis` (`created_at`, `updated_at`, `api_group`, `method`, `path`, `description`)
SELECT NOW(), NOW(), 'QQ缓存', 'POST', '/qqCache/sales/settle', '管理端按销售结算QQ缓存'
WHERE NOT EXISTS (
  SELECT 1 FROM `sys_apis` WHERE `path` = '/qqCache/sales/settle' AND `method` = 'POST' AND `deleted_at` IS NULL
);

INSERT INTO `sys_apis` (`created_at`, `updated_at`, `api_group`, `method`, `path`, `description`)
SELECT NOW(), NOW(), 'QQ缓存', 'GET', '/qqCache/sales/settlement/history', '管理端查询销售结算历史'
WHERE NOT EXISTS (
  SELECT 1 FROM `sys_apis` WHERE `path` = '/qqCache/sales/settlement/history' AND `method` = 'GET' AND `deleted_at` IS NULL
);

INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`)
SELECT 'p', role_id, api_path, api_method, '', '', ''
FROM (
  SELECT '600' AS role_id, '/menu/getMenu' AS api_path, 'POST' AS api_method
  UNION ALL SELECT '600', '/menu/getMenuList', 'POST'
  UNION ALL SELECT '600', '/menu/getBaseMenuTree', 'POST'
  UNION ALL SELECT '600', '/user/getUserInfo', 'GET'
  UNION ALL SELECT '600', '/user/changePassword', 'POST'
  UNION ALL SELECT '600', '/user/setSelfInfo', 'PUT'
  UNION ALL SELECT '600', '/user/setSelfSetting', 'PUT'
  UNION ALL SELECT '600', '/qqCache/sales/summary', 'GET'
  UNION ALL SELECT '600', '/qqCache/sales/extract', 'POST'
  UNION ALL SELECT '600', '/qqCache/sales/history', 'POST'
  UNION ALL SELECT '600', '/jwt/jsonInBlacklist', 'POST'
  UNION ALL SELECT '100', '/qqCache/sales/summaryList', 'GET'
  UNION ALL SELECT '100', '/qqCache/sales/settle', 'POST'
  UNION ALL SELECT '100', '/qqCache/sales/settlement/history', 'GET'
  UNION ALL SELECT '888', '/qqCache/sales/summaryList', 'GET'
  UNION ALL SELECT '888', '/qqCache/sales/settle', 'POST'
  UNION ALL SELECT '888', '/qqCache/sales/settlement/history', 'GET'
) p
WHERE NOT EXISTS (
  SELECT 1 FROM `casbin_rule` c
  WHERE c.`ptype` = 'p'
    AND c.`v0` = p.role_id
    AND c.`v1` = p.api_path
    AND c.`v2` = p.api_method
);

COMMIT;
