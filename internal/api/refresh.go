package api

import (
	"log"
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

	handle, err := database.GetRefreshHandle(req.RefreshToken)
	if err != nil {
		logApiErr(r, "invalid refresh token")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// make sure token is not expired

	w.WriteHeader(http.StatusOK)
}
