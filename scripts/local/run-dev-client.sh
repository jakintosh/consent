#!/usr/bin/env bash
set -euo pipefail

if [ "$#" -lt 2 ] || [ "$#" -gt 3 ]; then
	echo "usage: $0 <service-name> <port> [display-name]" >&2
	exit 1
fi

SERVICE_NAME="$1"
CLIENT_PORT="$2"
DISPLAY_NAME="${3:-${SERVICE_NAME}}"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

cd "${REPO_ROOT}"

LOCAL_TEST_ROOT="${LOCAL_TEST_ROOT:-${REPO_ROOT}/data/.local/consent-localtest}"
LOCAL_TEST_BASE_URL="${LOCAL_TEST_BASE_URL:-http://localhost:9001}"
LOCAL_TEST_ISSUER="${LOCAL_TEST_ISSUER:-localhost}"
LOCAL_CLIENT_HOST="${LOCAL_CLIENT_HOST:-localhost}"

SECRETS_DIR="${LOCAL_TEST_ROOT}/secrets"
API_KEY_PATH="${SECRETS_DIR}/api_key"
VERIFICATION_KEY_PATH="${SECRETS_DIR}/verification_key.der"

if [ ! -s "${API_KEY_PATH}" ]; then
	echo "missing API key: ${API_KEY_PATH}" >&2
	echo "run scripts/local/run-dev-consent.sh first" >&2
	exit 1
fi

if [ ! -f "${VERIFICATION_KEY_PATH}" ]; then
	echo "missing verification key: ${VERIFICATION_KEY_PATH}" >&2
	echo "run scripts/local/run-dev-consent.sh first" >&2
	exit 1
fi

API_KEY="$(<"${API_KEY_PATH}")"
AUDIENCE="${LOCAL_CLIENT_HOST}:${CLIENT_PORT}"
REDIRECT_URL="http://${LOCAL_CLIENT_HOST}:${CLIENT_PORT}/auth/callback"

if ! go run ./cmd/consent api services create "${SERVICE_NAME}" \
	--display "${DISPLAY_NAME}" \
	--audience "${AUDIENCE}" \
	--redirect "${REDIRECT_URL}" \
	--base-url "${LOCAL_TEST_BASE_URL}" \
	--api-key "${API_KEY}" >/dev/null 2>&1; then
	if ! go run ./cmd/consent api services update "${SERVICE_NAME}" \
		--display "${DISPLAY_NAME}" \
		--audience "${AUDIENCE}" \
		--redirect "${REDIRECT_URL}" \
		--base-url "${LOCAL_TEST_BASE_URL}" \
		--api-key "${API_KEY}" >/dev/null 2>&1; then
		echo "failed to register service ${SERVICE_NAME}" >&2
		exit 1
	fi
fi

echo "Starting dev client ${SERVICE_NAME} on :${CLIENT_PORT}"

exec go run ./examples/dev-client \
	--auth-url "${LOCAL_TEST_BASE_URL}" \
	--issuer-domain "${LOCAL_TEST_ISSUER}" \
	--port "${CLIENT_PORT}" \
	--service "${SERVICE_NAME}" \
	--audience "${AUDIENCE}" \
	--verification-key "${VERIFICATION_KEY_PATH}" \
	--verbose
