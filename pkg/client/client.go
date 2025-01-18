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
	ErrNoToken       = errors.New("no token")
	ErrTokenRequest  = errors.New("failed to fetch token")
	ErrTokenResponse = errors.New("invalid token response")
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
	logErr := func(w http.ResponseWriter, r *http.Request, msg string) {
		_log(LogLevelError, "handle authorization code error: %s\n", msg)
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}

	// extract 'auth_code' refresh token
	queries := r.URL.Query()
	code := queries.Get("auth_code")
	if code == "" {
		logErr(w, r, "missing required 'auth_code' query param")
		return
	}

	accessToken, refreshToken, err := RefreshTokens(code)
	if err != nil {
		logErr(w, r, fmt.Sprintf("%v", err))
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

A nil token means no authorization, and an error will be [ErrTokenIllegal],
[ErrTokenRequest], or [ErrTokenResponse]
*/
func VerifyAuthorization(
	w http.ResponseWriter,
	r *http.Request,
) (*AccessToken, error) {
	// validate access token in the request
	accessToken, err := validateAccessToken(r)
	if accessToken != nil {
		return accessToken, nil
	} else if !errorIsRefreshable(err) {
		return nil, err
	}

	// if in refreshable state, validate refresh token
	refreshToken, err := validateRefreshToken(r)
	if err != nil {
		_log(LogLevelDebug, "failed to validate refresh token: %v\n", err)
		return nil, err
	}

	// refresh the tokens
	accessToken, refreshToken, err = RefreshTokens(refreshToken.Encoded())
	if err != nil {
		_log(LogLevelDebug, "couldn't exchange refresh token: %v\n", err)
		return nil, err
	}

	SetTokenCookies(w, accessToken, refreshToken)
	return accessToken, err
}

func VerifyAuthorizationGetCSRF(
	w http.ResponseWriter,
	r *http.Request,
) (*AccessToken, string, error) {

	accessToken, err := VerifyAuthorization(w, r)
	if err != nil {
		return accessToken, "", err
	}

	// validate refresh token from request
	refreshToken, err := validateRefreshToken(r)
	if err != nil {
		_log(LogLevelDebug, "failed to validate refresh token: %v\n", err)
		return nil, "", err
	}

	return accessToken, refreshToken.Secret(), nil
}

/*
In this flow, we decode the RefreshToken first to see if the CSRF code matches.
*/
func VerifyAuthorizationCheckCSRF(
	w http.ResponseWriter,
	r *http.Request,
	csrf string,
) (*AccessToken, string, error) {
	// validate refresh token from request
	refreshToken, err := validateRefreshToken(r)
	if err != nil {
		_log(LogLevelDebug, "failed to validate refresh token: %v\n", err)
		return nil, "", err
	}

	// verify csrf token
	if refreshToken.Secret() != csrf {
		// csrf token verification failed
		return nil, "", nil
	}

	// validate access token in the request
	accessToken, err := validateAccessToken(r)
	if accessToken != nil {
		return accessToken, refreshToken.Secret(), nil
	} else if !errorIsRefreshable(err) {
		return nil, "", err
	}

	// refresh the tokens
	accessToken, refreshToken, err = RefreshTokens(refreshToken.Encoded())
	if err != nil {
		_log(LogLevelDebug, "couldn't exchange refresh token: %v\n", err)
		return nil, "", err
	}

	SetTokenCookies(w, accessToken, refreshToken)
	return accessToken, refreshToken.Secret(), err
}

/*
Uses the provided encoded RefreshToken to fetch new tokens from the auth
server. You can automatically invoke this behavior with VerifyAuthorization(),
but can use this on its own to compose custom refresh flows.

Returns decoded token structures, and an error that can be [ErrTokenRequest]
or [ErrTokenResponse].
*/
func RefreshTokens(refreshTokenStr string) (*AccessToken, *RefreshToken, error) {

	// construct a POST request to the /api/refresh route
	url := fmt.Sprintf("%s/api/refresh", _authUrl)
	body := bytes.NewBuffer([]byte(fmt.Sprintf(`{ "refreshToken" : "%s" }`, refreshTokenStr)))
	_log(LogLevelDebug, "posting refresh token to %s\n", url)
	apiResponse, err := http.Post(url, "application/json", body)
	if err != nil {
		_log(LogLevelError, "failed to post refresh: %v\n", err)
		return nil, nil, ErrTokenRequest
	}

	// decode api response
	refreshResponse := new(api.RefreshResponse)
	if err := json.NewDecoder(apiResponse.Body).Decode(refreshResponse); err != nil {
		_log(LogLevelError, "failed to decode api response: %v\n", err)
		return nil, nil, ErrTokenResponse
	}

	// decode tokens from response
	accessToken := new(AccessToken)
	if err := accessToken.Decode(refreshResponse.AccessToken); err != nil {
		_log(LogLevelError, "failed to decode access token: %v\n", err)
		return nil, nil, ErrTokenResponse
	}
	refreshToken := new(RefreshToken)
	if err := refreshToken.Decode(refreshResponse.RefreshToken); err != nil {
		_log(LogLevelError, "failed to decode refresh token: %v\n", err)
		return nil, nil, ErrTokenResponse
	}
	return accessToken, refreshToken, nil
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
	// get access token cookie
	cookie := getCookie(r, "accessToken")
	if cookie == nil {
		return nil, ErrNoToken
	}

	// decode + validate access token
	token := new(AccessToken)
	err := token.Decode(cookie.Value)
	if err == nil {
		return token, nil
	} else {
		return nil, err
	}
}

func validateRefreshToken(r *http.Request) (*RefreshToken, error) {
	cookie := getCookie(r, "refreshToken")
	if cookie == nil {
		return nil, ErrNoToken
	}

	token := new(RefreshToken)
	err := token.Decode(cookie.Value)
	if err == nil {
		return token, nil
	} else {
		return nil, err
	}
}

func errorIsRefreshable(err error) bool {
	if errors.Is(err, ErrNoToken) {
		return true
	} else if errors.Is(err, ErrTokenInvalid) {
		return true
	} else {
		return false
	}
}
