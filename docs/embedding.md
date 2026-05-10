# Embedding Design Notes

## Scope

This document captures deferred embedding design decisions.
Embeddings are out of implementation scope for phase 1.
Phase 1 only implements `sessions` and `session_records`.

## Current Decision

- Do not embed `session_records` in phase 1.
- Keep `session_records` as raw JSONL source (`messages_jsonl`).
- Keep session-level rolling summary in `sessions.overview_text`.

## Candidate Embeddings Table (future)

- `id` (uuidv7, primary key)
- `owner_type` (text; example: `preferences|entities|events`)
- `owner_id` (uuidv7)
- `chunk_index` (int, default 0)
- `embedding` (vector(1024), nullable)
- `embed_model` (text, nullable)
- `created_at` (timestamptz)

## Why keep chunk_index

- Supports multi-chunk embedding for long source text.
- Preserves source order for reconstruction/ranking.
- Enables uniqueness with `(owner_type, owner_id, chunk_index)`.
- Helps incremental re-embedding for changed chunks only.

## Constraints and indexes to consider

- CHECK/enum-like constraint for `owner_type`.
- Unique index on `(owner_type, owner_id, chunk_index)`.
- Lookup index on `(owner_type, owner_id)`.
- Vector index on `embedding` when enabled.

## Open Questions

- Keep polymorphic owner reference (`owner_type`, `owner_id`) or split by owner tables.
- Default embedding dimension (`1024` now, configurable later).
- Re-embedding strategy and model version migration policy.
