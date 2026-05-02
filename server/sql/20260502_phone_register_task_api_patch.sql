-- 手机号注册任务 API 元数据与权限补丁（幂等）
-- 适用：MySQL / MariaDB
-- 说明：用于已执行过结构脚本的本地库补齐 sys_apis / casbin_rule

START TRANSACTION;

UPDATE `sys_apis`
SET `description` = '获取反扫统计',
    `updated_at` = NOW()
WHERE `path`='/registerTask/summary'
  AND `method`='GET'
  AND `deleted_at` IS NULL;

UPDATE `sys_apis`
SET `description` = '获取注册统计',
    `updated_at` = NOW()
WHERE `path`='/phoneRegisterTask/summary'
  AND `method`='GET'
  AND `deleted_at` IS NULL;

INSERT INTO `sys_apis` (`created_at`,`updated_at`,`api_group`,`method`,`path`,`description`)
SELECT NOW(), NOW(), 'QQ缓存', 'POST', '/qqCache/uploadPhoneRegister', '手机号注册上传QQ缓存并完成任务'
WHERE NOT EXISTS (
  SELECT 1 FROM `sys_apis` WHERE `path`='/qqCache/uploadPhoneRegister' AND `method`='POST' AND `deleted_at` IS NULL
);

INSERT INTO `sys_apis` (`created_at`,`updated_at`,`api_group`,`method`,`path`,`description`)
SELECT NOW(), NOW(), '手机号注册任务', 'POST', '/phoneRegisterTask/create', '地推创建手机号注册任务'
WHERE NOT EXISTS (
  SELECT 1 FROM `sys_apis` WHERE `path`='/phoneRegisterTask/create' AND `method`='POST' AND `deleted_at` IS NULL
);

INSERT INTO `sys_apis` (`created_at`,`updated_at`,`api_group`,`method`,`path`,`description`)
SELECT NOW(), NOW(), '手机号注册任务', 'POST', '/phoneRegisterTask/submitCode', '地推提交手机号注册验证码'
WHERE NOT EXISTS (
  SELECT 1 FROM `sys_apis` WHERE `path`='/phoneRegisterTask/submitCode' AND `method`='POST' AND `deleted_at` IS NULL
);

INSERT INTO `sys_apis` (`created_at`,`updated_at`,`api_group`,`method`,`path`,`description`)
SELECT NOW(), NOW(), '手机号注册任务', 'GET', '/phoneRegisterTask/active', '获取地推当前手机号注册任务'
WHERE NOT EXISTS (
  SELECT 1 FROM `sys_apis` WHERE `path`='/phoneRegisterTask/active' AND `method`='GET' AND `deleted_at` IS NULL
);

INSERT INTO `sys_apis` (`created_at`,`updated_at`,`api_group`,`method`,`path`,`description`)
SELECT NOW(), NOW(), '手机号注册任务', 'GET', '/phoneRegisterTask/actives', '获取地推全部手机号注册任务'
WHERE NOT EXISTS (
  SELECT 1 FROM `sys_apis` WHERE `path`='/phoneRegisterTask/actives' AND `method`='GET' AND `deleted_at` IS NULL
);

INSERT INTO `sys_apis` (`created_at`,`updated_at`,`api_group`,`method`,`path`,`description`)
SELECT NOW(), NOW(), '手机号注册任务', 'POST', '/phoneRegisterTask/list', '分页查询手机号注册任务'
WHERE NOT EXISTS (
  SELECT 1 FROM `sys_apis` WHERE `path`='/phoneRegisterTask/list' AND `method`='POST' AND `deleted_at` IS NULL
);

INSERT INTO `sys_apis` (`created_at`,`updated_at`,`api_group`,`method`,`path`,`description`)
SELECT NOW(), NOW(), '手机号注册任务', 'GET', '/phoneRegisterTask/summary', '获取注册统计'
WHERE NOT EXISTS (
  SELECT 1 FROM `sys_apis` WHERE `path`='/phoneRegisterTask/summary' AND `method`='GET' AND `deleted_at` IS NULL
);

INSERT INTO `sys_apis` (`created_at`,`updated_at`,`api_group`,`method`,`path`,`description`)
SELECT NOW(), NOW(), '手机号注册任务', 'POST', '/phoneRegisterTask/device/poll', '设备拉取手机号注册任务'
WHERE NOT EXISTS (
  SELECT 1 FROM `sys_apis` WHERE `path`='/phoneRegisterTask/device/poll' AND `method`='POST' AND `deleted_at` IS NULL
);

INSERT INTO `sys_apis` (`created_at`,`updated_at`,`api_group`,`method`,`path`,`description`)
SELECT NOW(), NOW(), '手机号注册任务', 'GET', '/phoneRegisterTask/device/task', '设备查询当前手机号注册任务'
WHERE NOT EXISTS (
  SELECT 1 FROM `sys_apis` WHERE `path`='/phoneRegisterTask/device/task' AND `method`='GET' AND `deleted_at` IS NULL
);

INSERT INTO `sys_apis` (`created_at`,`updated_at`,`api_group`,`method`,`path`,`description`)
SELECT NOW(), NOW(), '手机号注册任务', 'POST', '/phoneRegisterTask/device/task', '设备查询当前手机号注册任务'
WHERE NOT EXISTS (
  SELECT 1 FROM `sys_apis` WHERE `path`='/phoneRegisterTask/device/task' AND `method`='POST' AND `deleted_at` IS NULL
);

INSERT INTO `sys_apis` (`created_at`,`updated_at`,`api_group`,`method`,`path`,`description`)
SELECT NOW(), NOW(), '手机号注册任务', 'POST', '/phoneRegisterTask/device/heartbeat', '设备上报手机号注册任务心跳'
WHERE NOT EXISTS (
  SELECT 1 FROM `sys_apis` WHERE `path`='/phoneRegisterTask/device/heartbeat' AND `method`='POST' AND `deleted_at` IS NULL
);

INSERT INTO `sys_apis` (`created_at`,`updated_at`,`api_group`,`method`,`path`,`description`)
SELECT NOW(), NOW(), '手机号注册任务', 'POST', '/phoneRegisterTask/device/report', '设备上报手机号注册任务进度'
WHERE NOT EXISTS (
  SELECT 1 FROM `sys_apis` WHERE `path`='/phoneRegisterTask/device/report' AND `method`='POST' AND `deleted_at` IS NULL
);

INSERT INTO `sys_apis` (`created_at`,`updated_at`,`api_group`,`method`,`path`,`description`)
SELECT NOW(), NOW(), '手机号注册任务', 'GET', '/phoneRegisterTask/device/config', '获取手机号注册设备配置'
WHERE NOT EXISTS (
  SELECT 1 FROM `sys_apis` WHERE `path`='/phoneRegisterTask/device/config' AND `method`='GET' AND `deleted_at` IS NULL
);

INSERT INTO `casbin_rule` (`ptype`,`v0`,`v1`,`v2`,`v3`,`v4`,`v5`)
SELECT 'p','888','/phoneRegisterTask/create','POST','','',''
WHERE NOT EXISTS (
  SELECT 1 FROM `casbin_rule` WHERE `ptype`='p' AND `v0`='888' AND `v1`='/phoneRegisterTask/create' AND `v2`='POST'
);

INSERT INTO `casbin_rule` (`ptype`,`v0`,`v1`,`v2`,`v3`,`v4`,`v5`)
SELECT 'p','888','/phoneRegisterTask/submitCode','POST','','',''
WHERE NOT EXISTS (
  SELECT 1 FROM `casbin_rule` WHERE `ptype`='p' AND `v0`='888' AND `v1`='/phoneRegisterTask/submitCode' AND `v2`='POST'
);

INSERT INTO `casbin_rule` (`ptype`,`v0`,`v1`,`v2`,`v3`,`v4`,`v5`)
SELECT 'p','888','/phoneRegisterTask/active','GET','','',''
WHERE NOT EXISTS (
  SELECT 1 FROM `casbin_rule` WHERE `ptype`='p' AND `v0`='888' AND `v1`='/phoneRegisterTask/active' AND `v2`='GET'
);

INSERT INTO `casbin_rule` (`ptype`,`v0`,`v1`,`v2`,`v3`,`v4`,`v5`)
SELECT 'p','888','/phoneRegisterTask/actives','GET','','',''
WHERE NOT EXISTS (
  SELECT 1 FROM `casbin_rule` WHERE `ptype`='p' AND `v0`='888' AND `v1`='/phoneRegisterTask/actives' AND `v2`='GET'
);

INSERT INTO `casbin_rule` (`ptype`,`v0`,`v1`,`v2`,`v3`,`v4`,`v5`)
SELECT 'p','888','/phoneRegisterTask/list','POST','','',''
WHERE NOT EXISTS (
  SELECT 1 FROM `casbin_rule` WHERE `ptype`='p' AND `v0`='888' AND `v1`='/phoneRegisterTask/list' AND `v2`='POST'
);

INSERT INTO `casbin_rule` (`ptype`,`v0`,`v1`,`v2`,`v3`,`v4`,`v5`)
SELECT 'p','888','/phoneRegisterTask/summary','GET','','',''
WHERE NOT EXISTS (
  SELECT 1 FROM `casbin_rule` WHERE `ptype`='p' AND `v0`='888' AND `v1`='/phoneRegisterTask/summary' AND `v2`='GET'
);

INSERT INTO `casbin_rule` (`ptype`,`v0`,`v1`,`v2`,`v3`,`v4`,`v5`)
SELECT 'p','100','/phoneRegisterTask/list','POST','','',''
WHERE NOT EXISTS (
  SELECT 1 FROM `casbin_rule` WHERE `ptype`='p' AND `v0`='100' AND `v1`='/phoneRegisterTask/list' AND `v2`='POST'
);

INSERT INTO `casbin_rule` (`ptype`,`v0`,`v1`,`v2`,`v3`,`v4`,`v5`)
SELECT 'p','100','/phoneRegisterTask/summary','GET','','',''
WHERE NOT EXISTS (
  SELECT 1 FROM `casbin_rule` WHERE `ptype`='p' AND `v0`='100' AND `v1`='/phoneRegisterTask/summary' AND `v2`='GET'
);

INSERT INTO `casbin_rule` (`ptype`,`v0`,`v1`,`v2`,`v3`,`v4`,`v5`)
SELECT 'p','200','/phoneRegisterTask/list','POST','','',''
WHERE NOT EXISTS (
  SELECT 1 FROM `casbin_rule` WHERE `ptype`='p' AND `v0`='200' AND `v1`='/phoneRegisterTask/list' AND `v2`='POST'
);

INSERT INTO `casbin_rule` (`ptype`,`v0`,`v1`,`v2`,`v3`,`v4`,`v5`)
SELECT 'p','200','/phoneRegisterTask/summary','GET','','',''
WHERE NOT EXISTS (
  SELECT 1 FROM `casbin_rule` WHERE `ptype`='p' AND `v0`='200' AND `v1`='/phoneRegisterTask/summary' AND `v2`='GET'
);

INSERT INTO `casbin_rule` (`ptype`,`v0`,`v1`,`v2`,`v3`,`v4`,`v5`)
SELECT 'p','300','/phoneRegisterTask/create','POST','','',''
WHERE NOT EXISTS (
  SELECT 1 FROM `casbin_rule` WHERE `ptype`='p' AND `v0`='300' AND `v1`='/phoneRegisterTask/create' AND `v2`='POST'
);

INSERT INTO `casbin_rule` (`ptype`,`v0`,`v1`,`v2`,`v3`,`v4`,`v5`)
SELECT 'p','300','/phoneRegisterTask/submitCode','POST','','',''
WHERE NOT EXISTS (
  SELECT 1 FROM `casbin_rule` WHERE `ptype`='p' AND `v0`='300' AND `v1`='/phoneRegisterTask/submitCode' AND `v2`='POST'
);

INSERT INTO `casbin_rule` (`ptype`,`v0`,`v1`,`v2`,`v3`,`v4`,`v5`)
SELECT 'p','300','/phoneRegisterTask/active','GET','','',''
WHERE NOT EXISTS (
  SELECT 1 FROM `casbin_rule` WHERE `ptype`='p' AND `v0`='300' AND `v1`='/phoneRegisterTask/active' AND `v2`='GET'
);

INSERT INTO `casbin_rule` (`ptype`,`v0`,`v1`,`v2`,`v3`,`v4`,`v5`)
SELECT 'p','300','/phoneRegisterTask/actives','GET','','',''
WHERE NOT EXISTS (
  SELECT 1 FROM `casbin_rule` WHERE `ptype`='p' AND `v0`='300' AND `v1`='/phoneRegisterTask/actives' AND `v2`='GET'
);

INSERT INTO `casbin_rule` (`ptype`,`v0`,`v1`,`v2`,`v3`,`v4`,`v5`)
SELECT 'p','300','/phoneRegisterTask/list','POST','','',''
WHERE NOT EXISTS (
  SELECT 1 FROM `casbin_rule` WHERE `ptype`='p' AND `v0`='300' AND `v1`='/phoneRegisterTask/list' AND `v2`='POST'
);

COMMIT;
