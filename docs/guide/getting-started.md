# Getting started

Run rmb locally, register hooks, verify capture and distillation.

## Prerequisites

- Go 1.26+
- PostgreSQL 16+ with pgvector (or use Docker Compose)
- OpenAI-compatible LLM API key (for workers)

## Local server

```bash
git clone https://github.com/colinleefish/rmb.git
cd rmb
cp .env.example .env   # set RMB_DB_URL, RMB_LLM_API_KEY, RMB_EMBED_API_KEY
make run               # or: docker compose up -d
curl localhost:8080/healthz
```

Web observer UI: `http://localhost:8080/ui/` — browse sessions, turns, atoms, scenes, memories, pipeline state.

## Build the CLI

```bash
make build
./bin/rmb --help
```

Install on your PATH or point hooks at `./bin/rmb`.

## Configure the client

`~/.rmb.conf` (flat) or `~/.rmb/config.yaml`:

```ini
RMB_URL=http://127.0.0.1:8080
# optional basic auth for remote server
```

For production recall from your laptop:

```ini
RMB_URL=https://rmb.colinleefish.com
RMB_USER=...
RMB_PASS=...
```

## Register hooks

### Cursor

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

### Claude Code

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

`--source` is mandatory. Mismatched payloads exit non-zero.

## Verify capture

1. Have a short agent conversation
2. Open `/ui/` → Sessions → pick your session → see turns
3. Wait for T1 (or run `rmb t1 backfill` on the server)
4. Atoms appear under the session; scenes follow after T2 delay

## Recall

```bash
rmb search "what do you know about me"
rmb cat rmb://profile
rmb tree rmb://sessions/<session-uuid>/
rmb cat rmb://turns/<turn-uuid>    # raw messages_jsonl
rmb cat rmb://atoms/<atom-uuid>    # extracted fact
```

Agents should read [CLI for agents](/guide/cli-for-agents) — paste into Cursor rules or skill.

## Production

Deployed at [rmb.colinleefish.com](https://rmb.colinleefish.com). Agent-driven deploy: `make ci && make deploy`. See [Deploy](/reference/deploy).

## Next steps

- [The pyramid](/concept/pyramid) — understand T0–T3
- [Implementation plan](/reference/plan) — what's shipped vs planned
- [Full design](/design/l0-l3) — URI scheme, storage, retrieval sketch
