-- +migrate Up
-- +migrate StatementBegin
SET @sql := IF((SELECT COUNT(*) FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='ai_agent_messages' AND COLUMN_NAME='tool_call_id')=0,
  'ALTER TABLE ai_agent_messages ADD COLUMN tool_call_id VARCHAR(128) NULL', 'DO 0');
PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;
SET @sql := IF((SELECT COUNT(*) FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='ai_agent_messages' AND COLUMN_NAME='tool_name')=0,
  'ALTER TABLE ai_agent_messages ADD COLUMN tool_name VARCHAR(64) NULL', 'DO 0');
PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;
SET @sql := IF((SELECT COUNT(*) FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='ai_agent_messages' AND COLUMN_NAME='turn_id')=0,
  'ALTER TABLE ai_agent_messages ADD COLUMN turn_id VARCHAR(26) NULL', 'DO 0');
PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;
SET @sql := IF((SELECT COUNT(*) FROM information_schema.STATISTICS
  WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='ai_agent_sessions' AND INDEX_NAME='idx_agent_sessions_updated')=0,
  'ALTER TABLE ai_agent_sessions ADD KEY idx_agent_sessions_updated (family_id, user_id, updated_at, id)', 'DO 0');
PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;
SET @sql := IF((SELECT COUNT(*) FROM information_schema.STATISTICS
  WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='ai_agent_messages' AND INDEX_NAME='idx_agent_messages_private')=0,
  'ALTER TABLE ai_agent_messages ADD KEY idx_agent_messages_private (family_id, session_id, visibility, generated_at, id)', 'DO 0');
PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;
SET @sql := IF((SELECT COUNT(*) FROM information_schema.STATISTICS
  WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='ai_agent_messages' AND INDEX_NAME='idx_agent_messages_tool_call')=0,
  'ALTER TABLE ai_agent_messages ADD KEY idx_agent_messages_tool_call (turn_id, tool_call_id)', 'DO 0');
PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;
-- +migrate StatementEnd
