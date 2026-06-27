-- +goose Up
-- Remove entity alias/canonical mechanism (replaced later by T1 slug registry
-- and a future grouping design). Drop tables and URI scope support data.

DROP TABLE IF EXISTS alias_candidates;
DROP TABLE IF EXISTS aliases;

-- +goose Down
-- Recreate from 00011_aliases.sql if rolling back.

CREATE TABLE IF NOT EXISTS aliases (
    id uuid PRIMARY KEY,
    uri text NOT NULL UNIQUE,
    alias_uri text NOT NULL,
    canonical_uri text NOT NULL,
    note text,
    superseded_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT aliases_no_self_alias CHECK (alias_uri <> canonical_uri)
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_aliases_alias_uri_active
    ON aliases (alias_uri)
    WHERE superseded_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_aliases_canonical_uri_active
    ON aliases (canonical_uri)
    WHERE superseded_at IS NULL;

CREATE TABLE IF NOT EXISTS alias_candidates (
    id uuid PRIMARY KEY,
    alias_uri text NOT NULL,
    canonical_uri text NOT NULL,
    similarity double precision,
    verdict text,
    rationale text,
    status text NOT NULL DEFAULT 'pending',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT alias_candidates_status_check CHECK (
        status IN ('pending', 'confirmed', 'rejected')
    )
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_alias_candidates_pair
    ON alias_candidates (
        LEAST(alias_uri, canonical_uri),
        GREATEST(alias_uri, canonical_uri)
    );
