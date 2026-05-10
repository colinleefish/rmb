# MyPast — Design

A small, personal memory store.
It replaces the OpenViking deployment for one user (me) without inheriting
its operational weight.

## Why

The current OpenViking instance works, but the operational tax is high:
AGFS object storage, async semantic plus embedding queues, lifecycle locks,
subtree locks, circuit breakers, and vendor patches.
Around 200 markdown files do not justify that surface area.
Each bug we hit (orphan locks, dropped `dimensions` param, and
normalization-rule changes producing duplicate URIs) lives in concurrency
machinery I never need.

MyPast is a deliberate downscale:
one user, one Postgres, one process, sync writes, and one embedding provider.

## Goals

1. Functionally cover what I actually use today:
   store, read, list, semantic recall, and hierarchical browsing.
2. Operate with no hidden state.
   The whole system is a single Postgres database.
3. Build incrementally.
   Each slice is end-to-end useful before the next.
4. Be cheap to throw away if a better idea comes along.

## Non-goals

- Multi-tenant, multi-account, and multi-agent isolation.
- Pluggable storage backends.
- Pluggable vector backends.
- Skill packs, session compression, dedup, and memory updaters.
- High write throughput.
  Single human writer, around tens of writes per day.

## Architecture

```text
  Cursor / Claude / CLI
        |  MCP (stdio)
        v
   +-----------------+
   |   mypast        |
   |   (one process) |
   |                 |
   |  store / read   |
   |  list / tree    |
   |  recall (sem)   |
   |  recall (fts)   |
   +--------+--------+
            |  SQL
            v
   +-----------------+
  |   Postgres 18   |
   |   + pgvector    |
   |   + tsvector    |  (built in)
   +-----------------+
            |  HTTPS
            v
   ZhipuAI embedding-3 (1024d)
```

One process, one database, one external dependency.

## Data model

```sql
CREATE EXTENSION vector;

-- Embedding dimension is configurable at install time via $MYPAST_EMBED_DIM.
-- The migration substitutes the value into the DDL. Default: 1024.
-- Changing it later requires re-embedding all rows.
CREATE TABLE memories (
  uri          TEXT PRIMARY KEY,           -- mypast://events/2026/05/08/foo.md
  category     TEXT NOT NULL,              -- events / profile / preferences / ...
  parent_uri   TEXT,                       -- mypast://events/2026/05/08
  content      TEXT NOT NULL,
  abstract     TEXT,                       -- L0, optional, generated later
  embedding    vector(:embed_dim),         -- nullable; backfilled async
  embed_model  TEXT,                       -- e.g. "zhipu/embedding-3@1024"
  ts_doc       tsvector
                 GENERATED ALWAYS AS (to_tsvector('simple', content)) STORED,
  created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX memories_embed_idx
  ON memories USING hnsw (embedding vector_cosine_ops);
CREATE INDEX memories_fts_idx
  ON memories USING gin (ts_doc);
CREATE INDEX memories_parent
  ON memories (parent_uri);
CREATE INDEX memories_category
  ON memories (category, updated_at DESC);
```

That is the entire storage layer.

## URI scheme

`mypast://<category>/<path...>` is flatter than
`viking://user/liguanghui/memories/...`.
It is single-user, so no scope or owner segments.

Examples:

- `mypast://events/2026/05/08/jenkins-incident.md`
- `mypast://profile.md`
- `mypast://preferences/coding-style.md`

## API surface

No MCP server in Slice 1.
The first cut is **a Go library plus a CLI**.
That is enough to dogfood, migrate from OpenViking, and write tests against.
MCP is a thin shell we add later once the surface is stable.

- `Store` (Slice 1): `mypast store <uri> < file.md`
  - Insert or replace.
  - Sync write.
- `Read` (Slice 1): `mypast read <uri>`
  - Read by URI.
- `List` (Slice 1): `mypast list <uri-prefix>`
  - List by URI prefix.
- `Delete` (Slice 1): `mypast delete <uri>`
  - Real delete.
  - No phantoms.
- `Recall` (Slice 2): `mypast recall "query" [--limit N]`
  - Hybrid recall: semantic + FTS, weighted blend.
- `Abstract` (Slice 3): `mypast abstract <uri>`
  - Read L0 abstract.

No tree, no observer, and no stats.
Use `psql` for that.

**MCP** (Slice 5, deferred):
wrap the same library in a stdio MCP server when I want Cursor or Claude to
call MyPast directly.
It does not block anything else.

## Slices

### Slice 1 — Local store, no vectors

- Postgres plus schema above (skip `embedding` index for now).
- `mp_store`, `mp_read`, `mp_list`, and `mp_delete`.
- Migration script:
  read every URI from OpenViking via its API and insert into `memories`.
- Acceptance:
  I can replace `openviking_store`, `openviking_read`, and `openviking_list`
  in my Cursor MCP config with MyPast equivalents and lose nothing.

### Slice 2 — Semantic + FTS recall

- Background worker embeds any row with `embedding IS NULL`.
  It is single-process, a simple loop, and has no queue.
- `mp_recall(query, limit)`:

  ```sql
  WITH q AS (
    SELECT $emb::vector AS v,
           plainto_tsquery('simple', $query) AS tsq
  )
  SELECT uri,
         abstract,
         (1 - (embedding <=> q.v)) AS dense,
         ts_rank(ts_doc, q.tsq) AS sparse,
         (1 - (embedding <=> q.v)) * 0.7
           + ts_rank(ts_doc, q.tsq) * 0.3 AS score
  FROM memories, q
  WHERE category = ANY($categories)
  ORDER BY score DESC LIMIT $limit;
  ```

- Acceptance:
  recall finds the right doc for the same five sanity queries used against
  OpenViking.

### Slice 3 — L0 abstracts

- `abstracts` worker:
  when a row has `abstract IS NULL`, call LLM and write back.
- `abstract` shows up in `mp_recall` and `mp_list` results.
- Acceptance:
  recall results are skimmable without opening files.

### Slice 4 — Quality of life (deferred until needed)

- `mp_tree` for nicer browsing.
- Per-category default ranking weights.
- Better Chinese tokenization (`pgroonga` or `zhparser`) only if simple
  tokenizer hurts in practice.
- Backups with `pg_dump` cron.

## Concurrency model

- One writer (me, via Cursor or CLI).
  There is no write contention.
- Embedding worker is a background goroutine in the same process.
  It picks up `embedding IS NULL` rows, embeds, and updates.
  If it dies, restart picks up where it left off.
  No queue, no lock, and no DAG.
- All writes are a single transaction:
  insert or update plus the `tsvector` auto-computed by Postgres in the same
  statement.
- Embedding failures do not block writes.
  The row exists.
  Recall just will not find it semantically until the embedder catches up.

This is the entire story.
No lifecycle locks, no circuit breakers, and no semantic refresh DAG.

## Tech choices

- Language
  - Decision: Go.
  - Why: Single binary, no Python virtual-env pain, and fits my stack.
- Database
  - Decision: Postgres 18 + pgvector + built-in FTS.
  - Why: One thing to operate with transactional consistency.
- Embeddings
  - Decision: ZhipuAI `embedding-3` with configurable dim.
  - Why: Already paid for.
    Default is 1024 and it is settable in config.
- L0 LLM
  - Decision: ZhipuAI `glm-4.7`.
  - Why: Same vendor and one API key.
- MCP transport
  - Decision: deferred to Slice 5.
  - Why: CLI plus library is enough until the surface stabilizes.
- Deployment
  - Decision: local Postgres plus `mypastd` on `mem.colinleefish.com`.
  - Why: Same host I already run, with the same SOCKS proxy reuse.

## Out-of-scope decisions deferred

- Cloud-hosted variant.
- Multi-device sync (only one machine reads and writes today).
- Web UI (`psql` is the UI, MCP is the agent UI).

## Configuration

A single config file at `~/.config/mypast/config.toml`
(or `$MYPAST_CONFIG`):

```toml
[db]
url = "postgres://colin@localhost:5432/mypast?sslmode=disable"

[embedding]
provider   = "zhipu"
api_base   = "https://open.bigmodel.cn/api/paas/v4"
api_key    = "..."
model      = "embedding-3"
dim        = 1024            # 256 / 512 / 1024 / 2048; must match table at install
batch_size = 16

[llm]                         # used by Slice 3 abstracts
provider   = "zhipu"
model      = "glm-4.7"
api_key    = "..."
```

Switching `dim` after install requires:
`ALTER TABLE memories ALTER COLUMN embedding TYPE vector(N)`,
plus a full re-embed pass.
Bake a `mypast reembed` subcommand for that.

## Open questions

1. Migration:
   rewrite `viking://` to `mypast://` at migration time, or keep aliases?
   Lean: rewrite.
   It is one-time pain, then cleaner forever.
2. Embedding dim:
   1024 vs 2048 default?
   Lean: 1024.
   It is half the disk, with indistinguishable recall quality at this scale.
   It is configurable either way.

## Inspirations / non-inspirations

- **OpenViking**:
  direct ancestor, and this project is its 80/20.
- **Memos** (`usememos`):
  UI-driven personal memo app.
  Different shape: I want LLM-native, not human-UI-native.
- **Letta / mem0**:
  agent-memory frameworks.
  Too opinionated about agent loops.
  I just want a store.
