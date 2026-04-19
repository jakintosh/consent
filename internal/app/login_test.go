package app

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"git.sr.ht/~jakintosh/consent/internal/service"
	consenttesting "git.sr.ht/~jakintosh/consent/pkg/testing"
)

func TestLogin_UnauthenticatedRendersForm(t *testing.T) {
	tv := consenttesting.NewTestVerifier("consent.test", "app.test")

	appServer, err := New(AppOptions{
		Service: &service.Service{},
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

	req := httptest.NewRequest(http.MethodGet, "/login?return_to=%2Fauthorize%3Fservice%3Dmock1", nil)
	rr := httptest.NewRecorder()

	appServer.Router().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	if !strings.Contains(rr.Body.String(), "Log In") {
		t.Fatalf("expected login form content")
	}
	if !strings.Contains(rr.Body.String(), `value="/authorize?service=mock1"`) {
		t.Fatalf("expected return_to to be preserved in form")
	}
}

func TestLogin_AuthenticatedRedirectsToReturnTo(t *testing.T) {
	tv := consenttesting.NewTestVerifier("consent.test", "app.test")

	appServer, err := New(AppOptions{
		Service: &service.Service{},
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

	req, err := tv.AuthenticatedRequest(http.MethodGet, "/login?return_to=%2Fauthorize%3Fservice%3Dmock1", "alice")
	if err != nil {
		t.Fatalf("AuthenticatedRequest failed: %v", err)
	}
	rr := httptest.NewRecorder()

	appServer.Router().ServeHTTP(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusSeeOther)
	}
	if rr.Header().Get("Location") != "/authorize?service=mock1" {
		t.Fatalf("location = %q, want %q", rr.Header().Get("Location"), "/authorize?service=mock1")
	}
}

func TestLogin_AuthenticatedRejectsAbsoluteReturnTo(t *testing.T) {
	tv := consenttesting.NewTestVerifier("consent.test", "app.test")

	appServer, err := New(AppOptions{
		Service: &service.Service{},
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

	req, err := tv.AuthenticatedRequest(http.MethodGet, "/login?return_to=http://evil.test/pwn", "alice")
	if err != nil {
		t.Fatalf("AuthenticatedRequest failed: %v", err)
	}
	rr := httptest.NewRecorder()

	appServer.Router().ServeHTTP(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusSeeOther)
	}
	if rr.Header().Get("Location") != "/" {
		t.Fatalf("location = %q, want %q", rr.Header().Get("Location"), "/")
	}
}
