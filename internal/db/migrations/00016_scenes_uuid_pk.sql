-- +goose Up
ALTER TABLE scenes ADD COLUMN id uuid;

UPDATE scenes
SET id = lower(regexp_replace(uri, '^[^:]+://scenes/', ''))::uuid;

ALTER TABLE scenes ALTER COLUMN id SET NOT NULL;

ALTER TABLE scenes DROP CONSTRAINT scenes_pkey;
ALTER TABLE scenes DROP COLUMN uri;
ALTER TABLE scenes ADD PRIMARY KEY (id);

-- +goose Down
ALTER TABLE scenes ADD COLUMN uri text;

UPDATE scenes
SET uri = 'rmb://scenes/' || lower(id::text);

ALTER TABLE scenes DROP CONSTRAINT scenes_pkey;
ALTER TABLE scenes ADD PRIMARY KEY (uri);
ALTER TABLE scenes DROP COLUMN id;
