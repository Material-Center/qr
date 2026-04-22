-- QQ 缓存：管理端批量导出 INI（zip）接口 + Casbin（幂等）
-- 适用：MySQL / MariaDB

START TRANSACTION;

INSERT INTO `sys_apis` (`created_at`,`updated_at`,`api_group`,`method`,`path`,`description`)
SELECT NOW(), NOW(), 'QQ缓存', 'POST', '/qqCache/exportIniZip', '管理端批量导出缓存INI(zip)'
WHERE NOT EXISTS (
  SELECT 1 FROM `sys_apis` WHERE `path`='/qqCache/exportIniZip' AND `method`='POST' AND `deleted_at` IS NULL
);

INSERT INTO `casbin_rule` (`ptype`,`v0`,`v1`,`v2`,`v3`,`v4`,`v5`)
SELECT 'p','888','/qqCache/exportIniZip','POST','','',''
WHERE NOT EXISTS (
  SELECT 1 FROM `casbin_rule` WHERE `ptype`='p' AND `v0`='888' AND `v1`='/qqCache/exportIniZip' AND `v2`='POST'
);

INSERT INTO `casbin_rule` (`ptype`,`v0`,`v1`,`v2`,`v3`,`v4`,`v5`)
SELECT 'p','100','/qqCache/exportIniZip','POST','','',''
WHERE NOT EXISTS (
  SELECT 1 FROM `casbin_rule` WHERE `ptype`='p' AND `v0`='100' AND `v1`='/qqCache/exportIniZip' AND `v2`='POST'
);

COMMIT;
