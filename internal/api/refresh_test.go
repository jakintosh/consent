package api_test

import (
	"net/http"
	"testing"

	"git.sr.ht/~jakintosh/consent/internal/api"
	"git.sr.ht/~jakintosh/consent/internal/testutil"
)

func TestRefresh_Success(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)

	// setup env
	env.RegisterTestUser(t, "alice", "password")
	token := env.StoreTestRefreshToken(t, "alice", []string{"test-audience"})

	// valid refresh returns new access and refresh tokens
	body := `{
		"refreshToken": "` + token.Encoded() + `"
	}`
	var response api.RefreshResponse
	result := testutil.PostJSON(env.Router, "/refresh", body, &response)
	testutil.ExpectStatus(t, http.StatusOK, result)
	if response.AccessToken == "" {
		t.Error("expected non-empty access token")
	}
	if response.RefreshToken == "" {
		t.Error("expected non-empty refresh token")
	}
}

func TestRefresh_InvalidToken(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)

	// malformed token returns 400
	body := `{
		"refreshToken": "invalid-token"
	}`
	result := testutil.PostJSON(env.Router, "/refresh", body, nil)
	testutil.ExpectStatus(t, http.StatusBadRequest, result)
}

func TestRefresh_TokenNotInStore(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)

	// setup env
	env.RegisterTestUser(t, "alice", "password")
	token := env.IssueTestRefreshToken(t, "alice", []string{"test-audience"})

	// valid token not in store returns 400
	body := `{
		"refreshToken": "` + token.Encoded() + `"
	}`
	result := testutil.PostJSON(env.Router, "/refresh", body, nil)
	testutil.ExpectStatus(t, http.StatusBadRequest, result)
}

func TestRefresh_InvalidatesOldToken(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)

	// setup env
	env.RegisterTestUser(t, "alice", "password")
	token := env.StoreTestRefreshToken(t, "alice", []string{"test-audience"})

	// first refresh succeeds
	body := `{
		"refreshToken": "` + token.Encoded() + `"
	}`
	result := testutil.PostJSON(env.Router, "/refresh", body, nil)
	testutil.ExpectStatus(t, http.StatusOK, result)

	// second refresh with same token fails (token was rotated)
	result = testutil.PostJSON(env.Router, "/refresh", body, nil)
	testutil.ExpectStatus(t, http.StatusBadRequest, result)
}

func TestRefresh_InvalidJSON(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)

	// malformed JSON returns 400
	result := testutil.PostJSON(env.Router, "/refresh", "bad-json", nil)
	testutil.ExpectStatus(t, http.StatusBadRequest, result)
}

func TestRefresh_NewTokenCanBeUsed(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)

	// setup env
	env.RegisterTestUser(t, "alice", "password")
	token := env.StoreTestRefreshToken(t, "alice", []string{"test-audience"})

	// first refresh returns new tokens
	body := `{
		"refreshToken": "` + token.Encoded() + `"
	}`
	var response1 api.RefreshResponse
	result := testutil.PostJSON(env.Router, "/refresh", body, &response1)
	testutil.ExpectStatus(t, http.StatusOK, result)

	// new refresh token can be used for another refresh
	body2 := `{
		"refreshToken": "` + response1.RefreshToken + `"
	}`
	var response2 api.RefreshResponse
	result = testutil.PostJSON(env.Router, "/refresh", body2, &response2)
	testutil.ExpectStatus(t, http.StatusOK, result)
	if response2.AccessToken == "" {
		t.Error("second refresh should return access token")
	}
}
