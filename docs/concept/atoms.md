# Atoms

An **atom** is **T1**: a small structured fact extracted from one or more turns inside a session.

Atoms are the first **searchable** distillation layer — raw turns are evidence; atoms are typed facts.

## What it stores

| Field | Purpose |
|-------|---------|
| `content` | The fact text |
| `category` | `profile` \| `preferences` \| `entities` \| `events` |
| `priority` | 0–100 importance (80+ = critical) |
| `scene_name` | Grouping hint for T2 scene segmentation |
| `slug` | Optional — routes T3 memories for slug categories |
| `source_turn_ids` | Which T0 turns this came from |
| `session_id` | Owning session (`rmb meta`) |

```bash
rmb cat rmb://atoms/<uuid>
rmb meta rmb://atoms/<uuid>
```

## URI

`rmb://atoms/<uuid>` — opaque UUID. Content may change only via explicit dedup/merge, not silent worker rewrites.

## Same categories as T3

| Category | Example atom |
|----------|----------------|
| `profile` | "Colin lives in Beijing." |
| `preferences` | "Prefers short answers." |
| `entities` | "Lisa handles finance." |
| `events` | "2026-05-17: chose Postgres-only storage." |

T3 rollup routes by category into `rmb://profile`, `rmb://preferences/<slug>`, etc.

## Append-first

Default worker behavior: **insert** new atoms. Near-duplicates get tagged or downweighted — not silently merged. `events` are always insert-only.

## Further reading

- [Turns](/concept/turns) — source evidence
- [Scenes](/concept/scenes) — atoms grouped into narrative segments
- [The pyramid](/concept/pyramid) — T1 in context
