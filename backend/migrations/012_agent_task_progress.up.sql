-- +migrate Up
-- +migrate StatementBegin
CREATE TABLE IF NOT EXISTS ai_agent_task_progress (
    family_id         VARCHAR(26) NOT NULL,
    task_key          VARCHAR(64) NOT NULL,
    status            VARCHAR(16) NOT NULL,
    last_success_at   DATETIME(3) NULL,
    last_failure_at   DATETIME(3) NULL,
    cursor_created_at DATETIME(3) NULL,
    cursor_id         VARCHAR(26) NOT NULL DEFAULT '',
    PRIMARY KEY (family_id, task_key)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE INDEX idx_daily_records_family_created ON daily_records (family_id, created_at, id);
-- +migrate StatementEnd
