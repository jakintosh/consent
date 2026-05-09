package client

import (
	"encoding/json"
	"net/http"
)

const (
	IntegrationManifestPath    = "/.well-known/consent-integration"
	IntegrationManifestVersion = 1
)

// IntegrationManifest describes the registration metadata a Consent-backed app
// exposes for admin-side integration setup.
type IntegrationManifest struct {
	ManifestVersion int    `json:"manifest_version"`
	Name            string `json:"name"`
	Display         string `json:"display"`
	Audience        string `json:"audience"`
	Redirect        string `json:"redirect"`
	Homepage        string `json:"homepage"`
	Logo            string `json:"logo"`
	ConsentIssuer   string `json:"consent_issuer,omitempty"`
	ConsentBaseURL  string `json:"consent_base_url,omitempty"`
}

// HandleIntegrationManifest returns an HTTP handler for the well-known Consent
// integration manifest route. Mount it at IntegrationManifestPath on the app's
// root router so admins can import the integration from the app's base URL.
func HandleIntegrationManifest(
	manifest IntegrationManifest,
) http.HandlerFunc {
	if manifest.ManifestVersion == 0 {
		manifest.ManifestVersion = IntegrationManifestVersion
	}

	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(manifest)
	}
}
