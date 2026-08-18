-- 手机号注册任务执行地区字段补丁（幂等）

SET @column_exists := (
  SELECT COUNT(*)
  FROM INFORMATION_SCHEMA.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME = 'sys_phone_register_tasks'
    AND COLUMN_NAME = 'region'
);

SET @task_source_exists := (
  SELECT COUNT(*)
  FROM INFORMATION_SCHEMA.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME = 'sys_phone_register_tasks'
    AND COLUMN_NAME = 'task_source'
);

SET @after_column := IF(@task_source_exists = 0, 'create_source', 'task_source');

SET @sql := IF(
  @column_exists = 0,
  CONCAT('ALTER TABLE `sys_phone_register_tasks` ADD COLUMN `region` varchar(128) NOT NULL DEFAULT '''' COMMENT ''执行地区'' AFTER `', @after_column, '`'),
  'SELECT 1'
);
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

UPDATE `sys_phone_register_tasks` SET `region` = '' WHERE `region` IS NULL;

SET @sql := CONCAT('ALTER TABLE `sys_phone_register_tasks` MODIFY COLUMN `region` varchar(128) NOT NULL DEFAULT '''' COMMENT ''执行地区'' AFTER `', @after_column, '`');
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;
