-- +migrate Down
-- +migrate StatementBegin
DROP TABLE IF EXISTS families;
DROP TABLE IF EXISTS users;
-- +migrate StatementEnd