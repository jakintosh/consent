package testing

import (
	"crypto/ecdsa"
	"time"

	"git.sr.ht/~jakintosh/consent/pkg/tokens"
)

// TestEnv provides token issuing and validation for tests.
type TestEnv struct {
	Issuer    tokens.Issuer
	Validator tokens.Validator
	Domain    string
	Audience  string
}

// NewTestEnv creates a test environment with a shared key.
// Most tests should use this for performance.
func NewTestEnv(
	domain string,
	audience string,
) *TestEnv {
	key := SharedTestKey()
	issuer, validator := tokens.InitServer(key, domain)
	return &TestEnv{
		Issuer:    issuer,
		Validator: validator,
		Domain:    domain,
		Audience:  audience,
	}
}

// NewTestEnvWithKey creates a test environment with a specific key.
// Use when testing key mismatch scenarios.
func NewTestEnvWithKey(
	key *ecdsa.PrivateKey,
	domain string,
	audience string,
) *TestEnv {
	issuer, validator := tokens.InitServer(key, domain)
	return &TestEnv{
		Issuer:    issuer,
		Validator: validator,
		Domain:    domain,
		Audience:  audience,
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
	return env.Issuer.IssueAccessToken(subject, []string{env.Audience}, lifetime)
}

// IssueRefreshToken creates a valid refresh token for the test audience.
func (env *TestEnv) IssueRefreshToken(
	subject string,
	lifetime time.Duration,
) (
	*tokens.RefreshToken,
	error,
) {
	return env.Issuer.IssueRefreshToken(subject, []string{env.Audience}, lifetime)
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
	return env.Issuer.IssueAccessToken(subject, audience, lifetime)
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
	return env.Issuer.IssueRefreshToken(subject, audience, lifetime)
}
