// Package client provides integration with the consent identity server for
// backend applications.
//
// This package implements the client side of an OAuth-style authorization flow,
// handling token validation, automatic token refresh, and secure cookie management.
// Backend applications use this package to protect their routes and verify user
// identity from tokens issued by the consent server.
//
// # Quick Start
//
// Initialize a client with your application's token validator and the consent
// server URL:
//
//	import (
//	    "git.sr.ht/~jakintosh/consent/pkg/client"
//	    "git.sr.ht/~jakintosh/consent/pkg/tokens"
//	)
//
//	// Get the consent server's public key and create a validator
//	validator := tokens.InitClient(
//	    publicKey,              // Consent server's ECDSA public key
//	    "consent.example.com",  // Consent server domain
//	    "myapp.example.com",    // Your app's identifier
//	)
//
//	// Initialize the client
//	authClient := client.Init(validator, "https://consent.example.com")
//
// # Protecting Routes
//
// Use VerifyAuthorization to protect your API routes. It automatically handles
// token refresh when access tokens expire:
//
//	func protectedHandler(w http.ResponseWriter, r *http.Request) {
//	    accessToken, err := authClient.VerifyAuthorization(w, r)
//	    if err != nil {
//	        http.Error(w, "Unauthorized", http.StatusUnauthorized)
//	        return
//	    }
//
//	    username := accessToken.Subject()
//	    fmt.Fprintf(w, "Hello, %s!", username)
//	}
//
// # Authorization Code Flow
//
// Register a handler for the OAuth authorization code callback. This is the
// redirect URL you configure with the consent server:
//
//	// Register the callback handler at /auth/callback
//	http.HandleFunc("/auth/callback", authClient.HandleAuthorizationCode())
//
//	// When users complete login at the consent server, they'll be redirected
//	// back to /auth/callback?auth_code=... and this handler will:
//	// 1. Exchange the code for tokens
//	// 2. Set secure cookies
//	// 3. Redirect to your home page
//
// # CSRF Protection
//
// For state-changing operations, use CSRF protection with refresh tokens:
//
//	// GET request - provide CSRF token to client
//	func showSettingsForm(w http.ResponseWriter, r *http.Request) {
//	    accessToken, csrfSecret, err := authClient.VerifyAuthorizationGetCSRF(w, r)
//	    if err != nil {
//	        http.Error(w, "Unauthorized", http.StatusUnauthorized)
//	        return
//	    }
//
//	    // Include csrfSecret in your form (e.g., as hidden field or query param)
//	    fmt.Fprintf(w, `<form action="/settings?csrf=%s" method="POST">...`, csrfSecret)
//	}
//
//	// POST request - verify CSRF token
//	func updateSettings(w http.ResponseWriter, r *http.Request) {
//	    csrfFromRequest := r.URL.Query().Get("csrf")
//	    accessToken, _, err := authClient.VerifyAuthorizationCheckCSRF(w, r, csrfFromRequest)
//	    if err == client.ErrCSRFInvalid {
//	        http.Error(w, "CSRF validation failed", http.StatusForbidden)
//	        return
//	    }
//	    if err != nil {
//	        http.Error(w, "Unauthorized", http.StatusUnauthorized)
//	        return
//	    }
//
//	    // Process the settings update...
//	}
//
// # Token Management
//
// Tokens are managed automatically through secure HTTP-only cookies:
//
//	// Set cookies after obtaining tokens
//	authClient.SetTokenCookies(w, accessToken, refreshToken)
//
//	// Clear cookies on logout
//	authClient.ClearTokenCookies(w)
//
// # Error Handling
//
// The package defines several error types for different failure modes:
//
//	accessToken, err := authClient.VerifyAuthorization(w, r)
//	switch err {
//	case client.ErrTokenAbsent:
//	    // No token cookie present - user needs to log in
//	case client.ErrTokenInvalid:
//	    // Token is malformed, expired (and refresh failed), or has wrong signature
//	case client.ErrCSRFInvalid:
//	    // CSRF secret doesn't match (only from VerifyAuthorizationCheckCSRF)
//	case client.ErrNetworkTokenRefresh:
//	    // Network error communicating with consent server during refresh
//	}
//
// # Testing
//
// For testability, depend on the Verifier interface rather than *Client:
//
//	type MyApp struct {
//	    auth client.Verifier  // Not *client.Client
//	}
//
// In production, pass a *client.Client. In tests, use the testing package's
// TestVerifier. See the testing package documentation for details.
package client
