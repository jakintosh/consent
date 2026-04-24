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

	err := env.Service.CreateIntegration("svc-a", "Service A", "aud-a", "https://svc-a.test/callback")
	if err != nil {
		t.Fatalf("CreateIntegration failed: %v", err)
	}
}

func TestCreateIntegration_DuplicateName(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)

	if err := env.Service.CreateIntegration("svc-a", "Service A", "aud-a", "https://svc-a.test/callback"); err != nil {
		t.Fatalf("CreateIntegration failed: %v", err)
	}

	err := env.Service.CreateIntegration("svc-a", "Service A", "aud-a", "https://svc-a.test/callback")
	if !errors.Is(err, service.ErrIntegrationExists) {
		t.Fatalf("expected ErrIntegrationExists, got %v", err)
	}
}

func TestCreateIntegration_InvalidRedirect(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)

	err := env.Service.CreateIntegration("svc-a", "Service A", "aud-a", "bad-url")
	if !errors.Is(err, service.ErrInvalidRedirect) {
		t.Fatalf("expected ErrInvalidRedirect, got %v", err)
	}
}

func TestCreateIntegration_InvalidName(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)

	err := env.Service.CreateIntegration("", "Service A", "aud-a", "https://svc-a.test/callback")
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
	)
	if !errors.Is(err, service.ErrIntegrationProtected) {
		t.Fatalf("expected ErrIntegrationProtected, got %v", err)
	}
}

func TestGetIntegration_Success(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)
	env.CreateTestIntegration(t, "svc-a", "Service A", "aud-a", "https://svc-a.test/callback")

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
}

func TestGetIntegration_NotFound(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)

	_, err := env.Service.GetIntegration("missing")
	if !errors.Is(err, service.ErrIntegrationNotFound) {
		t.Fatalf("expected ErrIntegrationNotFound, got %v", err)
	}
}

func TestUpdateIntegration_Success(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)
	env.CreateTestIntegration(t, "svc-a", "Service A", "aud-a", "https://svc-a.test/callback")

	display := "Service A2"
	audience := "aud-b"
	redirect := "https://svc-a.test/new"
	err := env.Service.UpdateIntegration("svc-a", &service.IntegrationUpdate{Display: &display, Audience: &audience, Redirect: &redirect})
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
}

func TestUpdateIntegration_NotFound(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)

	display := "Service A2"
	err := env.Service.UpdateIntegration("missing", &service.IntegrationUpdate{Display: &display})
	if !errors.Is(err, service.ErrIntegrationNotFound) {
		t.Fatalf("expected ErrIntegrationNotFound, got %v", err)
	}
}

func TestUpdateIntegration_InvalidRedirect(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)
	env.CreateTestIntegration(t, "svc-a", "Service A", "aud-a", "https://svc-a.test/callback")

	redirect := "bad-url"
	err := env.Service.UpdateIntegration("svc-a", &service.IntegrationUpdate{Redirect: &redirect})
	if !errors.Is(err, service.ErrInvalidRedirect) {
		t.Fatalf("expected ErrInvalidRedirect, got %v", err)
	}
}

func TestUpdateIntegration_ProtectedName(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)

	display := "Renamed"
	err := env.Service.UpdateIntegration(service.InternalIntegrationName, &service.IntegrationUpdate{Display: &display})
	if !errors.Is(err, service.ErrIntegrationProtected) {
		t.Fatalf("expected ErrIntegrationProtected, got %v", err)
	}
}

func TestDeleteIntegration_Success(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)
	env.CreateTestIntegration(t, "svc-a", "Service A", "aud-a", "https://svc-a.test/callback")

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
}

func TestListIntegrations_Multiple(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)
	env.CreateTestIntegration(t, "svc-a", "Service A", "aud-a", "https://svc-a.test/callback")
	env.CreateTestIntegration(t, "svc-b", "Service B", "aud-b", "https://svc-b.test/callback")

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
}
