-- QQ缓存客户端版本字段补丁（幂等）
-- 适用：MySQL / MariaDB

START TRANSACTION;

SET @has_client_version := (
  SELECT COUNT(1)
  FROM information_schema.columns
  WHERE table_schema = DATABASE()
    AND table_name = 'sys_qq_cache_records'
    AND column_name = 'client_version'
);
SET @sql_add_client_version := IF(
  @has_client_version = 0,
  'ALTER TABLE `sys_qq_cache_records` ADD COLUMN `client_version` varchar(64) NULL DEFAULT NULL COMMENT ''客户端版本号'' AFTER `qq_pwd`',
  'SELECT 1'
);
PREPARE stmt_add_client_version FROM @sql_add_client_version;
EXECUTE stmt_add_client_version;
DEALLOCATE PREPARE stmt_add_client_version;

SET @has_client_version_idx := (
  SELECT COUNT(1)
  FROM information_schema.statistics
  WHERE table_schema = DATABASE()
    AND table_name = 'sys_qq_cache_records'
    AND index_name = 'idx_sys_qq_cache_records_client_version'
);
SET @sql_add_client_version_idx := IF(
  @has_client_version_idx = 0,
  'ALTER TABLE `sys_qq_cache_records` ADD INDEX `idx_sys_qq_cache_records_client_version` (`client_version`)',
  'SELECT 1'
);
PREPARE stmt_add_client_version_idx FROM @sql_add_client_version_idx;
EXECUTE stmt_add_client_version_idx;
DEALLOCATE PREPARE stmt_add_client_version_idx;

COMMIT;
