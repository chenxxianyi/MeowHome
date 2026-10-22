-- +migrate Down
-- +migrate StatementBegin
ALTER TABLE reminders DROP INDEX uq_reminder_family_source_record;
ALTER TABLE reminders DROP COLUMN metadata;
ALTER TABLE reminders DROP COLUMN completed_record_id;
ALTER TABLE reminders DROP COLUMN source_record_id;
-- +migrate StatementEnd
