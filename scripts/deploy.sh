#!/usr/bin/env bash
# Deploy to production (rmb.colinleefish.com).
#
# Workflow:
#   1. ci.sh         — Go tests (also compile-checks via go test)
#   2. docker buildx — cross-build linux/amd64 image (builds UI + Go binary)
#                      and push to ACR as <branch-slug> and <branch-slug>-<sha>
#   3. SSH to prod   — sync deploy/ runtime files, pull :main, force-recreate
#                      rmb container, health check
#
# Production compose pins registry.../rmb:main. Deploy from main updates :main.
set -euo pipefail
cd "$(dirname "$0")/.."

# Optional: scripts/deploy.env (gitignored) with DEPLOY_* overrides
if [[ -f scripts/deploy.env ]]; then
  # shellcheck disable=SC1091
  source scripts/deploy.env
fi

DEPLOY_HOST="${DEPLOY_HOST:-rmb.colinleefish.com}"
DEPLOY_USER="${DEPLOY_USER:-root}"
DEPLOY_PATH="${DEPLOY_PATH:-/app/rmb}"
DEPLOY_PORT="${DEPLOY_PORT:-22}"
DEPLOY_SSH_KEY_FILE="${DEPLOY_SSH_KEY_FILE:-$HOME/.ssh/colinleefish_ed25519}"
REGISTRY="${REGISTRY:-registry.cn-beijing.aliyuncs.com/colinleefish/rmb}"
COMPOSE_FILE="deploy/docker-compose.yml"

if [[ ! -f "$DEPLOY_SSH_KEY_FILE" ]]; then
  echo "SSH key not found: $DEPLOY_SSH_KEY_FILE" >&2
  echo "Set DEPLOY_SSH_KEY_FILE or create scripts/deploy.env" >&2
  exit 1
fi

# Lowercase; replace runs of non [a-z0-9] with a single hyphen; trim hyphens.
branch_slug() {
  local raw="${1:-}"
  raw="$(printf '%s' "$raw" | tr '[:upper:]' '[:lower:]')"
  raw="$(printf '%s' "$raw" | sed -E 's/[^a-z0-9]+/-/g; s/^-+|-+$//g')"
  printf '%s' "${raw:-branch}"
}

echo "==> CI"
./scripts/ci.sh

BRANCH="$(git rev-parse --abbrev-ref HEAD)"
SHA="$(git rev-parse --short HEAD)"
SLUG="$(branch_slug "$BRANCH")"
TAG_BRANCH="${REGISTRY}:${SLUG}"
TAG_COMMIT="${REGISTRY}:${SLUG}-${SHA}"

echo "==> Build and push ${TAG_BRANCH}, ${TAG_COMMIT} (linux/amd64)"
docker buildx build --platform linux/amd64 \
  -t "${TAG_BRANCH}" \
  -t "${TAG_COMMIT}" \
  --push .

if [[ "$SLUG" != "main" ]]; then
  echo "WARN: not on main (branch=${BRANCH}, slug=${SLUG}); production still runs :main" >&2
fi

# IdentitiesOnly=yes forces use of the configured key only, so a loaded
# ssh-agent with many keys cannot trip the server's MaxAuthTries.
SSH_OPTS=(-i "$DEPLOY_SSH_KEY_FILE" -o IdentitiesOnly=yes -o StrictHostKeyChecking=accept-new -o BatchMode=yes -p "$DEPLOY_PORT")
SCP_OPTS=(-i "$DEPLOY_SSH_KEY_FILE" -o IdentitiesOnly=yes -o StrictHostKeyChecking=accept-new -o BatchMode=yes -P "$DEPLOY_PORT")
REMOTE="${DEPLOY_USER}@${DEPLOY_HOST}"

echo "==> Sync runtime files to ${REMOTE}:${DEPLOY_PATH}"
ssh "${SSH_OPTS[@]}" "${REMOTE}" "mkdir -p ${DEPLOY_PATH}/config"
scp "${SCP_OPTS[@]}" "${COMPOSE_FILE}" "${REMOTE}:${DEPLOY_PATH}/docker-compose.yml"
scp "${SCP_OPTS[@]}" deploy/config/Caddyfile "${REMOTE}:${DEPLOY_PATH}/config/Caddyfile"

echo "==> Pull :main and recreate rmb on ${REMOTE}"
ssh "${SSH_OPTS[@]}" "${REMOTE}" bash -s <<EOF
set -euo pipefail
cd "$DEPLOY_PATH"
docker compose pull rmb
docker compose up -d --force-recreate rmb
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

echo "Deploy OK: https://rmb.colinleefish.com (${TAG_COMMIT})"
