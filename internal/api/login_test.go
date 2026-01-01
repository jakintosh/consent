package api_test

import (
	"net/http"
	"strings"
	"testing"

	"git.sr.ht/~jakintosh/consent/internal/testutil"
)

func TestLogin_JSON_Success(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)

	// setup env
	env.RegisterTestUser(t, "alice", "password123")

	// valid login redirects with auth_code
	body := `{
		"handle": "alice",
		"secret": "password123",
		"service": "test-service"
	}`
	result := testutil.PostJSON(env.Router, "/login", body, nil)
	location := testutil.ExpectRedirect(t, result)
	if !strings.Contains(location, "auth_code=") {
		t.Errorf("redirect URL missing auth_code: %s", location)
	}
}

func TestLogin_JSON_RedirectTarget(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)

	// setup env
	env.RegisterTestUser(t, "alice", "password123")

	// valid login redirects to service callback URL
	body := `{
		"handle": "alice",
		"secret": "password123",
		"service": "test-service"
	}`
	result := testutil.PostJSON(env.Router, "/login", body, nil)
	location := testutil.ExpectRedirect(t, result)
	if !strings.Contains(location, "localhost:8080") {
		t.Errorf("redirect should be to service URL, got: %s", location)
	}
	if !strings.Contains(location, "/callback") {
		t.Errorf("redirect should include callback path, got: %s", location)
	}
}

func TestLogin_UnsupportedContentType(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)

	// non-JSON content type is rejected
	result := testutil.Post(env.Router, "/login", "data", nil,
		testutil.Header{Key: "Content-Type", Value: "text/plain"})
	testutil.ExpectStatus(t, http.StatusUnsupportedMediaType, result)
}

func TestLogin_InvalidCredentials(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)

	// setup env
	env.RegisterTestUser(t, "alice", "password123")

	// wrong password returns 401
	body := `{
		"handle": "alice",
		"secret": "wrongpassword",
		"service": "test-service"
	}`
	result := testutil.PostJSON(env.Router, "/login", body, nil)
	testutil.ExpectStatus(t, http.StatusUnauthorized, result)
}

func TestLogin_UnknownUser(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)

	// login with non-existent user returns 401
	body := `{
		"handle": "unknown",
		"secret": "password",
		"service": "test-service"
	}`
	result := testutil.PostJSON(env.Router, "/login", body, nil)
	testutil.ExpectStatus(t, http.StatusUnauthorized, result)
}

func TestLogin_UnknownService(t *testing.T) {
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
	result := testutil.PostJSON(env.Router, "/login", body, nil)
	testutil.ExpectStatus(t, http.StatusBadRequest, result)
}

func TestLogin_InvalidJSON(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)

	// malformed JSON returns 400
	result := testutil.PostJSON(env.Router, "/login", "not-json", nil)
	testutil.ExpectStatus(t, http.StatusBadRequest, result)
}

func TestLogin_MissingFields(t *testing.T) {
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
			result := testutil.PostJSON(env.Router, "/login", tt.body, nil)
			// should either fail at login or return auth error
			if result.Code == http.StatusSeeOther {
				t.Error("should not redirect with missing fields")
			}
		})
	}
}
