-- +migrate Down
-- +migrate StatementBegin
DROP TABLE IF EXISTS idempotency_keys;
DROP TABLE IF EXISTS export_jobs;
DROP TABLE IF EXISTS ai_evidence_links;
DROP TABLE IF EXISTS ai_analysis_reports;
DROP TABLE IF EXISTS ai_parse_sessions;
DROP TABLE IF EXISTS cat_interactions;
DROP TABLE IF EXISTS timeline_events;
DROP TABLE IF EXISTS expenses;
DROP TABLE IF EXISTS inventory_transactions;
DROP TABLE IF EXISTS inventory_items;
DROP TABLE IF EXISTS care_tasks;
DROP TABLE IF EXISTS reminder_occurrences;
DROP TABLE IF EXISTS reminders;
DROP TABLE IF EXISTS daily_records;
-- +migrate StatementEnd
