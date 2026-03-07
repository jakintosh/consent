BIN_NAME=consent
PORT ?=9001
BASE_URL ?=http://localhost:$(PORT)
ISSUER_DOMAIN ?=localhost

LOCAL_TEST_ROOT ?=./data/.local/consent-localtest
LOCAL_TEST_PORT ?=9001
LOCAL_TEST_BASE_URL ?=http://localhost:$(LOCAL_TEST_PORT)
LOCAL_TEST_ISSUER ?=localhost

.PHONY: build test keys dev-serve dev-client local-test-run local-test-clean

build:
	go generate ./...
	go build -o ./bin/$(BIN_NAME) ./cmd/$(BIN_NAME)

keys:
	mkdir -p ./secrets
	go run ./cmd/keygen -out ./secrets

test: build
	go test ./...

dev-serve:
	LOCAL_TEST_ROOT="$(LOCAL_TEST_ROOT)" \
	LOCAL_TEST_PORT="$(PORT)" \
	LOCAL_TEST_BASE_URL="$(BASE_URL)" \
	LOCAL_TEST_ISSUER="$(ISSUER_DOMAIN)" \
	./scripts/local/run-dev-consent.sh

dev-client:
	LOCAL_TEST_ROOT="$(LOCAL_TEST_ROOT)" \
	LOCAL_TEST_BASE_URL="$(BASE_URL)" \
	LOCAL_TEST_ISSUER="$(ISSUER_DOMAIN)" \
	./scripts/local/run-dev-client.sh "example@localhost" "10000" "Example"

local-test-run: local-test-clean
	LOCAL_TEST_ROOT="$(LOCAL_TEST_ROOT)" \
	LOCAL_TEST_PORT="$(LOCAL_TEST_PORT)" \
	LOCAL_TEST_BASE_URL="$(LOCAL_TEST_BASE_URL)" \
	LOCAL_TEST_ISSUER="$(LOCAL_TEST_ISSUER)" \
	./scripts/local/run-local-test-stack.sh

local-test-clean:
	rm -rf "$(LOCAL_TEST_ROOT)"
