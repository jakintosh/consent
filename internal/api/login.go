package api

import (
	"encoding/json"
	"fmt"
	"log"
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

func Login(w http.ResponseWriter, r *http.Request) {

	var req LoginRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		jsonErr(w, r)
		return
	}

	err = authenticate(w, req.Handle, req.Password)
	if err != nil {
		apiErr(r, "failed to authenticate")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	log.Printf("Login successful for %s", req.Handle)
	w.WriteHeader(http.StatusOK)
}

func authenticate(w http.ResponseWriter, handle string, secret string) error {

	if !checkPassword(handle, secret) {
		return fmt.Errorf("password check failed")
	}

	refTk, accTk, err := generateTokens(handle)
	if err != nil {
		return err
	}

	log.Printf("Refresh Token: %s\n", refTk)
	log.Printf("Access Token: %s\n", accTk)

	return nil
}

func checkPassword(handle string, secret string) bool {
	hash, err := database.GetSecret(handle)
	if err != nil {
		// TODO: log couldn't fetch secret
		return false
	}

	err = bcrypt.CompareHashAndPassword(hash, []byte(secret))
	if err != nil {
		// TODO: comparison failed
		return false
	}

	return true
}

func generateTokens(handle string) (refresh string, access string, err error) {

	now := time.Now()
	if refresh, err = genToken(handle, now, time.Minute*30); err != nil {
		return
	}
	if access, err = genToken(handle, now, time.Hour*24); err != nil {
		return
	}
	return
}

func genToken(handle string, issueTime time.Time, lifetime time.Duration) (string, error) {
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
