package api

import (
	"errors"
	"fmt"
	"net/http"

	"git.sr.ht/~jakintosh/consent/internal/database"
	"github.com/golang-jwt/jwt/v5"
)

type RefreshRequest struct {
	RefreshToken string `json:"refreshToken"`
}

type RefreshResponse struct {
	RefreshToken string `json:"refreshToken"`
	AccessToken  string `json:"accessToken"`
}

func Refresh(w http.ResponseWriter, r *http.Request) {

	var req RefreshRequest
	if ok := decodeRequest(&req, w, r); !ok {
		return
	}

	ok, err := database.DeleteRefresh(req.RefreshToken)
	if !ok {
		logApiErr(r, "refresh token not found")
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if err != nil {
		logApiErr(r, fmt.Sprintf("refresh couldn't be deleted: %v", err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	token, err := parseToken(req.RefreshToken)
	if err != nil {
		logApiErr(r, fmt.Sprintf("token parse error: %v\n", err))
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	subject, err := token.Claims.GetSubject()
	if err != nil {
		logApiErr(r, fmt.Sprintf("couldn't parse subject claim: %v", err))
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	refresh, access, err := issueTokens(subject)
	if err != nil {
		logApiErr(r, fmt.Sprintf("couldn't issue tokens: %v", err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	response := &RefreshResponse{
		RefreshToken: refresh,
		AccessToken:  access,
	}

	returnJson(response, w)
}

func parseToken(tokenStr string) (*jwt.Token, error) {
	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		return &signingKey.PublicKey, nil
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
