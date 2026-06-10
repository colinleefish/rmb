-- +goose Up
-- Human authority layer: append-only corrections that overlay distilled memory.
-- See docs/corrections.md. Not keyed/versioned by target like memories; a row is
-- retired only by setting superseded_at on that specific assertion.

CREATE TABLE IF NOT EXISTS assertions (
    id           uuid PRIMARY KEY,
    uri          text NOT NULL,
    author       text NOT NULL DEFAULT 'human',
    kind         text NOT NULL,
    target_uris  text[] NOT NULL DEFAULT '{}',
    statement    text,
    payload      jsonb,
    superseded_at timestamptz,
    created_at   timestamptz NOT NULL DEFAULT now(),
    updated_at   timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT assertions_kind_check CHECK (
        kind IN ('correct', 'assert', 'forget', 'split', 'alias')
    )
);

-- Overlay lookup: active assertions whose targets overlap a set of URIs.
CREATE INDEX IF NOT EXISTS idx_assertions_target_uris
    ON assertions USING gin (target_uris)
    WHERE superseded_at IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS idx_assertions_uri ON assertions (uri);

-- +goose Down
DROP INDEX IF EXISTS idx_assertions_uri;
DROP INDEX IF EXISTS idx_assertions_target_uris;
DROP TABLE IF EXISTS assertions;
