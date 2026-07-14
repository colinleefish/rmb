# Deploy (agent-driven)

No GitHub Actions. The agent (or you) runs scripts locally, then deploys over SSH.

## CI

```bash
make ci
# or: ./scripts/ci.sh
```

Runs `go test ./...` and a compile check.

## Deploy to production

**Target:** `rmb.colinleefish.com` — `/app/rmb`, `docker compose up -d`.

**Prerequisites**

- Code changes committed and pushed to GitHub before deploy (deploy does not commit).
- SSH key with access to `root@rmb.colinleefish.com` (default: `~/.ssh/colinleefish_ed25519`).
- Optional overrides: copy `scripts/deploy.env.example` → `scripts/deploy.env`.
- HTTP(S) proxy on port **1080** on your machine when pushing to GitHub from China.

### Proxy (China → GitHub)

| Where | When | Setting |
|-------|------|---------|
| **Your machine** | push / fetch to GitHub | `ssproxy && git push origin main` (`http(s)_proxy=http://127.0.0.1:1080`) |

The production server has **no git checkout**. Deploy syncs runtime files over SCP and
pulls the new image from ACR.

```bash
make deploy
# or: ./scripts/deploy.sh
```

Deploy runs CI, builds and pushes a Docker image tagged `<branch-slug>` and
`<branch-slug>-<sha>`, then SCPs runtime files to the server, pulls `:main`, and
force-recreates the `rmb` container. Waits for `/healthz`.

## Agent workflow

When the user asks to **ship**, **deploy**, or **release** after code changes:

1. Run `make ci` and fix failures.
2. Commit and push to `main` with `ssproxy && git push` when needed (ask the user if unclear).
3. Run `make deploy`.
4. Report `healthz` result and link https://rmb.colinleefish.com

For **test-only** or PR prep: stop after step 1.

## Server layout

```txt
/app/rmb/
├── .env                 # secrets (server only; not in git)
├── docker-compose.yml   # synced from deploy/docker-compose.yml
└── config/
    └── Caddyfile        # synced from deploy/config/Caddyfile
```

## Server notes

- Postgres runs on the host (`:5432`); app config in `/app/rmb/.env`.
- Repo source of truth: `deploy/docker-compose.yml` + `deploy/config/Caddyfile`.
- Do not commit secrets or `scripts/deploy.env`.
