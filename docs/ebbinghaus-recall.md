# Ebbinghaus recall — usage-based memory strength

> Status: plan (not yet implemented). Records how often each memory is recalled
> so useful memories stay prominent and unused ones fade. Memories are derived
> from T0 turns (append-only), so fading/forgetting a memory is always
> reversible by re-distillation — this makes the mechanism safe by construction.

## Idea

mypast currently distills and never forgets. Borrow the Ebbinghaus forgetting
curve: track each memory's **hits** (retrievals) as a usefulness signal. Two
forces shape strength:

- **Decay** — retention drops over time since last use: `R = e^(-Δt / S)`.
- **Reinforcement** — each successful recall increases strength `S`, flattening
  future decay (spaced repetition).

Forgetting is **soft**: a faded memory is ranked so low it rarely surfaces, not
deleted.

## Decisions (locked)

- **Hit signal:** weighted — appearing in `find`/`search` results = *weak* hit;
  being drilled into (`cat`) or cited = *strong* hit.
- **Forgetting action:** soft rank-decay only in v1 (no archive, no delete).
- **Storage:** append-only `memory_hits` table (matches the project's
  append-first philosophy; counts are aggregated, fully auditable).

## Phasing

The key sequencing call: **instrumentation before decay.** With zero historical
hit data, a decay function would only penalize memories by age and could hide
useful ones. So record and observe first, then calibrate the decay against real
data.

- **v1** — record hits + make them visible (`mypast memory stats`). No ranking
  change.
- **v2** — wire decay/reinforcement into `find`/`search` ranking, calibrated
  against v1 data.

## v1 scope (instrumentation)

### 1. Migration `00005_memory_hits.sql`

```sql
CREATE TABLE memory_hits (
    id uuid PRIMARY KEY,
    memory_uri text NOT NULL,        -- logical URI (stable across versions)
    kind text NOT NULL CHECK (kind IN ('weak', 'strong')),
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_memory_hits_uri ON memory_hits (memory_uri);
CREATE INDEX idx_memory_hits_created_at ON memory_hits (created_at);
```

Append-only. `memory_uri` is the stable logical URI, not a foreign key to a
specific version row (versions change; the logical handle does not).

### 2. Hit logging at single chokepoints

Logging at one shared chokepoint per signal covers both local-DB and
remote-API modes automatically:

- **Weak hit** — inside `recall.Find` / `recall.Search`, one row per returned
  `memories`-tier match (scenes excluded in v1).
- **Strong hit** — inside `inspect.catMemoryByURI` (the single path both the
  server handler and CLI-client `cat` flow through).
- Log-and-ignore on failure: a hit-logging error must never break a read.

### 3. Eval excluded for free

`mypast eval` calls the low-level `recall.FTSMemories` / `recall.VectorMemories`
directly, not `Find` / `Search`, so synthetic probe queries never inflate hits.

### 4. `mypast memory stats`

View the curve: per-memory weak/strong counts, last-hit time, and age; surface
both the hottest and coldest memories. DB-direct (operational, like
`embed status`).

### 5. Model + tests

`model.MemoryHit`, kind constants, and a unit test for the "is this URI a
memory" filter (so weak-hit logging skips scene URIs).

## Deferred to v2

- The decay/reinforcement scoring function (`R = e^(-Δt / S)`, `S` grows with
  hits) and an over-fetch-then-re-rank step in `find` / `search`.
- Archive or prune of long-cold memories.
- Remote `memory stats`.

## Risks / open questions (for v2)

- **Rare-but-critical memories** (e.g. a credential queried once a year): v2
  decay must protect `events` / `profile`, respect `priority`, and keep a score
  floor so nothing is ever fully hidden.
- **Cold start:** use `created_at` as the time reference when a memory has never
  been hit, so new memories are not faded before they can be used.
- **Hit volume:** each search logs up to `k` weak rows — fine for an append-only
  table, aggregated in `stats`.

## Open questions for v1

1. Confirm instrument-first / decay-in-v2 (vs wiring decay in the same pass,
   uncalibrated and riskier).
2. Weak hits: memories-only in v1 (recommended, since forgetting is a T3
   concept), or include scenes?
