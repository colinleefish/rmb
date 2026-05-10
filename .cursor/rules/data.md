# Data Rules

## Identity

- All table record IDs must use UUIDv7.

## Session Modeling

- `sessions` is the aggregate root and owns `overview_text`.
- `session_records` stores raw chunk uploads (`messages_jsonl`) only.

## Summary Lifecycle

- When a new `session_record` is submitted, update `sessions.overview_text`.
- `session_records.record_status` follows: `not_summarized -> summarizing -> summarized|failed`.
- Mark `session_records.record_status = summarized` only after overview update succeeds.
