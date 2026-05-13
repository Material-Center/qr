-- 手机号注册任务缓存处理状态字段补丁（幂等）
-- pending = 成功后待上传缓存；uploaded = 缓存已上传并绑定；timeout = 成功后缓存上传超时未上传

SET @db_name = DATABASE();

SET @sql = (
  SELECT IF(
    EXISTS (
      SELECT 1
      FROM information_schema.COLUMNS
      WHERE TABLE_SCHEMA = @db_name
        AND TABLE_NAME = 'sys_phone_register_tasks'
        AND COLUMN_NAME = 'cache_status'
    ),
    'SELECT 1',
    'ALTER TABLE `sys_phone_register_tasks` ADD COLUMN `cache_status` varchar(32) NOT NULL DEFAULT '''' COMMENT ''缓存处理状态'' AFTER `task_source`'
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
        AND INDEX_NAME = 'idx_sys_phone_register_tasks_cache_status'
    ),
    'SELECT 1',
    'ALTER TABLE `sys_phone_register_tasks` ADD INDEX `idx_sys_phone_register_tasks_cache_status` (`cache_status`)'
  )
);
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

UPDATE `sys_phone_register_tasks`
SET `cache_status` = 'uploaded'
WHERE `qq_cache_record_id` IS NOT NULL
  AND (`cache_status` IS NULL OR `cache_status` = '');

UPDATE `sys_phone_register_tasks`
SET `cache_status` = 'pending'
WHERE `task_source` = 'OPENAPI'
  AND `status` = 'succeeded'
  AND `qq_cache_record_id` IS NULL
  AND (`cache_status` IS NULL OR `cache_status` = '');
