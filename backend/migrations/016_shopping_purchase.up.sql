-- +migrate Up
-- +migrate StatementBegin

-- 待购买清单：用户计划采购的物品，独立于库存（inventory_items.quantity = 0 不表示待购）。
CREATE TABLE IF NOT EXISTS shopping_list_items (
    id                        VARCHAR(26)   NOT NULL PRIMARY KEY,
    created_by                VARCHAR(26)   NOT NULL,
    family_id                 VARCHAR(26)   NOT NULL,
    name                      VARCHAR(255)  NOT NULL,
    brand                     VARCHAR(120)  NULL DEFAULT NULL,
    specification             VARCHAR(120)  NULL DEFAULT NULL,
    category                  VARCHAR(64)   NULL DEFAULT NULL,
    quantity                  INT           NOT NULL DEFAULT 1,
    unit                      VARCHAR(32)   NULL DEFAULT NULL,
    note                      TEXT          NULL DEFAULT NULL,
    priority                  VARCHAR(20)   NOT NULL DEFAULT 'normal',
    status                    VARCHAR(20)   NOT NULL DEFAULT 'todo',
    matched_inventory_item_id VARCHAR(26)   NULL DEFAULT NULL,
    completed_at              DATETIME(3)   NULL DEFAULT NULL,
    created_at                DATETIME(3)   NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at                DATETIME(3)   NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    deleted_at                DATETIME(3)   NULL DEFAULT NULL,
    KEY idx_shopping_family_status (family_id, status),
    KEY idx_shopping_family_created (family_id, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 采购导入：一次订单截图识别及其确认状态。
CREATE TABLE IF NOT EXISTS purchase_imports (
    id                  VARCHAR(26)   NOT NULL PRIMARY KEY,
    created_by          VARCHAR(26)   NOT NULL,
    family_id           VARCHAR(26)   NOT NULL,
    platform            VARCHAR(32)   NULL DEFAULT NULL,
    order_date          VARCHAR(10)   NULL DEFAULT NULL,
    order_no            VARCHAR(64)   NULL DEFAULT NULL,
    total_amount        BIGINT        NOT NULL DEFAULT 0,
    currency            VARCHAR(8)    NOT NULL DEFAULT 'CNY',
    status              VARCHAR(20)   NOT NULL DEFAULT 'uploaded',
    media_keys          TEXT          NULL DEFAULT NULL,
    recognition_version INT           NOT NULL DEFAULT 0,
    error_summary       VARCHAR(512)  NULL DEFAULT NULL,
    confirmed_at        DATETIME(3)   NULL DEFAULT NULL,
    expense_id          VARCHAR(26)   NULL DEFAULT NULL,
    created_at          DATETIME(3)   NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at          DATETIME(3)   NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    deleted_at          DATETIME(3)   NULL DEFAULT NULL,
    KEY idx_purchase_family_status (family_id, status),
    KEY idx_purchase_family_created (family_id, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 采购导入商品行：可编辑草稿；确认后作为库存流水与支出明细的来源。
CREATE TABLE IF NOT EXISTS purchase_import_lines (
    id                        VARCHAR(26)   NOT NULL PRIMARY KEY,
    created_by                VARCHAR(26)   NOT NULL,
    family_id                 VARCHAR(26)   NOT NULL,
    import_id                 VARCHAR(26)   NOT NULL,
    line_no                   INT           NOT NULL DEFAULT 0,
    raw_title                 VARCHAR(512)  NULL DEFAULT NULL,
    name                      VARCHAR(255)  NOT NULL,
    specification             VARCHAR(120)  NULL DEFAULT NULL,
    quantity                  INT           NOT NULL DEFAULT 1,
    unit                      VARCHAR(32)   NULL DEFAULT NULL,
    line_amount               BIGINT        NULL DEFAULT NULL,
    confidence                VARCHAR(10)   NOT NULL DEFAULT 'none',
    evidence                  VARCHAR(512)  NULL DEFAULT NULL,
    matched_inventory_item_id VARCHAR(26)   NULL DEFAULT NULL,
    matched_shopping_item_id  VARCHAR(26)   NULL DEFAULT NULL,
    include_in_stock          TINYINT(1)    NOT NULL DEFAULT 0,
    created_at                DATETIME(3)   NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at                DATETIME(3)   NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    deleted_at                DATETIME(3)   NULL DEFAULT NULL,
    KEY idx_lines_import (import_id),
    KEY idx_lines_family_import (family_id, import_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 库存项增加可选品牌与规格，避免不同配方/重量的猫粮被合并。
SET @sql := IF((SELECT COUNT(*) FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='inventory_items' AND COLUMN_NAME='brand')=0,
  'ALTER TABLE inventory_items ADD COLUMN brand varchar(120) NULL AFTER name', 'DO 0');
PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;

SET @sql := IF((SELECT COUNT(*) FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='inventory_items' AND COLUMN_NAME='specification')=0,
  'ALTER TABLE inventory_items ADD COLUMN specification varchar(120) NULL AFTER brand', 'DO 0');
PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;

-- 支出增加来源引用：一个采购导入只生成一笔订单支出。
SET @sql := IF((SELECT COUNT(*) FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='expenses' AND COLUMN_NAME='source_type')=0,
  'ALTER TABLE expenses ADD COLUMN source_type varchar(32) NULL AFTER cat_ids', 'DO 0');
PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;

SET @sql := IF((SELECT COUNT(*) FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='expenses' AND COLUMN_NAME='source_id')=0,
  'ALTER TABLE expenses ADD COLUMN source_id varchar(26) NULL AFTER source_type', 'DO 0');
PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;

SET @sql := IF((SELECT COUNT(*) FROM information_schema.STATISTICS
  WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='expenses' AND INDEX_NAME='uq_expense_family_source')=0,
  'ALTER TABLE expenses ADD UNIQUE INDEX uq_expense_family_source (family_id, source_type, source_id)', 'DO 0');
PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;

-- 库存流水增加来源引用：每个采购商品行只生成一次入库流水。
SET @sql := IF((SELECT COUNT(*) FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='inventory_transactions' AND COLUMN_NAME='source_type')=0,
  'ALTER TABLE inventory_transactions ADD COLUMN source_type varchar(32) NULL AFTER reason', 'DO 0');
PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;

SET @sql := IF((SELECT COUNT(*) FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='inventory_transactions' AND COLUMN_NAME='source_id')=0,
  'ALTER TABLE inventory_transactions ADD COLUMN source_id varchar(26) NULL AFTER source_type', 'DO 0');
PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;

SET @sql := IF((SELECT COUNT(*) FROM information_schema.STATISTICS
  WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='inventory_transactions' AND INDEX_NAME='uq_inv_tx_family_source')=0,
  'ALTER TABLE inventory_transactions ADD UNIQUE INDEX uq_inv_tx_family_source (family_id, source_type, source_id)', 'DO 0');
PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;

-- 为已有库存建立期初流水（变更 = 当前数量，幂等：每个库存项最多一条）。
-- 说明：ID 由 'ob' + 库存项 ID 前 24 位生成，确定性且不依赖 SHA2()
--       （部分 MySQL 构建未启用 OpenSSL 时 SHA2 返回 NULL）。
INSERT INTO inventory_transactions
  (id, created_by, family_id, item_id, `change`, reason, source_type, source_id, created_at, updated_at)
SELECT CONCAT('ob', LEFT(ii.id, 24)),
       ii.created_by, ii.family_id, ii.id, ii.quantity, 'opening_balance', 'opening_balance', ii.id,
       ii.created_at, ii.updated_at
FROM inventory_items ii
WHERE ii.quantity <> 0
  AND ii.deleted_at IS NULL
  AND NOT EXISTS (
    SELECT 1 FROM inventory_transactions t
    WHERE t.family_id = ii.family_id AND t.source_type = 'opening_balance' AND t.source_id = ii.id
  );

-- +migrate StatementEnd
