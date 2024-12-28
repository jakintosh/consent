package main

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"git.sr.ht/~jakintosh/consent/internal/api"
	"github.com/golang-jwt/jwt/v5"
)

var verificationKey *ecdsa.PublicKey

func main() {

	verificationKeyBytes := loadCredential("verification_key.der", "./etc/secrets/")
	verificationKey = decodePublicKey(verificationKeyBytes)

	http.HandleFunc("/", home)
	http.HandleFunc("/api/authorize", authorize)
	http.HandleFunc("/api/example", example)

	err := http.ListenAndServe(":10000", nil)
	if err != nil {
		log.Fatalf("%v", err)
	}
}

func authenticate(accessTokenCookie *http.Cookie, err error) (string, bool) {
	if err != nil {
		return "", false
	}
	var tokenStr = accessTokenCookie.Value
	if tokenStr == "" {
		return "", false
	}
	token, err := parseToken(tokenStr)
	if err != nil {
		return "", false
	}

	sub, err := token.Claims.GetSubject()
	if err != nil {
		return "", false
	}

	return sub, true
}

func home(w http.ResponseWriter, r *http.Request) {
	var html string
	if _, ok := authenticate(r.Cookie("accessToken")); ok {
		html = `<!DOCTYPE html>
<html>
<body>
<a href="http://localhost:10000/api/example">Example API Call</a>
</body>
</html>`
	} else {
		html = `<!DOCTYPE html>
<html>
<body>
<a href="http://localhost:9001/login?audience=http://localhost:10000&redirect_url=http://localhost:10000/api/authorize">Log In with Pollinator</a>
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
	if sub, ok := authenticate(r.Cookie("accessToken")); ok {
		html = fmt.Sprintf(`<!DOCTYPE html>
<html>
<body>
<p>Secret logged in page for %s!</p>
</body>
</html>`, sub)
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

	accessToken, err := parseToken(res.AccessToken)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse access token: %v", err)
	}

	accessExp, err := accessToken.Claims.GetExpirationTime()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read access exp: %v", err)
	}

	refreshToken, err := parseToken(res.RefreshToken)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse refresh token: %v", err)
	}

	refreshExp, err := refreshToken.Claims.GetExpirationTime()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read refresh exp: %v", err)
	}

	accessTokenCookie := &http.Cookie{
		Name:     "accessToken",
		Path:     "/",
		Value:    res.AccessToken,
		Expires:  accessExp.Time,
		Secure:   true,
		HttpOnly: true,
	}
	refreshTokenCookie := &http.Cookie{
		Name:     "refreshToken",
		Path:     "/",
		Value:    res.RefreshToken,
		Expires:  refreshExp.Time,
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

func parseToken(tokenStr string) (*jwt.Token, error) {
	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		return verificationKey, nil
	})

	if err != nil {
		switch {
		case errors.Is(err, jwt.ErrTokenMalformed):
			return nil, err
		case errors.Is(err, jwt.ErrTokenSignatureInvalid):
			return nil, err
		case errors.Is(err, jwt.ErrTokenExpired):
			return nil, err
		case errors.Is(err, jwt.ErrTokenNotValidYet):
			return nil, err
		default:
			return nil, err
		}
	}

	if !token.Valid {
		return nil, fmt.Errorf("Token is not valid")
	}

	return token, nil
}
func loadCredential(name string, credsDir string) []byte {
	credPath := filepath.Join(credsDir, name)
	cred, err := os.ReadFile(credPath)
	if err != nil {
		log.Fatalf("failed to load required credential '%s': %v\n", name, err)
	}
	return cred
}
