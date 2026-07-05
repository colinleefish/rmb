APP := rmb
CMD := ./cmd/rmb
BIN := ./bin/$(APP)

.PHONY: build run ci deploy docs-dev docs-build

build:
	go build -o $(BIN) $(CMD)

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
