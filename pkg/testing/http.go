package testing

import (
	"net/http"
	"time"

	"git.sr.ht/~jakintosh/consent/pkg/tokens"
)

// AuthenticatedRequest creates an http.Request with valid auth cookies.
// The tokens are issued for the given subject with default lifetimes.
func (env *TestEnv) AuthenticatedRequest(
	method string,
	url string,
	subject string,
) (
	*http.Request,
	error,
) {
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, err
	}

	accessToken, err := env.IssueAccessToken(subject, 30*time.Minute)
	if err != nil {
		return nil, err
	}
	refreshToken, err := env.IssueRefreshToken(subject, 24*time.Hour)
	if err != nil {
		return nil, err
	}

	env.AddAuthCookies(req, accessToken, refreshToken)
	return req, nil
}

// AddAuthCookies adds auth cookies to an existing request.
func (env *TestEnv) AddAuthCookies(
	req *http.Request,
	accessToken *tokens.AccessToken,
	refreshToken *tokens.RefreshToken,
) {
	env.AddAccessTokenCookie(req, accessToken)
	env.AddRefreshTokenCookie(req, refreshToken)
}

// AddAccessTokenCookie adds only the access token cookie.
func (env *TestEnv) AddAccessTokenCookie(
	req *http.Request,
	accessToken *tokens.AccessToken,
) {
	req.AddCookie(&http.Cookie{
		Name:  "accessToken",
		Value: accessToken.Encoded(),
	})
}

// AddRefreshTokenCookie adds only the refresh token cookie.
func (env *TestEnv) AddRefreshTokenCookie(
	req *http.Request,
	refreshToken *tokens.RefreshToken,
) {
	req.AddCookie(&http.Cookie{
		Name:  "refreshToken",
		Value: refreshToken.Encoded(),
	})
}
