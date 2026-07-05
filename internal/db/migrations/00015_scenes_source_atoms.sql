-- +goose Up
ALTER TABLE scenes ADD COLUMN source_atoms uuid[];

UPDATE scenes
SET source_atoms = COALESCE((
    SELECT array_agg(
        lower(regexp_replace(elem, '^[^:]+://atoms/', ''))::uuid
        ORDER BY ord
    )
    FROM unnest(source_atom_uris) WITH ORDINALITY AS t(elem, ord)
    WHERE elem ~ 'atoms/'
), '{}');

ALTER TABLE scenes
    ALTER COLUMN source_atoms SET NOT NULL,
    ALTER COLUMN source_atoms SET DEFAULT '{}';

ALTER TABLE scenes DROP COLUMN source_atom_uris;

-- +goose Down
ALTER TABLE scenes ADD COLUMN source_atom_uris text[] NOT NULL DEFAULT '{}';

UPDATE scenes
SET source_atom_uris = COALESCE((
    SELECT array_agg('rmb://atoms/' || elem::text ORDER BY ord)
    FROM unnest(source_atoms) WITH ORDINALITY AS t(elem, ord)
), '{}');

ALTER TABLE scenes DROP COLUMN source_atoms;
