# URI scheme

Every artifact in rmb is addressed as `rmb://{scope}/{path}`. URIs are how the CLI, API, and agents refer to turns, atoms, scenes, and memories.

## Flat scopes

T0–T2 entities each have their **own top-level scope**. URIs describe *what* something is, not *where it lives in a folder tree*.

| Scope | Tier | URI | Session link |
|-------|------|-----|----------------|
| `sessions` | container | `rmb://sessions/<sid>` | — |
| `turns` | T0 | `rmb://turns/<uuid>` | `rmb meta` → `session_id` |
| `atoms` | T1 | `rmb://atoms/<uuid>` | `rmb meta` → `session_id` |
| `scenes` | T2 | `rmb://scenes/<uuid>` | `rmb meta` → `session_id` |
| `profile` | T3 | `rmb://profile` | cross-session |
| `preferences` | T3 | `rmb://preferences/<slug>` | cross-session |
| `entities` | T3 | `rmb://entities/<slug>` | cross-session |
| `events` | T3 | `rmb://events/<date-slug>` | cross-session |

**Eight public scopes.** `sessions` holds the conversation abstract only. Turns, atoms, and scenes are **not** nested under `rmb://sessions/<sid>/…` in the URI — even though `session_id` in the database still ties them to one session.

### Why flat?

- **Stable IDs** — turn URIs use the row's uuidv7 `id`; atom URIs use the atom's UUID. No ordinal renumbering.
- **Uniform pattern** — `rmb://turns/…`, `rmb://atoms/…`, `rmb://scenes/…` read the same way.
- **Recall handles stay short** — search results and `source_*_uris` arrays are compact.

Ownership is in **metadata** (`session_id` on `meta`), not in the path.

## Containers and `tree`

Trailing `/` means *container* (list children):

```bash
rmb tree rmb://                  # list all scopes
rmb tree rmb://sessions/<sid>/   # session abstract + flat turn/atom URIs for that session
rmb tree rmb://turns/            # list turns (global)
rmb tree rmb://atoms/            # list atoms (global)
rmb tree rmb://entities/         # list active entity memories
```

`rmb cat rmb://sessions/<sid>` prints the session **abstract** (not turns).  
`rmb cat rmb://turns/<uuid>` prints raw `messages_jsonl`.

## Short forms

The CLI accepts paths without the scheme:

```text
/turns/<uuid>     →  rmb://turns/<uuid>
sessions/<sid>/   →  rmb://sessions/<sid>/
```

Nested legacy paths such as `rmb://sessions/<sid>/turns/0` are **not** valid.

## Provenance chain

Foreign keys and URI arrays link tiers — not path nesting:

```text
memory.source_scene_uris  →  scene.source_atom_uris  →  atom.source_turn_ids  →  turn row
```

Typical drill-down after `rmb search`:

```bash
rmb meta rmb://entities/foo          # source_scene_uris
rmb cat rmb://scenes/<uuid>          # narrative + atom links in body/meta
rmb meta rmb://atoms/<uuid>          # source_turn_ids, session_id
rmb cat rmb://turns/<uuid>           # raw evidence
```

## Migrations

- **Atoms:** migration `00013_flat_atom_uris` rewrites stored `atoms.uri` and `scenes.source_atom_uris` from the old nested form.
- **Turns:** no migration — URIs were always computed at read time; new uploads emit `rmb://turns/<id>`.

## Further reading

- [The pyramid](/concept/pyramid) — what each tier means
- [Design §5](/design/l0-l3#_5-uri-scheme) — full rules (Unicode slugs, reserved syntax, T3 slug policy)
