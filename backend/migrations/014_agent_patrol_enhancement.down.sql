-- +migrate Down
-- +migrate StatementBegin
ALTER TABLE ai_agent_messages DROP COLUMN rule_body, DROP COLUMN enhanced_body, DROP COLUMN enhance_status, DROP COLUMN enhance_model, DROP COLUMN enhance_prompt_version;
-- +migrate StatementEnd
