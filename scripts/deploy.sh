#!/usr/bin/env bash
# Deploy to production (rmb.colinleefish.com).
#
# Workflow:
#   1. ci.sh         — Go tests (also compile-checks via go test)
#   2. docker buildx — cross-build linux/amd64 image (builds UI + Go binary)
#                      and push to ACR
#   3. Bump image tag in docker-compose.prod.yml, commit, push to main
#   4. SSH to prod   — git pull, docker compose pull, up -d, health check
set -euo pipefail
cd "$(dirname "$0")/.."

# Optional: scripts/deploy.env (gitignored) with DEPLOY_* overrides
if [[ -f scripts/deploy.env ]]; then
  # shellcheck disable=SC1091
  source scripts/deploy.env
fi

DEPLOY_HOST="${DEPLOY_HOST:-rmb.colinleefish.com}"
DEPLOY_USER="${DEPLOY_USER:-root}"
DEPLOY_PATH="${DEPLOY_PATH:-/opt/rmb}"
DEPLOY_PORT="${DEPLOY_PORT:-22}"
DEPLOY_SSH_KEY_FILE="${DEPLOY_SSH_KEY_FILE:-$HOME/.ssh/colinleefish_ed25519}"
REGISTRY="${REGISTRY:-registry.cn-beijing.aliyuncs.com/colinleefish/rmb}"

if [[ ! -f "$DEPLOY_SSH_KEY_FILE" ]]; then
  echo "SSH key not found: $DEPLOY_SSH_KEY_FILE" >&2
  echo "Set DEPLOY_SSH_KEY_FILE or create scripts/deploy.env" >&2
  exit 1
fi

echo "==> CI"
./scripts/ci.sh

SHA="$(git rev-parse --short HEAD)"
IMAGE="${REGISTRY}:${SHA}"

echo "==> Build and push ${IMAGE} (linux/amd64)"
docker buildx build --platform linux/amd64 -t "${IMAGE}" --push .

echo "==> Bump image tag → ${SHA}"
sed -i.bak "s|    image: ${REGISTRY}:.*|    image: ${IMAGE}|" docker-compose.prod.yml
rm -f docker-compose.prod.yml.bak
git add docker-compose.prod.yml
git commit -m "Bump prod image tag to ${SHA}"
git push origin main

# IdentitiesOnly=yes forces use of the configured key only, so a loaded
# ssh-agent with many keys cannot trip the server's MaxAuthTries.
SSH_OPTS=(-i "$DEPLOY_SSH_KEY_FILE" -o IdentitiesOnly=yes -o StrictHostKeyChecking=accept-new -o BatchMode=yes -p "$DEPLOY_PORT")

echo "==> Pull and restart on ${DEPLOY_USER}@${DEPLOY_HOST}"
ssh "${SSH_OPTS[@]}" "${DEPLOY_USER}@${DEPLOY_HOST}" bash -s <<EOF
set -euo pipefail
cd "$DEPLOY_PATH"
# Proxy only for GitHub — do not export globally; curl would use it for 127.0.0.1:8080.
git -c http.proxy=http://localhost:1080 -c https.proxy=http://localhost:1080 pull origin main
docker compose -f docker-compose.prod.yml pull rmb
docker compose -f docker-compose.prod.yml up -d rmb
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

echo "Deploy OK: https://rmb.colinleefish.com"
