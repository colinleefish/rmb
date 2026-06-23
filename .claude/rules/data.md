# Data Rules

## Identity

- All table record IDs must use UUIDv7.

## Session Modeling

- `sessions` is the aggregate root and owns `overview_text`.
- `session_turns` stores raw chunk uploads (`messages_jsonl`) only.

## Summary Lifecycle

- When new `session_turns` rows are submitted, update `sessions.overview_text`.
- `session_turns.turn_status` follows: `not_summarized -> summarizing -> summarized|failed`.
- Mark `session_turns.turn_status = summarized` only after overview update succeeds.
