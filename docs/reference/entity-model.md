# Entity model — how the pieces relate

> Short guide to what each table is and how data flows from a chat into long-term memory.
> Full design: [L0→L3 distillation](/design/l0-l3).

## One picture

```
Capture:  hook → POST /upload → sessions + session_turns (T0)

Distill:  turns → T1 → atoms → T2 → scenes → T3 → memories
          scenes also refresh sessions.abstract

Ops:      sessions ↔ pipeline_state; workers ↔ tasks
```

**Direction of truth:** conversations flow **up** the pyramid (raw → facts → scenes → long-term memory). Each layer keeps pointers back to what it came from.

## The pyramid (T0 → T3)

| Tier | Table | What it is | Parent |
|------|-------|------------|--------|
| **Session** | `sessions` | One agent conversation (Cursor / Claude session UUID) | — |
| **T0** | `session_turns` | One user + assistant exchange | `sessions` |
| **T1** | `atoms` | Small typed fact extracted from turns | `sessions` |
| **T2** | `scenes` | “What we were doing” in one session, built from atoms | `sessions` |
| **T3** | `memories` | Durable knowledge across sessions | global (not tied to one session) |

`sessions` is not a separate tier: it is the **container** for T0 turns and session-local T1 atoms / T2 scenes, plus a short `abstract` summary after T2 runs.

## Cardinality (typical)

```text
1 session
  ├── many session_turns     (one per hook upload)
  ├── many atoms             (0..N per extraction batch)
  ├── few scenes             (grouped atoms)
  └── 0..1 pipeline_state    (worker bookkeeping)

many scenes (across sessions)
  └── roll up into memories  (profile is a singleton)
```

## What each entity stores

### `sessions`

- **Identity:** `session_key` = the agent’s conversation UUID.
- **Status:** `status` (e.g. `active`).
- **Summaries:** `abstract` (new pipeline); `overview_text` (legacy summarizer, being retired).
- **URI:** `rmb://sessions/<session_key>`
- **“Body”:** not a single blob — read turns via `rmb tree rmb://sessions/<id>/`.

### `session_turns` (T0)

- **One row** = one captured Q/A pair (`messages_jsonl`).
- **Links:** `session_id` → `sessions`.
- **URI:** `rmb://turns/<uuid>` (`session_turns.id`, uuidv7). `meta` includes `session_id`.
- **Status:** `turn_status` tracks the old per-turn summarizer (`not_summarized` → …); future T1 worker uses `pipeline_state` instead.

### `atoms` (T1)

- **One row** = one structured fact, e.g. “Prefers Go monoliths”, “Colin lives in Beijing”.
- **Links:** `session_id`; `source_turn_ids[]` points at T0 rows it was extracted from.
- **Taxonomy:** `category` ∈ `profile` | `preferences` | `entities` | `events`.
- **Grouping:** `scene_name` hints which scene segment this atom belongs to (filled at extraction).
- **URI:** `rmb://atoms/<uuid>` (opaque id; content can change when deduped). `session_id` in `meta` links back to the owning session.

### `scenes` (T2)

- **One row** = one coherent segment inside a session (“debugging hooks”, “L0–L3 design”).
- **Links:** `session_id`; `source_atom_uris[]` → atoms in this scene.
- **Content:** `abstract` (~100 tokens, embedded) + `body` (Markdown, full text, FTS).
- **URI:** `rmb://scenes/<uuid>` with optional `display_name` for humans only.

### `memories` (T3)

- **Logical URI** = long-term memory in one of four categories (same names as atoms).
- **Versioning:** multiple physical rows per URI; `superseded_at IS NULL` = active (see `design-l0-l4.md` §7.1, migration `00003_memories_versioning.sql`). Workers INSERT + supersede — no in-place `body` updates.
- **Links:** `source_scene_uris[]` → scenes that contributed; not session-scoped.
- **Special cases:**
  - `profile` → singleton at `rmb://profile`
  - `preferences` / `entities` → semantic slug URIs when possible
  - `events` → dated slug, append-only (no merge at T1 or T3)
- **Content:** `abstract` + `body`, same facet idea as scenes.

### `pipeline_state` (per session)

- **One row per session** (when workers run): tracks `t1_status`, `t2_status`, `t3_status` (`idle` / `pending` / …).
- **Purpose:** coordinate async extraction without an in-memory scheduler; safe across restarts.
- **Not a memory tier** — operational only.

### `tasks`

- **Async jobs:** `kind` = `t1` | `t2` | `t3` | `backfill`; `status`, `progress`, optional `session_id`, `result_uri`.
- **Purpose:** observable two-phase commit (upload returns fast; poll task for extraction result).
- **Not a memory tier** — operational only.

## How layers connect (provenance)

| From | To | Link field |
|------|-----|------------|
| T0 turn | T1 atom | `atoms.source_turn_ids` |
| T1 atom | T2 scene | `scenes.source_atom_uris` |
| T2 scene | T3 memory | `memories.source_scene_uris` |
| T2 scenes | session summary | LLM writes `sessions.abstract` (and embedding) |

Recall can walk **down** the chain: memory → scenes → atoms → turns for evidence.

## Same categories at T1 and T3

| Category | At T1 (atom) | At T3 (memory) |
|----------|----------------|----------------|
| `profile` | Fact about you | Single merged profile |
| `preferences` | “Likes X”, “wants short answers” | One row per topic (slug) |
| `entities` | Person, company, project | One row per entity (slug) |
| `events` | Dated decision / milestone | Immutable dated record |

T3 routing is mechanical: roll up atoms/scenes **by category** into the matching memory row (merge or append per category rules).

## What exists in the repo today

| Piece | Status |
|-------|--------|
| `sessions`, `session_turns` | **Live** — hooks + upload API |
| `atoms` | **Live** — T1 worker; flat URIs (`rmb://atoms/<uuid>`) |
| `scenes` | **Live** — T2 worker |
| `memories` | **Live** — T3 worker; versioned (`00003`) |
| `pipeline_state`, `tasks` | **Live** — wired on upload |
| Observer UI | **Live** — `/ui/` lists all tables |
| T1 / T2 / T3 workers | **Live** — see [implementation plan](/reference/plan) |
| `rmb eval` | **Planned** — drift detection after rollup |

## See also

- [URI scheme](/concept/uri-scheme) — flat scopes, `tree`, provenance
- URI shapes: [design doc §5](/design/l0-l3#_5-uri-scheme)
- Worker triggers: [design doc §8–9](/design/l0-l3#_8-pipeline-sketch)
