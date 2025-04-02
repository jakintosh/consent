package api

import (
	"fmt"
	"net/http"
)

type LogoutRequest struct {
	RefreshToken string `json:"refreshToken"`
}

func Logout(w http.ResponseWriter, r *http.Request) {

	var req LogoutRequest
	if ok := decodeRequest(&req, w, r); !ok {
		return
	}

	ok, err := DeleteRefresh(req.RefreshToken)
	if !ok {
		logApiErr(r, fmt.Sprintf("invalid refresh token: %s", req.RefreshToken))
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if err != nil {
		logApiErr(r, fmt.Sprintf("failed to delete refresh token: %v", err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
