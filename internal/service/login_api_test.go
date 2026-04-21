package service_test

import (
	"net/http"
	"strings"
	"testing"

	"git.sr.ht/~jakintosh/command-go/pkg/wire"
	"git.sr.ht/~jakintosh/consent/internal/service"
	"git.sr.ht/~jakintosh/consent/internal/testutil"
)

var jsonHeader = wire.TestHeader{Key: "Content-Type", Value: "application/json"}

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
