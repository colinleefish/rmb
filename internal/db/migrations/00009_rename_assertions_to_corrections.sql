-- +goose Up
-- Unify terminology: the human authority layer is now "corrections" (one kind).
-- Rename the table/columns/URIs from "assertions" to "corrections" and drop the
-- dead multi-kind machinery (kind/payload): correct is the only surviving kind,
-- and split/alias leave the correction umbrella (alias becomes its own thing).
-- See docs/corrections.md.

ALTER TABLE assertions RENAME TO corrections;

ALTER INDEX IF EXISTS idx_assertions_uri RENAME TO idx_corrections_uri;
ALTER INDEX IF EXISTS idx_assertions_target_uris RENAME TO idx_corrections_target_uris;

ALTER TABLE corrections DROP CONSTRAINT IF EXISTS assertions_kind_check;
ALTER TABLE corrections DROP COLUMN IF EXISTS kind;
ALTER TABLE corrections DROP COLUMN IF EXISTS payload;

-- Rewrite each correction's own URI scheme segment.
UPDATE corrections
SET uri = replace(uri, '://assertions/', '://corrections/'),
    updated_at = now()
WHERE uri LIKE 'mem9://assertions/%';

-- Provenance column on memories, plus its stored URI contents.
ALTER TABLE memories RENAME COLUMN source_assertion_uris TO source_correction_uris;
UPDATE memories
SET source_correction_uris = ARRAY(
        SELECT replace(u, '://assertions/', '://corrections/')
        FROM unnest(source_correction_uris) AS u
    )
WHERE array_length(source_correction_uris, 1) > 0;

-- +goose Down
UPDATE memories
SET source_correction_uris = ARRAY(
        SELECT replace(u, '://corrections/', '://assertions/')
        FROM unnest(source_correction_uris) AS u
    )
WHERE array_length(source_correction_uris, 1) > 0;
ALTER TABLE memories RENAME COLUMN source_correction_uris TO source_assertion_uris;

UPDATE corrections
SET uri = replace(uri, '://corrections/', '://assertions/'),
    updated_at = now()
WHERE uri LIKE 'mem9://corrections/%';

ALTER TABLE corrections ADD COLUMN IF NOT EXISTS payload jsonb;
ALTER TABLE corrections ADD COLUMN IF NOT EXISTS kind text NOT NULL DEFAULT 'correct';
ALTER TABLE corrections DROP CONSTRAINT IF EXISTS assertions_kind_check;
ALTER TABLE corrections ADD CONSTRAINT assertions_kind_check CHECK (
    kind IN ('correct', 'split', 'alias')
);

ALTER INDEX IF EXISTS idx_corrections_uri RENAME TO idx_assertions_uri;
ALTER INDEX IF EXISTS idx_corrections_target_uris RENAME TO idx_assertions_target_uris;

ALTER TABLE corrections RENAME TO assertions;
