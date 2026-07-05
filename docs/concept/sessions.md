# Sessions

A **session** is one agent conversation — the Cursor or Claude Code chat identified by its session UUID.

It is **not a pyramid tier**. It is the **container** that groups turns, session-local atoms, and scenes under one `session_id`.

## What it stores

| Field | Purpose |
|-------|---------|
| `session_key` | Agent's conversation UUID (from the hook payload) |
| `abstract` | Short summary for search — refreshed after T2 scenes complete |
| `status` | Session lifecycle (`active`, …) |

There is no separate `body` column. The session's "body" is its turns, listed as flat URIs:

```bash
rmb cat rmb://sessions/<sid>           # abstract only
rmb tree rmb://sessions/<sid>/         # flat rmb://turns/… and rmb://atoms/…
rmb meta rmb://sessions/<sid>          # metadata
```

## URI

`rmb://sessions/<sid>` — the session entity.

`rmb://sessions/<sid>/` — container view (trailing slash). Turns and atoms are **not** nested in the path; see [URI scheme](/concept/uri-scheme).

## vs turns

| | Session | Turn (T0) |
|---|---------|-----------|
| Scope | Whole conversation | One user + assistant exchange |
| URI | `rmb://sessions/<sid>` | `rmb://turns/<uuid>` |
| Tier | Container (not T0–T3) | T0 — append-only evidence |
| Count | One per chat | Many per session |

## Further reading

- [Turns](/concept/turns) — T0 raw capture
- [The pyramid](/concept/pyramid) — where sessions sit relative to T0–T3
- [URI scheme](/concept/uri-scheme) — flat scopes and `tree`
