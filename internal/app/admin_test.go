package app

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"git.sr.ht/~jakintosh/consent/internal/service"
	"git.sr.ht/~jakintosh/consent/internal/testutil"
	"git.sr.ht/~jakintosh/consent/pkg/client"
	consenttesting "git.sr.ht/~jakintosh/consent/pkg/testing"
)

func TestAdmin_RequiresAdminRole(t *testing.T) {
	tv := consenttesting.NewTestVerifier("consent.test", "app.test")
	env := testutil.SetupTestEnv(t)
	user, err := env.Service.CreateUser("alice", "password", nil)
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}
	appServer := newTestApp(t, env, tv)

	req, err := tv.AuthenticatedRequest(http.MethodGet, "/admin", user.Subject)
	if err != nil {
		t.Fatalf("AuthenticatedRequest failed: %v", err)
	}
	rr := httptest.NewRecorder()

	appServer.Router().ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusForbidden)
	}
}

func TestAdmin_PostRequiresAdminRoleWithValidCSRF(t *testing.T) {
	tv := consenttesting.NewTestVerifier("consent.test", "app.test")
	env := testutil.SetupTestEnv(t)
	user, err := env.Service.CreateUser("alice", "password", nil)
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}
	appServer := newTestApp(t, env, tv)
	csrf, cookies := authenticatedCSRF(t, tv, user.Subject)
	form := url.Values{
		"csrf":    {csrf},
		"name":    {"operator"},
		"display": {"Operator"},
	}
	req := httptest.NewRequest(http.MethodPost, "/admin/roles", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}
	rr := httptest.NewRecorder()

	appServer.Router().ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d; body: %s", rr.Code, http.StatusForbidden, rr.Body.String())
	}
	if _, err := env.Service.GetRole("operator"); !errors.Is(err, service.ErrRoleNotFound) {
		t.Fatalf("expected operator role not to be created, got %v", err)
	}
}

func TestAdmin_DashboardForAdmin(t *testing.T) {
	tv := consenttesting.NewTestVerifier("consent.test", "app.test")
	env := testutil.SetupTestEnv(t)
	user, err := env.Service.CreateUser("alice", "password", []string{service.ProtectedAdminRoleName})
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}
	appServer := newTestApp(t, env, tv)

	body, _ := getAuthenticatedAdminPage(t, appServer, tv, user.Subject, "/admin")

	for _, want := range []string{"Dashboard", "Users", "Roles", "Integrations"} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected %q in admin dashboard, got: %s", want, body)
		}
	}
}

func TestAdmin_UpdateUserRoles(t *testing.T) {
	tv := consenttesting.NewTestVerifier("consent.test", "app.test")
	env := testutil.SetupTestEnv(t)
	env.CreateTestRole(t, "editor", "Editor")
	admin, err := env.Service.CreateUser("alice", "password", []string{service.ProtectedAdminRoleName})
	if err != nil {
		t.Fatalf("CreateUser admin failed: %v", err)
	}
	user, err := env.Service.CreateUser("bob", "password", nil)
	if err != nil {
		t.Fatalf("CreateUser bob failed: %v", err)
	}
	appServer := newTestApp(t, env, tv)

	body, cookies := getAuthenticatedAdminPage(t, appServer, tv, admin.Subject, "/admin/users/"+user.Subject+"/edit")
	csrf := csrfFromBody(t, body)
	form := url.Values{
		"csrf":   {csrf},
		"handle": {"bob"},
		"roles":  {"editor"},
	}
	req := httptest.NewRequest(http.MethodPost, "/admin/users/"+user.Subject, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}
	rr := httptest.NewRecorder()

	appServer.Router().ServeHTTP(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d; body: %s", rr.Code, http.StatusSeeOther, rr.Body.String())
	}
	updated, err := env.Service.GetUser(user.Subject)
	if err != nil {
		t.Fatalf("GetUser failed: %v", err)
	}
	if len(updated.Roles) != 1 || updated.Roles[0] != "editor" {
		t.Fatalf("roles = %#v, want [editor]", updated.Roles)
	}
}

func TestAdmin_RoleLinksEscapePathSegments(t *testing.T) {
	tv := consenttesting.NewTestVerifier("consent.test", "app.test")
	env := testutil.SetupTestEnv(t)
	admin, err := env.Service.CreateUser("alice", "password", []string{service.ProtectedAdminRoleName})
	if err != nil {
		t.Fatalf("CreateUser admin failed: %v", err)
	}
	roleName := "ops/team"
	env.CreateTestRole(t, roleName, "Ops Team")
	appServer := newTestApp(t, env, tv)

	body, _ := getAuthenticatedAdminPage(t, appServer, tv, admin.Subject, "/admin/roles")
	want := "/admin/roles/" + url.PathEscape(roleName) + "/edit"
	if !strings.Contains(body, want) {
		t.Fatalf("expected escaped role link %q, got: %s", want, body)
	}

	body, _ = getAuthenticatedAdminPage(t, appServer, tv, admin.Subject, want)
	for _, want := range []string{"Ops Team", "/admin/roles/" + url.PathEscape(roleName), "/admin/roles/" + url.PathEscape(roleName) + "/delete"} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected %q in role form, got: %s", want, body)
		}
	}
}

func TestAdmin_IntegrationLinksEscapePathSegments(t *testing.T) {
	tv := consenttesting.NewTestVerifier("consent.test", "app.test")
	env := testutil.SetupTestEnv(t)
	admin, err := env.Service.CreateUser("alice", "password", []string{service.ProtectedAdminRoleName})
	if err != nil {
		t.Fatalf("CreateUser admin failed: %v", err)
	}
	integrationName := "app/foo"
	if err := env.Service.CreateIntegration(integrationName, "App Foo", "app.example.com", "https://app.example.com/auth/callback", "https://app.example.com", "https://app.example.com/logo.png", nil); err != nil {
		t.Fatalf("CreateIntegration failed: %v", err)
	}
	appServer := newTestApp(t, env, tv)

	body, _ := getAuthenticatedAdminPage(t, appServer, tv, admin.Subject, "/admin/integrations")
	want := "/admin/integrations/" + url.PathEscape(integrationName) + "/edit"
	if !strings.Contains(body, want) {
		t.Fatalf("expected escaped integration link %q, got: %s", want, body)
	}

	body, _ = getAuthenticatedAdminPage(t, appServer, tv, admin.Subject, want)
	for _, want := range []string{"App Foo", "/admin/integrations/" + url.PathEscape(integrationName), "/admin/integrations/" + url.PathEscape(integrationName) + "/delete"} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected %q in integration form, got: %s", want, body)
		}
	}
}

func TestAdmin_ImportIntegrationManifest(t *testing.T) {
	tv := consenttesting.NewTestVerifier("consent.test", "app.test")
	env := testutil.SetupTestEnv(t)
	admin, err := env.Service.CreateUser("alice", "password", []string{service.ProtectedAdminRoleName})
	if err != nil {
		t.Fatalf("CreateUser admin failed: %v", err)
	}

	var appURL string
	app := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		client.HandleIntegrationManifest(client.IntegrationManifest{
			Name:     "manifest-app",
			Display:  "Manifest App",
			Audience: strings.TrimPrefix(appURL, "http://"),
			Redirect: appURL + "/auth/callback",
			Homepage: appURL,
			Logo:     appURL + "/logo.png",
		})(w, r)
	}))
	defer app.Close()
	appURL = app.URL

	appServer := newTestApp(t, env, tv)
	body, cookies := getAuthenticatedAdminPage(t, appServer, tv, admin.Subject, "/admin/integrations/new")
	csrf := csrfFromBody(t, body)
	form := url.Values{
		"csrf":     {csrf},
		"base_url": {app.URL},
	}
	req := httptest.NewRequest(http.MethodPost, "/admin/integrations/import", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}
	rr := httptest.NewRecorder()

	appServer.Router().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rr.Code, http.StatusOK, rr.Body.String())
	}
	body = rr.Body.String()
	for _, want := range []string{"manifest-app", "Manifest App", app.URL + "/auth/callback"} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected %q in imported form, got: %s", want, body)
		}
	}
}

func getAuthenticatedAdminPage(
	t *testing.T,
	appServer *App,
	tv *consenttesting.TestVerifier,
	subject string,
	path string,
) (string, []*http.Cookie) {
	t.Helper()
	req, err := tv.AuthenticatedRequest(http.MethodGet, path, subject)
	if err != nil {
		t.Fatalf("AuthenticatedRequest failed: %v", err)
	}
	rr := httptest.NewRecorder()

	appServer.Router().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rr.Code, http.StatusOK, rr.Body.String())
	}
	return rr.Body.String(), req.Cookies()
}

func csrfFromBody(
	t *testing.T,
	body string,
) string {
	t.Helper()
	prefix := `name="csrf" value="`
	start := strings.Index(body, prefix)
	if start == -1 {
		t.Fatalf("csrf field not found in body: %s", body)
	}
	start += len(prefix)
	end := strings.Index(body[start:], `"`)
	if end == -1 {
		t.Fatalf("csrf field unterminated in body: %s", body)
	}
	return body[start : start+end]
}

func authenticatedCSRF(
	t *testing.T,
	tv *consenttesting.TestVerifier,
	subject string,
) (string, []*http.Cookie) {
	t.Helper()
	req, err := tv.AuthenticatedRequest(http.MethodGet, "/", subject)
	if err != nil {
		t.Fatalf("AuthenticatedRequest failed: %v", err)
	}
	rr := httptest.NewRecorder()
	_, csrf, err := tv.VerifyAuthorizationGetCSRF(rr, req)
	if err != nil {
		t.Fatalf("VerifyAuthorizationGetCSRF failed: %v", err)
	}
	return csrf, req.Cookies()
}
