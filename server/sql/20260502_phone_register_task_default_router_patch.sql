-- 手机号注册任务默认首页补丁（幂等）
-- 适用：MySQL / MariaDB
-- 说明：将地推角色默认首页切换到本次新增的本地手机号注册页面

START TRANSACTION;

UPDATE `sys_authorities`
SET
  `default_router` = 'phoneRegisterTaskCenter',
  `updated_at` = NOW()
WHERE `authority_id` = 300
  AND `deleted_at` IS NULL
  AND (`default_router` IS NULL OR `default_router` <> 'phoneRegisterTaskCenter');

COMMIT;
