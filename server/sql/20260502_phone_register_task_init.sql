-- 手机号注册任务结构补丁（幂等）
-- 适用：MySQL / MariaDB
-- 说明：包含结构升级、API 元数据、Casbin 规则；菜单仍使用独立菜单补丁

START TRANSACTION;

CREATE TABLE IF NOT EXISTS `sys_phone_register_tasks` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  `phone` varchar(20) NOT NULL DEFAULT '',
  `promoter_id` bigint unsigned NOT NULL DEFAULT 0,
  `leader_id` bigint unsigned DEFAULT NULL,
  `sms_receive_mode` varchar(32) NOT NULL DEFAULT '',
  `create_source` varchar(32) NOT NULL DEFAULT '',
  `qq_num` varchar(32) NOT NULL DEFAULT '',
  `qq_cache_record_id` bigint unsigned DEFAULT NULL,
  `pending_code` varchar(32) NOT NULL DEFAULT '',
  `status` varchar(32) NOT NULL DEFAULT 'pending',
  `status_code` int DEFAULT NULL,
  `last_error` longtext,
  `finished_at` datetime(3) DEFAULT NULL,
  `holder_device_id` varchar(128) DEFAULT NULL,
  `claimed_at` datetime(3) DEFAULT NULL,
  `last_heartbeat_at` datetime(3) DEFAULT NULL,
  `expires_at` datetime(3) NOT NULL,
  `retry_count` bigint NOT NULL DEFAULT 0,
  PRIMARY KEY (`id`),
  KEY `idx_sys_phone_register_tasks_deleted_at` (`deleted_at`),
  KEY `idx_sys_phone_register_tasks_phone` (`phone`),
  KEY `idx_sys_phone_register_tasks_promoter_id` (`promoter_id`),
  KEY `idx_sys_phone_register_tasks_leader_id` (`leader_id`),
  KEY `idx_sys_phone_register_tasks_sms_receive_mode` (`sms_receive_mode`),
  KEY `idx_sys_phone_register_tasks_create_source` (`create_source`),
  KEY `idx_sys_phone_register_tasks_qq_num` (`qq_num`),
  KEY `idx_sys_phone_register_tasks_qq_cache_record_id` (`qq_cache_record_id`),
  KEY `idx_sys_phone_register_tasks_status` (`status`),
  KEY `idx_sys_phone_register_tasks_status_code` (`status_code`),
  KEY `idx_sys_phone_register_tasks_finished_at` (`finished_at`),
  KEY `idx_sys_phone_register_tasks_holder_device_id` (`holder_device_id`),
  KEY `idx_sys_phone_register_tasks_last_heartbeat_at` (`last_heartbeat_at`),
  KEY `idx_sys_phone_register_tasks_expires_at` (`expires_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

CREATE TABLE IF NOT EXISTS `sys_phone_register_task_logs` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  `task_id` bigint unsigned NOT NULL DEFAULT 0,
  `device_id` varchar(128) NOT NULL DEFAULT '',
  `client_time` datetime(3) DEFAULT NULL,
  `message` longtext,
  PRIMARY KEY (`id`),
  KEY `idx_sys_phone_register_task_logs_deleted_at` (`deleted_at`),
  KEY `idx_sys_phone_register_task_logs_task_id` (`task_id`),
  KEY `idx_sys_phone_register_task_logs_device_id` (`device_id`),
  KEY `idx_sys_phone_register_task_logs_client_time` (`client_time`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

SET @sql = (
  SELECT IF(
    EXISTS (
      SELECT 1
      FROM information_schema.COLUMNS
      WHERE TABLE_SCHEMA = @db_name
        AND TABLE_NAME = 'sys_phone_register_task_logs'
        AND COLUMN_NAME = 'client_time'
    ),
    'SELECT 1',
    'ALTER TABLE `sys_phone_register_task_logs` ADD COLUMN `client_time` datetime(3) DEFAULT NULL AFTER `device_id`'
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
        AND TABLE_NAME = 'sys_phone_register_task_logs'
        AND INDEX_NAME = 'idx_sys_phone_register_task_logs_client_time'
    ),
    'SELECT 1',
    'ALTER TABLE `sys_phone_register_task_logs` ADD INDEX `idx_sys_phone_register_task_logs_client_time` (`client_time`)'
  )
);
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @db_name = DATABASE();

SET @sql = (
  SELECT IF(
    EXISTS (
      SELECT 1
      FROM information_schema.COLUMNS
      WHERE TABLE_SCHEMA = @db_name
        AND TABLE_NAME = 'sys_register_configs'
        AND COLUMN_NAME = 'phone_image_provider'
    ),
    'SELECT 1',
    'ALTER TABLE `sys_register_configs` ADD COLUMN `phone_image_provider` varchar(32) NOT NULL DEFAULT '''' COMMENT ''手机号注册图片识别供应商'''
  )
);
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @sql = (
  SELECT IF(
    EXISTS (
      SELECT 1
      FROM information_schema.COLUMNS
      WHERE TABLE_SCHEMA = @db_name
        AND TABLE_NAME = 'sys_register_configs'
        AND COLUMN_NAME = 'phone_image_provider_username'
    ),
    'SELECT 1',
    'ALTER TABLE `sys_register_configs` ADD COLUMN `phone_image_provider_username` varchar(128) NOT NULL DEFAULT '''' COMMENT ''手机号注册图片识别账号'''
  )
);
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @sql = (
  SELECT IF(
    EXISTS (
      SELECT 1
      FROM information_schema.COLUMNS
      WHERE TABLE_SCHEMA = @db_name
        AND TABLE_NAME = 'sys_register_configs'
        AND COLUMN_NAME = 'phone_image_provider_password'
    ),
    'SELECT 1',
    'ALTER TABLE `sys_register_configs` ADD COLUMN `phone_image_provider_password` varchar(128) NOT NULL DEFAULT '''' COMMENT ''手机号注册图片识别密码'''
  )
);
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @sql = (
  SELECT IF(
    EXISTS (
      SELECT 1
      FROM information_schema.COLUMNS
      WHERE TABLE_SCHEMA = @db_name
        AND TABLE_NAME = 'sys_register_configs'
        AND COLUMN_NAME = 'phone_image_provider_secret_key'
    ),
    'SELECT 1',
    'ALTER TABLE `sys_register_configs` ADD COLUMN `phone_image_provider_secret_key` varchar(256) NOT NULL DEFAULT '''' COMMENT ''手机号注册图片识别密钥'''
  )
);
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- API 元数据
UPDATE `sys_apis`
SET `description` = '获取反扫统计',
    `updated_at` = NOW()
WHERE `path`='/registerTask/summary'
  AND `method`='GET'
  AND `deleted_at` IS NULL;

UPDATE `sys_apis`
SET `description` = '获取注册统计',
    `updated_at` = NOW()
WHERE `path`='/phoneRegisterTask/summary'
  AND `method`='GET'
  AND `deleted_at` IS NULL;

INSERT INTO `sys_apis` (`created_at`,`updated_at`,`api_group`,`method`,`path`,`description`)
SELECT NOW(), NOW(), 'QQ缓存', 'POST', '/qqCache/uploadPhoneRegister', '手机号注册上传QQ缓存并完成任务'
WHERE NOT EXISTS (
  SELECT 1 FROM `sys_apis` WHERE `path`='/qqCache/uploadPhoneRegister' AND `method`='POST' AND `deleted_at` IS NULL
);

INSERT INTO `sys_apis` (`created_at`,`updated_at`,`api_group`,`method`,`path`,`description`)
SELECT NOW(), NOW(), '手机号注册任务', 'POST', '/phoneRegisterTask/create', '地推创建手机号注册任务'
WHERE NOT EXISTS (
  SELECT 1 FROM `sys_apis` WHERE `path`='/phoneRegisterTask/create' AND `method`='POST' AND `deleted_at` IS NULL
);

INSERT INTO `sys_apis` (`created_at`,`updated_at`,`api_group`,`method`,`path`,`description`)
SELECT NOW(), NOW(), '手机号注册任务', 'POST', '/phoneRegisterTask/submitCode', '地推提交手机号注册验证码'
WHERE NOT EXISTS (
  SELECT 1 FROM `sys_apis` WHERE `path`='/phoneRegisterTask/submitCode' AND `method`='POST' AND `deleted_at` IS NULL
);

INSERT INTO `sys_apis` (`created_at`,`updated_at`,`api_group`,`method`,`path`,`description`)
SELECT NOW(), NOW(), '手机号注册任务', 'GET', '/phoneRegisterTask/active', '获取地推当前手机号注册任务'
WHERE NOT EXISTS (
  SELECT 1 FROM `sys_apis` WHERE `path`='/phoneRegisterTask/active' AND `method`='GET' AND `deleted_at` IS NULL
);

INSERT INTO `sys_apis` (`created_at`,`updated_at`,`api_group`,`method`,`path`,`description`)
SELECT NOW(), NOW(), '手机号注册任务', 'GET', '/phoneRegisterTask/actives', '获取地推全部手机号注册任务'
WHERE NOT EXISTS (
  SELECT 1 FROM `sys_apis` WHERE `path`='/phoneRegisterTask/actives' AND `method`='GET' AND `deleted_at` IS NULL
);

INSERT INTO `sys_apis` (`created_at`,`updated_at`,`api_group`,`method`,`path`,`description`)
SELECT NOW(), NOW(), '手机号注册任务', 'POST', '/phoneRegisterTask/list', '分页查询手机号注册任务'
WHERE NOT EXISTS (
  SELECT 1 FROM `sys_apis` WHERE `path`='/phoneRegisterTask/list' AND `method`='POST' AND `deleted_at` IS NULL
);

INSERT INTO `sys_apis` (`created_at`,`updated_at`,`api_group`,`method`,`path`,`description`)
SELECT NOW(), NOW(), '手机号注册任务', 'GET', '/phoneRegisterTask/summary', '获取注册统计'
WHERE NOT EXISTS (
  SELECT 1 FROM `sys_apis` WHERE `path`='/phoneRegisterTask/summary' AND `method`='GET' AND `deleted_at` IS NULL
);

INSERT INTO `sys_apis` (`created_at`,`updated_at`,`api_group`,`method`,`path`,`description`)
SELECT NOW(), NOW(), '手机号注册任务', 'POST', '/phoneRegisterTask/logs', '查询手机号注册任务日志'
WHERE NOT EXISTS (
  SELECT 1 FROM `sys_apis` WHERE `path`='/phoneRegisterTask/logs' AND `method`='POST' AND `deleted_at` IS NULL
);

INSERT INTO `sys_apis` (`created_at`,`updated_at`,`api_group`,`method`,`path`,`description`)
SELECT NOW(), NOW(), '手机号注册任务', 'POST', '/phoneRegisterTask/device/poll', '设备拉取手机号注册任务'
WHERE NOT EXISTS (
  SELECT 1 FROM `sys_apis` WHERE `path`='/phoneRegisterTask/device/poll' AND `method`='POST' AND `deleted_at` IS NULL
);

INSERT INTO `sys_apis` (`created_at`,`updated_at`,`api_group`,`method`,`path`,`description`)
SELECT NOW(), NOW(), '手机号注册任务', 'GET', '/phoneRegisterTask/device/task', '设备查询当前手机号注册任务'
WHERE NOT EXISTS (
  SELECT 1 FROM `sys_apis` WHERE `path`='/phoneRegisterTask/device/task' AND `method`='GET' AND `deleted_at` IS NULL
);

INSERT INTO `sys_apis` (`created_at`,`updated_at`,`api_group`,`method`,`path`,`description`)
SELECT NOW(), NOW(), '手机号注册任务', 'POST', '/phoneRegisterTask/device/task', '设备查询当前手机号注册任务'
WHERE NOT EXISTS (
  SELECT 1 FROM `sys_apis` WHERE `path`='/phoneRegisterTask/device/task' AND `method`='POST' AND `deleted_at` IS NULL
);

INSERT INTO `sys_apis` (`created_at`,`updated_at`,`api_group`,`method`,`path`,`description`)
SELECT NOW(), NOW(), '手机号注册任务', 'POST', '/phoneRegisterTask/device/heartbeat', '设备上报手机号注册任务心跳'
WHERE NOT EXISTS (
  SELECT 1 FROM `sys_apis` WHERE `path`='/phoneRegisterTask/device/heartbeat' AND `method`='POST' AND `deleted_at` IS NULL
);

INSERT INTO `sys_apis` (`created_at`,`updated_at`,`api_group`,`method`,`path`,`description`)
SELECT NOW(), NOW(), '手机号注册任务', 'POST', '/phoneRegisterTask/device/report', '设备上报手机号注册任务进度'
WHERE NOT EXISTS (
  SELECT 1 FROM `sys_apis` WHERE `path`='/phoneRegisterTask/device/report' AND `method`='POST' AND `deleted_at` IS NULL
);

INSERT INTO `sys_apis` (`created_at`,`updated_at`,`api_group`,`method`,`path`,`description`)
SELECT NOW(), NOW(), '手机号注册任务', 'POST', '/phoneRegisterTask/device/log', '设备上报手机号注册任务日志'
WHERE NOT EXISTS (
  SELECT 1 FROM `sys_apis` WHERE `path`='/phoneRegisterTask/device/log' AND `method`='POST' AND `deleted_at` IS NULL
);

INSERT INTO `sys_apis` (`created_at`,`updated_at`,`api_group`,`method`,`path`,`description`)
SELECT NOW(), NOW(), '手机号注册任务', 'GET', '/phoneRegisterTask/device/config', '获取手机号注册设备配置'
WHERE NOT EXISTS (
  SELECT 1 FROM `sys_apis` WHERE `path`='/phoneRegisterTask/device/config' AND `method`='GET' AND `deleted_at` IS NULL
);

-- Casbin 权限：设备端接口当前挂 PublicGroup，不写角色规则
INSERT INTO `casbin_rule` (`ptype`,`v0`,`v1`,`v2`,`v3`,`v4`,`v5`)
SELECT 'p','888','/phoneRegisterTask/create','POST','','',''
WHERE NOT EXISTS (
  SELECT 1 FROM `casbin_rule` WHERE `ptype`='p' AND `v0`='888' AND `v1`='/phoneRegisterTask/create' AND `v2`='POST'
);

INSERT INTO `casbin_rule` (`ptype`,`v0`,`v1`,`v2`,`v3`,`v4`,`v5`)
SELECT 'p','888','/phoneRegisterTask/submitCode','POST','','',''
WHERE NOT EXISTS (
  SELECT 1 FROM `casbin_rule` WHERE `ptype`='p' AND `v0`='888' AND `v1`='/phoneRegisterTask/submitCode' AND `v2`='POST'
);

INSERT INTO `casbin_rule` (`ptype`,`v0`,`v1`,`v2`,`v3`,`v4`,`v5`)
SELECT 'p','888','/phoneRegisterTask/active','GET','','',''
WHERE NOT EXISTS (
  SELECT 1 FROM `casbin_rule` WHERE `ptype`='p' AND `v0`='888' AND `v1`='/phoneRegisterTask/active' AND `v2`='GET'
);

INSERT INTO `casbin_rule` (`ptype`,`v0`,`v1`,`v2`,`v3`,`v4`,`v5`)
SELECT 'p','888','/phoneRegisterTask/actives','GET','','',''
WHERE NOT EXISTS (
  SELECT 1 FROM `casbin_rule` WHERE `ptype`='p' AND `v0`='888' AND `v1`='/phoneRegisterTask/actives' AND `v2`='GET'
);

INSERT INTO `casbin_rule` (`ptype`,`v0`,`v1`,`v2`,`v3`,`v4`,`v5`)
SELECT 'p','888','/phoneRegisterTask/list','POST','','',''
WHERE NOT EXISTS (
  SELECT 1 FROM `casbin_rule` WHERE `ptype`='p' AND `v0`='888' AND `v1`='/phoneRegisterTask/list' AND `v2`='POST'
);

INSERT INTO `casbin_rule` (`ptype`,`v0`,`v1`,`v2`,`v3`,`v4`,`v5`)
SELECT 'p','888','/phoneRegisterTask/summary','GET','','',''
WHERE NOT EXISTS (
  SELECT 1 FROM `casbin_rule` WHERE `ptype`='p' AND `v0`='888' AND `v1`='/phoneRegisterTask/summary' AND `v2`='GET'
);

INSERT INTO `casbin_rule` (`ptype`,`v0`,`v1`,`v2`,`v3`,`v4`,`v5`)
SELECT 'p','888','/phoneRegisterTask/logs','POST','','',''
WHERE NOT EXISTS (
  SELECT 1 FROM `casbin_rule` WHERE `ptype`='p' AND `v0`='888' AND `v1`='/phoneRegisterTask/logs' AND `v2`='POST'
);

INSERT INTO `casbin_rule` (`ptype`,`v0`,`v1`,`v2`,`v3`,`v4`,`v5`)
SELECT 'p','100','/phoneRegisterTask/list','POST','','',''
WHERE NOT EXISTS (
  SELECT 1 FROM `casbin_rule` WHERE `ptype`='p' AND `v0`='100' AND `v1`='/phoneRegisterTask/list' AND `v2`='POST'
);

INSERT INTO `casbin_rule` (`ptype`,`v0`,`v1`,`v2`,`v3`,`v4`,`v5`)
SELECT 'p','100','/phoneRegisterTask/summary','GET','','',''
WHERE NOT EXISTS (
  SELECT 1 FROM `casbin_rule` WHERE `ptype`='p' AND `v0`='100' AND `v1`='/phoneRegisterTask/summary' AND `v2`='GET'
);

INSERT INTO `casbin_rule` (`ptype`,`v0`,`v1`,`v2`,`v3`,`v4`,`v5`)
SELECT 'p','100','/phoneRegisterTask/logs','POST','','',''
WHERE NOT EXISTS (
  SELECT 1 FROM `casbin_rule` WHERE `ptype`='p' AND `v0`='100' AND `v1`='/phoneRegisterTask/logs' AND `v2`='POST'
);

INSERT INTO `casbin_rule` (`ptype`,`v0`,`v1`,`v2`,`v3`,`v4`,`v5`)
SELECT 'p','200','/phoneRegisterTask/list','POST','','',''
WHERE NOT EXISTS (
  SELECT 1 FROM `casbin_rule` WHERE `ptype`='p' AND `v0`='200' AND `v1`='/phoneRegisterTask/list' AND `v2`='POST'
);

INSERT INTO `casbin_rule` (`ptype`,`v0`,`v1`,`v2`,`v3`,`v4`,`v5`)
SELECT 'p','200','/phoneRegisterTask/summary','GET','','',''
WHERE NOT EXISTS (
  SELECT 1 FROM `casbin_rule` WHERE `ptype`='p' AND `v0`='200' AND `v1`='/phoneRegisterTask/summary' AND `v2`='GET'
);

INSERT INTO `casbin_rule` (`ptype`,`v0`,`v1`,`v2`,`v3`,`v4`,`v5`)
SELECT 'p','200','/phoneRegisterTask/logs','POST','','',''
WHERE NOT EXISTS (
  SELECT 1 FROM `casbin_rule` WHERE `ptype`='p' AND `v0`='200' AND `v1`='/phoneRegisterTask/logs' AND `v2`='POST'
);

INSERT INTO `casbin_rule` (`ptype`,`v0`,`v1`,`v2`,`v3`,`v4`,`v5`)
SELECT 'p','300','/phoneRegisterTask/create','POST','','',''
WHERE NOT EXISTS (
  SELECT 1 FROM `casbin_rule` WHERE `ptype`='p' AND `v0`='300' AND `v1`='/phoneRegisterTask/create' AND `v2`='POST'
);

INSERT INTO `casbin_rule` (`ptype`,`v0`,`v1`,`v2`,`v3`,`v4`,`v5`)
SELECT 'p','300','/phoneRegisterTask/submitCode','POST','','',''
WHERE NOT EXISTS (
  SELECT 1 FROM `casbin_rule` WHERE `ptype`='p' AND `v0`='300' AND `v1`='/phoneRegisterTask/submitCode' AND `v2`='POST'
);

INSERT INTO `casbin_rule` (`ptype`,`v0`,`v1`,`v2`,`v3`,`v4`,`v5`)
SELECT 'p','300','/phoneRegisterTask/active','GET','','',''
WHERE NOT EXISTS (
  SELECT 1 FROM `casbin_rule` WHERE `ptype`='p' AND `v0`='300' AND `v1`='/phoneRegisterTask/active' AND `v2`='GET'
);

INSERT INTO `casbin_rule` (`ptype`,`v0`,`v1`,`v2`,`v3`,`v4`,`v5`)
SELECT 'p','300','/phoneRegisterTask/actives','GET','','',''
WHERE NOT EXISTS (
  SELECT 1 FROM `casbin_rule` WHERE `ptype`='p' AND `v0`='300' AND `v1`='/phoneRegisterTask/actives' AND `v2`='GET'
);

INSERT INTO `casbin_rule` (`ptype`,`v0`,`v1`,`v2`,`v3`,`v4`,`v5`)
SELECT 'p','300','/phoneRegisterTask/list','POST','','',''
WHERE NOT EXISTS (
  SELECT 1 FROM `casbin_rule` WHERE `ptype`='p' AND `v0`='300' AND `v1`='/phoneRegisterTask/list' AND `v2`='POST'
);

COMMIT;
