-- 账号管理：用户级跳过手机号注册禁用号段开关（幂等）

SET @has_phone_register_skip_blocked_prefixes := (
  SELECT COUNT(1)
  FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME = 'sys_users'
    AND COLUMN_NAME = 'phone_register_skip_blocked_prefixes'
);
SET @sql_add_phone_register_skip_blocked_prefixes := IF(
  @has_phone_register_skip_blocked_prefixes = 0,
  'ALTER TABLE `sys_users` ADD COLUMN `phone_register_skip_blocked_prefixes` tinyint(1) NOT NULL DEFAULT 0 COMMENT ''是否跳过手机号注册禁用号段'' AFTER `phone_register_task_disabled`',
  'SELECT 1'
);
PREPARE stmt_add_phone_register_skip_blocked_prefixes FROM @sql_add_phone_register_skip_blocked_prefixes;
EXECUTE stmt_add_phone_register_skip_blocked_prefixes;
DEALLOCATE PREPARE stmt_add_phone_register_skip_blocked_prefixes;

UPDATE `sys_users`
SET `phone_register_skip_blocked_prefixes` = 0
WHERE `phone_register_skip_blocked_prefixes` IS NULL;
