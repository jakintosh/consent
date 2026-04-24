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
		t.Fatal("expected non-nil service instance")
	}
}

func TestService_DefaultIntegration(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)

	integrations, err := env.Service.ListIntegrations()
	if err != nil {
		t.Fatalf("ListIntegrations failed: %v", err)
	}
	if len(integrations) != 1 {
		t.Fatalf("expected internal integration only, got %d", len(integrations))
	}
	if integrations[0].Name != service.InternalIntegrationName {
		t.Fatalf("expected %s, got %s", service.InternalIntegrationName, integrations[0].Name)
	}
}

func TestBuildInternalIntegration(t *testing.T) {
	t.Parallel()

	integration, err := service.BuildInternalIntegration("https://consent.test/base/")
	if err != nil {
		t.Fatalf("BuildInternalIntegration failed: %v", err)
	}
	if integration.Name != service.InternalIntegrationName {
		t.Fatalf("Name = %s, want %s", integration.Name, service.InternalIntegrationName)
	}
	if integration.Display != "Consent" {
		t.Fatalf("Display = %s, want Consent", integration.Display)
	}
	if integration.Audience != "consent.test" {
		t.Fatalf("Audience = %s, want consent.test", integration.Audience)
	}
	if integration.Redirect != "https://consent.test/base/auth/callback" {
		t.Fatalf("Redirect = %s, want https://consent.test/base/auth/callback", integration.Redirect)
	}
}

func TestBuildInternalIntegration_InvalidURL(t *testing.T) {
	t.Parallel()

	_, err := service.BuildInternalIntegration("not-a-url")
	if !errors.Is(err, service.ErrInvalidUrl) {
		t.Fatalf("expected ErrInvalidUrl, got %v", err)
	}
}
