-- 注册配置迁移 + 登录调试菜单/API/权限补丁（幂等）
-- 适用：MySQL / MariaDB

START TRANSACTION;

-- 1) 菜单：注册任务 -> 登录调试（管理员）
INSERT INTO `sys_base_menus`
(`created_at`,`updated_at`,`menu_level`,`parent_id`,`path`,`name`,`hidden`,`component`,`sort`,`active_name`,`keep_alive`,`default_menu`,`title`,`icon`,`close_tab`,`transition_type`)
SELECT NOW(), NOW(), 1, p.id, 'debug-login', 'registerDebugLogin', 0, 'view/register/debugLogin.vue', 4, '', 0, 0, '登录调试', 'cpu', 0, ''
FROM `sys_base_menus` p
WHERE p.`name` = 'register'
  AND p.`deleted_at` IS NULL
  AND NOT EXISTS (
    SELECT 1 FROM `sys_base_menus` m
    WHERE m.`name` = 'registerDebugLogin'
      AND m.`deleted_at` IS NULL
  );

-- 2) API 元数据
INSERT INTO `sys_apis` (`created_at`,`updated_at`,`api_group`,`method`,`path`,`description`)
SELECT NOW(), NOW(), '注册任务', 'POST', '/registerTask/debug/login/start', '管理员启动登录调试'
WHERE NOT EXISTS (
  SELECT 1 FROM `sys_apis` WHERE `path`='/registerTask/debug/login/start' AND `method`='POST' AND `deleted_at` IS NULL
);

INSERT INTO `sys_apis` (`created_at`,`updated_at`,`api_group`,`method`,`path`,`description`)
SELECT NOW(), NOW(), '注册任务', 'POST', '/registerTask/debug/login/submit', '管理员提交调试登录验证码'
WHERE NOT EXISTS (
  SELECT 1 FROM `sys_apis` WHERE `path`='/registerTask/debug/login/submit' AND `method`='POST' AND `deleted_at` IS NULL
);

INSERT INTO `sys_apis` (`created_at`,`updated_at`,`api_group`,`method`,`path`,`description`)
SELECT NOW(), NOW(), '注册任务', 'GET', '/registerTask/debug/login/task', '管理员查询调试登录任务'
WHERE NOT EXISTS (
  SELECT 1 FROM `sys_apis` WHERE `path`='/registerTask/debug/login/task' AND `method`='GET' AND `deleted_at` IS NULL
);

INSERT INTO `sys_apis` (`created_at`,`updated_at`,`api_group`,`method`,`path`,`description`)
SELECT NOW(), NOW(), '注册配置', 'GET', '/registerConfig/checkMyConfig', '检测我的注册配置'
WHERE NOT EXISTS (
  SELECT 1 FROM `sys_apis` WHERE `path`='/registerConfig/checkMyConfig' AND `method`='GET' AND `deleted_at` IS NULL
);

-- 3) Casbin 权限：超级管理员(888) + 管理员(100)
INSERT INTO `casbin_rule` (`ptype`,`v0`,`v1`,`v2`,`v3`,`v4`,`v5`)
SELECT 'p','888','/registerTask/debug/login/start','POST','','',''
WHERE NOT EXISTS (
  SELECT 1 FROM `casbin_rule` WHERE `ptype`='p' AND `v0`='888' AND `v1`='/registerTask/debug/login/start' AND `v2`='POST'
);
INSERT INTO `casbin_rule` (`ptype`,`v0`,`v1`,`v2`,`v3`,`v4`,`v5`)
SELECT 'p','888','/registerTask/debug/login/submit','POST','','',''
WHERE NOT EXISTS (
  SELECT 1 FROM `casbin_rule` WHERE `ptype`='p' AND `v0`='888' AND `v1`='/registerTask/debug/login/submit' AND `v2`='POST'
);
INSERT INTO `casbin_rule` (`ptype`,`v0`,`v1`,`v2`,`v3`,`v4`,`v5`)
SELECT 'p','888','/registerTask/debug/login/task','GET','','',''
WHERE NOT EXISTS (
  SELECT 1 FROM `casbin_rule` WHERE `ptype`='p' AND `v0`='888' AND `v1`='/registerTask/debug/login/task' AND `v2`='GET'
);
INSERT INTO `casbin_rule` (`ptype`,`v0`,`v1`,`v2`,`v3`,`v4`,`v5`)
SELECT 'p','888','/registerConfig/checkMyConfig','GET','','',''
WHERE NOT EXISTS (
  SELECT 1 FROM `casbin_rule` WHERE `ptype`='p' AND `v0`='888' AND `v1`='/registerConfig/checkMyConfig' AND `v2`='GET'
);

INSERT INTO `casbin_rule` (`ptype`,`v0`,`v1`,`v2`,`v3`,`v4`,`v5`)
SELECT 'p','100','/registerTask/debug/login/start','POST','','',''
WHERE NOT EXISTS (
  SELECT 1 FROM `casbin_rule` WHERE `ptype`='p' AND `v0`='100' AND `v1`='/registerTask/debug/login/start' AND `v2`='POST'
);
INSERT INTO `casbin_rule` (`ptype`,`v0`,`v1`,`v2`,`v3`,`v4`,`v5`)
SELECT 'p','100','/registerTask/debug/login/submit','POST','','',''
WHERE NOT EXISTS (
  SELECT 1 FROM `casbin_rule` WHERE `ptype`='p' AND `v0`='100' AND `v1`='/registerTask/debug/login/submit' AND `v2`='POST'
);
INSERT INTO `casbin_rule` (`ptype`,`v0`,`v1`,`v2`,`v3`,`v4`,`v5`)
SELECT 'p','100','/registerTask/debug/login/task','GET','','',''
WHERE NOT EXISTS (
  SELECT 1 FROM `casbin_rule` WHERE `ptype`='p' AND `v0`='100' AND `v1`='/registerTask/debug/login/task' AND `v2`='GET'
);
INSERT INTO `casbin_rule` (`ptype`,`v0`,`v1`,`v2`,`v3`,`v4`,`v5`)
SELECT 'p','100','/registerConfig/checkMyConfig','GET','','',''
WHERE NOT EXISTS (
  SELECT 1 FROM `casbin_rule` WHERE `ptype`='p' AND `v0`='100' AND `v1`='/registerConfig/checkMyConfig' AND `v2`='GET'
);

-- 团长不再允许保存配置
DELETE FROM `casbin_rule`
WHERE `ptype`='p' AND `v0`='200' AND `v1`='/registerConfig/setMyConfig' AND `v2`='PUT';

-- 4) 菜单与角色绑定：给 888/100 增加 登录调试；团长(200)移除配置管理菜单
INSERT INTO `sys_authority_menus` (`sys_authority_authority_id`,`sys_base_menu_id`)
SELECT '888', CAST(m.id AS CHAR)
FROM `sys_base_menus` m
WHERE m.`name`='registerDebugLogin' AND m.`deleted_at` IS NULL
  AND NOT EXISTS (
    SELECT 1 FROM `sys_authority_menus` am
    WHERE am.`sys_authority_authority_id`='888'
      AND am.`sys_base_menu_id`=CAST(m.id AS CHAR)
  );

INSERT INTO `sys_authority_menus` (`sys_authority_authority_id`,`sys_base_menu_id`)
SELECT '100', CAST(m.id AS CHAR)
FROM `sys_base_menus` m
WHERE m.`name`='registerDebugLogin' AND m.`deleted_at` IS NULL
  AND NOT EXISTS (
    SELECT 1 FROM `sys_authority_menus` am
    WHERE am.`sys_authority_authority_id`='100'
      AND am.`sys_base_menu_id`=CAST(m.id AS CHAR)
  );

DELETE am FROM `sys_authority_menus` am
JOIN `sys_base_menus` m ON am.`sys_base_menu_id` = CAST(m.id AS CHAR)
WHERE am.`sys_authority_authority_id`='200'
  AND m.`name`='registerConfig'
  AND m.`deleted_at` IS NULL;

COMMIT;
