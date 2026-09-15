-- +migrate Up
-- +migrate StatementBegin

-- 展示用日期字段统一改为 VARCHAR(10)。
--
-- 原因：
--   1. DSN 开启 parseTime=True，驱动会把 DATE 列读成 time.Time；扫进 Go 的
--      string 字段会得到 "2021-03-15T00:00:00Z"，与 API 契约要求的
--      "2021-03-15" 不符。
--   2. Go 侧零值空字符串写入 DATE 列会被拒绝（Error 1292 Incorrect date value）。
--
-- ISO-8601 定长字符串的字典序与时间序一致，排序与区间过滤依然正确。
-- 需要精确定位与范围查询的 daily_records.occurred_at 保持 DATETIME 不变。
--
-- MODIFY COLUMN 可重复执行，因此本迁移是幂等的。

ALTER TABLE cats             MODIFY COLUMN birthday    VARCHAR(10) NULL DEFAULT NULL;
ALTER TABLE inventory_items  MODIFY COLUMN expiry      VARCHAR(10) NULL DEFAULT NULL;
ALTER TABLE expenses         MODIFY COLUMN occurred_on VARCHAR(10) NULL DEFAULT NULL;
ALTER TABLE timeline_events  MODIFY COLUMN occurred_on VARCHAR(10) NULL DEFAULT NULL;
ALTER TABLE cat_interactions MODIFY COLUMN occurred_on VARCHAR(10) NULL DEFAULT NULL;

-- +migrate StatementEnd
