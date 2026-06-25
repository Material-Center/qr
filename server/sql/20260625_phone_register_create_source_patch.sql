-- 手机号注册任务创建来源字段补丁（幂等）
-- MANUAL = 后台/地推手动创建；OPENAPI = 地推 OpenAPI 创建

SET @db_name = DATABASE();

SET @sql = (
  SELECT IF(
    EXISTS (
      SELECT 1
      FROM information_schema.COLUMNS
      WHERE TABLE_SCHEMA = @db_name
        AND TABLE_NAME = 'sys_phone_register_tasks'
        AND COLUMN_NAME = 'create_source'
    ),
    'SELECT 1',
    'ALTER TABLE `sys_phone_register_tasks` ADD COLUMN `create_source` varchar(32) NOT NULL DEFAULT '''' COMMENT ''任务创建来源'' AFTER `sms_receive_mode`'
  )
);
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @sql = (
  SELECT IF(
    EXISTS (
      SELECT 1
      FROM information_schema.STATISTICS
      WHERE TABLE_SCHEMA = @db_name
        AND TABLE_NAME = 'sys_phone_register_tasks'
        AND INDEX_NAME = 'idx_sys_phone_register_tasks_create_source'
    ),
    'SELECT 1',
    'ALTER TABLE `sys_phone_register_tasks` ADD INDEX `idx_sys_phone_register_tasks_create_source` (`create_source`)'
  )
);
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

UPDATE `sys_phone_register_tasks`
SET `create_source` = 'OPENAPI'
WHERE (`create_source` IS NULL OR `create_source` = '')
  AND `task_source` = 'OPENAPI';

UPDATE `sys_phone_register_tasks`
SET `create_source` = 'MANUAL'
WHERE `create_source` IS NULL OR `create_source` = '';
