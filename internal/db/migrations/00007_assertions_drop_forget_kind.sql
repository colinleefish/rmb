-- +goose Up
-- Drop the 'forget' kind: deliberate forgetting is not a human action mem9
-- supports. A wrong fact is a negative 'correct'; disuse is handled by passive
-- decay (docs/ebbinghaus-recall.md). See docs/forget-rationale.md.
-- No rows use 'forget' (the service only ever wrote correct/forget, and no
-- forget assertions were created).

ALTER TABLE assertions DROP CONSTRAINT IF EXISTS assertions_kind_check;
ALTER TABLE assertions ADD CONSTRAINT assertions_kind_check CHECK (
    kind IN ('correct', 'split', 'alias')
);

-- +goose Down
ALTER TABLE assertions DROP CONSTRAINT IF EXISTS assertions_kind_check;
ALTER TABLE assertions ADD CONSTRAINT assertions_kind_check CHECK (
    kind IN ('correct', 'forget', 'split', 'alias')
);
