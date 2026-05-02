-- 手机号注册统计看板菜单补丁（幂等）
-- 适用：MySQL / MariaDB
-- 说明：为本次本地手机号注册任务交付补齐管理员/团长注册统计菜单入口，并统一旧统计菜单命名

START TRANSACTION;

UPDATE `sys_base_menus`
SET `title` = '反扫统计',
    `updated_at` = NOW()
WHERE `name` = 'registerTaskManage'
  AND `deleted_at` IS NULL;

UPDATE `sys_base_menus`
SET `title` = '注册统计',
    `updated_at` = NOW()
WHERE `name` = 'phoneRegisterTaskManage'
  AND `deleted_at` IS NULL;

INSERT INTO `sys_base_menus`
(`created_at`,`updated_at`,`menu_level`,`parent_id`,`path`,`name`,`hidden`,`component`,`sort`,`active_name`,`keep_alive`,`default_menu`,`title`,`icon`,`close_tab`,`transition_type`)
SELECT NOW(), NOW(), 1, p.id, 'phone-manage', 'phoneRegisterTaskManage', 0, 'view/register/phoneTaskManage.vue', 2, '', 0, 0, '注册统计', 'data-analysis', 0, ''
FROM `sys_base_menus` p
WHERE p.`name` = 'register'
  AND p.`deleted_at` IS NULL
  AND NOT EXISTS (
    SELECT 1 FROM `sys_base_menus` m
    WHERE m.`name` = 'phoneRegisterTaskManage'
      AND m.`deleted_at` IS NULL
  );

INSERT INTO `sys_authority_menus` (`sys_authority_authority_id`,`sys_base_menu_id`)
SELECT '888', CAST(m.id AS CHAR)
FROM `sys_base_menus` m
WHERE m.`name`='phoneRegisterTaskManage' AND m.`deleted_at` IS NULL
  AND NOT EXISTS (
    SELECT 1 FROM `sys_authority_menus` am
    WHERE am.`sys_authority_authority_id`='888'
      AND am.`sys_base_menu_id`=CAST(m.id AS CHAR)
  );

INSERT INTO `sys_authority_menus` (`sys_authority_authority_id`,`sys_base_menu_id`)
SELECT '100', CAST(m.id AS CHAR)
FROM `sys_base_menus` m
WHERE m.`name`='phoneRegisterTaskManage' AND m.`deleted_at` IS NULL
  AND NOT EXISTS (
    SELECT 1 FROM `sys_authority_menus` am
    WHERE am.`sys_authority_authority_id`='100'
      AND am.`sys_base_menu_id`=CAST(m.id AS CHAR)
  );

INSERT INTO `sys_authority_menus` (`sys_authority_authority_id`,`sys_base_menu_id`)
SELECT '200', CAST(m.id AS CHAR)
FROM `sys_base_menus` m
WHERE m.`name`='phoneRegisterTaskManage' AND m.`deleted_at` IS NULL
  AND NOT EXISTS (
    SELECT 1 FROM `sys_authority_menus` am
    WHERE am.`sys_authority_authority_id`='200'
      AND am.`sys_base_menu_id`=CAST(m.id AS CHAR)
  );

COMMIT;
