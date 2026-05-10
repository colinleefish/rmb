BEGIN;

CREATE EXTENSION IF NOT EXISTS vector;

DROP TABLE IF EXISTS public.session_records;
DROP TABLE IF EXISTS public.sessions;
DROP TABLE IF EXISTS public.memories;

CREATE TABLE public.sessions (
  id            uuid PRIMARY KEY,
  session_key   text NOT NULL UNIQUE,
  scope_key     text,
  title         text,
  status        text NOT NULL DEFAULT 'active',
  overview_text text,
  created_at    timestamptz NOT NULL DEFAULT now(),
  updated_at    timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX sessions_scope_key_idx ON public.sessions (scope_key);

CREATE TABLE public.session_records (
  id                   uuid PRIMARY KEY,
  session_id           uuid NOT NULL REFERENCES public.sessions(id) ON DELETE CASCADE,
  record_index         integer NOT NULL CHECK (record_index >= 0),
  record_status        text NOT NULL DEFAULT 'not_summarized'
                        CHECK (record_status IN ('not_summarized', 'summarizing', 'summarized', 'failed')),
  summarize_started_at timestamptz,
  messages_jsonl       text NOT NULL,
  created_at           timestamptz NOT NULL DEFAULT now(),
  updated_at           timestamptz NOT NULL DEFAULT now(),
  UNIQUE (session_id, record_index)
);

CREATE INDEX session_records_status_created_idx
  ON public.session_records (record_status, session_id, created_at);

COMMIT;
