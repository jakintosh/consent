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
func VerifyAuthorization(w http.ResponseWriter, r *http.Request) (*AccessToken, error) {
	cookie := getCookie(r, "accessToken")
	if cookie == nil {
		_log(LogLevelDebug, "no access token; attempt to refresh auth\n")
		return refreshAuthorization(w, r)
	}

	token := new(AccessToken)
	err := token.Decode(cookie.Value)
	if err == nil {
		_log(LogLevelDebug, "access token OK\n")
		return token, nil
	} else if errors.Is(err, ErrTokenInvalid) {
		_log(LogLevelDebug, "access token invalid; attempt to refresh auth\n")
		return refreshAuthorization(w, r)
	} else {
		_log(LogLevelDebug, "access token illegal; deleting\n")
		ClearTokenCookies(w)
		return nil, ErrTokenIllegal
	}
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

func refreshAuthorization(w http.ResponseWriter, r *http.Request) (*AccessToken, error) {
	cookie := getCookie(r, "refreshToken")
	if cookie == nil {
		_log(LogLevelDebug, "no refresh token\n")
		return nil, nil
	}

	token := new(RefreshToken)
	if err := token.Decode(cookie.Value); err != nil {
		ClearTokenCookies(w)
		return nil, nil
	}

	accessToken, refreshToken, err := RefreshTokens(token.Encoded())
	if err != nil {
		_log(LogLevelDebug, "couldn't exchange refresh token: %v\n", err)
		return nil, err
	}

	SetTokenCookies(w, accessToken, refreshToken)
	return accessToken, nil
}
