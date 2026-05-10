-- MyPast crash course: pgvector + tsvector on a tiny demo table.
-- Run with: psql -h 127.0.0.1 -d mypast_dev -f scripts/crash_course.sql

\echo === 0. Reset demo table ===
DROP TABLE IF EXISTS memories;

\echo === 1. Create memories table ===
-- Embedding dim = 3 to keep numbers human-readable for learning.
CREATE TABLE memories (
  uri        TEXT PRIMARY KEY,
  content    TEXT NOT NULL,
  embedding  vector(3),
  ts_doc     tsvector
               GENERATED ALWAYS AS (to_tsvector('simple', content)) STORED,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX memories_fts_idx
  ON memories USING gin (ts_doc);
CREATE INDEX memories_embed_idx
  ON memories USING hnsw (embedding vector_cosine_ops);

\echo === 2. Insert a few memories with hand-crafted embeddings ===
-- Pretend dimensions are: [coffee, code, family]
-- Each memory leans into one or two of those topics.
INSERT INTO memories (uri, content, embedding) VALUES
  ('mypast://drinks/latte.md',
   'Drank a delicious oat milk latte at the new specialty coffee shop.',
   '[0.95, 0.05, 0.10]'),
  ('mypast://drinks/cold-brew.md',
   'Cold brew tasting at the cafe near the office.',
   '[0.90, 0.20, 0.05]'),
  ('mypast://work/refactor.md',
   'Refactored the CI pipeline in Go and removed three legacy scripts.',
   '[0.05, 0.95, 0.05]'),
  ('mypast://work/postgres.md',
   'Wrote a Postgres migration to add pgvector for semantic recall.',
   '[0.10, 0.90, 0.05]'),
  ('mypast://family/dinner.md',
   'Had dinner with my parents and talked about their trip plans.',
   '[0.05, 0.05, 0.95]');

\echo === 3. Inspect the table ===
SELECT uri, left(content, 40) AS preview, embedding FROM memories ORDER BY uri;

\echo === 4. tsvector: see what FTS actually indexes ===
SELECT uri, ts_doc FROM memories ORDER BY uri LIMIT 3;

\echo === 5. FTS recall: search for the word coffee ===
-- plainto_tsquery turns user text into a tsquery using simple parsing.
SELECT uri,
       ts_rank(ts_doc, plainto_tsquery('simple', 'coffee')) AS rank
FROM memories
WHERE ts_doc @@ plainto_tsquery('simple', 'coffee')
ORDER BY rank DESC;

\echo === 6. FTS recall: phrase-ish search for postgres migration ===
SELECT uri,
       ts_rank(ts_doc, plainto_tsquery('simple', 'postgres migration')) AS rank
FROM memories
WHERE ts_doc @@ plainto_tsquery('simple', 'postgres migration')
ORDER BY rank DESC;

\echo === 7. pgvector: cosine distance to a query vector ===
-- Query embedding leans coffee. Lower distance means more similar.
WITH q AS (SELECT '[0.92, 0.10, 0.05]'::vector AS v)
SELECT uri,
       embedding,
       embedding <=> q.v AS cosine_distance
FROM memories, q
ORDER BY cosine_distance ASC
LIMIT 3;

\echo === 8. pgvector: convert distance to similarity score ===
WITH q AS (SELECT '[0.10, 0.90, 0.05]'::vector AS v)  -- code-leaning query
SELECT uri,
       round((1 - (embedding <=> q.v))::numeric, 4) AS similarity
FROM memories, q
ORDER BY similarity DESC
LIMIT 3;

\echo === 9. Hybrid: combine semantic + FTS ===
WITH q AS (
  SELECT '[0.10, 0.90, 0.05]'::vector AS v,
         plainto_tsquery('simple', 'postgres') AS tsq
)
SELECT uri,
       round((1 - (embedding <=> q.v))::numeric, 4)        AS dense,
       round(ts_rank(ts_doc, q.tsq)::numeric, 4)           AS sparse,
       round(((1 - (embedding <=> q.v)) * 0.7
              + ts_rank(ts_doc, q.tsq) * 0.3)::numeric, 4) AS blended
FROM memories, q
ORDER BY blended DESC
LIMIT 5;
