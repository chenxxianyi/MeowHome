-- +migrate Down
-- +migrate StatementBegin
ALTER TABLE reminders DROP INDEX idx_agent_reminders_due;
-- +migrate StatementEnd
