-- +goose Up
-- Remove the retired overview_text summarizer's columns. The live T0→T3
-- atom/scene/memory pipeline tracks progress via session_turns.t1_extracted_at
-- and pipeline_state, so turn_status/summarize_started_at/overview_text are dead.

DROP INDEX IF EXISTS idx_session_turns_status_created;

ALTER TABLE session_turns
    DROP CONSTRAINT IF EXISTS session_turns_turn_status_check;

ALTER TABLE session_turns
    DROP COLUMN IF EXISTS turn_status,
    DROP COLUMN IF EXISTS summarize_started_at;

ALTER TABLE sessions
    DROP COLUMN IF EXISTS overview_text;

-- +goose Down
ALTER TABLE sessions
    ADD COLUMN IF NOT EXISTS overview_text text;

ALTER TABLE session_turns
    ADD COLUMN IF NOT EXISTS turn_status text NOT NULL DEFAULT 'not_summarized',
    ADD COLUMN IF NOT EXISTS summarize_started_at timestamptz;

ALTER TABLE session_turns
    ADD CONSTRAINT session_turns_turn_status_check CHECK (
        turn_status IN ('not_summarized', 'summarizing', 'summarized', 'failed')
    );

CREATE INDEX IF NOT EXISTS idx_session_turns_status_created
    ON session_turns (turn_status, created_at);
