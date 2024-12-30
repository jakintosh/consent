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

var _authUrl string

func Init(authUrl string, issuer string, publicKey *ecdsa.PublicKey) {
	tokens.InitPublic(publicKey, issuer)

	_authUrl = authUrl
}

/*
This should allow a client to pass in a request and determine weather or not
the request is authorized, and if so, return the access token.

A nil token means no authorization, and an error means the token was invalid.
*/
func VerifyAuthorization(r *http.Request) (*AccessToken, error) {
	cookie, err := r.Cookie("accessToken")
	if errors.Is(err, http.ErrNoCookie) {
		return nil, nil
	}

	token := new(AccessToken)
	if err := token.Decode(cookie.Value); err != nil {
		return nil, fmt.Errorf("couldn't decode cookie: %v", err)
	}

	return token, nil
}

func RefreshAuthorization(w http.ResponseWriter, r *http.Request) error {
	cookie, err := r.Cookie("refreshToken")
	if errors.Is(err, http.ErrNoCookie) {
		return fmt.Errorf("missing refresh token")
	}

	token := new(RefreshToken)
	if err := token.Decode(cookie.Value); err != nil {
		return fmt.Errorf("couldn't decode cookie: %v", err)
	}

	accessTokenCookie, refreshTokenCookie, err := refreshTokens(token.Encoded())
	if err != nil {
		return fmt.Errorf("couldn't exchange refresh token: %v", err)
	}

	http.SetCookie(w, accessTokenCookie)
	http.SetCookie(w, refreshTokenCookie)
	return nil
}

/*
This can be passed as a route handler to fully handle the authorization
code flow for a client. Set this to the same route you register with the
auth server as the redirect link, and it works out of the box.
*/
func HandleAuthorizationCode(w http.ResponseWriter, r *http.Request) {
	logErr := func(w http.ResponseWriter, r *http.Request, msg string) {
		log.Printf("handle authorization code error: %s\n", msg)
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}

	// extract 'auth_code' refresh token
	queries := r.URL.Query()
	code := queries.Get("auth_code")
	if code == "" {
		logErr(w, r, "missing required 'auth_code' query param")
		return
	}

	accessTokenCookie, refreshTokenCookie, err := refreshTokens(code)
	if err != nil {
		logErr(w, r, fmt.Sprintf("%v", err))
		return
	}

	http.SetCookie(w, accessTokenCookie)
	http.SetCookie(w, refreshTokenCookie)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func refreshTokens(refreshTokenStr string) (*http.Cookie, *http.Cookie, error) {

	// construct a POST request to the /api/refresh route
	url := fmt.Sprintf("%s/api/refresh", _authUrl)
	body := bytes.NewBuffer([]byte(fmt.Sprintf(`{ "refreshToken" : "%s" }`, refreshTokenStr)))
	apiResponse, err := http.Post(url, "application/json", body)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to post refresh: %v", err)
	}

	// decode api response
	refreshResponse := new(api.RefreshResponse)
	if err := json.NewDecoder(apiResponse.Body).Decode(refreshResponse); err != nil {
		return nil, nil, fmt.Errorf("failed to decode refresh response: %v", err)
	}

	// decode tokens from response
	accessToken := new(AccessToken)
	if err := accessToken.Decode(refreshResponse.AccessToken); err != nil {
		return nil, nil, fmt.Errorf("failed to decode access token: %v", err)
	}
	refreshToken := new(RefreshToken)
	if err := refreshToken.Decode(refreshResponse.RefreshToken); err != nil {
		return nil, nil, fmt.Errorf("failed to decode refresh token: %v", err)
	}

	// construct cookies and set on response
	now := time.Now()
	accessMaxAge := accessToken.Expiration().Sub(now).Seconds()
	refreshMaxAge := refreshToken.Expiration().Sub(now).Seconds()

	accessTokenCookie := &http.Cookie{
		Name:     "accessToken",
		Path:     "/",
		Value:    refreshResponse.AccessToken,
		MaxAge:   int(accessMaxAge),
		SameSite: http.SameSiteStrictMode,
		Secure:   true,
		HttpOnly: true,
	}

	refreshTokenCookie := &http.Cookie{
		Name:     "refreshToken",
		Path:     "/",
		Value:    refreshResponse.RefreshToken,
		MaxAge:   int(refreshMaxAge),
		SameSite: http.SameSiteStrictMode,
		Secure:   true,
		HttpOnly: true,
	}

	return accessTokenCookie, refreshTokenCookie, nil

}
