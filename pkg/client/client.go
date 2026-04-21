package client

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"time"

	"git.sr.ht/~jakintosh/command-go/pkg/wire"
	"git.sr.ht/~jakintosh/consent/internal/service"
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
	// wrong issuer/audience, or is expired and cannot be refreshed.
	ErrTokenInvalid = errors.New("token invalid")

	// ErrCSRFInvalid indicates the provided CSRF secret doesn't match the
	// refresh token's secret.
	ErrCSRFInvalid = errors.New("csrf secret incorrect")

	// ErrNetworkTokenRefresh indicates a network error occurred while
	// communicating with the consent server during token refresh.
	ErrNetworkTokenRefresh = errors.New("network issue during token refresh")
)

type MeResponse struct {
	Profile *MeProfile `json:"profile,omitempty"`
}

type MeProfile struct {
	Handle string `json:"handle"`
}

// Client manages authorization for a backend application integrating with
// the consent identity server. It handles token validation, automatic refresh,
// and cookie management.
//
// Create a Client using Init, then use its methods to protect your HTTP handlers.
type Client struct {
	apiClient       *wire.Client
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
func Init(
	validator TokenValidator,
	authUrl string,
) *Client {
	// TODO: Maybe we can take in client options here, and not require the caller t ocreate a token validator externally? We almost always do the same thing outside? We should investigate
	return &Client{
		apiClient: &wire.Client{
			BaseURL: authUrl,
		},
		insecureCookies: false,
		logLevel:        LogLevelDefault,
		authUrl:         authUrl,
		tokenValidator:  validator,
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
		accessToken, refreshToken, ok := c.RefreshTokens(code)
		if !ok {
			c.log(LogLevelDebug, "handle auth code error: error refreshing with auth server\n")
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

			if err := revokeRefreshToken(c.apiClient, refreshToken); err != nil {
				c.log(LogLevelError, "handle logout: failed to revoke refresh token (%v)\n", err)
			}
		}

		// always clear cookies and redirect
		c.ClearTokenCookies(w)
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
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
	accessToken, refreshToken, ok := c.RefreshTokens(refreshToken.Encoded())
	if !ok {
		c.log(LogLevelDebug, "couldn't exchange refresh token: error refreshing with auth server\n")
		return nil, ErrNetworkTokenRefresh
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
	accessToken, refreshToken, ok := c.RefreshTokens(refreshToken.Encoded())
	if !ok {
		c.log(LogLevelDebug, "couldn't exchange refresh token: error refreshing with auth server\n")
		return nil, "", ErrNetworkTokenRefresh
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
	body, err := json.Marshal(service.RefreshRequest{RefreshToken: refreshTokenStr})
	if err != nil {
		c.log(LogLevelError, "failed to encode refresh payload: %v\n", err)
		return nil, nil, false
	}

	response := service.RefreshResponse{}
	c.log(LogLevelDebug, "POST { refresh_token } => %s/api/v1/auth/refresh\n", c.authUrl)
	if err := c.apiClient.Post("/api/v1/auth/refresh", body, &response); err != nil {
		c.log(LogLevelDebug, "POST %s/api/v1/auth/refresh failed: %v\n", c.authUrl, err)
		return nil, nil, false
	}
	if response.AccessToken == "" || response.RefreshToken == "" {
		c.log(LogLevelError, "refresh endpoint returned empty tokens\n")
		return nil, nil, false
	}

	// decode tokens from response
	accessToken := new(AccessToken)
	if err := accessToken.Decode(response.AccessToken, c.tokenValidator); err != nil {
		c.log(LogLevelError, "failed to decode access token: %v\n", err)
		return nil, nil, false
	}
	refreshToken := new(RefreshToken)
	if err := refreshToken.Decode(response.RefreshToken, c.tokenValidator); err != nil {
		c.log(LogLevelError, "failed to decode refresh token: %v\n", err)
		return nil, nil, false
	}
	return accessToken, refreshToken, true
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

func (c *Client) FetchMe(
	accessToken string,
) (
	*MeResponse,
	error,
) {
	request, err := http.NewRequest(http.MethodGet, c.authUrl+"/api/v1/auth/me", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create /api/v1/auth/me request: %v", err)
	}
	request.Header.Set("Authorization", "Bearer "+accessToken)

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("failed to call /api/v1/auth/me: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("/api/v1/auth/me returned status %d", response.StatusCode)
	}

	var body struct {
		Data MeResponse `json:"data"`
	}
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("failed to decode /api/v1/auth/me response: %v", err)
	}

	return &body.Data, nil
}

func getCookie(r *http.Request, cookieName string) *http.Cookie {
	if cookie, err := r.Cookie(cookieName); err == nil {
		return cookie
	}
	return nil
}

func revokeRefreshToken(
	client *wire.Client,
	refreshToken *RefreshToken,
) error {
	body, err := json.Marshal(
		service.LogoutRequest{
			RefreshToken: refreshToken.Encoded(),
		},
	)
	if err != nil {
		return fmt.Errorf("failed to encode logout payload: %v\n", err)
	}

	err = client.Post("/api/v1/auth/logout", body, nil)
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

func errorIsRefreshable(err error) bool {
	if errors.Is(err, ErrTokenAbsent) {
		return true
	} else if errors.Is(err, tokens.ErrTokenExpired()) {
		return true
	} else {
		return false
	}
}
