-- 账号管理：用户级手机号注册任务创建禁用开关（幂等）

SET @has_phone_register_task_disabled := (
  SELECT COUNT(1)
  FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME = 'sys_users'
    AND COLUMN_NAME = 'phone_register_task_disabled'
);
SET @sql_add_phone_register_task_disabled := IF(
  @has_phone_register_task_disabled = 0,
  'ALTER TABLE `sys_users` ADD COLUMN `phone_register_task_disabled` tinyint(1) NOT NULL DEFAULT 0 COMMENT ''是否禁用创建手机号注册任务'' AFTER `enable`',
  'SELECT 1'
);
PREPARE stmt_add_phone_register_task_disabled FROM @sql_add_phone_register_task_disabled;
EXECUTE stmt_add_phone_register_task_disabled;
DEALLOCATE PREPARE stmt_add_phone_register_task_disabled;

UPDATE `sys_users`
SET `phone_register_task_disabled` = 0
WHERE `phone_register_task_disabled` IS NULL;

-- 兼容已执行过旧版“允许创建任务”字段的环境：enabled=0 迁移为 disabled=1。
SET @has_phone_register_task_enabled := (
  SELECT COUNT(1)
  FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME = 'sys_users'
    AND COLUMN_NAME = 'phone_register_task_enabled'
);
SET @sql_migrate_phone_register_task_enabled := IF(
  @has_phone_register_task_enabled > 0,
  'UPDATE `sys_users` SET `phone_register_task_disabled` = IF(`phone_register_task_enabled` = 0, 1, 0)',
  'SELECT 1'
);
PREPARE stmt_migrate_phone_register_task_enabled FROM @sql_migrate_phone_register_task_enabled;
EXECUTE stmt_migrate_phone_register_task_enabled;
DEALLOCATE PREPARE stmt_migrate_phone_register_task_enabled;
