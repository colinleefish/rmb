# Turns

A **turn** is **T0**: one complete user + assistant exchange, captured as-is from the agent hook.

Turns are **append-only evidence**. Workers never rewrite `messages_jsonl`.

## Capture

1. Hook fires after each agent response
2. `rmb hook-submit` POSTs to `POST /api/v1/sessions/:id/upload`
3. Server inserts a `session_turns` row and returns `rmb://turns/<uuid>`

The URI uses the row's uuidv7 `id`, not an ordinal index.

## What it stores

| Field | Purpose |
|-------|---------|
| `messages_jsonl` | Raw user + assistant messages |
| `session_id` | Parent session (also in `rmb meta`) |
| `turn_status` | Legacy per-turn summarizer state (being retired) |

```bash
rmb cat rmb://turns/<uuid>    # raw messages_jsonl
rmb meta rmb://turns/<uuid>   # session_id, timestamps, etc.
```

## URI

`rmb://turns/<uuid>` — flat top-level scope. Session ownership is in metadata, not the path.

## Up the pyramid

T1 worker reads pending turns, extracts **atoms**, and records `source_turn_ids` on each atom. Recall walks down: memory → scenes → atoms → **turns** for ground truth.

## Further reading

- [Sessions](/concept/sessions) — the conversation container
- [Atoms](/concept/atoms) — facts extracted from turns
- [How data flows](/concept/pipeline) — hook → T1 timing
