package api

import (
	"net/http"

	"git.sr.ht/~jakintosh/command-go/pkg/wire"
)

type RefreshRequest struct {
	RefreshToken string `json:"refreshToken"`
}

type RefreshResponse struct {
	RefreshToken string `json:"refreshToken"`
	AccessToken  string `json:"accessToken"`
}

func (a *API) Refresh() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req RefreshRequest
		if ok := decodeRequest(&req, w, r); !ok {
			return
		}

		accessToken, refreshToken, err := a.service.RefreshTokens(req.RefreshToken)
		if err != nil {
			wire.WriteError(w, httpStatusFromError(err), err.Error())
			return
		}

		response := RefreshResponse{
			RefreshToken: refreshToken,
			AccessToken:  accessToken,
		}
		wire.WriteData(w, http.StatusOK, response)
	}
}
