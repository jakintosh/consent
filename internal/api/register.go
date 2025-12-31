package api

import (
	"net/http"
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
			writeError(w, r, err)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}
