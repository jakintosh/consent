package service_test

import (
	"bytes"
	"net/http"
	"testing"

	"git.sr.ht/~jakintosh/consent/internal/service"
	"git.sr.ht/~jakintosh/consent/internal/testutil"
)

func TestMe_IdentityOnly(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)
	env.RegisterTestUser(t, "alice", "password")

	token := env.IssueTestAccessTokenWithScopes(t, "alice", []string{"test-audience"}, []string{"identity"})
	result := testutil.Get(env.Router, "/me", &struct {
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

	token := env.IssueTestAccessTokenWithScopes(t, "alice", []string{"test-audience"}, []string{"identity", "profile"})
	var response struct {
		Data service.MeResponse `json:"data"`
	}
	result := testutil.Get(env.Router, "/me", &response, testutil.Header{Key: "Authorization", Value: "Bearer " + token.Encoded()})
	testutil.ExpectStatus(t, http.StatusOK, result)
	if response.Data.Profile == nil || response.Data.Profile.Handle != "alice" {
		t.Fatalf("profile handle = %#v, want alice", response.Data.Profile)
	}
}

func TestMe_RequiresIdentityScope(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)
	env.RegisterTestUser(t, "alice", "password")

	token := env.IssueTestAccessTokenWithScopes(t, "alice", []string{"test-audience"}, []string{"profile"})
	result := testutil.Get(env.Router, "/me", nil, testutil.Header{Key: "Authorization", Value: "Bearer " + token.Encoded()})
	testutil.ExpectStatus(t, http.StatusForbidden, result)
}
