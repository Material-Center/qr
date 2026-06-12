-- 地推用户级 OpenAPI：创建收码任务、查询任务、提交验证码。
-- OpenAPI 走用户级 token 自校验，不走后台登录态 casbin；这里只登记 API 列表并清理误加的 casbin。

INSERT INTO `sys_apis` (`created_at`, `updated_at`, `api_group`, `method`, `path`, `description`)
SELECT NOW(), NOW(), '手机号注册任务', 'POST', '/phoneRegisterTask/open-api/promoter/receive-task', '地推OpenAPI创建收码手机号注册任务'
WHERE NOT EXISTS (
  SELECT 1 FROM `sys_apis`
  WHERE `path` = '/phoneRegisterTask/open-api/promoter/receive-task'
    AND `method` = 'POST'
    AND `deleted_at` IS NULL
);

INSERT INTO `sys_apis` (`created_at`, `updated_at`, `api_group`, `method`, `path`, `description`)
SELECT NOW(), NOW(), '手机号注册任务', 'GET', '/phoneRegisterTask/open-api/promoter/task/:taskId', '地推OpenAPI查询手机号注册任务'
WHERE NOT EXISTS (
  SELECT 1 FROM `sys_apis`
  WHERE `path` = '/phoneRegisterTask/open-api/promoter/task/:taskId'
    AND `method` = 'GET'
    AND `deleted_at` IS NULL
);

INSERT INTO `sys_apis` (`created_at`, `updated_at`, `api_group`, `method`, `path`, `description`)
SELECT NOW(), NOW(), '手机号注册任务', 'POST', '/phoneRegisterTask/open-api/promoter/submit-code', '地推OpenAPI提交手机号注册验证码'
WHERE NOT EXISTS (
  SELECT 1 FROM `sys_apis`
  WHERE `path` = '/phoneRegisterTask/open-api/promoter/submit-code'
    AND `method` = 'POST'
    AND `deleted_at` IS NULL
);

DELETE FROM `casbin_rule`
WHERE `ptype` = 'p'
  AND `v1` IN (
    '/phoneRegisterTask/open-api/promoter/receive-task',
    '/phoneRegisterTask/open-api/promoter/task/:taskId',
    '/phoneRegisterTask/open-api/promoter/submit-code'
  );
