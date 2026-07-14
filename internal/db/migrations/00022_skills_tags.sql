-- +goose Up
ALTER TABLE skills
    ADD COLUMN IF NOT EXISTS tags text[] NOT NULL DEFAULT '{}';

CREATE INDEX IF NOT EXISTS idx_skills_tags ON skills USING gin (tags);

-- +goose Down
DROP INDEX IF EXISTS idx_skills_tags;
ALTER TABLE skills DROP COLUMN IF EXISTS tags;
