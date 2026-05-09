package main

import "testing"

func TestBuildIntegrationManifest(t *testing.T) {
	cfg := Config{
		AuthURL:         "http://localhost:9000",
		AuthorityDomain: "localhost",
		Integration:     "mock1",
		Display:         "Mock Client 1",
		Audience:        "mock1.localhost:9001",
	}

	manifest := buildIntegrationManifest(cfg)

	if manifest.Name != "mock1" {
		t.Fatalf("Name = %q, want mock1", manifest.Name)
	}
	if manifest.Display != "Mock Client 1" {
		t.Fatalf("Display = %q, want Mock Client 1", manifest.Display)
	}
	if manifest.Audience != "mock1.localhost:9001" {
		t.Fatalf("Audience = %q, want mock1.localhost:9001", manifest.Audience)
	}
	if manifest.Redirect != "http://mock1.localhost:9001/auth/callback" {
		t.Fatalf("Redirect = %q", manifest.Redirect)
	}
	if manifest.Homepage != "http://mock1.localhost:9001" {
		t.Fatalf("Homepage = %q", manifest.Homepage)
	}
	if manifest.Logo != "http://localhost:9000/assets/default-integration-logo.png" {
		t.Fatalf("Logo = %q", manifest.Logo)
	}
	if manifest.ConsentIssuer != "localhost" {
		t.Fatalf("ConsentIssuer = %q, want localhost", manifest.ConsentIssuer)
	}
	if manifest.ConsentBaseURL != "http://localhost:9000" {
		t.Fatalf("ConsentBaseURL = %q, want http://localhost:9000", manifest.ConsentBaseURL)
	}
}
