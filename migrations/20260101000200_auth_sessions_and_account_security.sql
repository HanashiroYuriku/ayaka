-- +goose Up
ALTER TABLE users
    ADD COLUMN IF NOT EXISTS email_verified_at     TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS failed_login_attempts INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS locked_until          TIMESTAMPTZ;

-- Refresh tokens (one row per issued token, grouped by session "family").
CREATE TABLE IF NOT EXISTS refresh_tokens (
    id          VARCHAR(50)  PRIMARY KEY,
    user_id     VARCHAR(50)  NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    family_id   VARCHAR(50)  NOT NULL,
    token_hash  VARCHAR(64)  NOT NULL,
    expires_at  TIMESTAMPTZ  NOT NULL,
    revoked_at  TIMESTAMPTZ,
    replaced_by VARCHAR(50),
    user_agent  VARCHAR(255) NOT NULL DEFAULT '',
    ip          VARCHAR(64)  NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX IF NOT EXISTS uq_refresh_tokens_hash   ON refresh_tokens (token_hash);
CREATE INDEX        IF NOT EXISTS idx_refresh_tokens_user   ON refresh_tokens (user_id) WHERE revoked_at IS NULL;
CREATE INDEX        IF NOT EXISTS idx_refresh_tokens_family ON refresh_tokens (family_id);
CREATE INDEX        IF NOT EXISTS idx_refresh_tokens_expiry ON refresh_tokens (expires_at);

-- Single-use tokens sent by email (password reset, email verification).
CREATE TABLE IF NOT EXISTS user_tokens (
    id         VARCHAR(50) PRIMARY KEY,
    user_id    VARCHAR(50) NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    purpose    VARCHAR(32) NOT NULL CHECK (purpose IN ('password_reset', 'email_verification')),
    token_hash VARCHAR(64) NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    used_at    TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX IF NOT EXISTS uq_user_tokens_hash         ON user_tokens (token_hash);
CREATE INDEX        IF NOT EXISTS idx_user_tokens_user_purpose ON user_tokens (user_id, purpose, created_at DESC);
CREATE INDEX        IF NOT EXISTS idx_user_tokens_expiry       ON user_tokens (expires_at);

-- +goose Down
DROP TABLE IF EXISTS user_tokens;
DROP TABLE IF EXISTS refresh_tokens;
ALTER TABLE users
    DROP COLUMN IF EXISTS locked_until,
    DROP COLUMN IF EXISTS failed_login_attempts,
    DROP COLUMN IF EXISTS email_verified_at;
