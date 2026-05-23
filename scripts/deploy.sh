#!/usr/bin/env bash
# Agent-driven deploy to production (mem.colinleefish.com).
set -euo pipefail
cd "$(dirname "$0")/.."

# Optional: scripts/deploy.env (gitignored) with DEPLOY_* overrides
if [[ -f scripts/deploy.env ]]; then
  # shellcheck disable=SC1091
  source scripts/deploy.env
fi

DEPLOY_HOST="${DEPLOY_HOST:-mem.colinleefish.com}"
DEPLOY_USER="${DEPLOY_USER:-root}"
DEPLOY_PATH="${DEPLOY_PATH:-/opt/mypast}"
DEPLOY_PORT="${DEPLOY_PORT:-22}"
DEPLOY_SSH_KEY_FILE="${DEPLOY_SSH_KEY_FILE:-$HOME/.ssh/colinleefish_ed25519}"
GIT_REF="${GIT_REF:-main}"

if [[ ! -f "$DEPLOY_SSH_KEY_FILE" ]]; then
  echo "SSH key not found: $DEPLOY_SSH_KEY_FILE" >&2
  echo "Set DEPLOY_SSH_KEY_FILE or create scripts/deploy.env (see deploy.env.example)" >&2
  exit 1
fi

echo "==> CI before deploy"
./scripts/ci.sh

SHA="$(git rev-parse "$GIT_REF")"
echo "==> Deploy $SHA to ${DEPLOY_USER}@${DEPLOY_HOST}:${DEPLOY_PATH}"

SSH_OPTS=(-i "$DEPLOY_SSH_KEY_FILE" -o StrictHostKeyChecking=accept-new -o BatchMode=yes -p "$DEPLOY_PORT")

ssh "${SSH_OPTS[@]}" "${DEPLOY_USER}@${DEPLOY_HOST}" bash -s <<EOF
set -euo pipefail
cd "$DEPLOY_PATH"
# Proxy only for GitHub — do not export globally; curl would use it for 127.0.0.1:8080.
git -c http.proxy=http://localhost:1080 -c https.proxy=http://localhost:1080 fetch origin main
git checkout main
git reset --hard "$SHA"
docker compose -f docker-compose.prod.yml up -d --build
for i in 1 2 3 4 5 6 7 8 9 10; do
  if curl --noproxy '*' -fsS http://127.0.0.1:8080/healthz >/dev/null; then
    echo "healthz OK"
    exit 0
  fi
  sleep 3
done
echo "healthz failed after deploy" >&2
exit 1
EOF

echo "Deploy OK: https://mem.colinleefish.com"
