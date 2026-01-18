package service_test

import (
	"net/http"
	"testing"

	"git.sr.ht/~jakintosh/command-go/pkg/wire"
	"git.sr.ht/~jakintosh/consent/internal/service"
	"git.sr.ht/~jakintosh/consent/internal/testutil"
)

func TestAPIRefresh_Success(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)

	// setup env
	env.RegisterTestUser(t, "alice", "password")
	token := env.StoreTestRefreshToken(t, "alice", []string{"test-audience"})

	// valid refresh returns new access and refresh tokens
	body := `{
		"refreshToken": "` + token.Encoded() + `"
	}`
	result := wire.TestPost[service.RefreshResponse](env.Router, "/refresh", body, jsonHeader)
	response := result.ExpectOK(t)
	if response.AccessToken == "" {
		t.Error("expected non-empty access token")
	}
	if response.RefreshToken == "" {
		t.Error("expected non-empty refresh token")
	}
}

func TestAPIRefresh_InvalidToken(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)

	// malformed token returns 400
	body := `{
		"refreshToken": "invalid-token"
	}`
	result := wire.TestPost[any](env.Router, "/refresh", body, jsonHeader)
	result.ExpectStatus(t, http.StatusBadRequest)
	result.ExpectError(t)
}

func TestAPIRefresh_TokenNotInStore(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)

	// setup env
	env.RegisterTestUser(t, "alice", "password")
	token := env.IssueTestRefreshToken(t, "alice", []string{"test-audience"})

	// valid token not in store returns 400
	body := `{
		"refreshToken": "` + token.Encoded() + `"
	}`
	result := wire.TestPost[any](env.Router, "/refresh", body, jsonHeader)
	result.ExpectStatus(t, http.StatusBadRequest)
	result.ExpectError(t)
}

func TestAPIRefresh_InvalidatesOldToken(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)

	// setup env
	env.RegisterTestUser(t, "alice", "password")
	token := env.StoreTestRefreshToken(t, "alice", []string{"test-audience"})

	// first refresh succeeds
	body := `{
		"refreshToken": "` + token.Encoded() + `"
	}`
	result := wire.TestPost[service.RefreshResponse](env.Router, "/refresh", body, jsonHeader)
	result.ExpectOK(t)

	// second refresh with same token fails (token was rotated)
	badResult := wire.TestPost[any](env.Router, "/refresh", body, jsonHeader)
	badResult.ExpectStatus(t, http.StatusBadRequest)
	badResult.ExpectError(t)
}

func TestAPIRefresh_InvalidJSON(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)

	// malformed JSON returns 400
	result := wire.TestPost[any](env.Router, "/refresh", "bad-json", jsonHeader)
	result.ExpectStatus(t, http.StatusBadRequest)
	result.ExpectError(t)
}

func TestAPIRefresh_NewTokenCanBeUsed(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)

	// setup env
	env.RegisterTestUser(t, "alice", "password")
	token := env.StoreTestRefreshToken(t, "alice", []string{"test-audience"})

	// first refresh returns new tokens
	body := `{
		"refreshToken": "` + token.Encoded() + `"
	}`
	result := wire.TestPost[service.RefreshResponse](env.Router, "/refresh", body, jsonHeader)
	response1 := result.ExpectOK(t)

	// new refresh token can be used for another refresh
	body2 := `{
		"refreshToken": "` + response1.RefreshToken + `"
	}`
	result = wire.TestPost[service.RefreshResponse](env.Router, "/refresh", body2, jsonHeader)
	response2 := result.ExpectOK(t)
	if response2.AccessToken == "" {
		t.Error("second refresh should return access token")
	}
}
