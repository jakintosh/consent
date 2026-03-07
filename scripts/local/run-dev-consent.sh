#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

cd "${REPO_ROOT}"

LOCAL_TEST_ROOT="${LOCAL_TEST_ROOT:-${REPO_ROOT}/data/.local/consent-localtest}"
LOCAL_TEST_PORT="${LOCAL_TEST_PORT:-9001}"
LOCAL_TEST_BASE_URL="${LOCAL_TEST_BASE_URL:-http://localhost:${LOCAL_TEST_PORT}}"
LOCAL_TEST_ISSUER="${LOCAL_TEST_ISSUER:-localhost}"

DATA_DIR="${LOCAL_TEST_ROOT}/data"
SECRETS_DIR="${LOCAL_TEST_ROOT}/secrets"
CONFIG_DIR="${LOCAL_TEST_ROOT}/config"

DB_PATH="${DATA_DIR}/dev.db"
API_KEY_PATH="${SECRETS_DIR}/api_key"

mkdir -p "${DATA_DIR}" "${SECRETS_DIR}" "${CONFIG_DIR}"

if [ ! -f "${SECRETS_DIR}/signing_key" ] || [ ! -f "${SECRETS_DIR}/verification_key.der" ]; then
	go run ./cmd/keygen -out "${SECRETS_DIR}"
fi

if [ ! -s "${API_KEY_PATH}" ]; then
	go run ./cmd/consent env create local \
		--base-url "${LOCAL_TEST_BASE_URL}" \
		--bootstrap \
		--config-dir "${CONFIG_DIR}" > "${API_KEY_PATH}"
fi

echo "Starting consent dev server at ${LOCAL_TEST_BASE_URL}"
echo "  root: ${LOCAL_TEST_ROOT}"

exec go run ./cmd/consent serve \
	--db-path "${DB_PATH}" \
	--issuer-domain "${LOCAL_TEST_ISSUER}" \
	--port "${LOCAL_TEST_PORT}" \
	--credentials-dir "${SECRETS_DIR}" \
	--public-url "${LOCAL_TEST_BASE_URL}" \
	--dev-mode \
	--verbose
