-- One-off rebrand migration: rewrite the URI scheme mypast:// -> mem9:// across
-- ALL stored data. Run directly (NOT a goose migration) against the renamed
-- mem9_db AFTER the app is stopped and the database is renamed.
--
--   sudo -u postgres psql -d mem9_db -f scripts/rebrand_scheme_mem9.sql
--
-- The substring 'mypast://' is an unambiguous URI prefix, so this sweeps every
-- text and text[] column in the public schema (uri columns, array reference
-- columns like target_uris/source_*_uris, and any body/summary prose that cites
-- a URI). Idempotent: re-running it is a no-op once no 'mypast://' remains.

BEGIN;

DO $$
DECLARE
  r record;
BEGIN
  -- Scalar text / varchar columns.
  FOR r IN
    SELECT table_name, column_name
    FROM information_schema.columns
    WHERE table_schema = 'public'
      AND data_type IN ('text', 'character varying')
  LOOP
    EXECUTE format(
      'UPDATE public.%I SET %I = replace(%I, %L, %L) WHERE %I LIKE %L',
      r.table_name, r.column_name, r.column_name,
      'mypast://', 'mem9://',
      r.column_name, '%mypast://%'
    );
  END LOOP;

  -- text[] array columns (preserve element order via WITH ORDINALITY).
  FOR r IN
    SELECT table_name, column_name
    FROM information_schema.columns
    WHERE table_schema = 'public'
      AND data_type = 'ARRAY'
      AND udt_name = '_text'
  LOOP
    EXECUTE format(
      'UPDATE public.%I SET %I = ('
      || 'SELECT array_agg(replace(x, %L, %L) ORDER BY ord) '
      || 'FROM unnest(%I) WITH ORDINALITY AS t(x, ord)) '
      || 'WHERE array_to_string(%I, %L) LIKE %L',
      r.table_name, r.column_name,
      'mypast://', 'mem9://',
      r.column_name,
      r.column_name, ',', '%mypast://%'
    );
  END LOOP;
END $$;

-- Verification: should report 0 for every column after the sweep.
SELECT 'memories.uri'            AS col, count(*) FROM memories     WHERE uri LIKE '%mypast://%'
UNION ALL SELECT 'scenes.uri',            count(*) FROM scenes       WHERE uri LIKE '%mypast://%'
UNION ALL SELECT 'atoms.uri',             count(*) FROM atoms        WHERE uri LIKE '%mypast://%'
UNION ALL SELECT 'corrections.uri',       count(*) FROM corrections  WHERE uri LIKE '%mypast://%'
UNION ALL SELECT 'aliases.uri',           count(*) FROM aliases      WHERE uri LIKE '%mypast://%'
UNION ALL SELECT 'corrections.target',    count(*) FROM corrections  WHERE array_to_string(target_uris, ',') LIKE '%mypast://%'
UNION ALL SELECT 'memories.src_scene',    count(*) FROM memories     WHERE array_to_string(source_scene_uris, ',') LIKE '%mypast://%'
UNION ALL SELECT 'memories.src_corr',     count(*) FROM memories     WHERE array_to_string(source_correction_uris, ',') LIKE '%mypast://%'
UNION ALL SELECT 'scenes.src_atom',       count(*) FROM scenes       WHERE array_to_string(source_atom_uris, ',') LIKE '%mypast://%';

COMMIT;
