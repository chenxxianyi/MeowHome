-- +migrate Down
-- +migrate StatementBegin
DROP TABLE IF EXISTS media_assets;
DROP TABLE IF EXISTS cat_health_profiles;
DROP TABLE IF EXISTS cats;
DROP TABLE IF EXISTS family_members;
DROP TABLE IF EXISTS refresh_tokens;
-- +migrate StatementEnd
