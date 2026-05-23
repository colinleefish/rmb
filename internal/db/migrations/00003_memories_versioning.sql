-- +goose Up
-- Apply before Phase D (T3 worker). See docs/design-l0-l4.md §7.1.
--
-- Phase A used uri as PK on memories. Versioning requires:
--   - multiple rows per logical URI
--   - exactly one active row per URI (superseded_at IS NULL)
--   - no in-place body updates from workers

ALTER TABLE memories ADD COLUMN IF NOT EXISTS id uuid;
UPDATE memories SET id = gen_random_uuid() WHERE id IS NULL;
ALTER TABLE memories ALTER COLUMN id SET NOT NULL;

ALTER TABLE memories
    ADD COLUMN IF NOT EXISTS version int NOT NULL DEFAULT 1,
    ADD COLUMN IF NOT EXISTS superseded_at timestamptz;

UPDATE memories SET version = 1 WHERE version IS NULL;

ALTER TABLE memories DROP CONSTRAINT IF EXISTS memories_pkey;
ALTER TABLE memories ADD PRIMARY KEY (id);

DROP INDEX IF EXISTS idx_memories_category_slug;
CREATE UNIQUE INDEX IF NOT EXISTS idx_memories_uri_active
    ON memories (uri)
    WHERE superseded_at IS NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_memories_category_slug_active
    ON memories (category, slug)
    WHERE slug IS NOT NULL AND superseded_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_memories_uri_version ON memories (uri, version);
CREATE INDEX IF NOT EXISTS idx_memories_superseded_at ON memories (superseded_at);

-- +goose Down
DROP INDEX IF EXISTS idx_memories_superseded_at;
DROP INDEX IF EXISTS idx_memories_uri_version;
DROP INDEX IF EXISTS idx_memories_category_slug_active;
DROP INDEX IF EXISTS idx_memories_uri_active;

ALTER TABLE memories DROP CONSTRAINT IF EXISTS memories_pkey;
ALTER TABLE memories ADD PRIMARY KEY (uri);

CREATE UNIQUE INDEX IF NOT EXISTS idx_memories_category_slug
    ON memories (category, slug)
    WHERE slug IS NOT NULL;

ALTER TABLE memories
    DROP COLUMN IF EXISTS superseded_at,
    DROP COLUMN IF EXISTS version,
    DROP COLUMN IF EXISTS id;
