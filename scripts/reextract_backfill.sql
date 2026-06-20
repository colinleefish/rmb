-- One-off re-extraction backfill (NOT a migration; run manually with psql).
--
-- Purpose: re-run the T1->T2->T3 pipeline over existing turns so old atoms,
-- extracted before the T1 routing-prompt rewrite, are re-categorized correctly
-- (third parties -> entities, preferences -> preferences, transient noise dropped).
--
-- Safety: T0 `session_turns` is append-only evidence and is NOT touched. Atoms
-- are derived and deleted here so re-extraction does not duplicate them; scenes
-- and memories are left in place and get rebuilt/superseded by the workers.
--
-- How it works: deleting atoms + nulling t1_extracted_at + setting t1_status
-- 'pending' makes the running T1 worker re-extract; on success it sets
-- t2_status='pending', which cascades to T2 (scenes) and then T3 (memories).
--
-- Usage:
--   Scoped (validate on one session first):
--     psql "$MEM9_DB_URL" -v session_key="'<uuid>'" -f reextract_backfill.sql
--   Full (all sessions):
--     psql "$MEM9_DB_URL" -v session_key=NULL -f reextract_backfill.sql
--
-- Note: there is a window of degraded recall until the rebuild completes.

\if :{?session_key}
\else
\set session_key NULL
\endif

BEGIN;

DELETE FROM atoms
WHERE :session_key IS NULL
   OR session_id = (SELECT id FROM sessions WHERE session_key = :session_key);

UPDATE session_turns
SET t1_extracted_at = NULL
WHERE :session_key IS NULL
   OR session_id = (SELECT id FROM sessions WHERE session_key = :session_key);

UPDATE pipeline_state
SET t1_status = 'pending',
    t1_turns_since_advanced = 0,
    t1_advanced_at = NULL,
    t2_status = 'idle',
    t3_status = 'idle'
WHERE :session_key IS NULL
   OR session_id = (SELECT id FROM sessions WHERE session_key = :session_key);

COMMIT;
