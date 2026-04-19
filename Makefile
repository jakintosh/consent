BIN_NAME=consent

LOCAL_CONFIG_DIR=./config
LOCAL_DATA_DIR=./data
LOCAL_BASE_URL?=http://localhost:9001
LOCAL_ISSUER?=localhost
LOCAL_PORT?=9001

.PHONY: build init run-local test

build:
	go generate ./...
	go build -o ./bin/$(BIN_NAME) ./cmd/$(BIN_NAME)

init: build
	mkdir -p $(LOCAL_CONFIG_DIR) $(LOCAL_DATA_DIR)
	./bin/$(BIN_NAME) config init \
		--config-dir "$(LOCAL_CONFIG_DIR)" \
		--data-dir "$(LOCAL_DATA_DIR)" \
		--public-url "$(LOCAL_BASE_URL)" \
		--issuer-domain "$(LOCAL_ISSUER)" \
		--port "$(LOCAL_PORT)" \
		--dev-mode
	./bin/$(BIN_NAME) init \
		--config-dir "$(LOCAL_CONFIG_DIR)" \
		--data-dir "$(LOCAL_DATA_DIR)"
	./bin/$(BIN_NAME) env create local \
		--config-dir "$(LOCAL_CONFIG_DIR)" \
		--base-url "$(LOCAL_BASE_URL)" \
		--api-key "$$(tr -d '\n' < "$(LOCAL_CONFIG_DIR)/secrets/api_key")"

run-local: build
	./bin/$(BIN_NAME) serve \
		--config-dir "$(LOCAL_CONFIG_DIR)" \
		--data-dir "$(LOCAL_DATA_DIR)" \
		--verbose

test: build
	go test ./...
