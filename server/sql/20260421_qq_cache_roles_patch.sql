-- QQ缓存管理 + App提取/App上传角色补丁（幂等）
-- 适用：MySQL / MariaDB

START TRANSACTION;

-- 1) 账号缓存表（新系统独立表）
CREATE TABLE IF NOT EXISTS `sys_qq_cache_records` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) NULL DEFAULT NULL,
  `updated_at` datetime(3) NULL DEFAULT NULL,
  `deleted_at` datetime(3) NULL DEFAULT NULL,
  `phone` varchar(20) NULL DEFAULT NULL,
  `qq_num` varchar(64) NOT NULL,
  `qq_pwd` varchar(255) NULL DEFAULT NULL,
  `client_version` varchar(64) NULL DEFAULT NULL,
  `extractor` bigint unsigned NULL DEFAULT NULL,
  `extract_record_id` bigint unsigned NULL DEFAULT NULL,
  `extraction_at` datetime(3) NULL DEFAULT NULL,
  `ini` longtext NULL,
  `device_id` varchar(128) NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_sys_qq_cache_records_deleted_at` (`deleted_at`),
  UNIQUE KEY `uk_qq_cache_qq_num` (`qq_num`),
  KEY `idx_sys_qq_cache_records_client_version` (`client_version`),
  KEY `idx_sys_qq_cache_records_extractor` (`extractor`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- 老环境兜底：补唯一索引（若已存在则跳过）
SET @has_uk_qq_cache_qq_num := (
  SELECT COUNT(1)
  FROM information_schema.statistics
  WHERE table_schema = DATABASE()
    AND table_name = 'sys_qq_cache_records'
    AND index_name = 'uk_qq_cache_qq_num'
);
SET @sql_add_uk_qq_cache_qq_num := IF(
  @has_uk_qq_cache_qq_num = 0,
  'ALTER TABLE `sys_qq_cache_records` ADD UNIQUE INDEX `uk_qq_cache_qq_num` (`qq_num`)',
  'SELECT 1'
);
PREPARE stmt_add_uk_qq_cache_qq_num FROM @sql_add_uk_qq_cache_qq_num;
EXECUTE stmt_add_uk_qq_cache_qq_num;
DEALLOCATE PREPARE stmt_add_uk_qq_cache_qq_num;

-- 2) 角色
INSERT INTO `sys_authorities` (`authority_id`,`authority_name`,`parent_id`,`default_router`,`created_at`,`updated_at`)
SELECT 400, 'App提取', 100, 'about', NOW(), NOW()
WHERE NOT EXISTS (
  SELECT 1 FROM `sys_authorities` WHERE `authority_id`=400 AND `deleted_at` IS NULL
);

INSERT INTO `sys_authorities` (`authority_id`,`authority_name`,`parent_id`,`default_router`,`created_at`,`updated_at`)
SELECT 500, 'App上传', 100, 'about', NOW(), NOW()
WHERE NOT EXISTS (
  SELECT 1 FROM `sys_authorities` WHERE `authority_id`=500 AND `deleted_at` IS NULL
);

-- 3) 菜单：QQ缓存管理（独立一级菜单，仅管理员/超管绑定）
-- 兼容旧补丁：若已存在且挂在“注册任务”下，提升为一级菜单
UPDATE `sys_base_menus`
SET `menu_level` = 0,
    `parent_id` = 0,
    `path` = 'qq-cache-manage',
    `hidden` = 0,
    `component` = 'view/register/qqCacheManage.vue',
    `sort` = 6,
    `title` = 'QQ缓存管理',
    `icon` = 'folder-opened',
    `updated_at` = NOW()
WHERE `name` = 'qqCacheManage'
  AND `deleted_at` IS NULL;

INSERT INTO `sys_base_menus`
(`created_at`,`updated_at`,`menu_level`,`parent_id`,`path`,`name`,`hidden`,`component`,`sort`,`active_name`,`keep_alive`,`default_menu`,`title`,`icon`,`close_tab`,`transition_type`)
SELECT NOW(), NOW(), 0, 0, 'qq-cache-manage', 'qqCacheManage', 0, 'view/register/qqCacheManage.vue', 6, '', 0, 0, 'QQ缓存管理', 'folder-opened', 0, ''
WHERE NOT EXISTS (
    SELECT 1 FROM `sys_base_menus` m
    WHERE m.`name` = 'qqCacheManage'
      AND m.`deleted_at` IS NULL
  );

INSERT INTO `sys_authority_menus` (`sys_authority_authority_id`,`sys_base_menu_id`)
SELECT '888', CAST(m.id AS CHAR)
FROM `sys_base_menus` m
WHERE m.`name`='qqCacheManage' AND m.`deleted_at` IS NULL
  AND NOT EXISTS (
    SELECT 1 FROM `sys_authority_menus` am
    WHERE am.`sys_authority_authority_id`='888'
      AND am.`sys_base_menu_id`=CAST(m.id AS CHAR)
  );

INSERT INTO `sys_authority_menus` (`sys_authority_authority_id`,`sys_base_menu_id`)
SELECT '100', CAST(m.id AS CHAR)
FROM `sys_base_menus` m
WHERE m.`name`='qqCacheManage' AND m.`deleted_at` IS NULL
  AND NOT EXISTS (
    SELECT 1 FROM `sys_authority_menus` am
    WHERE am.`sys_authority_authority_id`='100'
      AND am.`sys_base_menu_id`=CAST(m.id AS CHAR)
  );

-- 4) API 元数据
INSERT INTO `sys_apis` (`created_at`,`updated_at`,`api_group`,`method`,`path`,`description`)
SELECT NOW(), NOW(), 'QQ缓存', 'POST', '/qqCache/upload', 'App上传QQ缓存'
WHERE NOT EXISTS (
  SELECT 1 FROM `sys_apis` WHERE `path`='/qqCache/upload' AND `method`='POST' AND `deleted_at` IS NULL
);

INSERT INTO `sys_apis` (`created_at`,`updated_at`,`api_group`,`method`,`path`,`description`)
SELECT NOW(), NOW(), 'QQ缓存', 'POST', '/qqCache/extract', 'App提取QQ缓存'
WHERE NOT EXISTS (
  SELECT 1 FROM `sys_apis` WHERE `path`='/qqCache/extract' AND `method`='POST' AND `deleted_at` IS NULL
);

INSERT INTO `sys_apis` (`created_at`,`updated_at`,`api_group`,`method`,`path`,`description`)
SELECT NOW(), NOW(), 'QQ缓存', 'POST', '/qqCache/list', '管理端分页查询QQ缓存'
WHERE NOT EXISTS (
  SELECT 1 FROM `sys_apis` WHERE `path`='/qqCache/list' AND `method`='POST' AND `deleted_at` IS NULL
);

INSERT INTO `sys_apis` (`created_at`,`updated_at`,`api_group`,`method`,`path`,`description`)
SELECT NOW(), NOW(), 'QQ缓存', 'POST', '/qqCache/resetExtract', '管理端重置提取锁'
WHERE NOT EXISTS (
  SELECT 1 FROM `sys_apis` WHERE `path`='/qqCache/resetExtract' AND `method`='POST' AND `deleted_at` IS NULL
);

INSERT INTO `sys_apis` (`created_at`,`updated_at`,`api_group`,`method`,`path`,`description`)
SELECT NOW(), NOW(), 'QQ缓存', 'GET', '/qqCache/roleHint', '获取App角色提示'
WHERE NOT EXISTS (
  SELECT 1 FROM `sys_apis` WHERE `path`='/qqCache/roleHint' AND `method`='GET' AND `deleted_at` IS NULL
);

INSERT INTO `sys_apis` (`created_at`,`updated_at`,`api_group`,`method`,`path`,`description`)
SELECT NOW(), NOW(), 'Base', 'POST', '/base/appLogin', 'App用户登录(免验证码，仅App角色)'
WHERE NOT EXISTS (
  SELECT 1 FROM `sys_apis` WHERE `path`='/base/appLogin' AND `method`='POST' AND `deleted_at` IS NULL
);

INSERT INTO `sys_ignore_apis` (`created_at`,`updated_at`,`method`,`path`)
SELECT NOW(), NOW(), 'POST', '/base/appLogin'
WHERE NOT EXISTS (
  SELECT 1 FROM `sys_ignore_apis` WHERE `path`='/base/appLogin' AND `method`='POST' AND `deleted_at` IS NULL
);

-- 5) Casbin 权限
INSERT INTO `casbin_rule` (`ptype`,`v0`,`v1`,`v2`,`v3`,`v4`,`v5`)
SELECT 'p','100','/qqCache/list','POST','','',''
WHERE NOT EXISTS (
  SELECT 1 FROM `casbin_rule` WHERE `ptype`='p' AND `v0`='100' AND `v1`='/qqCache/list' AND `v2`='POST'
);

INSERT INTO `casbin_rule` (`ptype`,`v0`,`v1`,`v2`,`v3`,`v4`,`v5`)
SELECT 'p','100','/qqCache/resetExtract','POST','','',''
WHERE NOT EXISTS (
  SELECT 1 FROM `casbin_rule` WHERE `ptype`='p' AND `v0`='100' AND `v1`='/qqCache/resetExtract' AND `v2`='POST'
);

INSERT INTO `casbin_rule` (`ptype`,`v0`,`v1`,`v2`,`v3`,`v4`,`v5`)
SELECT 'p','400','/qqCache/extract','POST','','',''
WHERE NOT EXISTS (
  SELECT 1 FROM `casbin_rule` WHERE `ptype`='p' AND `v0`='400' AND `v1`='/qqCache/extract' AND `v2`='POST'
);

INSERT INTO `casbin_rule` (`ptype`,`v0`,`v1`,`v2`,`v3`,`v4`,`v5`)
SELECT 'p','500','/qqCache/upload','POST','','',''
WHERE NOT EXISTS (
  SELECT 1 FROM `casbin_rule` WHERE `ptype`='p' AND `v0`='500' AND `v1`='/qqCache/upload' AND `v2`='POST'
);

INSERT INTO `casbin_rule` (`ptype`,`v0`,`v1`,`v2`,`v3`,`v4`,`v5`)
SELECT 'p','400','/qqCache/roleHint','GET','','',''
WHERE NOT EXISTS (
  SELECT 1 FROM `casbin_rule` WHERE `ptype`='p' AND `v0`='400' AND `v1`='/qqCache/roleHint' AND `v2`='GET'
);

INSERT INTO `casbin_rule` (`ptype`,`v0`,`v1`,`v2`,`v3`,`v4`,`v5`)
SELECT 'p','500','/qqCache/roleHint','GET','','',''
WHERE NOT EXISTS (
  SELECT 1 FROM `casbin_rule` WHERE `ptype`='p' AND `v0`='500' AND `v1`='/qqCache/roleHint' AND `v2`='GET'
);

COMMIT;
