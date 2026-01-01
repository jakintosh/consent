package api_test

import (
	"net/http"
	"testing"

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
	result := testutil.PostJSON(env.Router, "/logout", body, nil)
	testutil.ExpectStatus(t, http.StatusOK, result)
}

func TestLogout_TokenNotFound(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)

	// logout with invalid token fails
	body := `{
		"refreshToken": "nonexistent-token"
	}`
	result := testutil.PostJSON(env.Router, "/logout", body, nil)
	testutil.ExpectStatus(t, http.StatusBadRequest, result)
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
	result := testutil.PostJSON(env.Router, "/logout", logoutBody, nil)
	testutil.ExpectStatus(t, http.StatusOK, result)

	// refresh should now fail
	refreshBody := `{
		"refreshToken": "` + token.Encoded() + `"
	}`
	result = testutil.PostJSON(env.Router, "/refresh", refreshBody, nil)
	testutil.ExpectStatus(t, http.StatusBadRequest, result)
}

func TestLogout_InvalidJSON(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)

	// logout with malformed json fails
	result := testutil.PostJSON(env.Router, "/logout", "bad-json", nil)
	testutil.ExpectStatus(t, http.StatusBadRequest, result)
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
	result := testutil.PostJSON(env.Router, "/logout", body, nil)
	testutil.ExpectStatus(t, http.StatusOK, result)

	// second logout fails
	result = testutil.PostJSON(env.Router, "/logout", body, nil)
	testutil.ExpectStatus(t, http.StatusBadRequest, result)
}

func TestLogout_EmptyToken(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)

	// logout with empty token fails
	body := `{
		"refreshToken": ""
	}`
	result := testutil.PostJSON(env.Router, "/logout", body, nil)
	testutil.ExpectStatus(t, http.StatusBadRequest, result)
}
