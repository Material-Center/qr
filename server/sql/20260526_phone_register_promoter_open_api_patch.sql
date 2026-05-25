-- 地推用户级 OpenAPI：查询可用设备、创建手机号注册任务。
-- OpenAPI 走用户级 token 自校验，不走后台登录态 casbin；这里只登记 API 列表。

INSERT INTO `sys_apis` (`created_at`, `updated_at`, `api_group`, `method`, `path`, `description`)
SELECT NOW(), NOW(), '手机号注册任务', 'GET', '/phoneRegisterTask/open-api/promoter/device-stats', '地推OpenAPI查询可用设备'
WHERE NOT EXISTS (
  SELECT 1 FROM `sys_apis`
  WHERE `path` = '/phoneRegisterTask/open-api/promoter/device-stats'
    AND `method` = 'GET'
    AND `deleted_at` IS NULL
);

INSERT INTO `sys_apis` (`created_at`, `updated_at`, `api_group`, `method`, `path`, `description`)
SELECT NOW(), NOW(), '手机号注册任务', 'POST', '/phoneRegisterTask/open-api/promoter/task', '地推OpenAPI创建手机号注册任务'
WHERE NOT EXISTS (
  SELECT 1 FROM `sys_apis`
  WHERE `path` = '/phoneRegisterTask/open-api/promoter/task'
    AND `method` = 'POST'
    AND `deleted_at` IS NULL
);

DELETE FROM `casbin_rule`
WHERE `ptype` = 'p'
  AND `v1` IN (
    '/phoneRegisterTask/open-api/promoter/device-stats',
    '/phoneRegisterTask/open-api/promoter/task'
  );

-- 管理员允许签发/查看/作废用户级 API Token。
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`)
SELECT 'p', '100', '/sysApiToken/createApiToken', 'POST', '', '', ''
WHERE NOT EXISTS (
  SELECT 1 FROM `casbin_rule`
  WHERE `ptype` = 'p'
    AND `v0` = '100'
    AND `v1` = '/sysApiToken/createApiToken'
    AND `v2` = 'POST'
);

INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`)
SELECT 'p', '100', '/sysApiToken/getApiTokenList', 'POST', '', '', ''
WHERE NOT EXISTS (
  SELECT 1 FROM `casbin_rule`
  WHERE `ptype` = 'p'
    AND `v0` = '100'
    AND `v1` = '/sysApiToken/getApiTokenList'
    AND `v2` = 'POST'
);

INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`)
SELECT 'p', '100', '/sysApiToken/deleteApiToken', 'POST', '', '', ''
WHERE NOT EXISTS (
  SELECT 1 FROM `casbin_rule`
  WHERE `ptype` = 'p'
    AND `v0` = '100'
    AND `v1` = '/sysApiToken/deleteApiToken'
    AND `v2` = 'POST'
);

-- API Token 不单独挂系统工具菜单；管理员在用户管理里对地推账号生成。
DELETE am
FROM `sys_authority_menus` am
JOIN `sys_base_menus` m ON m.`id` = am.`sys_base_menu_id`
WHERE am.`sys_authority_authority_id` = '100'
  AND m.`name` = 'apiToken';
