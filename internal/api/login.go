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

	response, err := authenticate(req.Handle, req.Password)
	if err != nil {
		logApiErr(r, fmt.Sprintf("'%s' failed to authenticate: %v", req.Handle, err))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	returnJson(response, w)
}

func authenticate(handle string, secret string) (*LoginResponse, error) {

	hash, err := database.GetSecret(handle)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve secret: %v", err)
	}

	err = bcrypt.CompareHashAndPassword(hash, []byte(secret))
	if err != nil {
		return nil, fmt.Errorf("password does not match")
	}

	refTk, accTk, err := generateTokens(handle)
	if err != nil {
		return nil, err
	}

	response := &LoginResponse{
		RefreshToken: refTk,
		AccessToken:  accTk,
	}

	return response, nil
}

func generateTokens(handle string) (refresh string, access string, err error) {

	now := time.Now()
	if refresh, err = generateToken(handle, now, time.Minute*30); err != nil {
		return
	}
	if access, err = generateToken(handle, now, time.Hour*24); err != nil {
		return
	}
	return
}

func generateToken(handle string, issueTime time.Time, lifetime time.Duration) (string, error) {

	expiration := issueTime.Add(lifetime).Unix()
	claims := jwt.MapClaims{
		"iss": "auth.studiopollinator.com",
		"sub": handle,
		"exp": expiration,
	}

	return jwt.
		NewWithClaims(jwt.SigningMethodES256, claims).
		SignedString(signingKey)
}
