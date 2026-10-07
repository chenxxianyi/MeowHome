-- +migrate Up
-- +migrate StatementBegin
SET @ddl = IF((SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='ai_agent_messages' AND COLUMN_NAME='run_status')=0,
'ALTER TABLE ai_agent_messages ADD COLUMN run_status VARCHAR(16) NOT NULL DEFAULT '''', ADD COLUMN run_token VARCHAR(26) NOT NULL DEFAULT '''', ADD COLUMN run_lease_until DATETIME(3) NULL, ADD COLUMN degraded BOOLEAN NOT NULL DEFAULT FALSE, ADD COLUMN prompt_tokens INT NOT NULL DEFAULT 0, ADD COLUMN completion_tokens INT NOT NULL DEFAULT 0, ADD COLUMN total_tokens INT NOT NULL DEFAULT 0, ADD COLUMN run_error VARCHAR(64) NOT NULL DEFAULT '''', ADD COLUMN duration_ms BIGINT NOT NULL DEFAULT 0, ADD INDEX idx_agent_session_run (family_id, session_id, role, run_status, run_lease_until)', 'SELECT 1');
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;
-- Repair historical conversation drafts that were incorrectly shared.
UPDATE ai_agent_messages SET visibility='private' WHERE type='reminder_draft' AND session_id IS NOT NULL AND session_id<>'' AND turn_id IS NOT NULL AND turn_id<>'';
-- +migrate StatementEnd
