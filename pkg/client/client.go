package client

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"git.sr.ht/~jakintosh/consent/internal/api"
	"git.sr.ht/~jakintosh/consent/pkg/tokens"
)

// LogLevel controls the verbosity of client logging output.
type LogLevel int

const (
	LogLevelNone  LogLevel = iota // No logging
	LogLevelError                  // Log errors only
	LogLevelInfo                   // Log errors and informational messages
	LogLevelDebug                  // Log everything including debug details
)

const LogLevelDefault = LogLevelError

var (
	// ErrTokenAbsent indicates no token cookie was found in the request.
	ErrTokenAbsent = errors.New("token not present")

	// ErrTokenInvalid indicates the token is malformed, has an invalid signature,
	// wrong issuer/audience, or is expired and cannot be refreshed.
	ErrTokenInvalid = errors.New("token invalid")

	// ErrCSRFInvalid indicates the provided CSRF secret doesn't match the
	// refresh token's secret.
	ErrCSRFInvalid = errors.New("csrf secret incorrect")

	// ErrNetworkTokenRefresh indicates a network error occurred while
	// communicating with the consent server during token refresh.
	ErrNetworkTokenRefresh = errors.New("network issue during token refresh")
)

// Client manages authorization for a backend application integrating with
// the consent identity server. It handles token validation, automatic refresh,
// and cookie management.
//
// Create a Client using Init, then use its methods to protect your HTTP handlers.
type Client struct {
	logLevel       LogLevel
	authUrl        string
	tokenValidator TokenValidator
}

// Init creates a new Client for integrating with the consent identity server.
//
// Parameters:
//   - validator: Token validator (typically from tokens.InitClient)
//   - authUrl: Full URL of the consent server (e.g., "https://consent.example.com")
//
// The client defaults to LogLevelError. Use SetLogLevel to adjust verbosity.
func Init(
	validator TokenValidator,
	authUrl string,
) *Client {
	return &Client{
		logLevel:       LogLevelDefault,
		authUrl:        authUrl,
		tokenValidator: validator,
	}
}

func (c *Client) log(level LogLevel, format string, v ...any) {
	if c.logLevel >= level {
		log.Printf(format, v...)
	}
}

func (c *Client) SetLogLevel(logLevel LogLevel) {
	c.logLevel = logLevel
}

/*
HandleAuthorizationCode returns a handler that fully handles the authorization
code flow for a client. Set this to the same route you register with the
auth server as the redirect link, and it works out of the box.
*/
func (c *Client) HandleAuthorizationCode() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// extract 'auth_code' refresh token
		queries := r.URL.Query()
		code := queries.Get("auth_code")
		if code == "" {
			c.log(LogLevelDebug, "handle auth code error: missing required 'auth_code' query param\n")
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}

		// refresh tokens using code
		accessToken, refreshToken, ok := c.RefreshTokens(code)
		if !ok {
			c.log(LogLevelDebug, "handle auth code error: error refreshing with auth server\n")
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}

		c.SetTokenCookies(w, accessToken, refreshToken)
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}

/*
VerifyAuthorization allows a client to pass in an http.Request and determine
whether or not the request is authorized, and if so, return the access token.
If the access token is expired, this will attempt to call the authorization
server to refresh the tokens.
*/
func (c *Client) VerifyAuthorization(
	w http.ResponseWriter,
	r *http.Request,
) (*AccessToken, error) {

	// validate access token in the request
	accessToken, err := validateAccessToken(r, c.tokenValidator)
	if accessToken != nil {
		return accessToken, nil
	}
	if !errorIsRefreshable(err) {
		c.log(LogLevelDebug, "failed to validate access token: %v\n", err)
		return nil, ErrTokenInvalid
	}

	// if in refreshable state, validate refresh token
	refreshToken, err := validateRefreshToken(r, c.tokenValidator)
	if err != nil {
		if errors.Is(err, ErrTokenAbsent) {
			return nil, ErrTokenAbsent
		} else {
			c.log(LogLevelDebug, "failed to validate refresh token: %v\n", err)
			return nil, ErrTokenInvalid
		}
	}

	// refresh the tokens
	accessToken, refreshToken, ok := c.RefreshTokens(refreshToken.Encoded())
	if !ok {
		c.log(LogLevelDebug, "couldn't exchange refresh token: error refreshing with auth server\n")
		return nil, ErrNetworkTokenRefresh
	}
	c.SetTokenCookies(w, accessToken, refreshToken)

	return accessToken, err
}

// VerifyAuthorizationGetCSRF verifies authorization and returns the CSRF secret
// from the refresh token. Use this for GET requests that need to provide a CSRF
// token to the client (e.g., in a form or as a query parameter for subsequent
// state-changing requests).
//
// Returns the access token, CSRF secret, and any error. If the access token is
// expired, it will be automatically refreshed.
func (c *Client) VerifyAuthorizationGetCSRF(
	w http.ResponseWriter,
	r *http.Request,
) (
	*AccessToken,
	string,
	error,
) {

	// standard request verification
	accessToken, err := c.VerifyAuthorization(w, r)
	if err != nil {
		return accessToken, "", err
	}

	// if authorized success, validate refresh token and extract csrf secret
	refreshToken, err := validateRefreshToken(r, c.tokenValidator)
	if err != nil {
		c.log(LogLevelDebug, "failed to validate refresh token: %v\n", err)
		return nil, "", err
	}
	csrfSecret := refreshToken.Secret()

	return accessToken, csrfSecret, nil
}

/*
VerifyAuthorizationCheckCSRF decodes the RefreshToken first to see if the CSRF
code matches. Because the AccessToken may be legally expired, we check
RefreshToken's CSRF secret first, because after the AccessToken check the
RefreshToken may have been changed.
*/
func (c *Client) VerifyAuthorizationCheckCSRF(
	w http.ResponseWriter,
	r *http.Request,
	reqCSRFSecret string,
) (
	*AccessToken,
	string,
	error,
) {

	// validate refresh token from request
	refreshToken, err := validateRefreshToken(r, c.tokenValidator)
	if err != nil {
		c.log(LogLevelDebug, "failed to validate refresh token: %v\n", err)
		return nil, "", ErrTokenInvalid
	}

	currentCSRFSecret := refreshToken.Secret()
	if currentCSRFSecret != reqCSRFSecret {
		return nil, "", ErrCSRFInvalid
	}

	// validate access token in the request
	accessToken, err := validateAccessToken(r, c.tokenValidator)
	if accessToken != nil {
		return accessToken, currentCSRFSecret, nil
	}
	if !errorIsRefreshable(err) {
		return nil, "", ErrTokenInvalid
	}

	// refresh the tokens
	accessToken, refreshToken, ok := c.RefreshTokens(refreshToken.Encoded())
	if !ok {
		c.log(LogLevelDebug, "couldn't exchange refresh token: error refreshing with auth server\n")
		return nil, "", ErrNetworkTokenRefresh
	}
	newCSRFSecret := refreshToken.Secret()

	c.SetTokenCookies(w, accessToken, refreshToken)
	return accessToken, newCSRFSecret, nil
}

/*
RefreshTokens uses the provided encoded RefreshToken to fetch new tokens from
the auth server. You can automatically invoke this behavior with
VerifyAuthorization(), but can use this on its own to compose custom refresh
flows.

Returns decoded token structures and a bool indicating success.
*/
func (c *Client) RefreshTokens(
	refreshTokenStr string,
) (
	*AccessToken,
	*RefreshToken,
	bool,
) {

	// construct a POST request to the /api/refresh route
	url := fmt.Sprintf("%s/api/refresh", c.authUrl)
	body := bytes.NewBuffer(fmt.Appendf(nil, `{ "refreshToken" : "%s" }`, refreshTokenStr))
	c.log(LogLevelDebug, "POST { refresh_token } => %s\n", url)
	apiResponse, err := http.Post(url, "application/json", body)
	if err != nil {
		c.log(LogLevelError, "failed to post refresh: %v\n", err)
		return nil, nil, false
	}

	// decode api response
	if apiResponse.StatusCode != http.StatusOK {
		c.log(LogLevelDebug, "POST %s returned %s\n", url, apiResponse.Status)
		return nil, nil, false
	}
	defer apiResponse.Body.Close()
	refreshResponse := new(api.RefreshResponse)
	if err := json.NewDecoder(apiResponse.Body).Decode(refreshResponse); err != nil {
		c.log(LogLevelError, "failed to decode api response: %v\n", err)
		return nil, nil, false
	}

	// decode tokens from response
	accessToken := new(AccessToken)
	if err := accessToken.Decode(refreshResponse.AccessToken, c.tokenValidator); err != nil {
		c.log(LogLevelError, "failed to decode access token: %v\n", err)
		return nil, nil, false
	}
	refreshToken := new(RefreshToken)
	if err := refreshToken.Decode(refreshResponse.RefreshToken, c.tokenValidator); err != nil {
		c.log(LogLevelError, "failed to decode refresh token: %v\n", err)
		return nil, nil, false
	}
	return accessToken, refreshToken, true
}

// SetTokenCookies sets secure HTTP-only cookies for the access and refresh tokens.
// The cookies are configured with SameSite=Strict, Secure=true, and HttpOnly=true
// for security. Cookie expiration is set to match the token lifetimes.
//
// Call this after successful login or token refresh to store tokens in the client's browser.
func (c *Client) SetTokenCookies(w http.ResponseWriter, accessToken *AccessToken, refreshToken *RefreshToken) {
	now := time.Now()
	accessMaxAge := accessToken.Expiration().Sub(now).Seconds()
	refreshMaxAge := refreshToken.Expiration().Sub(now).Seconds()

	accessTokenCookie := &http.Cookie{
		Name:     "accessToken",
		Path:     "/",
		Value:    accessToken.Encoded(),
		MaxAge:   int(accessMaxAge),
		SameSite: http.SameSiteStrictMode,
		Secure:   true,
		HttpOnly: true,
	}
	refreshTokenCookie := &http.Cookie{
		Name:     "refreshToken",
		Path:     "/",
		Value:    refreshToken.Encoded(),
		MaxAge:   int(refreshMaxAge),
		SameSite: http.SameSiteStrictMode,
		Secure:   true,
		HttpOnly: true,
	}

	http.SetCookie(w, accessTokenCookie)
	http.SetCookie(w, refreshTokenCookie)

	c.log(LogLevelDebug, "set token cookies\n")
}

// ClearTokenCookies removes the access and refresh token cookies by setting
// their MaxAge to -1. Call this during logout to clear the user's session.
func (c *Client) ClearTokenCookies(w http.ResponseWriter) {
	accessTokenCookie := &http.Cookie{
		Name:   "accessToken",
		Path:   "/",
		MaxAge: -1,
	}
	refreshTokenCookie := &http.Cookie{
		Name:   "refreshToken",
		Path:   "/",
		MaxAge: -1,
	}

	http.SetCookie(w, accessTokenCookie)
	http.SetCookie(w, refreshTokenCookie)

	c.log(LogLevelDebug, "cleared token cookies\n")
}

func getCookie(r *http.Request, cookieName string) *http.Cookie {
	if cookie, err := r.Cookie(cookieName); err == nil {
		return cookie
	}
	return nil
}

func validateAccessToken(r *http.Request, validator TokenValidator) (*AccessToken, error) {
	cookie := getCookie(r, "accessToken")
	if cookie == nil {
		return nil, ErrTokenAbsent
	}

	token := new(AccessToken)
	err := token.Decode(cookie.Value, validator)
	if err != nil {
		return nil, err
	}
	return token, nil
}

func validateRefreshToken(r *http.Request, validator TokenValidator) (*RefreshToken, error) {
	cookie := getCookie(r, "refreshToken")
	if cookie == nil {
		return nil, ErrTokenAbsent
	}

	token := new(RefreshToken)
	err := token.Decode(cookie.Value, validator)
	if err != nil {
		return nil, err
	}
	return token, nil
}

func errorIsRefreshable(err error) bool {
	if errors.Is(err, ErrTokenAbsent) {
		return true
	} else if errors.Is(err, tokens.ErrTokenExpired()) {
		return true
	} else {
		return false
	}
}
