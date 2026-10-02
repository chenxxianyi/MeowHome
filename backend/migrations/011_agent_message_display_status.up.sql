-- +migrate Up
-- +migrate StatementBegin
SET @sql := IF((SELECT COUNT(*) FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='ai_agent_messages' AND COLUMN_NAME='display_status')=0,
  'ALTER TABLE ai_agent_messages ADD COLUMN display_status VARCHAR(16) NULL', 'DO 0');
PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;
UPDATE ai_agent_messages SET display_status='dismissed' WHERE action_status='dismissed' AND display_status IS NULL;
-- +migrate StatementEnd
