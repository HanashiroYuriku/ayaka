-- +goose Up
CREATE TABLE IF NOT EXISTS sys_audit_logs (
    id           VARCHAR(50) PRIMARY KEY,
    user_id      VARCHAR(50) NOT NULL,
    action       VARCHAR(20) NOT NULL CHECK (action IN ('CREATE', 'UPDATE', 'DELETE')),
    table_source VARCHAR(50) NOT NULL,
    record_id    VARCHAR(50) NOT NULL,
    old_data     TEXT, -- JSON string
    new_data     TEXT, -- JSON string
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_audit_logs_record ON sys_audit_logs (table_source, record_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_user   ON sys_audit_logs (user_id, created_at DESC);

-- +goose Down
DROP TABLE IF EXISTS sys_audit_logs;
