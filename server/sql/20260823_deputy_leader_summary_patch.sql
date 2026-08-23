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
  SELECT '/registerTask/summary' path, 'GET' method UNION ALL
  SELECT '/phoneRegisterTask/summary', 'GET'
) p
WHERE NOT EXISTS (
  SELECT 1 FROM `casbin_rule` c
  WHERE c.`ptype` = 'p' AND c.`v0` = '210' AND c.`v1` = p.path AND c.`v2` = p.method
);

DELETE FROM `casbin_rule`
WHERE `ptype` = 'p' AND `v0` = '210'
  AND ((`v1` = '/registerTask/list' AND `v2` = 'POST')
    OR (`v1` = '/phoneRegisterTask/list' AND `v2` = 'POST'));

COMMIT;
