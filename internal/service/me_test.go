package service_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"git.sr.ht/~jakintosh/consent/internal/service"
	"git.sr.ht/~jakintosh/consent/internal/testutil"
)

const consentAudience = "test.consent.local"

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
