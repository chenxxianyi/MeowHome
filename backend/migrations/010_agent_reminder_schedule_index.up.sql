-- +migrate Up
-- +migrate StatementBegin
SET @sql := IF((SELECT COUNT(*) FROM information_schema.STATISTICS
  WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='reminders' AND INDEX_NAME='idx_agent_reminders_due')=0,
  'ALTER TABLE reminders ADD KEY idx_agent_reminders_due (family_id, state, type, scheduled_at, id)', 'DO 0');
PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;
-- +migrate StatementEnd
