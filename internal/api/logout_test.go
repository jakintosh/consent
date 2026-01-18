package api_test

import (
	"net/http"
	"testing"

	"git.sr.ht/~jakintosh/command-go/pkg/wire"
	"git.sr.ht/~jakintosh/consent/internal/testutil"
)

func TestLogout_Success(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)

	// setup env
	env.RegisterTestUser(t, "alice", "password")
	token := env.StoreTestRefreshToken(t, "alice", []string{"test-audience"})

	// happy path token logout succeeds
	body := `{
		"refreshToken": "` + token.Encoded() + `"
	}`
	result := wire.TestPost[any](env.Router, "/logout", body, jsonHeader)
	result.ExpectStatus(t, http.StatusOK)
}

func TestLogout_TokenNotFound(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)

	// logout with invalid token fails
	body := `{
		"refreshToken": "nonexistent-token"
	}`
	result := wire.TestPost[any](env.Router, "/logout", body, jsonHeader)
	result.ExpectStatus(t, http.StatusBadRequest)
	result.ExpectError(t)
}

func TestLogout_InvalidatesToken(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)

	// setup env
	env.RegisterTestUser(t, "alice", "password")
	token := env.StoreTestRefreshToken(t, "alice", []string{"test-audience"})

	// valid logout succeeds
	logoutBody := `{
		"refreshToken": "` + token.Encoded() + `"
	}`
	result := wire.TestPost[any](env.Router, "/logout", logoutBody, jsonHeader)
	result.ExpectStatus(t, http.StatusOK)

	// refresh should now fail
	refreshBody := `{
		"refreshToken": "` + token.Encoded() + `"
	}`
	refreshResult := wire.TestPost[any](env.Router, "/refresh", refreshBody, jsonHeader)
	refreshResult.ExpectStatus(t, http.StatusBadRequest)
	refreshResult.ExpectError(t)
}

func TestLogout_InvalidJSON(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)

	// logout with malformed json fails
	result := wire.TestPost[any](env.Router, "/logout", "bad-json", jsonHeader)
	result.ExpectStatus(t, http.StatusBadRequest)
	result.ExpectError(t)
}

func TestLogout_DoubleLogout(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)

	// setup env
	env.RegisterTestUser(t, "alice", "password")
	token := env.StoreTestRefreshToken(t, "alice", []string{"test-audience"})

	// first logout succeeds
	body := `{
		"refreshToken": "` + token.Encoded() + `"
	}`
	result := wire.TestPost[any](env.Router, "/logout", body, jsonHeader)
	result.ExpectStatus(t, http.StatusOK)

	// second logout fails
	second := wire.TestPost[any](env.Router, "/logout", body, jsonHeader)
	second.ExpectStatus(t, http.StatusBadRequest)
	second.ExpectError(t)
}

func TestLogout_EmptyToken(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)

	// logout with empty token fails
	body := `{
		"refreshToken": ""
	}`
	result := wire.TestPost[any](env.Router, "/logout", body, jsonHeader)
	result.ExpectStatus(t, http.StatusBadRequest)
	result.ExpectError(t)
}
