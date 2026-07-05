BEGIN;

CREATE EXTENSION IF NOT EXISTS vector;

DROP TABLE IF EXISTS public.session_turns;
DROP TABLE IF EXISTS public.session_rounds;
DROP TABLE IF EXISTS public.session_records;
DROP TABLE IF EXISTS public.sessions;
DROP TABLE IF EXISTS public.memories;

CREATE TABLE public.sessions (
  id            uuid PRIMARY KEY,
  session_key   text NOT NULL UNIQUE,
  status        text NOT NULL DEFAULT 'active',
  overview_text text,
  created_at    timestamptz NOT NULL DEFAULT now(),
  updated_at    timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE public.session_turns (
  id                   uuid PRIMARY KEY,
  session_id           uuid NOT NULL REFERENCES public.sessions(id) ON DELETE CASCADE,
  turn_status          text NOT NULL DEFAULT 'not_summarized'
                        CHECK (turn_status IN ('not_summarized', 'summarizing', 'summarized', 'failed')),
  summarize_started_at timestamptz,
  messages_jsonl       text NOT NULL,
  created_at           timestamptz NOT NULL DEFAULT now(),
  updated_at           timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX session_turns_status_created_idx
  ON public.session_turns (turn_status, session_id, created_at);

CREATE INDEX session_turns_session_created_idx
  ON public.session_turns (session_id, created_at);

COMMIT;
