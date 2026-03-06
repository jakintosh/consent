package service_test

import (
	"errors"
	"testing"

	"git.sr.ht/~jakintosh/consent/internal/service"
	"git.sr.ht/~jakintosh/consent/internal/testutil"
)

func TestNew_CreatesService(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)

	if env.Service == nil {
		t.Fatal("expected non-nil service")
	}
}

func TestService_Defaults(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)

	services, err := env.Service.ListServices()
	if err != nil {
		t.Fatalf("ListServices failed: %v", err)
	}
	if len(services) != 1 {
		t.Fatalf("expected internal service only, got %d", len(services))
	}
	if services[0].Name != service.InternalServiceName {
		t.Fatalf("expected %s, got %s", service.InternalServiceName, services[0].Name)
	}
}

func TestBuildInternalService(t *testing.T) {
	t.Parallel()

	serviceDef, err := service.BuildInternalServiceDefinition("https://consent.test/base/")
	if err != nil {
		t.Fatalf("BuildInternalService failed: %v", err)
	}
	if serviceDef.Name != service.InternalServiceName {
		t.Fatalf("Name = %s, want %s", serviceDef.Name, service.InternalServiceName)
	}
	if serviceDef.Display != "Consent" {
		t.Fatalf("Display = %s, want Consent", serviceDef.Display)
	}
	if serviceDef.Audience != "consent.test" {
		t.Fatalf("Audience = %s, want consent.test", serviceDef.Audience)
	}
	if serviceDef.Redirect != "https://consent.test/base/auth/callback" {
		t.Fatalf("Redirect = %s, want https://consent.test/base/auth/callback", serviceDef.Redirect)
	}
}

func TestBuildInternalService_InvalidURL(t *testing.T) {
	t.Parallel()

	_, err := service.BuildInternalServiceDefinition("not-a-url")
	if !errors.Is(err, service.ErrInvalidUrl) {
		t.Fatalf("expected ErrInvalidUrl, got %v", err)
	}
}
