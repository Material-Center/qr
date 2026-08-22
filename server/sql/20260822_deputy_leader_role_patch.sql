-- 副团长角色（210）：仅账号管理，幂等补丁
START TRANSACTION;

INSERT INTO `sys_authorities` (`authority_id`,`authority_name`,`parent_id`,`default_router`,`created_at`,`updated_at`)
SELECT 210, '副团长', 200, 'accountManage', NOW(), NOW()
WHERE NOT EXISTS (
  SELECT 1 FROM `sys_authorities` WHERE `authority_id` = 210 AND `deleted_at` IS NULL
);

INSERT INTO `sys_authority_menus` (`sys_authority_authority_id`,`sys_base_menu_id`)
SELECT '210', CAST(m.id AS CHAR)
FROM `sys_base_menus` m
WHERE m.`name` IN ('account', 'accountManage') AND m.`deleted_at` IS NULL
  AND NOT EXISTS (
    SELECT 1 FROM `sys_authority_menus` am
    WHERE am.`sys_authority_authority_id` = '210'
      AND am.`sys_base_menu_id` = CAST(m.id AS CHAR)
  );

INSERT INTO `sys_data_authority_id` (`sys_authority_authority_id`,`data_authority_id_authority_id`)
SELECT '210', '300'
WHERE NOT EXISTS (
  SELECT 1 FROM `sys_data_authority_id`
  WHERE `sys_authority_authority_id` = '210' AND `data_authority_id_authority_id` = '300'
);

INSERT INTO `casbin_rule` (`ptype`,`v0`,`v1`,`v2`)
SELECT 'p', '210', p.path, p.method
FROM (
  SELECT '/menu/getMenu' path, 'POST' method UNION ALL
  SELECT '/menu/getMenuList', 'POST' UNION ALL
  SELECT '/menu/getBaseMenuTree', 'POST' UNION ALL
  SELECT '/user/getUserInfo', 'GET' UNION ALL
  SELECT '/user/changePassword', 'POST' UNION ALL
  SELECT '/user/setSelfInfo', 'PUT' UNION ALL
  SELECT '/user/setSelfSetting', 'PUT' UNION ALL
  SELECT '/user/admin_register', 'POST' UNION ALL
  SELECT '/user/getUserList', 'POST' UNION ALL
  SELECT '/user/setUserInfo', 'PUT' UNION ALL
  SELECT '/user/deleteUser', 'DELETE' UNION ALL
  SELECT '/user/resetPassword', 'POST' UNION ALL
  SELECT '/user/setUserAuthorities', 'POST' UNION ALL
  SELECT '/jwt/jsonInBlacklist', 'POST'
) p
WHERE NOT EXISTS (
  SELECT 1 FROM `casbin_rule` c
  WHERE c.`ptype` = 'p' AND c.`v0` = '210' AND c.`v1` = p.path AND c.`v2` = p.method
);

COMMIT;
