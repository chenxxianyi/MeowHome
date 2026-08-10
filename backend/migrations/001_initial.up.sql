-- +migrate Up
-- +migrate StatementBegin
CREATE TABLE IF NOT EXISTS users (
    id           VARCHAR(26)   NOT NULL PRIMARY KEY,
    created_by   VARCHAR(26)   NOT NULL,
    email        VARCHAR(255)  NOT NULL,
    name         VARCHAR(120)  NOT NULL,
    password     VARCHAR(255)  NOT NULL,
    created_at   DATETIME(3)   NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at   DATETIME(3)   NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    deleted_at   DATETIME(3)   NULL DEFAULT NULL,
    UNIQUE KEY uk_users_email (email)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS families (
    id           VARCHAR(26)   NOT NULL PRIMARY KEY,
    created_by   VARCHAR(26)   NOT NULL,
    name         VARCHAR(120)  NOT NULL,
    timezone     VARCHAR(64)   NOT NULL DEFAULT 'Asia/Shanghai',
    currency     VARCHAR(3)    NOT NULL DEFAULT 'CNY',
    created_at   DATETIME(3)   NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at   DATETIME(3)   NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    deleted_at   DATETIME(3)   NULL DEFAULT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- +migrate StatementEnd

-- +migrate Down
-- +migrate StatementBegin
DROP TABLE IF EXISTS families;
DROP TABLE IF EXISTS users;
-- +migrate StatementEnd