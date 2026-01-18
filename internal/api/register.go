package api

import (
	"net/http"

	"git.sr.ht/~jakintosh/command-go/pkg/wire"
)

type RegistrationRequest struct {
	Handle   string `json:"username"`
	Password string `json:"password"`
}

func (a *API) Register() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req RegistrationRequest
		if ok := decodeRequest(&req, w, r); !ok {
			return
		}

		err := a.service.Register(req.Handle, req.Password)
		if err != nil {
			wire.WriteError(w, httpStatusFromError(err), err.Error())
			return
		}

		wire.WriteData(w, http.StatusOK, nil)
	}
}
