-- +migrate Up
-- +migrate StatementBegin
SET @sql := IF((SELECT COUNT(*) FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='reminders' AND COLUMN_NAME='source_record_id')=0,
  'ALTER TABLE reminders ADD COLUMN source_record_id varchar(26) NULL', 'DO 0');
PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;

SET @sql := IF((SELECT COUNT(*) FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='reminders' AND COLUMN_NAME='completed_record_id')=0,
  'ALTER TABLE reminders ADD COLUMN completed_record_id varchar(26) NULL', 'DO 0');
PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;

SET @sql := IF((SELECT COUNT(*) FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='reminders' AND COLUMN_NAME='metadata')=0,
  'ALTER TABLE reminders ADD COLUMN metadata text NULL', 'DO 0');
PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;

SET @sql := IF((SELECT COUNT(*) FROM information_schema.STATISTICS
  WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='reminders' AND INDEX_NAME='uq_reminder_family_source_record')=0,
  'ALTER TABLE reminders ADD UNIQUE INDEX uq_reminder_family_source_record (family_id, source_record_id)', 'DO 0');
PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;
-- +migrate StatementEnd
