package testing

import (
	"crypto/ecdsa"
	"net/http"
	"time"

	"git.sr.ht/~jakintosh/consent/pkg/tokens"
)

// TestEnv provides token issuing and validation for tests.
type TestEnv struct {
	Issuer    tokens.Issuer
	Validator tokens.Validator
	Domain    string
	Audience  string
	Scopes    []string
}

// NewTestEnv creates a test environment with a shared key.
// Most tests should use this for performance.
func NewTestEnv(
	domain string,
	audience string,
) *TestEnv {
	opts := tokens.ServerOptions{
		SigningKey:   SharedTestKey(),
		IssuerDomain: domain,
	}
	issuer, validator := tokens.InitServer(opts)
	return &TestEnv{
		Issuer:    issuer,
		Validator: validator,
		Domain:    domain,
		Audience:  audience,
		Scopes:    nil,
	}
}

// NewTestEnvWithKey creates a test environment with a specific key.
// Use when testing key mismatch scenarios.
func NewTestEnvWithKey(
	key *ecdsa.PrivateKey,
	domain string,
	audience string,
) *TestEnv {
	opts := tokens.ServerOptions{
		SigningKey:   key,
		IssuerDomain: domain,
	}
	issuer, validator := tokens.InitServer(opts)

	return &TestEnv{
		Issuer:    issuer,
		Validator: validator,
		Domain:    domain,
		Audience:  audience,
		Scopes:    nil,
	}
}

// IssueAccessToken creates a valid access token for the test audience.
func (env *TestEnv) IssueAccessToken(
	subject string,
	lifetime time.Duration,
) (
	*tokens.AccessToken,
	error,
) {
	return env.Issuer.IssueAccessToken(subject, []string{env.Audience}, env.Scopes, lifetime)
}

// IssueRefreshToken creates a valid refresh token for the test audience.
func (env *TestEnv) IssueRefreshToken(
	subject string,
	lifetime time.Duration,
) (
	*tokens.RefreshToken,
	error,
) {
	return env.Issuer.IssueRefreshToken(subject, []string{env.Audience}, env.Scopes, lifetime)
}

// IssueAccessTokenWithAudience creates an access token with custom audiences.
func (env *TestEnv) IssueAccessTokenWithAudience(
	subject string,
	audience []string,
	lifetime time.Duration,
) (
	*tokens.AccessToken,
	error,
) {
	return env.Issuer.IssueAccessToken(subject, audience, env.Scopes, lifetime)
}

// IssueRefreshTokenWithAudience creates a refresh token with custom audiences.
func (env *TestEnv) IssueRefreshTokenWithAudience(
	subject string,
	audience []string,
	lifetime time.Duration,
) (
	*tokens.RefreshToken,
	error,
) {
	return env.Issuer.IssueRefreshToken(subject, audience, env.Scopes, lifetime)
}

// SetTokenCookies sets HTTP-only cookies for the access and refresh tokens.
// Cookies are intentionally insecure to support http://localhost in dev.
func (env *TestEnv) SetTokenCookies(
	w http.ResponseWriter,
	accessToken *AccessToken,
	refreshToken *RefreshToken,
) {
	setTokenCookies(w, accessToken, refreshToken)
}

// ClearTokenCookies removes the access and refresh token cookies by setting
// their MaxAge to -1. Cookies are intentionally insecure to support http://localhost in dev.
func (env *TestEnv) ClearTokenCookies(w http.ResponseWriter) {
	clearTokenCookies(w)
}
