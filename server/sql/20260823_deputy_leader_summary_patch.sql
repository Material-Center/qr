-- 副团长开放注册统计和反扫统计，只读查看本人创建的地推汇总。
START TRANSACTION;

INSERT INTO `sys_authority_menus` (`sys_authority_authority_id`,`sys_base_menu_id`)
SELECT '210', CAST(m.id AS CHAR)
FROM `sys_base_menus` m
WHERE m.`name` IN ('register', 'registerTaskManage', 'phoneRegisterTaskManage')
  AND m.`deleted_at` IS NULL
  AND NOT EXISTS (
    SELECT 1 FROM `sys_authority_menus` am
    WHERE am.`sys_authority_authority_id` = '210'
      AND am.`sys_base_menu_id` = CAST(m.id AS CHAR)
  );

INSERT INTO `casbin_rule` (`ptype`,`v0`,`v1`,`v2`)
SELECT 'p', '210', p.path, p.method
FROM (
  SELECT '/registerTask/list' path, 'POST' method UNION ALL
  SELECT '/registerTask/summary', 'GET' UNION ALL
  SELECT '/phoneRegisterTask/list', 'POST' UNION ALL
  SELECT '/phoneRegisterTask/summary', 'GET'
) p
WHERE NOT EXISTS (
  SELECT 1 FROM `casbin_rule` c
  WHERE c.`ptype` = 'p' AND c.`v0` = '210' AND c.`v1` = p.path AND c.`v2` = p.method
);

COMMIT;
