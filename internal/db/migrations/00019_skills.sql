-- +goose Up
-- Agent Skills bundles at rmb://skills/<name> (see agentskills.io specification).
CREATE TABLE IF NOT EXISTS skills (
    id             uuid PRIMARY KEY,
    slug           text NOT NULL,
    uri            text NOT NULL,
    version        int NOT NULL DEFAULT 1,
    superseded_at  timestamptz,
    name           text NOT NULL,
    description    text NOT NULL,
    bundle_sha256  text NOT NULL,
    embedding      vector(1024),
    created_at     timestamptz NOT NULL DEFAULT now(),
    updated_at     timestamptz NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_skills_slug_active
    ON skills (slug)
    WHERE superseded_at IS NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_skills_uri_active
    ON skills (uri)
    WHERE superseded_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_skills_uri_version ON skills (uri, version);
CREATE INDEX IF NOT EXISTS idx_skills_fts
    ON skills USING gin (
        to_tsvector('english', coalesce(name, '') || ' ' || coalesce(description, ''))
    );
CREATE INDEX IF NOT EXISTS idx_skills_embedding_hnsw
    ON skills USING hnsw (embedding vector_cosine_ops);

CREATE TABLE IF NOT EXISTS skill_files (
    id             uuid PRIMARY KEY,
    skill_id       uuid NOT NULL REFERENCES skills (id) ON DELETE CASCADE,
    rel_path       text NOT NULL,
    content        text NOT NULL,
    byte_size      int NOT NULL,
    content_sha256 text NOT NULL,
    created_at     timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT skill_files_skill_path_unique UNIQUE (skill_id, rel_path)
);

CREATE INDEX IF NOT EXISTS idx_skill_files_skill_id ON skill_files (skill_id);
CREATE INDEX IF NOT EXISTS idx_skill_files_content_fts
    ON skill_files USING gin (to_tsvector('english', coalesce(content, '')));

-- +goose Down
DROP TABLE IF EXISTS skill_files;
DROP TABLE IF EXISTS skills;
