package api_test

import (
	"net/http"
	"strings"
	"testing"

	"git.sr.ht/~jakintosh/command-go/pkg/wire"
	"git.sr.ht/~jakintosh/consent/internal/service"
	"git.sr.ht/~jakintosh/consent/internal/testutil"
)

func TestAPIRegister_Success(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)
	authHeader := env.APIKeyHeader(t)

	body := `{
		"username": "newuser",
		"password": "securepass"
	}`
	result := wire.TestPost[any](env.Router, "/admin/register", body, jsonHeader, authHeader)
	result.ExpectStatus(t, http.StatusOK)
}

func TestAPIRegister_InvalidJSON(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)
	authHeader := env.APIKeyHeader(t)

	result := wire.TestPost[any](env.Router, "/admin/register", "not-json", jsonHeader, authHeader)
	result.ExpectStatus(t, http.StatusBadRequest)
	result.ExpectError(t)
}

func TestAPIRegister_DuplicateUser(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)
	authHeader := env.APIKeyHeader(t)

	body := `{
		"username": "alice",
		"password": "pass1"
	}`
	wire.TestPost[any](env.Router, "/admin/register", body, jsonHeader, authHeader)

	body2 := `{
		"username": "alice",
		"password": "pass2"
	}`
	result := wire.TestPost[any](env.Router, "/admin/register", body2, jsonHeader, authHeader)
	result.ExpectStatus(t, http.StatusConflict)
	result.ExpectError(t)
}

func TestAPIRegister_ThenLogin(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)
	authHeader := env.APIKeyHeader(t)

	regBody := `{
		"username": "newuser",
		"password": "mypassword"
	}`
	result := wire.TestPost[any](env.Router, "/admin/register", regBody, jsonHeader, authHeader)
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
		t.Errorf("login after register should work, got redirect: %s", location)
	}
}

func TestAPIRegister_EmptyBody(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)
	authHeader := env.APIKeyHeader(t)

	result := wire.TestPost[any](env.Router, "/admin/register", "{}", jsonHeader, authHeader)
	result.ExpectStatus(t, http.StatusBadRequest)
	result.ExpectError(t)
}

func TestAPIRegister_MultipleUsers(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)
	authHeader := env.APIKeyHeader(t)

	users := []string{"alice", "bob", "charlie"}
	for _, user := range users {
		body := `{
			"username": "` + user + `",
			"password": "password"
		}`
		result := wire.TestPost[any](env.Router, "/admin/register", body, jsonHeader, authHeader)
		result.ExpectStatus(t, http.StatusOK)
	}
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
	result := wire.TestPut[any](env.Router, "/admin/services/svc-a", body, jsonHeader, authHeader)
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
	result := wire.TestPut[any](env.Router, "/admin/services/missing", body, jsonHeader, authHeader)
	result.ExpectStatus(t, http.StatusBadRequest)
	result.ExpectError(t)
}

func TestAPIUpdateService_InvalidRedirect(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)
	authHeader := env.APIKeyHeader(t)
	env.CreateTestService(t, "svc-a", "Service A", "aud-a", "https://svc-a.test/callback")

	body := `{"display":"Service A2","audience":"aud-b","redirect":"bad-url"}`
	result := wire.TestPut[any](env.Router, "/admin/services/svc-a", body, jsonHeader, authHeader)
	result.ExpectStatus(t, http.StatusBadRequest)
	result.ExpectError(t)
}

func TestAPIUpdateService_ProtectedName(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)
	authHeader := env.APIKeyHeader(t)

	body := `{"display":"Consent 2"}`
	result := wire.TestPut[any](env.Router, "/admin/services/"+service.InternalServiceName, body, jsonHeader, authHeader)

	result.ExpectStatus(t, http.StatusForbidden)
	result.ExpectError(t)
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
