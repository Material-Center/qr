-- QQ缓存管理：按数量提取未提取缓存INI(zip) API 数据补丁
-- 说明：只补 sys_apis 数据；casbin 权限请在管理后台手动添加。

INSERT INTO `sys_apis` (`created_at`,`updated_at`,`api_group`,`method`,`path`,`description`)
SELECT NOW(), NOW(), 'QQ缓存', 'POST', '/qqCache/exportPendingIniZip', '管理端按数量提取未提取缓存INI(zip)'
WHERE NOT EXISTS (
  SELECT 1 FROM `sys_apis`
  WHERE `path` = '/qqCache/exportPendingIniZip'
    AND `method` = 'POST'
    AND `deleted_at` IS NULL
);
