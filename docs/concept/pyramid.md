# The pyramid (T0–T3)

rmb organizes knowledge in four **tiers**. Think of them as increasing distillation: raw chat at the bottom, durable facts at the top.

```
                    ┌─────────────────────────────────────┐
                    │  T3 — memories (cross-session)      │
                    │  profile · preferences · entities   │
                    └──────────────────▲──────────────────┘
                                       │
              ┌────────────────────────┴─────────────────────────┐
              │  Session · rmb://sessions/<sid>                  │
              │  turns (T0) → atoms (T1) → scenes (T2)           │
              └──────────────────────────────────────────────────┘
```

## T0 — Turns

| | |
|---|---|
| **What** | One complete user + assistant exchange |
| **Table** | `session_turns` |
| **URI** | `rmb://turns/<uuid>` |
| **Source** | Hook capture (`POST …/upload`) |
| **Cardinality** | Many per session |

T0 is **append-only evidence**. Workers never rewrite turn content.

## T1 — Atoms

| | |
|---|---|
| **What** | A small typed fact extracted from turns |
| **Table** | `atoms` |
| **URI** | `rmb://atoms/<uuid>` |
| **Source** | T1 worker (LLM extraction) |
| **Cardinality** | Several per session |

Each atom carries:

- `category` — `profile` \| `preferences` \| `entities` \| `events`
- `priority` — 0–100 importance signal
- `scene_name` — which conversational segment this fact belongs to
- `source_turn_ids` — provenance back to T0

Atoms use the **same four categories** as long-term memories (T3). Extraction and storage share one taxonomy — no routing translation.

## T2 — Scenes

| | |
|---|---|
| **What** | A coherent "what we were doing" segment inside one session |
| **Table** | `scenes` |
| **URI** | `rmb://scenes/<uuid>` |
| **Source** | T2 worker (LLM narrative from grouped atoms) |
| **Cardinality** | **Few per session** (not one per turn, not one per session) |

Scenes are session-local narrative glue. See [Scenes](/concept/scenes) for how they are created.

## T3 — Memories

| | |
|---|---|
| **What** | Durable knowledge across sessions |
| **Table** | `memories` |
| **URI** | `rmb://profile`, `rmb://preferences/<slug>`, … |
| **Source** | T3 worker (LLM rollup of scenes) |
| **Cardinality** | Bounded — `profile` is a singleton; other categories have many rows keyed by slug |

See [Long-term memories](/concept/memories).

## Sessions are not a tier

The `sessions` row is the **container** for a conversation:

- Holds T0 turns and links to session-local atoms/scenes
- Gets a searchable `abstract` after T2 runs
- URI: `rmb://sessions/<sid>`

Its "body" is the chronological list of turns (`rmb tree rmb://sessions/<sid>/`).

## Recall direction

Search usually starts **high** and drills **down**:

```
memory → scenes → atoms → turns
```

`rmb meta <uri>` shows `source_*_uris` at each step so you can verify a fact against raw evidence.

## Facets (abstract vs body)

T2 scenes and T3 memories carry two text columns:

| Facet | Budget | Used for |
|-------|--------|----------|
| `abstract` | ~100 tokens | Vector embedding, quick filter |
| `body` | unbounded Markdown | Full-text search, rerank, display |

Turns and atoms are already short — they store content directly without a separate abstract.

## Further reading

- [URI scheme](/concept/uri-scheme) — flat scopes, containers, provenance
- [Entity model](/reference/entity-model) — tables, links, implementation status
- [Design: two-axis model](/design/l0-l3#_4-two-axis-model) — tiers + facets in full detail
