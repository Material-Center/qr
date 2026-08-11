-- QQ缓存销售结算性能索引补丁（幂等）
-- 适用：MySQL / MariaDB

SET @has_idx_qq_cache_extractor_sales_settled_deleted := (
  SELECT COUNT(1)
  FROM information_schema.statistics
  WHERE table_schema = DATABASE()
    AND table_name = 'sys_qq_cache_records'
    AND index_name = 'idx_qq_cache_extractor_sales_settled_deleted'
);
SET @sql_add_idx_qq_cache_extractor_sales_settled_deleted := IF(
  @has_idx_qq_cache_extractor_sales_settled_deleted = 0,
  'ALTER TABLE `sys_qq_cache_records` ADD INDEX `idx_qq_cache_extractor_sales_settled_deleted` (`extractor`, `sales_settled_at`, `deleted_at`)',
  'SELECT 1'
);
PREPARE stmt_add_idx_qq_cache_extractor_sales_settled_deleted FROM @sql_add_idx_qq_cache_extractor_sales_settled_deleted;
EXECUTE stmt_add_idx_qq_cache_extractor_sales_settled_deleted;
DEALLOCATE PREPARE stmt_add_idx_qq_cache_extractor_sales_settled_deleted;

SET @has_idx_qq_cache_extract_record_sales_settled_deleted := (
  SELECT COUNT(1)
  FROM information_schema.statistics
  WHERE table_schema = DATABASE()
    AND table_name = 'sys_qq_cache_records'
    AND index_name = 'idx_qq_cache_extract_record_sales_settled_deleted'
);
SET @sql_add_idx_qq_cache_extract_record_sales_settled_deleted := IF(
  @has_idx_qq_cache_extract_record_sales_settled_deleted = 0,
  'ALTER TABLE `sys_qq_cache_records` ADD INDEX `idx_qq_cache_extract_record_sales_settled_deleted` (`extract_record_id`, `sales_settled_at`, `deleted_at`)',
  'SELECT 1'
);
PREPARE stmt_add_idx_qq_cache_extract_record_sales_settled_deleted FROM @sql_add_idx_qq_cache_extract_record_sales_settled_deleted;
EXECUTE stmt_add_idx_qq_cache_extract_record_sales_settled_deleted;
DEALLOCATE PREPARE stmt_add_idx_qq_cache_extract_record_sales_settled_deleted;
