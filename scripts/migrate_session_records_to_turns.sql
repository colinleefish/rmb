BEGIN;

CREATE TABLE IF NOT EXISTS public.session_turns (
  id                   uuid PRIMARY KEY,
  session_id           uuid NOT NULL REFERENCES public.sessions(id) ON DELETE CASCADE,
  turn_status          text NOT NULL DEFAULT 'not_summarized'
                        CHECK (turn_status IN ('not_summarized', 'summarizing', 'summarized', 'failed')),
  summarize_started_at timestamptz,
  messages_jsonl       text NOT NULL,
  created_at           timestamptz NOT NULL DEFAULT now(),
  updated_at           timestamptz NOT NULL DEFAULT now()
);

INSERT INTO public.session_turns (
  id,
  session_id,
  turn_status,
  summarize_started_at,
  messages_jsonl,
  created_at,
  updated_at
)
SELECT
  sr.id,
  sr.session_id,
  sr.record_status AS turn_status,
  sr.summarize_started_at,
  sr.messages_jsonl,
  sr.created_at,
  sr.updated_at
FROM public.session_records sr
WHERE NOT EXISTS (
  SELECT 1 FROM public.session_turns t WHERE t.id = sr.id
)
ORDER BY sr.session_id, sr.record_index;

CREATE INDEX IF NOT EXISTS session_turns_status_created_idx
  ON public.session_turns (turn_status, session_id, created_at);

CREATE INDEX IF NOT EXISTS session_turns_session_created_idx
  ON public.session_turns (session_id, created_at);

COMMIT;
