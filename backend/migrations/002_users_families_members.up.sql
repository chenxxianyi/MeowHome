-- +migrate Up
-- +migrate StatementBegin
CREATE TABLE IF NOT EXISTS refresh_tokens (
    id           VARCHAR(26)   NOT NULL PRIMARY KEY,
    created_by   VARCHAR(26)   NOT NULL,
    user_id      VARCHAR(26)   NOT NULL,
    token_hash   VARCHAR(255)  NOT NULL,
    issued_at    DATETIME(3)   NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    expires_at   DATETIME(3)   NOT NULL,
    revoked_at   DATETIME(3)   NULL DEFAULT NULL,
    created_at   DATETIME(3)   NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at   DATETIME(3)   NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    KEY idx_refresh_tokens_user (user_id),
    KEY idx_refresh_tokens_expires (expires_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS family_members (
    id           VARCHAR(26)   NOT NULL PRIMARY KEY,
    created_by   VARCHAR(26)   NOT NULL,
    family_id    VARCHAR(26)   NOT NULL,
    user_id      VARCHAR(26)   NOT NULL,
    role         VARCHAR(20)   NOT NULL DEFAULT 'member',
    timezone     VARCHAR(64)   NOT NULL DEFAULT 'Asia/Shanghai',
    created_at   DATETIME(3)   NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at   DATETIME(3)   NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    deleted_at   DATETIME(3)   NULL DEFAULT NULL,
    UNIQUE KEY uk_member_user (user_id),
    KEY idx_member_family (family_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS cats (
    id           VARCHAR(26)   NOT NULL PRIMARY KEY,
    created_by   VARCHAR(26)   NOT NULL,
    family_id    VARCHAR(26)   NOT NULL,
    name         VARCHAR(120)  NOT NULL,
    gender       VARCHAR(10)   NOT NULL DEFAULT 'unknown',
    breed        VARCHAR(120)  NULL DEFAULT NULL,
    birthday     DATE          NULL DEFAULT NULL,
    neutered     TINYINT(1)    NOT NULL DEFAULT 0,
    avatar_key   VARCHAR(255)  NULL DEFAULT NULL,
    created_at   DATETIME(3)   NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at   DATETIME(3)   NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    deleted_at   DATETIME(3)   NULL DEFAULT NULL,
    KEY idx_cats_family (family_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS cat_health_profiles (
    id               VARCHAR(26)   NOT NULL PRIMARY KEY,
    created_by       VARCHAR(26)   NOT NULL,
    cat_id           VARCHAR(26)   NOT NULL,
    diseases         TEXT          NULL DEFAULT NULL,
    allergies        TEXT          NULL DEFAULT NULL,
    contraindications TEXT         NULL DEFAULT NULL,
    created_at       DATETIME(3)   NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at       DATETIME(3)   NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    UNIQUE KEY uk_health_cat (cat_id),
    KEY idx_health_family (cat_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS media_assets (
    id               VARCHAR(26)   NOT NULL PRIMARY KEY,
    created_by       VARCHAR(26)   NOT NULL,
    family_id        VARCHAR(26)   NOT NULL,
    object_key       VARCHAR(512)  NOT NULL,
    original_name    VARCHAR(512)  NULL DEFAULT NULL,
    mime_type        VARCHAR(128)  NULL DEFAULT NULL,
    size             BIGINT        NOT NULL DEFAULT 0,
    processing_status VARCHAR(20)  NOT NULL DEFAULT 'uploaded',
    created_at       DATETIME(3)   NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at       DATETIME(3)   NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    UNIQUE KEY uk_media_key (object_key),
    KEY idx_media_family (family_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE IF NOT EXISTS audit_logs (
    id               VARCHAR(26)   NOT NULL PRIMARY KEY,
    created_by       VARCHAR(26)   NOT NULL,
    family_id        VARCHAR(26)   NULL DEFAULT NULL,
    user_id          VARCHAR(26)   NULL DEFAULT NULL,
    action           VARCHAR(64)   NOT NULL,
    resource         VARCHAR(128)  NULL DEFAULT NULL,
    detail           TEXT          NULL DEFAULT NULL,
    created_at       DATETIME(3)   NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at       DATETIME(3)   NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    KEY idx_audit_family (family_id),
    KEY idx_audit_user (user_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- +migrate StatementEnd

-- +migrate Down
-- +migrate StatementBegin
DROP TABLE IF EXISTS media_assets;
DROP TABLE IF EXISTS cat_health_profiles;
DROP TABLE IF EXISTS cats;
DROP TABLE IF EXISTS family_members;
DROP TABLE IF EXISTS refresh_tokens;
DROP TABLE IF EXISTS audit_logs;
-- +migrate StatementEnd
