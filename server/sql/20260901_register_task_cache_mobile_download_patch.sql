-- 移动端注册任务缓存下载凭证 API 权限补丁（幂等）
-- 适用：MySQL / MariaDB

START TRANSACTION;

INSERT INTO `sys_apis` (`created_at`,`updated_at`,`api_group`,`method`,`path`,`description`)
SELECT NOW(), NOW(), '注册任务', 'POST', '/registerTask/cache/prepare', '签发任务缓存下载凭证'
WHERE NOT EXISTS (
  SELECT 1 FROM `sys_apis` WHERE `path`='/registerTask/cache/prepare' AND `method`='POST' AND `deleted_at` IS NULL
);

INSERT INTO `casbin_rule` (`ptype`,`v0`,`v1`,`v2`,`v3`,`v4`,`v5`)
SELECT 'p','888','/registerTask/cache/prepare','POST','','',''
WHERE NOT EXISTS (
  SELECT 1 FROM `casbin_rule` WHERE `ptype`='p' AND `v0`='888' AND `v1`='/registerTask/cache/prepare' AND `v2`='POST'
);

INSERT INTO `casbin_rule` (`ptype`,`v0`,`v1`,`v2`,`v3`,`v4`,`v5`)
SELECT 'p','100','/registerTask/cache/prepare','POST','','',''
WHERE NOT EXISTS (
  SELECT 1 FROM `casbin_rule` WHERE `ptype`='p' AND `v0`='100' AND `v1`='/registerTask/cache/prepare' AND `v2`='POST'
);

COMMIT;
