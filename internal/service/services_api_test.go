package service_test

import (
	"net/http"
	"testing"

	"git.sr.ht/~jakintosh/command-go/pkg/wire"
	"git.sr.ht/~jakintosh/consent/internal/testutil"
)

func TestAPICreateService_Success(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)

	body := `{"name":"svc-a","display":"Service A","audience":"aud-a","redirect":"https://svc-a.test/callback"}`
	result := wire.TestPost[any](env.Router, "/services", body, jsonHeader)
	result.ExpectStatus(t, http.StatusOK)

	svc, err := env.Service.GetServiceByName("svc-a")
	if err != nil {
		t.Fatalf("GetServiceByName failed: %v", err)
	}
	if svc.Display != "Service A" {
		t.Errorf("Display = %s, want Service A", svc.Display)
	}
}

func TestAPICreateService_DuplicateName(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)
	env.CreateTestService(t, "svc-a", "Service A", "aud-a", "https://svc-a.test/callback")

	body := `{"name":"svc-a","display":"Service A","audience":"aud-a","redirect":"https://svc-a.test/callback"}`
	result := wire.TestPost[any](env.Router, "/services", body, jsonHeader)
	result.ExpectStatus(t, http.StatusConflict)
	result.ExpectError(t)
}

func TestAPICreateService_InvalidRedirect(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)

	body := `{"name":"svc-a","display":"Service A","audience":"aud-a","redirect":"not-a-url"}`
	result := wire.TestPost[any](env.Router, "/services", body, jsonHeader)
	result.ExpectStatus(t, http.StatusBadRequest)
	result.ExpectError(t)
}

func TestAPICreateService_MissingName(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)

	body := `{"display":"Service A","audience":"aud-a","redirect":"https://svc-a.test/callback"}`
	result := wire.TestPost[any](env.Router, "/services", body, jsonHeader)
	result.ExpectStatus(t, http.StatusBadRequest)
	result.ExpectError(t)
}

func TestAPICreateService_MissingDisplay(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)

	body := `{"name":"svc-a","audience":"aud-a","redirect":"https://svc-a.test/callback"}`
	result := wire.TestPost[any](env.Router, "/services", body, jsonHeader)
	result.ExpectStatus(t, http.StatusBadRequest)
	result.ExpectError(t)
}

func TestAPICreateService_MissingAudience(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)

	body := `{"name":"svc-a","display":"Service A","redirect":"https://svc-a.test/callback"}`
	result := wire.TestPost[any](env.Router, "/services", body, jsonHeader)
	result.ExpectStatus(t, http.StatusBadRequest)
	result.ExpectError(t)
}

func TestAPICreateService_MissingRedirect(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)

	body := `{"name":"svc-a","display":"Service A","audience":"aud-a"}`
	result := wire.TestPost[any](env.Router, "/services", body, jsonHeader)
	result.ExpectStatus(t, http.StatusBadRequest)
	result.ExpectError(t)
}

func TestAPIGetService_Success(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)
	env.CreateTestService(t, "svc-a", "Service A", "aud-a", "https://svc-a.test/callback")

	result := wire.TestGet[map[string]string](env.Router, "/services/svc-a")
	response := result.ExpectOK(t)
	if response["name"] != "svc-a" {
		t.Errorf("name = %s, want svc-a", response["name"])
	}
	if response["display"] != "Service A" {
		t.Errorf("display = %s, want Service A", response["display"])
	}
}

func TestAPIGetService_NotFound(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)

	result := wire.TestGet[any](env.Router, "/services/missing")
	result.ExpectStatus(t, http.StatusBadRequest)
	result.ExpectError(t)
}

func TestAPIUpdateService_Success(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)
	env.CreateTestService(t, "svc-a", "Service A", "aud-a", "https://svc-a.test/callback")

	body := `{"display":"Service A2","audience":"aud-b","redirect":"https://svc-a.test/new"}`
	result := wire.TestPut[any](env.Router, "/services/svc-a", body, jsonHeader)
	result.ExpectStatus(t, http.StatusOK)

	svc, err := env.Service.GetServiceByName("svc-a")
	if err != nil {
		t.Fatalf("GetServiceByName failed: %v", err)
	}
	if svc.Display != "Service A2" {
		t.Errorf("Display = %s, want Service A2", svc.Display)
	}
}

func TestAPIUpdateService_NotFound(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)

	body := `{"display":"Service A2","audience":"aud-b","redirect":"https://svc-a.test/new"}`
	result := wire.TestPut[any](env.Router, "/services/missing", body, jsonHeader)
	result.ExpectStatus(t, http.StatusBadRequest)
	result.ExpectError(t)
}

func TestAPIUpdateService_InvalidRedirect(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)
	env.CreateTestService(t, "svc-a", "Service A", "aud-a", "https://svc-a.test/callback")

	body := `{"display":"Service A2","audience":"aud-b","redirect":"bad-url"}`
	result := wire.TestPut[any](env.Router, "/services/svc-a", body, jsonHeader)
	result.ExpectStatus(t, http.StatusBadRequest)
	result.ExpectError(t)
}

func TestAPIDeleteService_Success(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)
	env.CreateTestService(t, "svc-a", "Service A", "aud-a", "https://svc-a.test/callback")

	result := wire.TestDelete[any](env.Router, "/services/svc-a")
	result.ExpectStatus(t, http.StatusOK)

	_, err := env.Service.GetServiceByName("svc-a")
	if err == nil {
		t.Fatal("expected error after delete")
	}
}

func TestAPIDeleteService_NotFound(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)

	result := wire.TestDelete[any](env.Router, "/services/missing")
	result.ExpectStatus(t, http.StatusBadRequest)
	result.ExpectError(t)
}

func TestAPIListServices_Seeded(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)

	result := wire.TestGet[[]map[string]string](env.Router, "/services")
	response := result.ExpectOK(t)
	if len(response) != 1 {
		t.Fatalf("expected 1 service (seeded), got %d", len(response))
	}
	if response[0]["name"] != "test-service" {
		t.Fatalf("expected seeded test-service, got %s", response[0]["name"])
	}
}

func TestAPIListServices_Multiple(t *testing.T) {
	t.Parallel()
	env := testutil.SetupTestEnvWithRouter(t)
	env.CreateTestService(t, "svc-a", "Service A", "aud-a", "https://svc-a.test/callback")
	env.CreateTestService(t, "svc-b", "Service B", "aud-b", "https://svc-b.test/callback")

	result := wire.TestGet[[]map[string]string](env.Router, "/services")
	response := result.ExpectOK(t)
	if len(response) != 3 {
		t.Fatalf("expected 3 services, got %d", len(response))
	}
	if response[0]["name"] != "svc-a" {
		t.Errorf("expected svc-a first, got %s", response[0]["name"])
	}
}
