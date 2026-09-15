-- +migrate Up
-- +migrate StatementBegin

-- 日常记录事件头（记录/健康域的核心表）
CREATE TABLE IF NOT EXISTS daily_records (
    id               VARCHAR(26)   NOT NULL PRIMARY KEY,
    created_by       VARCHAR(26)   NOT NULL,
    family_id        VARCHAR(26)   NOT NULL,
    cat_ids          TEXT          NULL DEFAULT NULL,  -- JSON 数组
    record_type      VARCHAR(32)   NOT NULL,
    occurred_at      DATETIME(3)   NOT NULL,
    source           VARCHAR(20)   NOT NULL DEFAULT 'manual',
    severity         VARCHAR(20)   NOT NULL DEFAULT 'normal',
    title            VARCHAR(255)  NULL DEFAULT NULL,
    note             TEXT          NULL DEFAULT NULL,
    medical_visit_id VARCHAR(26)   NULL DEFAULT NULL,
    created_at       DATETIME(3)   NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at       DATETIME(3)   NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    deleted_at       DATETIME(3)   NULL DEFAULT NULL,
    KEY idx_records_family_time (family_id, occurred_at),
    KEY idx_records_type (record_type)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 提醒
CREATE TABLE IF NOT EXISTS reminders (
    id               VARCHAR(26)   NOT NULL PRIMARY KEY,
    created_by       VARCHAR(26)   NOT NULL,
    family_id        VARCHAR(26)   NOT NULL,
    cat_id           VARCHAR(64)   NULL DEFAULT NULL,  -- 猫咪 ID 或 "both"
    type             VARCHAR(32)   NOT NULL,
    title            VARCHAR(255)  NULL DEFAULT NULL,
    subtitle         VARCHAR(255)  NULL DEFAULT NULL,
    time_label       VARCHAR(64)   NULL DEFAULT NULL,
    state            VARCHAR(20)   NOT NULL DEFAULT 'todo',
    icon             VARCHAR(64)   NULL DEFAULT NULL,
    rule             TEXT          NULL DEFAULT NULL,
    created_at       DATETIME(3)   NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at       DATETIME(3)   NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    deleted_at       DATETIME(3)   NULL DEFAULT NULL,
    KEY idx_reminders_family (family_id),
    KEY idx_reminders_state (family_id, state)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS reminder_occurrences (
    id               VARCHAR(26)   NOT NULL PRIMARY KEY,
    created_by       VARCHAR(26)   NOT NULL,
    family_id        VARCHAR(26)   NOT NULL,
    reminder_id      VARCHAR(26)   NOT NULL,
    scheduled_at     VARCHAR(64)   NOT NULL,
    status           VARCHAR(20)   NOT NULL DEFAULT 'todo',
    created_at       DATETIME(3)   NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at       DATETIME(3)   NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    deleted_at       DATETIME(3)   NULL DEFAULT NULL,
    KEY idx_occurrence_family (family_id),
    KEY idx_occurrence_reminder (reminder_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS care_tasks (
    id               VARCHAR(26)   NOT NULL PRIMARY KEY,
    created_by       VARCHAR(26)   NOT NULL,
    family_id        VARCHAR(26)   NOT NULL,
    title            VARCHAR(255)  NOT NULL,
    status           VARCHAR(20)   NOT NULL DEFAULT 'todo',
    created_at       DATETIME(3)   NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at       DATETIME(3)   NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    deleted_at       DATETIME(3)   NULL DEFAULT NULL,
    KEY idx_care_tasks_family (family_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 库存
CREATE TABLE IF NOT EXISTS inventory_items (
    id                  VARCHAR(26)  NOT NULL PRIMARY KEY,
    created_by          VARCHAR(26)  NOT NULL,
    family_id           VARCHAR(26)  NOT NULL,
    name                VARCHAR(255) NOT NULL,
    category            VARCHAR(64)  NULL DEFAULT NULL,
    unit                VARCHAR(32)  NULL DEFAULT NULL,
    quantity            INT          NOT NULL DEFAULT 0,
    low_stock_threshold INT          NOT NULL DEFAULT 0,
    expiry              VARCHAR(10)  NULL DEFAULT NULL,
    created_at          DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at          DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    deleted_at          DATETIME(3)  NULL DEFAULT NULL,
    KEY idx_inventory_family (family_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS inventory_transactions (
    id               VARCHAR(26)   NOT NULL PRIMARY KEY,
    created_by       VARCHAR(26)   NOT NULL,
    family_id        VARCHAR(26)   NOT NULL,
    item_id          VARCHAR(26)   NOT NULL,
    `change`         INT           NOT NULL,  -- 正入负出
    reason           VARCHAR(120)  NULL DEFAULT NULL,
    created_at       DATETIME(3)   NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at       DATETIME(3)   NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    deleted_at       DATETIME(3)   NULL DEFAULT NULL,
    KEY idx_inv_tx_family (family_id),
    KEY idx_inv_tx_item (item_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 账目
CREATE TABLE IF NOT EXISTS expenses (
    id               VARCHAR(26)   NOT NULL PRIMARY KEY,
    created_by       VARCHAR(26)   NOT NULL,
    family_id        VARCHAR(26)   NOT NULL,
    occurred_on      VARCHAR(10)   NULL DEFAULT NULL,
    amount           BIGINT        NOT NULL DEFAULT 0,
    category         VARCHAR(64)   NULL DEFAULT NULL,
    label            VARCHAR(255)  NULL DEFAULT NULL,
    cat_ids          TEXT          NULL DEFAULT NULL,  -- JSON 数组
    created_at       DATETIME(3)   NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at       DATETIME(3)   NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    deleted_at       DATETIME(3)   NULL DEFAULT NULL,
    KEY idx_expenses_family_date (family_id, occurred_on)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 时光
CREATE TABLE IF NOT EXISTS timeline_events (
    id               VARCHAR(26)   NOT NULL PRIMARY KEY,
    created_by       VARCHAR(26)   NOT NULL,
    family_id        VARCHAR(26)   NOT NULL,
    cat_ids          TEXT          NULL DEFAULT NULL,  -- JSON 数组
    title            VARCHAR(255)  NULL DEFAULT NULL,
    body             TEXT          NULL DEFAULT NULL,
    media_id         VARCHAR(26)   NULL DEFAULT NULL,
    event_type       VARCHAR(32)   NOT NULL DEFAULT 'photo',
    occurred_on      VARCHAR(10)   NULL DEFAULT NULL,
    image_count      INT           NOT NULL DEFAULT 0,
    created_at       DATETIME(3)   NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at       DATETIME(3)   NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    deleted_at       DATETIME(3)   NULL DEFAULT NULL,
    KEY idx_timeline_family_date (family_id, occurred_on)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS cat_interactions (
    id               VARCHAR(26)   NOT NULL PRIMARY KEY,
    created_by       VARCHAR(26)   NOT NULL,
    family_id        VARCHAR(26)   NOT NULL,
    cat_ids          TEXT          NULL DEFAULT NULL,
    type             VARCHAR(32)   NULL DEFAULT NULL,
    notes            TEXT          NULL DEFAULT NULL,
    occurred_on      VARCHAR(10)   NULL DEFAULT NULL,
    created_at       DATETIME(3)   NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at       DATETIME(3)   NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    deleted_at       DATETIME(3)   NULL DEFAULT NULL,
    KEY idx_interactions_family (family_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- AI
CREATE TABLE IF NOT EXISTS ai_parse_sessions (
    id               VARCHAR(26)   NOT NULL PRIMARY KEY,
    created_by       VARCHAR(26)   NOT NULL,
    family_id        VARCHAR(26)   NOT NULL,
    original_input   TEXT          NULL DEFAULT NULL,
    status           VARCHAR(20)   NOT NULL DEFAULT 'processing',
    model            VARCHAR(120)  NULL DEFAULT NULL,
    result           LONGTEXT      NULL DEFAULT NULL,
    created_at       DATETIME(3)   NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at       DATETIME(3)   NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    deleted_at       DATETIME(3)   NULL DEFAULT NULL,
    KEY idx_ai_sessions_family (family_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS ai_analysis_reports (
    id               VARCHAR(26)   NOT NULL PRIMARY KEY,
    created_by       VARCHAR(26)   NOT NULL,
    family_id        VARCHAR(26)   NOT NULL,
    cat_id           VARCHAR(26)   NULL DEFAULT NULL,
    record_count     INT           NOT NULL DEFAULT 0,
    content          TEXT          NULL DEFAULT NULL,
    evidence         TEXT          NULL DEFAULT NULL,
    created_at       DATETIME(3)   NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at       DATETIME(3)   NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    deleted_at       DATETIME(3)   NULL DEFAULT NULL,
    KEY idx_ai_reports_family (family_id),
    KEY idx_ai_reports_cat (cat_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS ai_evidence_links (
    id               VARCHAR(26)   NOT NULL PRIMARY KEY,
    created_by       VARCHAR(26)   NOT NULL,
    report_id        VARCHAR(26)   NULL DEFAULT NULL,
    evidence_type    VARCHAR(32)   NULL DEFAULT NULL,
    evidence_id      VARCHAR(26)   NULL DEFAULT NULL,
    created_at       DATETIME(3)   NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at       DATETIME(3)   NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    deleted_at       DATETIME(3)   NULL DEFAULT NULL,
    KEY idx_evidence_report (report_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 其他
CREATE TABLE IF NOT EXISTS export_jobs (
    id               VARCHAR(26)   NOT NULL PRIMARY KEY,
    created_by       VARCHAR(26)   NOT NULL,
    family_id        VARCHAR(26)   NOT NULL,
    status           VARCHAR(20)   NOT NULL DEFAULT 'pending',
    `format`         VARCHAR(20)   NULL DEFAULT NULL,
    file_key         VARCHAR(512)  NULL DEFAULT NULL,
    created_at       DATETIME(3)   NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at       DATETIME(3)   NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    deleted_at       DATETIME(3)   NULL DEFAULT NULL,
    KEY idx_export_family (family_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS idempotency_keys (
    id               VARCHAR(26)   NOT NULL PRIMARY KEY,
    created_by       VARCHAR(26)   NOT NULL,
    family_id        VARCHAR(26)   NULL DEFAULT NULL,
    `key`            VARCHAR(255)  NOT NULL,
    result           TEXT          NULL DEFAULT NULL,
    created_at       DATETIME(3)   NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at       DATETIME(3)   NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    deleted_at       DATETIME(3)   NULL DEFAULT NULL,
    UNIQUE KEY uk_idem_key (`key`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- +migrate StatementEnd
