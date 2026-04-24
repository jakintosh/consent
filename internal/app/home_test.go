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

	req, err := tv.AuthenticatedRequest(http.MethodGet, "/", "alice")
	if err != nil {
		t.Fatalf("AuthenticatedRequest failed: %v", err)
	}
	rr := httptest.NewRecorder()

	appServer.Router().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()
	if !strings.Contains(body, "ready to approve access requests") {
		t.Fatalf("expected authenticated home content")
	}
	if !strings.Contains(body, "/logout?csrf=") {
		t.Fatalf("expected csrf-backed logout URL")
	}
}
