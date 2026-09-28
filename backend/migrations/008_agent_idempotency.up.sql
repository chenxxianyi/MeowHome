-- +migrate Up
-- +migrate StatementBegin

-- client_message_id 只对同一家庭、用户和角色唯一；NULL 的巡检消息不受影响。
SET @sql := IF((SELECT COUNT(*) FROM information_schema.STATISTICS
  WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='ai_agent_messages' AND INDEX_NAME='uk_agent_message_client')=0,
  'ALTER TABLE ai_agent_messages ADD UNIQUE KEY uk_agent_message_client (family_id, user_id, client_message_id, role)', 'DO 0');
PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;

-- +migrate StatementEnd

-- +migrate Down
-- +migrate StatementBegin
SET @sql := IF((SELECT COUNT(*) FROM information_schema.STATISTICS
  WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='ai_agent_messages' AND INDEX_NAME='uk_agent_message_client')>0,
  'ALTER TABLE ai_agent_messages DROP INDEX uk_agent_message_client', 'DO 0');
PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;
-- +migrate StatementEnd
