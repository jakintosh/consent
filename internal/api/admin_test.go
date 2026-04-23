package api_test

import (
	"net/http"
	"strings"
	"testing"

	"git.sr.ht/~jakintosh/command-go/pkg/wire"
	"git.sr.ht/~jakintosh/consent/internal/service"
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
	result := wire.TestPost[apiUser](env.Router, "/admin/users", body, jsonHeader, authHeader)
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
	result.ExpectStatus(t, http.StatusBadRequest)
	result.ExpectError(t)
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
	result.ExpectStatus(t, http.StatusConflict)
	result.ExpectError(t)
}

func TestAPICreateUser_InvalidRoles(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)
	authHeader := env.APIKeyHeader(t)

	body := `{
		"username": "alice",
		"password": "pass1",
		"roles": ["bad role"]
	}`
	result := wire.TestPost[any](env.Router, "/admin/users", body, jsonHeader, authHeader)
	result.ExpectStatus(t, http.StatusBadRequest)
	result.ExpectError(t)
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
		"service": "consent"
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

	result := wire.TestGet[apiUser](env.Router, "/admin/users/"+user.Subject, authHeader)
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
	result.ExpectStatus(t, http.StatusBadRequest)
	result.ExpectError(t)
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

	result := wire.TestGet[[]apiUser](env.Router, "/admin/users", authHeader)
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

	body := `{"username":"alice-2","roles":["operator","billing"]}`
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

	body := `{"username":"alice-2"}`
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

	body := `{"roles":["bad role"]}`
	result := wire.TestPatch[any](env.Router, "/admin/users/"+user.Subject, body, jsonHeader, authHeader)
	result.ExpectStatus(t, http.StatusBadRequest)
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
	result.ExpectStatus(t, http.StatusBadRequest)
	result.ExpectError(t)
}

func TestAPICreateService_Success(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)
	authHeader := env.APIKeyHeader(t)

	body := `{"name":"svc-a","display":"Service A","audience":"aud-a","redirect":"https://svc-a.test/callback"}`
	result := wire.TestPost[any](env.Router, "/admin/services", body, jsonHeader, authHeader)
	result.ExpectStatus(t, http.StatusOK)

	svc, err := env.Service.GetServiceByName("svc-a")
	if err != nil {
		t.Fatalf("GetServiceByName failed: %v", err)
	}
	if svc.Display != "Service A" {
		t.Errorf("Display = %s, want Service A", svc.Display)
	}
}

func TestAPICreateService_DuplicateName(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)
	authHeader := env.APIKeyHeader(t)
	env.CreateTestService(t, "svc-a", "Service A", "aud-a", "https://svc-a.test/callback")

	body := `{"name":"svc-a","display":"Service A","audience":"aud-a","redirect":"https://svc-a.test/callback"}`
	result := wire.TestPost[any](env.Router, "/admin/services", body, jsonHeader, authHeader)
	result.ExpectStatus(t, http.StatusConflict)
	result.ExpectError(t)
}

func TestAPICreateService_InvalidRedirect(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)
	authHeader := env.APIKeyHeader(t)

	body := `{"name":"svc-a","display":"Service A","audience":"aud-a","redirect":"not-a-url"}`
	result := wire.TestPost[any](env.Router, "/admin/services", body, jsonHeader, authHeader)

	result.ExpectStatus(t, http.StatusBadRequest)
	result.ExpectError(t)
}

func TestAPICreateService_MissingName(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)
	authHeader := env.APIKeyHeader(t)

	body := `{"display":"Service A","audience":"aud-a","redirect":"https://svc-a.test/callback"}`
	result := wire.TestPost[any](env.Router, "/admin/services", body, jsonHeader, authHeader)

	result.ExpectStatus(t, http.StatusBadRequest)
	result.ExpectError(t)
}

func TestAPICreateService_MissingDisplay(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)
	authHeader := env.APIKeyHeader(t)

	body := `{"name":"svc-a","audience":"aud-a","redirect":"https://svc-a.test/callback"}`
	result := wire.TestPost[any](env.Router, "/admin/services", body, jsonHeader, authHeader)

	result.ExpectStatus(t, http.StatusBadRequest)
	result.ExpectError(t)
}

func TestAPICreateService_MissingAudience(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)
	authHeader := env.APIKeyHeader(t)

	body := `{"name":"svc-a","display":"Service A","redirect":"https://svc-a.test/callback"}`
	result := wire.TestPost[any](env.Router, "/admin/services", body, jsonHeader, authHeader)

	result.ExpectStatus(t, http.StatusBadRequest)
	result.ExpectError(t)
}

func TestAPICreateService_MissingRedirect(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)
	authHeader := env.APIKeyHeader(t)

	body := `{"name":"svc-a","display":"Service A","audience":"aud-a"}`
	result := wire.TestPost[any](env.Router, "/admin/services", body, jsonHeader, authHeader)

	result.ExpectStatus(t, http.StatusBadRequest)
	result.ExpectError(t)
}

func TestAPICreateService_ProtectedName(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)
	authHeader := env.APIKeyHeader(t)

	body := `{"name":"consent","display":"Consent","audience":"consent.test","redirect":"https://consent.test/auth/callback"}`
	result := wire.TestPost[any](env.Router, "/admin/services", body, jsonHeader, authHeader)

	result.ExpectStatus(t, http.StatusForbidden)
	result.ExpectError(t)
}

func TestAPIGetService_Success(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)
	authHeader := env.APIKeyHeader(t)
	env.CreateTestService(t, "svc-a", "Service A", "aud-a", "https://svc-a.test/callback")

	result := wire.TestGet[map[string]string](env.Router, "/admin/services/svc-a", authHeader)
	response := result.ExpectOK(t)
	if response["name"] != "svc-a" {
		t.Errorf("name = %s, want svc-a", response["name"])
	}
	if response["display"] != "Service A" {
		t.Errorf("display = %s, want Service A", response["display"])
	}
}

func TestAPIGetService_NotFound(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)
	authHeader := env.APIKeyHeader(t)

	result := wire.TestGet[any](env.Router, "/admin/services/missing", authHeader)
	result.ExpectStatus(t, http.StatusBadRequest)
	result.ExpectError(t)
}

func TestAPIUpdateService_Success(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)
	authHeader := env.APIKeyHeader(t)
	env.CreateTestService(t, "svc-a", "Service A", "aud-a", "https://svc-a.test/callback")

	body := `{"display":"Service A2","audience":"aud-b","redirect":"https://svc-a.test/new"}`
	result := wire.TestPatch[any](env.Router, "/admin/services/svc-a", body, jsonHeader, authHeader)
	result.ExpectStatus(t, http.StatusOK)

	svc, err := env.Service.GetServiceByName("svc-a")
	if err != nil {
		t.Fatalf("GetServiceByName failed: %v", err)
	}
	if svc.Display != "Service A2" {
		t.Errorf("Display = %s, want Service A2", svc.Display)
	}
}

func TestAPIUpdateService_NotFound(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)
	authHeader := env.APIKeyHeader(t)

	body := `{"display":"Service A2","audience":"aud-b","redirect":"https://svc-a.test/new"}`
	result := wire.TestPatch[any](env.Router, "/admin/services/missing", body, jsonHeader, authHeader)
	result.ExpectStatus(t, http.StatusBadRequest)
}

func TestAPIUpdateService_InvalidRedirect(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)
	authHeader := env.APIKeyHeader(t)
	env.CreateTestService(t, "svc-a", "Service A", "aud-a", "https://svc-a.test/callback")

	body := `{"display":"Service A2","audience":"aud-b","redirect":"bad-url"}`
	result := wire.TestPatch[any](env.Router, "/admin/services/svc-a", body, jsonHeader, authHeader)
	result.ExpectStatus(t, http.StatusBadRequest)
}

func TestAPIUpdateService_ProtectedName(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)
	authHeader := env.APIKeyHeader(t)

	body := `{"display":"Consent 2"}`
	result := wire.TestPatch[any](env.Router, "/admin/services/"+service.InternalServiceName, body, jsonHeader, authHeader)
	result.ExpectStatus(t, http.StatusForbidden)
}

func TestAPIDeleteService_Success(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)
	authHeader := env.APIKeyHeader(t)
	env.CreateTestService(t, "svc-a", "Service A", "aud-a", "https://svc-a.test/callback")

	result := wire.TestDelete[any](env.Router, "/admin/services/svc-a", authHeader)
	result.ExpectStatus(t, http.StatusOK)

	_, err := env.Service.GetServiceByName("svc-a")
	if err == nil {
		t.Fatal("expected error after delete")
	}
}

func TestAPIDeleteService_NotFound(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)
	authHeader := env.APIKeyHeader(t)

	result := wire.TestDelete[any](env.Router, "/admin/services/missing", authHeader)
	result.ExpectStatus(t, http.StatusBadRequest)
	result.ExpectError(t)
}

func TestAPIDeleteService_ProtectedName(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)
	authHeader := env.APIKeyHeader(t)

	result := wire.TestDelete[any](env.Router, "/admin/services/"+service.InternalServiceName, authHeader)

	result.ExpectStatus(t, http.StatusForbidden)
	result.ExpectError(t)
}

func TestAPIListServices_Seeded(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)
	authHeader := env.APIKeyHeader(t)

	result := wire.TestGet[[]map[string]string](env.Router, "/admin/services", authHeader)
	response := result.ExpectOK(t)
	if len(response) != 2 {
		t.Fatalf("expected 2 services (system + seeded), got %d", len(response))
	}
	if response[0]["name"] != service.InternalServiceName {
		t.Fatalf("expected internal service first, got %s", response[0]["name"])
	}
	if response[1]["name"] != "test-service" {
		t.Fatalf("expected seeded test-service second, got %s", response[1]["name"])
	}
}

func TestAPIListServices_Multiple(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)
	authHeader := env.APIKeyHeader(t)
	env.CreateTestService(t, "svc-a", "Service A", "aud-a", "https://svc-a.test/callback")
	env.CreateTestService(t, "svc-b", "Service B", "aud-b", "https://svc-b.test/callback")

	result := wire.TestGet[[]map[string]string](env.Router, "/admin/services", authHeader)
	response := result.ExpectOK(t)
	if len(response) != 4 {
		t.Fatalf("expected 4 services, got %d", len(response))
	}
	if response[0]["name"] != service.InternalServiceName {
		t.Errorf("expected internal service first, got %s", response[0]["name"])
	}
	if response[1]["name"] != "svc-a" {
		t.Errorf("expected svc-a second, got %s", response[1]["name"])
	}
}

func TestAPICreateRole_Success(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)
	authHeader := env.APIKeyHeader(t)

	body := `{"name":"editor","display":"Content Editor"}`
	result := wire.TestPost[apiRole](env.Router, "/admin/roles", body, jsonHeader, authHeader)
	response := result.ExpectOK(t)
	if response.Name != "editor" {
		t.Fatalf("role.Name = %s, want editor", response.Name)
	}
	if response.Display != "Content Editor" {
		t.Fatalf("role.Display = %s, want Content Editor", response.Display)
	}
}

func TestAPICreateRole_EmptyName(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)
	authHeader := env.APIKeyHeader(t)

	body := `{"name":"","display":"Something"}`
	result := wire.TestPost[any](env.Router, "/admin/roles", body, jsonHeader, authHeader)
	result.ExpectStatus(t, http.StatusBadRequest)
	result.ExpectError(t)
}

func TestAPICreateRole_EmptyDisplay(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)
	authHeader := env.APIKeyHeader(t)

	body := `{"name":"editor","display":""}`
	result := wire.TestPost[any](env.Router, "/admin/roles", body, jsonHeader, authHeader)
	result.ExpectStatus(t, http.StatusBadRequest)
	result.ExpectError(t)
}

func TestAPICreateRole_AdminProtected(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)
	authHeader := env.APIKeyHeader(t)

	body := `{"name":"admin","display":"Administrator"}`
	result := wire.TestPost[any](env.Router, "/admin/roles", body, jsonHeader, authHeader)
	result.ExpectStatus(t, http.StatusForbidden)
	result.ExpectError(t)
}

func TestAPICreateRole_Duplicate(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)
	authHeader := env.APIKeyHeader(t)

	body := `{"name":"editor","display":"Editor"}`
	wire.TestPost[any](env.Router, "/admin/roles", body, jsonHeader, authHeader)

	body2 := `{"name":"editor","display":"Another"}`
	result := wire.TestPost[any](env.Router, "/admin/roles", body2, jsonHeader, authHeader)
	result.ExpectStatus(t, http.StatusConflict)
	result.ExpectError(t)
}

func TestAPIGetRole_Success(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)
	authHeader := env.APIKeyHeader(t)
	env.CreateTestRole(t, "editor", "Editor")

	result := wire.TestGet[apiRole](env.Router, "/admin/roles/editor", authHeader)
	response := result.ExpectOK(t)
	if response.Name != "editor" {
		t.Fatalf("role.Name = %s, want editor", response.Name)
	}
	if response.Display != "Editor" {
		t.Fatalf("role.Display = %s, want Editor", response.Display)
	}
}

func TestAPIGetRole_AdminSeeded(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)
	authHeader := env.APIKeyHeader(t)

	result := wire.TestGet[apiRole](env.Router, "/admin/roles/admin", authHeader)
	response := result.ExpectOK(t)
	if response.Name != "admin" {
		t.Fatalf("role.Name = %s, want admin", response.Name)
	}
}

func TestAPIGetRole_NotFound(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)
	authHeader := env.APIKeyHeader(t)

	result := wire.TestGet[any](env.Router, "/admin/roles/nonexistent", authHeader)
	result.ExpectStatus(t, http.StatusBadRequest)
	result.ExpectError(t)
}

func TestAPIGetRole_MissingName(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)
	authHeader := env.APIKeyHeader(t)

	result := wire.TestGet[any](env.Router, "/admin/roles/", authHeader)
	result.ExpectStatus(t, http.StatusOK)
}

func TestAPIUpdateRole_Success(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)
	authHeader := env.APIKeyHeader(t)
	env.CreateTestRole(t, "editor", "Editor")

	body := `{"display":"Senior Editor"}`
	result := wire.TestPut[apiRole](env.Router, "/admin/roles/editor", body, jsonHeader, authHeader)
	response := result.ExpectOK(t)
	if response.Display != "Senior Editor" {
		t.Fatalf("role.Display = %s, want Senior Editor", response.Display)
	}
}

func TestAPIUpdateRole_EmptyDisplay(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)
	authHeader := env.APIKeyHeader(t)
	env.CreateTestRole(t, "editor", "Editor")

	body := `{"display":""}`
	result := wire.TestPut[any](env.Router, "/admin/roles/editor", body, jsonHeader, authHeader)
	result.ExpectStatus(t, http.StatusBadRequest)
	result.ExpectError(t)
}

func TestAPIUpdateRole_AdminProtected(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)
	authHeader := env.APIKeyHeader(t)

	body := `{"display":"Super Admin"}`
	result := wire.TestPut[any](env.Router, "/admin/roles/admin", body, jsonHeader, authHeader)
	result.ExpectStatus(t, http.StatusForbidden)
	result.ExpectError(t)
}

func TestAPIUpdateRole_NotFound(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)
	authHeader := env.APIKeyHeader(t)

	body := `{"display":"Something"}`
	result := wire.TestPut[any](env.Router, "/admin/roles/nonexistent", body, jsonHeader, authHeader)
	result.ExpectStatus(t, http.StatusBadRequest)
	result.ExpectError(t)
}

func TestAPIDeleteRole_Success(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)
	authHeader := env.APIKeyHeader(t)
	env.CreateTestRole(t, "temp", "Temporary")

	result := wire.TestDelete[any](env.Router, "/admin/roles/temp", authHeader)
	result.ExpectStatus(t, http.StatusOK)

	_, err := env.Service.GetRole("temp")
	if err == nil {
		t.Fatal("expected error after delete")
	}
}

func TestAPIDeleteRole_AdminProtected(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)
	authHeader := env.APIKeyHeader(t)

	result := wire.TestDelete[any](env.Router, "/admin/roles/admin", authHeader)
	result.ExpectStatus(t, http.StatusForbidden)
	result.ExpectError(t)
}

func TestAPIDeleteRole_NotFound(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)
	authHeader := env.APIKeyHeader(t)

	result := wire.TestDelete[any](env.Router, "/admin/roles/nonexistent", authHeader)
	result.ExpectStatus(t, http.StatusBadRequest)
	result.ExpectError(t)
}

func TestAPIDeleteRole_InUse(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)
	authHeader := env.APIKeyHeader(t)

	env.CreateTestRole(t, "editor", "Editor")
	env.RegisterTestUser(t, "alice", "password")

	// Create a user with this role
	_, err := env.Service.CreateUser("bob", "password2", []string{"editor"})
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	result := wire.TestDelete[any](env.Router, "/admin/roles/editor", authHeader)
	result.ExpectStatus(t, http.StatusConflict)
	result.ExpectError(t)
}

func TestAPIListRoles(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)
	authHeader := env.APIKeyHeader(t)

	env.CreateTestRole(t, "editor", "Editor")
	env.CreateTestRole(t, "viewer", "Viewer")

	result := wire.TestGet[[]apiRole](env.Router, "/admin/roles", authHeader)
	response := result.ExpectOK(t)
	if len(response) != 3 {
		t.Fatalf("len(response) = %d, want 3 (admin seeded + editor + viewer)", len(response))
	}
	roleNames := make(map[string]bool)
	for _, r := range response {
		roleNames[r.Name] = true
	}
	if !roleNames["admin"] || !roleNames["editor"] || !roleNames["viewer"] {
		t.Fatalf("expected admin, editor, viewer in roles, got %#v", response)
	}
}

type apiRole struct {
	Name    string `json:"name"`
	Display string `json:"display"`
}

type apiUser struct {
	Subject string   `json:"subject"`
	Handle  string   `json:"username"`
	Roles   []string `json:"roles"`
}
