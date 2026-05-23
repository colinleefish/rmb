# Deploy (agent-driven)

No GitHub Actions. The agent (or you) runs scripts locally, then deploys over SSH.

## CI

```bash
make ci
# or: ./scripts/ci.sh
```

Runs `go test ./...` and a compile check.

## Deploy to production

**Target:** `mem.colinleefish.com` — `/opt/mypast`, `docker compose -f docker-compose.prod.yml up -d --build`.

**Prerequisites**

- `main` pushed to GitHub (server pulls via deploy key).
- SSH key with access to `root@mem.colinleefish.com` (default: `~/.ssh/colinleefish_ed25519`).
- Optional overrides: copy `scripts/deploy.env.example` → `scripts/deploy.env`.

```bash
make deploy
# or: ./scripts/deploy.sh
```

Deploy runs CI first, then `git reset --hard` to `main` on the server and rebuilds containers. Waits for `/healthz`.

## Agent workflow

When the user asks to **ship**, **deploy**, or **release** after code changes:

1. Run `make ci` and fix failures.
2. Commit and push to `main` (ask the user if unclear).
3. Run `make deploy`.
4. Report `healthz` result and link https://mem.colinleefish.com

For **test-only** or PR prep: stop after step 1.

## Server notes

- Postgres runs on the host (`:5432`); app config in `/opt/mypast/.env`.
- Production compose: `docker-compose.prod.yml` + `deploy/Caddyfile`.
- Do not commit secrets or `scripts/deploy.env`.
