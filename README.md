# mem9

Personal long-term memory store for AI-agent conversations. Captures every
turn from supported agent tools into PostgreSQL, then summarizes them in the
background so they can be recalled across sessions.

## Status

What works today:

- HTTP API for uploading conversation turns (`POST /api/v1/sessions/:id/upload`).
- `mem9 hook-submit --source=<cursor|cc>` for ingesting hook payloads from
  Cursor and Claude Code, with race-free user/assistant pairing.
- **T1 extraction worker** (Phase B): async atom extraction from turns into
  `atoms` (append-only; `MEM9_EXTRACTION_ENABLED=true` by default).
- **T2 scene worker** (Phase C): groups atoms into `scenes` and writes
  `sessions.abstract` (`MEM9_SCENE_ENABLED=true` by default).
- **T3 memory worker** (Phase D): rolls atoms across sessions into versioned
  long-term `memories` by category/slug (`MEM9_MEMORY_ENABLED=true` by default).
- Legacy summarizer (`overview_text`) is off by default (`MEM9_SUMMARIZER_ENABLED=false`).

Inspection CLI: `mem9 cat`, `mem9 tree`, `mem9 meta` (Phase A).

Web observer UI at `/ui/` when the server is running (browse sessions, turns, atoms,
scenes, memories, pipeline state, and tasks).

Production (`mem.colinleefish.com`): Caddy in Docker terminates TLS and proxies to
`mem9` on `:8080` — see `deploy/Caddyfile` and `docker-compose.prod.yml`.

**Roadmap:** [`docs/plan.md`](docs/plan.md) (Phase A–E, current status, next steps).

Planned (see `TODO.md`): storage CLI (`store/read/list/delete/search`),
embedding worker, hybrid recall, MCP wrapper.

## Architecture

```txt
agent (Cursor / Claude Code)
   │
   │ stdin JSON  ── hooks ──► mem9 hook-submit --source=…
   │                                │
   │                                ▼
   │                     POST /api/v1/sessions/:id/upload
   │                                │
   ▼                                ▼
                              ┌──────────────┐
                              │   mem9     │
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
cmd/mem9/            entry point (serve / hook-submit)
internal/
  cli/                 argv parsing
  config/              .env + ~/.config/mem9/config.toml + env overrides
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
cp .env.example .env       # edit MEM9_DB_URL, MEM9_LLM_API_KEY, …
make run                    # go run ./cmd/mem9 serve
# or
make build && ./bin/mem9 serve
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

**Production:** https://mem.colinleefish.com — app at `/opt/mem9`, Caddy + Docker
(`docker-compose.prod.yml`).

```bash
make deploy
```

Runs CI, then SSHs to the server, `git reset --hard` to `main`, `docker compose … up -d --build`, and waits for `/healthz`.

**Prerequisites**

- Latest `main` pushed to GitHub (server pulls via deploy key).
- SSH access to `root@mem.colinleefish.com` (default key: `~/.ssh/colinleefish_ed25519`).
- Optional overrides: `scripts/deploy.env` (copy from `scripts/deploy.env.example`; gitignored).
- HTTP(S) proxy on port **1080** (local machine and production server) for GitHub access from China.

**Proxy (China → GitHub)**

| Where | When | Setting |
|-------|------|---------|
| **Your machine** | `git push` / `git fetch` to GitHub | `ssproxy` then git (sets `http(s)_proxy=http://127.0.0.1:1080`) |
| **Production server** | `git fetch` during `make deploy` | `git -c https.proxy=http://localhost:1080 … fetch` (proxy not exported globally; `healthz` curl bypasses proxy) |

Example push from China:

```bash
ssproxy && git push origin main
```

### When the user says ship / deploy / release

1. `make ci` — fix any failures.
2. Commit and push to `main` with `ssproxy && git push` when GitHub is unreachable directly (ask first if unclear).
3. `make deploy`.
4. Report whether `/healthz` passed and link https://mem.colinleefish.com

More detail: [`docs/deploy.md`](docs/deploy.md).

## Configuration

Resolution order (later wins):

1. Defaults baked into `internal/config`.
2. `~/.config/mem9/config.toml` (path overridable via `MEM9_CONFIG`).
3. `.env` in the working directory.
4. Process environment.

Key variables (see `.env.example` for the full list):

| Var                                   | Default                                                            |
|---------------------------------------|--------------------------------------------------------------------|
| `MEM9_DB_URL`                       | `postgres://admin@127.0.0.1:5432/mem9_db?sslmode=disable`       |
| `MEM9_ADDR`                         | `:8080`                                                            |
| `MEM9_LLM_API_BASE` / `_API_KEY` / `_MODEL` | OpenAI-compatible endpoint used by the summarizer            |
| `MEM9_SUMMARIZER_ENABLED`           | `true`                                                             |
| `MEM9_SUMMARIZER_POLL_INTERVAL`     | `15s`                                                              |
| `MEM9_SUMMARIZER_MAX_TURNS_PER_MERGE` | `4`                                                              |

For `hook-submit`, the target API URL is read from `MEM9_URL` or
`~/.mem9.conf` (key `MEM9_URL=`), defaulting to `http://127.0.0.1:8080`.
To mirror turns to **local + production**, register **two hook entries** — see [`docs/hooks-dual.md`](docs/hooks-dual.md).

## Hook Integration

Each agent gets a single hook entry. `--source` is mandatory; mismatched
payloads exit non-zero.

`~/.cursor/hooks.json`:

```json
{
  "hooks": {
    "afterAgentResponse": [
      {
        "command": "/path/to/mem9/bin/mem9 hook-submit --source=cursor",
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
          { "type": "command", "command": "/path/to/mem9/bin/mem9 hook-submit --source=cc" }
        ]
      }
    ]
  }
}
```

Design notes for the extraction logic (status filter on Cursor, race-free
pairing on Claude Code) live in `internal/hook/cursor.go` and
`internal/hook/claude.go` as package-level comments.

## API

`POST /api/v1/sessions/:session_id/upload`

```json
{
  "scope_key": "optional",
  "title":     "optional",
  "started_at":"optional RFC3339",
  "messages":  [
    {"role": "user",      "content": "..."},
    {"role": "assistant", "content": "..."}
  ]
}
```

`session_id` must be a UUID. Each request appends one `session_turns` row
to the session (creating the session row on first upload). Response includes
the `mem9://sessions/<id>/turns/<n>` URI for the new turn.

`GET /healthz` — DB ping + `pg_extension` lookup for `vector`.

### Recall API

`GET /api/v1/find?q=<query>&k=<n>` — vector recall over long-term memories.

`GET /api/v1/search?q=<query>&k=<n>` — hybrid (vector + FTS) recall across
memories and scenes, fused with reciprocal rank fusion.

Both return `{ "items": [ { "uri", "tier", "rank", "snippet" } ] }` and require
the server to have an embedding client configured (`MEM9_EMBED_API_KEY`).

`GET /api/v1/inspect/{cat,tree,meta}?uri=<uri>` — text output of the inspection
commands (used by the CLI's remote mode).

## CLI: local vs remote

Operational CLI commands (`t1/t2/t3 backfill`, `embed status`, `eval`) talk
**directly to the database** (`MEM9_DB_URL`) — run them on the server or
against a local dev DB.

`find`, `search`, `cat`, `tree`, and `meta` are **dual-mode**:

- **Remote (client):** if `MEM9_URL` is set (env or `~/.mem9.conf`), they
  call the server's recall API over HTTP with basic auth — run them from your
  laptop against production.
- **Local:** otherwise they query the database directly and embed the query
  locally via `MEM9_EMBED_*`.

`hook-submit` is always an HTTP client (posts to `MEM9_URL`).

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
