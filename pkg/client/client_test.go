package client

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"git.sr.ht/~jakintosh/consent/pkg/consenttest"
)

func TestSetHTTPClient(t *testing.T) {
	// Create a custom HTTP client
	customClient := &http.Client{
		Timeout: 5 * time.Second,
	}

	// Set it
	SetHTTPClient(customClient)

	// Verify it was set (by checking the package variable)
	if httpClient != customClient {
		t.Error("SetHTTPClient did not set the HTTP client")
	}

	// Reset to default for other tests
	SetHTTPClient(http.DefaultClient)
}

func TestSetCookieOptions(t *testing.T) {
	// Save original value
	originalSecure := cookieOptions.Secure

	// Set insecure cookies
	SetCookieOptions(CookieOptions{
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
		Path:     "/custom",
	})

	if cookieOptions.Secure != false {
		t.Error("Secure was not set to false")
	}

	if cookieOptions.SameSite != http.SameSiteLaxMode {
		t.Error("SameSite was not set to Lax")
	}

	if cookieOptions.Path != "/custom" {
		t.Error("Path was not set to /custom")
	}

	// Reset to original
	cookieOptions.Secure = originalSecure
	cookieOptions.SameSite = http.SameSiteStrictMode
	cookieOptions.Path = "/"
	insecureCookieWarningEmitted = false
}

func TestSetTokenCookiesUsesOptions(t *testing.T) {
	// Create test tokens
	keys, _ := consenttest.NewKeys("test.example.com")
	session, _ := consenttest.NewSession(keys, "testuser", "test-audience", 30*time.Minute, 72*time.Hour)

	// Decode tokens
	validator := consenttest.Validator(keys, "test-audience")
	Init(validator, "http://test")

	accessToken := new(AccessToken)
	refreshToken := new(RefreshToken)

	if err := accessToken.Decode(session.AccessToken, validator); err != nil {
		t.Fatalf("failed to decode access token: %v", err)
	}

	if err := refreshToken.Decode(session.RefreshToken, validator); err != nil {
		t.Fatalf("failed to decode refresh token: %v", err)
	}

	// Set custom cookie options
	SetCookieOptions(CookieOptions{
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
		Path:     "/custom",
	})

	// Create response recorder
	rr := httptest.NewRecorder()
	SetTokenCookies(rr, accessToken, refreshToken)

	// Check cookies
	cookies := rr.Result().Cookies()
	if len(cookies) != 2 {
		t.Fatalf("expected 2 cookies, got %d", len(cookies))
	}

	for _, cookie := range cookies {
		if cookie.Secure != false {
			t.Errorf("cookie %s has Secure=%v, expected false", cookie.Name, cookie.Secure)
		}
		if cookie.SameSite != http.SameSiteLaxMode {
			t.Errorf("cookie %s has SameSite=%v, expected Lax", cookie.Name, cookie.SameSite)
		}
		if cookie.Path != "/custom" {
			t.Errorf("cookie %s has Path=%s, expected /custom", cookie.Name, cookie.Path)
		}
	}

	// Reset options
	SetCookieOptions(CookieOptions{
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		Path:     "/",
	})
	insecureCookieWarningEmitted = false
}

func TestRefreshTokensUsesHTTPClient(t *testing.T) {
	// Create a test server
	called := false
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true

		// Read and verify request body
		body, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(body), "test-token") {
			t.Errorf("expected request body to contain 'test-token', got: %s", string(body))
		}

		// Return mock response
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"accessToken": "mock-access",
			"refreshToken": "mock-refresh"
		}`))
	}))
	defer ts.Close()

	// Set custom HTTP client
	customClient := ts.Client()
	SetHTTPClient(customClient)

	// Set auth URL to test server
	_authUrl = ts.URL

	// Call RefreshTokens (it will fail to decode the mock tokens, but we just want to verify the HTTP call)
	RefreshTokens("test-token")

	if !called {
		t.Error("HTTP client was not used to make the request")
	}

	// Reset
	SetHTTPClient(http.DefaultClient)
}

func TestClearTokenCookiesUsesOptions(t *testing.T) {
	// Set custom path
	SetCookieOptions(CookieOptions{
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		Path:     "/custom",
	})

	rr := httptest.NewRecorder()
	ClearTokenCookies(rr)

	cookies := rr.Result().Cookies()
	if len(cookies) != 2 {
		t.Fatalf("expected 2 cookies, got %d", len(cookies))
	}

	for _, cookie := range cookies {
		if cookie.Path != "/custom" {
			t.Errorf("cookie %s has Path=%s, expected /custom", cookie.Name, cookie.Path)
		}
		if cookie.MaxAge != -1 {
			t.Errorf("cookie %s has MaxAge=%d, expected -1", cookie.Name, cookie.MaxAge)
		}
	}

	// Reset
	SetCookieOptions(CookieOptions{
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		Path:     "/",
	})
	insecureCookieWarningEmitted = false
}
