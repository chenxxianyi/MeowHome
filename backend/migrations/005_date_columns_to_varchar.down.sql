-- +migrate Down
-- +migrate StatementBegin
ALTER TABLE cats             MODIFY COLUMN birthday    DATE NULL DEFAULT NULL;
ALTER TABLE inventory_items  MODIFY COLUMN expiry      DATE NULL DEFAULT NULL;
ALTER TABLE expenses         MODIFY COLUMN occurred_on DATE NULL DEFAULT NULL;
ALTER TABLE timeline_events  MODIFY COLUMN occurred_on DATE NULL DEFAULT NULL;
ALTER TABLE cat_interactions MODIFY COLUMN occurred_on DATE NULL DEFAULT NULL;
-- +migrate StatementEnd
