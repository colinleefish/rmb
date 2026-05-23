# mypast

Personal long-term memory store for AI-agent conversations. Captures every
turn from supported agent tools into PostgreSQL, then summarizes them in the
background so they can be recalled across sessions.

## Status

What works today:

- HTTP API for uploading conversation turns (`POST /api/v1/sessions/:id/upload`).
- `mypast hook-submit --source=<cursor|cc>` for ingesting hook payloads from
  Cursor and Claude Code, with race-free user/assistant pairing.
- Background summarizer worker that merges new turns into a per-session
  overview via an OpenAI-compatible chat-completions endpoint.

Inspection CLI: `mypast cat`, `mypast tree`, `mypast meta` (Phase A).

Web observer UI at `/ui/` when the server is running (browse sessions, turns, atoms,
scenes, memories, pipeline state, and tasks).

Production (`mem.colinleefish.com`): Caddy in Docker terminates TLS and proxies to
`mypast` on `:8080` — see `deploy/Caddyfile` and `docker-compose.prod.yml`.

Planned (see `TODO.md`): storage CLI (`store/read/list/delete/search`),
embedding worker, hybrid recall, MCP wrapper.

## Architecture

```txt
agent (Cursor / Claude Code)
   │
   │ stdin JSON  ── hooks ──► mypast hook-submit --source=…
   │                                │
   │                                ▼
   │                     POST /api/v1/sessions/:id/upload
   │                                │
   ▼                                ▼
                              ┌──────────────┐
                              │   mypast     │
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
cmd/mypast/            entry point (serve / hook-submit)
internal/
  cli/                 argv parsing
  config/              .env + ~/.config/mypast/config.toml + env overrides
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
cp .env.example .env       # edit MYPAST_DB_URL, MYPAST_LLM_API_KEY, …
make run                    # go run ./cmd/mypast serve
# or
make build && ./bin/mypast serve
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

**Production:** https://mem.colinleefish.com — app at `/opt/mypast`, Caddy + Docker
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
2. `~/.config/mypast/config.toml` (path overridable via `MYPAST_CONFIG`).
3. `.env` in the working directory.
4. Process environment.

Key variables (see `.env.example` for the full list):

| Var                                   | Default                                                            |
|---------------------------------------|--------------------------------------------------------------------|
| `MYPAST_DB_URL`                       | `postgres://admin@127.0.0.1:5432/mypast_dev?sslmode=disable`       |
| `MYPAST_ADDR`                         | `:8080`                                                            |
| `MYPAST_LLM_API_BASE` / `_API_KEY` / `_MODEL` | OpenAI-compatible endpoint used by the summarizer            |
| `MYPAST_SUMMARIZER_ENABLED`           | `true`                                                             |
| `MYPAST_SUMMARIZER_POLL_INTERVAL`     | `15s`                                                              |
| `MYPAST_SUMMARIZER_MAX_TURNS_PER_MERGE` | `4`                                                              |

For `hook-submit` invocations, the target API URL is read from `MYPAST_URL`
or `~/.mypast.conf` (key `MYPAST_URL=`), defaulting to `http://127.0.0.1:8080`.

## Hook Integration

Each agent gets a single hook entry. `--source` is mandatory; mismatched
payloads exit non-zero.

`~/.cursor/hooks.json`:

```json
{
  "hooks": {
    "afterAgentResponse": [
      {
        "command": "/path/to/mypast/bin/mypast hook-submit --source=cursor",
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
          { "type": "command", "command": "/path/to/mypast/bin/mypast hook-submit --source=cc" }
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
the `mypast://sessions/<id>/turns/<n>` URI for the new turn.

`GET /healthz` — DB ping + `pg_extension` lookup for `vector`.

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
