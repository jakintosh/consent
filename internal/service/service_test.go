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

func TestService_Catalog(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnv(t)

	catalog := env.Service.Catalog()
	if catalog == nil {
		t.Fatal("expected non-nil catalog")
	}
}
