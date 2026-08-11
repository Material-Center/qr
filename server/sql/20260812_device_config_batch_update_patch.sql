-- 设备管理批量设置接口权限补丁（幂等）
-- 适用：MySQL / MariaDB

START TRANSACTION;

INSERT INTO `sys_apis` (`created_at`, `updated_at`, `api_group`, `method`, `path`, `description`)
SELECT NOW(), NOW(), '设备管理', 'POST', '/deviceConfig/batchUpdate', '批量设置设备配置'
WHERE NOT EXISTS (
  SELECT 1 FROM `sys_apis`
  WHERE `path` = '/deviceConfig/batchUpdate'
    AND `method` = 'POST'
    AND `deleted_at` IS NULL
);

INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`)
SELECT 'p', roles.`role`, '/deviceConfig/batchUpdate', 'POST', '', '', ''
FROM (
  SELECT '888' AS `role`
  UNION ALL SELECT '100'
) roles
WHERE NOT EXISTS (
  SELECT 1 FROM `casbin_rule` cr
  WHERE cr.`ptype` = 'p'
    AND cr.`v0` = roles.`role`
    AND cr.`v1` = '/deviceConfig/batchUpdate'
    AND cr.`v2` = 'POST'
);

COMMIT;
