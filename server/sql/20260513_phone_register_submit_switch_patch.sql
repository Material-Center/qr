-- 手机号注册提交开关补丁（幂等）
-- 适用：MySQL / MariaDB

START TRANSACTION;

SET @db_name = DATABASE();

SET @sql = (
  SELECT IF(
    EXISTS (
      SELECT 1
      FROM information_schema.COLUMNS
      WHERE TABLE_SCHEMA = @db_name
        AND TABLE_NAME = 'sys_register_configs'
        AND COLUMN_NAME = 'phone_register_enabled'
    ),
    'SELECT 1',
    'ALTER TABLE `sys_register_configs` ADD COLUMN `phone_register_enabled` tinyint(1) NULL DEFAULT 1 COMMENT ''是否允许地推提交手机号注册'''
  )
);
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

UPDATE `sys_register_configs`
SET `phone_register_enabled` = 1
WHERE `phone_register_enabled` IS NULL;

INSERT INTO `sys_apis` (`created_at`,`updated_at`,`api_group`,`method`,`path`,`description`)
SELECT NOW(), NOW(), '手机号注册任务', 'GET', '/phoneRegisterTask/submitStatus', '获取手机号注册提交开关'
WHERE NOT EXISTS (
  SELECT 1 FROM `sys_apis` WHERE `path`='/phoneRegisterTask/submitStatus' AND `method`='GET' AND `deleted_at` IS NULL
);

INSERT INTO `casbin_rule` (`ptype`,`v0`,`v1`,`v2`,`v3`,`v4`,`v5`)
SELECT 'p','888','/phoneRegisterTask/submitStatus','GET','','',''
WHERE NOT EXISTS (
  SELECT 1 FROM `casbin_rule` WHERE `ptype`='p' AND `v0`='888' AND `v1`='/phoneRegisterTask/submitStatus' AND `v2`='GET'
);

INSERT INTO `casbin_rule` (`ptype`,`v0`,`v1`,`v2`,`v3`,`v4`,`v5`)
SELECT 'p','300','/phoneRegisterTask/submitStatus','GET','','',''
WHERE NOT EXISTS (
  SELECT 1 FROM `casbin_rule` WHERE `ptype`='p' AND `v0`='300' AND `v1`='/phoneRegisterTask/submitStatus' AND `v2`='GET'
);

COMMIT;
