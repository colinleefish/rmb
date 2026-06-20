APP := mem9
CMD := ./cmd/mem9
BIN := ./bin/$(APP)

.PHONY: build run ci deploy

build:
	go build -o $(BIN) $(CMD)

run:
	go run $(CMD) serve

ci:
	./scripts/ci.sh

deploy:
	./scripts/deploy.sh
