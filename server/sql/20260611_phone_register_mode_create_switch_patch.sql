-- 手机号注册按收码方式禁止创建开关补丁（幂等）
-- 适用：MySQL / MariaDB

START TRANSACTION;

SET @db_name = DATABASE();

SET @has_user_sent_disabled := (
  SELECT COUNT(1)
  FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = @db_name
    AND TABLE_NAME = 'sys_register_configs'
    AND COLUMN_NAME = 'phone_register_user_sent_task_disabled'
);
SET @sql_add_user_sent_disabled := IF(
  @has_user_sent_disabled = 0,
  'ALTER TABLE `sys_register_configs` ADD COLUMN `phone_register_user_sent_task_disabled` tinyint(1) NOT NULL DEFAULT 0 COMMENT ''是否禁止创建自己发码任务'' AFTER `phone_register_enabled`',
  'SELECT 1'
);
PREPARE stmt_add_user_sent_disabled FROM @sql_add_user_sent_disabled;
EXECUTE stmt_add_user_sent_disabled;
DEALLOCATE PREPARE stmt_add_user_sent_disabled;

SET @has_receive_disabled := (
  SELECT COUNT(1)
  FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = @db_name
    AND TABLE_NAME = 'sys_register_configs'
    AND COLUMN_NAME = 'phone_register_receive_task_disabled'
);
SET @sql_add_receive_disabled := IF(
  @has_receive_disabled = 0,
  'ALTER TABLE `sys_register_configs` ADD COLUMN `phone_register_receive_task_disabled` tinyint(1) NOT NULL DEFAULT 0 COMMENT ''是否禁止创建收码任务'' AFTER `phone_register_user_sent_task_disabled`',
  'SELECT 1'
);
PREPARE stmt_add_receive_disabled FROM @sql_add_receive_disabled;
EXECUTE stmt_add_receive_disabled;
DEALLOCATE PREPARE stmt_add_receive_disabled;

UPDATE `sys_register_configs`
SET `phone_register_user_sent_task_disabled` = 0
WHERE `phone_register_user_sent_task_disabled` IS NULL;

UPDATE `sys_register_configs`
SET `phone_register_receive_task_disabled` = 0
WHERE `phone_register_receive_task_disabled` IS NULL;

COMMIT;
