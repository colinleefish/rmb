APP := mypast
CMD := ./cmd/mypast
BIN := ./bin/$(APP)

.PHONY: build run

build:
	go build -o $(BIN) $(CMD)

run:
	go run $(CMD) serve
