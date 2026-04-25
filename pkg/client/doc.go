// Package client provides integration with the consent identity server for
// backend applications.
//
// This package implements the client side of an OAuth-style authorization flow,
// handling token validation, automatic token refresh, mode-aware cookie
// management, and scoped calls back to Consent's resource API. Backend
// applications use this package to protect their routes and verify user
// identity from tokens issued by the Consent server.
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
//	clientOpts := tokens.ClientOptions{
//	    VerificationKey: publicKey,             // Consent server's ECDSA public key
//	    IssuerDomain:    "consent.example.com", // Consent server domain
//	    ValidAudience:   "myapp.example.com",   // Your app's identifier
//	}
//	validator := tokens.InitClient(clientOpts)
//
//	// Initialize the client
//	authClient := client.Init(validator, "https://consent.example.com")
//
//	// Optional: local development only (plain HTTP localhost)
//	// authClient.EnableInsecureCookies()
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
//	    subject := accessToken.Subject()
//	    fmt.Fprintf(w, "Opaque subject: %s", subject)
//	}
//
// # Authorization Code Flow
//
// Register a handler for the OAuth authorization code callback. Integrations should
// start browser authentication at Consent's `/authorize` endpoint and configure
// this handler as the registered redirect URL:
//
//	// Register the callback handler at /auth/callback
//	http.HandleFunc("/auth/callback", authClient.HandleAuthorizationCode())
//	// Redirect users to:
//	// https://consent.example.com/authorize?integration=myapp&scope=identity&scope=profile
//
//	// When users complete login at the consent server, they'll be redirected
//	// back to /auth/callback?auth_code=... and this handler will:
//	// 1. Exchange the code for tokens
//	// 2. Set auth cookies
//	// 3. Redirect to your home page
//
// If you want to abstract this callback for dependency injection, depend on
// AuthorizationCodeHandler or AuthClient instead of *Client.
//
// # Logout Handler
//
// Register HandleLogout to clear local cookies and proactively revoke the
// refresh token from the consent server:
//
//	http.HandleFunc("/logout", authClient.HandleLogout())
//
// The logout handler validates CSRF using the `csrf` query parameter against
// the refresh token secret in cookies. The handler supports both GET and POST
// routes; POST is preferred for state-changing operations.
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
// Tokens are managed automatically through HTTP-only cookies:
//
//	// Set cookies after obtaining tokens
//	authClient.SetTokenCookies(w, accessToken, refreshToken)
//
//	// Clear cookies on logout
//	authClient.ClearTokenCookies(w)
//
// By default, cookies use Secure=true.
// EnableInsecureCookies uses Secure=false cookies for localhost HTTP
// development only. Never use insecure cookies in production.
//
// # Error Handling
//
// The package defines several error types for different failure modes.
// Errors may wrap additional validation context, so use errors.Is:
//
//	accessToken, err := authClient.VerifyAuthorization(w, r)
//	switch {
//	case errors.Is(err, client.ErrTokenAbsent):
//	    // No token cookie present - user needs to log in
//	case errors.Is(err, client.ErrTokenInvalid):
//	    // Token is malformed, expired (and refresh failed), or has wrong signature
//	case errors.Is(err, client.ErrCSRFInvalid):
//	    // CSRF secret doesn't match (only from VerifyAuthorizationCheckCSRF)
//	case errors.Is(err, client.ErrNetworkTokenRefresh):
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
// If a component needs both verification and the auth code callback, depend on
// client.AuthClient (Verifier + AuthorizationCodeHandler).
//
// In production, pass a *client.Client. In tests, use the testing package's
// TestVerifier. See the testing package documentation for details.
//
// # Scoped User Info
//
// Tokens carry an opaque `sub` value plus the requested scopes for that
// authorization event. If your application needs user-facing profile data,
// call Consent's `/api/v1/auth/userinfo` resource endpoint with the scoped access token:
//
//	userInfo, err := authClient.FetchUserInfo(accessToken.Encoded())
//	if err != nil {
//	    return err
//	}
//	fmt.Println(userInfo.Sub)
//	if userInfo.Profile != nil {
//	    fmt.Println(userInfo.Profile.Handle)
//	}
//
// `/api/v1/auth/userinfo` is a bearer-token resource endpoint. It does not use cookie
// fallback; callers must present the access token explicitly.
package client
