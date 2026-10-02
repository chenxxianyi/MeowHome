-- +migrate Down
-- +migrate StatementBegin
ALTER TABLE ai_agent_messages DROP INDEX idx_agent_messages_tool_call;
ALTER TABLE ai_agent_messages DROP INDEX idx_agent_messages_private;
ALTER TABLE ai_agent_sessions DROP INDEX idx_agent_sessions_updated;
ALTER TABLE ai_agent_messages DROP COLUMN turn_id;
ALTER TABLE ai_agent_messages DROP COLUMN tool_name;
ALTER TABLE ai_agent_messages DROP COLUMN tool_call_id;
-- +migrate StatementEnd
