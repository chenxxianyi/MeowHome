-- +migrate Down
-- +migrate StatementBegin
ALTER TABLE ai_agent_messages DROP INDEX idx_agent_session_run, DROP COLUMN run_status, DROP COLUMN run_token, DROP COLUMN run_lease_until, DROP COLUMN degraded, DROP COLUMN prompt_tokens, DROP COLUMN completion_tokens, DROP COLUMN total_tokens, DROP COLUMN run_error, DROP COLUMN duration_ms;
-- +migrate StatementEnd
