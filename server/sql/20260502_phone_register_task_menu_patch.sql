-- 手机号注册任务页面菜单补丁（幂等）
-- 适用：MySQL / MariaDB

START TRANSACTION;

INSERT INTO `sys_base_menus`
(`created_at`,`updated_at`,`menu_level`,`parent_id`,`path`,`name`,`hidden`,`component`,`sort`,`active_name`,`keep_alive`,`default_menu`,`title`,`icon`,`close_tab`,`transition_type`)
SELECT NOW(), NOW(), 0, 0, 'phone-register-task-center', 'phoneRegisterTaskCenter', 1, 'view/register/phoneTaskCenter.vue', 0, '', 0, 1, '手机号注册任务', 'iphone', 0, ''
WHERE NOT EXISTS (
  SELECT 1 FROM `sys_base_menus` m
  WHERE m.`name` = 'phoneRegisterTaskCenter'
    AND m.`deleted_at` IS NULL
);

INSERT INTO `sys_authority_menus` (`sys_authority_authority_id`,`sys_base_menu_id`)
SELECT '300', CAST(m.id AS CHAR)
FROM `sys_base_menus` m
WHERE m.`name`='phoneRegisterTaskCenter' AND m.`deleted_at` IS NULL
  AND NOT EXISTS (
    SELECT 1 FROM `sys_authority_menus` am
    WHERE am.`sys_authority_authority_id`='300'
      AND am.`sys_base_menu_id`=CAST(m.id AS CHAR)
  );

COMMIT;
