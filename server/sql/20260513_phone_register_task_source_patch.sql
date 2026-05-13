-- 手机号注册任务执行来源字段补丁（幂等）
-- SCRIPT = AutoX脚本领取执行；OPENAPI = OpenAPI领取执行

SET @db_name = DATABASE();

SET @sql = (
  SELECT IF(
    EXISTS (
      SELECT 1
      FROM information_schema.COLUMNS
      WHERE TABLE_SCHEMA = @db_name
        AND TABLE_NAME = 'sys_phone_register_tasks'
        AND COLUMN_NAME = 'task_source'
    ),
    'SELECT 1',
    'ALTER TABLE `sys_phone_register_tasks` ADD COLUMN `task_source` varchar(32) NOT NULL DEFAULT '''' COMMENT ''任务执行来源'' AFTER `sms_receive_mode`'
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
        AND INDEX_NAME = 'idx_sys_phone_register_tasks_task_source'
    ),
    'SELECT 1',
    'ALTER TABLE `sys_phone_register_tasks` ADD INDEX `idx_sys_phone_register_tasks_task_source` (`task_source`)'
  )
);
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;
