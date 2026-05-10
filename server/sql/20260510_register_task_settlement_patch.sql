ALTER TABLE `sys_register_tasks`
  ADD COLUMN `settled_at` datetime(3) NULL COMMENT '结算时间' AFTER `exported_by`,
  ADD COLUMN `settled_by` bigint unsigned NULL COMMENT '结算管理员ID' AFTER `settled_at`,
  ADD INDEX `idx_sys_register_tasks_settled_at` (`settled_at`),
  ADD INDEX `idx_sys_register_tasks_settled_by` (`settled_by`);

ALTER TABLE `sys_phone_register_tasks`
  ADD COLUMN `settled_at` datetime(3) NULL COMMENT '结算时间' AFTER `finished_at`,
  ADD COLUMN `settled_by` bigint unsigned NULL COMMENT '结算管理员ID' AFTER `settled_at`,
  ADD INDEX `idx_sys_phone_register_tasks_settled_at` (`settled_at`),
  ADD INDEX `idx_sys_phone_register_tasks_settled_by` (`settled_by`);

INSERT INTO `sys_apis` (`created_at`, `updated_at`, `api_group`, `method`, `path`, `description`)
SELECT NOW(), NOW(), '注册任务', 'POST', '/registerTask/settle', '管理员结算团长反扫任务'
WHERE NOT EXISTS (
  SELECT 1 FROM `sys_apis` WHERE `path` = '/registerTask/settle' AND `method` = 'POST' AND `deleted_at` IS NULL
);

INSERT INTO `sys_apis` (`created_at`, `updated_at`, `api_group`, `method`, `path`, `description`)
SELECT NOW(), NOW(), '手机号注册任务', 'POST', '/phoneRegisterTask/settle', '管理员结算团长手机号注册任务'
WHERE NOT EXISTS (
  SELECT 1 FROM `sys_apis` WHERE `path` = '/phoneRegisterTask/settle' AND `method` = 'POST' AND `deleted_at` IS NULL
);

INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`)
SELECT 'p', '888', '/registerTask/settle', 'POST', '', '', ''
WHERE NOT EXISTS (
  SELECT 1 FROM `casbin_rule` WHERE `ptype` = 'p' AND `v0` = '888' AND `v1` = '/registerTask/settle' AND `v2` = 'POST'
);

INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`)
SELECT 'p', '100', '/registerTask/settle', 'POST', '', '', ''
WHERE NOT EXISTS (
  SELECT 1 FROM `casbin_rule` WHERE `ptype` = 'p' AND `v0` = '100' AND `v1` = '/registerTask/settle' AND `v2` = 'POST'
);

INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`)
SELECT 'p', '888', '/phoneRegisterTask/settle', 'POST', '', '', ''
WHERE NOT EXISTS (
  SELECT 1 FROM `casbin_rule` WHERE `ptype` = 'p' AND `v0` = '888' AND `v1` = '/phoneRegisterTask/settle' AND `v2` = 'POST'
);

INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`)
SELECT 'p', '100', '/phoneRegisterTask/settle', 'POST', '', '', ''
WHERE NOT EXISTS (
  SELECT 1 FROM `casbin_rule` WHERE `ptype` = 'p' AND `v0` = '100' AND `v1` = '/phoneRegisterTask/settle' AND `v2` = 'POST'
);
