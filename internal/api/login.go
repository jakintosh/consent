package api

import (
	"fmt"
	"net/http"
	"time"

	"git.sr.ht/~jakintosh/consent/internal/database"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type LoginRequest struct {
	Handle   string `json:"username"`
	Password string `json:"password"`
}

type LoginResponse struct {
	RefreshToken string `json:"refreshToken"`
	AccessToken  string `json:"accessToken"`
}

func Login(w http.ResponseWriter, r *http.Request) {

	var req LoginRequest
	if ok := decodeRequest(&req, w, r); !ok {
		return
	}

	err := authenticate(req.Handle, req.Password)
	if err != nil {
		logApiErr(r, fmt.Sprintf("'%s' failed to authenticate: %v", req.Handle, err))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	refreshStr, accessStr, err := issueTokens(req.Handle)
	if err != nil {
		logApiErr(r, fmt.Sprintf("failed to issue tokens: %v", err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	response := &LoginResponse{
		RefreshToken: refreshStr,
		AccessToken:  accessStr,
	}

	returnJson(response, w)
}

func authenticate(handle string, secret string) error {

	hash, err := database.GetSecret(handle)
	if err != nil {
		return fmt.Errorf("failed to retrieve secret: %v", err)
	}

	err = bcrypt.CompareHashAndPassword(hash, []byte(secret))
	if err != nil {
		return fmt.Errorf("password does not match")
	}

	return nil
}

func issueTokens(handle string) (string, string, error) {
	issueTime := time.Now()

	refresh := generateToken(handle, issueTime, time.Minute*30)
	refreshStr, err := refresh.toString()
	if err != nil {
		return "", "", fmt.Errorf("failed to encode refresh token: %v", err)
	}

	access := generateToken(handle, issueTime, time.Hour*24)
	accessStr, err := access.toString()
	if err != nil {
		return "", "", fmt.Errorf("failed to encode access token: %v", err)
	}

	err = database.InsertRefresh(handle, refreshStr, refresh.expiration)
	if err != nil {
		return "", "", fmt.Errorf("failed to insert refresh token: %v", err)
	}

	return refreshStr, accessStr, nil
}

type Token struct {
	expiration int64
	issuer     string
	subject    string
}

func (t Token) toString() (string, error) {
	claims := jwt.MapClaims{
		"iss": t.issuer,
		"sub": t.subject,
		"exp": t.expiration,
	}
	return jwt.
		NewWithClaims(jwt.SigningMethodES256, claims).
		SignedString(signingKey)
}

func generateToken(
	handle string,
	issueTime time.Time,
	lifetime time.Duration,
) *Token {

	return &Token{
		expiration: issueTime.Add(lifetime).Unix(),
		issuer:     "auth.studiopollinator.com",
		subject:    handle,
	}
}
