-- +migrate Down
-- +migrate StatementBegin
DROP INDEX idx_daily_records_family_created ON daily_records;
DROP TABLE IF EXISTS ai_agent_task_progress;
-- +migrate StatementEnd
