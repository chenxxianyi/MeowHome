-- +migrate Down
-- +migrate StatementBegin
SET @sql := IF((SELECT COUNT(*) FROM information_schema.STATISTICS
  WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='ai_agent_messages' AND INDEX_NAME='uk_agent_message_client')>0,
  'ALTER TABLE ai_agent_messages DROP INDEX uk_agent_message_client', 'DO 0');
PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;
-- +migrate StatementEnd
