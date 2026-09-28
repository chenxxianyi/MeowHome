-- +migrate Up
-- +migrate StatementBegin

CREATE TABLE IF NOT EXISTS ai_agent_sessions (
    id              VARCHAR(26) NOT NULL PRIMARY KEY,
    created_by      VARCHAR(26) NOT NULL,
    family_id       VARCHAR(26) NOT NULL,
    user_id         VARCHAR(26) NOT NULL,
    title           VARCHAR(200) NULL,
    status          VARCHAR(20) NOT NULL DEFAULT 'active',
    last_message_at DATETIME(3) NULL,
    created_at      DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at      DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    deleted_at      DATETIME(3) NULL,
    KEY idx_agent_sessions_owner (family_id, user_id, last_message_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS ai_agent_messages (
    id                    VARCHAR(26) NOT NULL PRIMARY KEY,
    created_by            VARCHAR(26) NOT NULL,
    family_id             VARCHAR(26) NOT NULL,
    session_id            VARCHAR(26) NULL,
    user_id               VARCHAR(26) NULL,
    role                  VARCHAR(16) NOT NULL,
    visibility            VARCHAR(16) NOT NULL,
    cat_id                VARCHAR(26) NULL,
    type                  VARCHAR(32) NOT NULL,
    severity              VARCHAR(16) NOT NULL DEFAULT 'info',
    title                 VARCHAR(200) NOT NULL,
    body                  TEXT NOT NULL,
    evidence              TEXT NULL,
    action_suggestions    TEXT NULL,
    draft_payload         TEXT NULL,
    draft_version         INT NOT NULL DEFAULT 0,
    action_status         VARCHAR(16) NULL,
    generated_at          DATETIME(3) NOT NULL,
    draft_expires_at      DATETIME(3) NULL,
    confirmed_reminder_id VARCHAR(26) NULL,
    model                 VARCHAR(80) NOT NULL,
    disclaimer            TEXT NULL,
    dedup_key             VARCHAR(255) NULL,
    client_message_id     VARCHAR(128) NULL,
    rule_id               VARCHAR(16) NULL,
    rule_version          VARCHAR(32) NULL,
    scope                 VARCHAR(255) NULL,
    window_start          DATETIME(3) NULL,
    window_end            DATETIME(3) NULL,
    created_at            DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at            DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    deleted_at            DATETIME(3) NULL,
    UNIQUE KEY uk_agent_message_dedup (dedup_key),
    KEY idx_agent_messages_family_time (family_id, visibility, generated_at, id),
    KEY idx_agent_messages_session (session_id, generated_at, id),
    KEY idx_agent_messages_status (family_id, action_status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

SET @sql := IF((SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='reminders' AND COLUMN_NAME='scheduled_at')=0,
  'ALTER TABLE reminders ADD COLUMN scheduled_at DATETIME(3) NULL', 'DO 0');
PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;
SET @sql := IF((SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='reminders' AND COLUMN_NAME='timezone')=0,
  'ALTER TABLE reminders ADD COLUMN timezone VARCHAR(64) NULL', 'DO 0');
PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;
SET @sql := IF((SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='reminders' AND COLUMN_NAME='completed_at')=0,
  'ALTER TABLE reminders ADD COLUMN completed_at DATETIME(3) NULL', 'DO 0');
PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;

-- +migrate StatementEnd
