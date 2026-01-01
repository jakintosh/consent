BIN_NAME=consent

.PHONY: build test

build:
	go generate ./...
	go build -o ./bin/$(BIN_NAME) ./cmd/$(BIN_NAME)

test: build
	go test ./...
