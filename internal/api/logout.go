package api

import (
	"net/http"
)

type LogoutRequest struct {
	RefreshToken string `json:"refreshToken"`
}

func (a *API) Logout() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req LogoutRequest
		if ok := decodeRequest(&req, w, r); !ok {
			return
		}

		err := a.service.RevokeRefreshToken(req.RefreshToken)
		if err != nil {
			writeError(w, r, err)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}
