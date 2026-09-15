-- +migrate Up
-- +migrate StatementBegin

-- 补齐缺失的 deleted_at 列。
--
-- 这 4 张表由 001/002 创建时未包含 deleted_at，但 Go 模型内嵌的 Base
-- 带有 DeletedAt 字段，GORM 会把它写进 INSERT 语句，导致
-- "Unknown column 'deleted_at' in 'field list'"（注册接口 500 即由此而来）。
--
-- MySQL 8 不支持 ADD COLUMN IF NOT EXISTS，用 information_schema 判断保证幂等。

SET @sql := IF(
  (SELECT COUNT(*) FROM information_schema.COLUMNS
   WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'refresh_tokens' AND COLUMN_NAME = 'deleted_at') = 0,
  'ALTER TABLE refresh_tokens ADD COLUMN deleted_at DATETIME(3) NULL DEFAULT NULL',
  'DO 0');
PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;

SET @sql := IF(
  (SELECT COUNT(*) FROM information_schema.COLUMNS
   WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'cat_health_profiles' AND COLUMN_NAME = 'deleted_at') = 0,
  'ALTER TABLE cat_health_profiles ADD COLUMN deleted_at DATETIME(3) NULL DEFAULT NULL',
  'DO 0');
PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;

SET @sql := IF(
  (SELECT COUNT(*) FROM information_schema.COLUMNS
   WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'media_assets' AND COLUMN_NAME = 'deleted_at') = 0,
  'ALTER TABLE media_assets ADD COLUMN deleted_at DATETIME(3) NULL DEFAULT NULL',
  'DO 0');
PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;

SET @sql := IF(
  (SELECT COUNT(*) FROM information_schema.COLUMNS
   WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'audit_logs' AND COLUMN_NAME = 'deleted_at') = 0,
  'ALTER TABLE audit_logs ADD COLUMN deleted_at DATETIME(3) NULL DEFAULT NULL',
  'DO 0');
PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;

-- +migrate StatementEnd
