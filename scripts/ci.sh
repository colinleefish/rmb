#!/usr/bin/env bash
# Local / agent CI: test and compile check.
set -euo pipefail
cd "$(dirname "$0")/.."

export GOPROXY="${GOPROXY:-https://goproxy.cn,direct}"

echo "==> go test"
go test ./...

echo "==> go build"
go build -o /dev/null ./cmd/rmb

echo "CI OK"
