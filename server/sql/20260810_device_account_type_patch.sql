-- 设备账号类型：QQ缓存账号类型字段、设备配置、管理员菜单与权限补丁（幂等）
-- 适用：MySQL / MariaDB
-- 注意：不预置 qq_cache_sales_allowed_account_types，发布后管理员未配置时销售不可导出。

START TRANSACTION;

CREATE TABLE IF NOT EXISTS `sys_device_configs` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) NULL DEFAULT NULL,
  `updated_at` datetime(3) NULL DEFAULT NULL,
  `deleted_at` datetime(3) NULL DEFAULT NULL,
  `device_id` varchar(128) NOT NULL COMMENT '设备ID',
  `account_type` varchar(32) NOT NULL DEFAULT 'default' COMMENT '账号类型',
  `remark` varchar(255) NULL DEFAULT NULL COMMENT '备注',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_sys_device_configs_device_id` (`device_id`),
  KEY `idx_sys_device_configs_deleted_at` (`deleted_at`),
  KEY `idx_sys_device_configs_account_type` (`account_type`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

SET @has_qq_cache_account_type := (
  SELECT COUNT(1)
  FROM information_schema.columns
  WHERE table_schema = DATABASE()
    AND table_name = 'sys_qq_cache_records'
    AND column_name = 'account_type'
);
SET @sql_add_qq_cache_account_type := IF(
  @has_qq_cache_account_type = 0,
  'ALTER TABLE `sys_qq_cache_records` ADD COLUMN `account_type` varchar(32) NOT NULL DEFAULT ''default'' COMMENT ''账号类型'' AFTER `device_id`',
  'SELECT 1'
);
PREPARE stmt_add_qq_cache_account_type FROM @sql_add_qq_cache_account_type;
EXECUTE stmt_add_qq_cache_account_type;
DEALLOCATE PREPARE stmt_add_qq_cache_account_type;

SET @has_qq_cache_account_type_idx := (
  SELECT COUNT(1)
  FROM information_schema.statistics
  WHERE table_schema = DATABASE()
    AND table_name = 'sys_qq_cache_records'
    AND index_name = 'idx_sys_qq_cache_records_account_type'
);
SET @sql_add_qq_cache_account_type_idx := IF(
  @has_qq_cache_account_type_idx = 0,
  'ALTER TABLE `sys_qq_cache_records` ADD INDEX `idx_sys_qq_cache_records_account_type` (`account_type`)',
  'SELECT 1'
);
PREPARE stmt_add_qq_cache_account_type_idx FROM @sql_add_qq_cache_account_type_idx;
EXECUTE stmt_add_qq_cache_account_type_idx;
DEALLOCATE PREPARE stmt_add_qq_cache_account_type_idx;

INSERT INTO `sys_base_menus`
(`created_at`, `updated_at`, `menu_level`, `parent_id`, `path`, `name`, `hidden`, `component`, `sort`, `active_name`, `keep_alive`, `default_menu`, `title`, `icon`, `close_tab`, `transition_type`)
SELECT NOW(), NOW(), 0, 0, 'device-manage', 'deviceManage', 0, 'view/device/deviceConfig.vue', 8, '', 0, 0, '设备管理', 'monitor', 0, ''
WHERE NOT EXISTS (
  SELECT 1 FROM `sys_base_menus` WHERE `name` = 'deviceManage' AND `deleted_at` IS NULL
);

INSERT INTO `sys_apis` (`created_at`, `updated_at`, `api_group`, `method`, `path`, `description`)
SELECT NOW(), NOW(), '设备管理', 'POST', '/deviceConfig/list', '分页查询设备配置'
WHERE NOT EXISTS (
  SELECT 1 FROM `sys_apis` WHERE `path` = '/deviceConfig/list' AND `method` = 'POST' AND `deleted_at` IS NULL
);

INSERT INTO `sys_apis` (`created_at`, `updated_at`, `api_group`, `method`, `path`, `description`)
SELECT NOW(), NOW(), '设备管理', 'POST', '/deviceConfig/save', '保存设备配置'
WHERE NOT EXISTS (
  SELECT 1 FROM `sys_apis` WHERE `path` = '/deviceConfig/save' AND `method` = 'POST' AND `deleted_at` IS NULL
);

INSERT INTO `sys_apis` (`created_at`, `updated_at`, `api_group`, `method`, `path`, `description`)
SELECT NOW(), NOW(), '设备管理', 'POST', '/deviceConfig/delete', '删除设备配置'
WHERE NOT EXISTS (
  SELECT 1 FROM `sys_apis` WHERE `path` = '/deviceConfig/delete' AND `method` = 'POST' AND `deleted_at` IS NULL
);

INSERT INTO `sys_apis` (`created_at`, `updated_at`, `api_group`, `method`, `path`, `description`)
SELECT NOW(), NOW(), 'QQ缓存', 'GET', '/qqCache/accountTypes', '查询QQ缓存账号类型'
WHERE NOT EXISTS (
  SELECT 1 FROM `sys_apis` WHERE `path` = '/qqCache/accountTypes' AND `method` = 'GET' AND `deleted_at` IS NULL
);

INSERT INTO `sys_apis` (`created_at`, `updated_at`, `api_group`, `method`, `path`, `description`)
SELECT NOW(), NOW(), 'QQ缓存', 'GET', '/qqCache/salesAllowedAccountTypes', '读取销售可导出账号类型'
WHERE NOT EXISTS (
  SELECT 1 FROM `sys_apis` WHERE `path` = '/qqCache/salesAllowedAccountTypes' AND `method` = 'GET' AND `deleted_at` IS NULL
);

INSERT INTO `sys_apis` (`created_at`, `updated_at`, `api_group`, `method`, `path`, `description`)
SELECT NOW(), NOW(), 'QQ缓存', 'PUT', '/qqCache/salesAllowedAccountTypes', '保存销售可导出账号类型'
WHERE NOT EXISTS (
  SELECT 1 FROM `sys_apis` WHERE `path` = '/qqCache/salesAllowedAccountTypes' AND `method` = 'PUT' AND `deleted_at` IS NULL
);

INSERT INTO `sys_authority_menus` (`sys_authority_authority_id`, `sys_base_menu_id`)
SELECT '888', CAST(m.`id` AS CHAR)
FROM `sys_base_menus` m
WHERE m.`name` = 'deviceManage'
  AND m.`deleted_at` IS NULL
  AND NOT EXISTS (
    SELECT 1 FROM `sys_authority_menus` am
    WHERE am.`sys_authority_authority_id` = '888'
      AND am.`sys_base_menu_id` = CAST(m.`id` AS CHAR)
  );

INSERT INTO `sys_authority_menus` (`sys_authority_authority_id`, `sys_base_menu_id`)
SELECT '100', CAST(m.`id` AS CHAR)
FROM `sys_base_menus` m
WHERE m.`name` = 'deviceManage'
  AND m.`deleted_at` IS NULL
  AND NOT EXISTS (
    SELECT 1 FROM `sys_authority_menus` am
    WHERE am.`sys_authority_authority_id` = '100'
      AND am.`sys_base_menu_id` = CAST(m.`id` AS CHAR)
  );

INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`)
SELECT 'p', roles.`role`, paths.`api_path`, paths.`method`, '', '', ''
FROM (
  SELECT '888' AS `role`
  UNION ALL SELECT '100'
) roles
CROSS JOIN (
  SELECT '/deviceConfig/list' AS `api_path`, 'POST' AS `method`
  UNION ALL SELECT '/deviceConfig/save', 'POST'
  UNION ALL SELECT '/deviceConfig/delete', 'POST'
  UNION ALL SELECT '/qqCache/accountTypes', 'GET'
  UNION ALL SELECT '/qqCache/salesAllowedAccountTypes', 'GET'
  UNION ALL SELECT '/qqCache/salesAllowedAccountTypes', 'PUT'
) paths
WHERE NOT EXISTS (
  SELECT 1 FROM `casbin_rule` cr
  WHERE cr.`ptype` = 'p'
    AND cr.`v0` = roles.`role`
    AND cr.`v1` = paths.`api_path`
    AND cr.`v2` = paths.`method`
);

COMMIT;
