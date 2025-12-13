package testharness

import (
	"crypto/ecdsa"
	"crypto/x509"
	"net/http"
	"net/http/httptest"
	"testing"

	"git.sr.ht/~jakintosh/consent/pkg/client"
	"git.sr.ht/~jakintosh/consent/pkg/tokens"
)

func TestStart(t *testing.T) {
	// Create a test app server
	appServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer appServer.Close()

	// Start the harness
	h := Start(t, Config{
		RedirectURL:     appServer.URL + "/callback",
		ServiceName:     "test-app",
		ServiceAudience: "test-audience",
		ServiceDisplay:  "Test Application",
		IssuerDomain:    "test.local",
		Users: []User{
			{Handle: "alice", Password: "password123"},
			{Handle: "bob", Password: "secret456"},
		},
		Quiet: true,
	})

	// Verify harness fields are populated
	if h.BaseURL == "" {
		t.Error("BaseURL is empty")
	}

	if h.IssuerDomain != "test.local" {
		t.Errorf("expected IssuerDomain 'test.local', got %s", h.IssuerDomain)
	}

	if h.ServiceName != "test-app" {
		t.Errorf("expected ServiceName 'test-app', got %s", h.ServiceName)
	}

	if h.ServiceAudience != "test-audience" {
		t.Errorf("expected ServiceAudience 'test-audience', got %s", h.ServiceAudience)
	}

	if len(h.Users) != 2 {
		t.Errorf("expected 2 users, got %d", len(h.Users))
	}

	if h.Users[0].Handle != "alice" || h.Users[0].Password != "password123" {
		t.Error("first user credentials don't match")
	}

	if h.Users[1].Handle != "bob" || h.Users[1].Password != "secret456" {
		t.Error("second user credentials don't match")
	}

	if h.VerificationKeyPath == "" {
		t.Error("VerificationKeyPath is empty")
	}

	if len(h.VerificationKeyDER) == 0 {
		t.Error("VerificationKeyDER is empty")
	}

	// Verify the verification key can be parsed
	pubKey, err := x509.ParsePKIXPublicKey(h.VerificationKeyDER)
	if err != nil {
		t.Errorf("failed to parse verification key: %v", err)
	}

	ecdsaKey, ok := pubKey.(*ecdsa.PublicKey)
	if !ok {
		t.Error("verification key is not an ECDSA public key")
	}

	// Verify we can create a validator with it
	validator := tokens.InitClient(ecdsaKey, h.IssuerDomain, h.ServiceAudience)
	if validator == nil {
		t.Error("failed to create validator")
	}

	// Verify the server is actually running
	resp, err := http.Get(h.BaseURL + "/")
	if err != nil {
		t.Errorf("failed to connect to test server: %v", err)
	} else {
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status 200, got %d", resp.StatusCode)
		}
	}

	// Note: t.Cleanup will automatically call h.Close() when the test finishes
}

func TestStartWithDefaults(t *testing.T) {
	// Start with minimal config
	appServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer appServer.Close()

	h := Start(t, Config{
		RedirectURL: appServer.URL + "/callback",
		Quiet:       true,
	})

	// Verify defaults
	if h.ServiceName != "test-service" {
		t.Errorf("expected default ServiceName 'test-service', got %s", h.ServiceName)
	}

	if h.ServiceAudience != "test-audience" {
		t.Errorf("expected default ServiceAudience 'test-audience', got %s", h.ServiceAudience)
	}

	if h.IssuerDomain != "consent.test" {
		t.Errorf("expected default IssuerDomain 'consent.test', got %s", h.IssuerDomain)
	}

	if len(h.Users) != 1 {
		t.Errorf("expected 1 default user, got %d", len(h.Users))
	}

	if h.Users[0].Handle != "test" || h.Users[0].Password != "test" {
		t.Error("default user credentials don't match 'test:test'")
	}
}

func TestIntegrationWithClient(t *testing.T) {
	// This test verifies the full integration between testharness, consent-testserver,
	// and the client library

	// Create a simple test app
	appServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/callback":
			client.HandleAuthorizationCode(w, r)
		default:
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer appServer.Close()

	// Start consent-testserver
	h := Start(t, Config{
		RedirectURL:     appServer.URL + "/callback",
		ServiceAudience: "integration-test",
		Quiet:           true,
	})

	// Configure the client library
	pubKey, err := x509.ParsePKIXPublicKey(h.VerificationKeyDER)
	if err != nil {
		t.Fatalf("failed to parse verification key: %v", err)
	}

	validator := tokens.InitClient(
		pubKey.(*ecdsa.PublicKey),
		h.IssuerDomain,
		h.ServiceAudience,
	)

	client.Init(validator, h.BaseURL)
	client.SetLogLevel(client.LogLevelNone)
	client.SetCookieOptions(client.CookieOptions{
		Secure:   false, // Required for HTTP testing
		SameSite: http.SameSiteStrictMode,
		Path:     "/",
	})

	// Verify the consent server is accessible
	resp, err := http.Get(h.BaseURL + "/login?service=" + h.ServiceName)
	if err != nil {
		t.Fatalf("failed to access consent server: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200 from login page, got %d", resp.StatusCode)
	}

	// Note: A full OAuth flow test would require simulating form submission
	// and following redirects, which is beyond the scope of this basic test.
	// The important thing is that the harness starts correctly and the
	// client can be configured to talk to it.
}
