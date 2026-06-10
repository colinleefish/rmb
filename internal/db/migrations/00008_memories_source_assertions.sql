-- +goose Up
-- Track which human corrections (assertions) were baked into a memory body at
-- distill time. Lets T3's provenance gate detect when active corrections change
-- (add/retract) and re-distill that bucket, even when source scenes are
-- unchanged. See docs/corrections.md (write-time distill injection).

ALTER TABLE memories
    ADD COLUMN IF NOT EXISTS source_assertion_uris text[] NOT NULL DEFAULT '{}';

-- +goose Down
ALTER TABLE memories DROP COLUMN IF EXISTS source_assertion_uris;
