DROP INDEX IF EXISTS idx_sessions_scope_key;

ALTER TABLE sessions
    DROP COLUMN IF EXISTS scope_key,
    DROP COLUMN IF EXISTS title;
