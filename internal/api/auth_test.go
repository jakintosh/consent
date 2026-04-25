package api_test

import (
	"net/http"
	"strings"
	"testing"

	"git.sr.ht/~jakintosh/command-go/pkg/wire"
	"git.sr.ht/~jakintosh/consent/internal/api"
	"git.sr.ht/~jakintosh/consent/internal/service"
	"git.sr.ht/~jakintosh/consent/internal/testutil"
	"git.sr.ht/~jakintosh/consent/pkg/tokens"
)

const consentAudience = "test.consent.local"

var formHeader = wire.TestHeader{
	Key:   "Content-Type",
	Value: "application/x-www-form-urlencoded",
}

func authHeader(token *tokens.AccessToken) wire.TestHeader {
	return wire.TestHeader{
		Key:   "Authorization",
		Value: "Bearer " + token.Encoded(),
	}
}

func TestAPILogin_JSONSuccess(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)
	env.RegisterTestUser(t, "alice", "password123")

	body := `{
		"handle": "alice",
		"secret": "password123",
		"integration": "consent"
	}`
	result := wire.TestPost[any](env.Router, "/auth/login", body, jsonHeader)
	result.ExpectStatus(t, http.StatusSeeOther)
	location := result.Headers.Get("Location")
	if location == "" {
		t.Fatal("expected Location header in redirect")
	}
	if !strings.Contains(location, "auth_code=") {
		t.Errorf("redirect URL missing auth_code: %s", location)
	}
}

func TestAPILogin_JSONRedirectTarget(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)
	env.RegisterTestUser(t, "alice", "password123")

	body := `{
		"handle": "alice",
		"secret": "password123",
		"integration": "consent"
	}`
	result := wire.TestPost[any](env.Router, "/auth/login", body, jsonHeader)
	result.ExpectStatus(t, http.StatusSeeOther)
	location := result.Headers.Get("Location")
	if location == "" {
		t.Fatal("expected Location header in redirect")
	}
	if !strings.Contains(location, "consent.test") {
		t.Errorf("redirect should be to integration URL, got: %s", location)
	}
	if !strings.Contains(location, "/auth/callback") {
		t.Errorf("redirect should include auth callback path, got: %s", location)
	}
}

func TestAPILogin_FormSuccess(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)
	env.RegisterTestUser(t, "alice", "password123")

	body := "handle=alice&secret=password123&integration=consent"
	result := wire.TestPost[any](env.Router, "/auth/login", body, formHeader)
	result.ExpectStatus(t, http.StatusSeeOther)
	location := result.Headers.Get("Location")
	if location == "" {
		t.Fatal("expected Location header in redirect")
	}
	if !strings.Contains(location, "auth_code=") {
		t.Errorf("redirect URL missing auth_code: %s", location)
	}
}

func TestAPILogin_FormMissingFields(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)

	body := "handle=alice&secret=password123"
	result := wire.TestPost[any](env.Router, "/auth/login", body, formHeader)
	result.ExpectStatusError(t, http.StatusBadRequest)
}

func TestAPILogin_UnsupportedContentType(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)

	contentTypeHeader := wire.TestHeader{
		Key:   "Content-Type",
		Value: "text/plain",
	}
	result := wire.TestPost[any](env.Router, "/auth/login", "data", contentTypeHeader)
	result.ExpectStatusError(t, http.StatusUnsupportedMediaType)
}

func TestAPILogin_InvalidJSON(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)

	result := wire.TestPost[any](env.Router, "/auth/login", "not-json", jsonHeader)
	result.ExpectStatusError(t, http.StatusBadRequest)
}

func TestAPILogin_InvalidCredentials(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)
	env.RegisterTestUser(t, "alice", "password123")

	body := `{
		"handle": "alice",
		"secret": "wrongpassword",
		"integration": "consent"
	}`
	result := wire.TestPost[any](env.Router, "/auth/login", body, jsonHeader)
	result.ExpectStatusError(t, http.StatusUnauthorized)
}

func TestAPILogin_UnknownUser(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)

	body := `{
		"handle": "unknown",
		"secret": "password",
		"integration": "consent"
	}`
	result := wire.TestPost[any](env.Router, "/auth/login", body, jsonHeader)
	result.ExpectStatusError(t, http.StatusUnauthorized)
}

func TestAPILogin_UnknownIntegration(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)
	env.RegisterTestUser(t, "alice", "password123")

	body := `{
		"handle": "alice",
		"secret": "password123",
		"integration": "unknown"
	}`
	result := wire.TestPost[any](env.Router, "/auth/login", body, jsonHeader)
	result.ExpectStatusError(t, http.StatusBadRequest)

	_, err := env.Service.GetIntegration(service.InternalIntegrationName)
	if err != nil {
		t.Fatalf("expected internal integration to exist: %v", err)
	}
}

func TestAPILogout_Success(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)
	env.RegisterTestUser(t, "alice", "password")
	token := env.StoreTestRefreshToken(t, "alice", []string{"test-audience"})

	body := `{
		"refreshToken": "` + token.Encoded() + `"
	}`
	result := wire.TestPost[any](env.Router, "/auth/logout", body, jsonHeader)
	result.ExpectStatus(t, http.StatusOK)
}

func TestAPILogout_InvalidatesToken(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)
	env.RegisterTestUser(t, "alice", "password")
	token := env.StoreTestRefreshToken(t, "alice", []string{"test-audience"})

	logoutBody := `{
		"refreshToken": "` + token.Encoded() + `"
	}`
	result := wire.TestPost[any](env.Router, "/auth/logout", logoutBody, jsonHeader)
	result.ExpectStatus(t, http.StatusOK)

	refreshBody := `{
		"refreshToken": "` + token.Encoded() + `"
	}`
	refreshResult := wire.TestPost[any](env.Router, "/auth/refresh", refreshBody, jsonHeader)
	refreshResult.ExpectStatusError(t, http.StatusBadRequest)
}

func TestAPILogout_TokenNotFound(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)

	body := `{
		"refreshToken": "nonexistent-token"
	}`
	result := wire.TestPost[any](env.Router, "/auth/logout", body, jsonHeader)
	result.ExpectStatusError(t, http.StatusBadRequest)
}

func TestAPILogout_InvalidJSON(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)

	result := wire.TestPost[any](env.Router, "/auth/logout", "bad-json", jsonHeader)
	result.ExpectStatusError(t, http.StatusBadRequest)
}

func TestAPILogout_EmptyToken(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)

	body := `{
		"refreshToken": ""
	}`
	result := wire.TestPost[any](env.Router, "/auth/logout", body, jsonHeader)
	result.ExpectStatusError(t, http.StatusBadRequest)
}

func TestAPILogout_DoubleLogout(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)
	env.RegisterTestUser(t, "alice", "password")
	token := env.StoreTestRefreshToken(t, "alice", []string{"test-audience"})

	body := `{
		"refreshToken": "` + token.Encoded() + `"
	}`
	result := wire.TestPost[any](env.Router, "/auth/logout", body, jsonHeader)
	result.ExpectStatus(t, http.StatusOK)

	second := wire.TestPost[any](env.Router, "/auth/logout", body, jsonHeader)
	second.ExpectStatusError(t, http.StatusBadRequest)
}

func TestAPIRefresh_Success(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)
	env.RegisterTestUser(t, "alice", "password")
	token := env.StoreTestRefreshToken(t, "alice", []string{"test-audience"})

	body := `{
		"refreshToken": "` + token.Encoded() + `"
	}`
	result := wire.TestPost[api.RefreshResponse](env.Router, "/auth/refresh", body, jsonHeader)
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

	body := `{
		"refreshToken": "invalid-token"
	}`
	result := wire.TestPost[any](env.Router, "/auth/refresh", body, jsonHeader)
	result.ExpectStatusError(t, http.StatusBadRequest)
}

func TestAPIRefresh_TokenNotInStore(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)
	env.RegisterTestUser(t, "alice", "password")
	token := env.IssueTestRefreshToken(t, "alice", []string{"test-audience"})

	body := `{
		"refreshToken": "` + token.Encoded() + `"
	}`
	result := wire.TestPost[any](env.Router, "/auth/refresh", body, jsonHeader)
	result.ExpectStatusError(t, http.StatusBadRequest)
}

func TestAPIRefresh_InvalidatesOldToken(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)
	env.RegisterTestUser(t, "alice", "password")
	token := env.StoreTestRefreshToken(t, "alice", []string{"test-audience"})

	body := `{
		"refreshToken": "` + token.Encoded() + `"
	}`
	result := wire.TestPost[api.RefreshResponse](env.Router, "/auth/refresh", body, jsonHeader)
	result.ExpectOK(t)

	badResult := wire.TestPost[any](env.Router, "/auth/refresh", body, jsonHeader)
	badResult.ExpectStatusError(t, http.StatusBadRequest)
}

func TestAPIRefresh_NewTokenCanBeUsed(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)
	env.RegisterTestUser(t, "alice", "password")
	token := env.StoreTestRefreshToken(t, "alice", []string{"test-audience"})

	body := `{
		"refreshToken": "` + token.Encoded() + `"
	}`
	result := wire.TestPost[api.RefreshResponse](env.Router, "/auth/refresh", body, jsonHeader)
	response1 := result.ExpectOK(t)

	body2 := `{
		"refreshToken": "` + response1.RefreshToken + `"
	}`
	result = wire.TestPost[api.RefreshResponse](env.Router, "/auth/refresh", body2, jsonHeader)
	response2 := result.ExpectOK(t)
	if response2.AccessToken == "" {
		t.Error("second refresh should return access token")
	}
}

func TestAPIRefresh_InvalidJSON(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)

	result := wire.TestPost[any](env.Router, "/auth/refresh", "bad-json", jsonHeader)
	result.ExpectStatusError(t, http.StatusBadRequest)
}

func TestAPIUserInfo_IdentityOnly(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)
	env.RegisterTestUser(t, "alice", "password")

	token := env.IssueTestAccessTokenWithScopes(t, "alice", []string{"test-audience", consentAudience}, []string{"identity"})
	result := wire.TestGet[api.UserInfo](env.Router, "/auth/userinfo", authHeader(token))
	response := result.ExpectOK(t)
	if response.Sub != token.Subject() {
		t.Fatalf("sub = %q, want %q", response.Sub, token.Subject())
	}
	if response.Profile != nil {
		t.Fatalf("profile = %#v, want nil", response.Profile)
	}
}

func TestAPIUserInfo_ProfileScope(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)
	env.RegisterTestUser(t, "alice", "password")

	token := env.IssueTestAccessTokenWithScopes(t, "alice", []string{"test-audience", consentAudience}, []string{"identity", "profile"})
	result := wire.TestGet[api.UserInfo](env.Router, "/auth/userinfo", authHeader(token))
	response := result.ExpectOK(t)
	if response.Sub != token.Subject() {
		t.Fatalf("sub = %q, want %q", response.Sub, token.Subject())
	}
	if response.Profile == nil || response.Profile.Handle != "alice" {
		t.Fatalf("profile handle = %#v, want alice", response.Profile)
	}
}

func TestAPIUserInfo_RequiresIdentityScope(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)
	env.RegisterTestUser(t, "alice", "password")

	token := env.IssueTestAccessTokenWithScopes(t, "alice", []string{"test-audience", consentAudience}, []string{"profile"})
	result := wire.TestGet[any](env.Router, "/auth/userinfo", authHeader(token))
	result.ExpectStatus(t, http.StatusForbidden)
}

func TestAPIUserInfo_RequiresBearerHeader(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)

	result := wire.TestGet[any](env.Router, "/auth/userinfo")
	result.ExpectStatusError(t, http.StatusBadRequest)
}

func TestAPIUserInfo_RequiresConsentAudience(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)
	env.RegisterTestUser(t, "alice", "password")

	token := env.IssueTestAccessTokenWithScopes(t, "alice", []string{"test-audience"}, []string{"identity"})
	result := wire.TestGet[any](env.Router, "/auth/userinfo", authHeader(token))
	result.ExpectStatus(t, http.StatusBadRequest)
}

func TestAPIUserInfo_InvalidBearerHeader(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)

	tests := []struct {
		name   string
		header string
	}{
		{name: "empty", header: ""},
		{name: "basic", header: "Basic abc"},
		{name: "missing token", header: "Bearer"},
		{name: "blank token", header: "Bearer   "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := wire.TestGet[any](env.Router, "/auth/userinfo", wire.TestHeader{Key: "Authorization", Value: tt.header})
			result.ExpectStatusError(t, http.StatusBadRequest)
		})
	}
}

func TestAPIUserInfo_BearerSchemeIsCaseInsensitive(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)
	env.RegisterTestUser(t, "alice", "password")

	token := env.IssueTestAccessTokenWithScopes(t, "alice", []string{"test-audience", consentAudience}, []string{"identity"})
	result := wire.TestGet[api.UserInfo](env.Router, "/auth/userinfo", wire.TestHeader{Key: "Authorization", Value: "bEaReR " + token.Encoded()})
	response := result.ExpectOK(t)
	if response.Sub != token.Subject() {
		t.Fatalf("sub = %q, want %q", response.Sub, token.Subject())
	}
}
