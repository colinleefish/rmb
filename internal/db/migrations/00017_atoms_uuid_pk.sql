-- +goose Up
ALTER TABLE atoms ADD COLUMN id uuid;

UPDATE atoms
SET id = lower(regexp_replace(uri, '^[^:]+://atoms/', ''))::uuid;

ALTER TABLE atoms ALTER COLUMN id SET NOT NULL;

ALTER TABLE atoms DROP CONSTRAINT atoms_pkey;
ALTER TABLE atoms DROP COLUMN uri;
ALTER TABLE atoms ADD PRIMARY KEY (id);

-- +goose Down
ALTER TABLE atoms ADD COLUMN uri text;

UPDATE atoms
SET uri = 'rmb://atoms/' || lower(id::text);

ALTER TABLE atoms DROP CONSTRAINT atoms_pkey;
ALTER TABLE atoms ADD PRIMARY KEY (uri);
ALTER TABLE atoms DROP COLUMN id;
