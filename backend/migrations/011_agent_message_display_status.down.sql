-- +migrate Down
-- +migrate StatementBegin
ALTER TABLE ai_agent_messages DROP COLUMN display_status;
-- +migrate StatementEnd
