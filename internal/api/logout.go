package api

import (
	"net/http"

	"git.sr.ht/~jakintosh/command-go/pkg/wire"
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
			wire.WriteError(w, httpStatusFromError(err), err.Error())
			return
		}

		wire.WriteData(w, http.StatusOK, nil)
	}
}
