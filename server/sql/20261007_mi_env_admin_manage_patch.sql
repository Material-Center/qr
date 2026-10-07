-- MI 环境管理后台：筛选索引、菜单、API 与管理员权限（幂等）
-- 适用：MySQL / MariaDB

START TRANSACTION;

INSERT INTO `sys_base_menus`
(`created_at`, `updated_at`, `menu_level`, `parent_id`, `path`, `name`, `hidden`, `component`, `sort`, `active_name`, `keep_alive`, `default_menu`, `title`, `icon`, `close_tab`, `transition_type`)
SELECT NOW(), NOW(), 0, 0, 'environment-manage', 'miEnvManage', 0, 'view/environment/miEnvManage.vue', 9, '', 0, 0, '环境管理', 'collection', 0, ''
WHERE NOT EXISTS (
  SELECT 1 FROM `sys_base_menus` WHERE `name` = 'miEnvManage' AND `deleted_at` IS NULL
);

INSERT INTO `sys_apis` (`created_at`, `updated_at`, `api_group`, `method`, `path`, `description`)
SELECT NOW(), NOW(), '环境管理', 'POST', '/miEnvAdmin/list', '分页查询环境数据'
WHERE NOT EXISTS (
  SELECT 1 FROM `sys_apis` WHERE `path` = '/miEnvAdmin/list' AND `method` = 'POST' AND `deleted_at` IS NULL
);

INSERT INTO `sys_apis` (`created_at`, `updated_at`, `api_group`, `method`, `path`, `description`)
SELECT NOW(), NOW(), '环境管理', 'POST', '/miEnvAdmin/deleteAll', '按条件删除全部环境数据'
WHERE NOT EXISTS (
  SELECT 1 FROM `sys_apis` WHERE `path` = '/miEnvAdmin/deleteAll' AND `method` = 'POST' AND `deleted_at` IS NULL
);

INSERT INTO `sys_apis` (`created_at`, `updated_at`, `api_group`, `method`, `path`, `description`)
SELECT NOW(), NOW(), '环境管理', 'POST', '/miEnvAdmin/types', '查询设备环境类型'
WHERE NOT EXISTS (
  SELECT 1 FROM `sys_apis` WHERE `path` = '/miEnvAdmin/types' AND `method` = 'POST' AND `deleted_at` IS NULL
);

INSERT INTO `sys_apis` (`created_at`, `updated_at`, `api_group`, `method`, `path`, `description`)
SELECT NOW(), NOW(), '环境管理', 'POST', '/miEnvAdmin/deleteSelected', '删除所选环境数据'
WHERE NOT EXISTS (
  SELECT 1 FROM `sys_apis` WHERE `path` = '/miEnvAdmin/deleteSelected' AND `method` = 'POST' AND `deleted_at` IS NULL
);

INSERT INTO `sys_authority_menus` (`sys_authority_authority_id`, `sys_base_menu_id`)
SELECT roles.`role`, CAST(m.`id` AS CHAR)
FROM (
  SELECT '888' AS `role`
  UNION ALL SELECT '100'
) roles
CROSS JOIN `sys_base_menus` m
WHERE m.`name` = 'miEnvManage'
  AND m.`deleted_at` IS NULL
  AND NOT EXISTS (
    SELECT 1 FROM `sys_authority_menus` am
    WHERE am.`sys_authority_authority_id` = roles.`role`
      AND am.`sys_base_menu_id` = CAST(m.`id` AS CHAR)
  );

INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`)
SELECT 'p', roles.`role`, paths.`api_path`, 'POST', '', '', ''
FROM (
  SELECT '888' AS `role`
  UNION ALL SELECT '100'
) roles
CROSS JOIN (
  SELECT '/miEnvAdmin/list' AS `api_path`
  UNION ALL SELECT '/miEnvAdmin/types'
  UNION ALL SELECT '/miEnvAdmin/deleteSelected'
  UNION ALL SELECT '/miEnvAdmin/deleteAll'
) paths
WHERE NOT EXISTS (
  SELECT 1 FROM `casbin_rule` cr
  WHERE cr.`ptype` = 'p'
    AND cr.`v0` = roles.`role`
    AND cr.`v1` = paths.`api_path`
    AND cr.`v2` = 'POST'
);

COMMIT;

SET @has_mi_env_created_idx := (
  SELECT COUNT(1) FROM information_schema.statistics
  WHERE table_schema = DATABASE()
    AND table_name = 'sys_mi_env_records'
    AND index_name = 'idx_sys_mi_env_records_created_at'
);
SET @sql_mi_env_created_idx := IF(
  @has_mi_env_created_idx = 0,
  'ALTER TABLE `sys_mi_env_records` ADD INDEX `idx_sys_mi_env_records_created_at` (`created_at`)',
  'SELECT 1'
);
PREPARE stmt_mi_env_created_idx FROM @sql_mi_env_created_idx;
EXECUTE stmt_mi_env_created_idx;
DEALLOCATE PREPARE stmt_mi_env_created_idx;

SET @has_mi_env_updated_idx := (
  SELECT COUNT(1) FROM information_schema.statistics
  WHERE table_schema = DATABASE()
    AND table_name = 'sys_mi_env_records'
    AND index_name = 'idx_sys_mi_env_records_updated_at'
);
SET @sql_mi_env_updated_idx := IF(
  @has_mi_env_updated_idx = 0,
  'ALTER TABLE `sys_mi_env_records` ADD INDEX `idx_sys_mi_env_records_updated_at` (`updated_at`)',
  'SELECT 1'
);
PREPARE stmt_mi_env_updated_idx FROM @sql_mi_env_updated_idx;
EXECUTE stmt_mi_env_updated_idx;
DEALLOCATE PREPARE stmt_mi_env_updated_idx;

SET @has_mi_env_usage_idx := (
  SELECT COUNT(1) FROM information_schema.statistics
  WHERE table_schema = DATABASE()
    AND table_name = 'sys_mi_env_records'
    AND index_name = 'idx_sys_mi_env_records_usage_count'
);
SET @sql_mi_env_usage_idx := IF(
  @has_mi_env_usage_idx = 0,
  'ALTER TABLE `sys_mi_env_records` ADD INDEX `idx_sys_mi_env_records_usage_count` (`usage_count`)',
  'SELECT 1'
);
PREPARE stmt_mi_env_usage_idx FROM @sql_mi_env_usage_idx;
EXECUTE stmt_mi_env_usage_idx;
DEALLOCATE PREPARE stmt_mi_env_usage_idx;

SET @has_mi_env_made_idx := (
  SELECT COUNT(1) FROM information_schema.statistics
  WHERE table_schema = DATABASE()
    AND table_name = 'sys_mi_env_records'
    AND index_name = 'idx_sys_mi_env_records_made_count'
);
SET @sql_mi_env_made_idx := IF(
  @has_mi_env_made_idx = 0,
  'ALTER TABLE `sys_mi_env_records` ADD INDEX `idx_sys_mi_env_records_made_count` (`made_count`)',
  'SELECT 1'
);
PREPARE stmt_mi_env_made_idx FROM @sql_mi_env_made_idx;
EXECUTE stmt_mi_env_made_idx;
DEALLOCATE PREPARE stmt_mi_env_made_idx;

SET @has_mi_env_active_updated_idx := (
  SELECT COUNT(1) FROM information_schema.statistics
  WHERE table_schema = DATABASE()
    AND table_name = 'sys_mi_env_records'
    AND index_name = 'idx_mi_env_admin_active_updated'
);
SET @sql_mi_env_active_updated_idx := IF(
  @has_mi_env_active_updated_idx = 0,
  'ALTER TABLE `sys_mi_env_records` ADD INDEX `idx_mi_env_admin_active_updated` (`deleted_at`, `updated_at`, `id`)',
  'SELECT 1'
);
PREPARE stmt_mi_env_active_updated_idx FROM @sql_mi_env_active_updated_idx;
EXECUTE stmt_mi_env_active_updated_idx;
DEALLOCATE PREPARE stmt_mi_env_active_updated_idx;

SET @has_mi_env_device_type_idx := (
  SELECT COUNT(1) FROM information_schema.statistics
  WHERE table_schema = DATABASE()
    AND table_name = 'sys_mi_env_records'
    AND index_name = 'idx_mi_env_admin_device_active_type'
);
SET @sql_mi_env_device_type_idx := IF(
  @has_mi_env_device_type_idx = 0,
  'ALTER TABLE `sys_mi_env_records` ADD INDEX `idx_mi_env_admin_device_active_type` (`device_id`, `deleted_at`, `type`)',
  'SELECT 1'
);
PREPARE stmt_mi_env_device_type_idx FROM @sql_mi_env_device_type_idx;
EXECUTE stmt_mi_env_device_type_idx;
DEALLOCATE PREPARE stmt_mi_env_device_type_idx;

SET @has_mi_env_active_usage_idx := (
  SELECT COUNT(1) FROM information_schema.statistics
  WHERE table_schema = DATABASE()
    AND table_name = 'sys_mi_env_records'
    AND index_name = 'idx_mi_env_admin_active_usage'
);
SET @sql_mi_env_active_usage_idx := IF(
  @has_mi_env_active_usage_idx = 0,
  'ALTER TABLE `sys_mi_env_records` ADD INDEX `idx_mi_env_admin_active_usage` (`deleted_at`, `usage_count`)',
  'SELECT 1'
);
PREPARE stmt_mi_env_active_usage_idx FROM @sql_mi_env_active_usage_idx;
EXECUTE stmt_mi_env_active_usage_idx;
DEALLOCATE PREPARE stmt_mi_env_active_usage_idx;

SET @has_mi_env_active_made_idx := (
  SELECT COUNT(1) FROM information_schema.statistics
  WHERE table_schema = DATABASE()
    AND table_name = 'sys_mi_env_records'
    AND index_name = 'idx_mi_env_admin_active_made'
);
SET @sql_mi_env_active_made_idx := IF(
  @has_mi_env_active_made_idx = 0,
  'ALTER TABLE `sys_mi_env_records` ADD INDEX `idx_mi_env_admin_active_made` (`deleted_at`, `made_count`)',
  'SELECT 1'
);
PREPARE stmt_mi_env_active_made_idx FROM @sql_mi_env_active_made_idx;
EXECUTE stmt_mi_env_active_made_idx;
DEALLOCATE PREPARE stmt_mi_env_active_made_idx;

SET @has_mi_env_active_frozen_idx := (
  SELECT COUNT(1) FROM information_schema.statistics
  WHERE table_schema = DATABASE()
    AND table_name = 'sys_mi_env_records'
    AND index_name = 'idx_mi_env_admin_active_frozen_updated'
);
SET @sql_mi_env_active_frozen_idx := IF(
  @has_mi_env_active_frozen_idx = 0,
  'ALTER TABLE `sys_mi_env_records` ADD INDEX `idx_mi_env_admin_active_frozen_updated` (`deleted_at`, `frozen`, `updated_at`)',
  'SELECT 1'
);
PREPARE stmt_mi_env_active_frozen_idx FROM @sql_mi_env_active_frozen_idx;
EXECUTE stmt_mi_env_active_frozen_idx;
DEALLOCATE PREPARE stmt_mi_env_active_frozen_idx;
