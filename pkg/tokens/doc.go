// Package tokens provides JWT token issuing and validation for the Consent
// identity server.
//
// This package implements ES256 (ECDSA with SHA-256) signed JSON Web Tokens
// with two distinct roles:
//
//   - Server: Issues and validates tokens using a private signing key
//   - Client: Validates tokens using a public verification key
//
// The package defines two token types:
//
//   - AccessToken: Short-lived tokens for API authorization
//   - RefreshToken: Long-lived tokens for obtaining new access tokens (includes CSRF secret)
//
// # Server Usage (Issuing Tokens)
//
// The Consent auth server uses InitServer to create an issuer that can
// generate new tokens:
//
//	// Initialize with your ECDSA private key
//	issuer, validator := tokens.InitServer(tokens.ServerOptions{
//		SigningKey:   signingKey,
//		IssuerDomain: "consent.example.com",
//	})
//
//	// Issue an access token valid for 1 hour
//	accessToken, err := issuer.IssueAccessToken(
//	    "opaque-subject",            // stable opaque subject
//	    []string{"app.example.com"}, // audience
//	    []string{"identity"},        // scopes
//	    time.Hour,                    // lifetime
//	)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Issue a refresh token valid for 30 days
//	refreshToken, err := issuer.IssueRefreshToken(
//	    "opaque-subject",
//	    []string{"app.example.com"},
//	    []string{"identity"},
//	    30*24*time.Hour,
//	)
//
//	// Get encoded token string for transmission
//	tokenString := accessToken.Encoded()
//
// # Client Usage (Validating Tokens)
//
// Backend applications use InitClient to validate tokens issued by the
// Consent server:
//
//	// Initialize with the consent server's public key
//	clientOpts := tokens.ClientOptions{
//	    VerificationKey: publicKey,             // ECDSA public key
//	    IssuerDomain:    "consent.example.com", // expected issuer
//	    ValidAudience:   "app.example.com",     // your application's audience
//	}
//	validator := tokens.InitClient(clientOpts)
//
//	// Validate an access token presented to your application
//	token := &tokens.AccessToken{}
//	if err := token.Decode(tokenString, validator); err != nil {
//	    // Token is invalid, expired, or has wrong issuer/audience
//	    return fmt.Errorf("invalid token: %w", err)
//	}
//
//	// Token is valid - use the claims
//	subject := token.Subject()
//	expiration := token.Expiration()
//
// Issuers must provide at least one non-blank audience value when creating
// access or refresh tokens. Audience matching is only enforced by validators
// created with InitClient.
//
// # Error Handling
//
// Token validation can fail for several reasons:
//
//	err := token.Decode(tokenString, validator)
//	switch {
//	case errors.Is(err, tokens.ErrTokenExpired()):
//	    // Token has expired
//	case errors.Is(err, tokens.ErrTokenInvalidAudience()):
//	    // Token not intended for this application
//	case errors.Is(err, tokens.ErrTokenBadSignature()):
//	    // Token signature verification failed
//	case errors.Is(err, tokens.ErrTokenMalformed()):
//	    // Token structure is invalid
//	}
//
// # CSRF Protection with Refresh Tokens
//
// Refresh tokens include a CSRF secret that can be used to protect
// token refresh endpoints:
//
//	refreshToken, _ := issuer.IssueRefreshToken("opaque-subject", audiences, []string{"identity"}, lifetime)
//	csrfSecret := refreshToken.Secret()
//
//	// Client must provide this secret when refreshing
//	// (typically as a query parameter or form field)
package tokens
