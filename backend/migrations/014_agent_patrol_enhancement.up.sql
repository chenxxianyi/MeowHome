-- +migrate Up
-- +migrate StatementBegin
SET @ddl = IF((SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='ai_agent_messages' AND COLUMN_NAME='rule_body')=0,
'ALTER TABLE ai_agent_messages ADD COLUMN rule_body TEXT NULL, ADD COLUMN enhanced_body TEXT NULL, ADD COLUMN enhance_status VARCHAR(16) NULL, ADD COLUMN enhance_model VARCHAR(80) NULL, ADD COLUMN enhance_prompt_version VARCHAR(32) NULL', 'SELECT 1');
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;
UPDATE ai_agent_messages SET rule_body=body WHERE visibility='family' AND rule_id IS NOT NULL AND rule_id<>'' AND rule_body IS NULL;
-- +migrate StatementEnd
