package service_test

import (
	"errors"
	"testing"

	"git.sr.ht/~jakintosh/consent/internal/service"
	"git.sr.ht/~jakintosh/consent/internal/testutil"
)

func TestCreateIntegration_Success(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)

	err := env.Service.CreateIntegration(
		"svc-a",
		"Service A",
		"svc-a.test",
		"https://svc-a.test/callback",
		"https://svc-a.test",
		"https://svc-a.test/logo.png",
		nil,
	)
	if err != nil {
		t.Fatalf("CreateIntegration failed: %v", err)
	}
}

func TestCreateIntegration_DuplicateName(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)

	if err := env.Service.CreateIntegration(
		"svc-a",
		"Service A",
		"svc-a.test",
		"https://svc-a.test/callback",
		"https://svc-a.test",
		"https://svc-a.test/logo.png",
		nil,
	); err != nil {
		t.Fatalf("CreateIntegration failed: %v", err)
	}

	err := env.Service.CreateIntegration(
		"svc-a",
		"Service A",
		"svc-a.test",
		"https://svc-a.test/callback",
		"https://svc-a.test",
		"https://svc-a.test/logo.png",
		nil,
	)
	if !errors.Is(err, service.ErrIntegrationExists) {
		t.Fatalf("expected ErrIntegrationExists, got %v", err)
	}
}

func TestCreateIntegration_DuplicateAudience(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)
	if err := env.Service.CreateIntegration(
		"svc-a",
		"Service A",
		"svc-a.test",
		"https://svc-a.test/callback",
		"https://svc-a.test",
		"https://svc-a.test/logo.png",
		nil,
	); err != nil {
		t.Fatalf("CreateIntegration failed: %v", err)
	}

	err := env.Service.CreateIntegration(
		"svc-b",
		"Service B",
		"svc-a.test",
		"https://svc-a.test/callback-b",
		"https://svc-a.test",
		"https://svc-b.test/logo.png",
		nil,
	)
	if !errors.Is(err, service.ErrIntegrationExists) {
		t.Fatalf("expected ErrIntegrationExists, got %v", err)
	}
}

func TestCreateIntegration_InvalidRedirect(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)

	err := env.Service.CreateIntegration(
		"svc-a",
		"Service A",
		"svc-a.test",
		"bad-url",
		"https://svc-a.test",
		"https://svc-a.test/logo.png",
		nil,
	)
	if !errors.Is(err, service.ErrInvalidRedirect) {
		t.Fatalf("expected ErrInvalidRedirect, got %v", err)
	}
}

func TestCreateIntegration_RedirectHostMustMatchAudience(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)

	err := env.Service.CreateIntegration(
		"svc-a",
		"Service A",
		"svc-a.test",
		"https://other.test/callback",
		"https://svc-a.test",
		"https://svc-a.test/logo.png",
		nil,
	)
	if !errors.Is(err, service.ErrInvalidRedirect) {
		t.Fatalf("expected ErrInvalidRedirect, got %v", err)
	}
}

func TestCreateIntegration_HomepageHostMustMatchAudience(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)

	err := env.Service.CreateIntegration(
		"svc-a",
		"Service A",
		"svc-a.test",
		"https://svc-a.test/callback",
		"https://other.test",
		"https://svc-a.test/logo.png",
		nil,
	)
	if !errors.Is(err, service.ErrInvalidIntegration) {
		t.Fatalf("expected ErrInvalidIntegration, got %v", err)
	}
}

func TestCreateIntegration_InvalidName(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)

	err := env.Service.CreateIntegration(
		"",
		"Service A",
		"svc-a.test",
		"https://svc-a.test/callback",
		"https://svc-a.test",
		"https://svc-a.test/logo.png",
		nil,
	)
	if !errors.Is(err, service.ErrInvalidIntegration) {
		t.Fatalf("expected ErrInvalidIntegration, got %v", err)
	}
}

func TestCreateIntegration_ProtectedName(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)

	err := env.Service.CreateIntegration(
		service.InternalIntegrationName,
		"Consent",
		"consent.test",
		"https://consent.test/auth/callback",
		"https://consent.test",
		"https://consent.test/logo.png",
		nil,
	)
	if !errors.Is(err, service.ErrIntegrationProtected) {
		t.Fatalf("expected ErrIntegrationProtected, got %v", err)
	}
}

func TestCreateIntegration_RequiredRoleNotFound(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)

	err := env.Service.CreateIntegration(
		"svc-a",
		"Service A",
		"svc-a.test",
		"https://svc-a.test/callback",
		"https://svc-a.test",
		"https://svc-a.test/logo.png",
		[]string{"missing"},
	)
	if !errors.Is(err, service.ErrRoleNotFound) {
		t.Fatalf("expected ErrRoleNotFound, got %v", err)
	}
}

func TestCreateIntegration_DuplicateRequiredRole(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)
	env.CreateTestRole(t, "editor", "Editor")

	err := env.Service.CreateIntegration(
		"svc-a",
		"Service A",
		"svc-a.test",
		"https://svc-a.test/callback",
		"https://svc-a.test",
		"https://svc-a.test/logo.png",
		[]string{"editor", "editor"},
	)
	if !errors.Is(err, service.ErrInvalidRole) {
		t.Fatalf("expected ErrInvalidRole, got %v", err)
	}
}

func TestGetIntegration_Success(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)
	env.CreateTestIntegration(
		t,
		"svc-a",
		"Service A",
		"svc-a.test",
		"https://svc-a.test/callback",
		"https://svc-a.test",
		"https://svc-a.test/logo.png",
		nil,
	)

	integration, err := env.Service.GetIntegration("svc-a")
	if err != nil {
		t.Fatalf("GetIntegration failed: %v", err)
	}
	if integration.Name != "svc-a" {
		t.Errorf("Name = %s, want svc-a", integration.Name)
	}
	if integration.Redirect != "https://svc-a.test/callback" {
		t.Errorf("Redirect = %s, want https://svc-a.test/callback", integration.Redirect)
	}
	if integration.Homepage != "https://svc-a.test" {
		t.Errorf("Homepage = %s, want https://svc-a.test", integration.Homepage)
	}
	if integration.Logo != "https://svc-a.test/logo.png" {
		t.Errorf("Logo = %s, want https://svc-a.test/logo.png", integration.Logo)
	}
}

func TestGetIntegration_NotFound(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)

	_, err := env.Service.GetIntegration("missing")
	if !errors.Is(err, service.ErrIntegrationNotFound) {
		t.Fatalf("expected ErrIntegrationNotFound, got %v", err)
	}
}

func TestCreateIntegration_MissingHomepage(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)

	err := env.Service.CreateIntegration(
		"svc-a",
		"Service A",
		"svc-a.test",
		"https://svc-a.test/callback",
		"",
		"https://svc-a.test/logo.png",
		nil,
	)
	if !errors.Is(err, service.ErrInvalidIntegration) {
		t.Fatalf("expected ErrInvalidIntegration, got %v", err)
	}
}

func TestCreateIntegration_MissingLogo(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)

	err := env.Service.CreateIntegration(
		"svc-a",
		"Service A",
		"svc-a.test",
		"https://svc-a.test/callback",
		"https://svc-a.test",
		"",
		nil,
	)
	if !errors.Is(err, service.ErrInvalidIntegration) {
		t.Fatalf("expected ErrInvalidIntegration, got %v", err)
	}
}

func TestUpdateIntegration_Success(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)
	env.CreateTestIntegration(
		t,
		"svc-a",
		"Service A",
		"svc-a.test",
		"https://svc-a.test/callback",
		"https://svc-a.test",
		"https://svc-a.test/logo.png",
		nil,
	)

	update := service.IntegrationUpdate{
		Display:  strPtr("Service A2"),
		Audience: strPtr("svc-b.test"),
		Redirect: strPtr("https://svc-b.test/new"),
		Homepage: strPtr("https://svc-b.test"),
		Logo:     strPtr("https://svc-a.test/logo-v2.png"),
	}
	err := env.Service.UpdateIntegration("svc-a", &update)
	if err != nil {
		t.Fatalf("UpdateIntegration failed: %v", err)
	}

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

func TestUpdateIntegration_NotFound(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)

	update := service.IntegrationUpdate{
		Display: strPtr("Service A2"),
	}
	err := env.Service.UpdateIntegration("missing", &update)
	if !errors.Is(err, service.ErrIntegrationNotFound) {
		t.Fatalf("expected ErrIntegrationNotFound, got %v", err)
	}
}

func TestUpdateIntegration_MissingHomepage(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)
	env.CreateTestIntegration(
		t,
		"svc-a",
		"Service A",
		"svc-a.test",
		"https://svc-a.test/callback",
		"https://svc-a.test",
		"https://svc-a.test/logo.png",
		nil,
	)

	update := service.IntegrationUpdate{
		Homepage: strPtr(""),
	}
	err := env.Service.UpdateIntegration("svc-a", &update)
	if !errors.Is(err, service.ErrInvalidIntegration) {
		t.Fatalf("expected ErrInvalidIntegration, got %v", err)
	}
}

func TestUpdateIntegration_MissingLogo(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)
	env.CreateTestIntegration(
		t,
		"svc-a",
		"Service A",
		"svc-a.test",
		"https://svc-a.test/callback",
		"https://svc-a.test",
		"https://svc-a.test/logo.png",
		nil,
	)

	logo := ""
	err := env.Service.UpdateIntegration("svc-a", &service.IntegrationUpdate{Logo: &logo})
	if !errors.Is(err, service.ErrInvalidIntegration) {
		t.Fatalf("expected ErrInvalidIntegration, got %v", err)
	}
}

func TestUpdateIntegration_InvalidRedirect(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)
	env.CreateTestIntegration(
		t,
		"svc-a",
		"Service A",
		"svc-a.test",
		"https://svc-a.test/callback",
		"https://svc-a.test",
		"https://svc-a.test/logo.png",
		nil,
	)

	update := service.IntegrationUpdate{
		Redirect: strPtr("bad-url"),
	}
	err := env.Service.UpdateIntegration("svc-a", &update)
	if !errors.Is(err, service.ErrInvalidRedirect) {
		t.Fatalf("expected ErrInvalidRedirect, got %v", err)
	}
}

func TestUpdateIntegration_RedirectHostMustMatchAudience(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)
	env.CreateTestIntegration(
		t,
		"svc-a",
		"Service A",
		"svc-a.test",
		"https://svc-a.test/callback",
		"https://svc-a.test",
		"https://svc-a.test/logo.png",
		nil,
	)

	redirect := "https://other.test/callback"
	err := env.Service.UpdateIntegration("svc-a", &service.IntegrationUpdate{Redirect: &redirect})
	if !errors.Is(err, service.ErrInvalidRedirect) {
		t.Fatalf("expected ErrInvalidRedirect, got %v", err)
	}
}

func TestUpdateIntegration_HomepageHostMustMatchAudience(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)
	env.CreateTestIntegration(
		t,
		"svc-a",
		"Service A",
		"svc-a.test",
		"https://svc-a.test/callback",
		"https://svc-a.test",
		"https://svc-a.test/logo.png",
		nil,
	)

	homepage := "https://other.test"
	err := env.Service.UpdateIntegration("svc-a", &service.IntegrationUpdate{Homepage: &homepage})
	if !errors.Is(err, service.ErrInvalidIntegration) {
		t.Fatalf("expected ErrInvalidIntegration, got %v", err)
	}
}

func TestUpdateIntegration_RequiredRoleNotFound(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)
	env.CreateTestIntegration(
		t,
		"svc-a",
		"Service A",
		"svc-a.test",
		"https://svc-a.test/callback",
		"https://svc-a.test",
		"https://svc-a.test/logo.png",
		nil,
	)

	roles := []string{"missing"}
	err := env.Service.UpdateIntegration("svc-a", &service.IntegrationUpdate{RequiredRoles: &roles})
	if !errors.Is(err, service.ErrRoleNotFound) {
		t.Fatalf("expected ErrRoleNotFound, got %v", err)
	}
}

func TestUpdateIntegration_ProtectedName(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)

	update := service.IntegrationUpdate{
		Display: strPtr("Renamed"),
	}
	err := env.Service.UpdateIntegration(service.InternalIntegrationName, &update)
	if !errors.Is(err, service.ErrIntegrationProtected) {
		t.Fatalf("expected ErrIntegrationProtected, got %v", err)
	}
}

func TestUpdateIntegration_RestoreHomepageLogo(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)
	env.CreateTestIntegration(
		t,
		"svc-a",
		"Service A",
		"svc-a.test",
		"https://svc-a.test/callback",
		"https://svc-a.test",
		"https://svc-a.test/logo.png",
		nil,
	)

	update := service.IntegrationUpdate{
		Homepage: strPtr("https://svc-a.test/v2"),
		Logo:     strPtr("https://svc-a.test/logo-v2.png"),
	}
	err := env.Service.UpdateIntegration("svc-a", &update)
	if err != nil {
		t.Fatalf("UpdateIntegration failed: %v", err)
	}

	integration, err := env.Service.GetIntegration("svc-a")
	if err != nil {
		t.Fatalf("GetIntegration failed: %v", err)
	}
	if integration.Homepage != "https://svc-a.test/v2" {
		t.Errorf("Homepage = %s, want https://svc-a.test/v2", integration.Homepage)
	}
	if integration.Logo != "https://svc-a.test/logo-v2.png" {
		t.Errorf("Logo = %s, want https://svc-a.test/logo-v2.png", integration.Logo)
	}
}

func TestUpdateIntegration_DuplicateAudience(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)
	env.CreateTestIntegration(t, "svc-a", "Service A", "svc-a.test", "https://svc-a.test/callback", "https://svc-a.test", "https://svc-a.test/logo.png", nil)
	env.CreateTestIntegration(t, "svc-b", "Service B", "svc-b.test", "https://svc-b.test/callback", "https://svc-b.test", "https://svc-b.test/logo.png", nil)

	audience := "svc-a.test"
	redirect := "https://svc-a.test/callback-b"
	homepage := "https://svc-a.test"
	err := env.Service.UpdateIntegration("svc-b", &service.IntegrationUpdate{Audience: &audience, Redirect: &redirect, Homepage: &homepage})
	if !errors.Is(err, service.ErrIntegrationExists) {
		t.Fatalf("expected ErrIntegrationExists, got %v", err)
	}
}

func TestDeleteIntegration_Success(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)
	env.CreateTestIntegration(
		t,
		"svc-a",
		"Service A",
		"svc-a.test",
		"https://svc-a.test/callback",
		"https://svc-a.test",
		"https://svc-a.test/logo.png",
		nil,
	)

	err := env.Service.DeleteIntegration("svc-a")
	if err != nil {
		t.Fatalf("DeleteIntegration failed: %v", err)
	}

	_, err = env.Service.GetIntegration("svc-a")
	if !errors.Is(err, service.ErrIntegrationNotFound) {
		t.Fatalf("expected ErrIntegrationNotFound, got %v", err)
	}
}

func TestDeleteIntegration_NotFound(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)

	err := env.Service.DeleteIntegration("missing")
	if !errors.Is(err, service.ErrIntegrationNotFound) {
		t.Fatalf("expected ErrIntegrationNotFound, got %v", err)
	}
}

func TestDeleteIntegration_ProtectedName(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)

	err := env.Service.DeleteIntegration(service.InternalIntegrationName)
	if !errors.Is(err, service.ErrIntegrationProtected) {
		t.Fatalf("expected ErrIntegrationProtected, got %v", err)
	}
}

func TestListIntegrations_Empty(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)

	integrations, err := env.Service.ListIntegrations()
	if err != nil {
		t.Fatalf("ListIntegrations failed: %v", err)
	}
	if len(integrations) != 1 {
		t.Fatalf("expected 1 integration, got %d", len(integrations))
	}
	if integrations[0].Name != service.InternalIntegrationName {
		t.Fatalf("expected internal integration first, got %s", integrations[0].Name)
	}
	if integrations[0].Homepage != "https://consent.test" {
		t.Errorf("internal integration Homepage = %s, want https://consent.test", integrations[0].Homepage)
	}
}

func TestListIntegrations_Multiple(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)
	env.CreateTestIntegration(
		t,
		"svc-a",
		"Service A",
		"svc-a.test",
		"https://svc-a.test/callback",
		"https://svc-a.test",
		"https://svc-a.test/logo.png",
		nil,
	)
	env.CreateTestIntegration(
		t,
		"svc-b",
		"Service B",
		"svc-b.test",
		"https://svc-b.test/callback",
		"https://svc-b.test",
		"https://svc-b.test/logo.png",
		nil,
	)

	integrations, err := env.Service.ListIntegrations()
	if err != nil {
		t.Fatalf("ListIntegrations failed: %v", err)
	}
	if len(integrations) != 3 {
		t.Fatalf("expected 3 integrations, got %d", len(integrations))
	}
	if integrations[0].Name != service.InternalIntegrationName {
		t.Errorf("expected internal integration first, got %s", integrations[0].Name)
	}
	if integrations[1].Name != "svc-a" {
		t.Errorf("expected svc-a second, got %s", integrations[1].Name)
	}
	if integrations[1].Homepage != "https://svc-a.test" {
		t.Errorf("svc-a Homepage = %s, want https://svc-a.test", integrations[1].Homepage)
	}
	if integrations[1].Logo != "https://svc-a.test/logo.png" {
		t.Errorf("svc-a Logo = %s, want https://svc-a.test/logo.png", integrations[1].Logo)
	}
}

func strPtr(s string) *string {
	return &s
}

func TestIntegrationsAccessibleTo_AllRoles(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)
	env.CreateTestRole(t, "editor", "Editor")
	env.CreateTestRole(t, "superadmin", "Admin")

	// Integration with no required roles should be accessible
	err := env.Service.CreateIntegration(
		"open-app",
		"Open App",
		"open.test",
		"https://open.test/callback",
		"https://open.test",
		"https://open.test/logo.png",
		[]string{},
	)
	if err != nil {
		t.Fatalf("CreateIntegration failed: %v", err)
	}

	// Integration requiring "editor" role
	err = env.Service.CreateIntegration(
		"editor-app",
		"Editor App",
		"editor.test",
		"https://editor.test/callback",
		"https://editor.test",
		"https://editor.test/logo.png",
		[]string{"editor"},
	)
	if err != nil {
		t.Fatalf("CreateIntegration failed: %v", err)
	}

	// Integration requiring "superadmin" role
	err = env.Service.CreateIntegration(
		"admin-app",
		"Admin App",
		"admin.test",
		"https://admin.test/callback",
		"https://admin.test",
		"https://admin.test/logo.png",
		[]string{"superadmin"},
	)
	if err != nil {
		t.Fatalf("CreateIntegration failed: %v", err)
	}

	// Create user with "editor" role
	editorUser, err := env.Service.CreateUser("editor-user", "password", []string{"editor"})
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	accessible, err := env.Service.IntegrationsAccessibleTo(editorUser.Subject)
	if err != nil {
		t.Fatalf("IntegrationsAccessibleTo failed: %v", err)
	}

	names := make([]string, 0, len(accessible))
	for _, a := range accessible {
		names = append(names, a.Name)
	}

	if len(accessible) != 3 {
		t.Fatalf("expected 3 accessible integrations, got %d: %v", len(accessible), names)
	}

	// Internal integration should always be included
	found := false
	for _, name := range names {
		if name == service.InternalIntegrationName {
			found = true
			break
		}
	}
	if !found {
		t.Error("internal integration should always be accessible")
	}

	// Open app (no required roles) should be accessible
	found = false
	for _, name := range names {
		if name == "open-app" {
			found = true
			break
		}
	}
	if !found {
		t.Error("open-app should be accessible (no required roles)")
	}

	// Editor app should be accessible to editor user
	found = false
	for _, name := range names {
		if name == "editor-app" {
			found = true
			break
		}
	}
	if !found {
		t.Error("editor-app should be accessible to editor user")
	}

	// Admin app should NOT be accessible to editor user
	found = false
	for _, name := range names {
		if name == "admin-app" {
			found = true
			break
		}
	}
	if found {
		t.Error("admin-app should NOT be accessible to editor user")
	}
}

func TestIntegrationsAccessibleTo_MultipleRoles(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)
	env.CreateTestRole(t, "editor", "Editor")
	env.CreateTestRole(t, "superadmin", "Admin")

	// Integration requiring either "editor" or "superadmin" role
	err := env.Service.CreateIntegration(
		"multi-app",
		"Multi Role App",
		"multi.test",
		"https://multi.test/callback",
		"https://multi.test",
		"https://multi.test/logo.png",
		[]string{"editor", "superadmin"},
	)
	if err != nil {
		t.Fatalf("CreateIntegration failed: %v", err)
	}

	// Create user with only "superadmin" role
	adminUser, err := env.Service.CreateUser("admin-user", "password", []string{"superadmin"})
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	accessible, err := env.Service.IntegrationsAccessibleTo(adminUser.Subject)
	if err != nil {
		t.Fatalf("IntegrationsAccessibleTo failed: %v", err)
	}

	// Should find internal integration and multi-app (user has "superadmin" which is in the list)
	if len(accessible) != 2 {
		t.Fatalf("expected 2 accessible integrations, got %d", len(accessible))
	}

	found := false
	for _, a := range accessible {
		if a.Name == "multi-app" {
			found = true
		}
	}
	if !found {
		t.Error("multi-app should be accessible (user has 'admin' which is one of required roles)")
	}
}

func TestIntegrationsAccessibleTo_NoMatchingRole(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)
	env.CreateTestRole(t, "editor", "Editor")
	env.CreateTestRole(t, "superadmin", "Admin")

	// Integration requiring "superadmin" role
	err := env.Service.CreateIntegration(
		"admin-app",
		"Admin App",
		"admin.test",
		"https://admin.test/callback",
		"https://admin.test",
		"https://admin.test/logo.png",
		[]string{"superadmin"},
	)
	if err != nil {
		t.Fatalf("CreateIntegration failed: %v", err)
	}

	// Create user with "editor" role only
	editorUser, err := env.Service.CreateUser("editor-user", "password", []string{"editor"})
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	accessible, err := env.Service.IntegrationsAccessibleTo(editorUser.Subject)
	if err != nil {
		t.Fatalf("IntegrationsAccessibleTo failed: %v", err)
	}

	// Should only have internal integration (admin-app is filtered out)
	if len(accessible) != 1 {
		t.Fatalf("expected 1 accessible integration, got %d", len(accessible))
	}
	if accessible[0].Name != service.InternalIntegrationName {
		t.Errorf("expected internal integration, got %s", accessible[0].Name)
	}
}
