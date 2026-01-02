package testing

import (
	"errors"
	"net/http"
	"time"

	"git.sr.ht/~jakintosh/consent/pkg/client"
	"git.sr.ht/~jakintosh/consent/pkg/tokens"
)

// TestVerifier implements client.Verifier for testing.
// It validates tokens locally and handles refresh without network calls.
type TestVerifier struct {
	env *TestEnv
}

// Compile-time check that TestVerifier implements client.Verifier.
var _ client.Verifier = (*TestVerifier)(nil)

// NewTestVerifier creates a Verifier for testing that requires no network.
func NewTestVerifier(
	domain string,
	audience string,
) *TestVerifier {
	return &TestVerifier{
		env: NewTestEnv(domain, audience),
	}
}

// NewTestVerifierWithEnv creates a TestVerifier using an existing TestEnv.
func NewTestVerifierWithEnv(
	env *TestEnv,
) *TestVerifier {
	return &TestVerifier{
		env: env,
	}
}

// TestEnv returns the underlying TestEnv for token issuance.
func (tv *TestVerifier) TestEnv() *TestEnv {
	return tv.env
}

// AuthenticatedRequest creates an http.Request with valid auth cookies.
func (tv *TestVerifier) AuthenticatedRequest(
	method string,
	url string,
	subject string,
) (
	*http.Request,
	error,
) {
	return tv.env.AuthenticatedRequest(method, url, subject)
}

// VerifyAuthorization implements client.Verifier.
func (tv *TestVerifier) VerifyAuthorization(
	w http.ResponseWriter,
	r *http.Request,
) (
	*client.AccessToken,
	error,
) {
	accessToken, err := tv.validateAccessToken(r)
	if accessToken != nil {
		return accessToken, nil
	}
	if !errorIsRefreshable(err) {
		return nil, client.ErrTokenInvalid
	}

	// If in refreshable state, validate refresh token
	refreshToken, err := tv.validateRefreshToken(r)
	if err != nil {
		if errors.Is(err, client.ErrTokenAbsent) {
			return nil, client.ErrTokenAbsent
		}
		return nil, client.ErrTokenInvalid
	}

	// Refresh the tokens locally (no network call)
	accessToken, refreshToken, err = tv.refreshTokens(refreshToken)
	if err != nil {
		return nil, err
	}
	setTokenCookies(w, accessToken, refreshToken)

	return accessToken, nil
}

// VerifyAuthorizationGetCSRF implements client.Verifier.
func (tv *TestVerifier) VerifyAuthorizationGetCSRF(
	w http.ResponseWriter,
	r *http.Request,
) (
	*client.AccessToken,
	string,
	error,
) {
	accessToken, err := tv.VerifyAuthorization(w, r)
	if err != nil {
		return accessToken, "", err
	}

	// If authorized, validate refresh token and extract CSRF secret
	refreshToken, err := tv.validateRefreshToken(r)
	if err != nil {
		return nil, "", err
	}
	csrfSecret := refreshToken.Secret()

	return accessToken, csrfSecret, nil
}

// VerifyAuthorizationCheckCSRF implements client.Verifier.
func (tv *TestVerifier) VerifyAuthorizationCheckCSRF(
	w http.ResponseWriter,
	r *http.Request,
	reqCSRFSecret string,
) (
	*client.AccessToken,
	string,
	error,
) {
	// Validate refresh token first (before checking access token)
	refreshToken, err := tv.validateRefreshToken(r)
	if err != nil {
		return nil, "", client.ErrTokenInvalid
	}

	currentCSRFSecret := refreshToken.Secret()
	if currentCSRFSecret != reqCSRFSecret {
		return nil, "", client.ErrCSRFInvalid
	}

	// Validate access token
	accessToken, err := tv.validateAccessToken(r)
	if accessToken != nil {
		return accessToken, currentCSRFSecret, nil
	}
	if !errorIsRefreshable(err) {
		return nil, "", client.ErrTokenInvalid
	}

	// Refresh the tokens locally
	accessToken, refreshToken, err = tv.refreshTokens(refreshToken)
	if err != nil {
		return nil, "", err
	}
	newCSRFSecret := refreshToken.Secret()

	setTokenCookies(w, accessToken, refreshToken)
	return accessToken, newCSRFSecret, nil
}

// refreshTokens issues new tokens locally without network calls.
func (tv *TestVerifier) refreshTokens(
	oldRefresh *tokens.RefreshToken,
) (
	*tokens.AccessToken,
	*tokens.RefreshToken,
	error,
) {
	subject := oldRefresh.Subject()
	audience := oldRefresh.Audience()

	accessToken, err := tv.env.Issuer.IssueAccessToken(subject, audience, defaultAccessTokenLifetime)
	if err != nil {
		return nil, nil, err
	}

	refreshToken, err := tv.env.Issuer.IssueRefreshToken(subject, audience, defaultRefreshTokenLifetime)
	if err != nil {
		return nil, nil, err
	}

	return accessToken, refreshToken, nil
}

func (tv *TestVerifier) validateAccessToken(
	r *http.Request,
) (
	*tokens.AccessToken,
	error,
) {
	cookie, err := r.Cookie(accessTokenCookieName)
	if err != nil {
		return nil, client.ErrTokenAbsent
	}

	token := new(tokens.AccessToken)
	if err := token.Decode(cookie.Value, tv.env.Validator); err != nil {
		return nil, err
	}
	return token, nil
}

func (tv *TestVerifier) validateRefreshToken(
	r *http.Request,
) (
	*tokens.RefreshToken,
	error,
) {
	cookie, err := r.Cookie(refreshTokenCookieName)
	if err != nil {
		return nil, client.ErrTokenAbsent
	}

	token := new(tokens.RefreshToken)
	if err := token.Decode(cookie.Value, tv.env.Validator); err != nil {
		return nil, err
	}
	return token, nil
}

func setTokenCookies(
	w http.ResponseWriter,
	accessToken *AccessToken,
	refreshToken *RefreshToken,
) {
	now := time.Now()
	accessMaxAge := int(accessToken.Expiration().Sub(now).Seconds())
	refreshMaxAge := int(refreshToken.Expiration().Sub(now).Seconds())

	http.SetCookie(w, &http.Cookie{
		Name:     accessTokenCookieName,
		Path:     defaultCookiePath,
		Value:    accessToken.Encoded(),
		MaxAge:   accessMaxAge,
		SameSite: http.SameSiteStrictMode,
		Secure:   false,
		HttpOnly: true,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     refreshTokenCookieName,
		Path:     defaultCookiePath,
		Value:    refreshToken.Encoded(),
		MaxAge:   refreshMaxAge,
		SameSite: http.SameSiteStrictMode,
		Secure:   false,
		HttpOnly: true,
	})
}

func errorIsRefreshable(err error) bool {
	if errors.Is(err, client.ErrTokenAbsent) {
		return true
	}
	if errors.Is(err, tokens.ErrTokenExpired()) {
		return true
	}
	return false
}
