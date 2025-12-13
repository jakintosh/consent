package consenttest

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewKeys(t *testing.T) {
	keys, err := NewKeys("test.example.com")
	if err != nil {
		t.Fatalf("NewKeys failed: %v", err)
	}

	if keys.SigningKey == nil {
		t.Error("SigningKey is nil")
	}

	if keys.VerificationKey == nil {
		t.Error("VerificationKey is nil")
	}

	if keys.IssuerDomain != "test.example.com" {
		t.Errorf("expected IssuerDomain 'test.example.com', got %s", keys.IssuerDomain)
	}
}

func TestNewSession(t *testing.T) {
	keys, err := NewKeys("test.example.com")
	if err != nil {
		t.Fatalf("NewKeys failed: %v", err)
	}

	session, err := NewSession(
		keys,
		"testuser",
		"test-audience",
		30*time.Minute,
		72*time.Hour,
	)
	if err != nil {
		t.Fatalf("NewSession failed: %v", err)
	}

	if session.AccessToken == "" {
		t.Error("AccessToken is empty")
	}

	if session.RefreshToken == "" {
		t.Error("RefreshToken is empty")
	}

	if session.CSRF == "" {
		t.Error("CSRF is empty")
	}

	if session.AccessExpiresAt.IsZero() {
		t.Error("AccessExpiresAt is zero")
	}

	if session.RefreshExpiresAt.IsZero() {
		t.Error("RefreshExpiresAt is zero")
	}
}

func TestCookies(t *testing.T) {
	keys, _ := NewKeys("test.example.com")
	session, _ := NewSession(keys, "testuser", "test-audience", 30*time.Minute, 72*time.Hour)

	opts := CookieOptions{
		Secure:   false,
		SameSite: http.SameSiteStrictMode,
		Path:     "/",
	}

	access, refresh := Cookies(session, opts)

	if access.Name != "accessToken" {
		t.Errorf("expected access cookie name 'accessToken', got %s", access.Name)
	}

	if access.Value != session.AccessToken {
		t.Error("access cookie value doesn't match session token")
	}

	if access.Secure != false {
		t.Error("expected access cookie Secure=false")
	}

	if refresh.Name != "refreshToken" {
		t.Errorf("expected refresh cookie name 'refreshToken', got %s", refresh.Name)
	}

	if refresh.Value != session.RefreshToken {
		t.Error("refresh cookie value doesn't match session token")
	}
}

func TestAddCookies(t *testing.T) {
	keys, _ := NewKeys("test.example.com")
	session, _ := NewSession(keys, "testuser", "test-audience", 30*time.Minute, 72*time.Hour)

	req := httptest.NewRequest("GET", "/test", nil)
	AddCookies(req, session, CookieOptions{
		Secure:   false,
		SameSite: http.SameSiteStrictMode,
	})

	cookies := req.Cookies()
	if len(cookies) != 2 {
		t.Errorf("expected 2 cookies, got %d", len(cookies))
	}

	var foundAccess, foundRefresh bool
	for _, cookie := range cookies {
		if cookie.Name == "accessToken" {
			foundAccess = true
		}
		if cookie.Name == "refreshToken" {
			foundRefresh = true
		}
	}

	if !foundAccess {
		t.Error("accessToken cookie not found")
	}

	if !foundRefresh {
		t.Error("refreshToken cookie not found")
	}
}

func TestValidator(t *testing.T) {
	keys, _ := NewKeys("test.example.com")

	validator := Validator(keys, "test-audience")
	if validator == nil {
		t.Fatal("Validator returned nil")
	}

	// Verify the validator works correctly
	if !validator.ValidateDomain("test.example.com") {
		t.Error("validator should accept correct domain")
	}

	if validator.ValidateDomain("wrong.example.com") {
		t.Error("validator should reject wrong domain")
	}

	if !validator.ShouldValidateAudience() {
		t.Error("client validator should validate audience")
	}
}
