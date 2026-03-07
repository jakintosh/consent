#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

cd "${REPO_ROOT}"

LOCAL_TEST_ROOT="${LOCAL_TEST_ROOT:-${REPO_ROOT}/data/.local/consent-localtest}"
LOCAL_TEST_PORT="${LOCAL_TEST_PORT:-9001}"
LOCAL_TEST_BASE_URL="${LOCAL_TEST_BASE_URL:-http://localhost:${LOCAL_TEST_PORT}}"
LOCAL_TEST_ISSUER="${LOCAL_TEST_ISSUER:-localhost}"

cleanup() {
	trap - INT TERM EXIT
	jobs -pr | xargs -r kill 2>/dev/null || true
	wait || true
}

trap cleanup INT TERM EXIT

start_client() {
	service_name="$1"
	client_port="$2"
	display_name="$3"

	./scripts/local/run-dev-client.sh "${service_name}" "${client_port}" "${display_name}" &
	client_pid=$!

	until curl --silent --show-error --fail "http://localhost:${client_port}/" >/dev/null 2>&1; do
		if ! kill -0 "${client_pid}" 2>/dev/null; then
			echo "client failed to start: ${service_name}" >&2
			wait "${client_pid}"
			exit 1
		fi
		sleep 0.2
	done
}

./scripts/local/run-dev-consent.sh &
consent_pid=$!

printf "Waiting for consent at %s ...\n" "${LOCAL_TEST_BASE_URL}"
until curl --silent --show-error --fail "${LOCAL_TEST_BASE_URL}/" >/dev/null 2>&1; do
	if ! kill -0 "${consent_pid}" 2>/dev/null; then
		echo "consent failed to start" >&2
		wait "${consent_pid}"
		exit 1
	fi
	sleep 0.2
done

start_client "example-a@localhost" "10000" "Example A"
start_client "example-b@localhost" "10001" "Example B"
start_client "example-c@localhost" "10002" "Example C"

echo "Local test stack running. Press Ctrl-C to stop all processes."
wait
