BIN_NAME=consent
DEV_CLIENT_BIN_NAME=dev-client

CONSENT_BIN=./bin/$(BIN_NAME)
DEV_CLIENT_BIN=./bin/$(DEV_CLIENT_BIN_NAME)

LOCAL_CONFIG_DIR=./config
LOCAL_DATA_DIR=./data
LOCAL_BASE_URL?=http://localhost:9001
LOCAL_ISSUER?=localhost
LOCAL_PORT?=9001

MOCK_ROOT=./mock
MOCK_CONFIG_DIR=$(MOCK_ROOT)/config
MOCK_DATA_DIR=$(MOCK_ROOT)/data
MOCK_LOG_DIR=$(MOCK_ROOT)/logs

MOCK_CONSENT_URL?=http://localhost:9000
MOCK_ISSUER?=localhost
MOCK_CONSENT_PORT?=9000

MOCK_CLIENT_1_HOST?=mock1.localhost
MOCK_CLIENT_1_PORT?=9001
MOCK_CLIENT_1_INTEGRATION?=mock1

MOCK_CLIENT_2_HOST?=mock2.localhost
MOCK_CLIENT_2_PORT?=9002
MOCK_CLIENT_2_INTEGRATION?=mock2

MOCK_CLIENT_3_HOST?=mock3.localhost
MOCK_CLIENT_3_PORT?=9003
MOCK_CLIENT_3_INTEGRATION?=mock3

MOCK_DEMO_USER?=alice
MOCK_DEMO_PASSWORD?=alice123

.PHONY: build init run-local mock-deployment-init mock-deployment test

build:
	go generate ./...
	go build -o $(CONSENT_BIN) ./cmd/$(BIN_NAME)
	go build -o $(DEV_CLIENT_BIN) ./cmd/$(DEV_CLIENT_BIN_NAME)

init: build
	mkdir -p $(LOCAL_CONFIG_DIR) $(LOCAL_DATA_DIR)
	$(CONSENT_BIN) config init \
		--config-dir "$(LOCAL_CONFIG_DIR)" \
		--data-dir "$(LOCAL_DATA_DIR)" \
		--public-url "$(LOCAL_BASE_URL)" \
		--authority-domain "$(LOCAL_ISSUER)" \
		--port "$(LOCAL_PORT)" \
		--dev-mode
	$(CONSENT_BIN) init \
		--config-dir "$(LOCAL_CONFIG_DIR)" \
		--data-dir "$(LOCAL_DATA_DIR)"
	$(CONSENT_BIN) env create local \
		--config-dir "$(LOCAL_CONFIG_DIR)" \
		--base-url "$(LOCAL_BASE_URL)" \
		--api-key "$$(tr -d '\n' < "$(LOCAL_CONFIG_DIR)/secrets/api_key")"

run-local: build
	$(CONSENT_BIN) serve \
		--config-dir "$(LOCAL_CONFIG_DIR)" \
		--data-dir "$(LOCAL_DATA_DIR)" \
		--verbose

mock-deployment-init: build
	rm -rf $(MOCK_ROOT)
	mkdir -p $(MOCK_CONFIG_DIR) $(MOCK_DATA_DIR) $(MOCK_LOG_DIR)
	$(CONSENT_BIN) config init \
		--force \
		--config-dir "$(MOCK_CONFIG_DIR)" \
		--data-dir "$(MOCK_DATA_DIR)" \
		--public-url "$(MOCK_CONSENT_URL)" \
		--authority-domain "$(MOCK_ISSUER)" \
		--port "$(MOCK_CONSENT_PORT)"
	$(CONSENT_BIN) init \
		--config-dir "$(MOCK_CONFIG_DIR)" \
		--data-dir "$(MOCK_DATA_DIR)"
	$(CONSENT_BIN) env create local \
		--config-dir "$(MOCK_CONFIG_DIR)" \
		--base-url "$(MOCK_CONSENT_URL)" \
		--api-key "$$(tr -d '\n' < "$(MOCK_CONFIG_DIR)/secrets/api_key")"
	@set -eu; \
		$(CONSENT_BIN) serve \
			--config-dir "$(MOCK_CONFIG_DIR)" \
			--data-dir "$(MOCK_DATA_DIR)" \
			--insecure-cookies \
			>"$(MOCK_LOG_DIR)/consent-init.log" 2>&1 & \
		server_pid=$$!; \
		trap 'kill $$server_pid >/dev/null 2>&1 || true' EXIT INT TERM; \
		until curl -fsS "$(MOCK_CONSENT_URL)/" >/dev/null 2>&1; do \
			if ! kill -0 $$server_pid >/dev/null 2>&1; then \
				printf '%s\n' "consent init server exited before becoming ready; see $(MOCK_LOG_DIR)/consent-init.log"; \
				exit 1; \
			fi; \
			sleep 1; \
		done; \
		$(CONSENT_BIN) api users create "$(MOCK_DEMO_USER)" \
			--config-dir "$(MOCK_CONFIG_DIR)" \
			--password "$(MOCK_DEMO_PASSWORD)" \
			--role admin; \
		register_integration() { \
			name="$$1"; \
			display="$$2"; \
			audience="$$3"; \
			redirect="$$4"; \
			if $(CONSENT_BIN) api integrations get "$$name" --config-dir "$(MOCK_CONFIG_DIR)" >/dev/null 2>&1; then \
				$(CONSENT_BIN) api integrations update "$$name" \
					--config-dir "$(MOCK_CONFIG_DIR)" \
					--display "$$display" \
					--audience "$$audience" \
					--redirect "$$redirect"; \
			else \
				$(CONSENT_BIN) api integrations create "$$name" \
					--config-dir "$(MOCK_CONFIG_DIR)" \
					--display "$$display" \
					--audience "$$audience" \
					--redirect "$$redirect"; \
			fi; \
		}; \
		register_integration "$(MOCK_CLIENT_1_INTEGRATION)" "Mock Client 1" \
			"$(MOCK_CLIENT_1_HOST):$(MOCK_CLIENT_1_PORT)" \
			"http://$(MOCK_CLIENT_1_HOST):$(MOCK_CLIENT_1_PORT)/auth/callback"; \
		register_integration "$(MOCK_CLIENT_2_INTEGRATION)" "Mock Client 2" \
			"$(MOCK_CLIENT_2_HOST):$(MOCK_CLIENT_2_PORT)" \
			"http://$(MOCK_CLIENT_2_HOST):$(MOCK_CLIENT_2_PORT)/auth/callback"; \
		register_integration "$(MOCK_CLIENT_3_INTEGRATION)" "Mock Client 3" \
			"$(MOCK_CLIENT_3_HOST):$(MOCK_CLIENT_3_PORT)" \
			"http://$(MOCK_CLIENT_3_HOST):$(MOCK_CLIENT_3_PORT)/auth/callback"; \
		kill $$server_pid; \
		wait $$server_pid || true; \
		trap - EXIT INT TERM

mock-deployment: mock-deployment-init
	@mkdir -p "$(MOCK_LOG_DIR)"
	@printf '%s\n' \
		"Consent: $(MOCK_CONSENT_URL)" \
		"Mock 1:  http://$(MOCK_CLIENT_1_HOST):$(MOCK_CLIENT_1_PORT)" \
		"Mock 2:  http://$(MOCK_CLIENT_2_HOST):$(MOCK_CLIENT_2_PORT)" \
		"Mock 3:  http://$(MOCK_CLIENT_3_HOST):$(MOCK_CLIENT_3_PORT)" \
		"Demo user: $(MOCK_DEMO_USER) / $(MOCK_DEMO_PASSWORD)"
	@set -eu; \
		$(CONSENT_BIN) serve \
			--config-dir "$(MOCK_CONFIG_DIR)" \
			--data-dir "$(MOCK_DATA_DIR)" \
			--insecure-cookies \
			--verbose \
			>"$(MOCK_LOG_DIR)/consent.log" 2>&1 & \
		consent_pid=$$!; \
		trap 'kill $$consent_pid $$client1_pid $$client2_pid $$client3_pid >/dev/null 2>&1 || true' EXIT INT TERM; \
		until curl -fsS "$(MOCK_CONSENT_URL)/" >/dev/null 2>&1; do \
			if ! kill -0 $$consent_pid >/dev/null 2>&1; then \
				printf '%s\n' "consent server exited before becoming ready; see $(MOCK_LOG_DIR)/consent.log"; \
				exit 1; \
			fi; \
			sleep 1; \
		done; \
		$(DEV_CLIENT_BIN) \
			--auth-url "$(MOCK_CONSENT_URL)" \
			--authority-domain "$(MOCK_ISSUER)" \
			--port "$(MOCK_CLIENT_1_PORT)" \
			--integration "$(MOCK_CLIENT_1_INTEGRATION)" \
			--audience "$(MOCK_CLIENT_1_HOST):$(MOCK_CLIENT_1_PORT)" \
			--config-dir "$(MOCK_CONFIG_DIR)" \
			--verbose \
			>"$(MOCK_LOG_DIR)/mock1.log" 2>&1 & \
		client1_pid=$$!; \
		$(DEV_CLIENT_BIN) \
			--auth-url "$(MOCK_CONSENT_URL)" \
			--authority-domain "$(MOCK_ISSUER)" \
			--port "$(MOCK_CLIENT_2_PORT)" \
			--integration "$(MOCK_CLIENT_2_INTEGRATION)" \
			--audience "$(MOCK_CLIENT_2_HOST):$(MOCK_CLIENT_2_PORT)" \
			--config-dir "$(MOCK_CONFIG_DIR)" \
			--verbose \
			>"$(MOCK_LOG_DIR)/mock2.log" 2>&1 & \
		client2_pid=$$!; \
		$(DEV_CLIENT_BIN) \
			--auth-url "$(MOCK_CONSENT_URL)" \
			--authority-domain "$(MOCK_ISSUER)" \
			--port "$(MOCK_CLIENT_3_PORT)" \
			--integration "$(MOCK_CLIENT_3_INTEGRATION)" \
			--audience "$(MOCK_CLIENT_3_HOST):$(MOCK_CLIENT_3_PORT)" \
			--config-dir "$(MOCK_CONFIG_DIR)" \
			--verbose \
			>"$(MOCK_LOG_DIR)/mock3.log" 2>&1 & \
		client3_pid=$$!; \
		wait $$consent_pid $$client1_pid $$client2_pid $$client3_pid

test: build
	go test ./...
