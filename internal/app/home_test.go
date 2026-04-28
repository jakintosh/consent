package app

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"git.sr.ht/~jakintosh/consent/internal/testutil"
	consenttesting "git.sr.ht/~jakintosh/consent/pkg/testing"
)

func TestHome_Unauthenticated(t *testing.T) {
	tv := consenttesting.NewTestVerifier("consent.test", "app.test")
	env := testutil.SetupTestEnv(t)

	appServer, err := New(Options{
		Service: env.Service,
		Auth: AuthConfig{
			Verifier:  tv,
			LoginURL:  "/login",
			LogoutURL: "/logout",
			Routes:    map[string]http.HandlerFunc{},
		},
	})
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()

	appServer.Router().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()
	if !strings.Contains(body, "Log In") {
		t.Fatalf("expected login prompt in home page")
	}
	if !strings.Contains(body, "/login") {
		t.Fatalf("expected login URL in home page")
	}
}

func TestHome_AuthenticatedIncludesCSRFLogoutURL(t *testing.T) {
	tv := consenttesting.NewTestVerifier("consent.test", "app.test")
	env := testutil.SetupTestEnv(t)
	env.RegisterTestUser(t, "alice", "password")
	user, err := env.DB.GetUserByHandle("alice")
	if err != nil {
		t.Fatalf("GetUserByHandle failed: %v", err)
	}

	appServer, err := New(Options{
		Service: env.Service,
		Auth: AuthConfig{
			Verifier:  tv,
			LoginURL:  "/login",
			LogoutURL: "/logout",
			Routes:    map[string]http.HandlerFunc{},
		},
	})
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	req, err := tv.AuthenticatedRequest(http.MethodGet, "/", user.Subject)
	if err != nil {
		t.Fatalf("AuthenticatedRequest failed: %v", err)
	}
	rr := httptest.NewRecorder()

	appServer.Router().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()
	if !strings.Contains(body, "Hello, alice") {
		t.Fatalf("expected handle in home page, got: %s", body)
	}
	if !strings.Contains(body, "ready to approve access requests") {
		t.Fatalf("expected authenticated home content")
	}
	if !strings.Contains(body, "/logout?csrf=") {
		t.Fatalf("expected csrf-backed logout URL")
	}
}

func TestHome_AuthenticatedShowsIntegrations(t *testing.T) {
	tv := consenttesting.NewTestVerifier("consent.test", "app.test")
	env := testutil.SetupTestEnv(t)
	env.RegisterTestUser(t, "alice", "password")
	env.CreateTestIntegration(t, "test-app", "Test App", "test-audience", "https://test.test/callback", "https://test-app.local", "https://test-app.local/logo.png")
	user, err := env.DB.GetUserByHandle("alice")
	if err != nil {
		t.Fatalf("GetUserByHandle failed: %v", err)
	}

	if err := env.DB.InsertGrants(user.Subject, "test-app", []string{"identity", "profile"}); err != nil {
		t.Fatalf("InsertGrants failed: %v", err)
	}

	appServer, err := New(Options{
		Service: env.Service,
		Auth: AuthConfig{
			Verifier:  tv,
			LoginURL:  "/login",
			LogoutURL: "/logout",
			Routes:    map[string]http.HandlerFunc{},
		},
	})
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	req, err := tv.AuthenticatedRequest(http.MethodGet, "/", user.Subject)
	if err != nil {
		t.Fatalf("AuthenticatedRequest failed: %v", err)
	}
	rr := httptest.NewRecorder()

	appServer.Router().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()
	if !strings.Contains(body, "Test App") {
		t.Fatalf("expected integration name in home page, got: %s", body)
	}
	if !strings.Contains(body, "<details") {
		t.Fatalf("expected details element in home page, got: %s", body)
	}
	if !strings.Contains(body, "Identity") {
		t.Fatalf("expected scope label in home page, got: %s", body)
	}
	if !strings.Contains(body, "Profile") {
		t.Fatalf("expected profile scope label in home page, got: %s", body)
	}
	if !strings.Contains(body, "Visit") {
		t.Fatalf("expected visit link in home page, got: %s", body)
	}
	if !strings.Contains(body, "https://test-app.local") {
		t.Fatalf("expected homepage URL in home page, got: %s", body)
	}
	if !strings.Contains(body, "https://test-app.local/logo.png") {
		t.Fatalf("expected logo URL in home page, got: %s", body)
	}
}

func TestHome_AuthenticatedShowsPartialGrants(t *testing.T) {
	tv := consenttesting.NewTestVerifier("consent.test", "app.test")
	env := testutil.SetupTestEnv(t)
	env.RegisterTestUser(t, "alice", "password")
	env.CreateTestIntegration(t, "partial-app", "Partial App", "partial-audience", "https://partial.test/callback", "https://partial-app.local", "https://partial-app.local/logo.png")
	user, err := env.DB.GetUserByHandle("alice")
	if err != nil {
		t.Fatalf("GetUserByHandle failed: %v", err)
	}

	if err := env.DB.InsertGrants(user.Subject, "partial-app", []string{"identity"}); err != nil {
		t.Fatalf("InsertGrants failed: %v", err)
	}

	appServer, err := New(Options{
		Service: env.Service,
		Auth: AuthConfig{
			Verifier:  tv,
			LoginURL:  "/login",
			LogoutURL: "/logout",
			Routes:    map[string]http.HandlerFunc{},
		},
	})
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	req, err := tv.AuthenticatedRequest(http.MethodGet, "/", user.Subject)
	if err != nil {
		t.Fatalf("AuthenticatedRequest failed: %v", err)
	}
	rr := httptest.NewRecorder()

	appServer.Router().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()
	if !strings.Contains(body, "Partial App") {
		t.Fatalf("expected integration name in home page, got: %s", body)
	}
	if !strings.Contains(body, "Identity") {
		t.Fatalf("expected granted scope label in home page, got: %s", body)
	}
	if strings.Contains(body, "profile") || strings.Contains(body, "Profile") {
		t.Fatalf("expected only identity scope, got: %s", body)
	}
	if !strings.Contains(body, "Visit") {
		t.Fatalf("expected visit link in home page, got: %s", body)
	}
}
