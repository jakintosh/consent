package app

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"git.sr.ht/~jakintosh/consent/internal/testutil"
	consenttesting "git.sr.ht/~jakintosh/consent/pkg/testing"
)

func TestAuthorize_UnauthenticatedRedirectsToLogin(t *testing.T) {
	env := testutil.SetupTestEnv(t)
	env.CreateTestService(t, "test-service", "Test Service", "test-audience", "https://service.test/callback")
	tv := consenttesting.NewTestVerifier("consent.test", "consent.test")

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

	req := httptest.NewRequest(http.MethodGet, "/authorize?service=test-service&scope=identity", nil)
	rr := httptest.NewRecorder()
	appServer.Router().ServeHTTP(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusSeeOther)
	}
	if !strings.Contains(rr.Header().Get("Location"), "/login?return_to=") {
		t.Fatalf("redirect = %q, want login return_to", rr.Header().Get("Location"))
	}
}

func TestAuthorize_AuthenticatedRendersApprovalPage(t *testing.T) {
	env := testutil.SetupTestEnv(t)
	env.RegisterTestUser(t, "alice", "password")
	env.CreateTestService(t, "test-service", "Test Service", "test-audience", "https://service.test/callback")
	identity, err := env.DB.GetIdentityByHandle("alice")
	if err != nil {
		t.Fatalf("GetIdentityByHandle failed: %v", err)
	}
	tv := consenttesting.NewTestVerifier("consent.test", "consent.test")

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

	req, err := tv.AuthenticatedRequest(http.MethodGet, "/authorize?service=test-service&scope=identity&scope=profile", identity.Subject)
	if err != nil {
		t.Fatalf("AuthenticatedRequest failed: %v", err)
	}
	rr := httptest.NewRecorder()
	appServer.Router().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	body := rr.Body.String()
	if !strings.Contains(body, "Authorize Test Service") {
		t.Fatalf("expected approval page content")
	}
	if !strings.Contains(body, "This app is asking for permission to:") {
		t.Fatalf("expected missing-scopes prompt")
	}
	if !strings.Contains(body, "value=\"profile\"") {
		t.Fatalf("expected profile scope in approval form")
	}
}

func TestAuthorize_AuthenticatedSeparatesGrantedAndMissingScopes(t *testing.T) {
	env := testutil.SetupTestEnv(t)
	env.RegisterTestUser(t, "alice", "password")
	env.CreateTestService(t, "test-service", "Test Service", "test-audience", "https://service.test/callback")
	identity, err := env.DB.GetIdentityByHandle("alice")
	if err != nil {
		t.Fatalf("GetIdentityByHandle failed: %v", err)
	}
	if err := env.DB.InsertGrants(identity.Subject, "test-service", []string{"identity"}); err != nil {
		t.Fatalf("InsertGrants failed: %v", err)
	}
	tv := consenttesting.NewTestVerifier("consent.test", "consent.test")

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

	req, err := tv.AuthenticatedRequest(http.MethodGet, "/authorize?service=test-service&scope=identity&scope=profile", identity.Subject)
	if err != nil {
		t.Fatalf("AuthenticatedRequest failed: %v", err)
	}
	rr := httptest.NewRecorder()
	appServer.Router().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	body := rr.Body.String()
	if !strings.Contains(body, "Already approved for this app:") {
		t.Fatalf("expected granted scopes section")
	}
	if !strings.Contains(body, "Approving now will additionally grant:") {
		t.Fatalf("expected missing scopes section")
	}
	if !strings.Contains(body, "Identity") || !strings.Contains(body, "Profile") {
		t.Fatalf("expected granted and missing scope labels")
	}
}

func TestAuthorizeSubmit_InvalidCSRFRendersStatusPage(t *testing.T) {
	env := testutil.SetupTestEnv(t)
	env.RegisterTestUser(t, "alice", "password")
	env.CreateTestService(t, "test-service", "Test Service", "test-audience", "https://service.test/callback")
	identity, err := env.DB.GetIdentityByHandle("alice")
	if err != nil {
		t.Fatalf("GetIdentityByHandle failed: %v", err)
	}
	tv := consenttesting.NewTestVerifier("consent.test", "consent.test")
	authReq, err := tv.AuthenticatedRequest(http.MethodGet, "/authorize?service=test-service&scope=identity", identity.Subject)
	if err != nil {
		t.Fatalf("AuthenticatedRequest failed: %v", err)
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

	body := strings.NewReader("service=test-service&scope=identity&action=approve&csrf=wrong")
	req := httptest.NewRequest(http.MethodPost, "/authorize", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	for _, cookie := range authReq.Cookies() {
		req.AddCookie(cookie)
	}
	rr := httptest.NewRecorder()

	appServer.Router().ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusForbidden)
	}
	if !strings.Contains(rr.Body.String(), "This approval form is no longer valid") {
		t.Fatalf("expected csrf status page content")
	}
}

func TestAuthorize_AuthenticatedAutoRedirectsWhenGrantExists(t *testing.T) {
	env := testutil.SetupTestEnv(t)
	env.RegisterTestUser(t, "alice", "password")
	env.CreateTestService(t, "test-service", "Test Service", "test-audience", "https://service.test/callback")
	identity, err := env.DB.GetIdentityByHandle("alice")
	if err != nil {
		t.Fatalf("GetIdentityByHandle failed: %v", err)
	}
	if err := env.DB.InsertGrants(identity.Subject, "test-service", []string{"identity"}); err != nil {
		t.Fatalf("InsertGrants failed: %v", err)
	}
	tv := consenttesting.NewTestVerifier("consent.test", "consent.test")

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

	req, err := tv.AuthenticatedRequest(http.MethodGet, "/authorize?service=test-service&scope=identity&state=test", identity.Subject)
	if err != nil {
		t.Fatalf("AuthenticatedRequest failed: %v", err)
	}
	rr := httptest.NewRecorder()
	appServer.Router().ServeHTTP(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusSeeOther)
	}
	location := rr.Header().Get("Location")
	if !strings.Contains(location, "auth_code=") || !strings.Contains(location, "state=test") {
		t.Fatalf("redirect = %q, want auth_code and state", location)
	}
}
