-- 设备分组：设备分组表、设备配置 group_id 字段与管理员权限补丁（幂等）
-- 适用：MySQL / MariaDB

START TRANSACTION;

CREATE TABLE IF NOT EXISTS `sys_device_groups` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) NULL DEFAULT NULL,
  `updated_at` datetime(3) NULL DEFAULT NULL,
  `deleted_at` datetime(3) NULL DEFAULT NULL,
  `name` varchar(64) NOT NULL COMMENT '分组名称',
  `remark` varchar(255) NULL DEFAULT NULL COMMENT '备注',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_sys_device_groups_name` (`name`),
  KEY `idx_sys_device_groups_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

SET @has_device_config_group_id := (
  SELECT COUNT(1)
  FROM information_schema.columns
  WHERE table_schema = DATABASE()
    AND table_name = 'sys_device_configs'
    AND column_name = 'group_id'
);
SET @sql_add_device_config_group_id := IF(
  @has_device_config_group_id = 0,
  'ALTER TABLE `sys_device_configs` ADD COLUMN `group_id` bigint unsigned NULL DEFAULT NULL COMMENT ''设备分组ID'' AFTER `account_type`',
  'SELECT 1'
);
PREPARE stmt_add_device_config_group_id FROM @sql_add_device_config_group_id;
EXECUTE stmt_add_device_config_group_id;
DEALLOCATE PREPARE stmt_add_device_config_group_id;

SET @has_device_config_group_id_idx := (
  SELECT COUNT(1)
  FROM information_schema.statistics
  WHERE table_schema = DATABASE()
    AND table_name = 'sys_device_configs'
    AND index_name = 'idx_sys_device_configs_group_id'
);
SET @sql_add_device_config_group_id_idx := IF(
  @has_device_config_group_id_idx = 0,
  'ALTER TABLE `sys_device_configs` ADD INDEX `idx_sys_device_configs_group_id` (`group_id`)',
  'SELECT 1'
);
PREPARE stmt_add_device_config_group_id_idx FROM @sql_add_device_config_group_id_idx;
EXECUTE stmt_add_device_config_group_id_idx;
DEALLOCATE PREPARE stmt_add_device_config_group_id_idx;

INSERT INTO `sys_apis` (`created_at`, `updated_at`, `api_group`, `method`, `path`, `description`)
SELECT NOW(), NOW(), '设备管理', 'GET', '/deviceConfig/group/list', '查询设备分组'
WHERE NOT EXISTS (
  SELECT 1 FROM `sys_apis` WHERE `path` = '/deviceConfig/group/list' AND `method` = 'GET' AND `deleted_at` IS NULL
);

INSERT INTO `sys_apis` (`created_at`, `updated_at`, `api_group`, `method`, `path`, `description`)
SELECT NOW(), NOW(), '设备管理', 'POST', '/deviceConfig/group/save', '保存设备分组'
WHERE NOT EXISTS (
  SELECT 1 FROM `sys_apis` WHERE `path` = '/deviceConfig/group/save' AND `method` = 'POST' AND `deleted_at` IS NULL
);

INSERT INTO `sys_apis` (`created_at`, `updated_at`, `api_group`, `method`, `path`, `description`)
SELECT NOW(), NOW(), '设备管理', 'POST', '/deviceConfig/group/delete', '删除设备分组'
WHERE NOT EXISTS (
  SELECT 1 FROM `sys_apis` WHERE `path` = '/deviceConfig/group/delete' AND `method` = 'POST' AND `deleted_at` IS NULL
);

INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`)
SELECT 'p', roles.`role`, paths.`api_path`, paths.`method`, '', '', ''
FROM (
  SELECT '888' AS `role`
  UNION ALL SELECT '100'
) roles
CROSS JOIN (
  SELECT '/deviceConfig/group/list' AS `api_path`, 'GET' AS `method`
  UNION ALL SELECT '/deviceConfig/group/save', 'POST'
  UNION ALL SELECT '/deviceConfig/group/delete', 'POST'
) paths
WHERE NOT EXISTS (
  SELECT 1 FROM `casbin_rule` cr
  WHERE cr.`ptype` = 'p'
    AND cr.`v0` = roles.`role`
    AND cr.`v1` = paths.`api_path`
    AND cr.`v2` = paths.`method`
);

COMMIT;
