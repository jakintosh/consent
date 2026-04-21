package service_test

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"git.sr.ht/~jakintosh/command-go/pkg/wire"
	"git.sr.ht/~jakintosh/consent/internal/service"
	"git.sr.ht/~jakintosh/consent/internal/testutil"
)

var jsonHeader = wire.TestHeader{Key: "Content-Type", Value: "application/json"}

const consentAudience = "test.consent.local"

func TestLogin_Success(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)

	// setup env
	env.RegisterTestUser(t, "alice", "password123")

	// valid login returns redirect URL with auth_code
	redirectURL, err := env.Service.Login("alice", "password123", service.InternalServiceName)
	if err != nil {
		t.Fatalf("Login failed: %v", err)
	}
	if redirectURL == nil {
		t.Fatal("expected redirect URL")
	}
	authCode := redirectURL.Query().Get("auth_code")
	if authCode == "" {
		t.Error("redirect URL missing auth_code parameter")
	}
}

func TestLogin_RedirectURL(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)

	// setup env
	env.RegisterTestUser(t, "alice", "password123")

	// login redirects to service's configured callback URL
	redirectURL, err := env.Service.Login("alice", "password123", service.InternalServiceName)
	if err != nil {
		t.Fatalf("Login failed: %v", err)
	}
	if redirectURL.Host != "consent.test" {
		t.Errorf("redirect host = %s, want consent.test", redirectURL.Host)
	}
	if redirectURL.Path != "/auth/callback" {
		t.Errorf("redirect path = %s, want /auth/callback", redirectURL.Path)
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)

	// setup env
	env.RegisterTestUser(t, "alice", "password123")

	// wrong password returns ErrInvalidCredentials
	_, err := env.Service.Login("alice", "wrongpassword", service.InternalServiceName)
	if !errors.Is(err, service.ErrInvalidCredentials) {
		t.Errorf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestLogin_UnknownUser(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)

	// unknown user returns ErrAccountNotFound
	_, err := env.Service.Login("unknown", "password", service.InternalServiceName)
	if !errors.Is(err, service.ErrAccountNotFound) {
		t.Errorf("expected ErrAccountNotFound, got %v", err)
	}
}

func TestLogin_UnknownService(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)

	// setup env
	env.RegisterTestUser(t, "alice", "password123")

	// unknown service returns ErrServiceNotFound
	_, err := env.Service.Login("alice", "password123", "nonexistent-service")
	if !errors.Is(err, service.ErrServiceNotFound) {
		t.Errorf("expected ErrServiceNotFound, got %v", err)
	}
}

func TestLogin_StoresRefreshToken(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)

	// setup env
	env.RegisterTestUser(t, "alice", "password123")

	// login and get auth_code
	redirectURL, err := env.Service.Login("alice", "password123", service.InternalServiceName)
	if err != nil {
		t.Fatalf("Login failed: %v", err)
	}
	authCode := redirectURL.Query().Get("auth_code")

	// token is stored and owner can be retrieved
	owner, err := env.DB.GetRefreshTokenOwner(authCode)
	if err != nil {
		t.Fatalf("Token not stored: %v", err)
	}
	identity, err := env.DB.GetIdentityByHandle("alice")
	if err != nil {
		t.Fatalf("GetIdentityByHandle failed: %v", err)
	}
	if owner != identity.Subject {
		t.Errorf("token owner = %s, want %s", owner, identity.Subject)
	}
}

func TestLogin_AuthCodeIsValidJWT(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)

	// setup env
	env.RegisterTestUser(t, "alice", "password123")

	// login and get auth_code
	redirectURL, err := env.Service.Login("alice", "password123", service.InternalServiceName)
	if err != nil {
		t.Fatalf("Login failed: %v", err)
	}
	authCode := redirectURL.Query().Get("auth_code")

	// auth_code is valid JWT format (3 dot-separated parts)
	parts := strings.Split(authCode, ".")
	if len(parts) != 3 {
		t.Errorf("auth_code not valid JWT format, has %d parts", len(parts))
	}
}

func TestAPILogin_JSON_Success(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)

	// setup env
	env.RegisterTestUser(t, "alice", "password123")

	// valid login redirects with auth_code
	body := `{
		"handle": "alice",
		"secret": "password123",
		"service": "consent"
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

func TestAPILogin_JSON_RedirectTarget(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)

	// setup env
	env.RegisterTestUser(t, "alice", "password123")

	// valid login redirects to service callback URL
	body := `{
		"handle": "alice",
		"secret": "password123",
		"service": "consent"
	}`
	result := wire.TestPost[any](env.Router, "/auth/login", body, jsonHeader)
	result.ExpectStatus(t, http.StatusSeeOther)
	location := result.Headers.Get("Location")
	if location == "" {
		t.Fatal("expected Location header in redirect")
	}
	if !strings.Contains(location, "consent.test") {
		t.Errorf("redirect should be to service URL, got: %s", location)
	}
	if !strings.Contains(location, "/auth/callback") {
		t.Errorf("redirect should include auth callback path, got: %s", location)
	}
}

func TestAPILogin_UnsupportedContentType(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)

	// non-JSON content type is rejected
	result := wire.TestPost[any](env.Router, "/auth/login", "data", wire.TestHeader{Key: "Content-Type", Value: "text/plain"})
	result.ExpectStatus(t, http.StatusUnsupportedMediaType)
	result.ExpectError(t)
}

func TestAPILogin_InvalidCredentials(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)

	// setup env
	env.RegisterTestUser(t, "alice", "password123")

	// wrong password returns 401
	body := `{
		"handle": "alice",
		"secret": "wrongpassword",
		"service": "consent"
	}`
	result := wire.TestPost[any](env.Router, "/auth/login", body, jsonHeader)
	result.ExpectStatus(t, http.StatusUnauthorized)
	result.ExpectError(t)
}

func TestAPILogin_UnknownUser(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)

	// login with non-existent user returns 401
	body := `{
		"handle": "unknown",
		"secret": "password",
		"service": "consent"
	}`
	result := wire.TestPost[any](env.Router, "/auth/login", body, jsonHeader)
	result.ExpectStatus(t, http.StatusUnauthorized)
	result.ExpectError(t)
}

func TestAPILogin_UnknownService(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)

	// setup env
	env.RegisterTestUser(t, "alice", "password123")

	// login with unknown service returns 400
	body := `{
		"handle": "alice",
		"secret": "password123",
		"service": "unknown"
	}`
	result := wire.TestPost[any](env.Router, "/auth/login", body, jsonHeader)
	result.ExpectStatus(t, http.StatusBadRequest)
	result.ExpectError(t)

	_, err := env.Service.GetServiceByName(service.InternalServiceName)
	if err != nil {
		t.Fatalf("expected internal service to exist: %v", err)
	}
}

func TestAPILogin_InvalidJSON(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)

	// malformed JSON returns 400
	result := wire.TestPost[any](env.Router, "/auth/login", "not-json", jsonHeader)
	result.ExpectStatus(t, http.StatusBadRequest)
	result.ExpectError(t)
}

func TestAPILogin_MissingFields(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)

	// table-driven test for missing required fields
	tests := []struct {
		name string
		body string
	}{
		{"missing handle", `{"secret":"pass","service":"test-service"}`},
		{"missing secret", `{"handle":"user","service":"test-service"}`},
		{"missing service", `{"handle":"user","secret":"pass"}`},
		{"empty object", `{}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := wire.TestPost[any](env.Router, "/auth/login", tt.body, jsonHeader)
			// should either fail at login or return auth error
			if result.Code == http.StatusSeeOther {
				t.Error("should not redirect with missing fields")
			}
		})
	}
}

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
	result := wire.TestPost[service.RefreshResponse](env.Router, "/auth/refresh", body, jsonHeader)
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
	result := wire.TestPost[any](env.Router, "/auth/refresh", body, jsonHeader)
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
	result := wire.TestPost[any](env.Router, "/auth/refresh", body, jsonHeader)
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
	result := wire.TestPost[service.RefreshResponse](env.Router, "/auth/refresh", body, jsonHeader)
	result.ExpectOK(t)

	// second refresh with same token fails (token was rotated)
	badResult := wire.TestPost[any](env.Router, "/auth/refresh", body, jsonHeader)
	badResult.ExpectStatus(t, http.StatusBadRequest)
	badResult.ExpectError(t)
}

func TestAPIRefresh_InvalidJSON(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)

	// malformed JSON returns 400
	result := wire.TestPost[any](env.Router, "/auth/refresh", "bad-json", jsonHeader)
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
	result := wire.TestPost[service.RefreshResponse](env.Router, "/auth/refresh", body, jsonHeader)
	response1 := result.ExpectOK(t)

	// new refresh token can be used for another refresh
	body2 := `{
		"refreshToken": "` + response1.RefreshToken + `"
	}`
	result = wire.TestPost[service.RefreshResponse](env.Router, "/auth/refresh", body2, jsonHeader)
	response2 := result.ExpectOK(t)
	if response2.AccessToken == "" {
		t.Error("second refresh should return access token")
	}
}

func TestMe_IdentityOnly(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)
	env.RegisterTestUser(t, "alice", "password")

	token := env.IssueTestAccessTokenWithScopes(t, "alice", []string{"test-audience", consentAudience}, []string{"identity"})
	result := testutil.Get(env.Router, "/auth/me", &struct {
		Data service.MeResponse `json:"data"`
	}{}, testutil.Header{Key: "Authorization", Value: "Bearer " + token.Encoded()})
	testutil.ExpectStatus(t, http.StatusOK, result)
	if string(result.Body) == "" {
		t.Fatal("expected response body")
	}
	if bytes.Contains(result.Body, []byte("profile")) {
		t.Fatalf("identity-only response should not include profile: %s", string(result.Body))
	}
}

func TestMe_ProfileScope(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)
	env.RegisterTestUser(t, "alice", "password")

	token := env.IssueTestAccessTokenWithScopes(t, "alice", []string{"test-audience", consentAudience}, []string{"identity", "profile"})
	var response struct {
		Data service.MeResponse `json:"data"`
	}
	result := testutil.Get(env.Router, "/auth/me", &response, testutil.Header{Key: "Authorization", Value: "Bearer " + token.Encoded()})
	testutil.ExpectStatus(t, http.StatusOK, result)
	if response.Data.Profile == nil || response.Data.Profile.Handle != "alice" {
		t.Fatalf("profile handle = %#v, want alice", response.Data.Profile)
	}
}

func TestMe_RequiresIdentityScope(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)
	env.RegisterTestUser(t, "alice", "password")

	token := env.IssueTestAccessTokenWithScopes(t, "alice", []string{"test-audience", consentAudience}, []string{"profile"})
	result := testutil.Get(env.Router, "/auth/me", nil, testutil.Header{Key: "Authorization", Value: "Bearer " + token.Encoded()})
	testutil.ExpectStatus(t, http.StatusForbidden, result)
}

func TestMe_RequiresBearerHeader(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)
	env.RegisterTestUser(t, "alice", "password")

	token := env.IssueTestAccessTokenWithScopes(t, "alice", []string{"test-audience", consentAudience}, []string{"identity"})
	req := httptest.NewRequest(http.MethodGet, "/auth/me", nil)
	req.AddCookie(&http.Cookie{Name: "accessToken", Value: token.Encoded()})
	rr := httptest.NewRecorder()
	env.Router.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d. Body: %s", rr.Code, http.StatusBadRequest, rr.Body.String())
	}
}

func TestMe_RequiresConsentAudience(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)
	env.RegisterTestUser(t, "alice", "password")

	token := env.IssueTestAccessTokenWithScopes(t, "alice", []string{"test-audience"}, []string{"identity"})
	result := testutil.Get(env.Router, "/auth/me", nil, testutil.Header{Key: "Authorization", Value: "Bearer " + token.Encoded()})
	testutil.ExpectStatus(t, http.StatusBadRequest, result)
}

func TestRefreshAccessToken_Success(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)

	// setup env
	env.RegisterTestUser(t, "alice", "password")
	token := env.StoreTestRefreshToken(t, "alice", []string{"test-audience"})

	// refreshing valid token returns new access and refresh tokens
	accessToken, newRefreshToken, err := env.Service.RefreshAccessToken(token.Encoded())
	if err != nil {
		t.Fatalf("RefreshAccessToken failed: %v", err)
	}
	if accessToken == "" {
		t.Error("expected non-empty access token")
	}
	if newRefreshToken == "" {
		t.Error("expected non-empty refresh token")
	}
}

func TestRefreshAccessToken_InvalidToken(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)

	// malformed token returns ErrTokenInvalid
	_, _, err := env.Service.RefreshAccessToken("invalid-token")
	if !errors.Is(err, service.ErrTokenInvalid) {
		t.Errorf("expected ErrTokenInvalid, got %v", err)
	}
}

func TestRefreshAccessToken_TokenNotInStore(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)

	// setup env
	env.RegisterTestUser(t, "alice", "password")
	token := env.IssueTestRefreshToken(t, "alice", []string{"test-audience"})

	// valid token not in store returns ErrTokenNotFound
	_, _, err := env.Service.RefreshAccessToken(token.Encoded())
	if !errors.Is(err, service.ErrTokenNotFound) {
		t.Errorf("expected ErrTokenNotFound, got %v", err)
	}
}

func TestRefreshAccessToken_DeletesOldToken(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)

	// setup env
	env.RegisterTestUser(t, "alice", "password")
	token := env.StoreTestRefreshToken(t, "alice", []string{"test-audience"})

	// first refresh succeeds
	_, _, err := env.Service.RefreshAccessToken(token.Encoded())
	if err != nil {
		t.Fatalf("RefreshAccessToken failed: %v", err)
	}

	// old token is deleted and can't be used again
	_, _, err = env.Service.RefreshAccessToken(token.Encoded())
	if !errors.Is(err, service.ErrTokenNotFound) {
		t.Errorf("old token should be deleted, got %v", err)
	}
}

func TestRefreshAccessToken_StoresNewToken(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)

	// setup env
	env.RegisterTestUser(t, "alice", "password")
	token := env.StoreTestRefreshToken(t, "alice", []string{"test-audience"})

	// refresh returns new token
	_, newRefreshToken, err := env.Service.RefreshAccessToken(token.Encoded())
	if err != nil {
		t.Fatalf("RefreshAccessToken failed: %v", err)
	}

	// new token is stored in database
	owner, err := env.DB.GetRefreshTokenOwner(newRefreshToken)
	if err != nil {
		t.Fatalf("new token not stored: %v", err)
	}
	identity, err := env.DB.GetIdentityByHandle("alice")
	if err != nil {
		t.Fatalf("GetIdentityByHandle failed: %v", err)
	}
	if owner != identity.Subject {
		t.Errorf("new token owner = %s, want %s", owner, identity.Subject)
	}
}

func TestRefreshAccessToken_CanBeRefreshedAgain(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)

	// setup env
	env.RegisterTestUser(t, "alice", "password")
	token := env.StoreTestRefreshToken(t, "alice", []string{"test-audience"})

	// first refresh succeeds
	_, newRefreshToken1, err := env.Service.RefreshAccessToken(token.Encoded())
	if err != nil {
		t.Fatalf("First RefreshAccessToken failed: %v", err)
	}

	// new token can be used for another refresh
	_, newRefreshToken2, err := env.Service.RefreshAccessToken(newRefreshToken1)
	if err != nil {
		t.Fatalf("Second RefreshAccessToken failed: %v", err)
	}
	if newRefreshToken2 == "" {
		t.Error("expected non-empty second refresh token")
	}
}

func TestRevokeRefreshToken_Success(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)

	// setup env
	env.RegisterTestUser(t, "alice", "password")
	token := env.StoreTestRefreshToken(t, "alice", []string{"test-audience"})

	// revoking valid token succeeds
	err := env.Service.RevokeRefreshToken(token.Encoded())
	if err != nil {
		t.Fatalf("RevokeRefreshToken failed: %v", err)
	}
}

func TestRevokeRefreshToken_NotFound(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)

	// revoking non-existent token returns ErrTokenNotFound
	err := env.Service.RevokeRefreshToken("nonexistent-token")
	if !errors.Is(err, service.ErrTokenNotFound) {
		t.Errorf("expected ErrTokenNotFound, got %v", err)
	}
}

func TestRevokeRefreshToken_CantRefreshAfterRevoke(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)

	// setup env
	env.RegisterTestUser(t, "alice", "password")
	token := env.StoreTestRefreshToken(t, "alice", []string{"test-audience"})

	// revoke the token
	if err := env.Service.RevokeRefreshToken(token.Encoded()); err != nil {
		t.Fatalf("RevokeRefreshToken failed: %v", err)
	}

	// revoked token can't be used for refresh
	_, _, err := env.Service.RefreshAccessToken(token.Encoded())
	if !errors.Is(err, service.ErrTokenNotFound) {
		t.Errorf("expected ErrTokenNotFound after revoke, got %v", err)
	}
}

func TestRevokeRefreshToken_DoubleRevoke(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)

	// setup env
	env.RegisterTestUser(t, "alice", "password")
	token := env.StoreTestRefreshToken(t, "alice", []string{"test-audience"})

	// first revoke succeeds
	if err := env.Service.RevokeRefreshToken(token.Encoded()); err != nil {
		t.Fatalf("First RevokeRefreshToken failed: %v", err)
	}

	// second revoke returns ErrTokenNotFound
	err := env.Service.RevokeRefreshToken(token.Encoded())
	if !errors.Is(err, service.ErrTokenNotFound) {
		t.Errorf("expected ErrTokenNotFound on second revoke, got %v", err)
	}
}
