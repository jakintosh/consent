package api_test

import (
	"net/http"
	"testing"

	"git.sr.ht/~jakintosh/command-go/pkg/wire"
	"git.sr.ht/~jakintosh/consent/internal/service"
	"git.sr.ht/~jakintosh/consent/internal/testutil"
)

func TestAPICreateIntegration_Success(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)
	authHeader := env.APIKeyHeader(t)

	body := `{
		"name":"svc-a",
		"display":"Service A",
		"audience":"aud-a",
		"redirect":"https://svc-a.test/callback"
	}`
	result := wire.TestPost[any](env.Router, "/admin/integrations", body, jsonHeader, authHeader)
	result.ExpectStatus(t, http.StatusOK)

	integration, err := env.Service.GetIntegration("svc-a")
	if err != nil {
		t.Fatalf("GetIntegration failed: %v", err)
	}
	if integration.Display != "Service A" {
		t.Errorf("Display = %s, want Service A", integration.Display)
	}
}

func TestAPICreateIntegration_DuplicateName(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)
	authHeader := env.APIKeyHeader(t)
	env.CreateTestIntegration(t, "svc-a", "Service A", "aud-a", "https://svc-a.test/callback")

	body := `{
		"name":"svc-a",
		"display":"Service A",
		"audience":"aud-a",
		"redirect":"https://svc-a.test/callback"
	}`
	result := wire.TestPost[any](env.Router, "/admin/integrations", body, jsonHeader, authHeader)
	result.ExpectStatusError(t, http.StatusConflict)
}

func TestAPICreateIntegration_InvalidRedirect(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)
	authHeader := env.APIKeyHeader(t)

	body := `{
		"name":"svc-a",
		"display":"Service A",
		"audience":"aud-a",
		"redirect":"not-a-url"
	}`
	result := wire.TestPost[any](env.Router, "/admin/integrations", body, jsonHeader, authHeader)

	result.ExpectStatusError(t, http.StatusBadRequest)
}

func TestAPICreateIntegration_MissingName(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)
	authHeader := env.APIKeyHeader(t)

	body := `{
		"display":"Service A",
		"audience":"aud-a",
		"redirect":"https://svc-a.test/callback"
	}`
	result := wire.TestPost[any](env.Router, "/admin/integrations", body, jsonHeader, authHeader)

	result.ExpectStatusError(t, http.StatusBadRequest)
}

func TestAPICreateIntegration_MissingDisplay(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)
	authHeader := env.APIKeyHeader(t)

	body := `{
		"name":"svc-a",
		"audience":"aud-a",
		"redirect":"https://svc-a.test/callback"
	}`
	result := wire.TestPost[any](env.Router, "/admin/integrations", body, jsonHeader, authHeader)

	result.ExpectStatusError(t, http.StatusBadRequest)
}

func TestAPICreateIntegration_MissingAudience(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)
	authHeader := env.APIKeyHeader(t)

	body := `{
		"name":"svc-a",
		"display":"Service A",
		"redirect":"https://svc-a.test/callback"
	}`
	result := wire.TestPost[any](env.Router, "/admin/integrations", body, jsonHeader, authHeader)

	result.ExpectStatusError(t, http.StatusBadRequest)
}

func TestAPICreateIntegration_MissingRedirect(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)
	authHeader := env.APIKeyHeader(t)

	body := `{
		"name":"svc-a",
		"display":"Service A",
		"audience":"aud-a"
	}`
	result := wire.TestPost[any](env.Router, "/admin/integrations", body, jsonHeader, authHeader)

	result.ExpectStatusError(t, http.StatusBadRequest)
}

func TestAPICreateIntegration_ProtectedName(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)
	authHeader := env.APIKeyHeader(t)

	body := `{
		"name":"consent",
		"display":"Consent",
		"audience":"consent.test",
		"redirect":"https://consent.test/auth/callback"
	}`
	result := wire.TestPost[any](env.Router, "/admin/integrations", body, jsonHeader, authHeader)

	result.ExpectStatusError(t, http.StatusForbidden)
}

func TestAPIGetIntegration_Success(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)
	authHeader := env.APIKeyHeader(t)
	env.CreateTestIntegration(t, "svc-a", "Service A", "aud-a", "https://svc-a.test/callback")

	result := wire.TestGet[map[string]string](env.Router, "/admin/integrations/svc-a", authHeader)
	response := result.ExpectOK(t)
	if response["name"] != "svc-a" {
		t.Errorf("name = %s, want svc-a", response["name"])
	}
	if response["display"] != "Service A" {
		t.Errorf("display = %s, want Service A", response["display"])
	}
}

func TestAPIGetIntegration_NotFound(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)
	authHeader := env.APIKeyHeader(t)

	result := wire.TestGet[any](env.Router, "/admin/integrations/missing", authHeader)
	result.ExpectStatusError(t, http.StatusBadRequest)
}

func TestAPIUpdateIntegration_Success(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)
	authHeader := env.APIKeyHeader(t)
	env.CreateTestIntegration(t, "svc-a", "Service A", "aud-a", "https://svc-a.test/callback")

	body := `{
		"display":"Service A2",
		"audience":"aud-b",
		"redirect":"https://svc-a.test/new"
	}`
	result := wire.TestPatch[any](env.Router, "/admin/integrations/svc-a", body, jsonHeader, authHeader)
	result.ExpectStatus(t, http.StatusOK)

	integration, err := env.Service.GetIntegration("svc-a")
	if err != nil {
		t.Fatalf("GetIntegration failed: %v", err)
	}
	if integration.Display != "Service A2" {
		t.Errorf("Display = %s, want Service A2", integration.Display)
	}
}

func TestAPIUpdateIntegration_NotFound(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)
	authHeader := env.APIKeyHeader(t)

	body := `{
		"display":"Service A2",
		"audience":"aud-b",
		"redirect":"https://svc-a.test/new"
	}`
	result := wire.TestPatch[any](env.Router, "/admin/integrations/missing", body, jsonHeader, authHeader)
	result.ExpectStatus(t, http.StatusBadRequest)
}

func TestAPIUpdateIntegration_InvalidRedirect(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)
	authHeader := env.APIKeyHeader(t)
	env.CreateTestIntegration(t, "svc-a", "Service A", "aud-a", "https://svc-a.test/callback")

	body := `{
		"display":"Service A2",
		"audience":"aud-b",
		"redirect":"bad-url"
	}`
	result := wire.TestPatch[any](env.Router, "/admin/integrations/svc-a", body, jsonHeader, authHeader)
	result.ExpectStatus(t, http.StatusBadRequest)
}

func TestAPIUpdateIntegration_ProtectedName(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)
	authHeader := env.APIKeyHeader(t)

	body := `{
		"display":"Consent 2"
	}`
	result := wire.TestPatch[any](env.Router, "/admin/integrations/"+service.InternalIntegrationName, body, jsonHeader, authHeader)
	result.ExpectStatus(t, http.StatusForbidden)
}

func TestAPIDeleteIntegration_Success(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)
	authHeader := env.APIKeyHeader(t)
	env.CreateTestIntegration(t, "svc-a", "Service A", "aud-a", "https://svc-a.test/callback")

	result := wire.TestDelete[any](env.Router, "/admin/integrations/svc-a", authHeader)
	result.ExpectStatus(t, http.StatusOK)

	_, err := env.Service.GetIntegration("svc-a")
	if err == nil {
		t.Fatal("expected error after delete")
	}
}

func TestAPIDeleteIntegration_NotFound(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)
	authHeader := env.APIKeyHeader(t)

	result := wire.TestDelete[any](env.Router, "/admin/integrations/missing", authHeader)
	result.ExpectStatusError(t, http.StatusBadRequest)
}

func TestAPIDeleteIntegration_ProtectedName(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)
	authHeader := env.APIKeyHeader(t)

	result := wire.TestDelete[any](env.Router, "/admin/integrations/"+service.InternalIntegrationName, authHeader)

	result.ExpectStatusError(t, http.StatusForbidden)
}

func TestAPIListIntegrations_Seeded(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)
	authHeader := env.APIKeyHeader(t)

	result := wire.TestGet[[]map[string]string](env.Router, "/admin/integrations", authHeader)
	response := result.ExpectOK(t)
	if len(response) != 2 {
		t.Fatalf("expected 2 integrations (system + seeded), got %d", len(response))
	}
	if response[0]["name"] != service.InternalIntegrationName {
		t.Fatalf("expected internal integration first, got %s", response[0]["name"])
	}
	if response[1]["name"] != "test-integration" {
		t.Fatalf("expected seeded test-integration second, got %s", response[1]["name"])
	}
}

func TestAPIListIntegrations_Multiple(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)
	authHeader := env.APIKeyHeader(t)
	env.CreateTestIntegration(t, "svc-a", "Service A", "aud-a", "https://svc-a.test/callback")
	env.CreateTestIntegration(t, "svc-b", "Service B", "aud-b", "https://svc-b.test/callback")

	result := wire.TestGet[[]map[string]string](env.Router, "/admin/integrations", authHeader)
	response := result.ExpectOK(t)
	if len(response) != 4 {
		t.Fatalf("expected 4 integrations, got %d", len(response))
	}
	if response[0]["name"] != service.InternalIntegrationName {
		t.Errorf("expected internal integration first, got %s", response[0]["name"])
	}
	if response[1]["name"] != "svc-a" {
		t.Errorf("expected svc-a second, got %s", response[1]["name"])
	}
}
