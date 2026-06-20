# Deploy (agent-driven)

No GitHub Actions. The agent (or you) runs scripts locally, then deploys over SSH.

## CI

```bash
make ci
# or: ./scripts/ci.sh
```

Runs `go test ./...` and a compile check.

## Deploy to production

**Target:** `rmb.colinleefish.com` — `/opt/rmb`, `docker compose -f docker-compose.prod.yml up -d --build`.

**Prerequisites**

- `main` pushed to GitHub (server pulls via deploy key).
- SSH key with access to `root@rmb.colinleefish.com` (default: `~/.ssh/colinleefish_ed25519`).
- Optional overrides: copy `scripts/deploy.env.example` → `scripts/deploy.env`.
- HTTP(S) proxy on port **1080** on both your machine and the server (GitHub is blocked from China).

### Proxy (China → GitHub)

| Where | When | Setting |
|-------|------|---------|
| **Your machine** | push / fetch to GitHub | `ssproxy && git push origin main` (`http(s)_proxy=http://127.0.0.1:1080`) |
| **Production server** | `git fetch` in deploy | `https_proxy=http://localhost:1080` (set in `scripts/deploy.sh`) |

Deploy uses `git -c http.proxy=… -c https.proxy=… fetch` (proxy scoped to Git only).
The post-deploy `curl` to `127.0.0.1:8080/healthz` uses `--noproxy '*'` so it never goes through the proxy.

```bash
make deploy
# or: ./scripts/deploy.sh
```

Deploy runs CI first, then `git reset --hard` to `main` on the server and rebuilds containers. Waits for `/healthz`.

## Agent workflow

When the user asks to **ship**, **deploy**, or **release** after code changes:

1. Run `make ci` and fix failures.
2. Commit and push to `main` with `ssproxy && git push` when needed (ask the user if unclear).
3. Run `make deploy`.
4. Report `healthz` result and link https://rmb.colinleefish.com

For **test-only** or PR prep: stop after step 1.

## Server notes

- Postgres runs on the host (`:5432`); app config in `/opt/rmb/.env`.
- Production compose: `docker-compose.prod.yml` + `deploy/Caddyfile`.
- Do not commit secrets or `scripts/deploy.env`.
