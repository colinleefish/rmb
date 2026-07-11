APP := rmb
CMD := ./cmd/rmb
BIN := ./bin/$(APP)
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
LDFLAGS := -X github.com/colinleefish/rmb/internal/buildinfo.Commit=$(COMMIT)

.PHONY: build run ci deploy docs-dev docs-build

build:
	go build -ldflags "$(LDFLAGS)" -o $(BIN) $(CMD)

run:
	go run $(CMD) serve

ci:
	./scripts/ci.sh

deploy:
	./scripts/deploy.sh

docs-dev:
	cd docs && pnpm install && pnpm dev

docs-build:
	cd docs && pnpm install && pnpm build
