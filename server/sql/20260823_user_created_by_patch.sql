-- 记录账号创建人。字段由服务端根据当前登录账号写入，客户端不可指定。
-- 现有数据保持 NULL，不根据历史关系猜测创建人。
SET @column_exists = (
  SELECT COUNT(*)
  FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME = 'sys_users'
    AND COLUMN_NAME = 'created_by'
);

SET @ddl = IF(
  @column_exists = 0,
  'ALTER TABLE `sys_users` ADD COLUMN `created_by` bigint unsigned NOT NULL DEFAULT 0 COMMENT ''创建人账号ID'' AFTER `leader_id`, ADD INDEX `idx_sys_users_created_by` (`created_by`)',
  'SELECT 1'
);

PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;
