// Package testing provides utilities for testing applications that integrate
// with the consent identity server.
//
// This package enables consuming projects to write tests for their own API
// logic without running a real consent server. It provides:
//
//   - TestEnv: Token issuing and validation utilities
//   - TestVerifier: A client.Verifier implementation that works locally
//   - HTTP helpers: Functions to create authenticated test requests
//
// # Basic Usage
//
// The simplest way to test authenticated routes:
//
//	func TestProtectedRoute(t *testing.T) {
//	    // Create a test verifier - no network, no real consent server
//	    tv := testing.NewTestVerifier("consent.example.com", "my-app")
//
//	    // Wire up your app with the test verifier
//	    router := myapp.NewRouter(tv)  // tv implements client.Verifier
//
//	    // Create an authenticated request
//	    req, _ := tv.AuthenticatedRequest("GET", "/api/profile", "alice")
//	    rr := httptest.NewRecorder()
//
//	    router.ServeHTTP(rr, req)
//
//	    if rr.Code != http.StatusOK {
//	        t.Errorf("expected 200, got %d", rr.Code)
//	    }
//	}
//
// # Lower-Level Token Control
//
// For more control over token creation, use TestEnv directly:
//
//	func TestExpiredToken(t *testing.T) {
//	    env := testing.NewTestEnv("consent.example.com", "my-app")
//
//	    // Issue an already-expired access token
//	    accessToken, _ := env.IssueAccessToken("alice", -1*time.Hour)
//
//	    req, _ := http.NewRequest("GET", "/api/profile", nil)
//	    env.AddAccessTokenCookie(req, accessToken)
//
//	    // Test that your app handles expired tokens correctly...
//	}
//
// # CSRF Testing
//
// To test CSRF-protected endpoints:
//
//	func TestCSRFProtection(t *testing.T) {
//	    tv := testing.NewTestVerifier("consent.example.com", "my-app")
//	    env := tv.TestEnv()
//
//	    // Issue tokens to get the CSRF secret
//	    refreshToken, _ := env.IssueRefreshToken("alice", time.Hour)
//	    csrfSecret := refreshToken.Secret()
//
//	    // Build request with CSRF
//	    req, _ := http.NewRequest("POST", "/api/settings?csrf="+csrfSecret, nil)
//	    accessToken, _ := env.IssueAccessToken("alice", time.Hour)
//	    env.AddAuthCookies(req, accessToken, refreshToken)
//
//	    // Test...
//	}
//
// # Integration with Your Application
//
// To enable testing, your application should depend on the client.Verifier
// interface rather than *client.Client:
//
//	type MyApp struct {
//	    auth client.Verifier  // Not *client.Client
//	}
//
// In production, pass a *client.Client (which implements Verifier).
// In tests, pass a *testing.TestVerifier.
package testing
