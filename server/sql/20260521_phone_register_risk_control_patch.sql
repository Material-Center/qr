-- 手机号注册风控比例补丁（幂等）
-- 说明：按地推+当天维护成功上报口径统计；真实失败不计入风控比例。

SET @db_name = DATABASE();

CREATE TABLE IF NOT EXISTS `sys_phone_register_risk_daily_stats` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  `promoter_id` bigint unsigned NOT NULL DEFAULT 0 COMMENT '地推账号ID',
  `biz_date` varchar(10) NOT NULL DEFAULT '' COMMENT '业务日期YYYY-MM-DD',
  `success_report_count` bigint NOT NULL DEFAULT 0 COMMENT '当天成功上报口径总数',
  `risk_fail_count` bigint NOT NULL DEFAULT 0 COMMENT '当天风控失败数',
  `last_risk_success_seq` bigint NOT NULL DEFAULT 0 COMMENT '最近一次风控所在成功上报序号',
  `last_risk_reason` varchar(32) NOT NULL DEFAULT '' COMMENT '最近一次风控原因',
  `last_risk_gap` bigint NOT NULL DEFAULT 0 COMMENT '最近一次风控间隔',
  `previous_risk_gap` bigint NOT NULL DEFAULT 0 COMMENT '上上次风控间隔',
  `previous_risk_reason` varchar(32) NOT NULL DEFAULT '' COMMENT '上上次风控原因',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_phone_register_risk_day` (`promoter_id`, `biz_date`),
  KEY `idx_sys_phone_register_risk_daily_stats_deleted_at` (`deleted_at`),
  KEY `idx_sys_phone_register_risk_daily_stats_promoter_id` (`promoter_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

SET @sql = (
  SELECT IF(
    EXISTS (
      SELECT 1
      FROM information_schema.STATISTICS
      WHERE TABLE_SCHEMA = @db_name
        AND TABLE_NAME = 'sys_phone_register_tasks'
        AND INDEX_NAME = 'idx_phone_register_risk_day_count'
    ),
    'SELECT 1',
    'ALTER TABLE `sys_phone_register_tasks` ADD INDEX `idx_phone_register_risk_day_count` (`promoter_id`, `finished_at`, `status_code`)'
  )
);
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;
