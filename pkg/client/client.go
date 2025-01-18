package client

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"git.sr.ht/~jakintosh/consent/internal/api"
	"git.sr.ht/~jakintosh/consent/internal/tokens"
)

type LogLevel int

const (
	LogLevelNone LogLevel = iota
	LogLevelError
	LogLevelInfo
	LogLevelDebug
)
const LogLevelDefault = LogLevelError

var (
	ErrTokenAbsent         = errors.New("token not present")
	ErrTokenInvalid        = errors.New("token invalid")
	ErrCSRFInvalid         = errors.New("csrf secret incorrect")
	ErrNetworkTokenRefresh = errors.New("network issue during token refresh")
)

var _logLevel LogLevel = LogLevelDefault
var _authUrl string

func _log(level LogLevel, format string, v ...any) {
	if _logLevel >= level {
		log.Printf(format, v...)
	}
}

func Init(publicKey *ecdsa.PublicKey, issuer string, audience string, authUrl string) {
	tokens.InitClient(publicKey, issuer, audience)

	_authUrl = authUrl
}

func SetLogLevel(logLevel LogLevel) {
	_logLevel = logLevel
}

/*
This can be passed as a route handler to fully handle the authorization
code flow for a client. Set this to the same route you register with the
auth server as the redirect link, and it works out of the box.
*/
func HandleAuthorizationCode(w http.ResponseWriter, r *http.Request) {
	// extract 'auth_code' refresh token
	queries := r.URL.Query()
	code := queries.Get("auth_code")
	if code == "" {
		_log(LogLevelDebug, "handle auth code error: missing required 'auth_code' query param\n")
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	// refresh tokens using code
	accessToken, refreshToken, ok := RefreshTokens(code)
	if !ok {
		_log(LogLevelDebug, "handle auth code error: error refreshing with auth server\n")
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	SetTokenCookies(w, accessToken, refreshToken)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

/*
This allows a client to pass in an http.Request and determine weather or not
the request is authorized, and if so, return the access token. If the access
token is expired, this will attempt to call the authorization server to
refresh the tokens.
*/
func VerifyAuthorization(
	w http.ResponseWriter,
	r *http.Request,
) (*AccessToken, error) {
	// validate access token in the request
	accessToken, err := validateAccessToken(r)
	if accessToken != nil {
		return accessToken, nil
	}
	if !errorIsRefreshable(err) {
		_log(LogLevelDebug, "failed to validate access token: %v\n", err)
		return nil, ErrTokenInvalid
	}

	// if in refreshable state, validate refresh token
	refreshToken, err := validateRefreshToken(r)
	if err != nil {
		_log(LogLevelDebug, "failed to validate refresh token: %v\n", err)
		return nil, ErrTokenInvalid
	}

	// refresh the tokens
	accessToken, refreshToken, ok := RefreshTokens(refreshToken.Encoded())
	if !ok {
		_log(LogLevelDebug, "couldn't exchange refresh token: error refreshing with auth server\n")
		return nil, ErrNetworkTokenRefresh
	}
	SetTokenCookies(w, accessToken, refreshToken)

	return accessToken, err
}

func VerifyAuthorizationGetCSRF(
	w http.ResponseWriter,
	r *http.Request,
) (*AccessToken, string, error) {
	// standard request verification
	accessToken, err := VerifyAuthorization(w, r)
	if err != nil {
		return accessToken, "", err
	}

	// if authorized success, validate refresh token and extract csrf secret
	refreshToken, err := validateRefreshToken(r)
	if err != nil {
		_log(LogLevelDebug, "failed to validate refresh token: %v\n", err)
		return nil, "", err
	}
	csrfSecret := refreshToken.Secret()

	return accessToken, csrfSecret, nil
}

/*
In this flow, we decode the RefreshToken first to see if the CSRF code matches.
Because the AccessToken may be legally expired, we check RefreshToken's CSRF secret
first, because after the AccessToken check the RefreshToken may have been changed.
*/
func VerifyAuthorizationCheckCSRF(
	w http.ResponseWriter,
	r *http.Request,
	reqCSRFSecret string,
) (*AccessToken, string, error) {
	// validate refresh token from request
	refreshToken, err := validateRefreshToken(r)
	if err != nil {
		_log(LogLevelDebug, "failed to validate refresh token: %v\n", err)
		return nil, "", ErrTokenInvalid
	}

	currentCSRFSecret := refreshToken.Secret()
	if currentCSRFSecret != reqCSRFSecret {
		return nil, "", ErrCSRFInvalid
	}

	// validate access token in the request
	accessToken, err := validateAccessToken(r)
	if accessToken != nil {
		return accessToken, currentCSRFSecret, nil
	}
	if !errorIsRefreshable(err) {
		return nil, "", ErrTokenInvalid
	}

	// refresh the tokens
	accessToken, refreshToken, ok := RefreshTokens(refreshToken.Encoded())
	if !ok {
		_log(LogLevelDebug, "couldn't exchange refresh token: error refreshing with auth server\n")
		return nil, "", ErrNetworkTokenRefresh
	}
	newCSRFSecret := refreshToken.Secret()

	SetTokenCookies(w, accessToken, refreshToken)
	return accessToken, newCSRFSecret, nil
}

/*
Uses the provided encoded RefreshToken to fetch new tokens from the auth
server. You can automatically invoke this behavior with VerifyAuthorization(),
but can use this on its own to compose custom refresh flows.

Returns decoded token structures, and an error that can be [ErrTokenRequest]
or [ErrTokenResponse].
*/
func RefreshTokens(refreshTokenStr string) (*AccessToken, *RefreshToken, bool) {

	// construct a POST request to the /api/refresh route
	url := fmt.Sprintf("%s/api/refresh", _authUrl)
	body := bytes.NewBuffer([]byte(fmt.Sprintf(`{ "refreshToken" : "%s" }`, refreshTokenStr)))
	_log(LogLevelDebug, "POST { refresh_token } => %s\n", url)
	apiResponse, err := http.Post(url, "application/json", body)
	if err != nil {
		_log(LogLevelError, "failed to post refresh: %v\n", err)
		return nil, nil, false
	}

	// decode api response
	if apiResponse.StatusCode != http.StatusOK {
		_log(LogLevelDebug, "POST %s returned %s\n", url, apiResponse.Status)
		return nil, nil, false
	}
	defer apiResponse.Body.Close()
	refreshResponse := new(api.RefreshResponse)
	if err := json.NewDecoder(apiResponse.Body).Decode(refreshResponse); err != nil {
		_log(LogLevelError, "failed to decode api response: %v\n", err)
		return nil, nil, false
	}

	// decode tokens from response
	accessToken := new(AccessToken)
	if err := accessToken.Decode(refreshResponse.AccessToken); err != nil {
		_log(LogLevelError, "failed to decode access token: %v\n", err)
		return nil, nil, false
	}
	refreshToken := new(RefreshToken)
	if err := refreshToken.Decode(refreshResponse.RefreshToken); err != nil {
		_log(LogLevelError, "failed to decode refresh token: %v\n", err)
		return nil, nil, false
	}
	return accessToken, refreshToken, true
}

func SetTokenCookies(w http.ResponseWriter, accessToken *AccessToken, refreshToken *RefreshToken) {
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

	_log(LogLevelDebug, "set token cookies\n")
}

func ClearTokenCookies(w http.ResponseWriter) {
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

	_log(LogLevelDebug, "cleared token cookies\n")
}

func getCookie(r *http.Request, cookieName string) *http.Cookie {
	if cookie, err := r.Cookie(cookieName); err == nil {
		return cookie
	}
	return nil
}

func validateAccessToken(r *http.Request) (*AccessToken, error) {
	cookie := getCookie(r, "accessToken")
	if cookie == nil {
		return nil, ErrTokenAbsent
	}

	token := new(AccessToken)
	err := token.Decode(cookie.Value)
	if err != nil {
		return nil, err
	}
	return token, nil
}

func validateRefreshToken(r *http.Request) (*RefreshToken, error) {
	cookie := getCookie(r, "refreshToken")
	if cookie == nil {
		return nil, ErrTokenAbsent
	}

	token := new(RefreshToken)
	err := token.Decode(cookie.Value)
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
