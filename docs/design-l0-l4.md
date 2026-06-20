# mem9 — L0 → L3 Knowledge Distillation Design

> Status: draft. Synthesizes lessons from TencentDB Agent Memory (TDAI) and OpenViking against mem9's tool-agnostic, hook-driven capture model.
>
> See also: [`design-l0-l4.zh.md`](./design-l0-l4.zh.md) for the Mandarin version.
>
> History: an earlier revision proposed five tiers T0 → T4 with a separate `identity` artifact. After collapsing the 3-vs-8 category split into one unified 4-category taxonomy and parking the scope concept, T4 had no responsibilities that T3 `profile` did not already cover, so the pyramid was reduced to T0 → T3. The doc name is kept for continuity.

## 1. Goals

1. Turn raw agent conversations into progressively distilled, retrievable knowledge — from a single Q/A turn up to a self-describing profile of the user.
2. Stay **tool-agnostic at capture** — anything that can fire a hook and POST JSON is a supported agent. No SDK lives inside the agent.
3. Stay **inspectable** — every artifact is addressable by URI and dumpable via CLI; nothing hides in opaque format.
4. Stay **simple operationally** — single Go binary, single Postgres, no filesystem trees to babysit.

## 2. Non-goals (for this design)

- In-task short-term memory / Mermaid-style symbolic compression (TDAI has it; it's a different problem).
- Scope-keyed multi-context (work / personal / project-X). Single namespace for now; revisit when a real need shows up.
- A separate T4 "identity" tier. After taxonomy + scope simplifications, T3 `profile` already serves as the singleton self-description; an extra tier on top would carry no distinct responsibility.
- Production multi-tenant isolation with separate accounts/keys (OpenViking has it; out of scope until needed).
- At-rest encryption.
- Audit-log / rollback infrastructure (`memory_diff.json`). Not yet justified; can be added later as an append-only table.
- Filesystem-backed artifacts (`~/.mem9/`). DB columns are the source of truth.
- Anything that requires editing an agent's runtime.

## 3. Background: what we're taking from each

| Idea | Source | Why we take it |
|---|---|---|
| Four-tier pyramid T0 → T3 | TDAI's L0–L3 layering | Layering matches how we want to recall: top-level first, drill down on demand. |
| Atom records with `category / priority / scene_name / source_turn_ids` | TDAI L1 (renamed) | Structured atoms are retrievable. A prose blob is not. |
| In-call scene segmentation | TDAI | One LLM call does extract + segment. Cheap. |
| Per-node abstract / detail facets | OpenViking (trimmed) | Cheap vector recall via the small abstract; full body used for rerank and display. Overview facet considered and dropped — see §4.2. |
| URI scheme as the universal addressing layer | OpenViking | One namespace for sessions, atoms, scenes, memories. |
| 4-category unified taxonomy at T1 and T3 — `profile / preferences / entities / events` | Trimmed from OpenViking's 8-category model (user-side) | Self-documenting names; same names at extraction and distillation, no routing translation. |
| Two-phase commit (`task_id` + async extraction) | OpenViking | Hook returns fast; extraction observable via polling. |
| Hierarchical retrieval with score propagation | OpenViking | When search lands, don't go flat top-K. |
| Per-session idle-debounce + threshold + warmup ramp | TDAI | Hooks fire often; don't pay an LLM call per turn. |
| **Tool-agnostic hook capture** | mem9 | Already differentiating. Keep. |
| **Single Go binary, Postgres-native** | mem9 | Operationally simple. Keep. |

## 4. Two-axis model

Distillation lives on two orthogonal axes.

### 4.1 Vertical axis — **Tiers** (T0 → T3)

| Tier | What | Source | Cardinality |
|---|---|---|---|
| **T0 — Turn** | One raw user + assistant pair | Hook capture | Many per session |
| **T1 — Atom** | Typed structured fact (`category`, `priority`, `scene_name`, `source_turn_ids`) extracted from T0 | LLM extraction (TDAI-style, 4-category) | 0–N per session |
| **T2 — Scene** | Group of atoms forming a coherent "what we were doing" segment, rendered as Markdown | LLM aggregation of T1 within a session | Few per session |
| **T3 — Memory** | Long-term, cross-session distillation, in 4 categories | LLM rollup of T2 across sessions | Bounded (singleton `profile`; many per other category) |

**Sessions are also a facet-bearing node.** The `sessions` row itself (parent of T0 turns) carries an `abstract` column for searchability. It is not a separate tier — it is the *aggregate view of one conversation*, addressable as `mem9://sessions/<sid>`. Populated as a small post-step after T2 finishes a session's scenes. A session's "body" is its turns (queried via `mem9 tree mem9://sessions/<sid>`), so no separate body column.

### 4.2 Horizontal axis — **Facets** (per row)

Each aggregate row (T2 scenes, T3 memories) carries two columns serving different retrieval budgets:

| Facet | Column | Budget | Purpose |
|---|---|---|---|
| **abstract** | `abstract text` | ~100 tokens | Vector recall, one-line filter |
| **detail** | `body text` (Markdown) | unbounded | Full content, used for rerank and display |

Sessions carry only `abstract` (their detail is the chronological `session_turns`). T0 turns and T1 atoms do not need facets — they're already short. `session_turns.messages_jsonl` and `atoms.content` are their own content.

We considered a third middle facet (an `~1 k token` overview) modelled on OpenViking. Dropped: OpenViking's overview earns its keep as a navigation guide between directory nodes, but mem9 has no such tree — drill-down between layers is via foreign-key arrays (`source_*_uris`), not free text. Rerank can chew on the full body at mem9's expected scale. Two views per row instead of three keeps drift risk and generation cost down. We can revisit if rerank cost becomes a real bottleneck.

`abstract` is the column we embed (pgvector); `body` is FTS-indexed (`tsvector`).

## 5. URI scheme

The single addressing layer for everything.

```
mem9://{scope}/{path}
```

### 5.1 Public scopes and addressing styles

| Scope | Tier | Addressing | Example URIs |
|---|---|---|---|
| `sessions` | session / T0 / T1 | session UUID (from agent); turns ordinal; atoms UUID | `mem9://sessions/<sid>` (session abstract)<br>`mem9://sessions/<sid>/turns/<n>` (T0)<br>`mem9://sessions/<sid>/atoms/<uuid>` (T1) |
| `scenes` | T2 | UUID; optional `display_name` for readable rendering in `mem9 cat` | `mem9://scenes/<scene-uuid>` |
| `profile` | T3 | singleton; no path | `mem9://profile` |
| `preferences` | T3 | **semantic slug** (topic name); UUID fallback if no slug | `mem9://preferences/coffee`<br>`mem9://preferences/ai-tone` |
| `entities` | T3 | **semantic slug** (entity name); UUID fallback | `mem9://entities/tesla`<br>`mem9://entities/colin-mom` |
| `events` | T3 | **date-prefixed slug**; UUID fallback | `mem9://events/2026-05-17-postgres-only-decision` |

Six public top-level scopes. No scope-keying, no T4 namespace.

Internal scopes (e.g. `tasks`, `_backfill`) are reserved for the server and not addressable from the CLI by default. URI helpers expose an `allow_internal=true` flag for the server's own code path.

### 5.2 URI rules

- **Trailing slash = container.** `mem9://sessions/<sid>` is the session entity itself (`mem9 cat` prints its `abstract`). `mem9://sessions/<sid>/` is its container (`mem9 tree` lists the turns and atoms beneath it). The same convention applies to every scope.
- **Short forms.** CLI commands accept `/sessions/abc/turns/0` and `sessions/abc/turns/0`, both normalized to the canonical `mem9://...` form. Lowers typing friction; programmatic callers always emit the canonical form.
- **Unicode-safe segments.** CJK / Cyrillic / Latin extended / Hiragana / Katakana / Hangul are preserved literally (no percent-encoding); `mem9://entities/李广慧` is a valid URI. Other special characters collapse to `_`. Max 50 chars per segment.
- **Reserved future syntax.** `{namespace:key}` shapes (e.g. `{date:today}`) are reserved and rejected as invalid for now, leaving room to add path-variable templates later without breaking compatibility.
- **Forbidden slug values.** A slug must not equal a scope name (no `mem9://preferences/profile`). Sanitization rejects with an error rather than silently mangling.

### 5.3 Slugs and stable IDs

Semantic vs opaque is decided per tier based on whether the row has an intrinsic stable name:

| Row | Why this style |
|---|---|
| T0 turn (ordinal) | Turns are chronological; numbering IS the name. |
| T1 atom (UUID) | Append by default; merge only via explicit `mem9 atom merge`. Workers must not silently rewrite content. |
| T2 scene (UUID + `display_name`) | Scene rows update as atoms accumulate; URI stable via UUID, name surfaced separately for display. |
| T3 `preferences` / `entities` (slug) | Inherently named topics / entities. URI describes the *topic* or *identity*, not the current content — so the slug stays stable as the body evolves. |
| T3 `events` (date + slug) | Events are immutable by category rule; date prefix sorts naturally. |

**Source.** The LLM emits `slug` as part of the T1 extraction prompt for atoms tagged `preferences` / `entities` / `events`. T3 routes it directly into the corresponding `memories` row.

**Stability.** Slugs are stable post-creation. Renames require explicit human action (`mem9 mv <old-uri> <new-uri>`) which atomically updates the URI and all `source_*_uris` references. Auto-renaming on content drift is forbidden.

**Collisions.** `memories` carries `UNIQUE (category, slug) WHERE slug IS NOT NULL`. On conflict, the T3 worker appends `-2`, `-3`, …, and logs a warning so we can detect "the LLM keeps generating colliding slugs for genuinely distinct entities."

**Empty fallback.** If the LLM produces an empty or unusable slug, the row falls back to UUID addressing (`mem9://preferences/<uuid>`). Worst case is ugly URI, never breakage.

## 6. Memory taxonomy (T1 and T3 share these)

Four categories, same names at extraction (T1) and storage (T3). T3 routing is mechanical: aggregate atoms/scenes by category, then **append a new version** at the target memory URI (see §7 `memories` versioning) — never in-place `body` updates.

### 6.1 Consolidation stance (continuous consolidation paper)

Aligned with arXiv:2605.12978 (*Useful Memories Become Faulty When Continuously Updated by LLMs*) and [`memory-consolidation-review.zh.md`](./memory-consolidation-review.zh.md):

- **T0 (`session_turns`)** is append-only episodic evidence; no worker may rewrite it.
- **Abstract tiers (T1–T3)** are LLM-produced; default policy is **Retain**: insert new rows; consolidation (merge / overwrite `body`) must be **sparse, explicit, and observable** — not the worker default path.
- **T1 dedup merge is controlled and sparse, not the worker default.** Default: embedding top-K only tags `near_duplicate_uri` or downweights; **insert a new atom**. Merge runs only via `mem9 atom merge` (or an equivalent human-triggered task).
- **`events` (T1 and T3):** append only — no merge, delete, or in-place update.
- **T3 `profile` / slug rows:** no in-place `body` updates; each rollup **INSERT**s a new row and sets `superseded_at` on the previous active row; reads use `WHERE superseded_at IS NULL`.
- **Drift detection:** `mem9 eval` (§12) after T3 rollup compares "T0+FTS" vs "full stack"; sustained regression triggers an alert.

| Category | T1 worker | T3 worker | What it captures | Example |
|---|---|---|---|---|
| `profile` | insert atom; no merge | append new version (`mem9://profile`) | Stable identity — basics, health/taboos, core traits | "Colin lives in Beijing." "Allergic to peanuts." |
| `preferences` | insert atom; no merge | append new version per slug | Recurring "prefers X / wants X", **including AI-behavior rules** | "Prefers single-binary Go services." "Always wants short answers." |
| `entities` | insert atom; no merge | append new version per slug | Third parties: people, projects, companies, places | "Lisa from accounting prefers email." |
| `events` | **insert only** | **insert only** | Dated facts, decisions, milestones — immutable | "2026-05-17: chose Postgres-only storage." |

**`instruction` rolls into `preferences`.** "Always give short answers" is operationally a preference about AI conduct. If you ever need to separate AI-behavior rules from lifestyle preferences for retrieval (e.g. only load behavior rules into system prompt), add a `subkind text` column on `preferences` with values `lifestyle | ai-behavior` rather than a fifth category.

Priority semantics (inherited from TDAI, applies to all four categories):
- 80–100: critical (health/taboo/core trait, important event/plan, strict rule).
- 50–79: ordinary.
- < 50: weak signal, candidate to drop or downweight.
- `-1`: sentinel meaning "never drop" (use sparingly for absolute behavior rules).

## 7. Storage layout

Everything lives in Postgres. No filesystem tree.

| Table | Tier | Columns of note |
|---|---|---|
| `sessions` | session aggregate | (exists, extended) — session metadata; **adds** `abstract text` and `embedding vector(1024)` populated by a small post-T2 step |
| `session_turns` | T0 | (exists) — `messages_jsonl text` |
| **`atoms`** | T1 | `uri`, `session_id`, `category` (4-value CHECK), `priority int`, `scene_name`, `slug text?` (carried up to T3 for slug-bearing categories), `content text`, `source_turn_ids uuid[]`, `embedding vector(1024)`, timestamps |
| **`scenes`** | T2 | `uri`, `session_id`, `display_name text?`, `abstract text`, `body text`, `source_atom_uris text[]`, `embedding vector(1024)`, **`version int`**, **`superseded_at timestamptz?`**, timestamps (§7.1; worker prefers append-version over in-place body rewrite) |
| **`memories`** | T3 | **`id uuid` PK**, `uri` (stable logical URI), `category`, `slug text?`, **`version int`**, **`superseded_at timestamptz?`**, `abstract text`, `body text`, `source_scene_uris text[]`, `embedding vector(1024)`, timestamps; **active row** `UNIQUE (uri) WHERE superseded_at IS NULL`; active slug `UNIQUE (category, slug) WHERE slug IS NOT NULL AND superseded_at IS NULL` (sketch: `00003_memories_versioning.sql`) |
| **`pipeline_state`** | — | `session_id`, `t1_status`, `t1_advanced_at`, `t2_status`, `t2_advanced_at`, `t3_status`, `t3_advanced_at`, `warmup_threshold int` |
| **`tasks`** | — | `id`, `kind` (`t1`/`t2`/`t3`/`backfill`), `status`, `progress`, `result_uri`, `error`, `session_id?`, timestamps |

T0–T2 keep `uri text primary key`; **`memories` uses `id uuid` as PK** (multiple version rows per logical `uri`). `body text` columns are FTS-indexed; `embedding` columns are `vector(1024)`.

### 7.1 `memories` versioning (migrate before Phase D)

Phase A created `memories` with `uri` as PK. **Before the T3 worker (Phase D)**, apply `00003_memories_versioning.sql`:

- Add `id uuid`, `version int`, `superseded_at timestamptz`.
- Logical URI stays user-visible (`mem9 cat mem9://profile` → latest row where `superseded_at IS NULL`).
- Workers **must not** `UPDATE … SET body = …`; rollup is always `INSERT` + supersede the previous active row.
- `mem9 cat` / retrieval / `mem9 meta` default to the active row; `--version=N` / `--all-versions` for audit (CLI lands with Phase D).

`scenes` may adopt the same pattern in Phase C if eval shows drift; in-place updates are acceptable initially.

Inspection CLI:

- `mem9 cat <uri>` — print the row's `body` (or `messages_jsonl` for T0).
- `mem9 tree <uri-prefix>` — list child URIs.
- `mem9 meta <uri>` — print row metadata.

## 8. Capture flow (two-phase)

### Phase 1 — synchronous

```
POST /api/v1/sessions/:id/upload
  → insert T0 turn row
  → mark pipeline_state.t1_status = 'pending'
  → return 202 { task_id, turn_uri }
```

### Phase 2 — asynchronous (workers)

```
T1 worker (per session)
  trigger: (turn_count_since_last_t1 >= everyN)
        OR (idle for idle_seconds)
        OR (warmup: 2 → 4 → 8 → ... → everyN)
  action: read pending T0 turns
        → one LLM call: scene-segment + extract atoms (TDAI prompt, 4-category)
        → compare to existing atoms (embedding top-K): default **INSERT new atom**
            · near-duplicates: metadata (e.g. near_duplicate_uri) or downweight — **no** LLM merge
            · `events`: always INSERT; never dedup-merge
            · merge only via `mem9 atom merge <uri-a> <uri-b>` (explicit, sparse)
        → set pipeline_state.t2_status='pending'

T2 worker (per session)
  trigger: downward-only timer
           fire = max(now + delay_after_t1, last_t2 + min_interval)
           hard ceiling at last_t2 + max_interval
  action: read changed atoms for session
        → LLM call: generate scene (abstract + body)
        → default **append scene version** (or rewrite only on large atom delta; §7.1)
        → post-step: re-derive sessions.{abstract, embedding}
          from **active** scene abstracts (short LLM call or template)
        → set pipeline_state.t3_status='pending'

T3 worker (global mutex)
  trigger: any session has t3_status='pending'
        OR scheduled rollup tick
  action: collect changed scenes
        → LLM call: distill into category-specific memory rows (abstract + body)
        → **INSERT new memories row** + supersede prior active row for same URI (no UPDATE body)
        → optional: run `mem9 eval` (§12)
```

## 9. Triggers and discipline

| Tier | Trigger | Config knob |
|---|---|---|
| T1 | every-N turns + idle timer + warmup ramp | `extraction.every_n=8`, `extraction.idle_seconds=600`, `extraction.warmup=true` |
| T2 | downward-only timer (delay-after-T1, min, max) | `scene.delay_after_t1=90s`, `scene.min_interval=15m`, `scene.max_interval=1h` |
| T3 | session-pending or scheduled rollup | `memory.poll_interval=15m` |

Coordination is via Postgres status columns + advisory locks. No in-memory scheduler state required; restart is safe by construction (mem9 already does this for the current summarizer).

## 10. Retrieval (sketch, deferred implementation)

Two operations, mirroring OpenViking:

- **`find <query>`** — single-query vector recall on `abstract` embedding → rerank → return top-K MatchedContext (uri, abstract, score).
- **`search <query>`** — LLM intent analysis → 0–N TypedQueries (memory / scene / turn) → for each, hierarchical descent (vector enters at category level, recurses with score propagation `α·child + (1-α)·parent`, converges when top-K stops moving) → rerank → consolidated result.

Both expose facets: the response carries `abstract` by default; `?facet=detail` widens it to the full body. URI is the only thing needed to drill down further.

## 11. Migration from today's state

Current state:
- `sessions` table exists.
- `session_turns` table exists, holds raw turns.
- `sessions.overview_text` is a rolling prose blob produced by a 15 s ticker.
- No atoms, no scenes, no memory rows.

Migration steps:

1. **Phase A (additive)** — add new Postgres tables (`atoms`, `scenes`, `memories`, `pipeline_state`, `tasks`) and the new columns on `sessions` (`abstract`, `embedding`). Existing tables otherwise untouched. Inspection CLI lands.
2. **Phase B** — add T1 worker (§6.1 append policy); populate `atoms` from new turns. Backfill via `mem9 t1 backfill`. Disable legacy `overview_text` summarizer in production (`MEM9_SUMMARIZER_ENABLED=false`) so it does not compete with T2 `abstract`.
3. **Phase C** — add T2 worker; refresh `sessions.{abstract, embedding}` from scenes.
4. **Phase B+ (before Phase D)** — apply `00003_memories_versioning.sql`, then ship T3.
5. **Phase D** — add T3 worker; `memories` populate; run `mem9 eval` after rollups.
6. **Phase E** — drop `sessions.overview_text` and retire the summarizer worker. Session narrative = `sessions.abstract` + scene bodies via `mem9 tree`.

Each phase ships independently. Existing capture surface (`POST /sessions/:id/upload`) does not change.

## 12. Open decisions

### 12.1 Locked (consolidation policy)

Fixed before implementing T1/T3 workers — see §6.1 and [`memory-consolidation-review.zh.md`](./memory-consolidation-review.zh.md):

| Decision | Stance |
|---|---|
| T0 evidence | append-only; workers never rewrite |
| T1 dedup | default **append atom**; merge **not** worker default |
| T1/T3 `events` | insert only; no merge |
| T3 `memories` | versioned rows + `superseded_at`; no in-place `body` update |
| Paper | Zhang et al., arXiv:2605.12978v1 — continuous LLM consolidation is unreliable; keep episodic layer, consolidate sparingly |

### 12.2 Still to confirm

1. **Embedding model and dimension.** Recommendation: same OpenAI-compatible provider as extraction; **1024** dims (matches Phase A).
2. **Warmup ramp.** Recommendation: `2 → 4 → 8 → N=8` (`extraction.every_n=8`), not TDAI's 1→2→4→5.
3. **`mem9 eval` probe set.** At least five fixed recall queries; after each T3 rollup compare T0+FTS baseline vs full stack. If full stack underperforms baseline → alert / pause T3.

### 12.3 `mem9 eval` (minimal spec)

```
mem9 eval [--queries=path] [--baseline=t0-fts|full]
```

- Default queries file: `scripts/eval_queries.txt` (one query per line; optional expected URI prefix).
- Output: per-query hit@k and baseline vs full-stack delta; non-zero exit on regression.
- v1 may be manual-only; T3 worker may invoke it post-rollup.

## 13. References

- TencentDB Agent Memory (`tmp/TencentDB-Agent-Memory/`):
  - `README.md` — layered memory + symbolic memory thesis
  - `src/core/prompts/l1-extraction.ts` — atom prompt
  - `src/utils/pipeline-manager.ts` — three-timer scheduler
  - `src/core/persona/persona-generator.ts` — T3-equivalent generator
- OpenViking (`tmp/OpenViking/`):
  - `docs/en/concepts/01-architecture.md` — system overview
  - `docs/en/concepts/03-context-layers.md` — per-node L0/L1/L2 facets
  - `docs/en/concepts/04-viking-uri.md` — URI scheme
  - `docs/en/concepts/08-session.md` — two-phase commit + 8-category memory
  - `docs/en/concepts/07-retrieval.md` — hierarchical retrieval with score propagation
- Consolidation review: [`memory-consolidation-review.zh.md`](./memory-consolidation-review.zh.md) (arXiv:2605.12978)
