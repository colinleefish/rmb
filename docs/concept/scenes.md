# Scenes

A **scene** is T2: a coherent segment of work inside **one session**.

Examples: "debugging Cursor hooks", "reviewing the L0–L3 design", "planning production deploy."

## One session, several scenes

A long chat often covers multiple topics. rmb expects **a few scenes per session**, not one scene for the whole conversation and not one scene per turn.

```
1 session
  ├── many turns
  ├── many atoms
  └── few scenes        ← typically 2–5 segments
```

## How scenes are created

Scene creation is a **two-step** pipeline.

### Step 1 — T1 tags atoms with `scene_name`

When the T1 worker runs, a single LLM call does **atom extraction + scene segmentation**:

- Each atom gets a short `scene_name` label (e.g. `people`, `infra`, `decisions`)
- This is a grouping hint, not a scene row yet
- T1 sets `pipeline_state.t2_status = pending`

### Step 2 — T2 builds scene rows

The T2 worker runs per session after a short delay (`delay_after_t1`, default ~90s):

1. Load **all atoms** for the session
2. **Group by `scene_name`** (atoms without a name → `"General"`)
3. **LLM call** per batch → `display_name`, `abstract`, `body`, `atom_uris`
4. **Upsert** scene rows with stable URIs (derived from session + display name)
5. **Prune** scenes whose segment no longer exists
6. Refresh `sessions.abstract` from scene abstracts
7. Mark T3 pending for cross-session rollup

1. Hook appends a turn (T0) to the database
2. T1 worker extracts atoms and sets `scene_name`; `t2_status = pending`
3. T2 worker groups atoms into scenes and refreshes `sessions.abstract`

## What a scene stores

| Field | Purpose |
|-------|---------|
| `display_name` | Human label ("Hook debugging") |
| `abstract` | ~100 tokens for vector search |
| `body` | Markdown narrative of what happened |
| `source_atom_uris[]` | Which atoms built this scene |
| `session_id` | Always one session — scenes do not cross sessions |

## Scenes vs memories

| | Scene (T2) | Memory (T3) |
|---|------------|-------------|
| Scope | One session | Cross-session |
| Question | "What were we doing in this chat?" | "What do I know about the user long-term?" |
| Categories | No — narrative, not typed | `profile`, `preferences`, `entities`, `events` |
| In `rmb search` | Yes (`scene` tier) | Yes (`memory` tier) |

Scenes are the **bridge** between session-local facts and durable knowledge. T3 reads changed scenes and distills them into memory rows.

## Rebuild behavior

T2 **rebuilds all scenes for a session** from current atoms when new extraction runs — not incrementally per atom. Existing URIs stay stable; content refreshes.

## Further reading

- [The pyramid](/concept/pyramid) — where scenes sit in T0–T3
- [How data flows](/concept/pipeline) — worker triggers and timing
- [Design doc §6 pipeline](/design/l0-l3#_8-pipeline-sketch) — full worker pseudocode
