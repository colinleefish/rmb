-- +goose Up
DROP INDEX IF EXISTS idx_sessions_scope_key;

ALTER TABLE sessions
    DROP COLUMN IF EXISTS scope_key,
    DROP COLUMN IF EXISTS title;

-- +goose Down
ALTER TABLE sessions
    ADD COLUMN IF NOT EXISTS scope_key text,
    ADD COLUMN IF NOT EXISTS title text;

CREATE INDEX IF NOT EXISTS idx_sessions_scope_key ON sessions (scope_key);
