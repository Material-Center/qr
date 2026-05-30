-- 手机号注册任务支持服务端延迟可领取时间。

SET @column_exists := (
  SELECT COUNT(*)
  FROM INFORMATION_SCHEMA.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME = 'sys_phone_register_tasks'
    AND COLUMN_NAME = 'available_at'
);

SET @sql := IF(
  @column_exists = 0,
  'ALTER TABLE `sys_phone_register_tasks` ADD COLUMN `available_at` datetime(3) NULL DEFAULT NULL COMMENT ''可领取时间'' AFTER `last_heartbeat_at`',
  'SELECT 1'
);
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @index_exists := (
  SELECT COUNT(*)
  FROM INFORMATION_SCHEMA.STATISTICS
  WHERE TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME = 'sys_phone_register_tasks'
    AND INDEX_NAME = 'idx_sys_phone_register_tasks_available_at'
);

SET @sql := IF(
  @index_exists = 0,
  'ALTER TABLE `sys_phone_register_tasks` ADD INDEX `idx_sys_phone_register_tasks_available_at` (`available_at`)',
  'SELECT 1'
);
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;
