-- QQ缓存管理：导出账号列表(txt) API 数据补丁（幂等）

INSERT INTO `sys_apis` (`created_at`, `updated_at`, `api_group`, `method`, `path`, `description`)
SELECT NOW(), NOW(), 'QQ缓存', 'POST', '/qqCache/exportAccountList', '管理端导出QQ账号列表(txt)'
WHERE NOT EXISTS (
  SELECT 1 FROM `sys_apis`
  WHERE `path` = '/qqCache/exportAccountList'
    AND `method` = 'POST'
    AND `deleted_at` IS NULL
);

INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`)
SELECT 'p', '888', '/qqCache/exportAccountList', 'POST', '', '', ''
WHERE NOT EXISTS (
  SELECT 1 FROM `casbin_rule`
  WHERE `ptype` = 'p'
    AND `v0` = '888'
    AND `v1` = '/qqCache/exportAccountList'
    AND `v2` = 'POST'
);

INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`)
SELECT 'p', '100', '/qqCache/exportAccountList', 'POST', '', '', ''
WHERE NOT EXISTS (
  SELECT 1 FROM `casbin_rule`
  WHERE `ptype` = 'p'
    AND `v0` = '100'
    AND `v1` = '/qqCache/exportAccountList'
    AND `v2` = 'POST'
);
