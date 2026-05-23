-- +goose Up
-- Phase B: track which turns have been processed by the T1 atom extractor.

ALTER TABLE session_turns
    ADD COLUMN IF NOT EXISTS t1_extracted_at timestamptz;

CREATE INDEX IF NOT EXISTS idx_session_turns_t1_pending
    ON session_turns (session_id, created_at)
    WHERE t1_extracted_at IS NULL;

ALTER TABLE pipeline_state
    ADD COLUMN IF NOT EXISTS t1_turns_since_advanced int NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE pipeline_state DROP COLUMN IF EXISTS t1_turns_since_advanced;

DROP INDEX IF EXISTS idx_session_turns_t1_pending;

ALTER TABLE session_turns DROP COLUMN IF EXISTS t1_extracted_at;
