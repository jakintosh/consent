package app

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"git.sr.ht/~jakintosh/consent/internal/service"
	"git.sr.ht/~jakintosh/consent/internal/testutil"
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

func TestLoginSubmit_InvalidCredentialsRendersHTML(t *testing.T) {
	env := testutil.SetupTestEnv(t)
	env.RegisterTestUser(t, "alice", "password123")
	tv := consenttesting.NewTestVerifier("consent.test", "app.test")

	appServer, err := New(AppOptions{
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

	body := url.Values{
		"handle":    []string{"alice"},
		"secret":    []string{"wrong-password"},
		"return_to": []string{"/authorize?service=mock1"},
	}
	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(body.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()

	appServer.Router().ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusUnauthorized)
	}
	if contentType := rr.Header().Get("Content-Type"); !strings.Contains(contentType, "text/html") {
		t.Fatalf("content type = %q, want html", contentType)
	}
	bodyText := rr.Body.String()
	if !strings.Contains(bodyText, "Invalid handle or secret.") {
		t.Fatalf("expected browser-friendly login error")
	}
	if !strings.Contains(bodyText, `value="alice"`) {
		t.Fatalf("expected handle to remain in form")
	}
	if strings.Contains(bodyText, `{"error"`) {
		t.Fatalf("unexpected json error response")
	}
	if !strings.Contains(bodyText, `value="/authorize?service=mock1"`) {
		t.Fatalf("expected sanitized return_to to remain in form")
	}
}

func TestLoginSubmit_SuccessRedirectsToAuthCallback(t *testing.T) {
	env := testutil.SetupTestEnv(t)
	env.RegisterTestUser(t, "alice", "password123")
	tv := consenttesting.NewTestVerifier("consent.test", "app.test")

	appServer, err := New(AppOptions{
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

	body := url.Values{
		"handle":    []string{"alice"},
		"secret":    []string{"password123"},
		"return_to": []string{"/authorize?service=mock1&scope=identity"},
	}
	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(body.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()

	appServer.Router().ServeHTTP(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusSeeOther)
	}
	location := rr.Header().Get("Location")
	if !strings.Contains(location, "/auth/callback?") {
		t.Fatalf("redirect = %q, want auth callback", location)
	}
	if !strings.Contains(location, "return_to=%2Fauthorize%3Fservice%3Dmock1%26scope%3Didentity") {
		t.Fatalf("redirect = %q, want preserved return_to", location)
	}
}

func TestLoginSubmit_InvalidReturnToFallsBackHome(t *testing.T) {
	env := testutil.SetupTestEnv(t)
	env.RegisterTestUser(t, "alice", "password123")
	tv := consenttesting.NewTestVerifier("consent.test", "app.test")

	appServer, err := New(AppOptions{
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

	body := url.Values{
		"handle":    []string{"alice"},
		"secret":    []string{"password123"},
		"return_to": []string{"https://evil.test/pwn"},
	}
	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(body.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()

	appServer.Router().ServeHTTP(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusSeeOther)
	}
	location := rr.Header().Get("Location")
	if !strings.Contains(location, "return_to=%2F") {
		t.Fatalf("redirect = %q, want sanitized home return_to", location)
	}
}
