#!/usr/bin/env bash
# Local / CI: test the Go backend.
# go test ./... compiles every package as a side effect, so a separate
# go build step is redundant. The authoritative binary is produced by
# the Docker build stage in deploy.sh.
set -euo pipefail
cd "$(dirname "$0")/.."

export GOPROXY="${GOPROXY:-https://goproxy.cn,direct}"

echo "==> go test"
go test ./cmd/... ./internal/...

echo "CI OK"
