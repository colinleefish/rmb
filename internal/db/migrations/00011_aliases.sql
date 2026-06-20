-- +goose Up
-- Entity resolution layer: human-authored aliases declaring that one memory URI
-- is the same entity as another. Append-only and orthogonal to the memories tier
-- (survives T3 re-distillation), same discipline as corrections. A row is retired
-- by setting superseded_at. See docs/aliases.md.
--
-- Topology is a flat star (depth 1): an alias points directly to a canonical, and
-- a canonical may not itself be an active alias (enforced at write time in the
-- service, not by a constraint).

CREATE TABLE IF NOT EXISTS aliases (
    id            uuid PRIMARY KEY,
    uri           text NOT NULL,
    alias_uri     text NOT NULL,
    canonical_uri text NOT NULL,
    note          text,
    superseded_at timestamptz,
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT aliases_no_self_alias CHECK (alias_uri <> canonical_uri)
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_aliases_uri ON aliases (uri);

-- One active canonical per alias: a URI cannot be an alias of two things at once.
CREATE UNIQUE INDEX IF NOT EXISTS idx_aliases_alias_uri_active
    ON aliases (alias_uri)
    WHERE superseded_at IS NULL;

-- Reverse lookup: "what aliases point to this canonical?"
CREATE INDEX IF NOT EXISTS idx_aliases_canonical_uri_active
    ON aliases (canonical_uri)
    WHERE superseded_at IS NULL;

-- Scaffold only (no worker this milestone): proposed alias pairs awaiting human
-- confirmation. A future suggest-worker writes candidates here (vector neighbors +
-- LLM judgment); confirmation is the only path that writes a live alias above.
CREATE TABLE IF NOT EXISTS alias_candidates (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    alias_uri     text NOT NULL,
    canonical_uri text NOT NULL,
    similarity    double precision,
    verdict       text,
    rationale     text,
    status        text NOT NULL DEFAULT 'pending',
    resolved_at   timestamptz,
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT alias_candidates_status_check
        CHECK (status IN ('pending', 'confirmed', 'rejected'))
);

-- Rejection memory: never re-propose the same directed pair.
CREATE UNIQUE INDEX IF NOT EXISTS idx_alias_candidates_pair
    ON alias_candidates (alias_uri, canonical_uri);

-- +goose Down
DROP TABLE IF EXISTS alias_candidates;

DROP INDEX IF EXISTS idx_aliases_canonical_uri_active;
DROP INDEX IF EXISTS idx_aliases_alias_uri_active;
DROP INDEX IF EXISTS idx_aliases_uri;
DROP TABLE IF EXISTS aliases;
