package main

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"git.sr.ht/~jakintosh/consent/internal/api"
	"git.sr.ht/~jakintosh/consent/internal/tokens"
)

var verificationKey *ecdsa.PublicKey

func main() {

	verificationKeyBytes := loadCredential("verification_key.der", "./etc/secrets/")
	verificationKey = decodePublicKey(verificationKeyBytes)
	tokens.InitPublic(verificationKey, "auth.studiopollinator.com")

	http.HandleFunc("/", home)
	http.HandleFunc("/api/authorize", authorize)
	http.HandleFunc("/api/example", example)

	err := http.ListenAndServe(":10000", nil)
	if err != nil {
		log.Fatalf("%v", err)
	}
}

func authenticate(accessTokenCookie *http.Cookie, err error) *tokens.AccessToken {
	if err != nil {
		return nil
	}

	tokenStr := accessTokenCookie.Value
	if tokenStr == "" {
		return nil
	}

	token := new(tokens.AccessToken)
	if err := token.Decode(tokenStr); err != nil {
		return nil
	}

	return token
}

func home(w http.ResponseWriter, r *http.Request) {
	var html string
	if token := authenticate(r.Cookie("accessToken")); token != nil {
		html = fmt.Sprintf(`<!DOCTYPE html>
<html>
<body>
<a href="http://localhost:10000/api/example">Example API Call</a>
</body>
</html>`)
	} else {
		html = `<!DOCTYPE html>
<html>
<body>
<a href="http://localhost:9001/login?service=example@localhost">Log In with Pollinator</a>
</body>
</html>`
	}
	w.Write([]byte(html))
}
func authorize(w http.ResponseWriter, r *http.Request) {
	queries := r.URL.Query()
	code := queries.Get("auth_code")
	if code == "" {
		log.Printf("error: called %s without 'auth_code' query param", r.RequestURI)
	} else {
		accessTokenCookie, refreshTokenCookie, err := refreshTokenCookies(code)
		if err != nil {
			log.Printf("error: failed to refresh token cookies: %v", err)
		} else {
			http.SetCookie(w, accessTokenCookie)
			http.SetCookie(w, refreshTokenCookie)
		}
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}
func example(w http.ResponseWriter, r *http.Request) {
	var html string
	if token := authenticate(r.Cookie("accessToken")); token != nil {
		html = fmt.Sprintf(`<!DOCTYPE html>
<html>
<body>
<p>Secret logged in page for %s!</p>
<form>
	<input hidden value="%s"/>
</form>
</body>
</html>`, token.Subject(), token.Secret())
	} else {
		html = `<!DOCTYPE html>
<html>
<body>
<p>You are not logged in.</p>
</body>
</html>`
	}
	w.Write([]byte(html))
}

func postRefresh(baseURL string, token string) (*http.Response, error) {
	url := fmt.Sprintf("%s/api/refresh", baseURL)
	json := fmt.Sprintf(`{ "refreshToken" : "%s" }`, token)
	body := bytes.NewBuffer([]byte(json))
	return http.Post(url, "application/json", body)
}

func refreshTokenCookies(code string) (*http.Cookie, *http.Cookie, error) {

	response, err := postRefresh("http://localhost:9001", code)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to post refresh: %v", err)
	}

	var res api.RefreshResponse
	if ok := decodeResponse(&res, response); !ok {
		return nil, nil, fmt.Errorf("failed to decode refresh response: %v", err)
	}

	accessToken := new(tokens.AccessToken)
	if err := accessToken.Decode(res.AccessToken); err != nil {
		return nil, nil, fmt.Errorf("failed to decode access token: %v", err)
	}

	refreshToken := new(tokens.RefreshToken)
	if err := refreshToken.Decode(res.RefreshToken); err != nil {
		return nil, nil, fmt.Errorf("failed to decode refresh token: %v", err)
	}

	now := time.Now()
	accessMaxAge := accessToken.Expiration().Sub(now).Seconds()
	refreshMaxAge := refreshToken.Expiration().Sub(now).Seconds()

	accessTokenCookie := &http.Cookie{
		Name:     "accessToken",
		Path:     "/",
		Value:    res.AccessToken,
		MaxAge:   int(accessMaxAge),
		SameSite: http.SameSiteStrictMode,
		Secure:   true,
		HttpOnly: true,
	}
	refreshTokenCookie := &http.Cookie{
		Name:     "refreshToken",
		Path:     "/",
		Value:    res.RefreshToken,
		MaxAge:   int(refreshMaxAge),
		SameSite: http.SameSiteStrictMode,
		Secure:   true,
		HttpOnly: true,
	}

	return accessTokenCookie, refreshTokenCookie, nil
}

func decodeResponse[T any](res *T, r *http.Response) bool {
	err := json.NewDecoder(r.Body).Decode(&res)
	if err != nil {
		return false
	}
	return true
}

func decodePublicKey(bytes []byte) *ecdsa.PublicKey {
	parsedKey, err := x509.ParsePKIXPublicKey(bytes)
	if err != nil {
		log.Fatalf("decodePublicKey: failed to parse ecdsa verification key from PEM block\n")
	}

	ecdsaKey, ok := parsedKey.(*ecdsa.PublicKey)
	if !ok {
		log.Fatalf("decodePublicKey: failed to cast parsed key as *ecdsa.PublicKey")
	}

	return ecdsaKey
}

func loadCredential(name string, credsDir string) []byte {
	credPath := filepath.Join(credsDir, name)
	cred, err := os.ReadFile(credPath)
	if err != nil {
		log.Fatalf("failed to load required credential '%s': %v\n", name, err)
	}
	return cred
}
