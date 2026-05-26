-- QQ缓存管理：导入缓存包 API 权限补丁（幂等）

INSERT INTO `sys_apis` (`created_at`, `updated_at`, `api_group`, `method`, `path`, `description`)
SELECT NOW(), NOW(), 'QQ缓存', 'POST', '/qqCache/importZip', '管理端导入QQ缓存zip'
WHERE NOT EXISTS (
  SELECT 1 FROM `sys_apis`
  WHERE `path` = '/qqCache/importZip'
    AND `method` = 'POST'
    AND `deleted_at` IS NULL
);

INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`)
SELECT 'p', '888', '/qqCache/importZip', 'POST', '', '', ''
WHERE NOT EXISTS (
  SELECT 1 FROM `casbin_rule`
  WHERE `ptype` = 'p'
    AND `v0` = '888'
    AND `v1` = '/qqCache/importZip'
    AND `v2` = 'POST'
);

INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`)
SELECT 'p', '100', '/qqCache/importZip', 'POST', '', '', ''
WHERE NOT EXISTS (
  SELECT 1 FROM `casbin_rule`
  WHERE `ptype` = 'p'
    AND `v0` = '100'
    AND `v1` = '/qqCache/importZip'
    AND `v2` = 'POST'
);
