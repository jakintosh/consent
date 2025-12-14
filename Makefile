GO ?= go
BIN_DIR ?= bin

CONSENT_BIN := $(BIN_DIR)/consent
CLIENT_BIN := $(BIN_DIR)/client
TESTSERVER_BIN := $(BIN_DIR)/consent-testserver

.PHONY: all build build-consent build-client build-testserver test clean

all: build

build: build-consent build-client build-testserver

build-consent: $(CONSENT_BIN)

build-client: $(CLIENT_BIN)

build-testserver: $(TESTSERVER_BIN)

$(CONSENT_BIN): cmd/consent/*.go
	$(GO) build -o $@ ./cmd/consent

$(CLIENT_BIN): cmd/client/*.go
	$(GO) build -o $@ ./cmd/client

$(TESTSERVER_BIN): cmd/consent-testserver/*.go
	$(GO) build -o $@ ./cmd/consent-testserver

test: $(TESTSERVER_BIN)
	CONSENT_TESTSERVER_BIN=$(abspath $(TESTSERVER_BIN)) $(GO) test ./...

clean:
	rm -rf $(BIN_DIR)
