# What is rmb?

**rmb** is a long-term memory store for AI-agent conversations.

It sits between your agents (Cursor, Claude Code, …) and PostgreSQL:

1. **Capture** — hooks fire after each turn; `rmb hook-submit` uploads the raw exchange.
2. **Distill** — background workers extract facts, group them into scenes, roll up cross-session memories.
3. **Recall** — `rmb search` blends vector + keyword search so the next agent can find what you already said.

The design goal is not "make the model smarter in one chat." It is **carry durable knowledge across chats and across tools** — configs, people, preferences, decisions — without you repeating yourself.

## Design principles

### Tool-agnostic at capture

Anything that can run a shell hook and POST JSON is supported. The agent does not load an SDK or change its system prompt for ingestion.

### Inspectable artifacts

Nothing hides in an opaque embedding-only blob. Every distilled item has a URI (`rmb://…`), metadata, and provenance links back to source turns.

### Simple operations

Single Go binary + PostgreSQL (+ pgvector). Workers are goroutines polling pipeline state — no separate queue cluster.

### Append-first consolidation

Workers insert new rows; they do not silently rewrite history. Human **corrections** overlay machine-distilled facts and always win. See [Corrections](/guide/corrections).

## What rmb is not

- **In-task short-term memory** — compressing the current chat context is a different problem.
- **Multi-tenant SaaS** — personal / single-user deployment today.
- **A replacement for the agent's context window** — recall supplements the model; it does not stream entire archives into every prompt.

## Read next

- [URI scheme](/concept/uri-scheme) — flat scopes and provenance
- [The pyramid (T0–T3)](/concept/pyramid) — the core mental model
- [How data flows](/concept/pipeline) — hooks → workers → recall
- [Full design doc](/design/l0-l3) — goals, URI scheme, storage layout
