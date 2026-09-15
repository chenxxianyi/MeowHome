-- +migrate Up
-- +migrate StatementBegin

-- daily_records 需要一个 payload 列承载类型专属字段
-- （喂食的 amount/appetite、呕吐的 count/content、体重的 weight 等）。
-- 事件头保持统一，差异部分以 JSON 存放，避免为每种记录类型建表。
--
-- MySQL 8 不支持 ADD COLUMN IF NOT EXISTS，用 information_schema 判断保证幂等。

SET @sql := IF(
  (SELECT COUNT(*) FROM information_schema.COLUMNS
   WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'daily_records' AND COLUMN_NAME = 'payload') = 0,
  'ALTER TABLE daily_records ADD COLUMN payload TEXT NULL DEFAULT NULL AFTER note',
  'DO 0');
PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;

-- +migrate StatementEnd
