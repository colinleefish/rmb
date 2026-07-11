# rmb

Personal long-term memory store for AI-agent conversations. Captures every
turn from supported agent tools into PostgreSQL, then summarizes them in the
background so they can be recalled across sessions.

## Status

What works today:

- HTTP API for uploading conversation turns (`POST /api/v1/sessions/:id/upload`).
- `rmb hook-submit --source=<cursor|cc|codex|pi>` for ingesting hook payloads from
  Cursor, Claude Code, Codex, and [Pi](https://pi.dev).
- **T1 extraction worker** (Phase B): async atom extraction from turns into
  `atoms` (append-only; `RMB_EXTRACTION_ENABLED=true` by default).
- **T2 scene worker** (Phase C): groups atoms into `scenes` and writes
  `sessions.abstract` (`RMB_SCENE_ENABLED=true` by default).
- **T3 memory worker** (Phase D): rolls atoms across sessions into versioned
  long-term `memories` by category/slug (`RMB_MEMORY_ENABLED=true` by default).
- Legacy summarizer (`overview_text`) is off by default (`RMB_SUMMARIZER_ENABLED=false`).

Inspection CLI: `rmb cat`, `rmb tree`, `rmb meta` (Phase A).

Web observer UI at `/ui/` when the server is running (browse sessions, turns, atoms,
scenes, memories, pipeline state, and tasks).

Production (`rmb.colinleefish.com`): Caddy in Docker terminates TLS and proxies to
`rmb` on `:8080` — see `deploy/config/Caddyfile` and `deploy/docker-compose.yml`.

**Roadmap:** [`docs/reference/plan.md`](docs/reference/plan.md) (Phase A–E). Run `make docs-dev` for the full docs site.

Planned (see `TODO.md`): storage CLI (`store/read/list/delete/search`),
embedding worker, hybrid recall, MCP wrapper.

## Architecture

```txt
agent (Cursor / Claude Code)
   │
   │ stdin JSON  ── hooks ──► rmb hook-submit --source=…
   │                                │
   │                                ▼
   │                     POST /api/v1/sessions/:id/upload
   │                                │
   ▼                                ▼
                              ┌──────────────┐
                              │   rmb     │
                              │   (Gin HTTP) │
                              └──────┬───────┘
                                     │
                                     ▼
                              ┌──────────────┐       ┌──────────────────┐
                              │  PostgreSQL  │◄──────┤ summarize.Worker │
                              │ (sessions,   │       │ (per-session     │
                              │  session_   │       │  overview merge) │
                              │  turns)     │       └──────────────────┘
                              └──────────────┘
```

Data model (PostgreSQL, [goose](https://github.com/pressly/goose) migrations on startup):

- `sessions` — one row per agent conversation, identified by `session_key`
  (the agent's session/conversation UUID). Holds the rolling `overview_text`.
- `session_turns` — one row per ingested turn. `messages_jsonl` stores the
  user + assistant pair; `turn_status` tracks summarization
  (`not_summarized → summarizing → summarized | failed`).

## Layout

```
cmd/rmb/            entry point (serve / hook-submit)
internal/
  cli/                 argv parsing
  config/              .env + ~/.rmb.conf or ~/.rmb/config.yaml + env overrides
  db/                  gorm connection + auto-migrate
  hook/                hook payload → upload adapter
    hook.go            Submit() + routing + HTTP post
    cursor.go          Cursor afterAgentResponse extraction
    claude.go          Claude Code Stop extraction (race-free)
  http/                gin router + handlers
  llm/                 OpenAI-compatible chat-completions client
  model/               gorm schemas
  server/              http.Server lifecycle
  service/
    health/            DB ping + pgvector probe
    session/           Upload service (turn insert, archive URI)
    summarize/         Background worker (claim → merge → mark summarized)
docker/                init.sql for the pgvector container
scripts/               ci.sh, deploy.sh, SQL utilities
```

## Running

### Prerequisites

- Go 1.26+
- PostgreSQL 16+ (the docker-compose target uses `pgvector/pgvector:pg18`)

### Local

```
cp .env.example .env       # edit RMB_DB_URL, RMB_LLM_API_KEY, …
make run                    # go run ./cmd/rmb serve
# or
make build && ./bin/rmb serve
```

### Docker

```
docker compose up -d        # postgres + app on :8080
```

Health check: `curl localhost:8080/healthz`.

## CI / Deploy (agents)

There is **no GitHub Actions**. CI and production deploy are **agent-driven**: run
scripts from the repo root on a machine that can SSH to production.

### CI

```bash
make ci
```

Runs `go test ./...` and a compile check (`scripts/ci.sh`). Use after code changes
and before deploy. For test-only or PR prep, stop here.

### Deploy

**Production:** https://rmb.colinleefish.com — runtime at `/app/rmb` (`.env`,
`docker-compose.yml`, `config/Caddyfile`; no git checkout on the server).

```bash
make deploy
```

Runs CI, builds and pushes a Docker image as `<branch-slug>` and
`<branch-slug>-<sha>` (production uses `:main`), SCPs runtime files to `/app/rmb`,
pulls and force-recreates the `rmb` container, and waits for `/healthz`.

**Prerequisites**

- SSH access to `root@rmb.colinleefish.com` (default key: `~/.ssh/colinleefish_ed25519`).
- Optional overrides: `scripts/deploy.env` (copy from `scripts/deploy.env.example`; gitignored).
- HTTP(S) proxy on port **1080** on your machine when pushing to GitHub from China.

**Proxy (China → GitHub)**

| Where | When | Setting |
|-------|------|---------|
| **Your machine** | `git push` / `git fetch` to GitHub | `ssproxy` then git (sets `http(s)_proxy=http://127.0.0.1:1080`) |

Example push from China:

```bash
ssproxy && git push origin main
```

### When the user says ship / deploy / release

1. `make ci` — fix any failures.
2. Commit and push to `main` with `ssproxy && git push` when GitHub is unreachable directly (ask first if unclear).
3. `make deploy`.
4. Report whether `/healthz` passed and link https://rmb.colinleefish.com

More detail: [deploy guide](docs/reference/deploy.md) (`make docs-dev` to preview).

## Configuration

Resolution order (later wins):

1. Defaults baked into `internal/config`.
2. `~/.rmb.conf` (flat `KEY=value`) or `~/.rmb/config.yaml` (structured; path overridable via `RMB_CONFIG`).
3. `.env` in the working directory.
4. Process environment.

Key variables (see `.env.example` for the full list):

| Var                                   | Default                                                            |
|---------------------------------------|--------------------------------------------------------------------|
| `RMB_DB_URL`                       | `postgres://admin@127.0.0.1:5432/rmb_db?sslmode=disable`       |
| `RMB_ADDR`                         | `:8080`                                                            |
| `RMB_LLM_API_BASE` / `_API_KEY` / `_MODEL` | OpenAI-compatible endpoint used by the summarizer            |
| `RMB_SUMMARIZER_ENABLED`           | `true`                                                             |
| `RMB_SUMMARIZER_POLL_INTERVAL`     | `15s`                                                              |
| `RMB_SUMMARIZER_MAX_TURNS_PER_MERGE` | `4`                                                              |

For `hook-submit`, the target API URL is read from `RMB_URL` or
`~/.rmb.conf` or `~/.rmb/config.yaml` (`client.url` / `RMB_URL=`), defaulting to `http://127.0.0.1:8080`.

## Hook Integration

Each agent gets a single hook entry. `--source` is mandatory; mismatched
payloads exit non-zero.

`~/.cursor/hooks.json`:

```json
{
  "hooks": {
    "afterAgentResponse": [
      {
        "command": "/path/to/rmb/bin/rmb hook-submit --source=cursor",
        "timeout": 5
      }
    ]
  },
  "version": 1
}
```

`~/.claude/settings.json`:

```json
{
  "hooks": {
    "Stop": [
      {
        "hooks": [
          { "type": "command", "command": "/path/to/rmb/bin/rmb hook-submit --source=cc" }
        ]
      }
    ]
  }
}
```

Design notes for the extraction logic (status filter on Cursor, race-free
pairing on Claude Code) live in `internal/hook/cursor.go` and
`internal/hook/claude.go` as package-level comments.

### Pi

Pi has no shell hooks. Install the extension from `integrations/pi/` — it
listens for `agent_settled` and pipes a JSON payload to
`rmb hook-submit --source=pi`. See [`integrations/pi/README.md`](integrations/pi/README.md).

## API

`POST /api/v1/sessions/:session_id/upload`

```json
{
  "started_at":"optional RFC3339",
  "messages":  [
    {"role": "user",      "content": "..."},
    {"role": "assistant", "content": "..."}
  ]
}
```

`session_id` must be a UUID. Each request appends one `session_turns` row
to the session (creating the session row on first upload). Response includes
the `rmb://turns/<uuid>` URI for the new turn.

`GET /healthz` — DB ping + `pg_extension` lookup for `vector`.

### Recall API

`GET /api/v1/find?q=<query>&k=<n>` — vector recall over long-term memories.

`GET /api/v1/search?q=<query>&k=<n>` — hybrid (vector + FTS) recall across
memories and scenes, fused with reciprocal rank fusion.

Both return `{ "items": [ { "uri", "tier", "rank", "snippet" } ] }` and require
the server to have an embedding client configured (`RMB_EMBED_API_KEY`).

`GET /api/v1/inspect/{cat,tree,meta}?uri=<uri>` — text output of the inspection
commands (used by the CLI's remote mode).

## CLI: local vs remote

Operational CLI commands (`t1/t2/t3 backfill`, `embed status`, `eval`) talk
**directly to the database** (`RMB_DB_URL`) — run them on the server or
against a local dev DB.

`find`, `search`, `cat`, `tree`, and `meta` are **dual-mode**:

- **Remote (client):** if `RMB_URL` is set (env, `~/.rmb.conf`, or `~/.rmb/config.yaml`), they
  call the server's recall API over HTTP with basic auth — run them from your
  laptop against production.
- **Local:** otherwise they query the database directly and embed the query
  locally via `RMB_EMBED_*`.

`hook-submit` is always an HTTP client (posts to `RMB_URL`).

## Testing

Same as CI (preferred):

```bash
make ci
```

Or directly:

```bash
go test ./...
```

## License

TBD.
