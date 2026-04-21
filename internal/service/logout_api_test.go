package service_test

import (
	"net/http"
	"testing"

	"git.sr.ht/~jakintosh/command-go/pkg/wire"
	"git.sr.ht/~jakintosh/consent/internal/testutil"
)

func TestAPILogout_Success(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)

	// setup env
	env.RegisterTestUser(t, "alice", "password")
	token := env.StoreTestRefreshToken(t, "alice", []string{"test-audience"})

	// happy path token logout succeeds
	body := `{
		"refreshToken": "` + token.Encoded() + `"
	}`
	result := wire.TestPost[any](env.Router, "/auth/logout", body, jsonHeader)
	result.ExpectStatus(t, http.StatusOK)
}

func TestAPILogout_TokenNotFound(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)

	// logout with invalid token fails
	body := `{
		"refreshToken": "nonexistent-token"
	}`
	result := wire.TestPost[any](env.Router, "/auth/logout", body, jsonHeader)
	result.ExpectStatus(t, http.StatusBadRequest)
	result.ExpectError(t)
}

func TestAPILogout_InvalidatesToken(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)

	// setup env
	env.RegisterTestUser(t, "alice", "password")
	token := env.StoreTestRefreshToken(t, "alice", []string{"test-audience"})

	// valid logout succeeds
	logoutBody := `{
		"refreshToken": "` + token.Encoded() + `"
	}`
	result := wire.TestPost[any](env.Router, "/auth/logout", logoutBody, jsonHeader)
	result.ExpectStatus(t, http.StatusOK)

	// refresh should now fail
	refreshBody := `{
		"refreshToken": "` + token.Encoded() + `"
	}`
	refreshResult := wire.TestPost[any](env.Router, "/auth/refresh", refreshBody, jsonHeader)
	refreshResult.ExpectStatus(t, http.StatusBadRequest)
	refreshResult.ExpectError(t)
}

func TestAPILogout_InvalidJSON(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)

	// logout with malformed json fails
	result := wire.TestPost[any](env.Router, "/auth/logout", "bad-json", jsonHeader)
	result.ExpectStatus(t, http.StatusBadRequest)
	result.ExpectError(t)
}

func TestAPILogout_DoubleLogout(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)

	// setup env
	env.RegisterTestUser(t, "alice", "password")
	token := env.StoreTestRefreshToken(t, "alice", []string{"test-audience"})

	// first logout succeeds
	body := `{
		"refreshToken": "` + token.Encoded() + `"
	}`
	result := wire.TestPost[any](env.Router, "/auth/logout", body, jsonHeader)
	result.ExpectStatus(t, http.StatusOK)

	// second logout fails
	second := wire.TestPost[any](env.Router, "/auth/logout", body, jsonHeader)
	second.ExpectStatus(t, http.StatusBadRequest)
	second.ExpectError(t)
}

func TestAPILogout_EmptyToken(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)

	// logout with empty token fails
	body := `{
		"refreshToken": ""
	}`
	result := wire.TestPost[any](env.Router, "/auth/logout", body, jsonHeader)
	result.ExpectStatus(t, http.StatusBadRequest)
	result.ExpectError(t)
}
