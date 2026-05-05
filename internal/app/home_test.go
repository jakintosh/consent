package app

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"git.sr.ht/~jakintosh/consent/internal/service"
	"git.sr.ht/~jakintosh/consent/internal/testutil"
	consenttesting "git.sr.ht/~jakintosh/consent/pkg/testing"
)

func TestHome_Unauthenticated(t *testing.T) {
	tv := consenttesting.NewTestVerifier("consent.test", "app.test")
	env := testutil.SetupTestEnv(t)
	appServer := newTestApp(t, env, tv)

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

func TestHome_AuthenticatedIncludesUserRolesAndCSRFLogoutURL(t *testing.T) {
	tv := consenttesting.NewTestVerifier("consent.test", "app.test")
	env := testutil.SetupTestEnv(t)
	env.CreateTestRole(t, "editor", "Editor")
	user, err := env.Service.CreateUser("alice", "password", []string{"editor"})
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}
	appServer := newTestApp(t, env, tv)

	body := getAuthenticatedHome(t, appServer, tv, user.Subject)

	if !strings.Contains(body, "Hello, alice") {
		t.Fatalf("expected handle in home page, got: %s", body)
	}
	if !strings.Contains(body, "You are signed in as <strong>alice</strong>") {
		t.Fatalf("expected signed-in user summary")
	}
	if !strings.Contains(body, "Editor") {
		t.Fatalf("expected friendly role display in home page, got: %s", body)
	}
	if !strings.Contains(body, "/logout?csrf=") {
		t.Fatalf("expected csrf-backed logout URL")
	}
}

func TestHome_AuthenticatedShowsAccessibleIntegrationsAndHidesInaccessible(t *testing.T) {
	tv := consenttesting.NewTestVerifier("consent.test", "app.test")
	env := testutil.SetupTestEnv(t)
	env.CreateTestRole(t, "editor", "Editor")
	env.CreateTestRole(t, "superadmin", "Super Admin")
	env.CreateTestIntegration(t, "open-app", "Open App", "open-audience", "https://open.test/callback", "https://open.test", "https://open.test/logo.png", nil)
	env.CreateTestIntegration(t, "editor-app", "Editor App", "editor-audience", "https://editor.test/callback", "https://editor.test", "https://editor.test/logo.png", []string{"editor"})
	env.CreateTestIntegration(t, "admin-app", "Admin App", "admin-audience", "https://admin.test/callback", "https://admin.test", "https://admin.test/logo.png", []string{"superadmin"})
	user, err := env.Service.CreateUser("alice", "password", []string{"editor"})
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}
	appServer := newTestApp(t, env, tv)

	body := getAuthenticatedHome(t, appServer, tv, user.Subject)

	if !strings.Contains(body, "Open App") {
		t.Fatalf("expected open integration in home page, got: %s", body)
	}
	if !strings.Contains(body, "Editor App") {
		t.Fatalf("expected role-accessible integration in home page, got: %s", body)
	}
	if strings.Contains(body, "Admin App") {
		t.Fatalf("expected inaccessible integration to be hidden, got: %s", body)
	}
	if !strings.Contains(body, "No scopes granted") {
		t.Fatalf("expected no-grants status for accessible integrations")
	}
	if !strings.Contains(body, "No scopes granted yet.") {
		t.Fatalf("expected empty grants message")
	}
	if !strings.Contains(body, "Not granted") {
		t.Fatalf("expected not-granted scopes section")
	}
}

func TestHome_AuthenticatedShowsGrantedAndUngrantedScopes(t *testing.T) {
	tv := consenttesting.NewTestVerifier("consent.test", "app.test")
	env := testutil.SetupTestEnv(t)
	env.RegisterTestUser(t, "alice", "password")
	env.CreateTestIntegration(t, "partial-app", "Partial App", "partial-audience", "https://partial.test/callback", "https://partial-app.local", "https://partial-app.local/logo.png", nil)
	user, err := env.DB.GetUserByHandle("alice")
	if err != nil {
		t.Fatalf("GetUserByHandle failed: %v", err)
	}
	if err := env.DB.InsertGrants(user.Subject, "partial-app", []string{service.ScopeIdentity}); err != nil {
		t.Fatalf("InsertGrants failed: %v", err)
	}
	appServer := newTestApp(t, env, tv)

	body := getAuthenticatedHome(t, appServer, tv, user.Subject)

	if !strings.Contains(body, "Partial App") {
		t.Fatalf("expected integration name in home page, got: %s", body)
	}
	if !strings.Contains(body, "Some scopes granted") {
		t.Fatalf("expected partial grant status, got: %s", body)
	}
	if !strings.Contains(body, "Granted scopes") {
		t.Fatalf("expected granted scopes section")
	}
	if !strings.Contains(body, "Identity") {
		t.Fatalf("expected granted identity scope label, got: %s", body)
	}
	if !strings.Contains(body, "Not granted") {
		t.Fatalf("expected ungranted scopes section")
	}
	if !strings.Contains(body, "Profile") {
		t.Fatalf("expected ungranted profile scope label, got: %s", body)
	}
	if !strings.Contains(body, "Visit") || !strings.Contains(body, "https://partial-app.local") {
		t.Fatalf("expected visit link in home page, got: %s", body)
	}
}

func TestHome_AuthenticatedShowsAllScopesGranted(t *testing.T) {
	tv := consenttesting.NewTestVerifier("consent.test", "app.test")
	env := testutil.SetupTestEnv(t)
	env.RegisterTestUser(t, "alice", "password")
	env.CreateTestIntegration(t, "test-app", "Test App", "test-audience", "https://test.test/callback", "https://test-app.local", "https://test-app.local/logo.png", nil)
	user, err := env.DB.GetUserByHandle("alice")
	if err != nil {
		t.Fatalf("GetUserByHandle failed: %v", err)
	}
	if err := env.DB.InsertGrants(user.Subject, "test-app", []string{service.ScopeIdentity, service.ScopeProfile}); err != nil {
		t.Fatalf("InsertGrants failed: %v", err)
	}
	appServer := newTestApp(t, env, tv)

	body := getAuthenticatedHome(t, appServer, tv, user.Subject)

	if !strings.Contains(body, "All scopes granted") {
		t.Fatalf("expected all-granted status, got: %s", body)
	}
	if !strings.Contains(body, "All known scopes are granted.") {
		t.Fatalf("expected all-granted details message, got: %s", body)
	}
	if !strings.Contains(body, "https://test-app.local/logo.png") {
		t.Fatalf("expected configured logo URL in home page, got: %s", body)
	}
}

func TestHome_AuthenticatedFallsBackToDefaultLogo(t *testing.T) {
	tv := consenttesting.NewTestVerifier("consent.test", "app.test")
	env := testutil.SetupTestEnv(t)
	env.RegisterTestUser(t, "alice", "password")
	if err := env.DB.InsertIntegration("logo-less", "Logo Less", "logo-less-audience", "https://logo-less.test/callback", "https://logo-less.test", "", nil); err != nil {
		t.Fatalf("InsertIntegration failed: %v", err)
	}
	user, err := env.DB.GetUserByHandle("alice")
	if err != nil {
		t.Fatalf("GetUserByHandle failed: %v", err)
	}
	appServer := newTestApp(t, env, tv)

	body := getAuthenticatedHome(t, appServer, tv, user.Subject)

	if !strings.Contains(body, "Logo Less") {
		t.Fatalf("expected integration in home page, got: %s", body)
	}
	if !strings.Contains(body, service.DefaultIntegrationLogoPath) {
		t.Fatalf("expected default logo path in home page, got: %s", body)
	}
}

func TestHome_DefaultLogoAssetRoute(t *testing.T) {
	tv := consenttesting.NewTestVerifier("consent.test", "app.test")
	env := testutil.SetupTestEnv(t)
	appServer := newTestApp(t, env, tv)

	req := httptest.NewRequest(http.MethodGet, service.DefaultIntegrationLogoPath, nil)
	rr := httptest.NewRecorder()
	appServer.Router().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	if contentType := rr.Header().Get("Content-Type"); contentType != "image/png" {
		t.Fatalf("Content-Type = %q, want image/png", contentType)
	}
	if rr.Body.Len() == 0 {
		t.Fatalf("expected logo response body")
	}
}

func newTestApp(
	t *testing.T,
	env *testutil.TestEnv,
	tv *consenttesting.TestVerifier,
) *App {
	t.Helper()
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
	return appServer
}

func getAuthenticatedHome(
	t *testing.T,
	appServer *App,
	tv *consenttesting.TestVerifier,
	subject string,
) string {
	t.Helper()
	req, err := tv.AuthenticatedRequest(http.MethodGet, "/", subject)
	if err != nil {
		t.Fatalf("AuthenticatedRequest failed: %v", err)
	}
	rr := httptest.NewRecorder()

	appServer.Router().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	return rr.Body.String()
}
