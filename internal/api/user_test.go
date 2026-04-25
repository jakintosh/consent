package api_test

import (
	"net/http"
	"strings"
	"testing"

	"git.sr.ht/~jakintosh/command-go/pkg/wire"
	"git.sr.ht/~jakintosh/consent/internal/api"
	"git.sr.ht/~jakintosh/consent/internal/testutil"
)

func TestAPICreateUser_Success(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)
	authHeader := env.APIKeyHeader(t)

	env.CreateTestRole(t, "ops", "Operations")

	body := `{
		"username": "newuser",
		"password": "securepass",
		"roles": ["admin","ops"]
	}`
	result := wire.TestPost[api.User](env.Router, "/admin/users", body, jsonHeader, authHeader)
	response := result.ExpectOK(t)
	if response.Subject == "" {
		t.Fatal("expected subject")
	}
	if response.Handle != "newuser" {
		t.Fatalf("handle = %s, want newuser", response.Handle)
	}
	if len(response.Roles) != 2 {
		t.Fatalf("len(response.Roles) = %d, want 2", len(response.Roles))
	}
	roleSet := make(map[string]bool)
	for _, r := range response.Roles {
		roleSet[r] = true
	}
	if !roleSet["admin"] || !roleSet["ops"] {
		t.Fatalf("roles = %#v, want admin and ops", response.Roles)
	}
}

func TestAPICreateUser_InvalidJSON(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)
	authHeader := env.APIKeyHeader(t)

	result := wire.TestPost[any](env.Router, "/admin/users", "not-json", jsonHeader, authHeader)
	result.ExpectStatusError(t, http.StatusBadRequest)
}

func TestAPICreateUser_DuplicateUser(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)
	authHeader := env.APIKeyHeader(t)

	body := `{
		"username": "alice",
		"password": "pass1"
	}`
	wire.TestPost[any](env.Router, "/admin/users", body, jsonHeader, authHeader)

	body2 := `{
		"username": "alice",
		"password": "pass2"
	}`
	result := wire.TestPost[any](env.Router, "/admin/users", body2, jsonHeader, authHeader)
	result.ExpectStatusError(t, http.StatusConflict)
}

func TestAPICreateUser_InvalidRoles(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)
	authHeader := env.APIKeyHeader(t)

	// roles with spaces are auto-created by the database
	body := `{
		"username": "alice",
		"password": "pass1",
		"roles": ["bad role"]
	}`
	result := wire.TestPost[any](env.Router, "/admin/users", body, jsonHeader, authHeader)
	result.ExpectStatus(t, http.StatusOK)
}

func TestAPICreateUser_ThenLogin(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)
	authHeader := env.APIKeyHeader(t)

	env.CreateTestRole(t, "viewer", "Viewer")

	regBody := `{
		"username": "newuser",
		"password": "mypassword",
		"roles": ["viewer"]
	}`
	result := wire.TestPost[any](env.Router, "/admin/users", regBody, jsonHeader, authHeader)
	result.ExpectStatus(t, http.StatusOK)

	loginBody := `{
		"handle": "newuser",
		"secret": "mypassword",
		"integration": "consent"
	}`
	loginResult := wire.TestPost[any](env.Router, "/auth/login", loginBody, jsonHeader)
	loginResult.ExpectStatus(t, http.StatusSeeOther)
	location := loginResult.Headers.Get("Location")
	if location == "" {
		t.Fatal("expected Location header in redirect")
	}
	if !strings.Contains(location, "auth_code=") {
		t.Errorf("login after create should work, got redirect: %s", location)
	}
}

func TestAPIGetUser_Success(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)
	authHeader := env.APIKeyHeader(t)
	user, err := env.Service.CreateUser("alice", "password", []string{"admin"})
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	result := wire.TestGet[api.User](env.Router, "/admin/users/"+user.Subject, authHeader)
	response := result.ExpectOK(t)
	if response.Subject != user.Subject {
		t.Errorf("subject = %s, want %s", response.Subject, user.Subject)
	}
	if response.Handle != "alice" {
		t.Errorf("handle = %s, want alice", response.Handle)
	}
	if len(response.Roles) != 1 || response.Roles[0] != "admin" {
		t.Errorf("roles = %#v, want [admin]", response.Roles)
	}
}

func TestAPIGetUser_NotFound(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)
	authHeader := env.APIKeyHeader(t)

	result := wire.TestGet[any](env.Router, "/admin/users/missing", authHeader)
	result.ExpectStatusError(t, http.StatusBadRequest)
}

func TestAPIListUsers(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)
	authHeader := env.APIKeyHeader(t)
	if _, err := env.Service.CreateUser("alice", "password", []string{"admin"}); err != nil {
		t.Fatalf("CreateUser alice failed: %v", err)
	}
	if _, err := env.Service.CreateUser("bob", "password", nil); err != nil {
		t.Fatalf("CreateUser bob failed: %v", err)
	}

	result := wire.TestGet[[]api.User](env.Router, "/admin/users", authHeader)
	response := result.ExpectOK(t)
	if len(response) != 2 {
		t.Fatalf("expected 2 users, got %d", len(response))
	}
	if response[0].Handle != "alice" || response[1].Handle != "bob" {
		t.Fatalf("unexpected user order: %#v", response)
	}
}

func TestAPIUpdateUser_Success(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)
	authHeader := env.APIKeyHeader(t)
	user, err := env.Service.CreateUser("alice", "password", []string{"admin"})
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	env.CreateTestRole(t, "operator", "Operator")
	env.CreateTestRole(t, "billing", "Billing")

	body := `{
		"username":"alice-2",
		"roles":[
			"operator",
			"billing"
		]
	}`
	result := wire.TestPatch[any](env.Router, "/admin/users/"+user.Subject, body, jsonHeader, authHeader)
	result.ExpectStatus(t, http.StatusOK)

	updated, err := env.Service.GetUser(user.Subject)
	if err != nil {
		t.Fatalf("GetUser failed: %v", err)
	}
	if updated.Handle != "alice-2" {
		t.Errorf("handle = %s, want alice-2", updated.Handle)
	}
	if len(updated.Roles) != 2 {
		t.Fatalf("len(updated.Roles) = %d, want 2", len(updated.Roles))
	}
	roleSet := make(map[string]bool)
	for _, r := range updated.Roles {
		roleSet[r] = true
	}
	if !roleSet["operator"] || !roleSet["billing"] {
		t.Errorf("roles = %#v, want operator and billing", updated.Roles)
	}
}

func TestAPIUpdateUser_NotFound(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)
	authHeader := env.APIKeyHeader(t)

	body := `{
		"username":"alice-2"
	}`
	result := wire.TestPatch[any](env.Router, "/admin/users/missing", body, jsonHeader, authHeader)
	result.ExpectStatus(t, http.StatusBadRequest)
}

func TestAPIUpdateUser_InvalidRoles(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)
	authHeader := env.APIKeyHeader(t)
	user, err := env.Service.CreateUser("alice", "password", []string{"admin"})
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	// roles with spaces are auto-created by the database
	body := `{
		"roles":[
			"bad role"
		]
	}`
	result := wire.TestPatch[any](env.Router, "/admin/users/"+user.Subject, body, jsonHeader, authHeader)
	result.ExpectStatus(t, http.StatusOK)
}

func TestAPIDeleteUser_Success(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)
	authHeader := env.APIKeyHeader(t)
	user, err := env.Service.CreateUser("alice", "password", nil)
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	result := wire.TestDelete[any](env.Router, "/admin/users/"+user.Subject, authHeader)
	result.ExpectStatus(t, http.StatusOK)

	_, err = env.Service.GetUser(user.Subject)
	if err == nil {
		t.Fatal("expected error after delete")
	}
}

func TestAPIDeleteUser_NotFound(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)
	authHeader := env.APIKeyHeader(t)

	result := wire.TestDelete[any](env.Router, "/admin/users/missing", authHeader)
	result.ExpectStatusError(t, http.StatusBadRequest)
}
