-- 为 registerTask 新增 actives 接口的线上补丁（幂等）
-- 适用：MySQL / MariaDB

START TRANSACTION;

-- 1) API 元数据（用于接口管理页）
INSERT INTO `sys_apis` (`api_group`, `method`, `path`, `description`, `created_at`, `updated_at`)
SELECT '注册任务', 'GET', '/registerTask/actives', '获取地推全部未完成任务', NOW(), NOW()
WHERE NOT EXISTS (
  SELECT 1
  FROM `sys_apis`
  WHERE `path` = '/registerTask/actives'
    AND `method` = 'GET'
    AND `deleted_at` IS NULL
);

-- 2) Casbin 权限：超级管理员（888）
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`)
SELECT 'p', '888', '/registerTask/actives', 'GET', '', '', ''
WHERE NOT EXISTS (
  SELECT 1
  FROM `casbin_rule`
  WHERE `ptype` = 'p'
    AND `v0` = '888'
    AND `v1` = '/registerTask/actives'
    AND `v2` = 'GET'
);

-- 3) Casbin 权限：地推（300）
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`)
SELECT 'p', '300', '/registerTask/actives', 'GET', '', '', ''
WHERE NOT EXISTS (
  SELECT 1
  FROM `casbin_rule`
  WHERE `ptype` = 'p'
    AND `v0` = '300'
    AND `v1` = '/registerTask/actives'
    AND `v2` = 'GET'
);

COMMIT;
