ALTER TABLE `sys_phone_register_tasks`
  ADD COLUMN `code_requested_at` datetime(3) NULL COMMENT '进入待地推验证码时间' AFTER `pending_code`,
  ADD INDEX `idx_sys_phone_register_tasks_code_requested_at` (`code_requested_at`);
