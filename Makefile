BIN_NAME=consent

.PHONY: build test keys

build:
	go generate ./...
	go build -o ./bin/$(BIN_NAME) ./cmd/$(BIN_NAME)

keys:
	mkdir -p ./secrets
	go run ./cmd/keygen -out ./secrets

test: build
	go test ./...
