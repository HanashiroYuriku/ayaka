-- +goose Up
CREATE TABLE IF NOT EXISTS users (
    id           VARCHAR(50)  PRIMARY KEY,
    username     VARCHAR(20)  NOT NULL,
    email        VARCHAR(100) NOT NULL,
    display_name VARCHAR(100) NOT NULL DEFAULT '',
    password     TEXT         NOT NULL,
    role         VARCHAR(20)  NOT NULL DEFAULT 'user' CHECK (role IN ('admin', 'user')),
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    deleted_at   TIMESTAMPTZ
);

-- Partial unique indexes: a soft-deleted user does not block re-registration.
CREATE UNIQUE INDEX IF NOT EXISTS uq_users_username ON users (username) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX IF NOT EXISTS uq_users_email    ON users (email)    WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_users_deleted_at     ON users (deleted_at);
CREATE INDEX IF NOT EXISTS idx_users_created_at     ON users (created_at DESC);

-- +goose Down
DROP TABLE IF EXISTS users;
