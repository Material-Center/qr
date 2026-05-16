-- 手机号注册禁用号段配置补丁（幂等）
-- 适用：MySQL / MariaDB

START TRANSACTION;

SET @db_name = DATABASE();
SET @default_phone_register_blocked_prefixes = '133,149,153,173,177,180,181,189,190,193,199';

SET @sql = (
  SELECT IF(
    EXISTS (
      SELECT 1
      FROM information_schema.COLUMNS
      WHERE TABLE_SCHEMA = @db_name
        AND TABLE_NAME = 'sys_register_configs'
        AND COLUMN_NAME = 'phone_register_blocked_prefixes'
    ),
    'SELECT 1',
    CONCAT(
      'ALTER TABLE `sys_register_configs` ADD COLUMN `phone_register_blocked_prefixes` varchar(256) NOT NULL DEFAULT ''',
      @default_phone_register_blocked_prefixes,
      ''' COMMENT ''手机号注册禁用手机号前缀，逗号分隔'''
    )
  )
);
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

UPDATE `sys_register_configs`
SET `phone_register_blocked_prefixes` = @default_phone_register_blocked_prefixes
WHERE `phone_register_blocked_prefixes` IS NULL
   OR `phone_register_blocked_prefixes` = '';

COMMIT;
