-- +migrate Down
-- +migrate StatementBegin
ALTER TABLE daily_records DROP COLUMN payload;
-- +migrate StatementEnd
