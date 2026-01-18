package service_test

import (
	"testing"

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
	if len(services) != 0 {
		t.Fatalf("expected no services, got %d", len(services))
	}
}
