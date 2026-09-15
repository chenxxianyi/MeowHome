-- +migrate Down
-- +migrate StatementBegin
ALTER TABLE refresh_tokens DROP COLUMN deleted_at;
ALTER TABLE cat_health_profiles DROP COLUMN deleted_at;
ALTER TABLE media_assets DROP COLUMN deleted_at;
ALTER TABLE audit_logs DROP COLUMN deleted_at;
-- +migrate StatementEnd
