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
		"redirect":"https://svc-a.test/callback",
		"homepage":"https://svc-a.test",
		"logo":"https://svc-a.test/logo.png"
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
	if integration.Homepage != "https://svc-a.test" {
		t.Errorf("Homepage = %s, want https://svc-a.test", integration.Homepage)
	}
	if integration.Logo != "https://svc-a.test/logo.png" {
		t.Errorf("Logo = %s, want https://svc-a.test/logo.png", integration.Logo)
	}
}

func TestAPICreateIntegration_DuplicateName(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)
	authHeader := env.APIKeyHeader(t)
	env.CreateTestIntegration(t, "svc-a", "Service A", "aud-a", "https://svc-a.test/callback", "https://svc-a.test", "https://svc-a.test/logo.png")

	body := `{
		"name":"svc-a",
		"display":"Service A",
		"audience":"aud-a",
		"redirect":"https://svc-a.test/callback",
		"homepage":"https://svc-a.test",
		"logo":"https://svc-a.test/logo.png"
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
		"redirect":"not-a-url",
		"homepage":"https://svc-a.test",
		"logo":"https://svc-a.test/logo.png"
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
		"redirect":"https://svc-a.test/callback",
		"homepage":"https://svc-a.test",
		"logo":"https://svc-a.test/logo.png"
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
		"redirect":"https://svc-a.test/callback",
		"homepage":"https://svc-a.test",
		"logo":"https://svc-a.test/logo.png"
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
		"redirect":"https://svc-a.test/callback",
		"homepage":"https://svc-a.test",
		"logo":"https://svc-a.test/logo.png"
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
		"audience":"aud-a",
		"homepage":"https://svc-a.test",
		"logo":"https://svc-a.test/logo.png"
	}`
	result := wire.TestPost[any](env.Router, "/admin/integrations", body, jsonHeader, authHeader)

	result.ExpectStatusError(t, http.StatusBadRequest)
}

func TestAPICreateIntegration_MissingHomepage(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)
	authHeader := env.APIKeyHeader(t)

	body := `{
		"name":"svc-a",
		"display":"Service A",
		"audience":"aud-a",
		"redirect":"https://svc-a.test/callback",
		"logo":"https://svc-a.test/logo.png"
	}`
	result := wire.TestPost[any](env.Router, "/admin/integrations", body, jsonHeader, authHeader)

	result.ExpectStatusError(t, http.StatusBadRequest)
}

func TestAPICreateIntegration_MissingLogo(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)
	authHeader := env.APIKeyHeader(t)

	body := `{
		"name":"svc-a",
		"display":"Service A",
		"audience":"aud-a",
		"redirect":"https://svc-a.test/callback",
		"homepage":"https://svc-a.test"
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
		"redirect":"https://consent.test/auth/callback",
		"homepage":"https://consent.test",
		"logo":"https://consent.test/logo.png"
	}`
	result := wire.TestPost[any](env.Router, "/admin/integrations", body, jsonHeader, authHeader)

	result.ExpectStatusError(t, http.StatusForbidden)
}

func TestAPIGetIntegration_Success(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)
	authHeader := env.APIKeyHeader(t)
	env.CreateTestIntegration(t, "svc-a", "Service A", "aud-a", "https://svc-a.test/callback", "https://svc-a.test", "https://svc-a.test/logo.png")

	result := wire.TestGet[map[string]string](env.Router, "/admin/integrations/svc-a", authHeader)
	response := result.ExpectOK(t)
	if response["name"] != "svc-a" {
		t.Errorf("name = %s, want svc-a", response["name"])
	}
	if response["display"] != "Service A" {
		t.Errorf("display = %s, want Service A", response["display"])
	}
	if response["homepage"] != "https://svc-a.test" {
		t.Errorf("homepage = %s, want https://svc-a.test", response["homepage"])
	}
	if response["logo"] != "https://svc-a.test/logo.png" {
		t.Errorf("logo = %s, want https://svc-a.test/logo.png", response["logo"])
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
	env.CreateTestIntegration(t, "svc-a", "Service A", "aud-a", "https://svc-a.test/callback", "https://svc-a.test", "https://svc-a.test/logo.png")

	body := `{
		"display":"Service A2",
		"audience":"aud-b",
		"redirect":"https://svc-a.test/new",
		"homepage":"https://svc-a-v2.test",
		"logo":"https://svc-a.test/logo-v2.png"
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
	if integration.Homepage != "https://svc-a-v2.test" {
		t.Errorf("Homepage = %s, want https://svc-a-v2.test", integration.Homepage)
	}
	if integration.Logo != "https://svc-a.test/logo-v2.png" {
		t.Errorf("Logo = %s, want https://svc-a.test/logo-v2.png", integration.Logo)
	}
}

func TestAPIUpdateIntegration_NotFound(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)
	authHeader := env.APIKeyHeader(t)

	body := `{
		"display":"Service A2",
		"audience":"aud-b",
		"redirect":"https://svc-a.test/new",
		"homepage":"https://svc-a-v2.test",
		"logo":"https://svc-a.test/logo-v2.png"
	}`
	result := wire.TestPatch[any](env.Router, "/admin/integrations/missing", body, jsonHeader, authHeader)
	result.ExpectStatus(t, http.StatusBadRequest)
}

func TestAPIUpdateIntegration_InvalidRedirect(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)
	authHeader := env.APIKeyHeader(t)
	env.CreateTestIntegration(t, "svc-a", "Service A", "aud-a", "https://svc-a.test/callback", "https://svc-a.test", "https://svc-a.test/logo.png")

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
		"display":"Consent 2",
		"homepage":"https://consent.test",
		"logo":"https://consent.test/logo.png"
	}`
	result := wire.TestPatch[any](env.Router, "/admin/integrations/"+service.InternalIntegrationName, body, jsonHeader, authHeader)
	result.ExpectStatus(t, http.StatusForbidden)
}

func TestAPIDeleteIntegration_Success(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)
	authHeader := env.APIKeyHeader(t)
	env.CreateTestIntegration(t, "svc-a", "Service A", "aud-a", "https://svc-a.test/callback", "https://svc-a.test", "https://svc-a.test/logo.png")

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

func TestAPIUpdateIntegration_MissingHomepage(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)
	authHeader := env.APIKeyHeader(t)
	env.CreateTestIntegration(t, "svc-a", "Service A", "aud-a", "https://svc-a.test/callback", "https://svc-a.test", "https://svc-a.test/logo.png")

	body := `{
		"display":"Service A2",
		"homepage":""
	}`
	result := wire.TestPatch[any](env.Router, "/admin/integrations/svc-a", body, jsonHeader, authHeader)
	result.ExpectStatus(t, http.StatusBadRequest)
}

func TestAPIUpdateIntegration_MissingLogo(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)
	authHeader := env.APIKeyHeader(t)
	env.CreateTestIntegration(t, "svc-a", "Service A", "aud-a", "https://svc-a.test/callback", "https://svc-a.test", "https://svc-a.test/logo.png")

	body := `{
		"display":"Service A2",
		"logo":""
	}`
	result := wire.TestPatch[any](env.Router, "/admin/integrations/svc-a", body, jsonHeader, authHeader)
	result.ExpectStatus(t, http.StatusBadRequest)
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
	if response[1]["homepage"] != "http://test-integration.local" {
		t.Errorf("homepage = %s, want http://test-integration.local", response[1]["homepage"])
	}
	if response[1]["logo"] != "http://test-integration.local/logo.png" {
		t.Errorf("logo = %s, want http://test-integration.local/logo.png", response[1]["logo"])
	}
}

func TestAPIListIntegrations_Multiple(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)
	authHeader := env.APIKeyHeader(t)
	env.CreateTestIntegration(t, "svc-a", "Service A", "aud-a", "https://svc-a.test/callback", "https://svc-a.test", "https://svc-a.test/logo.png")
	env.CreateTestIntegration(t, "svc-b", "Service B", "aud-b", "https://svc-b.test/callback", "https://svc-b.test", "https://svc-b.test/logo.png")

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
	if response[1]["homepage"] != "https://svc-a.test" {
		t.Errorf("svc-a homepage = %s, want https://svc-a.test", response[1]["homepage"])
	}
}
