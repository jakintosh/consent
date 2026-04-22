package service_test

import (
	"errors"
	"testing"

	"git.sr.ht/~jakintosh/consent/internal/service"
	"git.sr.ht/~jakintosh/consent/internal/testutil"
)

func TestCreateService_Success(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)

	err := env.Service.CreateService("svc-a", "Service A", "aud-a", "https://svc-a.test/callback")
	if err != nil {
		t.Fatalf("CreateService failed: %v", err)
	}
}

func TestCreateService_DuplicateName(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)

	if err := env.Service.CreateService("svc-a", "Service A", "aud-a", "https://svc-a.test/callback"); err != nil {
		t.Fatalf("CreateService failed: %v", err)
	}

	err := env.Service.CreateService("svc-a", "Service A", "aud-a", "https://svc-a.test/callback")
	if !errors.Is(err, service.ErrServiceExists) {
		t.Fatalf("expected ErrServiceExists, got %v", err)
	}
}

func TestCreateService_InvalidRedirect(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)

	err := env.Service.CreateService("svc-a", "Service A", "aud-a", "bad-url")
	if !errors.Is(err, service.ErrInvalidRedirect) {
		t.Fatalf("expected ErrInvalidRedirect, got %v", err)
	}
}

func TestCreateService_InvalidName(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)

	err := env.Service.CreateService("", "Service A", "aud-a", "https://svc-a.test/callback")
	if !errors.Is(err, service.ErrInvalidService) {
		t.Fatalf("expected ErrInvalidService, got %v", err)
	}
}

func TestCreateService_ProtectedName(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)

	err := env.Service.CreateService(
		service.InternalServiceName,
		"Consent",
		"consent.test",
		"https://consent.test/auth/callback",
	)
	if !errors.Is(err, service.ErrServiceProtected) {
		t.Fatalf("expected ErrServiceProtected, got %v", err)
	}
}

func TestGetServiceByName_Success(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)
	env.CreateTestService(t, "svc-a", "Service A", "aud-a", "https://svc-a.test/callback")

	serviceDef, err := env.Service.GetServiceByName("svc-a")
	if err != nil {
		t.Fatalf("GetServiceByName failed: %v", err)
	}
	if serviceDef.Name != "svc-a" {
		t.Errorf("Name = %s, want svc-a", serviceDef.Name)
	}
	if serviceDef.Redirect != "https://svc-a.test/callback" {
		t.Errorf("Redirect = %s, want https://svc-a.test/callback", serviceDef.Redirect)
	}
}

func TestGetServiceByName_NotFound(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)

	_, err := env.Service.GetServiceByName("missing")
	if !errors.Is(err, service.ErrServiceNotFound) {
		t.Fatalf("expected ErrServiceNotFound, got %v", err)
	}
}

func TestUpdateService_Success(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)
	env.CreateTestService(t, "svc-a", "Service A", "aud-a", "https://svc-a.test/callback")

	display := "Service A2"
	audience := "aud-b"
	redirect := "https://svc-a.test/new"
	err := env.Service.UpdateService("svc-a", &display, &audience, &redirect)
	if err != nil {
		t.Fatalf("UpdateService failed: %v", err)
	}

	serviceDef, err := env.Service.GetServiceByName("svc-a")
	if err != nil {
		t.Fatalf("GetServiceByName failed: %v", err)
	}
	if serviceDef.Display != "Service A2" {
		t.Errorf("Display = %s, want Service A2", serviceDef.Display)
	}
}

func TestUpdateService_NotFound(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)

	display := "Service A2"
	err := env.Service.UpdateService("missing", &display, nil, nil)
	if !errors.Is(err, service.ErrServiceNotFound) {
		t.Fatalf("expected ErrServiceNotFound, got %v", err)
	}
}

func TestUpdateService_InvalidRedirect(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)
	env.CreateTestService(t, "svc-a", "Service A", "aud-a", "https://svc-a.test/callback")

	redirect := "bad-url"
	err := env.Service.UpdateService("svc-a", nil, nil, &redirect)
	if !errors.Is(err, service.ErrInvalidRedirect) {
		t.Fatalf("expected ErrInvalidRedirect, got %v", err)
	}
}

func TestUpdateService_ProtectedName(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)

	display := "Renamed"
	err := env.Service.UpdateService(service.InternalServiceName, &display, nil, nil)
	if !errors.Is(err, service.ErrServiceProtected) {
		t.Fatalf("expected ErrServiceProtected, got %v", err)
	}
}

func TestDeleteService_Success(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)
	env.CreateTestService(t, "svc-a", "Service A", "aud-a", "https://svc-a.test/callback")

	err := env.Service.DeleteService("svc-a")
	if err != nil {
		t.Fatalf("DeleteService failed: %v", err)
	}

	_, err = env.Service.GetServiceByName("svc-a")
	if !errors.Is(err, service.ErrServiceNotFound) {
		t.Fatalf("expected ErrServiceNotFound, got %v", err)
	}
}

func TestDeleteService_NotFound(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)

	err := env.Service.DeleteService("missing")
	if !errors.Is(err, service.ErrServiceNotFound) {
		t.Fatalf("expected ErrServiceNotFound, got %v", err)
	}
}

func TestDeleteService_ProtectedName(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)

	err := env.Service.DeleteService(service.InternalServiceName)
	if !errors.Is(err, service.ErrServiceProtected) {
		t.Fatalf("expected ErrServiceProtected, got %v", err)
	}
}

func TestListServices_Empty(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)

	services, err := env.Service.ListServices()
	if err != nil {
		t.Fatalf("ListServices failed: %v", err)
	}
	if len(services) != 1 {
		t.Fatalf("expected 1 service, got %d", len(services))
	}
	if services[0].Name != service.InternalServiceName {
		t.Fatalf("expected internal service first, got %s", services[0].Name)
	}
}

func TestListServices_Multiple(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)
	env.CreateTestService(t, "svc-a", "Service A", "aud-a", "https://svc-a.test/callback")
	env.CreateTestService(t, "svc-b", "Service B", "aud-b", "https://svc-b.test/callback")

	services, err := env.Service.ListServices()
	if err != nil {
		t.Fatalf("ListServices failed: %v", err)
	}
	if len(services) != 3 {
		t.Fatalf("expected 3 services, got %d", len(services))
	}
	if services[0].Name != service.InternalServiceName {
		t.Errorf("expected internal service first, got %s", services[0].Name)
	}
	if services[1].Name != "svc-a" {
		t.Errorf("expected svc-a second, got %s", services[1].Name)
	}
}
