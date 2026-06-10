-- +goose Up
-- Drop the target-less 'assert' kind: corrections must correct something.
-- See docs/corrections.md. No rows use it (the service only writes correct/forget).

ALTER TABLE assertions DROP CONSTRAINT IF EXISTS assertions_kind_check;
ALTER TABLE assertions ADD CONSTRAINT assertions_kind_check CHECK (
    kind IN ('correct', 'forget', 'split', 'alias')
);

-- +goose Down
ALTER TABLE assertions DROP CONSTRAINT IF EXISTS assertions_kind_check;
ALTER TABLE assertions ADD CONSTRAINT assertions_kind_check CHECK (
    kind IN ('correct', 'assert', 'forget', 'split', 'alias')
);
