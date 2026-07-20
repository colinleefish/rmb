# rmb

English / [简体中文](README_CN.md)

**Long-term memory for AI agents.**

rmb captures conversation turns from your coding agents, distills them into structured facts in the background, and lets any agent recall what it learned across sessions. You run the server; your data stays on your infrastructure.

## Why rmb?

AI agents forget everything when a session ends. rmb fixes that with a layered memory pipeline:

1. **Capture** — hooks from Cursor, Claude Code, Codex, or Pi upload each turn to your server.
2. **Distill** — background workers extract atoms, group them into scenes, and roll durable memories across sessions.
3. **Recall** — agents search and browse memories with the `rmb` CLI before asking you the same question twice.

Everything is stored in PostgreSQL (with pgvector for semantic search) and addressed with stable `rmb://` URIs.

## Memory pyramid

Knowledge is organized in tiers from raw chat to long-term facts:

```
                    ┌─────────────────────────────────────┐
                    │  T3 — memories (cross-session)      │
                    │  profile · preferences · entities   │
                    └──────────────────▲──────────────────┘
                                       │
              ┌────────────────────────┴─────────────────────────┐
              │  Session · rmb://sessions/<sid>                  │
              │  turns (T0) → atoms (T1) → scenes (T2)           │
              └──────────────────────────────────────────────────┘
```

| Tier | What | URI example |
|------|------|-------------|
| T0 | Raw user + assistant exchange | `rmb://turns/<uuid>` |
| T1 | Small extracted fact | `rmb://atoms/<uuid>` |
| T2 | Session-local narrative segment | `rmb://scenes/<uuid>` |
| T3 | Durable cross-session memory | `rmb://profile`, `rmb://entities/<slug>` |

See [`docs/concept/pyramid.md`](docs/concept/pyramid.md) for the full model.

## Features

- **Agent hooks** — `rmb hook-submit --source=<cursor|cc|codex|pi|opencode>` ingests turns from supported tools.
- **Background workers** — T1 extraction, T2 scene synthesis, and T3 memory rollup (enabled by default).
- **Hybrid recall** — vector + full-text search fused with reciprocal rank fusion.
- **Skills** — curated agent playbooks stored in rmb (`rmb://skills/<name>`).
- **Corrections** — human overrides that agents must respect.
- **Web UI** — browse sessions, turns, atoms, scenes, memories, and pipeline state at `/ui/`.
- **CLI** — `search`, `cat`, `tree`, `meta`, `correction`, and `skill` commands for agents and operators.

## Quick start

### Prerequisites

- Go 1.26+ (to build from source)
- PostgreSQL 16+ with [pgvector](https://github.com/pgvector/pgvector)
- An OpenAI-compatible LLM API key (for distillation workers)
- An embedding API key (for semantic search)

### 1. Clone and configure

```bash
git clone https://github.com/colinleefish/rmb.git
cd rmb
cp .env.example .env
```

Edit `.env` by concern (`cp .env.example .env`, then replace `replace_me` and connection strings):

**Database**

- `RMB_DB_URL` — PostgreSQL connection string  
  - Source + local Postgres: set for your instance (see `.env.example`)  
  - `docker compose`: in-container default is `postgres://rmb:rmb@postgres:5432/rmb_db`; if you run `make run` on the host against the compose DB, use `postgres://rmb:rmb@127.0.0.1:5433/rmb_db`

**Chat API (T1–T3 background workers — all three required)**

- `RMB_LLM_API_BASE` — OpenAI-compatible API base URL  
- `RMB_LLM_API_KEY` — API key  
- `RMB_LLM_MODEL` — model name (e.g. `gpt-4o-mini`, `glm-4.7`)

**Embedding API (semantic search + embed worker)**

- `RMB_EMBED_API_KEY` — API key (`rmb search` is unavailable without it)  
- `RMB_EMBED_API_BASE` — API base (defaults to Zhipu; change when switching providers)  
- `RMB_EMBED_MODEL` — model name (default `embedding-3`)  
- `RMB_EMBED_DIMENSIONS` — vector size (default `1024`; must match the model)

**HTTP auth**

- Optional when listening on `127.0.0.1`; **required** when binding to `0.0.0.0`, `:8080`, or any address reachable off localhost — set both `USERNAME` and `PASSWORD` (or `RMB_USERNAME` / `RMB_PASSWORD`)

With the root `docker compose`, add `env_file: .env` (or explicit `environment` entries) under the `rmb` service so LLM and embed keys are passed into the container.

### 2. Start the server

**Docker Compose** (PostgreSQL + app):

```bash
docker compose up -d
curl http://localhost:8080/healthz
```

**From source:**

```bash
make run
# or
make build && ./bin/rmb serve
```

Open the observer UI at `http://localhost:8080/ui/`.

### 3. Build the CLI

```bash
make build
./bin/rmb help
```

Install `./bin/rmb` on your PATH, or point agent hooks at the binary directly.

### 4. Point the CLI at your server

Create `~/.rmb.conf` or `~/.rmb/config.yaml`:

```ini
RMB_URL=http://127.0.0.1:8080
RMB_USERNAME=your-user
RMB_PASSWORD=your-password
```

Recall commands (`search`, `cat`, `tree`, `meta`, `correction`, `skill`) call the server over HTTP. `hook-submit` posts turns to the same URL.

### 5. Register agent hooks

**Cursor** — `~/.cursor/hooks.json`:

```json
{
  "hooks": {
    "afterAgentResponse": [
      {
        "command": "/path/to/rmb hook-submit --source=cursor",
        "timeout": 5
      }
    ]
  },
  "version": 1
}
```

**Claude Code** — `~/.claude/settings.json`:

```json
{
  "hooks": {
    "Stop": [
      {
        "hooks": [
          { "type": "command", "command": "/path/to/rmb hook-submit --source=cc" }
        ]
      }
    ]
  }
}
```

**Pi** — install the extension from [`integrations/pi/`](integrations/pi/README.md) (no shell hooks; uses `agent_settled` events).

**OpenCode** — install the plugin from [`integrations/opencode/`](integrations/opencode/README.md) (uses `session.status` idle / `session.idle` events).

`--source` is mandatory. Mismatched payloads exit non-zero.

### 6. Verify it works

1. Have a short agent conversation.
2. Open `/ui/` → Sessions → pick your session → see turns appear.
3. Wait for background workers to extract atoms and scenes.
4. Recall from the CLI:

```bash
rmb search "what do you know about me"
rmb cat rmb://profile
rmb tree rmb://sessions/<session-uuid>/
```

## Deploy on your server

rmb is designed to run on infrastructure you control. A typical production layout:

```
Internet
   │
   ▼
┌─────────┐     ┌──────────────┐     ┌────────────┐
│  Caddy  │────►│  rmb :8080   │────►│ PostgreSQL │
│  :443   │     │  (Docker)    │     │  + pgvector│
└─────────┘     └──────────────┘     └────────────┘
```

### Option A — Docker Compose (all-in-one dev / small deploy)

The root [`docker-compose.yml`](docker-compose.yml) runs PostgreSQL and rmb together. Good for a single VM or homelab:

```bash
docker compose up -d
```

Customize environment in `docker-compose.yml` or an `.env` file alongside it.

### Option B — Split database + app container

For production, run PostgreSQL on the host (or a managed service) and deploy only the app container. See [`deploy/docker-compose.yml`](deploy/docker-compose.yml) for a minimal two-service layout (rmb + Caddy reverse proxy).

1. Copy `deploy/` to your server (e.g. `/app/rmb`).
2. Create `.env` with `RMB_DB_URL`, LLM keys, and auth credentials.
3. Edit [`deploy/config/Caddyfile`](deploy/config/Caddyfile) — replace the hostname with your domain:

```
memory.example.com {
    reverse_proxy 127.0.0.1:8080
}
```

4. Build and push your own image, or build on the server:

```bash
docker build -t rmb:local .
```

5. Start:

```bash
cd /app/rmb
docker compose up -d
curl -fsS https://memory.example.com/healthz
```

6. Set `RMB_URL=https://memory.example.com` in client config on machines that run hooks or recall.

### Security checklist

- Always enable `USERNAME` / `PASSWORD` (or `RMB_USERNAME` / `RMB_PASSWORD`) when the server is not bound to localhost.
- Terminate TLS at a reverse proxy (Caddy, nginx, Traefik).
- Keep PostgreSQL off the public internet; rmb connects to it over a private network or localhost.
- Store API keys in `.env` on the server, not in hook configs.

## Architecture

```txt
agent (Cursor / Claude Code / Pi)
   │
   │ stdin JSON  ── hooks ──► rmb hook-submit --source=…
   │                                │
   │                                ▼
   │                     POST /api/v1/sessions/:id/upload
   │                                │
   ▼                                ▼
                              ┌──────────────┐
                              │     rmb      │
                              │  (Gin HTTP)  │
                              └──────┬───────┘
                                     │
                    ┌────────────────┼────────────────┐
                    ▼                ▼                ▼
             ┌────────────┐   ┌────────────┐   ┌────────────┐
             │ T1 extract │   │ T2 scene   │   │ T3 memory  │
             │  worker    │   │  worker    │   │  worker    │
             └─────┬──────┘   └─────┬──────┘   └─────┬──────┘
                   │                │                │
                   └────────────────┼────────────────┘
                                    ▼
                              ┌──────────────┐
                              │  PostgreSQL  │
                              │  + pgvector  │
                              └──────────────┘
```

Background workers run inside the `rmb serve` process. They need an OpenAI-compatible chat API. The embed worker (for recall) uses a separate embedding endpoint.

## CLI reference

| Command | Purpose |
|---------|---------|
| `rmb serve` | Start the HTTP server (run on the host that owns the database) |
| `rmb hook-submit --source=<src>` | Ingest a hook payload (always HTTP client) |
| `rmb search "<query>"` | Hybrid recall across memories, scenes, and skills |
| `rmb cat <uri>` | Print the body of a memory artifact |
| `rmb tree <uri-prefix>` | List children under a scope |
| `rmb meta <uri>` | Show metadata (provenance, session links) |
| `rmb correction add <uri> "…"` | Attach a human correction |
| `rmb skill ls` / `pull` / `put` | Manage curated agent skills |

Agents should run `rmb search` before asking the user, then `rmb cat` on relevant URIs. See [`docs/guide/cli-for-agents.md`](docs/guide/cli-for-agents.md).

## API

| Endpoint | Description |
|----------|-------------|
| `POST /api/v1/sessions/:id/upload` | Append a turn (hook target) |
| `GET /api/v1/search?q=…&k=…` | Hybrid recall |
| `GET /api/v1/inspect/{cat,tree,meta}?uri=…` | Inspection (used by CLI) |
| `GET /healthz` | DB ping + pgvector check |

Upload body:

```json
{
  "started_at": "optional RFC3339",
  "messages": [
    {"role": "user", "content": "..."},
    {"role": "assistant", "content": "..."}
  ]
}
```

## Configuration

Resolution order (later wins):

1. Defaults in `internal/config`
2. `~/.rmb.conf` or `~/.rmb/config.yaml` (client); server uses `.env` in the working directory
3. Process environment

Key server variables (see [`.env.example`](.env.example)):

| Variable | Default | Purpose |
|----------|---------|---------|
| `RMB_DB_URL` | `postgres://admin@127.0.0.1:5432/rmb_db?sslmode=disable` | PostgreSQL |
| `RMB_ADDR` | `:8080` | Listen address |
| `RMB_LLM_API_BASE` / `_API_KEY` / `_MODEL` | — | Chat API for workers |
| `RMB_EMBED_API_KEY` | — | Embeddings for recall (required for search) |
| `RMB_EXTRACTION_ENABLED` | `true` | T1 atom extraction |
| `RMB_SCENE_ENABLED` | `true` | T2 scene synthesis |
| `RMB_MEMORY_ENABLED` | `true` | T3 memory rollup |

Client variables: `RMB_URL`, `RMB_USERNAME`, `RMB_PASSWORD`.

## Project layout

```
cmd/rmb/              CLI entry point (serve / hook-submit / recall)
internal/
  hook/               hook payload → upload adapter (cursor, cc, codex, pi, opencode)
  http/               Gin router, handlers, embedded web UI
  service/
    extract/          T1 extraction worker
    scene/            T2 scene worker
    memory/           T3 memory worker
    embed/            embedding worker
  llm/                OpenAI-compatible clients
ui-next/              Next.js observer UI (built into the binary)
integrations/pi/      Pi agent extension
integrations/opencode/ OpenCode plugin
deploy/               production compose + Caddy example
docs/                 full documentation site (make docs-dev)
```

## Development

```bash
make ci          # go test ./... + compile check
go test ./...    # tests only
make docs-dev    # documentation site at localhost:5173
```

Roadmap: [`docs/reference/plan.md`](docs/reference/plan.md). Planned: MCP wrapper, `rmb eval` drift probes.

## Documentation

- [Getting started](docs/guide/getting-started.md)
- [The memory pyramid](docs/concept/pyramid.md)
- [URI scheme](docs/concept/uri-scheme.md)
- [Hook integration details](docs/guide/getting-started.md#register-hooks)

Run `make docs-dev` to browse the full docs site locally.

## License

TBD.
