package api_test

import (
	"net/http"
	"testing"

	"git.sr.ht/~jakintosh/command-go/pkg/wire"
	"git.sr.ht/~jakintosh/consent/internal/api"
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
		"audience":"svc-a.test",
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
	env.CreateTestIntegration(t, "svc-a", "Service A", "svc-a.test", "https://svc-a.test/callback", "https://svc-a.test", "https://svc-a.test/logo.png", nil)

	body := `{
		"name":"svc-a",
		"display":"Service A",
		"audience":"svc-a.test",
		"redirect":"https://svc-a.test/callback",
		"homepage":"https://svc-a.test",
		"logo":"https://svc-a.test/logo.png"
	}`
	result := wire.TestPost[any](env.Router, "/admin/integrations", body, jsonHeader, authHeader)
	result.ExpectStatusError(t, http.StatusConflict)
}

func TestAPICreateIntegration_RequiredRoles(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)
	authHeader := env.APIKeyHeader(t)
	env.CreateTestRole(t, "editor", "Editor")

	body := `{
		"name":"svc-a",
		"display":"Service A",
		"audience":"svc-a.test",
		"redirect":"https://svc-a.test/callback",
		"homepage":"https://svc-a.test",
		"logo":"https://svc-a.test/logo.png",
		"required_roles":["editor"]
	}`
	result := wire.TestPost[any](env.Router, "/admin/integrations", body, jsonHeader, authHeader)
	result.ExpectStatus(t, http.StatusOK)

	integration, err := env.Service.GetIntegration("svc-a")
	if err != nil {
		t.Fatalf("GetIntegration failed: %v", err)
	}
	if len(integration.RequiredRoles) != 1 || integration.RequiredRoles[0] != "editor" {
		t.Fatalf("RequiredRoles = %v, want [editor]", integration.RequiredRoles)
	}
}

func TestAPICreateIntegration_RequiredRoleNotFound(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)
	authHeader := env.APIKeyHeader(t)

	body := `{
		"name":"svc-a",
		"display":"Service A",
		"audience":"svc-a.test",
		"redirect":"https://svc-a.test/callback",
		"homepage":"https://svc-a.test",
		"logo":"https://svc-a.test/logo.png",
		"required_roles":["missing"]
	}`
	result := wire.TestPost[any](env.Router, "/admin/integrations", body, jsonHeader, authHeader)
	result.ExpectStatusError(t, http.StatusBadRequest)
}

func TestAPICreateIntegration_RedirectHostMustMatchAudience(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)
	authHeader := env.APIKeyHeader(t)

	body := `{
		"name":"svc-a",
		"display":"Service A",
		"audience":"svc-a.test",
		"redirect":"https://other.test/callback",
		"homepage":"https://svc-a.test",
		"logo":"https://svc-a.test/logo.png"
	}`
	result := wire.TestPost[any](env.Router, "/admin/integrations", body, jsonHeader, authHeader)
	result.ExpectStatusError(t, http.StatusBadRequest)
}

func TestAPICreateIntegration_HomepageHostMustMatchAudience(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)
	authHeader := env.APIKeyHeader(t)

	body := `{
		"name":"svc-a",
		"display":"Service A",
		"audience":"svc-a.test",
		"redirect":"https://svc-a.test/callback",
		"homepage":"https://other.test",
		"logo":"https://svc-a.test/logo.png"
	}`
	result := wire.TestPost[any](env.Router, "/admin/integrations", body, jsonHeader, authHeader)
	result.ExpectStatusError(t, http.StatusBadRequest)
}

func TestAPICreateIntegration_InvalidRedirect(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)
	authHeader := env.APIKeyHeader(t)

	body := `{
		"name":"svc-a",
		"display":"Service A",
		"audience":"svc-a.test",
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
		"audience":"svc-a.test",
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
		"audience":"svc-a.test",
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
		"audience":"svc-a.test",
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
		"audience":"svc-a.test",
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
		"audience":"svc-a.test",
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
	env.CreateTestIntegration(t, "svc-a", "Service A", "svc-a.test", "https://svc-a.test/callback", "https://svc-a.test", "https://svc-a.test/logo.png", nil)

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

func TestAPIGetIntegration_RequiredRoles(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)
	authHeader := env.APIKeyHeader(t)
	env.CreateTestRole(t, "editor", "Editor")
	env.CreateTestIntegration(t, "svc-a", "Service A", "svc-a.test", "https://svc-a.test/callback", "https://svc-a.test", "https://svc-a.test/logo.png", []string{"editor"})

	result := wire.TestGet[api.Integration](env.Router, "/admin/integrations/svc-a", authHeader)
	response := result.ExpectOK(t)
	if len(response.RequiredRoles) != 1 || response.RequiredRoles[0] != "editor" {
		t.Fatalf("RequiredRoles = %v, want [editor]", response.RequiredRoles)
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
	env.CreateTestIntegration(t, "svc-a", "Service A", "svc-a.test", "https://svc-a.test/callback", "https://svc-a.test", "https://svc-a.test/logo.png", nil)

	body := `{
		"display":"Service A2",
		"audience":"svc-b.test",
		"redirect":"https://svc-b.test/new",
		"homepage":"https://svc-b.test",
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
	if integration.Homepage != "https://svc-b.test" {
		t.Errorf("Homepage = %s, want https://svc-b.test", integration.Homepage)
	}
	if integration.Logo != "https://svc-a.test/logo-v2.png" {
		t.Errorf("Logo = %s, want https://svc-a.test/logo-v2.png", integration.Logo)
	}
}

func TestAPIUpdateIntegration_RequiredRoles(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)
	authHeader := env.APIKeyHeader(t)
	env.CreateTestRole(t, "editor", "Editor")
	env.CreateTestIntegration(t, "svc-a", "Service A", "svc-a.test", "https://svc-a.test/callback", "https://svc-a.test", "https://svc-a.test/logo.png", nil)

	body := `{"required_roles":["editor"]}`
	result := wire.TestPatch[any](env.Router, "/admin/integrations/svc-a", body, jsonHeader, authHeader)
	result.ExpectStatus(t, http.StatusOK)

	integration, err := env.Service.GetIntegration("svc-a")
	if err != nil {
		t.Fatalf("GetIntegration failed: %v", err)
	}
	if len(integration.RequiredRoles) != 1 || integration.RequiredRoles[0] != "editor" {
		t.Fatalf("RequiredRoles = %v, want [editor]", integration.RequiredRoles)
	}
}

func TestAPIUpdateIntegration_ClearRequiredRoles(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)
	authHeader := env.APIKeyHeader(t)
	env.CreateTestRole(t, "editor", "Editor")
	env.CreateTestIntegration(t, "svc-a", "Service A", "svc-a.test", "https://svc-a.test/callback", "https://svc-a.test", "https://svc-a.test/logo.png", []string{"editor"})

	body := `{"required_roles":[]}`
	result := wire.TestPatch[any](env.Router, "/admin/integrations/svc-a", body, jsonHeader, authHeader)
	result.ExpectStatus(t, http.StatusOK)

	integration, err := env.Service.GetIntegration("svc-a")
	if err != nil {
		t.Fatalf("GetIntegration failed: %v", err)
	}
	if len(integration.RequiredRoles) != 0 {
		t.Fatalf("RequiredRoles = %v, want []", integration.RequiredRoles)
	}
}

func TestAPIUpdateIntegration_NotFound(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)
	authHeader := env.APIKeyHeader(t)

	body := `{
		"display":"Service A2",
		"audience":"svc-b.test",
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
	env.CreateTestIntegration(t, "svc-a", "Service A", "svc-a.test", "https://svc-a.test/callback", "https://svc-a.test", "https://svc-a.test/logo.png", nil)

	body := `{
		"display":"Service A2",
		"audience":"svc-b.test",
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
	env.CreateTestIntegration(t, "svc-a", "Service A", "svc-a.test", "https://svc-a.test/callback", "https://svc-a.test", "https://svc-a.test/logo.png", nil)

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
	env.CreateTestIntegration(t, "svc-a", "Service A", "svc-a.test", "https://svc-a.test/callback", "https://svc-a.test", "https://svc-a.test/logo.png", nil)

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
	env.CreateTestIntegration(t, "svc-a", "Service A", "svc-a.test", "https://svc-a.test/callback", "https://svc-a.test", "https://svc-a.test/logo.png", nil)

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
	if response[1]["homepage"] != "http://localhost:8080" {
		t.Errorf("homepage = %s, want http://localhost:8080", response[1]["homepage"])
	}
	if response[1]["logo"] != "http://localhost:8080/logo.png" {
		t.Errorf("logo = %s, want http://localhost:8080/logo.png", response[1]["logo"])
	}
}

func TestAPIListIntegrations_Multiple(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)
	authHeader := env.APIKeyHeader(t)
	env.CreateTestIntegration(t, "svc-a", "Service A", "svc-a.test", "https://svc-a.test/callback", "https://svc-a.test", "https://svc-a.test/logo.png", nil)
	env.CreateTestIntegration(t, "svc-b", "Service B", "svc-b.test", "https://svc-b.test/callback", "https://svc-b.test", "https://svc-b.test/logo.png", nil)

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

func TestAPIListIntegrations_RequiredRoles(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)
	authHeader := env.APIKeyHeader(t)
	env.CreateTestRole(t, "editor", "Editor")
	env.CreateTestIntegration(t, "svc-a", "Service A", "svc-a.test", "https://svc-a.test/callback", "https://svc-a.test", "https://svc-a.test/logo.png", []string{"editor"})

	result := wire.TestGet[[]api.Integration](env.Router, "/admin/integrations", authHeader)
	response := result.ExpectOK(t)
	var got *api.Integration
	for i := range response {
		if response[i].Name == "svc-a" {
			got = &response[i]
		}
	}
	if got == nil {
		t.Fatal("svc-a not found")
	}
	if len(got.RequiredRoles) != 1 || got.RequiredRoles[0] != "editor" {
		t.Fatalf("RequiredRoles = %v, want [editor]", got.RequiredRoles)
	}
}
