# How data flows

End-to-end path from a hook firing to an agent recalling a fact.

```
Hook → hook-submit → POST /upload → session_turns (T0)
                                      ↓
                    T1 worker → atoms (T1) → T2 worker → scenes (T2)
                                      ↓                        ↓
                    T3 worker → memories (T3)          sessions.abstract
                                      ↓
                    embed worker → search API → rmb CLI / agent
```

## Phase 1 — Capture (milliseconds)

Hooks run after each agent turn. `rmb hook-submit`:

1. Parses Cursor or Claude Code payload
2. Pairs user + assistant messages
3. `POST /api/v1/sessions/:id/upload`
4. Returns immediately — **202** with turn URI

No LLM work on the hot path.

## Phase 2 — T1 extraction

**Trigger** (per session):

- Every N turns (`extraction.every_n`, default 8)
- Idle timeout (`extraction.idle_seconds`, default 600)
- Warmup ramp (2 → 4 → 8 → … on new sessions)

**Action:**

- Read pending T0 turns
- One LLM call: segment + extract atoms (4 categories)
- INSERT new atoms (default — no silent merge)
- Set `t2_status = pending`

## Phase 3 — T2 scenes

**Trigger:**

- `t2_status` pending or failed
- T1 not running
- `delay_after_t1` elapsed (default 90s)

**Action:**

- Group atoms by `scene_name`
- LLM writes scene abstract + body
- Upsert scenes; refresh `sessions.abstract`
- Set `t3_status = pending`

## Phase 4 — T3 memories

**Trigger:**

- Any session has `t3_status = pending`
- Or scheduled poll (`memory.poll_interval`, default 15m)

**Action:**

- Collect changed scenes
- Distill per category/slug into versioned memory rows
- INSERT + supersede (no in-place body updates)

## Phase 5 — Embeddings

Embed worker fills `vector(1024)` on atoms, scenes, and active memories when `abstract` changes.

## Phase 6 — Recall

| Command | What it searches |
|---------|------------------|
| `rmb search` | Hybrid vector + FTS over memories **and** scenes |

The CLI calls the server over HTTP when `RMB_URL` is configured.

## Coordination

Workers coordinate through Postgres:

- `pipeline_state` per session (`t1/t2/t3_status`, timestamps)
- `tasks` table for observable async jobs
- Advisory locks per session — safe across restarts

No in-memory scheduler required.

## Configuration knobs

| Worker | Key env vars |
|--------|----------------|
| T1 | `RMB_EXTRACTION_ENABLED`, `RMB_EXTRACTION_EVERY_N`, … |
| T2 | `RMB_SCENE_ENABLED`, `RMB_SCENE_DELAY_AFTER_T1`, … |
| T3 | `RMB_MEMORY_ENABLED`, `RMB_MEMORY_POLL_INTERVAL`, … |
| Embed | `RMB_EMBED_API_KEY`, `RMB_EMBED_ENABLED` |

See `.env.example` in the repo root.

## Further reading

- [Getting started](/guide/getting-started) — run locally + register hooks
- [Implementation plan](/reference/plan) — phase status
- [Design: pipeline sketch](/design/l0-l3#_8-pipeline-sketch)
