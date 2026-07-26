package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"git.sr.ht/~jakintosh/command-go/pkg/wire"
	"git.sr.ht/~jakintosh/consent/internal/api"
	"git.sr.ht/~jakintosh/consent/pkg/tokens"
)

// LogLevel controls the verbosity of client logging output.
type LogLevel int

const (
	LogLevelNone  LogLevel = iota // No logging
	LogLevelError                 // Log errors only
	LogLevelInfo                  // Log errors and informational messages
	LogLevelDebug                 // Log everything including debug details
)

const LogLevelDefault = LogLevelError

var (
	// ErrTokenAbsent indicates no token cookie was found in the request.
	ErrTokenAbsent = errors.New("token not present")

	// ErrTokenInvalid indicates the token is malformed, has an invalid signature,
	// has the wrong issuer or audience, or cannot be refreshed because the token
	// itself is not refreshable.
	ErrTokenInvalid = errors.New("token invalid")

	// ErrCSRFInvalid indicates the provided CSRF secret doesn't match the
	// refresh token's secret.
	ErrCSRFInvalid = errors.New("csrf secret incorrect")

	// ErrTokenRefresh indicates that a token refresh request or response failed.
	ErrTokenRefresh = errors.New("token refresh failed")
)

type UserInfo struct {
	Sub     string           `json:"sub"`
	Profile *UserInfoProfile `json:"profile,omitempty"`
}

type UserInfoProfile struct {
	Handle string `json:"handle"`
}

// Client manages authorization for a backend application integrating with
// the consent identity server. It handles token validation, automatic refresh,
// and cookie management.
//
// Create a Client using Init, then use its methods to protect your HTTP handlers.
type Client struct {
	apiClient       wire.Client
	insecureCookies bool
	logLevel        LogLevel
	authUrl         string
	tokenValidator  TokenValidator
}

// Init creates a new Client for integrating with the consent identity server.
//
// Parameters:
//   - validator: Token validator (typically from tokens.InitClient)
//   - authUrl: Full URL of the consent server (e.g., "https://consent.example.com")
//
// The client defaults to LogLevelError. Use SetLogLevel to adjust verbosity.
// Init returns an error when authUrl is not an absolute HTTP or HTTPS URL.
func Init(
	validator TokenValidator,
	authUrl string,
) (
	*Client,
	error,
) {
	// TODO: Maybe we can take in client options here, and not require the caller to create a token validator externally? We almost always do the same thing outside? We should investigate
	authUrl, err := normalizeAuthURL(authUrl)
	if err != nil {
		return nil, fmt.Errorf("client: invalid auth URL: %w", err)
	}

	apiClient, err := wire.NewClient(authUrl, wire.ClientOptions{})
	if err != nil {
		return nil, fmt.Errorf("client: initialize API client: %w", err)
	}

	return &Client{
		apiClient:       apiClient,
		insecureCookies: false,
		logLevel:        LogLevelDefault,
		authUrl:         authUrl,
		tokenValidator:  validator,
	}, nil
}

func (c *Client) SetLogLevel(logLevel LogLevel) {
	c.logLevel = logLevel
}

// EnableInsecureCookies configures this client to emit Secure=false cookies.
//
// This is intended for local HTTP environments such as localhost testing.
// Never enable this in production.
func (c *Client) EnableInsecureCookies() {
	if !c.insecureCookies {
		fmt.Println("WARNING: Cookies have been set to INSECURE. Do not use in production.")
	}
	c.insecureCookies = true
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
		accessToken, refreshToken, err := c.RefreshTokens(r.Context(), code)
		if err != nil {
			c.log(LogLevelDebug, "handle auth code error: %v\n", err)
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}

		c.SetTokenCookies(w, accessToken, refreshToken)
		http.Redirect(w, r, callbackReturnTo(r.URL.Query().Get("return_to")), http.StatusSeeOther)
	}
}

// HandleLogout returns a handler that revokes the current refresh token,
// clears auth cookies, and redirects to "/".
//
// The request must include a CSRF token in the `csrf` query parameter that
// matches the refresh token secret. The handler is method-agnostic and may be
// registered for GET, POST, or both.
func (c *Client) HandleLogout() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		// check refresh token
		refreshToken, err := validateRefreshToken(r, c.tokenValidator)
		if err != nil {
			// note missing token
			c.log(LogLevelDebug, "handle logout: invalid refresh token: %v\n", err)
		} else {
			// if present, validate CSRF and revoke
			csrfSecret := r.URL.Query().Get("csrf")
			if csrfSecret == "" || refreshToken.Secret() != csrfSecret {
				// if csrf fails, do not clear or revoke—invalid logout request
				http.Error(w, "CSRF validation failed", http.StatusForbidden)
				return
			}

			if err := revokeRefreshToken(r.Context(), c.apiClient, refreshToken); err != nil {
				c.log(LogLevelError, "handle logout: failed to revoke refresh token (%v)\n", err)
			}
		}

		// always clear cookies and redirect
		c.ClearTokenCookies(w)
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
) (
	*AccessToken,
	error,
) {

	// validate access token in the request
	accessToken, err := validateAccessToken(r, c.tokenValidator)
	if accessToken != nil {
		return accessToken, nil
	}
	if !errorIsRefreshable(err) {
		c.log(LogLevelDebug, "failed to validate access token: %v\n", err)
		return nil, fmt.Errorf("%w: %v", ErrTokenInvalid, err)
	}

	// if in refreshable state, validate refresh token
	refreshToken, err := validateRefreshToken(r, c.tokenValidator)
	if err != nil {
		c.log(LogLevelDebug, "failed to validate refresh token: %v\n", err)
		return nil, err
	}

	// refresh the tokens
	accessToken, refreshToken, err = c.RefreshTokens(r.Context(), refreshToken.Encoded())
	if err != nil {
		c.log(LogLevelDebug, "couldn't exchange refresh token: %v\n", err)
		return nil, err
	}
	c.SetTokenCookies(w, accessToken, refreshToken)

	return accessToken, nil
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
	// validate refresh token from request
	refreshToken, err := validateRefreshToken(r, c.tokenValidator)
	if err != nil {
		c.log(LogLevelDebug, "failed to validate refresh token: %v\n", err)
		return nil, "", err
	}

	// validate access token in the request
	accessToken, err := validateAccessToken(r, c.tokenValidator)
	if accessToken != nil {
		return accessToken, refreshToken.Secret(), nil
	}
	if !errorIsRefreshable(err) {
		return nil, "", fmt.Errorf("%w: %v", ErrTokenInvalid, err)
	}

	// refresh the tokens
	accessToken, refreshToken, err = c.RefreshTokens(r.Context(), refreshToken.Encoded())
	if err != nil {
		c.log(LogLevelDebug, "couldn't exchange refresh token: %v\n", err)
		return nil, "", err
	}

	c.SetTokenCookies(w, accessToken, refreshToken)

	return accessToken, refreshToken.Secret(), nil
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
		return nil, "", err
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
		return nil, "", fmt.Errorf("%w: %v", ErrTokenInvalid, err)
	}

	// refresh the tokens
	accessToken, refreshToken, err = c.RefreshTokens(r.Context(), refreshToken.Encoded())
	if err != nil {
		c.log(LogLevelDebug, "couldn't exchange refresh token: %v\n", err)
		return nil, "", err
	}
	newCSRFSecret := refreshToken.Secret()

	c.SetTokenCookies(w, accessToken, refreshToken)
	return accessToken, newCSRFSecret, nil
}

/*
RefreshTokens uses the provided context and encoded RefreshToken to fetch new
tokens from the auth server. You can automatically invoke this behavior with
VerifyAuthorization(), but can use this on its own to compose custom refresh
flows.

Returns decoded token structures or an error wrapping ErrTokenRefresh. Transport
and cancellation errors remain available through errors.Is and errors.As.
*/
func (c *Client) RefreshTokens(
	ctx context.Context,
	refreshTokenStr string,
) (
	*AccessToken,
	*RefreshToken,
	error,
) {
	body, err := json.Marshal(api.RefreshRequest{RefreshToken: refreshTokenStr})
	if err != nil {
		return nil, nil, fmt.Errorf("%w: encode request: %w", ErrTokenRefresh, err)
	}

	response := api.RefreshResponse{}
	c.log(LogLevelDebug, "POST { refresh_token } => %s/api/v1/auth/refresh\n", c.authUrl)
	if err := c.apiClient.Post(ctx, "/api/v1/auth/refresh", body, &response); err != nil {
		return nil, nil, fmt.Errorf("%w: request: %w", ErrTokenRefresh, err)
	}
	if response.AccessToken == "" || response.RefreshToken == "" {
		return nil, nil, fmt.Errorf("%w: response contains empty tokens", ErrTokenRefresh)
	}

	// decode tokens from response
	accessToken := new(AccessToken)
	if err := accessToken.Decode(response.AccessToken, c.tokenValidator); err != nil {
		return nil, nil, fmt.Errorf("%w: decode access token: %w", ErrTokenRefresh, err)
	}
	refreshToken := new(RefreshToken)
	if err := refreshToken.Decode(response.RefreshToken, c.tokenValidator); err != nil {
		return nil, nil, fmt.Errorf("%w: decode refresh token: %w", ErrTokenRefresh, err)
	}
	return accessToken, refreshToken, nil
}

// SetTokenCookies sets HTTP-only cookies for the access and refresh tokens.
//
// By default, cookies are configured with
// SameSite=Lax, Secure=true, and HttpOnly=true.
// When EnableInsecureCookies is set, cookies are configured with
// SameSite=Lax, Secure=false, and HttpOnly=true to support local HTTP.
//
// Call this after successful login or token refresh to store tokens in the client's browser.
func (c *Client) SetTokenCookies(
	w http.ResponseWriter,
	accessToken *AccessToken,
	refreshToken *RefreshToken,
) {
	now := time.Now()
	accessMaxAge := accessToken.Expiration().Sub(now).Seconds()
	refreshMaxAge := refreshToken.Expiration().Sub(now).Seconds()
	secureCookie := !c.insecureCookies

	accessTokenCookie := &http.Cookie{
		Name:     "accessToken",
		Path:     "/",
		Value:    accessToken.Encoded(),
		MaxAge:   int(accessMaxAge),
		SameSite: http.SameSiteLaxMode,
		Secure:   secureCookie,
		HttpOnly: true,
	}
	refreshTokenCookie := &http.Cookie{
		Name:     "refreshToken",
		Path:     "/",
		Value:    refreshToken.Encoded(),
		MaxAge:   int(refreshMaxAge),
		SameSite: http.SameSiteLaxMode,
		Secure:   secureCookie,
		HttpOnly: true,
	}

	http.SetCookie(w, accessTokenCookie)
	http.SetCookie(w, refreshTokenCookie)

	c.log(LogLevelDebug, "set token cookies\n")
}

// ClearTokenCookies removes the access and refresh token cookies by setting
// their MaxAge to -1. Call this during logout to clear the user's session.
func (c *Client) ClearTokenCookies(
	w http.ResponseWriter,
) {
	secureCookie := !c.insecureCookies

	accessTokenCookie := &http.Cookie{
		Name:     "accessToken",
		Path:     "/",
		MaxAge:   -1,
		SameSite: http.SameSiteLaxMode,
		Secure:   secureCookie,
		HttpOnly: true,
	}
	refreshTokenCookie := &http.Cookie{
		Name:     "refreshToken",
		Path:     "/",
		MaxAge:   -1,
		SameSite: http.SameSiteLaxMode,
		Secure:   secureCookie,
		HttpOnly: true,
	}

	http.SetCookie(w, accessTokenCookie)
	http.SetCookie(w, refreshTokenCookie)

	c.log(LogLevelDebug, "cleared token cookies\n")
}

// FetchUserInfo returns the user information authorized by accessToken.
//
// The request uses ctx for cancellation and deadlines.
func (c *Client) FetchUserInfo(
	ctx context.Context,
	accessToken string,
) (
	*UserInfo,
	error,
) {
	client := c.apiClient.WithAPIKey(wire.Secret(accessToken))

	var userInfo UserInfo
	if err := client.Get(ctx, "/api/v1/auth/userinfo", &userInfo); err != nil {
		return nil, fmt.Errorf("client: fetch user info: %w", err)
	}

	return &userInfo, nil
}

func (c *Client) log(level LogLevel, format string, v ...any) {
	if c.logLevel >= level {
		log.Printf(format, v...)
	}
}

func revokeRefreshToken(
	ctx context.Context,
	client wire.Client,
	refreshToken *RefreshToken,
) error {
	body, err := json.Marshal(
		api.LogoutRequest{
			RefreshToken: refreshToken.Encoded(),
		},
	)
	if err != nil {
		return fmt.Errorf("failed to encode logout payload: %v\n", err)
	}

	err = client.Post(ctx, "/api/v1/auth/logout", body, nil)
	if err != nil {
		return fmt.Errorf("POST /api/v1/auth/logout failed: %v\n", err)
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
		return nil, fmt.Errorf("%w: %w", ErrTokenInvalid, err)
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
		return nil, fmt.Errorf("%w: %w", ErrTokenInvalid, err)
	}
	return token, nil
}

func normalizeAuthURL(raw string) (string, error) {
	if strings.TrimSpace(raw) != raw || raw == "" {
		return "", errors.New("must be a non-empty URL without surrounding whitespace")
	}

	parsed, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("parse: %w", err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", fmt.Errorf("scheme must be http or https, got %q", parsed.Scheme)
	}
	if parsed.Host == "" {
		return "", errors.New("host is required")
	}
	if parsed.User != nil {
		return "", errors.New("user credentials are not allowed")
	}
	if parsed.RawQuery != "" {
		return "", errors.New("query string is not allowed")
	}
	if parsed.Fragment != "" {
		return "", errors.New("fragment is not allowed")
	}

	parsed.Path = strings.TrimRight(parsed.Path, "/")
	parsed.RawPath = strings.TrimRight(parsed.RawPath, "/")
	return parsed.String(), nil
}

func getCookie(r *http.Request, cookieName string) *http.Cookie {
	if cookie, err := r.Cookie(cookieName); err == nil {
		return cookie
	}
	return nil
}

func callbackReturnTo(returnTo string) string {
	if returnTo == "" {
		return "/"
	}
	parsed, err := url.Parse(returnTo)
	if err != nil || parsed == nil || parsed.IsAbs() || parsed.Host != "" || parsed.Path == "" || parsed.Path[0] != '/' {
		return "/"
	}
	return parsed.String()
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
