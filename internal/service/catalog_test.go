package service_test

import (
	"testing"

	"git.sr.ht/~jakintosh/consent/internal/testutil"
)

func TestServiceCatalog_GetService_Exists(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)

	svc, err := env.Service.Catalog().GetService("test-service")
	if err != nil {
		t.Fatalf("GetService failed: %v", err)
	}

	if svc.Display != "Test Service" {
		t.Errorf("Display = %s, want Test Service", svc.Display)
	}
	if svc.Audience != "test-audience" {
		t.Errorf("Audience = %s, want test-audience", svc.Audience)
	}
	if svc.Redirect == nil {
		t.Fatal("Redirect is nil")
	}
	if svc.Redirect.Host != "localhost:8080" {
		t.Errorf("Redirect.Host = %s, want localhost:8080", svc.Redirect.Host)
	}
}

func TestServiceCatalog_GetService_NotExists(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)

	_, err := env.Service.Catalog().GetService("nonexistent-service")
	if err == nil {
		t.Error("expected error for nonexistent service")
	}
}
