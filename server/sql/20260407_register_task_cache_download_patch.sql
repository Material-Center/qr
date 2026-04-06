-- 注册任务缓存下载 API 权限补丁（幂等）
-- 适用：MySQL / MariaDB

START TRANSACTION;

-- 1) API 元数据
INSERT INTO `sys_apis` (`created_at`,`updated_at`,`api_group`,`method`,`path`,`description`)
SELECT NOW(), NOW(), '注册任务', 'GET', '/registerTask/cache/download', '下载任务登录缓存INI'
WHERE NOT EXISTS (
  SELECT 1 FROM `sys_apis` WHERE `path`='/registerTask/cache/download' AND `method`='GET' AND `deleted_at` IS NULL
);

-- 2) Casbin 权限：超级管理员(888) + 管理员(100)
INSERT INTO `casbin_rule` (`ptype`,`v0`,`v1`,`v2`,`v3`,`v4`,`v5`)
SELECT 'p','888','/registerTask/cache/download','GET','','',''
WHERE NOT EXISTS (
  SELECT 1 FROM `casbin_rule` WHERE `ptype`='p' AND `v0`='888' AND `v1`='/registerTask/cache/download' AND `v2`='GET'
);

INSERT INTO `casbin_rule` (`ptype`,`v0`,`v1`,`v2`,`v3`,`v4`,`v5`)
SELECT 'p','100','/registerTask/cache/download','GET','','',''
WHERE NOT EXISTS (
  SELECT 1 FROM `casbin_rule` WHERE `ptype`='p' AND `v0`='100' AND `v1`='/registerTask/cache/download' AND `v2`='GET'
);

COMMIT;
