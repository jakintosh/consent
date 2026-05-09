package client

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandleIntegrationManifest(t *testing.T) {
	manifest := IntegrationManifest{
		Name:     "test-app",
		Display:  "Test App",
		Audience: "app.test",
		Redirect: "https://app.test/auth/callback",
		Homepage: "https://app.test",
		Logo:     "https://app.test/logo.png",
	}

	req := httptest.NewRequest(http.MethodGet, IntegrationManifestPath, nil)
	rr := httptest.NewRecorder()

	HandleIntegrationManifest(manifest).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}

	var got IntegrationManifest
	if err := json.NewDecoder(rr.Body).Decode(&got); err != nil {
		t.Fatalf("Decode failed: %v", err)
	}
	if got.ManifestVersion != IntegrationManifestVersion {
		t.Fatalf("ManifestVersion = %d, want %d", got.ManifestVersion, IntegrationManifestVersion)
	}
	if got.Name != manifest.Name {
		t.Fatalf("Name = %q, want %q", got.Name, manifest.Name)
	}
}

func TestHandleIntegrationManifest_MethodNotAllowed(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, IntegrationManifestPath, nil)
	rr := httptest.NewRecorder()

	HandleIntegrationManifest(IntegrationManifest{}).ServeHTTP(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusMethodNotAllowed)
	}
}
