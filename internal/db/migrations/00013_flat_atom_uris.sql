-- +goose Up
-- Flatten atom URIs: {scheme}://sessions/<sid>/atoms/<id> -> {scheme}://atoms/<id>
-- Entity relationships (session_id FK) are unchanged; only the address string moves.

UPDATE atoms
SET uri = regexp_replace(uri, '^(rmb|mypast|mem9)://sessions/[^/]+/atoms/', '\1://atoms/')
WHERE uri ~ '^(rmb|mypast|mem9)://sessions/[^/]+/atoms/';

UPDATE scenes
SET source_atom_uris = COALESCE((
    SELECT array_agg(
        regexp_replace(elem, '^(rmb|mypast|mem9)://sessions/[^/]+/atoms/', '\1://atoms/')
        ORDER BY ord
    )
    FROM unnest(source_atom_uris) WITH ORDINALITY AS t(elem, ord)
), '{}')
WHERE EXISTS (
    SELECT 1
    FROM unnest(source_atom_uris) AS elem
    WHERE elem ~ '^(rmb|mypast|mem9)://sessions/[^/]+/atoms/'
);

UPDATE corrections
SET target_uris = COALESCE((
    SELECT array_agg(
        regexp_replace(elem, '^(rmb|mypast|mem9)://sessions/[^/]+/atoms/', '\1://atoms/')
        ORDER BY ord
    )
    FROM unnest(target_uris) WITH ORDINALITY AS t(elem, ord)
), '{}')
WHERE EXISTS (
    SELECT 1
    FROM unnest(target_uris) AS elem
    WHERE elem ~ '^(rmb|mypast|mem9)://sessions/[^/]+/atoms/'
);

-- +goose Down
-- Irreversible without recovering the session id from the old path (not stored in uri).
